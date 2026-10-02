// app_theme.go - theme management Wails bindings.

package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/settings"
	"github.com/Aether-Launcher/Aether/pkg/theme"
)

// ── Themes ───────────────────────────────────────────────────────────────
//
// A theme (.theme, a renamed zip) is a CSS overwrite plus an optional set of
// PNG asset overrides for a small, fixed whitelist of "slots" (see
// pkg/theme/protected.go). It cannot touch the launcher's built-in app icon
// or its "Aether" name.

// GetThemes returns metadata for every installed theme, flagging the active one.
func (a *App) GetThemes() ([]theme.Info, error) {
	s := settings.Load()
	return theme.List(s.ActiveTheme)
}

// GetActiveThemeCSS returns the sanitized CSS for the currently active theme,
// or "" if no theme is active.
func (a *App) GetActiveThemeCSS() string {
	s := settings.Load()
	return theme.GetCSS(s.ActiveTheme)
}

// GetActiveThemeAssets returns the key→URL map of PNG overrides for the
// currently active theme (only ever the whitelisted slots).
func (a *App) GetActiveThemeAssets() map[string]string {
	s := settings.Load()
	return theme.GetAssetURLs(s.ActiveTheme)
}

// SelectAndInstallTheme opens a file picker for a .theme package, installs
// it, and returns any non-fatal warnings (e.g. a rejected overwrite.json key)
// so the UI can show them to the user.
func (a *App) SelectAndInstallTheme() (*theme.InstallResult, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Theme Package",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Aether Themes (*.theme)",
				Pattern:     "*.theme",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if file == "" {
		// User cancelled
		return nil, nil
	}
	return theme.InstallFromArchive(file)
}

// SetActiveTheme sets the active theme by ID ("" disables all themes,
// reverting to Aether's default look).
func (a *App) SetActiveTheme(id string) error {
	s := settings.Load()
	s.ActiveTheme = id
	return settings.Save(s)
}

// UninstallTheme removes an installed theme by ID. If it was active, the
// caller is responsible for also calling SetActiveTheme("") if desired.
func (a *App) UninstallTheme(id string) error {
	s := settings.Load()
	if s.ActiveTheme == id {
		s.ActiveTheme = ""
		if err := settings.Save(s); err != nil {
			return err
		}
	}
	return theme.Uninstall(id)
}
