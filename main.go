package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// version is the version of this build. The release pipeline injects it with
// -ldflags "-X main.version=<tag>"; a plain `wails build` leaves the "dev"
// placeholder, which is never compared against published releases.
var version = "dev"

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "SilentOpen",
		Width:     1200,
		Height:    760,
		MinWidth:  360,
		MinHeight: 520,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Linux:            &linux.Options{Icon: appIcon},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
