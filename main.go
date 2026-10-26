// Davish is a desktop app for managing Linux servers over SSH.
package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"davish/internal/app"
	"davish/internal/apperr"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:     "Davish",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		// macOS keeps its native traffic lights over a transparent title bar;
		// other platforms draw everything in the webview.
		Frameless: runtime.GOOS != "darwin",
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
		},
		// The app draws its own desktop, so it starts filling the screen.
		WindowStartState: options.Maximised,
		Menu:             appMenu(),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 205, G: 215, B: 234, A: 255},
		ErrorFormatter:   apperr.Format,
		OnStartup:        a.Startup,
		OnShutdown:       a.Shutdown,
		Bind: []any{
			a,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// appMenu keeps the standard macOS menus: without the Edit menu, copy/paste
// shortcuts do not reach the webview. The Window menu is left out on purpose:
// its ⌘M would minimize the whole app instead of the focused in-app window.
func appMenu() *menu.Menu {
	if runtime.GOOS != "darwin" {
		return nil
	}
	m := menu.NewMenu()
	m.Append(menu.AppMenu())
	m.Append(menu.EditMenu())
	return m
}
