package extensions

// Package extensions handles the Aether extension lifecycle


// Extension represents the UI representation of a Manifest
type Extension struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Memory      string `json:"memory"`
	CPU         string `json:"cpu"`
	Trust       string `json:"trust"`
	IconURL     string `json:"iconUrl,omitempty"`
	Reloading   bool   `json:"reloading"`
}

// GetExtensions returns all installed extensions
func GetExtensions() []Extension {
	if GlobalManager != nil {
		return GlobalManager.GetExtensions()
	}
	return []Extension{}
}
