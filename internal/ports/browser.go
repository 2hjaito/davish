package ports

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"davish/internal/apperr"
)

// The in-app browser (docs/adr/0008-in-app-browser.md): one proxy listener
// per server port, on 127.0.0.1 at a port the OS picks, shown in an iframe
// at its own origin.

//go:embed nav.js
var navJS []byte

// tokenParam carries a site's token on the page load Davish starts.
const tokenParam = "__davish"

// navPath serves the script that reports the page's address to Davish.
const navPath = "/__davish/nav.js"

// Site is one server port opened in the browser.
type Site struct {
	ServerID   string `json:"serverID"`
	RemotePort int    `json:"remotePort"`
	LocalPort  int    `json:"localPort"`
	// Token must be in the first URL loaded; see ADR 0008.
	Token string `json:"token"`
}

type site struct {
	Site
	srv *http.Server
	jar http.CookieJar
}

// Browser owns the proxy listeners. Nothing is saved; they end with the app.
type Browser struct {
	conns ClientSource
	log   *slog.Logger

	mu sync.Mutex
	// sites holds the open listeners by server and remote port.
	sites map[string]*site
}

// NewBrowser creates the proxy service; call Close on shutdown.
func NewBrowser(conns ClientSource, log *slog.Logger) *Browser {
	return &Browser{conns: conns, log: log, sites: map[string]*site{}}
}

func siteKey(serverID string, port int) string { return serverID + "|" + strconv.Itoa(port) }

