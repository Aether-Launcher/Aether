package extensions

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ExtensionUpdate describes an available update for an installed extension.
type ExtensionUpdate struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CurrentVersion string `json:"currentVersion"`
	NewVersion     string `json:"newVersion"`
	URL            string `json:"url"`
}

// CompareVersions compares two dot-separated numeric version strings.
// It returns 1 if a > b, -1 if a < b, and 0 if they are equal.
// Non-numeric suffixes are ignored (e.g. "1.0.0-beta" parses as 1.0.0).
func CompareVersions(a, b string) int {
	parts := func(v string) []int {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		raw := strings.Split(v, ".")
		out := make([]int, 0, len(raw))
		for _, p := range raw {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				break
			}
			out = append(out, n)
		}
		return out
	}

	aa, bb := parts(a), parts(b)
	maxLen := len(aa)
	if len(bb) > maxLen {
		maxLen = len(bb)
	}
	for i := 0; i < maxLen; i++ {
		var x, y int
		if i < len(aa) {
			x = aa[i]
		}
		if i < len(bb) {
			y = bb[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}

// CheckForUpdates compares installed extensions against the registry and
// returns any newer versions that are available. It is a no-op when offline
// or when the registry is unreachable.
func CheckForUpdates() []ExtensionUpdate {
	gallery := GetGalleryExtensions()
	byID := make(map[string]GalleryExtension, len(gallery))
	for _, ge := range gallery {
		byID[ge.ID] = ge
	}

	var updates []ExtensionUpdate
	for _, ext := range GetExtensions() {
		ge, ok := byID[ext.ID]
		if !ok {
			continue
		}
		if CompareVersions(ge.Version, ext.Version) > 0 {
			updates = append(updates, ExtensionUpdate{
				ID:             ge.ID,
				Name:           ge.Name,
				CurrentVersion: ext.Version,
				NewVersion:     ge.Version,
				URL:            ge.URL,
			})
		}
	}
	sort.Slice(updates, func(i, j int) bool {
		return strings.ToLower(updates[i].Name) < strings.ToLower(updates[j].Name)
	})
	return updates
}

// UpdateExtension downloads and installs the newest registry version of the
// given extension, replacing the installed copy, then triggers an async reload.
func UpdateExtension(id string) (ExtensionUpdate, error) {
	for _, update := range CheckForUpdates() {
		if update.ID == id {
			if err := DownloadAndInstallExtension(update.URL); err != nil {
				return ExtensionUpdate{}, fmt.Errorf("failed to download update for '%s': %w", id, err)
			}
			if GlobalManager != nil {
				GlobalManager.ReloadAsync()
			}
			return update, nil
		}
	}
	return ExtensionUpdate{}, fmt.Errorf("no update available for '%s'", id)
}
