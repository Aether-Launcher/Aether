package extensions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Aether/pkg/fs"
	"Aether/pkg/instance"
	"Aether/pkg/mojang"
)

// Manager handles the lifecycle of all extensions
type Manager struct {
	ctx              context.Context
	server           *Server
	serverURL        string
	LoadedExtensions map[string]Extension
	sandboxes        map[string]*Sandbox
	SidebarPages     []map[string]interface{}
	ModLoaders       map[string]ModLoaderConfig
	pending          map[string]chan bool
	pendingMu        sync.Mutex
	auditMu          sync.Mutex
	reloadMu         sync.Mutex
	reloading        bool
	reloadDone       chan error
	emit             func(context.Context, string, ...interface{})
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
// If emit is nil, event broadcasting is disabled (useful for tests).
func NewManager(ctx context.Context, emit func(context.Context, string, ...interface{})) *Manager {
	return &Manager{
		ctx:              ctx,
		server:           NewServer(),
		LoadedExtensions: make(map[string]Extension),
		sandboxes:        make(map[string]*Sandbox),
		SidebarPages:     make([]map[string]interface{}, 0),
		ModLoaders:       make(map[string]ModLoaderConfig),
		pending:          make(map[string]chan bool),
		emit:             emit,
	}
}

// LoadAll scans the directory, parses manifests, and updates metadata quickly without blocking.
// Sandboxes are reloaded in the background via ReloadAsync.
func (m *Manager) LoadAll() error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()

	clearModLoaderCallbackCache("")
	m.LoadedExtensions = make(map[string]Extension)
	m.SidebarPages = make([]map[string]interface{}, 0)
	m.ModLoaders = make(map[string]ModLoaderConfig)

	// Ensure local extension server is running (reuses port if already started)
	url, err := m.server.Start()
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

			m.LoadedExtensions[manifest.ID] = Extension{
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
				Reloading:   true, // Mark as loading until sandboxes complete
			}
		}
	}

	// Trigger background sandbox reload
	go m.reloadSandboxes()

	return nil
}

// ReloadAsync triggers a non-blocking full reload of all extensions and sandboxes.
func (m *Manager) ReloadAsync() error {
	if !m.reloadMu.TryLock() {
		return fmt.Errorf("extension reload already in progress")
	}
	m.reloadMu.Unlock()

	if m.emit != nil {
		m.emit(m.ctx, "extension:reload:start", nil)
	}
	return m.LoadAll()
}

