// Package app is the thin facade bound to the Wails runtime. It only
// receives parameters, calls services and returns results.
package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/applog"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/cron"
	"davish/internal/diag"
	"davish/internal/disk"
	"davish/internal/docker"
	"davish/internal/fs"
	"davish/internal/git"
	"davish/internal/lock"
	"davish/internal/logs"
	"davish/internal/nginx"
	"davish/internal/packages"
	"davish/internal/ports"
	"davish/internal/security"
	"davish/internal/sshkeys"
	"davish/internal/store"
	"davish/internal/svc"
	"davish/internal/sysmon"
	"davish/internal/term"
	"davish/internal/transfer"
	"davish/internal/update"
)

// App is bound to the frontend; every exported method becomes a binding.
type App struct {
	ctx context.Context
	// wails is the runtime context for events from background goroutines;
	// nil until Startup, and EventsEmit aborts the process on any other ctx.
	wails atomic.Pointer[context.Context]

	store     *store.Store
	settings  *store.Settings
	secrets   *store.Secrets
	conns     *conn.Manager
	terms     *term.Service
	sysmon    *sysmon.Service
	files     *fs.Service
	transfers *transfer.Service
	audit     *audit.Log
	services  *svc.Service
	docker    *docker.Service
	logs      *logs.Service
	cron      *cron.Service
	nginx     *nginx.Service
	lock      *lock.Service
	keys      *sshkeys.Service
	packages  *packages.Service
	git       *git.Service
	ports     *ports.Lister
	tunnels   *ports.Tunnels
	browser   *ports.Browser
	disk      *disk.Service
	security  *security.Service
	updates   *update.Service
	diag      *diag.Service
	// logFile is the app log, nil when it could not be opened.
	logFile *applog.Writer
	log     *slog.Logger
	version string
}

// New wires the services with their config under os.UserConfigDir()/davish.
func New(version string) (*App, error) {
	dir, err := store.DefaultDir()
	if err != nil {
		return nil, err
	}
	a := &App{ctx: context.Background(), version: version}
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if logDir, err := applog.DefaultDir(); err == nil {
		if w, err := applog.Open(logDir); err == nil {
			a.logFile, log = w, applog.New(w)
		}
	}
	a.log = log
	log.Info("davish started", "version", version, "os", goruntime.GOOS, "arch", goruntime.GOARCH)
	a.secrets = store.NewSecrets(store.OSKeyring{})
	a.store = store.New(dir, a.secrets)
	a.lock = lock.New(store.OSKeyring{}, lock.NewBiometric(), emitter{a}, log,
		func() error { return a.store.ForgetSecrets(context.Background()) }, lock.Options{})
	a.settings = store.NewSettings(dir, log)
	hostKeys := conn.NewHostKeys(filepath.Join(dir, "known_hosts"))
	a.conns = conn.NewManager(a.store, a.secrets, hostKeys, emitter{a}, log, conn.Options{})
	a.terms = term.New(a.conns, emitter{a}, log)
	a.sysmon = sysmon.New(a.conns, emitter{a}, log, sysmon.Options{})
	a.audit = audit.New(emitter{a})
	runner := audit.NewRunner(a.conns, a.audit)
	runner.LogTo(log)
	a.files = fs.New(a.conns, runner, log)
	a.services = svc.New(runner)
	a.docker = docker.New(runner)
	a.cron = cron.New(runner, a.conns)
	a.nginx = nginx.New(runner, a.conns, emitter{a})
	a.keys = sshkeys.New(a.conns, runner)
	a.packages = packages.New(runner, emitter{a})
	a.git = git.New(runner)
	// The app replaced by the last update is no longer needed.
	update.Cleanup(goruntime.GOOS, os.Executable)
	a.updates = update.New(version, emitter{a}, log, update.Options{})
	a.ports = ports.NewLister(runner)
	a.disk = disk.New(runner)
	a.security = security.New(runner)
	a.diag = diag.New(runner, a.conns)
	a.tunnels = ports.NewTunnels(a.conns, emitter{a}, log)
	a.browser = ports.NewBrowser(a.conns, log)
	a.logs = logs.New(a.conns, a.audit, emitter{a}, log)
	if a.transfers, err = transfer.New(a.conns, emitter{a}, log, transfer.Options{}); err != nil {
		return nil, err
	}
	return a, nil
}

// Startup is called by Wails once the webview is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.wails.Store(&ctx)
	a.lock.Watch(ctx)
}

// Shutdown is called by Wails before the process exits.
func (a *App) Shutdown(_ context.Context) {
	a.lock.Close()
	a.sysmon.Close()
	a.logs.Close()
	a.transfers.Close()
	a.terms.CloseAll()
	a.tunnels.Close()
	a.browser.Close()
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
