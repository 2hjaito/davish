package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"davish/internal/apperr"
)

// fakeKeyring is an in-memory Keyring; tests never touch the real keychain.
type fakeKeyring struct {
	m   map[string]string
	err error
}

func newFakeKeyring() *fakeKeyring { return &fakeKeyring{m: map[string]string{}} }

func (f *fakeKeyring) Set(service, user, secret string) error {
	if f.err != nil {
		return f.err
	}
	f.m[service+"|"+user] = secret
	return nil
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.m[service+"|"+user]
	if !ok {
		return "", ErrNoSecret
	}
	return v, nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.m[service+"|"+user]; !ok {
		return ErrNoSecret
	}
	delete(f.m, service+"|"+user)
	return nil
}

func newTestStore(t *testing.T) (*Store, *fakeKeyring, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "davish")
	kr := newFakeKeyring()
	return New(dir, NewSecrets(kr)), kr, dir
}

func vps() Server {
	return Server{Name: "tsc-vps", Host: "vps.example.com", User: "dangth", Auth: AuthAgent}
}

func codeOf(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestEmptyStore(t *testing.T) {
	s, _, _ := newTestStore(t)
	got, err := s.Servers(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatalf("Servers() = %v, %v; want empty, nil", got, err)
	}
}

func TestSaveAddsThenUpdates(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newTestStore(t)

	added, err := s.Save(ctx, vps())
	if err != nil {
		t.Fatal(err)
	}
	if added.ID == "" || added.Port != 22 {
		t.Fatalf("added = %+v; want an ID and default port 22", added)
	}

	added.Name = "  production  "
	if _, err := s.Save(ctx, added); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Servers(ctx)
	if len(got) != 1 || got[0].Name != "production" {
		t.Fatalf("Servers() = %+v; want one server named production", got)
	}

	missing := added
	missing.ID = "nope"
	if _, err := s.Save(ctx, missing); codeOf(err) != apperr.NotFound {
		t.Errorf("update unknown id: err = %v; want NOT_FOUND", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Server)
	}{
		{"empty name", func(s *Server) { s.Name = " " }},
		{"host with space", func(s *Server) { s.Host = "evil host" }},
		{"host with shell", func(s *Server) { s.Host = "$(reboot)" }},
		{"port too big", func(s *Server) { s.Port = 70000 }},
		{"bad user", func(s *Server) { s.User = "root; rm" }},
		{"unknown auth", func(s *Server) { s.Auth = "magic" }},
		{"key without path", func(s *Server) { s.Auth = AuthKey }},
		{"bad color", func(s *Server) { s.Color = "red" }},
	}
	s, _, _ := newTestStore(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := vps()
			tt.edit(&srv)
			if _, err := s.Save(context.Background(), srv); codeOf(err) != apperr.InvalidInput {
				t.Errorf("err = %v; want INVALID_INPUT", err)
			}
		})
	}

	for _, host := range []string{"10.0.0.5", "::1", "my-vps", "a.b.example.org"} {
		srv := vps()
		srv.Host = host
		if _, err := s.Save(context.Background(), srv); err != nil {
			t.Errorf("host %q rejected: %v", host, err)
		}
	}
}

func TestKeyPathOnlyForKeyAuth(t *testing.T) {
	s, _, _ := newTestStore(t)
	srv := vps()
	srv.KeyPath = "/home/u/.ssh/id_ed25519"
	got, err := s.Save(context.Background(), srv)
	if err != nil || got.KeyPath != "" {
		t.Fatalf("agent auth kept key path %q (err %v)", got.KeyPath, err)
	}
}

func TestFileHoldsNoSecretsAndIsPrivate(t *testing.T) {
	ctx := context.Background()
	s, _, dir := newTestStore(t)
	srv, err := s.Save(ctx, vps())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.secrets.Set(srv.ID, SecretPassword, "hunter2-very-secret"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, serversFile)) //nolint:gosec // path is under t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Fatal("servers.json contains the password")
	}

	if runtime.GOOS == "windows" {
		return // POSIX modes do not apply
	}
	for path, want := range map[string]os.FileMode{dir: 0o700, filepath.Join(dir, serversFile): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o; want %o", filepath.Base(path), got, want)
		}
	}
}

func TestNoTempFilesLeft(t *testing.T) {
	s, _, dir := newTestStore(t)
	for range 3 {
		if _, err := s.Save(context.Background(), vps()); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("config dir has %d entries; want only servers.json", len(entries))
	}
}

func TestDeleteRemovesSecrets(t *testing.T) {
	ctx := context.Background()
	s, kr, _ := newTestStore(t)
	a, _ := s.Save(ctx, vps())
	b, _ := s.Save(ctx, vps())
	_ = s.secrets.Set(a.ID, SecretPassword, "pw-a")
	_ = s.secrets.Set(a.ID, SecretPassphrase, "pp-a")
	_ = s.secrets.Set(b.ID, SecretPassword, "pw-b")

	if err := s.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Servers(ctx)
	if len(got) != 1 || got[0].ID != b.ID {
		t.Fatalf("Servers() = %+v; want only b", got)
	}
	if len(kr.m) != 1 {
		t.Errorf("keychain has %d entries; want only b's password", len(kr.m))
	}
	if err := s.Delete(ctx, a.ID); codeOf(err) != apperr.NotFound {
		t.Errorf("second delete: err = %v; want NOT_FOUND", err)
	}
}

func TestReorder(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newTestStore(t)
	a, _ := s.Save(ctx, vps())
	b, _ := s.Save(ctx, vps())
	c, _ := s.Save(ctx, vps())

	if err := s.Reorder(ctx, []string{c.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Servers(ctx)
	if got[0].ID != c.ID || got[1].ID != a.ID || got[2].ID != b.ID {
		t.Errorf("order = %s %s %s; want c a b", got[0].ID, got[1].ID, got[2].ID)
	}

	for name, ids := range map[string][]string{
		"missing one": {a.ID, b.ID},
		"duplicate":   {a.ID, a.ID, b.ID},
		"unknown":     {a.ID, b.ID, "zzz"},
	} {
		if err := s.Reorder(ctx, ids); codeOf(err) != apperr.InvalidInput {
			t.Errorf("%s: err = %v; want INVALID_INPUT", name, err)
		}
	}
}

func TestCorruptFile(t *testing.T) {
	s, _, dir := newTestStore(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"not json":      "{oops",
		"newer version": `{"version": 99, "servers": []}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, serversFile), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Servers(context.Background()); codeOf(err) != apperr.StoreCorrupt {
			t.Errorf("%s: err = %v; want STORE_CORRUPT", name, err)
		}
	}
}

func TestCanceledContext(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Save(ctx, vps()); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
}
