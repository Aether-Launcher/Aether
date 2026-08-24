package extensions

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

// InstallFromArchive extracts an extension archive (.aex/.zip) to a temporary location,
// validates the manifest, and moves it to the extensions directory under its ID.
func InstallFromArchive(archivePath string) error {
	extDir := filepath.Join(fs.GetDataDir(), "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(extDir, "installing-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir) // Clean up temp dir in case of failure

	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer r.Close()

	// Extract all files with ZipSlip protection and size limits
	var totalExtracted int64
	const maxTotalBytes = 50 * 1024 * 1024
	const maxFileBytes = 20 * 1024 * 1024
	for _, f := range r.File {
		cleanName := filepath.Clean(filepath.FromSlash(f.Name))
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
			continue
		}
		fpath, err := fs.ContainedPath(tempDir, cleanName)
		if err != nil {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
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

		n, err := io.Copy(outFile, io.LimitReader(rc, maxFileBytes+1))
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if n > maxFileBytes {
			_ = os.Remove(fpath)
			return fmt.Errorf("file %s exceeds maximum size", f.Name)
		}
		totalExtracted += n
		if totalExtracted > maxTotalBytes {
			return fmt.Errorf("archive exceeds maximum total size")
		}
	}

	// Some zips contain a single root folder (e.g. 'my-extension/manifest.json')
	// Let's find the manifest.json
	manifestPath := ""
	rootDir := tempDir

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "manifest.json" {
			manifestPath = path
			rootDir = filepath.Dir(path)
			return filepath.SkipDir // Stop walking
		}
		return nil
	})

	if err != nil {
		return err
	}

	if manifestPath == "" {
		return fmt.Errorf("invalid extension: manifest.json not found in zip")
	}

	// Parse manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.ID == "" {
		return fmt.Errorf("invalid manifest: missing 'id'")
	}
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, manifest.ID); !matched {
		return fmt.Errorf("invalid manifest id %q", manifest.ID)
	}

	// Target directory with containment check
	targetDir, err := fs.ContainedPath(extDir, manifest.ID)
	if err != nil {
		return fmt.Errorf("invalid manifest id: %w", err)
	}
	
	// Remove old version if it exists
	os.RemoveAll(targetDir)

	// Move the rootDir to the targetDir
	if err := os.Rename(rootDir, targetDir); err != nil {
		// Fallback for cross-device rename issues if needed, though they are in the same folder here
		return fmt.Errorf("failed to move extension to final directory: %w", err)
	}

	return nil
}
