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

type sidebarPinConsent struct {
	Version  string `json:"version"`
	Approved bool   `json:"approved"`
}

func (m *Manager) sidebarConsentPath() string {
	return filepath.Join(fs.GetDataDir(), "extension-permissions.json")
}

func (m *Manager) loadSidebarConsents() error {
	data, err := os.ReadFile(m.sidebarConsentPath())
	if os.IsNotExist(err) {
		m.sidebarMu.Lock()
		m.sidebarConsents = make(map[string]sidebarPinConsent)
		m.sidebarMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	var consents map[string]sidebarPinConsent
	if err := json.Unmarshal(data, &consents); err != nil {
		return fmt.Errorf("parse extension permission state: %w", err)
	}
	if consents == nil {
		consents = make(map[string]sidebarPinConsent)
	}
	m.sidebarMu.Lock()
	m.sidebarConsents = consents
	m.sidebarMu.Unlock()
	return nil
}

// RequestSidebarPinConsent asks once for each installed extension version.
// A denial is also remembered for that version; a later update will ask again.
func (m *Manager) RequestSidebarPinConsent(manifest Manifest) error {
	if manifest.ID == "" || manifest.Version == "" || !manifest.PinToSidebar || !manifest.HasPermission("ui:sidebar") {
		return nil
	}

	m.sidebarMu.RLock()
	consent, found := m.sidebarConsents[manifest.ID]
	m.sidebarMu.RUnlock()
	if found && consent.Version == manifest.Version {
		return nil
	}

	approved := false
	if m.emit != nil {
		approved = m.requestConfirmation(map[string]interface{}{
			"permission":    "ui:sidebar",
			"extensionId":   manifest.ID,
			"extensionName": manifest.Name,
			"action":        "pin its pages in the sidebar",
		})
	}

	m.sidebarMu.Lock()
	previous, hadPrevious := m.sidebarConsents[manifest.ID]
	m.sidebarConsents[manifest.ID] = sidebarPinConsent{Version: manifest.Version, Approved: approved}
	data, err := json.MarshalIndent(m.sidebarConsents, "", "  ")
	if err == nil {
		err = os.MkdirAll(fs.GetDataDir(), 0755)
	}
	if err == nil {
		err = os.WriteFile(m.sidebarConsentPath(), data, 0600)
	}
	if err != nil {
		if hadPrevious {
			m.sidebarConsents[manifest.ID] = previous
		} else {
			delete(m.sidebarConsents, manifest.ID)
		}
	}
	m.sidebarMu.Unlock()
	if err != nil {
		return fmt.Errorf("save sidebar permission: %w", err)
	}

	m.audit("sidebar_pin_consent", map[string]interface{}{
		"extensionId": manifest.ID,
		"version":     manifest.Version,
		"approved":    approved,
	})
	return nil
}

func (m *Manager) hasSidebarPinConsent(manifest Manifest) bool {
	if !manifest.PinToSidebar || !manifest.HasPermission("ui:sidebar") {
		return false
	}
	m.sidebarMu.RLock()
	defer m.sidebarMu.RUnlock()
	consent, ok := m.sidebarConsents[manifest.ID]
	return ok && consent.Version == manifest.Version && consent.Approved
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
