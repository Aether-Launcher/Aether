package theme

// Manifest represents the structure of a theme's package.json.
// A .theme file is just a zip archive (renamed) containing this manifest
// alongside a CSS overwrite file and, optionally, an overwrite.json asset map.
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	// CSS is the path (relative to the package root) to the stylesheet that
	// gets injected after Aether's base styles. Defaults to "theme.css".
	CSS string `json:"css,omitempty"`

	// Overwrite is the path to the JSON file mapping asset keys to PNG
	// filenames within the package. Defaults to "overwrite.json".
	Overwrite string `json:"overwrite,omitempty"`

	// MinAetherVersion specifies the minimum launcher version required for
	// this theme. If set and the running launcher is older, the installer
	// will emit a warning but will not block installation. Format: semver
	// (e.g., "1.2.0"). Optional.
	MinAetherVersion string `json:"minAetherVersion,omitempty"`
}

// applyDefaults fills in optional fields with their conventional filenames.
func (m *Manifest) applyDefaults() {
	if m.CSS == "" {
		m.CSS = "theme.css"
	}
	if m.Overwrite == "" {
		m.Overwrite = "overwrite.json"
	}
}

// Info is the metadata returned to the frontend for display purposes.
type Info struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
	Active      bool   `json:"active"`
}
