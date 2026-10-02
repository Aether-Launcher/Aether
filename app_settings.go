// app_settings.go - global settings and self-update Wails bindings.

package main

import (
	"fmt"
	stdruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/settings"
	"github.com/Aether-Launcher/Aether/pkg/update"
)

// CheckForUpdates queries GitHub Releases for a newer launcher version.
// Returns nil when the app is up to date.
func (a *App) CheckForUpdates() (*update.Info, error) {
	if Version == "dev" {
		return nil, nil
	}
	s := settings.Load()
	return update.Check(a.ctx, Version, s.IncludeBetaUpdates)
}

// DownloadAndUpdate downloads and applies the newest release. On success
// the app relaunches (or the DMG is opened on macOS).
func (a *App) DownloadAndUpdate() error {
	if Version == "dev" {
		return fmt.Errorf("local builds cannot self-update")
	}
	s := settings.Load()
	info, err := update.Check(a.ctx, Version, s.IncludeBetaUpdates)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("no update available")
	}

	// macOS has no in-place apply path — hand the DMG to the user.
	if stdruntime.GOOS == "darwin" {
		runtime.BrowserOpenURL(a.ctx, info.DownloadURL)
		return nil
	}

	runtime.EventsEmit(a.ctx, "update:status", map[string]interface{}{
		"phase": "downloading", "version": info.Version,
	})
	path, err := update.Download(a.ctx, info)
	if err != nil {
		runtime.EventsEmit(a.ctx, "update:status", map[string]interface{}{
			"phase": "error", "message": err.Error(),
		})
		return err
	}
	runtime.EventsEmit(a.ctx, "update:status", map[string]interface{}{
		"phase": "ready", "version": info.Version,
	})
	return update.Apply(a.ctx, info, path)
}

// GetSettings returns the global launcher settings
func (a *App) GetSettings() settings.GlobalSettings {
	return settings.Load()
}

// SaveSettings updates the global launcher settings
func (a *App) SaveSettings(s settings.GlobalSettings) error {
	return settings.Save(s)
}
