// Package update implements self-updates from GitHub Releases.
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"Aether/pkg/extensions"
	"Aether/pkg/netutil"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	repoAPI   = "https://api.github.com/repos/wayback09/Aether"
	repoDownload = "https://github.com/wayback09/Aether/releases/download"
)

// Info describes an available update.
type Info struct {
	Version       string `json:"version"`
	AssetName     string `json:"assetName"`
	DownloadURL   string `json:"downloadUrl"`
	ReleaseNotes  string `json:"releaseNotes"`
	IsPrerelease  bool   `json:"isPrerelease"`
	InstallerOnly bool   `json:"installerOnly,omitempty"`
}

// githubRelease is a partial GitHub Releases API response.
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Prerelease  bool   `json:"prerelease"`
	Body        string `json:"body"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var httpClient = &http.Client{}

// Check queries GitHub Releases for a newer version. When includeBeta is
// true, the newest release on the beta channel (pre-releases included) is
// considered; otherwise only the latest stable release is.
func Check(ctx context.Context, currentVersion string, includeBeta bool) (*Info, error) {
	var releases []githubRelease
	url := repoAPI + "/releases/latest"
	if includeBeta {
		url = repoAPI + "/releases?per_page=15"
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update check failed: GitHub API returned %s", resp.Status)
	}

	if includeBeta {
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to parse releases response: %w", err)
		}
		if len(releases) == 0 {
			return nil, nil
		}
		// /releases returns newest first; the first is the candidate.
		candidate := releases[0]
		if extensions.CompareVersions(candidate.TagName, currentVersion) <= 0 {
			return nil, nil
		}
		return buildInfo(candidate)
	}

	var latest githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return nil, fmt.Errorf("failed to parse latest release response: %w", err)
	}
	if extensions.CompareVersions(latest.TagName, currentVersion) <= 0 {
		return nil, nil
	}
	return buildInfo(latest)
}

func buildInfo(r githubRelease) (*Info, error) {
	assetName := assetNameForPlatform()
	if assetName == "" {
		return nil, nil
	}
	for _, a := range r.Assets {
		if a.Name == assetName {
			return &Info{
				Version:      strings.TrimPrefix(r.TagName, "v"),
				AssetName:    a.Name,
				DownloadURL:  a.BrowserDownloadURL,
				ReleaseNotes: r.Body,
				IsPrerelease: r.Prerelease,
			}, nil
		}
	}
	return nil, nil
}

// assetNameForPlatform returns the release asset name for the current
// platform, or "" when the platform has no self-appliable asset.
func assetNameForPlatform() string {
	switch runtime.GOOS {
	case "linux":
		if os.Getenv("APPIMAGE") != "" {
			// AppImage installs must be updated with the AppImage asset.
			// Swapping a tar.gz over the AppImage path would leave a file
			// the kernel cannot execute.
			return fmt.Sprintf("Aether-%s-%s.AppImage", runtime.GOOS, runtime.GOARCH)
		}
		return "Aether-linux-amd64.tar.gz"
	case "windows":
		return "Aether-windows-amd64.exe"
	case "darwin":
		return "Aether-macos-" + runtime.GOARCH + ".dmg"
	}
	return ""
}

// Download fetches the update payload into a temp file, emitting
// update:progress events, and returns the local path.
func Download(ctx context.Context, info *Info) (string, error) {
	destPath := filepath.Join(os.TempDir(), "aether-update-"+info.AssetName)
	_ = os.Remove(destPath)

	var lastReport int64
	onProgress := func(downloaded, total int64) {
		if total > 0 && downloaded-lastReport > 2*1024*1024 {
			lastReport = downloaded
			wailsruntime.EventsEmit(ctx, "update:progress", map[string]interface{}{
				"progress": int(downloaded * 100 / total),
			})
		}
	}

	if err := netutil.DownloadFile(ctx, info.DownloadURL, destPath, onProgress); err != nil {
		return "", fmt.Errorf("update download failed: %w", err)
	}
	return destPath, nil
}

// Apply installs the downloaded update and relaunches the app. On success
// it never returns (the process is replaced).
func Apply(ctx context.Context, info *Info, downloadedPath string) error {
	switch runtime.GOOS {
	case "linux":
		return applyLinux(ctx, info, downloadedPath)
	case "windows":
		return applyWindows(ctx, info, downloadedPath)
	case "darwin":
		// DMGs cannot be applied in place; the caller opens them instead.
		return fmt.Errorf("macOS updates are applied by opening the downloaded disk image")
	}
	return fmt.Errorf("auto-update is not supported on %s", runtime.GOOS)
}

// applyLinux swaps the running binary. Under an AppImage ($APPIMAGE is set)
// the AppImage file itself is replaced; otherwise the tar.gz payload is
// extracted and the executable is swapped in place.
func applyLinux(ctx context.Context, info *Info, downloadedPath string) error {
	appImage := os.Getenv("APPIMAGE")
	if appImage != "" {
		if err := verifyAppImagePayload(downloadedPath); err != nil {
			return fmt.Errorf("refusing to replace the AppImage: %w", err)
		}
		if err := replaceFile(appImage, downloadedPath); err != nil {
			return err
		}
		if err := os.Chmod(appImage, 0755); err != nil {
			return err
		}
		return relaunch(appImage)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}

	extractDir, err := os.MkdirTemp("", "aether-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	if err := extractTarGz(downloadedPath, extractDir); err != nil {
		return err
	}
	newBin := filepath.Join(extractDir, "Aether")
	if err := os.Chmod(newBin, 0755); err != nil {
		return err
	}
	if err := replaceFile(exePath, newBin); err != nil {
		return err
	}
	return relaunch(exePath)
}

// verifyAppImagePayload rejects downloads that are not executable ELF images
// (e.g. a tar.gz that was fetched by mistake) so a broken file can never be
// swapped over a working AppImage.
func verifyAppImagePayload(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("cannot read downloaded payload: %w", err)
	}
	if magic[0] != 0x7f || magic[1] != 'E' || magic[2] != 'L' || magic[3] != 'F' {
		return fmt.Errorf("downloaded file is not an AppImage (bad payload: %x)", magic)
	}
	return nil
}

// applyWindows swaps the exe in place; if the location is not writable
// (e.g. Program Files), it falls back to running the silent NSIS installer.
func applyWindows(ctx context.Context, info *Info, downloadedPath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	// Try an in-place swap first.
	if swapErr := replaceFile(exePath, downloadedPath); swapErr == nil {
		return relaunch(exePath)
	}

	// Fall back to the silent installer. It is a separate asset, so fetch it.
	installerAsset := strings.Replace(info.AssetName, ".exe", "-installer.exe", 1)
	installerURL := repoDownload + "/v" + info.Version + "/" + installerAsset
	installerPath := filepath.Join(os.TempDir(), "aether-installer-"+installerAsset)
	_ = os.Remove(installerPath)

	if err := netutil.DownloadFile(ctx, installerURL, installerPath, nil); err != nil {
		return fmt.Errorf("in-place update failed and the installer could not be downloaded: %w", err)
	}

	// NSIS silent install: /S. The installer relaunches nothing itself, so
	// re-run the app afterwards.
	cmd := exec.Command(installerPath, "/S")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("silent installer failed: %w", err)
	}
	return relaunch(exePath)
}

// replaceFile replaces target with replacement. On Unix this is a single
// atomic rename(2) — the target is never missing, even momentarily. Windows
// uses a .old sibling dance because rename-over-existing can fail there.
func replaceFile(target, replacement string) error {
	if runtime.GOOS == "windows" {
		oldPath := target + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(target, oldPath); err != nil {
			return fmt.Errorf("cannot move existing binary (%w) — is the install location writable?", err)
		}
		if err := os.Rename(replacement, target); err != nil {
			// Try to restore the original.
			_ = os.Rename(oldPath, target)
			return fmt.Errorf("cannot install new binary: %w", err)
		}
		_ = os.Remove(oldPath)
		return nil
	}
	if err := os.Rename(replacement, target); err != nil {
		return fmt.Errorf("cannot install new binary: %w", err)
	}
	return nil
}

// relaunch starts a detached instance of the binary and exits the current
// process. It returns an error only if the spawn fails.
func relaunch(binPath string) error {
	cmd := exec.Command(binPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if runtime.GOOS == "windows" {
		// Detach on Windows so the parent can exit.
		cmd.SysProcAttr = detachedSysProcAttr()
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to relaunch the application: %w", err)
	}
	os.Exit(0)
	return nil
}

// extractTarGz extracts a tar.gz archive into destDir.
func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, destDir) {
			return fmt.Errorf("archive entry escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}
