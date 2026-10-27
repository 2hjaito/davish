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
	if plan.Sudo != "sudo rm -- /etc/hostname" {
		t.Errorf("sudo = %q", plan.Sudo)
	}
}

func TestCompress(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	src := e.path("my site")
	e.run(t, "mkdir -p "+shellx.Quote(src+"/css")+" && echo hi > "+shellx.Quote(src+"/index.html")+" && echo x > "+shellx.Quote(src+"/css/a.css"))

	for _, f := range []string{fs.FormatZip, fs.FormatTarGz, fs.FormatTar} {
		plan, err := e.svc.PlanCompress(ctx, e.id, src, f)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Output != src+"."+f {
			t.Errorf("%s: output = %q", f, plan.Output)
		}
		if _, err := e.svc.Compress(ctx, e.id, src, f, plan.Command); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	list := e.run(t, "cd "+shellx.Quote(e.dir)+" && zip -sf 'my site.zip' | grep -c 'my site/' && tar -tzf 'my site.tar.gz' | sort && tar -tf 'my site.tar' | wc -l")
	if list != "4\nmy site/\nmy site/css/\nmy site/css/a.css\nmy site/index.html\n4" {
		t.Errorf("archives: %q", list)
	}

	// The next archive of the same thing gets a new name, never overwriting.
	plan, err := e.svc.PlanCompress(ctx, e.id, src, fs.FormatZip)
	if err != nil || plan.Output != src+" (1).zip" {
		t.Errorf("second plan = %+v, %v", plan, err)
	}
	// A plan gone stale is refused.
	e.run(t, "touch "+shellx.Quote(plan.Output))
	_, err = e.svc.Compress(ctx, e.id, src, fs.FormatZip, plan.Command)
	wantCode(t, err, apperr.FSStale)

	_, err = e.svc.PlanCompress(ctx, e.id, src, "rar")
	wantCode(t, err, apperr.InvalidInput)
	_, err = e.svc.PlanCompress(ctx, e.id, "/", fs.FormatTar)
	wantCode(t, err, apperr.InvalidInput)

	_, err = e.svc.PlanCompress(ctx, e.id, e.path("missing"), fs.FormatTar)
	wantCode(t, err, apperr.NotFound)

	// A failed run leaves no partial archive behind.
	locked := e.path("locked")
	e.run(t, "mkdir "+shellx.Quote(locked)+" && echo s > "+shellx.Quote(locked+"/secret")+" && chmod 000 "+shellx.Quote(locked+"/secret"))
	plan, err = e.svc.PlanCompress(ctx, e.id, locked, fs.FormatTar)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Compress(ctx, e.id, locked, fs.FormatTar, plan.Command)
	wantCode(t, err, apperr.FSPermissionDenied)
	if want := shellx.InDir(e.dir, shellx.Cmd("sudo", "tar", "-cf", "locked.tar", "--", "locked")); plan.Sudo != want {
		t.Errorf("sudo = %q, want %q", plan.Sudo, want)
	}
	if out := e.run(t, "ls -A "+shellx.Quote(e.dir)+" | grep -c locked.tar || true"); out != "0" {
		t.Errorf("partial archive left: %s", out)
	}
	e.run(t, "chmod 644 "+shellx.Quote(locked+"/secret"))
}

// The M6 acceptance: edit nginx.conf, save, the mode stays and a backup
// holds the previous version.
func TestSaveNginxConfWithBackup(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	const conf = "/etc/nginx/nginx.conf"
	orig, err := e.svc.Read(ctx, e.id, conf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := e.svc.Write(ctx, e.id, conf, orig.Data); err != nil {
			t.Errorf("restore nginx.conf: %v", err)
		}
		e.run(t, "rm -f /etc/nginx/nginx.conf.bak-*")
	})

	edited := append(append([]byte(nil), orig.Data...), []byte("\n# edited by davish test\n")...)
	res, err := e.svc.Save(ctx, e.id, conf, edited, fs.SaveOptions{Backup: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Perm != orig.Perm {
		t.Errorf("mode = %o; want %o", res.Entry.Perm, orig.Perm)
	}
	if !strings.HasPrefix(res.Backup, conf+".bak-") {
		t.Fatalf("backup = %q", res.Backup)
	}
	bak, err := e.svc.Read(ctx, e.id, res.Backup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bak.Data, orig.Data) || bak.Perm != orig.Perm || bak.ModTime != orig.ModTime {
		t.Errorf("backup differs from the previous file: perm %o, mtime %d", bak.Perm, bak.ModTime)
	}
	now, _ := e.svc.Read(ctx, e.id, conf)
	if !bytes.Equal(now.Data, edited) {
		t.Error("saved content differs")
	}

	// A second save in the same second keeps both backups.
	res2, err := e.svc.Save(ctx, e.id, conf, orig.Data, fs.SaveOptions{Backup: true})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Backup == res.Backup || res2.Backup == "" {
		t.Errorf("second backup = %q, first = %q", res2.Backup, res.Backup)
	}

	// Without the option, or for a new file, no backup is made.
	res3, err := e.svc.Save(ctx, e.id, e.path("new.conf"), []byte("x"), fs.SaveOptions{Backup: true})
	if err != nil || res3.Backup != "" {
		t.Errorf("new file: %+v, %v", res3, err)
	}
	res4, err := e.svc.Save(ctx, e.id, e.path("new.conf"), []byte("y"), fs.SaveOptions{})
	if err != nil || res4.Backup != "" {
		t.Errorf("no backup asked: %+v, %v", res4, err)
	}
}

func TestSaveRefusesWhenChangedOnServer(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	p := e.path("app.conf")
	if _, err := e.svc.Write(ctx, e.id, p, []byte("a=1\n")); err != nil {
		t.Fatal(err)
	}
	opened, err := e.svc.Read(ctx, e.id, p)
	if err != nil {
		t.Fatal(err)
	}
	expect := &fs.Expect{ModTime: opened.ModTime, Size: opened.Size}

	// Unchanged: saves, and the new stat is what the next save expects.
	res, err := e.svc.Save(ctx, e.id, p, []byte("a=2\n"), fs.SaveOptions{Expect: expect})
	if err != nil {
		t.Fatal(err)
	}

	// Someone else writes it; a save based on our copy is refused.
	e.run(t, "echo 'b=9' >> "+shellx.Quote(p))
	_, err = e.svc.Save(ctx, e.id, p, []byte("a=3\n"), fs.SaveOptions{Expect: &fs.Expect{ModTime: res.Entry.ModTime, Size: res.Entry.Size}})
	wantCode(t, err, apperr.FSChanged)
	if got := e.run(t, "cat "+shellx.Quote(p)); got != "a=2\nb=9" {
		t.Errorf("file after refused save: %q", got)
	}

	// Deleted meanwhile counts as changed too.
	e.run(t, "rm "+shellx.Quote(p))
	_, err = e.svc.Save(ctx, e.id, p, []byte("a=4\n"), fs.SaveOptions{Expect: expect})
	wantCode(t, err, apperr.FSChanged)

	// Without Expect (the user chose to overwrite), it writes.
	if _, err := e.svc.Save(ctx, e.id, p, []byte("a=5\n"), fs.SaveOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestWritable(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	mine := e.path("mine.conf")
	e.run(t, "touch "+shellx.Quote(mine))
	locked := e.path("locked")
	e.run(t, "mkdir "+shellx.Quote(locked)+" && touch "+shellx.Quote(locked+"/f")+" && chmod 555 "+shellx.Quote(locked))
	t.Cleanup(func() { e.run(t, "chmod 755 "+shellx.Quote(locked)) })

	tests := []struct {
		p    string
		want bool
	}{
		{mine, true},
		{"/etc/nginx/nginx.conf", true}, // owned by tester in the test image
		{"/etc/hostname", false},        // root's
		{"/etc/passwd", false},
		{locked + "/f", false}, // file writable, folder not: no room for the temporary file
		{e.path("missing"), false},
	}
	for _, tt := range tests {
		got, err := e.svc.Access(ctx, e.id, tt.p)
		if err != nil || got.Writable != tt.want {
			t.Errorf("Access(%s) = %+v, %v; want writable %v", tt.p, got, err, tt.want)
		}
	}
	a, _ := e.svc.Access(ctx, e.id, "/etc/my site.conf")
	if a.SudoEdit != "sudoedit -- '/etc/my site.conf'" {
		t.Errorf("SudoEdit = %q", a.SudoEdit)
	}
}
