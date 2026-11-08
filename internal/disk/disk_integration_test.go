//go:build integration

package disk_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/disk"
	"davish/internal/fs"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func newDisk(m *conn.Manager) *disk.Service {
	run := audit.NewRunner(m, audit.New(discard{}))
	return disk.New(run, fs.New(m, run, testsshd.Quiet()))
}

func TestUsageAndMounts(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := newDisk(m)
	testsshd.Run(t, m, id, `rm -rf ~/du && mkdir -p ~/du/big ~/du/small && head -c 2000000 /dev/zero > ~/du/big/f && head -c 1000 /dev/zero > ~/du/small/f`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/du`) })

	u, err := s.Usage(context.Background(), id, "/home/tester/du", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(u.Entries) != 2 || u.Entries[0].Path != "/home/tester/du/big" || u.Entries[0].Size < 1_900_000 {
		t.Fatalf("usage = %+v", u)
	}
	ms, err := s.Mounts(context.Background(), id)
	if err != nil || len(ms) == 0 {
		t.Fatalf("mounts = %+v, %v", ms, err)
	}
	if _, err := s.Cleanups(context.Background(), id); err != nil {
		t.Fatal(err)
	}
}

func TestCancel(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := newDisk(m)
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.Cancel("scan-1")
	}()
	// / of the whole box takes longer than 200ms.
	_, err := s.Usage(context.Background(), id, "/usr", "scan-1")
	if !errors.Is(err, context.Canceled) {
		t.Skipf("du finished before the cancel (err = %v)", err)
	}
}

func TestOverviewFindsWhatFillsTheDisk(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := newDisk(m)
	// A big file, a deleted file a process still holds, and a folder only
	// root may read.
	testsshd.Run(t, m, id, `rm -rf ~/ov && mkdir ~/ov && head -c 100000000 /dev/zero > ~/ov/big.bin
setsid sh -c 'exec 3>"$HOME/ov/held.log"; head -c 60000000 /dev/zero >&3; rm "$HOME/ov/held.log"; exec sleep 120' </dev/null >/dev/null 2>&1 &
sleep 1`)
	testsshd.Exec(t, "sh", "-c", "mkdir -p /var/lib/davish-secret && head -c 20000000 /dev/zero > /var/lib/davish-secret/f && chmod 700 /var/lib/davish-secret")
	t.Cleanup(func() {
		testsshd.Run(t, m, id, `pkill -u tester -x sleep; rm -rf ~/ov; true`)
		testsshd.Exec(t, "rm", "-rf", "/var/lib/davish-secret")
	})

	o, err := s.Overview(context.Background(), id, "/", "")
	if err != nil {
		t.Fatal(err)
	}
	if o.Root || o.Measured == 0 || o.Mount.Used == 0 {
		t.Fatalf("overview = root %v, measured %d, mount %+v", o.Root, o.Measured, o.Mount)
	}
	found := false
	for _, f := range o.Files {
		found = found || (f.Path == "/home/tester/ov/big.bin" && f.Size == 100000000)
	}
	if !found {
		t.Errorf("big.bin not in %+v", o.Files)
	}
	held := false
	for _, d := range o.Deleted {
		held = held || (d.Path == "/home/tester/ov/held.log" && d.Size == 60000000 && d.PID > 0)
	}
	if !held {
		t.Errorf("held.log not in %+v", o.Deleted)
	}
	if o.SudoMeasure != "sudo du -xh -d 2 -- / 2>/dev/null | sort -h | tail -n 25" {
		t.Errorf("sudo measure = %q", o.SudoMeasure)
	}
	if !slices.Contains(o.Unreadable, "/var/lib/davish-secret") {
		t.Errorf("unreadable = %q", o.Unreadable)
	}
	if len(o.Dirs) == 0 || o.Dirs[0].Size < o.Dirs[len(o.Dirs)-1].Size {
		t.Errorf("dirs not largest first: %d", len(o.Dirs))
	}
}

func TestCleanupsSeeCachesAndOldTmp(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := newDisk(m)
	testsshd.Run(t, m, id, `mkdir -p ~/.cache/davish-test && head -c 3000000 /dev/zero > ~/.cache/davish-test/blob
head -c 2000000 /dev/zero > /tmp/davish-old && touch -a -d '10 days ago' /tmp/davish-old`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/.cache/davish-test /tmp/davish-old`) })

	cs, err := s.Cleanups(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]disk.Cleanup{}
	for _, c := range cs {
		got[c.ID] = c
	}
	if c := got["home-cache"]; c.Bytes < 3000000 || c.Root {
		t.Errorf("home cache = %+v", c)
	}
	if c := got["tmp"]; c.Bytes < 2000000 || !c.Root {
		t.Errorf("tmp = %+v", c)
	}
	// Davish runs a cleanup the user may run, and only the one confirmed.
	if err := s.Clean(context.Background(), id, "home-cache", got["home-cache"].Command); err != nil {
		t.Fatal(err)
	}
	if out := testsshd.Run(t, m, id, `ls -A ~/.cache | wc -l`); strings.TrimSpace(out) != "0" {
		t.Errorf("~/.cache still holds %s entries", out)
	}
}

func TestRemoveAndTruncate(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := newDisk(m)
	ctx := context.Background()
	testsshd.Run(t, m, id, `rm -rf ~/rm && mkdir -p ~/rm/dir && head -c 1000 /dev/zero > ~/rm/dir/f && head -c 5000 /dev/zero > ~/rm/app.log`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/rm`) })

	plan, err := s.PlanRemove(ctx, id, "/home/tester/rm/dir")
	if err != nil || !plan.NonEmpty || plan.Command != "rm -r -- /home/tester/rm/dir" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if err := s.Remove(ctx, id, "/home/tester/rm/dir", plan.Command); err != nil {
		t.Fatal(err)
	}
	tp, err := disk.PlanTruncate("/home/tester/rm/app.log")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Truncate(ctx, id, "/home/tester/rm/app.log", tp.Command); err != nil {
		t.Fatal(err)
	}
	if out := testsshd.Run(t, m, id, `ls ~/rm; stat -c %s ~/rm/app.log`); out != "app.log\n0" {
		t.Errorf("after remove and truncate: %q", out)
	}
	// Root's files are left for sudo in a terminal.
	tp, _ = disk.PlanTruncate("/var/log/dpkg.log")
	var ae *apperr.Error
	if err := s.Truncate(ctx, id, "/var/log/dpkg.log", tp.Command); !errors.As(err, &ae) || ae.Code != apperr.FSPermissionDenied {
		t.Errorf("root's file: %v", err)
	}
	if _, err := s.PlanRemove(ctx, id, "/home/tester"); err == nil {
		t.Error("planned removing the home folder")
	}
}
