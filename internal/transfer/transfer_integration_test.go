//go:build integration

package transfer_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/conn"
	"davish/internal/shellx"
	"davish/internal/testsshd"
	"davish/internal/transfer"
)

// recorder collects transfer events. onProgress, when set, runs inside
// Emit, so it can cancel a job at a precise point.
type recorder struct {
	mu         sync.Mutex
	done       map[string]transfer.Done
	progress   []transfer.Progress
	onProgress func(transfer.Progress)
}

func (r *recorder) Emit(_ string, payload any) {
	switch p := payload.(type) {
	case transfer.Progress:
		r.mu.Lock()
		r.progress = append(r.progress, p)
		hook := r.onProgress
		r.mu.Unlock()
		if hook != nil {
			hook(p)
		}
	case transfer.Done:
		r.mu.Lock()
		r.done[p.JobID] = p
		r.mu.Unlock()
	}
}

func (r *recorder) wait(t *testing.T, id string) transfer.Done {
	t.Helper()
	return r.waitFor(t, id, 30*time.Second)
}

func (r *recorder) waitFor(t *testing.T, id string, limit time.Duration) transfer.Done {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		d, ok := r.done[id]
		r.mu.Unlock()
		if ok {
			return d
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish", id)
	return transfer.Done{}
}

type env struct {
	m      *conn.Manager
	id     string
	svc    *transfer.Service
	rec    *recorder
	remote string // scratch folder on the server
	local  string // scratch folder here
	dl     string // download folder
}

func setup(t *testing.T) env {
	t.Helper()
	m, id := testsshd.Connect(t)
	rec := &recorder{done: map[string]transfer.Done{}}
	e := env{m: m, id: id, rec: rec, local: t.TempDir(), dl: t.TempDir()}
	svc, err := transfer.New(m, rec, testsshd.Quiet(), transfer.Options{DownloadDir: e.dl, Parallel: 1, ProgressEvery: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	e.svc = svc

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	home := testsshd.Run(t, m, id, "pwd")
	e.remote = path.Join(home, "transfer-test-"+hex.EncodeToString(b))
	e.run(t, shellx.Cmd("mkdir", e.remote).String())
	t.Cleanup(func() {
		// Uploaded read-only folders must be made writable to be removed.
		e.run(t, shellx.Cmd("chmod", "-R", "u+w", "--", e.remote).String()+" && "+shellx.Cmd("rm", "-rf", "--", e.remote).String())
	})
	return e
}

func (e env) run(t *testing.T, cmd string) string { return testsshd.Run(t, e.m, e.id, cmd) }

func (e env) ls(t *testing.T) string {
	return e.run(t, "ls -A "+shellx.Quote(e.remote))
}

func writeLocal(t *testing.T, p string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, mode); err != nil {
		t.Fatal(err)
	}
}

func ok(t *testing.T, d transfer.Done) {
	t.Helper()
	if d.Error != nil || d.Canceled {
		t.Fatalf("job ended with %+v", d)
	}
}

func TestUploadFileKeepsBothOrReplaces(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	src := filepath.Join(e.local, "run.sh")
	writeLocal(t, src, []byte("echo v1\n"), 0o755)

	jobs, err := e.svc.Upload(ctx, e.id, e.remote, []transfer.UploadRequest{{Local: src}})
	if err != nil {
		t.Fatal(err)
	}
	ok(t, e.rec.wait(t, jobs[0].ID))
	if got := e.run(t, "stat -c %a "+shellx.Quote(path.Join(e.remote, "run.sh"))); got != "755" {
		t.Errorf("mode = %s; want the local 755", got)
	}

	plan, err := e.svc.PlanUpload(ctx, e.id, e.remote, []string{src})
	if err != nil {
		t.Fatal(err)
	}
	dst := path.Join(e.remote, "run.sh")
	if !plan[0].Exists || plan[0].ReplaceCommand != "put "+src+" "+dst {
		t.Fatalf("plan = %+v", plan[0])
	}

	// Keep both.
	jobs, err = e.svc.Upload(ctx, e.id, e.remote, []transfer.UploadRequest{{Local: src}})
	if err != nil {
		t.Fatal(err)
	}
	ok(t, e.rec.wait(t, jobs[0].ID))
	if jobs[0].Name != "run (1).sh" {
		t.Errorf("kept-both name = %q", jobs[0].Name)
	}

	// Replace keeps the remote file's mode.
	e.run(t, "chmod 640 "+shellx.Quote(dst))
	writeLocal(t, src, []byte("echo v2\n"), 0o755)
	jobs, err = e.svc.Upload(ctx, e.id, e.remote, []transfer.UploadRequest{{Local: src, Replace: true}})
	if err != nil {
		t.Fatal(err)
	}
	ok(t, e.rec.wait(t, jobs[0].ID))
	if got := e.run(t, "stat -c %a "+shellx.Quote(dst)+" && cat "+shellx.Quote(dst)); got != "640\necho v2" {
		t.Errorf("after replace: %q", got)
	}
	if got := e.ls(t); got != "run (1).sh\nrun.sh" {
		t.Errorf("remote folder: %q", got)
	}
}

func TestUploadFolder(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	src := filepath.Join(e.local, "site")
	writeLocal(t, filepath.Join(src, "index.html"), []byte("<h1>hi</h1>"), 0o644)
	writeLocal(t, filepath.Join(src, "assets", "app.js"), []byte("x()"), 0o644)
	if err := os.Symlink("assets/app.js", filepath.Join(src, "app.js")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(src, "assets"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(src, "assets"), 0o755) })

	jobs, err := e.svc.Upload(ctx, e.id, e.remote, []transfer.UploadRequest{{Local: src}})
	if err != nil {
		t.Fatal(err)
	}
	ok(t, e.rec.wait(t, jobs[0].ID))
	dst := shellx.Quote(path.Join(e.remote, "site"))
	got := e.run(t, "cd "+dst+" && cat index.html assets/app.js && readlink app.js && stat -c %a assets")
	if got != "<h1>hi</h1>x()assets/app.js\n555" {
		t.Errorf("uploaded folder: %q", got)
	}
	if got := e.ls(t); got != "site" {
		t.Errorf("remote folder: %q", got)
	}
}

func TestUploadCanceledLeavesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	src := filepath.Join(e.local, "big")
	writeLocal(t, src, bytes.Repeat([]byte("x"), 8<<20), 0o644)

	var once sync.Once
	e.rec.onProgress = func(p transfer.Progress) {
		if p.Done > 0 {
			once.Do(func() { _ = e.svc.Cancel(p.JobID) })
		}
	}
	jobs, err := e.svc.Upload(ctx, e.id, e.remote, []transfer.UploadRequest{{Local: src}})
	if err != nil {
		t.Fatal(err)
	}
	if d := e.rec.wait(t, jobs[0].ID); !d.Canceled {
		t.Fatalf("done = %+v; want canceled", d)
	}
	if got := e.ls(t); got != "" {
		t.Errorf("left on the server: %q", got)
	}
}

