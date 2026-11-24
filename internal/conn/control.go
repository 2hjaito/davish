package conn

import (
	"context"
	"errors"
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/store"
)

// dialControl attaches to an OpenSSH ControlMaster socket the user started
// themselves (ssh -M -S <path> -fN host). OpenSSH did the login and checked
// the host key; Davish never sees a credential or a certificate. This is how
// agents that only answer a bound session (authentik, #240) can be used.
func (m *Manager) dialControl(ctx context.Context, srv store.Server) (*ssh.Client, error) {
	path := expandHome(srv.ControlPath)
	if err := checkControlSocket(path); err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: m.opts.DialTimeout}
	nc, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		return nil, apperr.Wrap(apperr.ConnControlSocket, "cannot connect to the control socket; is the ssh master still running?", err)
	}
	// Bound the handshake like a normal connection.
	timer := time.AfterFunc(m.opts.DialTimeout, func() { nc.Close() }) //nolint:errcheck,gosec // aborting
	c, chans, reqs, err := ssh.NewControlClientConn(nc)
	if !timer.Stop() && err == nil { // timed out just as the handshake finished
		c.Close() //nolint:errcheck,gosec // already closed underneath
		err = errors.New("control socket handshake timed out")
	}
	if err != nil {
		nc.Close() //nolint:errcheck,gosec // handshake failed
		return nil, apperr.Wrap(apperr.ConnControlSocket, "the control socket did not accept the connection", err)
	}
	return ssh.NewClient(c, chans, reqs), nil
}
