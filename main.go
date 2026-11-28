// Davish is a desktop app for managing Linux servers over SSH.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"davish/internal/app"
	"davish/internal/conn"
	"davish/internal/perf"
	"davish/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

// wails.json holds the version a release is tagged with; the updater
// compares against it.
//
//go:embed wails.json
var wailsJSON []byte

// The window icon on Linux, where the window manager has no bundle to
// take it from (#181).
//
//go:embed build/appicon.png
var appIcon []byte

func main() {
	// On Windows Davish also runs as its own relay to the SSH agent's pipe,
	// so 1Password can name the app asking for a key (#332). That run has
	// no window and exits when the relay ends.
	if conn.RunAgentProxy(os.Args) {
		return
	}
	// WebKit reads these while creating the webview, so they have to be set
	// before wails.Run. A missing config dir just leaves the defaults.
	if dir, err := store.DefaultDir(); err == nil {
		perf.Apply(dir)
	}
	// After the settings above, so a WebKit switch set there is respected
	// (docs/adr/0023).
	gpu := perf.GPUDecision{Policy: perf.GPUNever, Reason: "not Linux"}
	if runtime.GOOS == "linux" {
		gpu = perf.DecideGPU("/sys/class/drm", perf.OSEnv)
	}
	start := time.Now()
	var cfg struct {
		Info struct{ ProductVersion string }
	}
	if err := json.Unmarshal(wailsJSON, &cfg); err != nil {
		log.Fatal(err)
	}
	a, err := app.New(cfg.Info.ProductVersion)
	if err != nil {
		log.Fatal(err)
	}
	app.InitAgentEnv(a)
	if runtime.GOOS == "linux" {
		// A blank-window report starts from this line in the app log.
		app.LogInfo(a, "linux webview gpu", "policy", gpu.Policy, "reason", gpu.Reason)
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
		Linux: &linux.Options{
			Icon: appIcon,
			// The window's app id on Wayland. Its icon comes from a desktop
			// entry named Davish.desktop (build/linux/install-desktop.sh,
			// shipped in the tarball), not from Icon above (#305).
			ProgramName: "Davish",
			// Always unless the user or the NVIDIA workaround says otherwise
			// (perf.DecideGPU, docs/adr/0023).
			WebviewGpuPolicy: webviewPolicy(gpu.Policy),
		},
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
		// Files dropped from the OS upload into the Files window under the
		// pointer. The webview must not open them as a page instead.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		ErrorFormatter: app.ErrorFormatter(a),
		OnStartup:      a.Startup,
		// make bench reads this line to time the start; nothing else does.
		OnDomReady: func(context.Context) {
			if os.Getenv("DAVISH_BENCH") != "" {
				fmt.Fprintf(os.Stderr, "davish-bench ready_ms=%d\n", time.Since(start).Milliseconds())
			}
		},
		OnShutdown: a.Shutdown,
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

// webviewPolicy maps perf's policy name to Wails'.
func webviewPolicy(p string) linux.WebviewGpuPolicy {
	switch p {
	case perf.GPUAlways:
		return linux.WebviewGpuPolicyAlways
	case perf.GPUOnDemand:
		return linux.WebviewGpuPolicyOnDemand
	default:
		return linux.WebviewGpuPolicyNever
	}
}
