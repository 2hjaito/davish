package app

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/ansibleinv"
)

// AnsibleDefaultInventory is the inventory Ansible would use on this
// machine, "" when there is none.
func (a *App) AnsibleDefaultInventory() string {
	return ansibleinv.Default()
}

// AnsiblePickInventory asks the user for an inventory file; "" when canceled.
func (a *App) AnsiblePickInventory(title string) (string, error) {
	if err := a.lock.Check(); err != nil {
		return "", err
	}
	opts := runtime.OpenDialogOptions{Title: title}
	if p := ansibleinv.Default(); p != "" {
		opts.DefaultDirectory = filepath.Dir(p)
	}
	return runtime.OpenFileDialog(a.ctx, opts)
}

// AnsibleInventoryHosts reads the inventory at path, marking the hosts
// already saved.
func (a *App) AnsibleInventoryHosts(path string) (ansibleinv.Listing, error) {
	inv, err := ansibleinv.Load(path, localUser())
	if err != nil {
		return ansibleinv.Listing{}, err
	}
	saved, err := a.store.Servers(a.ctx)
	if err != nil {
		return ansibleinv.Listing{}, err
	}
	return ansibleinv.List(path, inv, saved), nil
}

// ImportAnsibleHosts saves the chosen inventory hosts, and the jump hosts
// they need, and returns the servers added.
func (a *App) ImportAnsibleHosts(path string, names []string) ([]ServerView, error) {
	if err := a.lock.Check(); err != nil {
		return nil, err
	}
	name := localUser()
	inv, err := ansibleinv.Load(path, name)
	if err != nil {
		return nil, err
	}
	added, err := ansibleinv.Import(a.ctx, a.store, inv, names, name)
	views := make([]ServerView, 0, len(added))
	for _, s := range added {
		v, verr := a.view(s)
		if verr != nil {
			return views, verr
		}
		views = append(views, v)
	}
	return views, err
}

// localUser is ssh's default login, for hosts the inventory gives none.
func localUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}
