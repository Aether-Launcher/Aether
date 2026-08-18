package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Aether/pkg/fs"
)

// Create creates a new instance folder and instance.json on disk
func Create(name, version, loader string) (*Instance, error) {
	// Sanitize name for folder ID
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	
	instancesDir := filepath.Join(fs.GetDataDir(), "instances")
	instancePath := filepath.Join(instancesDir, id)

	// Check if exists
	if stat, err := os.Stat(instancePath); err == nil && stat.IsDir() {
		return nil, fmt.Errorf("instance with ID '%s' already exists", id)
	}

	// Create directory structure (root + the well-known subdirs in one pass)
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return nil, err
	}
	for _, sub := range []string{"bin", "mods", "resourcepacks"} {
		if err := os.MkdirAll(filepath.Join(instancePath, sub), 0755); err != nil {
			return nil, err
		}
	}

	// Default memory allocation based on version (simplified)
	memory := "4G"

	inst := &Instance{
		ID:         id,
		Name:       name,
		Version:    version,
		Loader:     loader,
		Memory:     memory,
		LastPlayed: "Never",
		Installed:  false,
	}

	// Write instance.json
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filepath.Join(instancePath, "instance.json"), data, 0644)
	if err != nil {
		return nil, err
	}

	return inst, nil
}