// reloadSandboxes creates sandboxes and executes main scripts in the background
func (m *Manager) reloadSandboxes() {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()

	m.reloading = true
	defer func() {
		m.reloading = false
		if m.emit != nil {
		m.emit(m.ctx, "extension:reload:complete", nil)
	}
	}()

	newSandboxes := make(map[string]*Sandbox)
	newSidebarPages := make([]map[string]interface{}, 0)
	newModLoaders := make(map[string]ModLoaderConfig)
	extDir := filepath.Join(fs.GetDataDir(), "extensions")

	for id, ext := range m.LoadedExtensions {
		manifestPath := filepath.Join(extDir, id, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			ext.Status = "Error"
			ext.Reloading = false
			m.LoadedExtensions[id] = ext
			continue
		}
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			ext.Status = "Error"
			ext.Reloading = false
			m.LoadedExtensions[id] = ext
			continue
		}
		if manifest.ID == "" {
			manifest.ID = id
		}

		sandbox := NewSandbox(
			m.ctx, manifest, m.serverURL,
			func(payload map[string]interface{}) {
				newSidebarPages = append(newSidebarPages, payload)
			},
			func(config ModLoaderConfig) {
				newModLoaders[config.ID] = config
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

				doDownload := func(targetURL string) error {
					req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)
					if err != nil {
						return err
					}
					req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
						return fmt.Errorf("mod download failed with status %s", resp.Status)
					}
					if resp.ContentLength > maxExtensionModSize {
						return fmt.Errorf("mod exceeds the %d MB size limit", maxExtensionModSize/(1024*1024))
					}
					out, err := os.Create(destPath)
					if err != nil {
						return err
					}
					defer out.Close()
					written, err := io.Copy(out, io.LimitReader(resp.Body, maxExtensionModSize+1))
					if err != nil {
						_ = os.Remove(destPath)
						return err
					}
					if written > maxExtensionModSize {
						_ = os.Remove(destPath)
						return fmt.Errorf("mod exceeds the %d MB size limit", maxExtensionModSize/(1024*1024))
					}
					return nil
				}

				dlErr := doDownload(downloadURL)
				if dlErr != nil && (parsedURL.Hostname() == "edge.forgecdn.net" || parsedURL.Hostname() == "media.forgecdn.net") {
					// Fallback to alternative CDN domain if DNS or connection fails
					altHost := "media.forgecdn.net"
					if parsedURL.Hostname() == "media.forgecdn.net" {
						altHost = "edge.forgecdn.net"
					}
					altURL := strings.Replace(downloadURL, parsedURL.Hostname(), altHost, 1)
					if altErr := doDownload(altURL); altErr == nil {
						return destPath, nil
					}
				}
				if dlErr != nil {
					return "", dlErr
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
					if strings.HasSuffix(jarName, ".disabled") {
						newPath := filepath.Join(modsDir, strings.TrimSuffix(jarName, ".disabled"))
						return os.Rename(currentPath, newPath)
					}
					return nil
				} else {
					if !strings.HasSuffix(jarName, ".disabled") {
						newPath := filepath.Join(modsDir, jarName+".disabled")
						return os.Rename(currentPath, newPath)
					}
					return nil
				}
			},
			m.emit,
			m.requestConfirmation,
			func(packURL, packName string) (string, error) {
				parsedURL, err := neturl.Parse(packURL)
				if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" {
					return "", fmt.Errorf("modpack downloads require an HTTPS URL")
				}
				targetRoot := filepath.Join(fs.GetDataDir(), "instances")
				inst, err := instance.InstallMrpack(context.Background(), packURL, packName, targetRoot, nil)
				if err != nil {
					return "", fmt.Errorf("modpack install failed: %w", err)
				}

				// Option A: Auto-trigger Minecraft installation pipeline in background
				go func() {
					info, err := mojang.GetVersionInfo(inst.Version)
					if err != nil {
						fmt.Printf("[Mrpack] auto-install failed to fetch version info: %v\n", err)
						return
					}
					basePath := filepath.Join(targetRoot, inst.ID)
					assetsDir := fs.GetAssetsDir()
					engine := mojang.NewDownloadEngine(m.ctx, inst.ID, basePath)

					if m.emit != nil {
						m.emit(m.ctx, "instance:state", map[string]interface{}{
							"id":    inst.ID,
							"state": "Installing",
						})
					}

					if err := engine.Install(info, assetsDir); err != nil {
						fmt.Printf("[Mrpack] auto-install failed: %v\n", err)
						if m.emit != nil {
							m.emit(m.ctx, "instance:error", map[string]interface{}{
								"id":      inst.ID,
								"message": fmt.Sprintf("Installation failed: %v", err),
							})
							m.emit(m.ctx, "instance:state", map[string]interface{}{
								"id":    inst.ID,
								"state": "Error",
							})
						}
					} else {
						if m.emit != nil {
							m.emit(m.ctx, "instance:state", map[string]interface{}{
								"id":    inst.ID,
								"state": "Idle",
							})
						}
					}
				}()

				return inst.ID, nil
			},
			func(instanceID, fileName, downloadURL string) (string, error) {
				return downloadInstanceAsset(instanceID, "resourcepacks", fileName, downloadURL, map[string]bool{".zip": true, ".jar": true})
			},
			func(instanceID, fileName, downloadURL string) (string, error) {
				return downloadInstanceAsset(instanceID, "shaderpacks", fileName, downloadURL, map[string]bool{".zip": true})
			},
			func(instanceID string) ([]map[string]interface{}, error) {
				return listInstanceScreenshots(m.serverURL, instanceID)
			},
			func(instanceID, fileName string) error {
				return deleteInstanceScreenshot(instanceID, fileName)
			},
			func(instanceID, fileName string) error {
				return openInstanceScreenshot(instanceID, fileName)
			},
			func(instanceID, fileName string) (string, error) {
				return getInstanceScreenshotData(instanceID, fileName)
			},
		)
		newSandboxes[id] = sandbox

		if manifest.Main != "" {
			scriptPath := filepath.Join(extDir, id, manifest.Main)
			scriptData, err := os.ReadFile(scriptPath)
			if err == nil {
				// Execute with 10s timeout – prevents a hung extension from blocking the whole reload
				timeoutCtx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
				done := make(chan error, 1)
				go func() { done <- sandbox.Execute(string(scriptData)) }()
				var execErr error
				select {
				case execErr = <-done:
				case <-timeoutCtx.Done():
					sandbox.vm.Interrupt(fmt.Errorf("extension %s timed out after 10s", id))
					execErr = <-done
					sandbox.vm.ClearInterrupt()
					if execErr == nil {
						execErr = fmt.Errorf("timeout")
					}
				}
				cancel()
				sandbox.vm.ClearInterrupt()
				if execErr != nil {
					fmt.Printf("[Manager] Failed to execute %s for %s: %v\n", manifest.Main, id, execErr)
					ext.Status = "Error"
				} else {
					fmt.Printf("[Manager] Successfully loaded extension isolate: %s\n", id)
				}
			} else {
				fmt.Printf("[Manager] Missing main script %s for %s\n", manifest.Main, id)
				ext.Status = "Error (Missing Main)"
			}
		}

		ext.Reloading = false
		m.LoadedExtensions[id] = ext
	}

	m.sandboxes = newSandboxes
	m.SidebarPages = newSidebarPages
	m.ModLoaders = newModLoaders
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
	if m.emit != nil {
		m.emit(m.ctx, "extension:confirmation", action)
	}

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
		if m.emit != nil {
			m.emit(m.ctx, "extension:message:"+extID, response)
		}
		return
	}

	// Emit the response back to the frontend as a Wails event
	if m.emit != nil {
		m.emit(m.ctx, "extension:message:"+extID, result)
	}
}

