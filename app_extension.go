// app_extension.go - extension management Wails bindings.

package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/extensions"
)

func (a *App) GetExtensions() []extensions.Extension {
	return extensions.GetExtensions()
}

// GetExtensionSidebarPages returns sidebar pages contributed by extensions
func (a *App) GetExtensionSidebarPages() []map[string]interface{} {
	if extensions.GlobalManager != nil {
		return extensions.GlobalManager.GetSidebarPages()
	}
	return []map[string]interface{}{}
}

// SendExtensionMessage routes an IPC message from the UI iframe to the extension sandbox
func (a *App) SendExtensionMessage(extID string, payload map[string]interface{}) {
	if extensions.GlobalManager != nil {
		extensions.GlobalManager.HandleIPCMessage(extID, payload)
	}
}

// ResolveExtensionConfirmation approves or rejects a sensitive extension action.
func (a *App) ResolveExtensionConfirmation(requestID string, approved bool) error {
	if extensions.GlobalManager == nil {
		return fmt.Errorf("extensions are disabled")
	}
	return extensions.GlobalManager.ResolveConfirmation(requestID, approved)
}

// ModLoaderInfo represents a registered mod loader
type ModLoaderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetModLoaders returns all mod loaders registered by extensions
func (a *App) GetModLoaders() []ModLoaderInfo {
	var loaders []ModLoaderInfo
	if extensions.GlobalManager != nil {
		for _, loader := range extensions.GlobalManager.ModLoaders {
			loaders = append(loaders, ModLoaderInfo{
				ID:          loader.ID,
				Name:        loader.Name,
				Description: loader.Description,
			})
		}
	}
	return loaders
}

func (a *App) SelectAndInstallExtension() (bool, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Extension Package",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Aether Extensions (*.aex)",
				Pattern:     "*.aex",
			},
		},
	})
	if err != nil {
		return false, err
	}
	if file == "" {
		// User cancelled
		return false, nil
	}

	if err := extensions.InstallFromArchive(file); err != nil {
		return false, err
	}

	// Reload all extensions dynamically!
	if extensions.GlobalManager != nil {
		extensions.GlobalManager.LoadAll()
	}

	return true, nil
}

// DownloadAndInstallExtension downloads a remote zip and installs it
func (a *App) DownloadAndInstallExtension(url string) (bool, error) {
	if err := extensions.DownloadAndInstallExtension(url); err != nil {
		return false, err
	}

	if extensions.GlobalManager != nil {
		extensions.GlobalManager.LoadAll()
	}

	return true, nil
}

// UninstallExtension removes a locally installed extension and reloads the manager.
func (a *App) UninstallExtension(id string) error {
	if extensions.GlobalManager == nil {
		return fmt.Errorf("extensions are disabled")
	}
	return extensions.GlobalManager.Uninstall(id)
}

// SetExtensionEnabled pauses or resumes an installed extension and reloads the
// extension registry so its pages and hooks match the selected state.
func (a *App) SetExtensionEnabled(id string, enabled bool) error {
	if extensions.GlobalManager == nil {
		return fmt.Errorf("extensions are disabled")
	}
	return extensions.GlobalManager.SetEnabled(id, enabled)
}

// GetExtensionUpdates force-refreshes the registry and returns available
// updates for installed extensions. An error is returned when the registry
// cannot be reached, so the UI can tell the user the check actually failed
// instead of reporting "no updates".
func (a *App) GetExtensionUpdates() ([]extensions.ExtensionUpdate, error) {
	if extensions.GlobalManager == nil {
		return nil, fmt.Errorf("extensions are disabled")
	}
	if _, err := extensions.RefreshGallery(); err != nil {
		return nil, err
	}
	return extensions.CheckForUpdates(), nil
}

// UpdateExtension updates an installed extension to its newest registry version.
func (a *App) UpdateExtension(id string) (extensions.ExtensionUpdate, error) {
	if extensions.GlobalManager == nil {
		return extensions.ExtensionUpdate{}, fmt.Errorf("extensions are disabled")
	}
	if _, err := extensions.RefreshGallery(); err != nil {
		return extensions.ExtensionUpdate{}, err
	}
	return extensions.UpdateExtension(id)
}

// ReloadExtensions re-scans the extensions directory and reloads all
// extensions asynchronously, refreshing the sidebar and mod loader registrations.
func (a *App) ReloadExtensions() error {
	if extensions.GlobalManager == nil {
		return fmt.Errorf("extensions are disabled")
	}
	return extensions.GlobalManager.ReloadAsync()
}
