package extensions

import (
	"Aether/pkg/netutil"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const galleryIndexURL = "https://raw.githubusercontent.com/wayback09/Aether-Extensions/main/index.json"

// GalleryExtension represents an extension in the Aether Registry.
// Trust tier is assigned by the Aether team in the registry — never by the extension itself.
type GalleryExtension struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Trust       string `json:"trust"`
	URL         string `json:"url"`
}

var (
	galleryCache    []GalleryExtension
	galleryCacheAt  time.Time
	galleryCacheMu  sync.Mutex
	galleryCacheTTL = 5 * time.Minute
)

// GetGalleryExtensions returns the live registry from GitHub, with a 5-minute in-memory cache.
// Gracefully falls back to an empty slice if the user is offline or GitHub is unreachable.
func GetGalleryExtensions() []GalleryExtension {
	galleryCacheMu.Lock()
	defer galleryCacheMu.Unlock()

	if galleryCache != nil && time.Since(galleryCacheAt) < galleryCacheTTL {
		return galleryCache
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(galleryIndexURL)
	if err != nil {
		fmt.Printf("[Gallery] Could not fetch registry (offline?): %v\n", err)
		return galleryCache // return stale cache rather than nothing
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[Gallery] Registry returned HTTP %d\n", resp.StatusCode)
		return galleryCache
	}

	var entries []GalleryExtension
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		fmt.Printf("[Gallery] Failed to parse registry: %v\n", err)
		return galleryCache
	}

	galleryCache = entries
	galleryCacheAt = time.Now()
	fmt.Printf("[Gallery] Fetched %d extensions from registry\n", len(entries))
	return galleryCache
}

// DownloadAndInstallExtension downloads an extension package from a trusted Registry URL and installs it.
func DownloadAndInstallExtension(url string) error {
	if err := validateGalleryDownloadURL(url); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "aether-ext-*.aex")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpName)       // remove the 0-byte file so DownloadFile doesn't skip it
	defer os.Remove(tmpName) // clean up the downloaded file after installation

	if err := netutil.DownloadFile(context.Background(), url, tmpName, nil); err != nil {
		return fmt.Errorf("failed to download extension: %w", err)
	}

	return InstallFromArchive(tmpName)
}

// validateGalleryDownloadURL only permits packages explicitly published by an
// approved Gallery registry entry. Keeping this check in the backend ensures
// callers cannot bypass the Gallery UI and download an arbitrary URL.
func validateGalleryDownloadURL(downloadURL string) error {
	u, err := neturl.Parse(downloadURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || strings.ContainsAny(downloadURL, "\r\n") {
		return fmt.Errorf("extension source is not a trusted Gallery URL")
	}

	for _, entry := range GetGalleryExtensions() {
		if entry.URL == downloadURL && approvedGalleryTrust(entry.Trust) {
			return nil
		}
	}
	return fmt.Errorf("extension source is not an approved Gallery registry entry")
}

func approvedGalleryTrust(trust string) bool {
	switch strings.ToLower(strings.TrimSpace(trust)) {
	case "official", "verified", "community":
		return true
	default:
		return false

	}
}
