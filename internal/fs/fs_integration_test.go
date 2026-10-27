//go:build integration

package fs_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/apperr"
	"davish/internal/conn"
	"davish/internal/fs"
	"davish/internal/shellx"
	"davish/internal/store"
	"davish/internal/testsshd"
)

type keyring map[string]string

func (k keyring) Set(s, u, v string) error { k[s+u] = v; return nil }
func (k keyring) Get(s, u string) (string, error) {
	if v, ok := k[s+u]; ok {
		return v, nil
	}
	return "", store.ErrNoSecret
}
func (k keyring) Delete(s, u string) error { delete(k, s+u); return nil }

type discard struct{}

func (discard) Emit(string, any) {}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type env struct {
	m   *conn.Manager
	svc *fs.Service
	id  string
	dir string // scratch directory on the server, removed after the test
}

func setup(t *testing.T) env {
	t.Helper()
	dir := t.TempDir()
	secrets := store.NewSecrets(keyring{})
	st := store.New(dir, secrets)
	srv, err := st.Save(context.Background(), store.Server{
		Name: "test", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(),
	})
	if err != nil {
		t.Fatal(err)
	}
	hk := filepath.Join(dir, "known_hosts")
	line := knownhosts.Line([]string{knownhosts.Normalize("127.0.0.1:2222")}, testsshd.HostKey(t))
	if err := os.WriteFile(hk, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), discard{}, quiet(), conn.Options{})
	t.Cleanup(m.Close)
	if err := m.Connect(context.Background(), srv.ID); err != nil {
		t.Fatal(err)
	}

	e := env{m: m, svc: fs.New(m, quiet()), id: srv.ID}
	home, err := e.svc.Home(context.Background(), e.id)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	e.dir = path.Join(home, "fs-test-"+hex.EncodeToString(b))
	e.run(t, shellx.Cmd("mkdir", e.dir).String())
	t.Cleanup(func() { e.run(t, shellx.Cmd("rm", "-rf", "--", e.dir).String()) })
	return e
}

