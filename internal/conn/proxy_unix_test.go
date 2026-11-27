//go:build !windows

package conn

import (
	"strings"
	"testing"
	"time"

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
	pc, err := dialProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "cat"})
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
	pc, err := dialProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "sleep 30"})
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
	pc, err := dialProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: `echo sandbox is down >&2; exit 3`})
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
	pc, err := dialProxy(store.Server{Host: "h", User: "u", Port: 22, ProxyCommand: "no-such-relay-9917 --tunnel"})
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck // test cleanup
	waitFor(t, func() bool {
		return strings.Contains(pc.explain(), "no-such-relay-9917")
	})
}
