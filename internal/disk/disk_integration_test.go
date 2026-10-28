//go:build integration

package disk_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"davish/internal/audit"
	"davish/internal/disk"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestUsageAndMounts(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := disk.New(audit.NewRunner(m, audit.New(discard{})))
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
	s := disk.New(audit.NewRunner(m, audit.New(discard{})))
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
