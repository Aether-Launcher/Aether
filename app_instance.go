// app_instance.go - instance lifecycle Wails bindings (create/launch/install/import).

package main

import (
	"fmt"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/instance"
	"github.com/Aether-Launcher/Aether/pkg/logger"
	"github.com/Aether-Launcher/Aether/pkg/mojang"
)

// GetInstances returns all installed instances
func (a *App) GetInstances() []instance.Instance {
	return instance.GetInstances()
}

// GetActiveInstance returns the currently selected instance
func (a *App) GetActiveInstance() *instance.Instance {
	return instance.GetActiveInstance()
}

// SelectAndImportInstance imports an existing instance folder from Aether,
// Prism/MultiMC, Modrinth, or CurseForge. It returns a display label for the
// imported instance, or "" when the user cancelled the dialog.
func (a *App) SelectAndImportInstance() (string, error) {
	source, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select Minecraft Instance"})
	if err != nil {
		return "", err
	}
	if source == "" {
		return "", nil
	}

	if instance.DetectFormat(source) == instance.FormatUnknown {
		return "", fmt.Errorf("this doesn't look like an Aether, Prism/MultiMC, Modrinth, or CurseForge instance folder (expected one of: instance.json, mmc-pack.json, profile.json, modrinth.index.json, manifest.json, minecraftinstance.json) — make sure you selected the instance folder itself, not the launcher root")
	}

	instancesDir := filepath.Join(fs.GetDataDir(), "instances")
	inst, err := instance.ImportInstance(source, instancesDir, func(done, total int, file string) {
		runtime.EventsEmit(a.ctx, "instance:import-progress", map[string]interface{}{
			"done":  done,
			"total": total,
			"file":  file,
		})
	})
	if err != nil {
		return "", err
	}

	launcher := map[instance.Format]string{
		instance.FormatNative:     "Aether",
		instance.FormatMultiMC:    "Prism/MultiMC",
		instance.FormatCurseForge: "CurseForge",
		instance.FormatModrinth:   "Modrinth",
		instance.FormatGeneric:    "Minecraft folder",
	}[instance.DetectFormat(source)]

	return fmt.Sprintf("%s (%s)", inst.Name, launcher), nil
}

// LaunchInstance starts the specified instance
func (a *App) LaunchInstance(id string) error {
	instances := instance.GetInstances()
	var target *instance.Instance
	for i := range instances {
		if instances[i].ID == id {
			target = &instances[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("instance not found: %s", id)
	}
	return instance.Launch(a.ctx, target)
}

// ListInstanceWorlds returns the singleplayer worlds of an instance.
func (a *App) ListInstanceWorlds(id string) ([]instance.WorldInfo, error) {
	return instance.ListWorlds(id)
}

// LaunchInstanceToServer launches an instance and auto-connects to a
// multiplayer server (vanilla --server/--port flags).
func (a *App) LaunchInstanceToServer(id string, host string, port int) error {
	return instance.LaunchToServer(a.ctx, id, host, port)
}

// LaunchInstanceToWorld launches an instance and auto-loads a singleplayer
// world (Mojang Quick Play, requires Minecraft 1.20+).
func (a *App) LaunchInstanceToWorld(id string, world string) error {
	return instance.LaunchToWorld(a.ctx, id, world)
}

// InstallInstance triggers the Mojang download pipeline
func (a *App) InstallInstance(id string) error {
	instances := instance.GetInstances()
	var target *instance.Instance
	for i := range instances {
		if instances[i].ID == id {
			target = &instances[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("instance not found: %s", id)
	}

	info, err := mojang.GetVersionInfo(target.Version)
	if err != nil {
		status := mojang.CheckConnectivity()
		if status.Overall == "offline" || status.Overall == "degraded" {
			return fmt.Errorf("can't reach Minecraft servers — check your internet connection (couldn't fetch version info for %s)", target.Version)
		}
		return fmt.Errorf("failed to get version info for %s: %w", target.Version, err)
	}

	basePath := filepath.Join(fs.GetDataDir(), "instances", target.ID)
	assetsDir := fs.GetAssetsDir()
	engine := mojang.NewDownloadEngine(a.ctx, target.ID, basePath)

	// Refuse to start a second pipeline for the same instance: concurrent
	// pipelines write the same library files and corrupt each other's
	// downloads (rename collisions, checksum mismatches).
	if !mojang.ClaimInstall(target.ID) {
		return fmt.Errorf("installation already in progress for %s", target.ID)
	}

	go func() {
		defer mojang.ReleaseInstall(target.ID)
		if err := engine.Install(info, assetsDir); err != nil {
			msg := fmt.Sprintf("Installation failed: %v", err)
			logger.Error("Install", fmt.Sprintf("Install error: %v", err))
			runtime.EventsEmit(a.ctx, "instance:error", map[string]interface{}{
				"id":      target.ID,
				"message": msg,
			})
			runtime.EventsEmit(a.ctx, "instance:state", map[string]interface{}{
				"id":    target.ID,
				"state": "Error",
			})
		} else {
			runtime.EventsEmit(a.ctx, "instance:state", map[string]interface{}{
				"id":    target.ID,
				"state": "Idle",
			})
		}
	}()

	return nil
}

// GetAvailableVersions fetches releases from Mojang
func (a *App) GetAvailableVersions(includeSnapshots bool) ([]string, error) {
	manifest, err := mojang.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range manifest.Versions {
		if v.Type == "release" || (includeSnapshots && v.Type == "snapshot") {
			versions = append(versions, v.ID)
		}
	}
	return versions, nil
}

// GetConnectivityStatus returns the health of the services Aether depends on
// for installing and launching Minecraft instances.
func (a *App) GetConnectivityStatus() mojang.ConnectivityStatus {
	return mojang.CheckConnectivity()
}

// CreateInstance creates a new instance on disk and returns the created instance
func (a *App) CreateInstance(name, version, loader string) (*instance.Instance, error) {
	inst, err := instance.Create(name, version, loader)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// UpdateInstance saves changes to an instance
func (a *App) UpdateInstance(inst *instance.Instance) error {
	return instance.UpdateInstance(inst)
}

// DeleteInstance deletes an instance completely
func (a *App) DeleteInstance(id string) error {
	return instance.DeleteInstance(id)
}
