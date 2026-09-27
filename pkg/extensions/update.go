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

// CompareVersions compares two version strings.
// It returns 1 if a > b, -1 if a < b, and 0 if they are equal.
// Supports semantic versioning including pre-release tags (e.g. "1.0.0-beta.15" vs "1.0.0-beta.14").
func CompareVersions(a, b string) int {
	type semver struct {
		nums   []int
		hasPre bool
		preTag string
		preNum int
	}

	parse := func(v string) semver {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		var s semver
		if idx := strings.IndexByte(v, '-'); idx != -1 {
			pre := v[idx+1:]
			v = v[:idx]
			s.hasPre = true
			parts := strings.Split(pre, ".")
			if len(parts) > 1 {
				if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					s.preNum = n
					s.preTag = strings.Join(parts[:len(parts)-1], ".")
				} else {
					s.preTag = pre
				}
			} else {
				s.preTag = pre
			}
		}

		for _, p := range strings.Split(v, ".") {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				break
			}
			s.nums = append(s.nums, n)
		}
		return s
	}

	sa, sb := parse(a), parse(b)
	maxLen := len(sa.nums)
	if len(sb.nums) > maxLen {
		maxLen = len(sb.nums)
	}
	for i := 0; i < maxLen; i++ {
		var x, y int
		if i < len(sa.nums) {
			x = sa.nums[i]
		}
		if i < len(sb.nums) {
			y = sb.nums[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}

	// Main versions are equal; compare pre-releases.
	// A stable release (no pre-release) is greater than a pre-release.
	if !sa.hasPre && sb.hasPre {
		return 1
	}
	if sa.hasPre && !sb.hasPre {
		return -1
	}
	if sa.hasPre && sb.hasPre {
		if sa.preTag != sb.preTag {
			return strings.Compare(sa.preTag, sb.preTag)
		}
		if sa.preNum != sb.preNum {
			if sa.preNum > sb.preNum {
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
