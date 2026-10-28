// Package store persists non-secret config as JSON and secrets in the OS keychain.
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"davish/internal/apperr"
)

// AuthMethod is how the app authenticates to a server.
type AuthMethod string

// Supported auth methods, tried as configured per server (see conn).
const (
	AuthAgent    AuthMethod = "agent"
	AuthKey      AuthMethod = "key"
	AuthPassword AuthMethod = "password"
)

// Server is one saved server. It never holds a secret: passwords and key
// passphrases live in the keychain (see Secrets).
type Server struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Host string     `json:"host"`
	Port int        `json:"port"`
	User string     `json:"user"`
	Auth AuthMethod `json:"auth"`
	// KeyPath points at the user's private key; the key itself is never copied.
	KeyPath string `json:"keyPath,omitempty"`
	Color   string `json:"color,omitempty"`
}

// file is the on-disk layout of servers.json. Slice order is display order.
type file struct {
	Version int      `json:"version"`
	Servers []Server `json:"servers"`
}

const (
	fileVersion = 1
	serversFile = "servers.json"
	defaultPort = 22
)

// Store reads and writes servers.json in dir.
type Store struct {
	dir     string
	secrets *Secrets

	// mu serializes read-modify-write cycles on servers.json.
	mu sync.Mutex
}

// New returns a Store keeping its files in dir and secrets in secrets.
func New(dir string, secrets *Secrets) *Store {
	return &Store{dir: dir, secrets: secrets}
}

// DefaultDir is os.UserConfigDir()/davish.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(base, "davish"), nil
}

// Servers returns all saved servers in display order.
func (s *Store) Servers(ctx context.Context) ([]Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	return f.Servers, nil
}

// Server returns one saved server, or NOT_FOUND.
func (s *Store) Server(ctx context.Context, id string) (Server, error) {
	all, err := s.Servers(ctx)
	if err != nil {
		return Server{}, err
	}
	i := slices.IndexFunc(all, func(x Server) bool { return x.ID == id })
	if i < 0 {
		return Server{}, apperr.New(apperr.NotFound, "server not found")
	}
	return all[i], nil
}

// Save adds srv, or replaces the server with the same ID. A new server gets
// an ID and goes to the end of the list. It returns the stored server.
func (s *Store) Save(ctx context.Context, srv Server) (Server, error) {
	if err := ctx.Err(); err != nil {
		return Server{}, err
	}
	srv = normalize(srv)
	if err := validate(srv); err != nil {
		return Server{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return Server{}, err
	}
	if srv.ID == "" {
		if srv.ID, err = newID(); err != nil {
			return Server{}, err
		}
		f.Servers = append(f.Servers, srv)
	} else {
		i := slices.IndexFunc(f.Servers, func(x Server) bool { return x.ID == srv.ID })
		if i < 0 {
			return Server{}, apperr.New(apperr.NotFound, "server not found")
		}
		f.Servers[i] = srv
	}
	if err := s.write(f); err != nil {
		return Server{}, err
	}
	return srv, nil
}

// Delete removes the server and every secret stored for it.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(f.Servers, func(x Server) bool { return x.ID == id })
	if i < 0 {
		return apperr.New(apperr.NotFound, "server not found")
	}
	f.Servers = slices.Delete(f.Servers, i, i+1)
	if err := s.write(f); err != nil {
		return err
	}
	// The config no longer references the server, so a leftover secret would
	// be unreachable; still report it so the user knows the keychain is dirty.
	return s.secrets.DeleteAll(id)
}

// ForgetSecrets removes the passwords and passphrases of every saved
// server but keeps the servers: the app lock's reset (ADR 0007).
func (s *Store) ForgetSecrets(ctx context.Context) error {
	servers, err := s.Servers(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, srv := range servers {
		if err := s.secrets.DeleteAll(srv.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Reorder sets the display order. ids must list every saved server once.
func (s *Store) Reorder(ctx context.Context, ids []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.load()
	if err != nil {
		return err
	}
	byID := make(map[string]Server, len(f.Servers))
	for _, srv := range f.Servers {
		byID[srv.ID] = srv
	}
	if len(ids) != len(byID) {
		return apperr.New(apperr.InvalidInput, "order must list every server once")
	}
	ordered := make([]Server, 0, len(ids))
	for _, id := range ids {
		srv, ok := byID[id]
		if !ok {
			return apperr.New(apperr.InvalidInput, "order must list every server once")
		}
		delete(byID, id)
		ordered = append(ordered, srv)
	}
	f.Servers = ordered
	return s.write(f)
}

func (s *Store) path() string { return filepath.Join(s.dir, serversFile) }

// load reads servers.json; a missing file is an empty list.
func (s *Store) load() (file, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, fs.ErrNotExist) {
		return file{Version: fileVersion}, nil
	}
	if err != nil {
		return file{}, fmt.Errorf("read %s: %w", serversFile, err)
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return file{}, apperr.Wrap(apperr.StoreCorrupt, "servers.json is not valid JSON", err)
	}
	if f.Version > fileVersion {
		return file{}, apperr.New(apperr.StoreCorrupt, "servers.json was written by a newer version")
	}
	f.Version = fileVersion
	return f, nil
}

// write replaces servers.json atomically.
func (s *Store) write(f file) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", serversFile, err)
	}
	return writeAtomic(s.dir, serversFile, data)
}

// writeAtomic replaces dir/name: a crash leaves either the old or the new
// file, never a truncated one.
func writeAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*.davish-tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck // gone after a successful rename

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close() //nolint:errcheck,gosec // already failing
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close() //nolint:errcheck,gosec // already failing
		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	return nil
}

func normalize(srv Server) Server {
	srv.Name = strings.TrimSpace(srv.Name)
	srv.Host = strings.TrimSpace(srv.Host)
	srv.User = strings.TrimSpace(srv.User)
	srv.KeyPath = strings.TrimSpace(srv.KeyPath)
	if srv.Port == 0 {
		srv.Port = defaultPort
	}
	if srv.Auth != AuthKey {
		srv.KeyPath = ""
	}
	return srv
}

var (
	// hostname follows RFC 1123 labels; IP literals are checked separately.
	hostname = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)
	// username is a portable POSIX login name.
	username = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,31}$`)
	hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

func validate(srv Server) error {
	invalid := func(msg string) error { return apperr.New(apperr.InvalidInput, msg) }
	switch {
	case srv.Name == "" || len(srv.Name) > 64:
		return invalid("name must be 1-64 characters")
	case net.ParseIP(srv.Host) == nil && !hostname.MatchString(srv.Host):
		return invalid("host must be a hostname or IP address")
	case srv.Port < 1 || srv.Port > 65535:
		return invalid("port must be 1-65535")
	case !username.MatchString(srv.User):
		return invalid("user is not a valid login name")
	case srv.Auth != AuthAgent && srv.Auth != AuthKey && srv.Auth != AuthPassword:
		return invalid("unknown auth method")
	case srv.Auth == AuthKey && srv.KeyPath == "":
		return invalid("key auth needs a key path")
	case srv.Color != "" && !hexColor.MatchString(srv.Color):
		return invalid("color must be #rrggbb")
	}
	return nil
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate server id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
