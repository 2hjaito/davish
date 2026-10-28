package sshconfig

import (
	"context"
	"fmt"
	"slices"

	"davish/internal/apperr"
	"davish/internal/store"
)

// Saver is the part of store.Store the import needs.
type Saver interface {
	Servers(ctx context.Context) ([]store.Server, error)
	Save(ctx context.Context, srv store.Server) (store.Server, error)
}

// Candidate is a config host as the import dialog lists it.
type Candidate struct {
	Host
	// Exists is set when a saved server already has this user@host:port.
	Exists bool `json:"exists"`
}

// Candidates marks which hosts are already saved.
func Candidates(hosts []Host, saved []store.Server) []Candidate {
	out := make([]Candidate, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, Candidate{Host: h, Exists: find(saved, h.User, h.HostName, h.Port) >= 0})
	}
	return out
}

func find(saved []store.Server, user, host string, port int) int {
	return slices.IndexFunc(saved, func(s store.Server) bool {
		return s.User == user && s.Host == host && s.Port == port
	})
}

// maxHops bounds a ProxyJump chain, and catches aliases that jump through
// each other.
const maxHops = 8

type importer struct {
	ctx   context.Context
	saver Saver
	hosts map[string]Host
	saved []store.Server
	added []store.Server
	user  string
}

// Import saves the chosen aliases, and any jump host they need, as servers.
// A host already saved (same user@host:port) is reused, never changed. It
// returns the servers it added.
func Import(ctx context.Context, saver Saver, hosts []Host, aliases []string, localUser string) ([]store.Server, error) {
	saved, err := saver.Servers(ctx)
	if err != nil {
		return nil, err
	}
	im := &importer{ctx: ctx, saver: saver, hosts: map[string]Host{}, saved: saved, user: localUser}
	for _, h := range hosts {
		im.hosts[h.Alias] = h
	}
	for _, a := range aliases {
		h, ok := im.hosts[a]
		if !ok {
			return im.added, apperr.New(apperr.NotFound, fmt.Sprintf("host %q is not in ~/.ssh/config", a))
		}
		if _, err := im.ensure(h, 0); err != nil {
			return im.added, err
		}
	}
	return im.added, nil
}

// ensure returns the ID of the server for h, saving it (and its jump chain)
// when it is new.
func (im *importer) ensure(h Host, depth int) (string, error) {
	if depth > maxHops {
		return "", apperr.New(apperr.InvalidInput, "ProxyJump chain is too long or loops")
	}
	if i := find(im.saved, h.User, h.HostName, h.Port); i >= 0 {
		return im.saved[i].ID, nil
	}
	jumpID := ""
	if h.ProxyJump != "" {
		hops, err := ParseJump(h.ProxyJump)
		if err != nil {
			return "", apperr.Wrap(apperr.InvalidInput, "cannot read ProxyJump", err)
		}
		// Each hop is reached through the one before it.
		for _, hop := range hops {
			hh := im.hopHost(hop)
			if jumpID != "" {
				hh.ProxyJump = ""
			}
			id, err := im.ensureVia(hh, jumpID, depth+1)
			if err != nil {
				return "", err
			}
			jumpID = id
		}
	}
	return im.save(h, jumpID)
}

// ensureVia is ensure for a hop after the first, which jumps through prev
// instead of its own ProxyJump.
func (im *importer) ensureVia(h Host, prev string, depth int) (string, error) {
	if prev == "" {
		return im.ensure(h, depth)
	}
	if i := find(im.saved, h.User, h.HostName, h.Port); i >= 0 {
		return im.saved[i].ID, nil
	}
	return im.save(h, prev)
}

// hopHost resolves a hop: a config alias takes its settings, an explicit
// address overrides them.
func (im *importer) hopHost(hop Hop) Host {
	if h, ok := im.hosts[hop.Alias]; ok {
		return h
	}
	h, ok := im.hosts[hop.Host]
	if !ok {
		h = Host{Alias: hop.Host, HostName: hop.Host, User: im.user, Port: 22}
	}
	if hop.User != "" {
		h.User = hop.User
	}
	if hop.Port != 0 {
		h.Port = hop.Port
	}
	return h
}

func (im *importer) save(h Host, jumpID string) (string, error) {
	srv := store.Server{
		Name:   h.Alias,
		Host:   h.HostName,
		Port:   h.Port,
		User:   h.User,
		Auth:   store.AuthAgent,
		JumpID: jumpID,
	}
	if h.IdentityFile != "" {
		srv.Auth, srv.KeyPath = store.AuthKey, h.IdentityFile
	}
	saved, err := im.saver.Save(im.ctx, srv)
	if err != nil {
		return "", err
	}
	im.saved = append(im.saved, saved)
	im.added = append(im.added, saved)
	return saved.ID, nil
}
