package extensions

// Manifest represents the structure of an extension's manifest.json
type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Icon        string   `json:"icon,omitempty"`
	Main               string   `json:"main"`
	API                string   `json:"api"`
	MinLauncherVersion string   `json:"minLauncherVersion,omitempty"`
	Permissions        []string `json:"permissions"`
	Hosts              []string `json:"hosts,omitempty"`
}

// HasPermission checks if the extension has requested a specific capability
func (m *Manifest) HasPermission(perm string) bool {
	for _, p := range m.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// HasAnyPermission reports whether the manifest grants one of the requested
// capabilities. The legacy instances:patch permission remains supported as a
// migration path for existing extensions.
func (m *Manifest) HasAnyPermission(permissions ...string) bool {
	for _, permission := range permissions {
		if m.HasPermission(permission) {
			return true
		}
	}
	return m.HasPermission("instances:patch") && hasInstancePatchCapability(permissions...)
}

func hasInstancePatchCapability(permissions ...string) bool {
	for _, permission := range permissions {
		switch permission {
		case "instances:list", "mods:list", "mods:install", "mods:delete", "mods:toggle":
			return true
		}
	}
	return false
}
