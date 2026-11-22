package ports

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAllowed(t *testing.T) {
	s := &site{Site: Site{LocalPort: 4567, Token: "abc"}}
	cases := []struct {
		host, target, origin, referer string
		want                          bool
	}{
		{"127.0.0.1:4567", "/?__davish=abc", "", "", true},
		{"127.0.0.1:4567", "/?__davish=wrong", "", "", false},
		{"127.0.0.1:4567", "/app.js", "", "http://127.0.0.1:4567/", true},
		{"127.0.0.1:4567", "/ws", "http://127.0.0.1:4567", "", true},
		{"127.0.0.1:4567", "/", "", "", false},
		{"127.0.0.1:4567", "/", "https://evil.example", "", false},
		{"127.0.0.1:4567", "/", "", "http://127.0.0.1:45678/", false},
		{"evil.example:4567", "/?__davish=abc", "", "", false}, // DNS rebinding
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", c.target, nil)
		r.Host = c.host
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.referer != "" {
			r.Header.Set("Referer", c.referer)
		}
		if got := s.allowed(r); got != c.want {
			t.Errorf("%+v: allowed = %v", c, got)
		}
	}
}

func TestInjectNav(t *testing.T) {
	got := string(injectNav([]byte(`<html><HEAD lang="x"><title>a</title></head></html>`)))
	if !strings.Contains(got, `<HEAD lang="x"><script src="/__davish/nav.js"></script><title>`) {
		t.Errorf("got %s", got)
	}
	if !strings.HasPrefix(string(injectNav([]byte("<p>x"))), "<script") {
		t.Error("no head: script not first")
	}
}

func TestMapLocationSamePort(t *testing.T) {
	b := NewBrowser(nil, nil)
	s := &site{Site: Site{ServerID: "s", RemotePort: 3000, LocalPort: 5000, Token: "tok"}}
	for in, want := range map[string]string{
		// The server's own address goes to the listener, with its token.
		"http://localhost:3000/login?x=1": "http://127.0.0.1:5000/login?x=1&__davish=tok",
		"http://127.0.0.1:3000/":          "http://127.0.0.1:5000/?__davish=tok",
		// A relative redirect (fastify's "/" to "/ui/") stays here, with the token.
		"/ui/":     "/ui/?__davish=tok",
		"/a/b?c=d": "/a/b?c=d&__davish=tok",
		"login":    "login?__davish=tok",
		"/ui/#top": "/ui/?__davish=tok#top",
		// Other sites are left alone and never get the token.
		"https://github.com/login": "https://github.com/login",
		"http://example.com/x":     "http://example.com/x",
		"//evil.example/x":         "//evil.example/x",
	} {
		if got := b.mapLocation(s, in); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}

// The request after a redirect has a Referer from the frame's parent, not the
// proxy, so the token in Location is what lets it through.
func TestRedirectKeepsRequestAllowed(t *testing.T) {
	b := NewBrowser(nil, nil)
	s := &site{Site: Site{ServerID: "s", RemotePort: 12103, LocalPort: 5000, Token: "tok"}}
	next := b.mapLocation(s, "/ui/")
	r := httptest.NewRequest("GET", next, nil)
	r.Host = "127.0.0.1:5000"
	r.Header.Set("Referer", "http://wails.localhost/")
	if !s.allowed(r) {
		t.Errorf("redirect target %s refused", next)
	}
	other := &site{Site: Site{LocalPort: 5000, Token: "other"}}
	if other.allowed(r) {
		t.Error("a token from another listener was accepted")
	}
}

func TestFrameAncestorsRemoved(t *testing.T) {
	got := frameAncestors.ReplaceAllString("default-src 'self'; frame-ancestors 'none'; img-src *", "")
	if strings.Contains(got, "frame-ancestors") || !strings.Contains(got, "img-src") {
		t.Errorf("csp = %q", got)
	}
}
