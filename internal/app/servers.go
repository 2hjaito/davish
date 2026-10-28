package app

import (
	"davish/internal/conn"
	"davish/internal/store"
)

// ServerView is a saved server as the frontend sees it: no secrets, only
// whether they are stored.
type ServerView struct {
	store.Server
	HasPassword   bool       `json:"hasPassword"`
	HasPassphrase bool       `json:"hasPassphrase"`
	State         conn.State `json:"state"`
}

// ServerInput is what the add/edit dialog sends. Password and Passphrase are
// optional: empty keeps the stored one. They go straight to the keychain.
type ServerInput struct {
	store.Server
	Password   string `json:"password"`
	Passphrase string `json:"passphrase"`
}

// ListServers returns saved servers in display order with their state.
func (a *App) ListServers() ([]ServerView, error) {
	all, err := a.store.Servers(a.ctx)
	if err != nil {
		return nil, err
	}
	views := make([]ServerView, 0, len(all))
	for _, s := range all {
		v, err := a.view(s)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

// SaveServer adds or updates a server and stores any secret given with it.
func (a *App) SaveServer(in ServerInput) (ServerView, error) {
	if err := a.lock.Check(); err != nil {
		return ServerView{}, err
	}
	srv, err := a.store.Save(a.ctx, in.Server)
	if err != nil {
		return ServerView{}, err
	}
	if in.Password != "" {
		if err := a.secrets.Set(srv.ID, store.SecretPassword, in.Password); err != nil {
			return ServerView{}, err
		}
	}
	if in.Passphrase != "" {
		if err := a.secrets.Set(srv.ID, store.SecretPassphrase, in.Passphrase); err != nil {
			return ServerView{}, err
		}
	}
	return a.view(srv)
}

// DeleteServer disconnects and removes a server with its secrets.
func (a *App) DeleteServer(id string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	a.conns.Disconnect(id)
	return a.store.Delete(a.ctx, id)
}

// ReorderServers sets the display order.
func (a *App) ReorderServers(ids []string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.store.Reorder(a.ctx, ids)
}

// Connect opens the server's connection; state changes arrive as conn:status.
func (a *App) Connect(id string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.conns.Connect(a.ctx, id)
}

// Disconnect closes the server's connection.
func (a *App) Disconnect(id string) {
	a.conns.Disconnect(id)
}

// AcceptHostKey trusts the host key the user approved, then connects.
func (a *App) AcceptHostKey(id, fingerprint string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.conns.AcceptHostKey(a.ctx, id, fingerprint)
}

func (a *App) view(s store.Server) (ServerView, error) {
	hasPw, err := a.secrets.Has(s.ID, store.SecretPassword)
	if err != nil {
		return ServerView{}, err
	}
	hasPp, err := a.secrets.Has(s.ID, store.SecretPassphrase)
	if err != nil {
		return ServerView{}, err
	}
	return ServerView{Server: s, HasPassword: hasPw, HasPassphrase: hasPp, State: a.conns.State(s.ID)}, nil
}
