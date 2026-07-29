package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// On macOS, use native window chrome (traffic-light controls, standard titlebar).
	// On Windows/Linux, keep our custom frameless titlebar.
	frameless := runtime.GOOS != "darwin"

	// macOS-specific options: use full-size content view so our sidebar extends
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
