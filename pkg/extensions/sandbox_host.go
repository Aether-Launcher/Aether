// sandbox_host.go - host-side callbacks passed to NewSandbox during reload.
//
// reloadSandboxes used to pass ~20 anonymous functions inline in a single
// NewSandbox call, which was unreadable and untestable. They now live here
// as methods on sandboxHost; the call site passes bound method values.

package extensions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/instance"
	"github.com/Aether-Launcher/Aether/pkg/logger"
	"github.com/Aether-Launcher/Aether/pkg/mojang"
)

// sandboxHost carries per-reload accumulator state (sidebar pages, mod
// loaders) plus the Manager for context, event emission and confirmations.
type sandboxHost struct {
	m            *Manager
	sidebarPages []map[string]interface{}
	modLoaders   map[string]ModLoaderConfig
}

func newSandboxHost(m *Manager) *sandboxHost {
	return &sandboxHost{
		m:            m,
		sidebarPages: make([]map[string]interface{}, 0),
		modLoaders:   make(map[string]ModLoaderConfig),
	}
}

func (h *sandboxHost) onSidebarPage(payload map[string]interface{}) {
	h.sidebarPages = append(h.sidebarPages, payload)
}

func (h *sandboxHost) onModLoader(config ModLoaderConfig) {
	h.modLoaders[config.ID] = config
}

func (h *sandboxHost) listInstances() []InstanceInfo {
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
}

func (h *sandboxHost) installMod(instanceID, jarName, downloadURL string) (string, error) {
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
}

func (h *sandboxHost) listMods(instanceID string) ([]string, error) {
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
}

func (h *sandboxHost) deleteMod(instanceID, jarName string) error {
	jarName = filepath.Base(jarName)
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	modPath := filepath.Join(instanceDir, "mods", jarName)
	return os.Remove(modPath)
}

func (h *sandboxHost) toggleMod(instanceID, jarName string, enable bool) error {
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
}

func (h *sandboxHost) installModpack(packURL, packName, iconURL string) (string, error) {
	parsedURL, err := neturl.Parse(packURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" {
		return "", fmt.Errorf("modpack downloads require an HTTPS URL")
	}
	targetRoot := filepath.Join(fs.GetDataDir(), "instances")
	inst, err := instance.InstallMrpack(context.Background(), packURL, packName, targetRoot, nil)
	if err != nil {
		return "", fmt.Errorf("modpack install failed: %w", err)
	}

	// Best-effort pack icon: the pack still installs if this fails.
	if iconURL != "" {
		if err := instance.SetIconFromURL(context.Background(), inst.ID, iconURL); err != nil {
			logger.Warn("Mrpack", fmt.Sprintf("pack icon skipped for %s: %v", inst.ID, err))
		}
	}

// Option A: Auto-trigger Minecraft installation pipeline in background.
// Shares the per-instance claim with App.InstallInstance so a manual
// Install click can't start a second pipeline for the same instance.
go func() {
	if !mojang.ClaimInstall(inst.ID) {
		logger.Info("Mrpack", fmt.Sprintf("install already in progress for %s, skipping duplicate pipeline", inst.ID))
		return
	}
	defer mojang.ReleaseInstall(inst.ID)

	info, err := mojang.GetVersionInfo(inst.Version)
		if err != nil {
			logger.Error("Mrpack", fmt.Sprintf("auto-install failed to fetch version info: %v", err))
			return
		}
		basePath := filepath.Join(targetRoot, inst.ID)
		assetsDir := fs.GetAssetsDir()
		engine := mojang.NewDownloadEngine(h.m.ctx, inst.ID, basePath)

		if h.m.emit != nil {
			h.m.emit(h.m.ctx, "instance:state", map[string]interface{}{
				"id":    inst.ID,
				"state": "Installing",
			})
		}

		if err := engine.Install(info, assetsDir); err != nil {
			logger.Error("Mrpack", fmt.Sprintf("auto-install failed: %v", err))
			if h.m.emit != nil {
				h.m.emit(h.m.ctx, "instance:error", map[string]interface{}{
					"id":      inst.ID,
					"message": fmt.Sprintf("Installation failed: %v", err),
				})
				h.m.emit(h.m.ctx, "instance:state", map[string]interface{}{
					"id":    inst.ID,
					"state": "Error",
				})
			}
		} else {
			if h.m.emit != nil {
				h.m.emit(h.m.ctx, "instance:state", map[string]interface{}{
					"id":    inst.ID,
					"state": "Idle",
				})
			}
		}
	}()

	return inst.ID, nil
}

func (h *sandboxHost) installResourcePack(instanceID, fileName, downloadURL string) (string, error) {
	return downloadInstanceAsset(instanceID, "resourcepacks", fileName, downloadURL, map[string]bool{".zip": true, ".jar": true})
}

func (h *sandboxHost) installShaderPack(instanceID, fileName, downloadURL string) (string, error) {
	return downloadInstanceAsset(instanceID, "shaderpacks", fileName, downloadURL, map[string]bool{".zip": true})
}

func (h *sandboxHost) listScreenshots(instanceID string) ([]map[string]interface{}, error) {
	return listInstanceScreenshots(h.m.serverURL, instanceID)
}

func (h *sandboxHost) deleteScreenshot(instanceID, fileName string) error {
	return deleteInstanceScreenshot(instanceID, fileName)
}

func (h *sandboxHost) openScreenshot(instanceID, fileName string) error {
	return openInstanceScreenshot(instanceID, fileName)
}

func (h *sandboxHost) getScreenshotData(instanceID, fileName string) (string, error) {
	return getInstanceScreenshotData(instanceID, fileName)
}

func (h *sandboxHost) listWorlds(instanceID string) ([]map[string]interface{}, error) {
	worlds, err := instance.ListWorlds(instanceID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(worlds))
	for _, w := range worlds {
		out = append(out, map[string]interface{}{
			"id":         w.ID,
			"name":       w.Name,
			"lastPlayed": w.LastPlayed,
			"gameMode":   w.GameMode,
		})
	}
	return out, nil
}

func (h *sandboxHost) launchToServer(instanceID, host string, port int) error {
	return instance.LaunchToServer(h.m.ctx, instanceID, host, port)
}

func (h *sandboxHost) launchToWorld(instanceID, world string) error {
	return instance.LaunchToWorld(h.m.ctx, instanceID, world)
}
