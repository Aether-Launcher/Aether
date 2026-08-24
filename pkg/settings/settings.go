package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"Aether/pkg/fs"
)

// GlobalSettings holds launcher-wide configuration
type GlobalSettings struct {
	DefaultMemory      string `json:"defaultMemory"`
	CloseOnLaunch      bool   `json:"closeOnLaunch"`
	DeveloperMode      bool   `json:"developerMode"`
	DisableExtensions  bool   `json:"disableExtensions"`
	GarbageCollector   string `json:"garbageCollector,omitempty"`
	CustomJVMArgs      string `json:"customJvmArgs,omitempty"`
	AutoCheckUpdates   bool   `json:"autoCheckUpdates"`
	IncludeBetaUpdates bool   `json:"includeBetaUpdates"`
}

// GetDefaultSettings returns the default configuration
func GetDefaultSettings() GlobalSettings {
	return GlobalSettings{
		DefaultMemory:      "4096",
		CloseOnLaunch:      false,
		DeveloperMode:      false,
		DisableExtensions:  false,
		GarbageCollector:   "G1GC",
		CustomJVMArgs:      "",
		AutoCheckUpdates:   true,
		IncludeBetaUpdates: false,
	}
}

// getSettingsPath returns the path to the settings JSON file
func getSettingsPath() string {
	return filepath.Join(fs.GetDataDir(), "settings.json")
}

// Load reads the settings from disk or returns defaults if not found/invalid
func Load() GlobalSettings {
	path := getSettingsPath()
	
	data, err := os.ReadFile(path)
	if err != nil {
		return GetDefaultSettings()
	}

	var s GlobalSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return GetDefaultSettings()
	}

	// Basic validation / migration
	if s.DefaultMemory == "" {
		s.DefaultMemory = "4096"
	}

	return s
}

// Save writes the settings to disk atomically
func Save(s GlobalSettings) error {
	path := getSettingsPath()
	
	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
