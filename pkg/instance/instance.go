package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Aether/pkg/fs"
)

type Instance struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Loader     string `json:"loader"`
	Memory     string `json:"memory"`
	LastPlayed string `json:"lastPlayed"`
	Installed  bool   `json:"installed"`
}

// GetInstances returns a list of instances parsed from the disk
func GetInstances() []Instance {
	instancesDir := filepath.Join(fs.GetDataDir(), "instances")
	instances := []Instance{} // Initialize as empty slice, not nil

	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return instances
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wg.Add(1)
		go func(entry os.DirEntry) {
			defer wg.Done()

			manifestPath := filepath.Join(instancesDir, entry.Name(), "instance.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return
			}

			var inst Instance
			if err := json.Unmarshal(data, &inst); err != nil {
				return
			}

			if inst.ID == "" {
				inst.ID = entry.Name()
			}
			// Check if client.jar exists
			jarPath := filepath.Join(instancesDir, entry.Name(), "bin", inst.Version+".jar")
			if _, err := os.Stat(jarPath); err == nil {
				inst.Installed = true
			}

			mu.Lock()
			instances = append(instances, inst)
			mu.Unlock()
		}(entry)
	}

	wg.Wait()
	return instances
}

// GetActiveInstance returns the first loaded instance
func GetActiveInstance() *Instance {
	instances := GetInstances()
	if len(instances) > 0 {
		return &instances[0]
	}
	return nil
}

// UpdateInstance saves the modified instance data to disk atomically
func UpdateInstance(inst *Instance) error {
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), inst.ID)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(instanceDir, "instance.json")
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return err
	}
	tmp := manifestPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, manifestPath)
}

// DeleteInstance permanently removes an instance directory from disk
func DeleteInstance(id string) error {
	if id == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), id)
	if err != nil {
		return err
	}
	return os.RemoveAll(instanceDir)
}
