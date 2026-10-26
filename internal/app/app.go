// Package app is the thin facade bound to the Wails runtime. It only
// receives parameters, calls services and returns results.
package app

import "context"

// App is bound to the frontend; every exported method becomes a binding.
type App struct {
	ctx context.Context
}

// New creates the facade. Services are injected here as they are added.
func New() *App {
	return &App{}
}

// Startup is called by Wails once the webview is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called by Wails before the process exits.
func (a *App) Shutdown(_ context.Context) {}