func (e env) run(t *testing.T, cmd string) string {
	t.Helper()
	client, err := e.m.Client(e.id)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		t.Fatalf("%s: %v\n%s", cmd, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (e env) path(name string) string { return path.Join(e.dir, name) }

func wantCode(t *testing.T, err error, code apperr.Code) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != code {
		t.Fatalf("err = %v; want %s", err, code)
	}
}

func TestListAndStat(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.run(t, "cd "+shellx.Quote(e.dir)+" && mkdir sub && printf hello > 'a file.txt' && chmod 640 'a file.txt' && ln -s sub to-sub && ln -s missing broken")

	list, err := e.svc.List(ctx, e.id, e.dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]fs.Entry{}
	var names []string
	for _, en := range list {
		got[en.Name] = en
		names = append(names, en.Name)
	}
	if strings.Join(names, ",") != "a file.txt,broken,sub,to-sub" {
		t.Fatalf("names = %v", names)
	}
	if f := got["a file.txt"]; f.Kind != fs.KindFile || f.Size != 5 || f.Mode != "-rw-r-----" || f.Perm != 0o640 || f.Path != e.path("a file.txt") {
		t.Errorf("file = %+v", f)
	}
	if d := got["sub"]; d.Kind != fs.KindDir || !d.Dir {
		t.Errorf("dir = %+v", d)
	}
	if l := got["to-sub"]; l.Kind != fs.KindLink || !l.Dir || l.Target != "sub" {
		t.Errorf("link = %+v", l)
	}
	if l := got["broken"]; l.Kind != fs.KindLink || l.Dir || l.Target != "missing" {
		t.Errorf("broken link = %+v", l)
	}

	st, err := e.svc.Stat(ctx, e.id, e.path("to-sub"))
	if err != nil || st.Kind != fs.KindLink || !st.Dir {
		t.Errorf("stat link = %+v, %v", st, err)
	}
	_, err = e.svc.Stat(ctx, e.id, e.path("nope"))
	wantCode(t, err, apperr.NotFound)
	_, err = e.svc.List(ctx, e.id, "relative/path")
	wantCode(t, err, apperr.InvalidInput)
	_, err = e.svc.List(ctx, e.id, "/root")
	wantCode(t, err, apperr.FSPermissionDenied)
}

func TestReadWrite(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	p := e.path("conf")

	if _, err := e.svc.Write(ctx, e.id, p, []byte("v1\n")); err != nil {
		t.Fatal(err)
	}
	e.run(t, "chmod 4751 "+shellx.Quote(p))
	en, err := e.svc.Write(ctx, e.id, p, []byte("v2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if en.Perm != 0o4751 {
		t.Errorf("mode after write = %o; want 4751", en.Perm)
	}
	f, err := e.svc.Read(ctx, e.id, p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Data, []byte("v2\n")) || f.Size != 3 {
		t.Errorf("read = %q (%d bytes)", f.Data, f.Size)
	}
	if out := e.run(t, "ls -A "+shellx.Quote(e.dir)); out != "conf" {
		t.Errorf("leftovers in dir: %q", out)
	}

	// Writing through a link replaces the file it points to, not the link.
	e.run(t, "cd "+shellx.Quote(e.dir)+" && ln -s conf link")
	if _, err := e.svc.Write(ctx, e.id, e.path("link"), []byte("v3\n")); err != nil {
		t.Fatal(err)
	}
	if out := e.run(t, "cd "+shellx.Quote(e.dir)+" && readlink link && cat conf"); out != "conf\nv3" {
		t.Errorf("after write through link: %q", out)
	}

	_, err = e.svc.Write(ctx, e.id, e.dir, []byte("x"))
	wantCode(t, err, apperr.InvalidInput)
	_, err = e.svc.Read(ctx, e.id, e.dir)
	wantCode(t, err, apperr.InvalidInput)
	_, err = e.svc.Write(ctx, e.id, "/etc/hostname", []byte("x"))
	wantCode(t, err, apperr.FSPermissionDenied)
	_, err = e.svc.Read(ctx, e.id, "/etc/shadow")
	wantCode(t, err, apperr.FSPermissionDenied)

	e.run(t, "head -c 11000000 /dev/zero > "+shellx.Quote(e.path("big")))
	_, err = e.svc.Read(ctx, e.id, e.path("big"))
	wantCode(t, err, apperr.FSTooLarge)
}

func TestMkdirRename(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	if err := e.svc.Mkdir(ctx, e.id, e.path("d")); err != nil {
		t.Fatal(err)
	}
	wantCode(t, e.svc.Mkdir(ctx, e.id, e.path("d")), apperr.FSExists)
	wantCode(t, e.svc.Mkdir(ctx, e.id, e.path("no/parent")), apperr.NotFound)

	if _, err := e.svc.Write(ctx, e.id, e.path("a"), []byte("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Write(ctx, e.id, e.path("b"), []byte("b")); err != nil {
		t.Fatal(err)
	}
	wantCode(t, e.svc.Rename(ctx, e.id, e.path("a"), e.path("b")), apperr.FSExists)
	if err := e.svc.Rename(ctx, e.id, e.path("a"), e.path("d/a")); err != nil {
		t.Fatal(err)
	}
	if out := e.run(t, "cat "+shellx.Quote(e.path("d/a"))+" "+shellx.Quote(e.path("b"))); out != "ab" {
		t.Errorf("after rename: %q", out)
	}
}

func TestRemove(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.run(t, "cd "+shellx.Quote(e.dir)+" && mkdir empty full && touch full/x \"it's\" && ln -s full link")

	tests := []struct {
		name     string
		nonEmpty bool
		prog     string
	}{
		{"it's", false, "rm -- "},
		{"link", false, "rm -- "},
		{"empty", false, "rmdir -- "},
		{"full", true, "rm -r -- "},
	}
	for _, tt := range tests {
		p := e.path(tt.name)
		plan, err := e.svc.PlanRemove(ctx, e.id, p)
		if err != nil {
			t.Fatal(err)
		}
		if want := tt.prog + shellx.Quote(p); plan.Command != want || plan.NonEmpty != tt.nonEmpty {
			t.Errorf("%s: plan = %q nonEmpty=%v; want %q", tt.name, plan.Command, plan.NonEmpty, want)
		}
		if err := e.svc.Remove(ctx, e.id, p, plan.Command); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
	}
	if out := e.run(t, "ls -A "+shellx.Quote(e.dir)); out != "" {
		t.Errorf("left after remove: %q", out)
	}
}

func TestRemoveRefusesStaleOrForeignCommand(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	p := e.path("d")
	e.run(t, "mkdir "+shellx.Quote(p))

	plan, err := e.svc.PlanRemove(ctx, e.id, p)
	if err != nil {
		t.Fatal(err)
	}
	e.run(t, "touch "+shellx.Quote(p+"/new"))
	wantCode(t, e.svc.Remove(ctx, e.id, p, plan.Command), apperr.FSStale)
	wantCode(t, e.svc.Remove(ctx, e.id, p, "rm -rf ~"), apperr.FSStale)

	_, err = e.svc.PlanRemove(ctx, e.id, "/")
	wantCode(t, err, apperr.InvalidInput)

	plan, err = e.svc.PlanRemove(ctx, e.id, "/etc/hostname")
	if err != nil {
		t.Fatal(err)
	}
	wantCode(t, e.svc.Remove(ctx, e.id, "/etc/hostname", plan.Command), apperr.FSPermissionDenied)
}
