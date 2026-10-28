//go:build integration

package ports_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"davish/internal/audit"
	"davish/internal/ports"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestListSeesSSHD(t *testing.T) {
	m, id := testsshd.Connect(t)
	l := ports.NewLister(audit.NewRunner(m, audit.New(discard{})))
	list, err := l.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range list.Ports {
		found = found || p.Port == 22
	}
	if !found || list.Root {
		t.Fatalf("list = %+v", list)
	}
}

func TestTunnelReachesServerPort(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := ports.NewTunnels(m, discard{}, testsshd.Quiet())
	t.Cleanup(s.Close)

	// The server's own sshd answers with its banner through the tunnel.
	tun, err := s.Start(id, "0.0.0.0", 22, "")
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(tun.LocalPort), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "SSH-2.0-") {
		t.Fatalf("banner = %q, %v", line, err)
	}
	c.Close()

	s.Stop(tun.ID)
	if len(s.List()) != 0 {
		t.Fatal("tunnel still listed")
	}
	if _, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(tun.LocalPort), time.Second); err == nil {
		t.Fatal("local port still open after stop")
	}
}

func TestTunnelRewritesHost(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := ports.NewTunnels(m, discard{}, testsshd.Quiet())
	t.Cleanup(s.Close)
	// nginx on the test box serves davish.test only to that name.
	tun, err := s.Start(id, "0.0.0.0", 80, "davish.test")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(tun.LocalPort) + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "davish site") {
		t.Fatalf("body = %q", body)
	}
}

func TestBrowserProxy(t *testing.T) {
	m, id := testsshd.Connect(t)
	b := ports.NewBrowser(m, testsshd.Quiet())
	t.Cleanup(b.Close)
	site, err := b.Open(id, 80)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://127.0.0.1:" + strconv.Itoa(site.LocalPort)
	get := func(path string, hdr map[string]string) (int, string) {
		req, _ := http.NewRequest("GET", base+path, nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}

	// The first load carries the token; nginx's page comes with nav.js.
	code, body := get("/?__davish="+site.Token, nil)
	if code != 200 || !strings.Contains(body, "/__davish/nav.js") || !strings.Contains(strings.ToLower(body), "nginx") {
		t.Fatalf("token load: %d %q", code, body)
	}
	// Anything else from the page has the page as Referer.
	if code, _ := get("/__davish/nav.js", map[string]string{"Referer": base + "/"}); code != 200 {
		t.Errorf("nav.js with referer: %d", code)
	}
	// No token, no same-origin Referer: refused.
	if code, _ := get("/", nil); code != 403 {
		t.Errorf("no token: %d", code)
	}
	if code, _ := get("/", map[string]string{"Referer": "https://evil.example/"}); code != 403 {
		t.Errorf("foreign referer: %d", code)
	}
	if len(b.Sites()) != 1 {
		t.Errorf("sites = %+v", b.Sites())
	}
}
