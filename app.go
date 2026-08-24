package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"Aether/pkg/auth"
	"Aether/pkg/discord"
	"Aether/pkg/extensions"
	"Aether/pkg/fs"
	"Aether/pkg/instance"
	"Aether/pkg/java"
	"Aether/pkg/mojang"
	"Aether/pkg/settings"
	"Aether/pkg/update"
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
	fs.EnsureDirectories()

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
			fmt.Printf("[StateChangeHook] id=%s state=%s\n", id, state)
			if extensions.GlobalManager != nil {
				extensions.GlobalManager.BroadcastEvent("instance:state", map[string]interface{}{"id": id, "state": state})
			}
			// Direct Go fallback for Discord presence – ensures vanilla and early
			// launches work even if the JS extension hasn't registered yet
			go func() {
				defer func() { _ = recover() }()
				fmt.Printf("[Discord-Go] StateChangeHook state=%s id=%s\n", state, id)
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
						fmt.Printf("[Discord-Go] SetActivity details=%q state=%q small=%q\n", details, stateText, small)
						if err := discord.SetActivity(details, stateText, "grass-block", target.Name, small, loader, &start); err != nil {
							fmt.Printf("[Discord-Go] SetActivity failed: %v\n", err)
						}
					} else {
						fmt.Printf("[Discord-Go] SetActivity fallback for id=%s\n", id)
						if err := discord.SetActivity("Playing Minecraft", state, "grass-block", "", "", "", &start); err != nil {
							fmt.Printf("[Discord-Go] SetActivity fallback failed: %v\n", err)
						}
					}
				} else if state == "Stopped" || state == "Crashed" {
					fmt.Printf("[Discord-Go] Clear to Idle\n")
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

// GetInstances returns all installed instances
func (a *App) GetInstances() []instance.Instance {
	return instance.GetInstances()
}

// GetActiveInstance returns the currently selected instance
func (a *App) GetActiveInstance() *instance.Instance {
	return instance.GetActiveInstance()
}

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

// JavaRuntimeStatus describes the status of a managed or system Java runtime.
type JavaRuntimeStatus struct {
	Version   int    `json:"version"`
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	IsSystem  bool   `json:"isSystem"`
}

// GetJavaStatus returns the installation status for each required Java version.
func (a *App) GetJavaStatus() []JavaRuntimeStatus {
	versions := []int{8, 17, 21}
	var statuses []JavaRuntimeStatus
	for _, v := range versions {
		installed := java.IsManagedJavaInstalled(v)
		path := ""
		isSystem := false
		if installed {
			path = java.GetManagedJavaPath(v)
		} else if sysPath, err := java.FindJava(v); err == nil {
			installed = true
			path = sysPath
			isSystem = true
		}
		statuses = append(statuses, JavaRuntimeStatus{
			Version:   v,
			Installed: installed,
			Path:      path,
			IsSystem:  isSystem,
		})
	}
	return statuses
}

// DownloadJavaRuntime downloads a managed JRE for the given major version.
func (a *App) DownloadJavaRuntime(version int) error {
	return java.DownloadJava(a.ctx, version)
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
	// Clear the sidebar first so removed extensions don't leave stale tabs.
	runtime.EventsEmit(a.ctx, "extension:sidebar:reset")
	return extensions.GlobalManager.ReloadAsync()
}

// SelectAndImportInstance imports an existing instance folder from Aether,
// Prism/MultiMC, or CurseForge. It returns a display label for the imported
// instance, or "" when the user cancelled the dialog.
func (a *App) SelectAndImportInstance() (string, error) {
	source, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select Minecraft Instance"})
	if err != nil {
		return "", err
	}
	if source == "" {
		return "", nil
	}

	if instance.DetectFormat(source) == instance.FormatUnknown {
		return "", fmt.Errorf("this doesn't look like an Aether, Prism/MultiMC, or CurseForge instance folder")
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
	}[instance.DetectFormat(source)]

	return fmt.Sprintf("%s (%s)", inst.Name, launcher), nil
}

func (a *App) GetActiveAccount() *auth.Account {
	return auth.GetActiveAccount()
}

// GetAccounts returns all saved accounts
func (a *App) GetAccounts() []auth.Account {
	return auth.GetAccounts()
}

// LoginOffline creates or switches to an offline account with the given username
func (a *App) LoginOffline(username string) (auth.Account, error) {
	return auth.AddOfflineAccount(username)
}

// StartMicrosoftAuth initiates the Microsoft OAuth2 PKCE login flow
func (a *App) StartMicrosoftAuth() (auth.Account, error) {
	acc, err := auth.StartPKCEAuthFlow(a.ctx)
	if err != nil {
		return auth.Account{}, err
	}
	if err := auth.AddMicrosoftAccount(*acc); err != nil {
		return auth.Account{}, err
	}
	return *acc, nil
}

// SetActiveAccount sets the active account by ID
func (a *App) SetActiveAccount(id string) error {
	return auth.SetActiveAccount(id)
}

// RemoveAccount removes an account by ID
func (a *App) RemoveAccount(id string) error {
	return auth.RemoveAccount(id)
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

	go func() {
		if err := engine.Install(info, assetsDir); err != nil {
			msg := fmt.Sprintf("Installation failed: %v", err)
			fmt.Printf("Install error: %v\n", err)
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
