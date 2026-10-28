package ports

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/events"
)

// ClientSource hands out a server's live SSH client.
type ClientSource interface {
	Client(serverID string) (*ssh.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Tunnel forwards 127.0.0.1:LocalPort on this machine to RemoteHost:RemotePort
// as seen from the server.
type Tunnel struct {
	ID         string `json:"id"`
	ServerID   string `json:"serverID"`
	RemoteHost string `json:"remoteHost"`
	RemotePort int    `json:"remotePort"`
	LocalPort  int    `json:"localPort"`
	// Moved is set when RemotePort was taken here and another was used.
	Moved bool `json:"moved"`
	// HostHeader, when set, makes the tunnel an HTTP proxy that sends this
	// Host to the server, for sites nginx tells apart by name.
	HostHeader string `json:"hostHeader,omitempty"`
	Conns      int    `json:"conns"`
}

type tunnel struct {
	Tunnel
	ln  net.Listener
	srv *http.Server // nil for a plain TCP tunnel
	wg  sync.WaitGroup
	// open holds the live connections of a plain tunnel, closed on stop.
	open map[net.Conn]struct{}
}

// Tunnels owns every forward. Nothing is saved: they end with the app.
type Tunnels struct {
	conns ClientSource
	emit  Emitter
	log   *slog.Logger

	mu sync.Mutex
	// byID holds the running tunnels; Stop and Close remove them.
	byID map[string]*tunnel
}

// NewTunnels creates the tunnel service; call Close on shutdown.
func NewTunnels(conns ClientSource, emit Emitter, log *slog.Logger) *Tunnels {
	return &Tunnels{conns: conns, emit: emit, log: log, byID: map[string]*tunnel{}}
}

// Start forwards remoteHost:remotePort of the server to this machine,
// keeping the port number when it is free here. Each connection takes the
// server's current SSH client, so a tunnel outlives reconnects.
func (s *Tunnels) Start(serverID, remoteHost string, remotePort int, hostHeader string) (Tunnel, error) {
	remoteHost = dialHost(remoteHost)
	if err := checkHost(remoteHost); err != nil {
		return Tunnel{}, err
	}
	if remotePort < 1 || remotePort > 65535 {
		return Tunnel{}, apperr.New(apperr.InvalidInput, "port must be 1-65535")
	}
	if hostHeader != "" {
		if err := checkHostHeader(hostHeader); err != nil {
			return Tunnel{}, err
		}
	}
	if _, err := s.conns.Client(serverID); err != nil {
		return Tunnel{}, err
	}

	// Never a wildcard address: nothing outside this machine may use it.
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(remotePort)))
	moved := false
	if err != nil {
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			return Tunnel{}, apperr.Wrap(apperr.Internal, "cannot open a local port", err)
		}
		moved = true
	}
	id, err := newID()
	if err != nil {
		ln.Close() //nolint:errcheck,gosec // unused
		return Tunnel{}, err
	}
	t := &tunnel{
		Tunnel: Tunnel{
			ID: id, ServerID: serverID, RemoteHost: remoteHost, RemotePort: remotePort,
			LocalPort: ln.Addr().(*net.TCPAddr).Port, Moved: moved, HostHeader: hostHeader,
		},
		ln:   ln,
		open: map[net.Conn]struct{}{},
	}
	out := t.Tunnel // read before connections start counting
	s.mu.Lock()
	s.byID[id] = t
	s.mu.Unlock()

	remote := net.JoinHostPort(remoteHost, strconv.Itoa(remotePort))
	t.wg.Add(1)
	if hostHeader != "" {
		t.srv = s.httpProxy(t, remote)
		go func() {
			defer t.wg.Done()
			t.srv.Serve(ln) //nolint:errcheck,gosec // ends with Shutdown
		}()
	} else {
		go func() {
			defer t.wg.Done()
			s.acceptLoop(t, remote)
		}()
	}
	s.log.Info("tunnel started", "server", serverID, "remote", remote, "local", t.LocalPort)
	s.changed()
	return out, nil
}

func (s *Tunnels) acceptLoop(t *tunnel, remote string) {
	for {
		c, err := t.ln.Accept()
		if err != nil {
			return
		}
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			s.pipe(t, c, remote)
		}()
	}
}

