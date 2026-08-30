package java

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"Aether/pkg/fs"
	"Aether/pkg/netutil"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// adoptiumAsset is a partial unmarshal of the Adoptium API response.
type adoptiumAsset struct {
	Binary struct {
		Package struct {
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
			Size     int64  `json:"size"`
		} `json:"package"`
	} `json:"binary"`
}

// GetManagedJavaPath returns the path to the java(.exe) binary for the
// given major version inside Aether's managed runtimes directory.
func GetManagedJavaPath(majorVersion int) string {
	base := getManagedJavaDir(majorVersion)
	if runtime.GOOS == "windows" {
		return filepath.Join(base, "bin", "java.exe")
	}
	if runtime.GOOS == "darwin" {
		macPath := filepath.Join(base, "Contents", "Home", "bin", "java")
		if _, err := os.Stat(macPath); err == nil {
			return macPath
		}
	}
	return filepath.Join(base, "bin", "java")
}

// IsManagedJavaInstalled reports whether a managed JRE for the given major version exists.
func IsManagedJavaInstalled(majorVersion int) bool {
	path := GetManagedJavaPath(majorVersion)
	_, err := os.Stat(path)
	return err == nil
}

// getManagedJavaDir returns the directory where the managed JRE is stored.
func getManagedJavaDir(majorVersion int) string {
	return filepath.Join(fs.GetDataDir(), "runtimes", fmt.Sprintf("java-%d", majorVersion))
}

// DownloadJava downloads and installs a managed JRE from Adoptium for the
// given major version. Emits progress events on ctx via Wails.
func DownloadJava(ctx context.Context, majorVersion int) error {
	os_ := runtime.GOOS
	if os_ == "darwin" {
		os_ = "mac"
	}
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	}

	apiURL := fmt.Sprintf(
		"https://api.adoptium.net/v3/assets/latest/%d/hotspot?os=%s&architecture=%s&image_type=jre",
		majorVersion, os_, arch,
	)

	wailsruntime.EventsEmit(ctx, "java:status", map[string]interface{}{
		"version": majorVersion,
		"phase":   "fetching",
		"message": fmt.Sprintf("Fetching Java %d download info...", majorVersion),
	})

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("adoptium API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adoptium API returned status %s", resp.Status)
	}

	var assets []adoptiumAsset
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return fmt.Errorf("failed to parse adoptium API response: %w", err)
	}
	if len(assets) == 0 {
		return fmt.Errorf("no JRE packages found for Java %d on %s/%s", majorVersion, os_, arch)
	}

	downloadURL := assets[0].Binary.Package.Link
	totalSize := assets[0].Binary.Package.Size

	wailsruntime.EventsEmit(ctx, "java:status", map[string]interface{}{
		"version": majorVersion,
		"phase":   "downloading",
		"message": fmt.Sprintf("Downloading Java %d...", majorVersion),
		"total":   totalSize,
	})

	// Download Java with resume support and progress reporting
	destPath := filepath.Join(os.TempDir(), fmt.Sprintf("aether-java-%d.zip", majorVersion))
	
	// Create progress callback
	var lastReport int64 = 0
	onProgress := func(downloaded, total int64) {
		// Emit progress every ~2 MB
		if total > 0 && downloaded-lastReport > 2*1024*1024 {
			lastReport = downloaded
			pct := int(downloaded * 100 / total)
			wailsruntime.EventsEmit(ctx, "java:status", map[string]interface{}{
				"version":  majorVersion,
				"phase":    "downloading",
				"message":  fmt.Sprintf("Downloading Java %d... %d%%", majorVersion, pct),
				"progress": pct,
				"total":    total,
			})
		}
	}

	if err := netutil.DownloadFile(ctx, downloadURL, destPath, onProgress); err != nil {
		return fmt.Errorf("java download failed: %w", err)
	}
	defer os.Remove(destPath)

	wailsruntime.EventsEmit(ctx, "java:status", map[string]interface{}{
		"version": majorVersion,
		"phase":   "extracting",
		"message": fmt.Sprintf("Extracting Java %d...", majorVersion),
	})

	destDir := getManagedJavaDir(majorVersion)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create runtime dir: %w", err)
	}

	if err := extractZip(destPath, destDir); err != nil {
		return fmt.Errorf("failed to extract JRE: %w", err)
	}

	wailsruntime.EventsEmit(ctx, "java:status", map[string]interface{}{
		"version": majorVersion,
		"phase":   "done",
		"message": fmt.Sprintf("Java %d ready.", majorVersion),
	})

	return nil
}


// extractZip extracts a zip archive into destDir, stripping the top-level
// directory that Adoptium bundles include (e.g. jdk-21.0.1-jre/).
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Determine the common prefix to strip (the bundled root folder name)
	prefix := ""
	if len(r.File) > 0 {
		parts := strings.SplitN(r.File[0].Name, "/", 2)
		if len(parts) > 1 {
			prefix = parts[0] + "/"
		}
	}

	var totalExtracted int64
	const maxTotalBytes = 500 * 1024 * 1024 // 500MB zip bomb limit
	const maxFileBytes = 100 * 1024 * 1024  // 100MB per file

	for _, f := range r.File {
		// Strip the top-level directory
		relPath := strings.TrimPrefix(f.Name, prefix)
		if relPath == "" {
			continue
		}

		cleanRel := filepath.Clean(filepath.FromSlash(relPath))
		if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
			continue
		}
		target, err := fs.ContainedPath(destDir, cleanRel)
		if err != nil {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		n, err := io.Copy(outFile, io.LimitReader(rc, maxFileBytes+1))
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
		if n > maxFileBytes {
			_ = os.Remove(target)
			return fmt.Errorf("file %s exceeds maximum size", relPath)
		}
		totalExtracted += n
		if totalExtracted > maxTotalBytes {
			return fmt.Errorf("archive exceeds maximum total size")
		}
	}
	return nil
}