func TestDownloadFileAndFolder(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	q := shellx.Quote(e.remote)
	e.run(t, "cd "+q+" && printf data > f.txt && mkdir -p d/sub && printf deep > d/sub/x && ln -s sub/x d/link && chmod 700 d/sub")

	jobs, err := e.svc.Download(ctx, e.id, []string{path.Join(e.remote, "f.txt"), path.Join(e.remote, "d")})
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range jobs {
		ok(t, e.rec.wait(t, j.ID))
	}
	if data, err := os.ReadFile(filepath.Join(e.dl, "f.txt")); err != nil || string(data) != "data" {
		t.Errorf("f.txt = %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(e.dl, "d", "sub", "x")); err != nil || string(data) != "deep" {
		t.Errorf("d/sub/x = %q, %v", data, err)
	}
	if target, err := os.Readlink(filepath.Join(e.dl, "d", "link")); err != nil || target != "sub/x" {
		t.Errorf("link = %q, %v", target, err)
	}
	if fi, err := os.Stat(filepath.Join(e.dl, "d", "sub")); err != nil || fi.Mode().Perm() != 0o700 {
		t.Errorf("sub mode = %v, %v", fi.Mode(), err)
	}

	// A second download keeps both.
	jobs, err = e.svc.Download(ctx, e.id, []string{path.Join(e.remote, "f.txt")})
	if err != nil {
		t.Fatal(err)
	}
	ok(t, e.rec.wait(t, jobs[0].ID))
	if names := localNames(t, e.dl); names != "d,f (1).txt,f.txt" {
		t.Errorf("download folder: %s", names)
	}
}

func TestDownloadCanceledLeavesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.run(t, "head -c 8000000 /dev/zero > "+shellx.Quote(path.Join(e.remote, "big")))

	var once sync.Once
	e.rec.onProgress = func(p transfer.Progress) {
		if p.Done > 0 {
			once.Do(func() { _ = e.svc.Cancel(p.JobID) })
		}
	}
	jobs, err := e.svc.Download(ctx, e.id, []string{path.Join(e.remote, "big")})
	if err != nil {
		t.Fatal(err)
	}
	if d := e.rec.wait(t, jobs[0].ID); !d.Canceled {
		t.Fatalf("done = %+v; want canceled", d)
	}
	if names := localNames(t, e.dl); names != "" {
		t.Errorf("left locally: %s", names)
	}
}

func TestDownloadRefusesUnsafeNames(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.run(t, "mkdir "+shellx.Quote(path.Join(e.remote, "d"))+` && touch `+shellx.Quote(path.Join(e.remote, "d", `..\evil`)))

	jobs, err := e.svc.Download(ctx, e.id, []string{path.Join(e.remote, "d")})
	if err != nil {
		t.Fatal(err)
	}
	d := e.rec.wait(t, jobs[0].ID)
	var want *apperr.Error
	if d.Error == nil || !errors.As(error(d.Error), &want) || want.Code != apperr.InvalidInput {
		t.Fatalf("done = %+v; want INVALID_INPUT", d)
	}
	if names := localNames(t, e.dl); names != "" {
		t.Errorf("left locally: %s", names)
	}
}

func TestCancelUnknownJob(t *testing.T) {
	e := setup(t)
	var ae *apperr.Error
	if err := e.svc.Cancel("nope"); !errors.As(err, &ae) || ae.Code != apperr.NotFound {
		t.Errorf("Cancel = %v", err)
	}
}

func localNames(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
