// app.go - core App struct, startup, update check, logs and version bindings.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/discord"
	"github.com/Aether-Launcher/Aether/pkg/extensions"
	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/instance"
	"github.com/Aether-Launcher/Aether/pkg/logger"
	"github.com/Aether-Launcher/Aether/pkg/settings"
	"github.com/Aether-Launcher/Aether/pkg/theme"
	"github.com/Aether-Launcher/Aether/pkg/update"
)

var modLoaderLaunchMu sync.Mutex

type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// to call runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.SetEmitter(ctx, runtime.EventsEmit)
	logger.Info("Launcher", "Aether core runtime initializing...")

	fs.EnsureDirectories()

	if _, err := theme.GlobalServer.Start(); err != nil {
		logger.Warn("Theme", fmt.Sprintf("Failed to start asset server: %v", err))
	}

	globalSettings := settings.Load()

	if !globalSettings.DisableExtensions {
		// Initialize and load all extensions into their isolates
		extensions.GlobalManager = extensions.NewManager(ctx, runtime.EventsEmit)
		extensions.GlobalManager.LoadAll()

		// Wire the mod loader hook so launcher.go can call extension mod loaders
		// without an import cycle (instance → extensions → instance)
		instance.ModLoaderHook = func(loaderID string, hookCtx map[string]interface{}) (map[string]interface{}, error) {
			loader, ok := extensions.GlobalManager.ModLoaders[loaderID]
			if !ok {
				return nil, fmt.Errorf("mod loader '%s' is not installed — available: %v", loaderID, registeredLoaderIDs())
			}
			if loader.Callback == nil {
				return nil, fmt.Errorf("mod loader '%s' has no onLaunch callback (extension failed to register it)", loaderID)
			}
			modLoaderLaunchMu.Lock()
			defer modLoaderLaunchMu.Unlock()
			return loader.Callback(hookCtx)
		}
		instance.StateChangeHook = func(id, state string) {
			logger.Debug("StateChangeHook", fmt.Sprintf("id=%s state=%s", id, state))
			if extensions.GlobalManager != nil {
				extensions.GlobalManager.BroadcastEvent("instance:state", map[string]interface{}{"id": id, "state": state})
			}
			// Direct Go fallback for Discord presence – ensures vanilla and early
			// launches work even if the JS extension hasn't registered yet
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("Discord-Go", fmt.Sprintf("StateChangeHook panic state=%s id=%s: %v", state, id, r))
					}
				}()
				logger.Debug("Discord-Go", fmt.Sprintf("StateChangeHook state=%s id=%s", state, id))
				if state == "Running" {
					var target *instance.Instance
					for _, inst := range instance.GetInstances() {
						if inst.ID == id {
							c := inst
							target = &c
							break
						}
					}
					start := time.Now()
					if target != nil {
						loader := target.Loader
						if loader == "" {
							loader = "vanilla"
						}
						stateText := target.Version
						if loader != "" {
							stateText = target.Version + " \u2022 " + loader
						}
						if len(stateText) < 2 {
							stateText = "Playing Minecraft"
						}
						details := target.Name
						if len(details) < 2 {
							details = "Playing Minecraft"
						}
						small := ""
						l := strings.ToLower(loader)
						if strings.Contains(l, "fabric") {
							small = "fabric"
						} else if strings.Contains(l, "forge") {
							small = "forge"
						} else if strings.Contains(l, "neoforge") {
							small = "neoforge"
						} else if strings.Contains(l, "quilt") {
							small = "quilt"
						}
						logger.Debug("Discord-Go", fmt.Sprintf("SetActivity details=%q state=%q small=%q", details, stateText, small))
						if err := discord.SetActivity(details, stateText, "grass-block", target.Name, small, loader, &start); err != nil {
							logger.Warn("Discord-Go", fmt.Sprintf("SetActivity failed: %v", err))
						}
					} else {
						logger.Debug("Discord-Go", fmt.Sprintf("SetActivity fallback for id=%s", id))
						if err := discord.SetActivity("Playing Minecraft", state, "grass-block", "", "", "", &start); err != nil {
							logger.Warn("Discord-Go", fmt.Sprintf("SetActivity fallback failed: %v", err))
						}
					}
				} else if state == "Stopped" || state == "Crashed" {
					logger.Debug("Discord-Go", "Clear to Idle")
					_ = discord.SetActivity("Idle in Launcher", "Aether", "aether-logo", "Aether Launcher", "", "", nil)
				}
			}()
		}
	}

	go checkForUpdatesDelayed(ctx)
}

// checkForUpdatesDelayed runs a background update check shortly after startup.
func checkForUpdatesDelayed(ctx context.Context) {
	time.Sleep(3 * time.Second)

	// Clean up a leftover .old binary from a previous interrupted update.
	if exePath, err := os.Executable(); err == nil {
		_ = os.Remove(exePath + ".old")
	}

	if Version == "dev" {
		return
	}
	s := settings.Load()
	if !s.AutoCheckUpdates {
		return
	}

	runtime.EventsEmit(ctx, "update:status", map[string]interface{}{"phase": "checking"})
	info, err := update.Check(ctx, Version, s.IncludeBetaUpdates)
	if err != nil {
		// Background check failures (offline etc.) stay silent; user-initiated
		// checks surface their errors through the bound method.
		runtime.EventsEmit(ctx, "update:status", map[string]interface{}{"phase": "none"})
		return
	}
	if info == nil {
		runtime.EventsEmit(ctx, "update:status", map[string]interface{}{"phase": "none"})
		return
	}
	runtime.EventsEmit(ctx, "update:status", map[string]interface{}{
		"phase":   "available",
		"version": info.Version,
		"notes":   info.ReleaseNotes,
	})
}

// registeredLoaderIDs lists the currently registered mod loader IDs for error messages.
func registeredLoaderIDs() []string {
	ids := make([]string, 0, len(extensions.GlobalManager.ModLoaders))
	for id := range extensions.GlobalManager.ModLoaders {
		ids = append(ids, id)
	}
	return ids
}

// WindowChrome reports whether the native window title bar ("system") or
// Aether's custom frameless title bar ("custom") is active on this platform.
func (a *App) WindowChrome() string {
	return windowChrome()
}

// GetLauncherVersion returns the running launcher version string (e.g. "v1.2.3").
// Returns "dev" for local development builds.
func (a *App) GetLauncherVersion() string {
	return Version
}

// GetLogs returns the buffered background log entries.
func (a *App) GetLogs() []logger.LogEntry {
	return logger.GetEntries()
}

// ClearLogs flushes all buffered background log entries.
func (a *App) ClearLogs() {
	logger.Clear()
}
