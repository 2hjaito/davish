// Package app is the thin facade bound to the Wails runtime. It only
// receives parameters, calls services and returns results.
package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/conn"
	"davish/internal/store"
	"davish/internal/sysmon"
	"davish/internal/term"
)

// App is bound to the frontend; every exported method becomes a binding.
type App struct {
	ctx context.Context
	// wails is the runtime context for events from background goroutines;
	// nil until Startup, and EventsEmit aborts the process on any other ctx.
	wails atomic.Pointer[context.Context]

	store   *store.Store
	secrets *store.Secrets
	conns   *conn.Manager
	terms   *term.Service
	sysmon  *sysmon.Service
}

// New wires the services with their config under os.UserConfigDir()/davish.
func New() (*App, error) {
	dir, err := store.DefaultDir()
	if err != nil {
		return nil, err
	}
	// TODO(M10): JSON log file with rotation under os.UserCacheDir(), per CONVENTIONS.md.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	a := &App{ctx: context.Background()}
	a.secrets = store.NewSecrets(store.OSKeyring{})
	a.store = store.New(dir, a.secrets)
	hostKeys := conn.NewHostKeys(filepath.Join(dir, "known_hosts"))
	a.conns = conn.NewManager(a.store, a.secrets, hostKeys, emitter{a}, log, conn.Options{})
	a.terms = term.New(a.conns, emitter{a}, log)
	a.sysmon = sysmon.New(a.conns, emitter{a}, log, sysmon.Options{})
	return a, nil
}

// Startup is called by Wails once the webview is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.wails.Store(&ctx)
}

// Shutdown is called by Wails before the process exits.
func (a *App) Shutdown(_ context.Context) {
	a.sysmon.Close()
	a.terms.CloseAll()
	a.conns.Close()
}

// emitter lets services push events. It is a separate type because Wails
// binds every exported method of App, and the frontend must not be able to
// forge events such as conn:status.
type emitter struct{ a *App }

func (e emitter) Emit(name string, payload any) {
	if ctx := e.a.wails.Load(); ctx != nil {
		runtime.EventsEmit(*ctx, name, payload)
	}
}
