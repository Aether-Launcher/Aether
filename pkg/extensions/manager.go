package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Aether/pkg/fs"
	"Aether/pkg/instance"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Manager handles the lifecycle of all extensions
type Manager struct {
	ctx              context.Context
	serverURL        string
	LoadedExtensions map[string]Extension
	sandboxes        map[string]*Sandbox
	SidebarPages     []map[string]interface{}
	ModLoaders       map[string]ModLoaderConfig
	pending          map[string]chan bool
	pendingMu        sync.Mutex
	auditMu          sync.Mutex
}

// ModLoaderConfig holds info about a registered mod loader
type ModLoaderConfig struct {
	ID          string
	Name        string
	Description string
	ExtensionID string
	Callback    func(ctx map[string]interface{}) (map[string]interface{}, error)
}

const maxExtensionModSize int64 = 100 * 1024 * 1024

// GlobalManager holds the singleton instance for UI access
var GlobalManager *Manager

// NewManager creates a new extension manager
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		ctx:              ctx,
		LoadedExtensions: make(map[string]Extension),
		sandboxes:        make(map[string]*Sandbox),
		SidebarPages:     make([]map[string]interface{}, 0),
		ModLoaders:       make(map[string]ModLoaderConfig),
		pending:          make(map[string]chan bool),
	}
}

// LoadAll scans the directory, parses manifests, and executes the JS isolate
func (m *Manager) LoadAll() error {
	// Reloading replaces the in-memory view so removed extensions do not linger.
	clearModLoaderCallbackCache("")
	m.LoadedExtensions = make(map[string]Extension)
	m.sandboxes = make(map[string]*Sandbox)
	m.SidebarPages = make([]map[string]interface{}, 0)
	m.ModLoaders = make(map[string]ModLoaderConfig)

	// Start the local extension server
	server := NewServer()
	url, err := server.Start()
	if err != nil {
		fmt.Printf("[Extensions] Warning: Failed to start UI server: %v\n", err)
	}
	m.serverURL = url
	extDir := filepath.Join(fs.GetDataDir(), "extensions")
	entries, err := os.ReadDir(extDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			manifestPath := filepath.Join(extDir, entry.Name(), "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}

			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				continue
			}

			if manifest.ID == "" {
				manifest.ID = entry.Name()
			}

			// Determine trust level: Check against gallery API mock
			trust := "local"
			for _, ge := range GetGalleryExtensions() {
				if ge.ID == manifest.ID {
					trust = ge.Trust
					break
				}
			}

			iconUrl := ""
			if manifest.Icon != "" {
				iconUrl = fmt.Sprintf("%s/%s/%s", m.serverURL, manifest.ID, manifest.Icon)
			}

			ext := Extension{
				ID:          manifest.ID,
				Name:        manifest.Name,
				Version:     manifest.Version,
				Author:      manifest.Author,
				Description: manifest.Description,
				Status:      "Running",
				Memory:      "0MB",
				CPU:         "0%",
				Trust:       trust,
				IconURL:     iconUrl,
			}

			sandbox := NewSandbox(
				m.ctx, manifest, m.serverURL,
				func(payload map[string]interface{}) {
					m.SidebarPages = append(m.SidebarPages, payload)
				},
				func(config ModLoaderConfig) {
					m.ModLoaders[config.ID] = config
				},
				func() []InstanceInfo {
					all := instance.GetInstances()
					var out []InstanceInfo
					for _, inst := range all {
						out = append(out, InstanceInfo{
							ID:      inst.ID,
							Name:    inst.Name,
							Version: inst.Version,
							Loader:  inst.Loader,
						})
					}
					return out
				},
				func(instanceID, jarName, downloadURL string) (string, error) {
					jarName = filepath.Base(jarName)
					if strings.ToLower(filepath.Ext(jarName)) != ".jar" {
						return "", fmt.Errorf("mod file must have a .jar extension")
					}
					parsedURL, err := neturl.Parse(downloadURL)
					if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" {
						return "", fmt.Errorf("mod downloads require an HTTPS URL")
					}
					instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
					if err != nil {
						return "", err
					}
					modsDir := filepath.Join(instanceDir, "mods")
					if err := os.MkdirAll(modsDir, 0755); err != nil {
						return "", err
					}
					destPath := filepath.Join(modsDir, jarName)
					req, err := http.NewRequestWithContext(m.ctx, http.MethodGet, downloadURL, nil)
					if err != nil {
						return "", err
					}
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return "", err
					}
					defer resp.Body.Close()
					if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
						return "", fmt.Errorf("mod download failed with status %s", resp.Status)
					}
					if resp.ContentLength > maxExtensionModSize {
						return "", fmt.Errorf("mod exceeds the %d MB size limit", maxExtensionModSize/(1024*1024))
					}
					out, err := os.Create(destPath)
					if err != nil {
						return "", err
					}
					defer out.Close()
					written, err := io.Copy(out, io.LimitReader(resp.Body, maxExtensionModSize+1))
					if err != nil {
						_ = os.Remove(destPath)
						return "", err
					}
					if written > maxExtensionModSize {
						_ = os.Remove(destPath)
						return "", fmt.Errorf("mod exceeds the %d MB size limit", maxExtensionModSize/(1024*1024))
					}
					return destPath, nil
				},
				func(instanceID string) ([]string, error) {
					instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
					if err != nil {
						return nil, err
					}
					modsDir := filepath.Join(instanceDir, "mods")
					entries, err := os.ReadDir(modsDir)
					if err != nil {
						if os.IsNotExist(err) {
							return []string{}, nil
						}
						return nil, err
					}
					var mods []string
					for _, e := range entries {
						if !e.IsDir() {
							mods = append(mods, e.Name())
						}
					}
					return mods, nil
				},
				func(instanceID, jarName string) error {
					jarName = filepath.Base(jarName)
					instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
					if err != nil {
						return err
					}
					modPath := filepath.Join(instanceDir, "mods", jarName)
					return os.Remove(modPath)
				},
				func(instanceID, jarName string, enable bool) error {
					jarName = filepath.Base(jarName)
					instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
					if err != nil {
						return err
					}
					modsDir := filepath.Join(instanceDir, "mods")

					currentPath := filepath.Join(modsDir, jarName)

					if enable {
						// We want to enable it. It must currently end in .disabled
						if strings.HasSuffix(jarName, ".disabled") {
							newPath := filepath.Join(modsDir, strings.TrimSuffix(jarName, ".disabled"))
							return os.Rename(currentPath, newPath)
						}
						// Already enabled
						return nil
					} else {
						// We want to disable it.
						if !strings.HasSuffix(jarName, ".disabled") {
							newPath := filepath.Join(modsDir, jarName+".disabled")
							return os.Rename(currentPath, newPath)
						}
						// Already disabled
						return nil
					}
				},
				runtime.EventsEmit,
				m.requestConfirmation,
			)
			m.sandboxes[manifest.ID] = sandbox

			if manifest.Main != "" {
				scriptPath := filepath.Join(extDir, entry.Name(), manifest.Main)
				scriptData, err := os.ReadFile(scriptPath)
				if err == nil {
					if err := sandbox.Execute(string(scriptData)); err != nil {
						fmt.Printf("[Manager] Failed to execute %s for %s: %v\n", manifest.Main, manifest.ID, err)
						ext.Status = "Error"
					} else {
						fmt.Printf("[Manager] Successfully loaded extension isolate: %s\n", manifest.ID)
					}
				} else {
					fmt.Printf("[Manager] Missing main script %s for %s\n", manifest.Main, manifest.ID)
					ext.Status = "Error (Missing Main)"
				}
			}

			m.LoadedExtensions[manifest.ID] = ext
		}
	}
	return nil
}

