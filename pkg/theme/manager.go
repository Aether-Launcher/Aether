package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"Aether/pkg/fs"
)

// GlobalServer is the singleton local file server used to expose installed
// themes' PNG assets to the frontend (an <img> tag can't read from disk
// directly, so it needs an http:// URL, same approach as extensions/server.go).
var GlobalServer = NewServer()

// List returns metadata for every installed theme, in no particular order.
func List(activeID string) ([]Info, error) {
	themesDir := getThemesDir()
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Info{}, nil
		}
		return nil, err
	}

	baseURL := GlobalServer.URL()

	infos := make([]Info, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := readManifest(filepath.Join(themesDir, entry.Name()))
		if err != nil {
			continue
		}
		info := Info{
			ID:          manifest.ID,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Author:      manifest.Author,
			Description: manifest.Description,
			Active:      manifest.ID == activeID,
		}
		if manifest.Icon != "" && baseURL != "" {
			if p, err := fs.ContainedPath(filepath.Join(themesDir, entry.Name()), manifest.Icon); err == nil {
				if _, statErr := os.Stat(p); statErr == nil {
					info.IconURL = baseURL + "/" + manifest.ID + "/" + manifest.Icon
				}
			}
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// GetCSS returns the sanitized CSS of the theme with the given ID, or "" if
// the theme doesn't exist / no theme is active.
func GetCSS(id string) string {
	if id == "" {
		return ""
	}
	themeDir, err := fs.ContainedPath(getThemesDir(), id)
	if err != nil {
		return ""
	}
	manifest, err := readManifest(themeDir)
	if err != nil {
		return ""
	}
	cssPath, err := fs.ContainedPath(themeDir, manifest.CSS)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetAssetURLs returns the key→URL map of PNG overrides for the theme with
// the given ID, using the local asset server. Only keys present in
// AllowedAssetKeys can ever appear here (installer.go already filtered
// overwrite.json down to that set, but we double-check here too).
func GetAssetURLs(id string) map[string]string {
	result := map[string]string{}
	if id == "" {
		return result
	}
	baseURL := GlobalServer.URL()
	if baseURL == "" {
		return result
	}
	themeDir, err := fs.ContainedPath(getThemesDir(), id)
	if err != nil {
		return result
	}
	manifest, err := readManifest(themeDir)
	if err != nil {
		return result
	}
	overwritePath, err := fs.ContainedPath(themeDir, manifest.Overwrite)
	if err != nil {
		return result
	}
	data, err := os.ReadFile(overwritePath)
	if err != nil {
		return result
	}
	var mapping map[string]string
	if err := json.Unmarshal(data, &mapping); err != nil {
		return result
	}
	for key, filename := range mapping {
		if !AllowedAssetKeys[key] || LockedAssetKeys[key] {
			continue
		}
		if _, err := fs.ContainedPath(themeDir, filename); err != nil {
			continue
		}
		result[key] = baseURL + "/" + id + "/" + filename
	}
	return result
}

func readManifest(themeDir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(themeDir, "package.json"))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	manifest.applyDefaults()
	if manifest.ID == "" {
		manifest.ID = filepath.Base(themeDir)
	}
	return &manifest, nil
}
