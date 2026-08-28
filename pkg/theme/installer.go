package theme

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"Aether/pkg/fs"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// pngMagic is the 8-byte signature every valid PNG file starts with.
var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

const (
	maxThemeTotalBytes int64 = 20 * 1024 * 1024
	maxThemeFileBytes  int64 = 8 * 1024 * 1024
)

// InstallResult carries back anything the user should know about after an
// install — most importantly, which parts of a theme (if any) were rejected
// or modified for safety reasons.
type InstallResult struct {
	Manifest Manifest
	Warnings []string
}

func getThemesDir() string {
	return filepath.Join(fs.GetDataDir(), "themes")
}

// InstallFromArchive extracts a .theme archive (a zip file, regardless of its
// extension), validates package.json, sanitizes theme.css, filters
// overwrite.json down to the allowed asset keys, and moves the result into
// the themes directory under its manifest ID.
func InstallFromArchive(archivePath string) (*InstallResult, error) {
	themesDir := getThemesDir()
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp(themesDir, "installing-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if err := extractZip(archivePath, tempDir); err != nil {
		return nil, err
	}

	// A .theme may wrap its contents in a single root folder, same as .aex.
	manifestPath := ""
	rootDir := tempDir
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "package.json" {
			manifestPath = path
			rootDir = filepath.Dir(path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if manifestPath == "" {
		return nil, fmt.Errorf("invalid theme: package.json not found")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}
	manifest.applyDefaults()

	if manifest.ID == "" {
		return nil, fmt.Errorf("invalid package.json: missing 'id'")
	}
	if !idPattern.MatchString(manifest.ID) {
		return nil, fmt.Errorf("invalid theme id %q", manifest.ID)
	}
	if manifest.Name == "" {
		return nil, fmt.Errorf("invalid package.json: missing 'name'")
	}

	var warnings []string

	// ── CSS ──────────────────────────────────────────────────────────────
	cssPath, err := fs.ContainedPath(rootDir, manifest.CSS)
	if err != nil {
		return nil, fmt.Errorf("invalid css path: %w", err)
	}
	rawCSS, err := os.ReadFile(cssPath)
	if err != nil {
		return nil, fmt.Errorf("theme is missing its stylesheet (%s)", manifest.CSS)
	}
	cleanCSS, cssWarnings := SanitizeCSS(string(rawCSS))
	warnings = append(warnings, cssWarnings...)
	if err := os.WriteFile(cssPath, []byte(cleanCSS), 0644); err != nil {
		return nil, fmt.Errorf("failed to write sanitized stylesheet: %w", err)
	}

	// ── overwrite.json (optional PNG asset overrides) ────────────────────
	overwritePath, err := fs.ContainedPath(rootDir, manifest.Overwrite)
	if err == nil {
		if raw, readErr := os.ReadFile(overwritePath); readErr == nil {
			var requested map[string]string
			if jsonErr := json.Unmarshal(raw, &requested); jsonErr == nil {
				filtered := map[string]string{}
				for key, filename := range requested {
					if LockedAssetKeys[key] {
						warnings = append(warnings, fmt.Sprintf("overwrite.json cannot set %q — this is locked to the launcher's built-in branding and was ignored", key))
						continue
					}
					if !AllowedAssetKeys[key] {
						warnings = append(warnings, fmt.Sprintf("overwrite.json key %q is not a recognized asset slot and was ignored", key))
						continue
					}
					assetPath, pathErr := fs.ContainedPath(rootDir, filename)
					if pathErr != nil {
						warnings = append(warnings, fmt.Sprintf("overwrite.json entry %q points outside the theme package and was ignored", key))
						continue
					}
					if !isValidPNG(assetPath) {
						warnings = append(warnings, fmt.Sprintf("overwrite.json entry %q does not point at a valid PNG and was ignored", key))
						continue
					}
					filtered[key] = filename
				}
				cleanOverwrite, _ := json.MarshalIndent(filtered, "", "  ")
				if err := os.WriteFile(overwritePath, cleanOverwrite, 0644); err != nil {
					return nil, fmt.Errorf("failed to write sanitized overwrite.json: %w", err)
				}
			} else {
				warnings = append(warnings, "overwrite.json was present but could not be parsed and was ignored")
				_ = os.WriteFile(overwritePath, []byte("{}"), 0644)
			}
		}
	}

	// Target directory with containment check
	targetDir, err := fs.ContainedPath(themesDir, manifest.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid theme id: %w", err)
	}
	os.RemoveAll(targetDir)
	if err := os.Rename(rootDir, targetDir); err != nil {
		return nil, fmt.Errorf("failed to move theme to final directory: %w", err)
	}

	return &InstallResult{Manifest: manifest, Warnings: warnings}, nil
}

// Uninstall removes an installed theme by ID.
func Uninstall(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid theme id")
	}
	dir, err := fs.ContainedPath(getThemesDir(), id)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func isValidPNG(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 8)
	if _, err := io.ReadFull(f, header); err != nil {
		return false
	}
	for i, b := range pngMagic {
		if header[i] != b {
			return false
		}
	}
	return true
}

// extractZip extracts archivePath (a zip file, whatever its extension) into
// destDir with ZipSlip protection and size limits.
func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer r.Close()

	var totalExtracted int64
	for _, f := range r.File {
		cleanName := filepath.Clean(filepath.FromSlash(f.Name))
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
			continue
		}
		fpath, err := fs.ContainedPath(destDir, cleanName)
		if err != nil {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		n, err := io.Copy(outFile, io.LimitReader(rc, maxThemeFileBytes+1))
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if n > maxThemeFileBytes {
			_ = os.Remove(fpath)
			return fmt.Errorf("file %s exceeds maximum size", f.Name)
		}
		totalExtracted += n
		if totalExtracted > maxThemeTotalBytes {
			return fmt.Errorf("archive exceeds maximum total size")
		}
	}
	return nil
}