// requestConfirmation pauses a sensitive extension operation until the UI
// responds or the request expires.
func (m *Manager) requestConfirmation(action map[string]interface{}) bool {
	id := fmt.Sprintf("extension-confirm-%d", time.Now().UnixNano())
	response := make(chan bool, 1)
	m.pendingMu.Lock()
	m.pending[id] = response
	m.pendingMu.Unlock()

	action["requestId"] = id
	m.audit("confirmation_requested", action)
	runtime.EventsEmit(m.ctx, "extension:confirmation", action)

	select {
	case approved := <-response:
		return approved
	case <-time.After(2 * time.Minute):
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
		m.audit("confirmation_expired", map[string]interface{}{"requestId": id})
		return false
	}
}

// ResolveConfirmation completes a pending extension operation from the UI.
func (m *Manager) ResolveConfirmation(requestID string, approved bool) error {
	m.pendingMu.Lock()
	response, ok := m.pending[requestID]
	if ok {
		delete(m.pending, requestID)
	}
	m.pendingMu.Unlock()
	if !ok {
		return fmt.Errorf("extension confirmation request not found: %s", requestID)
	}
	response <- approved
	m.audit("confirmation_resolved", map[string]interface{}{
		"requestId": requestID,
		"approved":  approved,
	})
	return nil
}

func (m *Manager) audit(event string, details map[string]interface{}) {
	record := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}
	for key, value := range details {
		record[key] = value
	}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}

	m.auditMu.Lock()
	defer m.auditMu.Unlock()
	logPath := filepath.Join(fs.GetDataDir(), "logs", "extension-security.log")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
}

// Uninstall removes an extension directory after validating its ID.
func (m *Manager) Uninstall(id string) error {
	if id == "" {
		return fmt.Errorf("extension ID cannot be empty")
	}
	extDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "extensions"), id)
	if err != nil {
		return err
	}
	clearModLoaderCallbackCache(id)
	if err := os.RemoveAll(extDir); err != nil {
		return err
	}
	return m.LoadAll()
}

// GetExtensions returns the list of loaded extensions
func (m *Manager) GetExtensions() []Extension {
	var list []Extension
	for _, ext := range m.LoadedExtensions {
		list = append(list, ext)
	}
	return list
}

// HandleIPCMessage routes a message from the UI iframe to the extension sandbox
func (m *Manager) HandleIPCMessage(extID string, payload map[string]interface{}) {
	sandbox, ok := m.sandboxes[extID]
	if !ok {
		fmt.Printf("[Manager] IPC message for unknown extension: %s\n", extID)
		return
	}

	// Sandbox invocations are serialized per extension and panic-recovered so
	// a busy or broken extension cannot leave the UI hanging forever.
	result, err := sandbox.InvokeMessage(payload)
	if err != nil {
		fmt.Printf("[Manager] IPC callback error for %s: %v\n", extID, err)
		// Reply with an error carrying the requestId so the iframe's pending
		// promise resolves instead of showing "Loading..." indefinitely.
		response := map[string]interface{}{"error": err.Error()}
		if requestID, ok := payload["requestId"]; ok {
			response["requestId"] = requestID
		}
		runtime.EventsEmit(m.ctx, "extension:message:"+extID, response)
		return
	}

	// Emit the response back to the frontend as a Wails event
	runtime.EventsEmit(m.ctx, "extension:message:"+extID, result)
}

// GetSidebarPages returns all registered sidebar pages
func (m *Manager) GetSidebarPages() []map[string]interface{} {
	return m.SidebarPages
}