// Open returns the listener for the server's port, starting it if needed.
func (b *Browser) Open(serverID string, port int) (Site, error) {
	if port < 1 || port > 65535 {
		return Site{}, apperr.New(apperr.InvalidInput, "port must be 1-65535")
	}
	if _, err := b.conns.Client(serverID); err != nil {
		return Site{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.open(serverID, port)
}

// open needs b.mu.
func (b *Browser) open(serverID string, port int) (Site, error) {
	if s, ok := b.sites[siteKey(serverID, port)]; ok {
		return s.Site, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Site{}, apperr.Wrap(apperr.Internal, "cannot open a local port", err)
	}
	tok := make([]byte, 16)
	if _, err := rand.Read(tok); err != nil {
		ln.Close() //nolint:errcheck,gosec // unused
		return Site{}, apperr.Wrap(apperr.Internal, "cannot create a token", err)
	}
	jar, _ := cookiejar.New(nil) // never fails without options
	s := &site{
		Site: Site{
			ServerID: serverID, RemotePort: port,
			LocalPort: ln.Addr().(*net.TCPAddr).Port, Token: hex.EncodeToString(tok),
		},
		jar: jar,
	}
	s.srv = &http.Server{Handler: b.handler(s), ReadHeaderTimeout: 30 * time.Second}
	b.sites[siteKey(serverID, port)] = s
	go s.srv.Serve(ln) //nolint:errcheck,gosec // ends with Close
	b.log.Info("browser site opened", "server", serverID, "remote", port, "local", s.LocalPort)
	return s.Site, nil
}

// Sites lists the open listeners, for mapping local ports back to the VPS.
func (b *Browser) Sites() []Site {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Site, 0, len(b.sites))
	for _, s := range b.sites {
		out = append(out, s.Site)
	}
	slices.SortFunc(out, func(x, y Site) int { return x.LocalPort - y.LocalPort })
	return out
}

// Close stops every listener; the app is quitting.
func (b *Browser) Close() {
	b.mu.Lock()
	all := b.sites
	b.sites = map[string]*site{}
	b.mu.Unlock()
	for _, s := range all {
		s.srv.Close() //nolint:errcheck,gosec // shutting down
	}
}

// allowed applies ADR 0008: the right Host, and the token or a same-origin
// Referer/Origin.
func (s *site) allowed(r *http.Request) bool {
	self := "127.0.0.1:" + strconv.Itoa(s.LocalPort)
	if r.Host != self {
		return false
	}
	if t := r.URL.Query().Get(tokenParam); t != "" {
		return subtle.ConstantTimeCompare([]byte(t), []byte(s.Token)) == 1
	}
	origin := "http://" + self
	if o := r.Header.Get("Origin"); o != "" {
		return o == origin
	}
	ref := r.Header.Get("Referer")
	return ref == origin || strings.HasPrefix(ref, origin+"/")
}

func (b *Browser) handler(s *site) http.Handler {
	remote := net.JoinHostPort("127.0.0.1", strconv.Itoa(s.RemotePort))
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			client, err := b.conns.Client(s.ServerID)
			if err != nil {
				return nil, err
			}
			return client.DialContext(ctx, "tcp", remote)
		},
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(&url.URL{Scheme: "http", Host: remote})
			// The app sees itself addressed as on the server.
			r.Out.Host = "localhost:" + strconv.Itoa(s.RemotePort)
			q := r.Out.URL.Query()
			if q.Has(tokenParam) {
				q.Del(tokenParam)
				r.Out.URL.RawQuery = q.Encode()
			}
			// Plain bodies, so nav.js can be added to HTML.
			r.Out.Header.Del("Accept-Encoding")
			r.Out.Header.Del("Cookie")
			for _, c := range s.jar.Cookies(r.Out.URL) {
				r.Out.AddCookie(c)
			}
			// The app must not see Davish's addresses.
			r.Out.Header.Del("Referer")
			r.Out.Header.Del("Origin")
		},
		Transport:      transport,
		ModifyResponse: func(resp *http.Response) error { return b.rewrite(s, resp) },
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, "Davish: cannot reach the server's port: "+err.Error(), http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.allowed(r) {
			http.Error(w, "Davish: this address only opens inside Davish", http.StatusForbidden)
			return
		}
		if r.URL.Path == navPath {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(navJS) //nolint:errcheck,gosec // client gone
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

var frameAncestors = regexp.MustCompile(`(?i)\s*frame-ancestors[^;]*;?`)

// rewrite keeps the page in the frame and on the server: cookies to the
// jar, redirects to the matching listener, framing headers off, nav.js in.
func (b *Browser) rewrite(s *site, resp *http.Response) error {
	h := resp.Header
	if cs := resp.Cookies(); len(cs) > 0 {
		s.jar.SetCookies(resp.Request.URL, cs)
		h.Del("Set-Cookie")
	}
	h.Del("X-Frame-Options")
	for _, k := range []string{"Content-Security-Policy", "Content-Security-Policy-Report-Only"} {
		if v := h.Get(k); v != "" {
			h.Set(k, frameAncestors.ReplaceAllString(v, ""))
		}
	}
	h.Set("Referrer-Policy", "same-origin")
	if loc := h.Get("Location"); loc != "" {
		h.Set("Location", b.mapLocation(s, loc))
	}
	if resp.StatusCode == http.StatusSwitchingProtocols || resp.Request.Method == http.MethodHead {
		return nil
	}
	ct := h.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") || h.Get("Content-Encoding") != "" {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	resp.Body.Close() //nolint:errcheck,gosec // replaced below
	if err != nil {
		return err
	}
	body = injectNav(body)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	h.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}

var headTag = regexp.MustCompile(`(?i)<head[^>]*>`)

func injectNav(body []byte) []byte {
	tag := []byte(`<script src="` + navPath + `"></script>`)
	if loc := headTag.FindIndex(body); loc != nil {
		return slices.Concat(body[:loc[1]], tag, body[loc[1]:])
	}
	return slices.Concat(tag, body)
}

// mapLocation turns a redirect to the server's own localhost into the
// listener for that port, so the page stays on the VPS.
func (b *Browser) mapLocation(s *site, loc string) string {
	u, err := url.Parse(loc)
	if err != nil || !u.IsAbs() || u.Scheme != "http" {
		return loc
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "0.0.0.0", "::1":
	default:
		return loc
	}
	port := 80
	if p := u.Port(); p != "" {
		if port, err = strconv.Atoi(p); err != nil {
			return loc
		}
	}
	target := s.Site
	if port != s.RemotePort {
		b.mu.Lock()
		target, err = b.open(s.ServerID, port)
		b.mu.Unlock()
		if err != nil {
			return loc
		}
		q := u.Query()
		q.Set(tokenParam, target.Token)
		u.RawQuery = q.Encode()
	}
	u.Host = "127.0.0.1:" + strconv.Itoa(target.LocalPort)
	return u.String()
}
