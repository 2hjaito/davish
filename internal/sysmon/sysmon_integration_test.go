//go:build integration

package sysmon_test

import (
	"context"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/conn"
	"davish/internal/store"
	"davish/internal/sysmon"
	"davish/internal/testsshd"
)

// ticks records sysmon:tick events.
type ticks struct {
	mu  sync.Mutex
	all []sysmon.Tick
}

func (r *ticks) Emit(_ string, payload any) {
	if t, ok := payload.(sysmon.Tick); ok {
		r.mu.Lock()
		r.all = append(r.all, t)
		r.mu.Unlock()
	}
}

func (r *ticks) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.all)
}

func (r *ticks) last() sysmon.Tick {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.all[len(r.all)-1]
}

// waitCount waits until more than n ticks arrived.
func (r *ticks) waitCount(t *testing.T, n int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if r.count() > n {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no tick after %d", n)
}

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

func connect(t *testing.T) (*conn.Manager, string) {
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
	return m, srv.ID
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func run(t *testing.T, m *conn.Manager, id, cmd string) string {
	t.Helper()
	client, err := m.Client(id)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()
	out, err := sess.Output(cmd)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestTicksMatchFreeAndDf(t *testing.T) {
	m, id := connect(t)
	rec := &ticks{}
	svc := sysmon.New(m, rec, quiet(), sysmon.Options{})
	t.Cleanup(svc.Close)

	svc.Watch(id)
	rec.waitCount(t, 0)
	tick := rec.last()

	// free -b: "Mem: total used ..."
	mem := strings.Fields(strings.Split(run(t, m, id, "free -b"), "\n")[1])
	total, _ := strconv.ParseUint(mem[1], 10, 64)
	if tick.MemTotal != total {
		t.Errorf("memTotal = %d; free says %d", tick.MemTotal, total)
	}
	if tick.MemUsed == 0 || tick.MemUsed >= tick.MemTotal {
		t.Errorf("memUsed = %d of %d", tick.MemUsed, tick.MemTotal)
	}
	// df rounds Capacity up the same way; allow a point for files written in between.
	capacity := strings.Fields(strings.Split(run(t, m, id, "df -kP /"), "\n")[1])[4]
	pct, _ := strconv.ParseFloat(strings.TrimSuffix(capacity, "%"), 64)
	if math.Abs(tick.DiskPct-pct) > 1 {
		t.Errorf("diskPct = %v; df says %v", tick.DiskPct, pct)
	}
	if tick.CPU < 0 || tick.CPU > 100 || tick.Uptime <= 0 || tick.ServerID != id {
		t.Errorf("tick = %+v", tick)
	}
	if n, _ := strconv.Atoi(run(t, m, id, "nproc --all")); len(tick.Cores) != n {
		t.Errorf("%d cores; nproc says %d", len(tick.Cores), n)
	}
	// free -b: "Swap: total used free"
	swap := strings.Fields(strings.Split(run(t, m, id, "free -b"), "\n")[2])
	if swapTotal, _ := strconv.ParseUint(swap[1], 10, 64); tick.SwapTotal != swapTotal {
		t.Errorf("swapTotal = %d; free says %d", tick.SwapTotal, swapTotal)
	}
	if len(tick.Mounts) == 0 || tick.Mounts[0].Path != "/" || tick.GPUs == nil || tick.Ifaces == nil {
		t.Errorf("mounts %+v, gpus %+v, ifaces %+v", tick.Mounts, tick.GPUs, tick.Ifaces)
	}
}

func TestStopsWithTheLastUnwatch(t *testing.T) {
	m, id := connect(t)
	rec := &ticks{}
	svc := sysmon.New(m, rec, quiet(), sysmon.Options{})
	t.Cleanup(svc.Close)

	svc.Watch(id) // menu bar
	svc.Watch(id) // a Monitor window
	rec.waitCount(t, 0)

	svc.Unwatch(id)
	n := rec.count()
	rec.waitCount(t, n) // the menu bar still watches

	svc.Unwatch(id)
	time.Sleep(500 * time.Millisecond) // a tick may already be in flight
	n = rec.count()
	time.Sleep(5 * time.Second)
	if got := rec.count(); got != n {
		t.Errorf("%d ticks after the last Unwatch", got-n)
	}
	// Match whole command lines: the shell running this check mentions "sh -s" too.
	if out := run(t, m, id, "ps -eo args= | grep -cx 'sh -s' || true"); out != "0" {
		t.Errorf("collector still running on the server: %s processes", out)
	}
}

func TestResumesAfterReconnect(t *testing.T) {
	m, id := connect(t)
	rec := &ticks{}
	svc := sysmon.New(m, rec, quiet(), sysmon.Options{RetryEvery: 200 * time.Millisecond})
	t.Cleanup(svc.Close)

	svc.Watch(id)
	rec.waitCount(t, 0)

	m.Disconnect(id)
	if err := m.Connect(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	n := rec.count()
	rec.waitCount(t, n)
}