// pipe copies one local connection to the server and back.
func (s *Tunnels) pipe(t *tunnel, c net.Conn, remote string) {
	client, err := s.conns.Client(t.ServerID)
	if err != nil {
		c.Close() //nolint:errcheck,gosec // server offline
		return
	}
	rc, err := client.Dial("tcp", remote)
	if err != nil {
		c.Close() //nolint:errcheck,gosec // nothing listening there
		return
	}
	if !s.track(t, c, 1) {
		c.Close()  //nolint:errcheck,gosec // stopped meanwhile
		rc.Close() //nolint:errcheck,gosec // stopped meanwhile
		return
	}
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		io.Copy(dst, src) //nolint:errcheck,gosec // ends when either side closes
		if cw, ok := dst.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite() //nolint:errcheck,gosec // half-close
		}
		done <- struct{}{}
	}
	go cp(rc, c)
	go cp(c, rc)
	<-done
	<-done
	c.Close()  //nolint:errcheck,gosec // finished
	rc.Close() //nolint:errcheck,gosec // finished
	s.track(t, c, -1)
}

// track counts a connection in or out; false when the tunnel is stopping.
func (s *Tunnels) track(t *tunnel, c net.Conn, delta int) bool {
	s.mu.Lock()
	if _, live := s.byID[t.ID]; !live && delta > 0 {
		s.mu.Unlock()
		return false
	}
	if delta > 0 {
		t.open[c] = struct{}{}
	} else {
		delete(t.open, c)
	}
	t.Conns += delta
	s.mu.Unlock()
	s.changed()
	return true
}

// httpProxy rewrites Host on every request, keep-alive and websocket
// upgrades included, and dials the server through the current client.
func (s *Tunnels) httpProxy(t *tunnel, remote string) *http.Server {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			client, err := s.conns.Client(t.ServerID)
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
			r.Out.Host = t.HostHeader
		},
		Transport: transport,
	}
	return &http.Server{
		Handler:           proxy,
		ReadHeaderTimeout: 30 * time.Second,
		ConnState: func(c net.Conn, st http.ConnState) {
			switch st {
			case http.StateNew:
				s.track(t, c, 1)
			case http.StateClosed, http.StateHijacked:
				s.track(t, c, -1)
			}
		},
	}
}

// Stop closes the tunnel and its connections.
func (s *Tunnels) Stop(id string) {
	s.mu.Lock()
	t, ok := s.byID[id]
	delete(s.byID, id)
	var open []net.Conn
	if ok {
		for c := range t.open {
			open = append(open, c)
		}
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	s.shutdown(t, open)
	s.log.Info("tunnel stopped", "server", t.ServerID, "local", t.LocalPort)
	s.changed()
}

func (s *Tunnels) shutdown(t *tunnel, open []net.Conn) {
	if t.srv != nil {
		t.srv.Close() //nolint:errcheck,gosec // closes listener and connections
	} else {
		t.ln.Close() //nolint:errcheck,gosec // stops accepting
		for _, c := range open {
			c.Close() //nolint:errcheck,gosec // ends its pipe
		}
	}
	t.wg.Wait()
}

// List returns the running tunnels, oldest first.
func (s *Tunnels) List() []Tunnel {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Tunnel, 0, len(s.byID))
	for _, t := range s.byID {
		out = append(out, t.Tunnel)
	}
	slices.SortFunc(out, func(a, b Tunnel) int { return a.LocalPort - b.LocalPort })
	return out
}

// Close stops every tunnel; the app is quitting.
func (s *Tunnels) Close() {
	s.mu.Lock()
	all := s.byID
	s.byID = map[string]*tunnel{}
	open := map[*tunnel][]net.Conn{}
	for _, t := range all {
		for c := range t.open {
			open[t] = append(open[t], c)
		}
	}
	s.mu.Unlock()
	for _, t := range all {
		s.shutdown(t, open[t])
	}
}

func (s *Tunnels) changed() {
	s.emit.Emit(events.TunnelChanged, s.List())
}

func newID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Join(apperr.New(apperr.Internal, "cannot create tunnel id"), err)
	}
	return hex.EncodeToString(b), nil
}
