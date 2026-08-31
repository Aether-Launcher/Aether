package main

import (
	"embed"
	"os"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

// windowChrome reports which title bar style the current platform uses.
//   - "custom": Aether's frameless title bar (Windows, and Linux sessions
//     whose window manager honors the frameless hint).
//   - "system": the native window title bar (macOS, and KDE Plasma on
//     Wayland, where KWin ignores gtk_window_set_decorated(FALSE) — KDE
//     bug 484800 — and keeps the system title bar on frameless windows).
func windowChrome() string {
	if runtime.GOOS == "darwin" {
		return "system"
	}
	if runtime.GOOS == "linux" &&
		strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") &&
		strings.Contains(strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP")), "kde") {
		return "system"
	}
	return "custom"
}

//go:embed all:frontend/dist
var assets embed.FS

// Version is the app version, stamped at build time via
// -ldflags "-X main.Version=vX.Y.Z". It stays "dev" for local builds,
// which disables the auto-update checks.
var Version = "dev"

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// On macOS, use native window chrome (traffic-light controls, standard titlebar).
	// On Windows/Linux, keep the custom frameless titlebar where the platform allows it.
	frameless := windowChrome() == "custom"

	// macOS-specific options: use full-size content view so the sidebar extends
	// edge-to-edge, but keep the native title bar visible.
	var macOptions *mac.Options
	if runtime.GOOS == "darwin" {
		macOptions = &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			// Extend content behind the titlebar for a seamless look
			WebviewIsTransparent:          true,
			WindowIsTranslucent:           false,
			Preferences: &mac.Preferences{
				TabFocusesLinks: mac.Enabled,
			},
		}
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:    "Aether",
		Width:    1100,
		Height:   768,
		MinWidth: 1100,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:        frameless,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Mac:              macOptions,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
