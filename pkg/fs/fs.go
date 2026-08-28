package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetDataDir returns the root data directory for Aether
func GetDataDir() string {
	// For dev mode, if an .aether folder exists locally, use it
	if stat, err := os.Stat(".aether"); err == nil && stat.IsDir() {
		abs, _ := filepath.Abs(".aether")
		return abs
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback if config dir is unresolvable
		return filepath.Join(".", "AetherData")
	}
	return filepath.Join(configDir, "Aether")
}

// GetAssetsDir returns the shared assets directory path
func GetAssetsDir() string {
	return filepath.Join(GetDataDir(), "assets")
}

// EnsureDirectories creates the required directory structure if it doesn't exist
func EnsureDirectories() error {
	base := GetDataDir()
	dirs := []string{
		filepath.Join(base, "instances"),
		filepath.Join(base, "extensions"),
		filepath.Join(base, "themes"),
		filepath.Join(base, "assets"),
		filepath.Join(base, "logs"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}
	return nil
}
