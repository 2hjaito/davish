//go:build !windows

package conn

import (
	"errors"
	"io"
	"os"
	"os/exec"
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

// A proxy that dies must end the connection: the reader gets EOF at once,
// not after the handshake timeout. That only happens when Davish itself does
// not hold the write end of the proxy's stdout.
func TestProxyConnReadEndsWhenTheProxyExits(t *testing.T) {
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "echo bye; exit 3"}))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck,gosec // always nil
	done := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(pc)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("read after the proxy exited: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Read did not end after the proxy exited: Davish holds the write end of its stdout")
	}
}

// openFDs counts this process's open file descriptors.
func openFDs(t *testing.T) int {
	t.Helper()
	for _, dir := range []string{"/proc/self/fd", "/dev/fd"} {
		if ents, err := os.ReadDir(dir); err == nil {
			return len(ents)
		}
	}
	t.Skip("no way to count file descriptors here")
	return 0
}

// Every connect and reconnect starts a proxy; none may leave descriptors
// behind, or a long session ends with "too many open files".
func TestProxyConnLeavesNoDescriptorsBehind(t *testing.T) {
	cycle := func() {
		pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "cat"}))
		if err != nil {
			t.Fatal(err)
		}
		pc.Close() //nolint:errcheck,gosec // always nil
		waitFor(t, func() bool { return pc.explain() != "" })
	}
	cycle() // warm up anything lazily opened
	before := openFDs(t)
	for range 20 {
		cycle()
	}
	time.Sleep(200 * time.Millisecond)
	if after := openFDs(t); after > before+2 {
		t.Fatalf("descriptors grew from %d to %d over 20 proxies", before, after)
	}
}

// Once nothing is left in the command's group its number is free for the
// system to give to another process, so Close must not signal it: that could
// kill an unrelated process of the user's.
func TestCloseDoesNotSignalAnEmptyGroup(t *testing.T) {
	var calls int
	old := killProxyGroup
	killProxyGroup = func(*exec.Cmd) { calls++ }
	t.Cleanup(func() { killProxyGroup = old })

	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "exit 0"}))
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return pc.explain() != "" })
	pc.mu.Lock()
	gone := pc.groupGone
	pc.mu.Unlock()
	if !gone {
		t.Fatal("a proxy that exited with nothing left behind did not mark its group gone")
	}
	pc.Close() //nolint:errcheck,gosec // always nil
	if calls != 0 {
		t.Fatalf("Close signalled a group with no process in it (%d times)", calls)
	}

	// A proxy that is still running does get the group signal.
	pc2, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "sleep 30"}))
	if err != nil {
		t.Fatal(err)
	}
	pc2.Close() //nolint:errcheck,gosec // always nil
	if calls != 1 {
		t.Fatalf("a running proxy: the group was signalled %d times; want 1", calls)
	}
	waitFor(t, func() bool { return pc2.explain() != "" })
}

// Something the shell left running is still in the group after the shell is
// gone: nothing kills it before Close (ssh would not either), and Close ends it.
func TestCloseEndsWhatOutlivedTheShell(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	pc, err := dialProxy(typedProxy(store.Server{
		Host: "h", User: "u", Port: 22,
		ProxyCommand: "sleep 30 & echo $! > " + pidFile,
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
	// The shell exits at once; Wait gives up on the orphan's pipes after WaitDelay.
	waitFor(t, func() bool { return pc.explain() != "" })
	pc.mu.Lock()
	gone := pc.groupGone
	pc.mu.Unlock()
	if gone {
		t.Fatal("the group was marked gone while the child still ran")
	}
	if err := syscall.Kill(child, 0); err != nil {
		t.Fatalf("the child was ended before Close: %v", err)
	}
	pc.Close() //nolint:errcheck,gosec // always nil
	waitFor(t, func() bool { return syscall.Kill(child, 0) == syscall.ESRCH })
}

// The proxy is a session leader with no controlling terminal: a proxy that
// reads /dev/tty fails at once instead of being stopped by SIGTTIN (which
// would hang the connection until the handshake timeout).
func TestProxyRunsInASessionOfItsOwn(t *testing.T) {
	pc, err := dialProxy(typedProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "sleep 30"}))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck,gosec // always nil
	pid := pc.cmd.Process.Pid
	sid, err := getsid(pid)
	if err != nil {
		t.Fatal(err)
	}
	ours, _ := getsid(os.Getpid())
	if sid != pid || sid == ours {
		t.Fatalf("session of the proxy = %d (pid %d), ours = %d; want its own", sid, pid, ours)
	}
}