// BroadcastEvent dispatches an event to all sandboxes that registered via Aether.events.on
func (m *Manager) BroadcastEvent(event string, payload map[string]interface{}) {
	// Copy sandboxes to avoid holding lock while invoking JS
	m.reloadMu.Lock()
	sandboxes := make([]*Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		sandboxes = append(sandboxes, sb)
	}
	m.reloadMu.Unlock()
	for _, sb := range sandboxes {
		sb.EmitEvent(event, payload)
	}
}

// GetSidebarPages returns all registered sidebar pages
func (m *Manager) GetSidebarPages() []map[string]interface{} {
	return m.SidebarPages
}

func downloadInstanceAsset(instanceID, subFolder, fileName, downloadURL string, validExts map[string]bool) (string, error) {
	fileName = filepath.Base(fileName)
	ext := strings.ToLower(filepath.Ext(fileName))
	if !validExts[ext] {
		return "", fmt.Errorf("file extension %s is not permitted for %s", ext, subFolder)
	}
	parsedURL, err := neturl.Parse(downloadURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" {
		return "", fmt.Errorf("asset downloads require an HTTPS URL")
	}
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(instanceDir, subFolder)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}
	destPath := filepath.Join(targetDir, fileName)

	doDownload := func(targetURL string) error {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("download failed with status %s", resp.Status)
		}
		const maxAssetSize int64 = 250 * 1024 * 1024
		if resp.ContentLength > maxAssetSize {
			return fmt.Errorf("file exceeds size limit of %d MB", maxAssetSize/(1024*1024))
		}
		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()
		written, err := io.Copy(out, io.LimitReader(resp.Body, maxAssetSize+1))
		if err != nil {
			_ = os.Remove(destPath)
			return err
		}
		if written > maxAssetSize {
			_ = os.Remove(destPath)
			return fmt.Errorf("file exceeds size limit of %d MB", maxAssetSize/(1024*1024))
		}
		return nil
	}

	dlErr := doDownload(downloadURL)
	if dlErr != nil && (parsedURL.Hostname() == "edge.forgecdn.net" || parsedURL.Hostname() == "media.forgecdn.net") {
		altHost := "media.forgecdn.net"
		if parsedURL.Hostname() == "media.forgecdn.net" {
			altHost = "edge.forgecdn.net"
		}
		altURL := strings.Replace(downloadURL, parsedURL.Hostname(), altHost, 1)
		if altErr := doDownload(altURL); altErr == nil {
			return destPath, nil
		}
	}
	if dlErr != nil {
		return "", dlErr
	}
	return destPath, nil
}

func listInstanceScreenshots(serverURL, instanceID string) ([]map[string]interface{}, error) {
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return nil, err
	}
	screenshotsDir := filepath.Join(instanceDir, "screenshots")
	entries, err := os.ReadDir(screenshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	var results []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		imgURL := fmt.Sprintf("%s/_screenshots/%s/%s", serverURL, neturl.PathEscape(instanceID), neturl.PathEscape(entry.Name()))
		results = append(results, map[string]interface{}{
			"name":    entry.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime().Format(time.RFC3339),
			"url":     imgURL,
		})
	}
	return results, nil
}

func getInstanceScreenshotData(instanceID, fileName string) (string, error) {
	fileName = filepath.Base(fileName)
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return "", err
	}
	imgPath, err := fs.ContainedPath(filepath.Join(instanceDir, "screenshots"), fileName)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	mime := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		mime = "image/jpeg"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

func deleteInstanceScreenshot(instanceID, fileName string) error {
	fileName = filepath.Base(fileName)
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	imgPath, err := fs.ContainedPath(filepath.Join(instanceDir, "screenshots"), fileName)
	if err != nil {
		return err
	}
	return os.Remove(imgPath)
}

func openInstanceScreenshot(instanceID, fileName string) error {
	fileName = filepath.Base(fileName)
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	imgPath, err := fs.ContainedPath(filepath.Join(instanceDir, "screenshots"), fileName)
	if err != nil {
		return err
	}
	return exec.Command("cmd", "/c", "start", "", imgPath).Run()
}
