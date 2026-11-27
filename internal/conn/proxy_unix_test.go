//go:build !windows

package conn

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/store"
)

// waitFor polls until cond passes, to give the proxy process time to start,
// say something and exit.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the proxy process")
}

func TestProxyConnCarriesBothWays(t *testing.T) {
	// cat echoes what ssh writes into the tunnel, the shape a relay where
	// the server answers first exercises.
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "cat"}))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck // test cleanup
	if _, err := pc.Write([]byte("hello sandbox")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 13)
	if _, err := readFull(pc, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello sandbox" {
		t.Fatalf("read %q", buf)
	}
}

func readFull(pc *proxyConn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := pc.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestProxyConnClosesTheProcess(t *testing.T) {
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "sleep 30"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.Close(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return pc.explain() != "" })
	if msg := pc.explain(); !strings.Contains(msg, "killed") && !strings.Contains(msg, "exited") {
		t.Fatalf("explain = %q", msg)
	}
}

func TestProxyConnExplainsItsDeath(t *testing.T) {
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: `echo sandbox is down >&2; exit 3`}))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck // test cleanup
	waitFor(t, func() bool {
		return strings.Contains(pc.explain(), "sandbox is down")
	})
	msg := pc.explain()
	if !strings.Contains(msg, "exit status 3") && !strings.Contains(msg, "stderr") {
		t.Errorf("explain = %q; want the exit status or the stderr in it", msg)
	}
}

func TestProxyConnSaysWhenTheProgramIsMissing(t *testing.T) {
	// sh starts fine and then fails, exactly like with ssh itself, so the
	// answer comes through stderr once the process has been reaped.
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "no-such-relay-9917 --tunnel"}))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck // test cleanup
	waitFor(t, func() bool {
		return strings.Contains(pc.explain(), "no-such-relay-9917")
	})
}

// A command nobody typed or confirmed is not started at all: the marker file
// its first word would make is the proof.
func TestDialProxyDoesNotRunAnUnconfirmedCommand(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	srv := store.Server{Host: "sandbox.internal", Port: 22, User: "admin", ProxyCommand: "touch " + marker}
	if pc, err := dialProxy(srv); err == nil || proxyCode(err) != apperr.ConnProxyUnconfirmed {
		if pc != nil {
			pc.Close() //nolint:errcheck,gosec // test cleanup
		}
		t.Fatalf("err = %v; want CONN_PROXY_UNCONFIRMED", err)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the unconfirmed command ran")
	}

	// Confirmed, it runs.
	srv.ProxyTrust = store.ProxyHash(srv.ProxyCommand)
	pc, err := dialProxy(srv)
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck,gosec // test cleanup
	waitFor(t, func() bool { _, err := os.Stat(marker); return err == nil })
}

func proxyCode(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

// What the shell started must end with it. A proxy command with two commands
// keeps the shell as their parent on every system (dash does not exec even a
// single one), so killing only the shell left the tunnel client running.
func TestProxyConnCloseEndsWhatTheShellStarted(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	pc, err := dialProxy(typedProxy(store.Server{
		Host: "h", User: "u", Port: 22,
		ProxyCommand: "sleep 30 & echo $! > " + pidFile + "; wait",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var child int
	waitFor(t, func() bool {
		b, err := os.ReadFile(pidFile) //nolint:gosec // a file in the test's temp dir
		if err != nil {
			return false
		}
		child, err = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil && child > 0
	})
	if err := syscall.Kill(child, 0); err != nil {
		t.Fatalf("the child is not running before Close: %v", err)
	}

	pc.Close() //nolint:errcheck,gosec // always nil
	waitFor(t, func() bool { return syscall.Kill(child, 0) == syscall.ESRCH })
	// And Wait returns, so the process is collected and explain has an answer.
	waitFor(t, func() bool { return pc.explain() != "" })
}
