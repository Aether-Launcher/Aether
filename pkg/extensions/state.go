package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Aether-Launcher/Aether/pkg/fs"
)

func (m *Manager) extensionStatePath() string {
	return filepath.Join(fs.GetDataDir(), "extension-state.json")
}

func (m *Manager) loadDisabledExtensions() error {
	data, err := os.ReadFile(m.extensionStatePath())
	if os.IsNotExist(err) {
		m.disabledMu.Lock()
		m.disabledIDs = make(map[string]bool)
		m.disabledMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	var ids map[string]bool
	if err := json.Unmarshal(data, &ids); err != nil {
		return fmt.Errorf("parse extension state: %w", err)
	}
	if ids == nil {
		ids = make(map[string]bool)
	}
	m.disabledMu.Lock()
	m.disabledIDs = ids
	m.disabledMu.Unlock()
	return nil
}

func (m *Manager) isDisabled(id string) bool {
	m.disabledMu.RLock()
	defer m.disabledMu.RUnlock()
	return m.disabledIDs[id]
}

// SetEnabled persists an extension's enabled state and reloads extension sandboxes.
func (m *Manager) SetEnabled(id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("extension ID cannot be empty")
	}
	if _, ok := m.LoadedExtensions[id]; !ok {
		return fmt.Errorf("extension not found: %s", id)
	}

	m.disabledMu.Lock()
	previous, hadPrevious := m.disabledIDs[id]
	if enabled {
		delete(m.disabledIDs, id)
	} else {
		m.disabledIDs[id] = true
	}
	data, err := json.MarshalIndent(m.disabledIDs, "", "  ")
	if err == nil {
		dataDir := fs.GetDataDir()
		err = os.MkdirAll(dataDir, 0755)
		if err == nil {
			err = os.WriteFile(m.extensionStatePath(), data, 0600)
		}
	}
	if err != nil {
		if hadPrevious {
			m.disabledIDs[id] = previous
		} else {
			delete(m.disabledIDs, id)
		}
		m.disabledMu.Unlock()
		return fmt.Errorf("save extension state: %w", err)
	}
	m.disabledMu.Unlock()

	return m.ReloadAsync()
}
