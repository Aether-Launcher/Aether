package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"Aether/pkg/fs"
)

var validIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,64}$`)
var windowsReserved = map[string]bool{"con": true, "prn": true, "aux": true, "nul": true, "com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true, "lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true}

// sanitizeID normalizes the display name into a filesystem-safe instance ID.
func sanitizeID(name string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(name))
	if id == "" {
		return "", fmt.Errorf("instance name must not be empty")
	}
	// Replace any run of characters outside the allow-list with a single dash.
	// Allow-list: a-z, 0-9, dot, underscore, hyphen (validated by validIDRe).
	invalidSeq := regexp.MustCompile(`[^a-z0-9._-]+`)
	id = invalidSeq.ReplaceAllString(id, "-")
	// Collapse multiple dashes and trim leading/trailing separators.
	id = regexp.MustCompile(`-+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-._")
	if id == "" {
		return "", fmt.Errorf("instance name must not be empty")
	}
	// Must start with alphanumeric
	if id[0] == '.' || id[0] == '_' || id[0] == '-' {
		id = "a" + id
	}
	// Truncate to max length for validIDRe (1 + 64 chars)
	if len(id) > 65 {
		id = id[:65]
		id = strings.TrimRight(id, "-._")
	}
	if !validIDRe.MatchString(id) {
		return "", fmt.Errorf("instance id %q contains invalid characters", id)
	}
	if windowsReserved[id] {
		return "", fmt.Errorf("instance id %q is reserved", id)
	}
	return id, nil
}

// Create creates a new instance folder and instance.json on disk
func Create(name, version, loader string) (*Instance, error) {
	id, err := sanitizeID(name)
	if err != nil {
		return nil, err
	}

	instancesDir := filepath.Join(fs.GetDataDir(), "instances")
	instancePath, err := fs.ContainedPath(instancesDir, id)
	if err != nil {
		return nil, err
	}

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

	// Write instance.json atomically
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return nil, err
	}

	tmpPath := filepath.Join(instancePath, "instance.json.tmp")
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, filepath.Join(instancePath, "instance.json")); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	return inst, nil
}
