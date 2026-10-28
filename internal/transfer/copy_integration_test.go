//go:build integration

package transfer_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"davish/internal/conn"
	"davish/internal/shellx"
	"davish/internal/testsshd"
	"davish/internal/transfer"
)

// pair is servers A and B with a scratch folder on each.
type pair struct {
	m          *conn.Manager
	a, b       string
	dirA, dirB string
	svc        *transfer.Service
	rec        *recorder
}

func setupPair(t *testing.T) pair {
	t.Helper()
	m, a, b := testsshd.ConnectBoth(t)
	rec := &recorder{done: map[string]transfer.Done{}}
	svc, err := transfer.New(m, rec, testsshd.Quiet(), transfer.Options{DownloadDir: t.TempDir(), Parallel: 1, ProgressEvery: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	p := pair{m: m, a: a, b: b, svc: svc, rec: rec}
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "copy-test-" + hex.EncodeToString(suffix)
	p.dirA = path.Join(testsshd.Run(t, m, a, "pwd"), name)
	p.dirB = path.Join(testsshd.Run(t, m, b, "pwd"), name)
	testsshd.Run(t, m, a, shellx.Cmd("mkdir", p.dirA).String())
	testsshd.Run(t, m, b, shellx.Cmd("mkdir", p.dirB).String())
	t.Cleanup(func() {
		testsshd.Run(t, m, a, shellx.Cmd("rm", "-rf", "--", p.dirA).String())
		testsshd.Run(t, m, b, shellx.Cmd("rm", "-rf", "--", p.dirB).String())
	})
	return p
}

func (p pair) onA(t *testing.T, cmd string) string { return testsshd.Run(t, p.m, p.a, cmd) }
func (p pair) onB(t *testing.T, cmd string) string { return testsshd.Run(t, p.m, p.b, cmd) }

func TestCopyFolderBetweenServers(t *testing.T) {
	p := setupPair(t)
	src := path.Join(p.dirA, "site data")
	p.onA(t, "cd "+shellx.Quote(p.dirA)+" && mkdir -p 'site data/img' 'site data/empty' && "+
		"echo hello > 'site data/index.html' && head -c 300000 /dev/urandom > 'site data/img/a.bin' && "+
		"chmod 640 'site data/index.html' && ln -s index.html 'site data/home'")
	// Taken on B: the copy must keep both.
	p.onB(t, "mkdir "+shellx.Quote(path.Join(p.dirB, "site data")))

	jobs, err := p.svc.Copy(context.Background(), p.a, []string{src}, p.b, p.dirB)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].Kind != transfer.KindCopy || jobs[0].Name != "site data (1)" ||
		jobs[0].FromServerID != p.a || jobs[0].ServerID != p.b {
		t.Fatalf("jobs = %+v", jobs)
	}
	ok(t, p.rec.wait(t, jobs[0].ID))

	dst := shellx.Quote(path.Join(p.dirB, "site data (1)"))
	sumA := p.onA(t, "cd "+shellx.Quote(src)+" && find . -type f | sort | xargs sha256sum")
	sumB := p.onB(t, "cd "+dst+" && find . -type f | sort | xargs sha256sum")
	if sumA != sumB {
		t.Errorf("content differs:\nA %s\nB %s", sumA, sumB)
	}
	if got := p.onB(t, "stat -c %a "+dst+"/index.html; readlink "+dst+"/home; test -d "+dst+"/empty && echo empty"); got != "640\nindex.html\nempty" {
		t.Errorf("modes/links = %q", got)
	}
	if got := p.onB(t, "ls -A "+shellx.Quote(p.dirB)); strings.Contains(got, "davish-part") {
		t.Errorf("leftovers: %q", got)
	}
}

func TestCopyFileAndRefusals(t *testing.T) {
	p := setupPair(t)
	f := path.Join(p.dirA, "notes.txt")
	p.onA(t, "printf 'abc' > "+shellx.Quote(f)+" && chmod 600 "+shellx.Quote(f))
	jobs, err := p.svc.Copy(context.Background(), p.a, []string{f}, p.b, p.dirB)
	if err != nil {
		t.Fatal(err)
	}
	ok(t, p.rec.wait(t, jobs[0].ID))
	if got := p.onB(t, "cat "+shellx.Quote(path.Join(p.dirB, "notes.txt"))+"; stat -c %a "+shellx.Quote(path.Join(p.dirB, "notes.txt"))); got != "abc600" {
		t.Errorf("copied = %q", got)
	}

	// Into itself on the same server, from /, or to a missing folder.
	for _, c := range []struct{ from, dst string }{
		{p.dirA, path.Join(p.dirA, "sub")},
		{"/", p.dirB},
	} {
		if _, err := p.svc.Copy(context.Background(), p.a, []string{c.from}, p.a, c.dst); err == nil {
			t.Errorf("copy %s into %s accepted", c.from, c.dst)
		}
	}
	if _, err := p.svc.Copy(context.Background(), p.a, []string{f}, p.b, "/no/such/dir"); err == nil {
		t.Error("copy into a missing folder accepted")
	}
}

func TestCopyCanceledLeavesNothing(t *testing.T) {
	p := setupPair(t)
	p.onA(t, "cd "+shellx.Quote(p.dirA)+" && mkdir big && for i in 1 2 3 4; do head -c 8000000 /dev/urandom > big/$i; done")
	var jobID string
	p.rec.onProgress = func(pr transfer.Progress) {
		if pr.JobID == jobID && pr.Done > 1<<20 {
			_ = p.svc.Cancel(jobID)
		}
	}
	jobs, err := p.svc.Copy(context.Background(), p.a, []string{path.Join(p.dirA, "big")}, p.b, p.dirB)
	if err != nil {
		t.Fatal(err)
	}
	p.rec.mu.Lock()
	jobID = jobs[0].ID
	p.rec.mu.Unlock()
	if d := p.rec.wait(t, jobs[0].ID); !d.Canceled {
		t.Fatalf("done = %+v", d)
	}
	if got := p.onB(t, "ls -A "+shellx.Quote(p.dirB)); got != "" {
		t.Errorf("left behind: %q", got)
	}
}

// The M9 acceptance: copy a 500MB folder from A to B. Slow, so it runs
// only with DAVISH_BIG=1.
func TestCopy500MB(t *testing.T) {
	if os.Getenv("DAVISH_BIG") == "" {
		t.Skip("set DAVISH_BIG=1 to copy 500MB between the test servers")
	}
	p := setupPair(t)
	p.onA(t, "cd "+shellx.Quote(p.dirA)+" && mkdir -p big/sub && for i in 1 2 3 4; do head -c 100000000 /dev/urandom > big/$i.bin; done && head -c 100000000 /dev/urandom > big/sub/5.bin")
	start := time.Now()
	jobs, err := p.svc.Copy(context.Background(), p.a, []string{path.Join(p.dirA, "big")}, p.b, p.dirB)
	if err != nil {
		t.Fatal(err)
	}
	ok(t, p.rec.waitFor(t, jobs[0].ID, 10*time.Minute))
	t.Logf("500MB in %s", time.Since(start).Round(time.Second))
	sumA := p.onA(t, "cd "+shellx.Quote(path.Join(p.dirA, "big"))+" && find . -type f | sort | xargs sha256sum")
	sumB := p.onB(t, "cd "+shellx.Quote(path.Join(p.dirB, "big"))+" && find . -type f | sort | xargs sha256sum")
	if sumA != sumB {
		t.Error("content differs")
	}
}
