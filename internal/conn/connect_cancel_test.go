package conn

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/apperr"
	"davish/internal/events"
	"davish/internal/store"
)

type oneServer store.Server

func (s oneServer) Server(context.Context, string) (store.Server, error) { return store.Server(s), nil }

type statusLog struct {
	mu  sync.Mutex
	got []Status
}

func (l *statusLog) Emit(name string, payload any) {
	if st, ok := payload.(Status); ok && name == events.ConnStatus {
		l.mu.Lock()
		l.got = append(l.got, st)
		l.mu.Unlock()
	}
}

func (l *statusLog) states() []State {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]State, len(l.got))
	for i, st := range l.got {
		out[i] = st.State
	}
	return out
}

// stallAgent lists its key at once and holds every signing request until
// release is closed, like an agent waiting on a prompt nobody answers.
type stallAgent struct {
	agent.Agent
	signing chan struct{}
	release chan struct{}
	once    sync.Once
}

func (a *stallAgent) Sign(ssh.PublicKey, []byte) (*ssh.Signature, error) {
	a.once.Do(func() { close(a.signing) })
	<-a.release
	return nil, errors.New("released")
}

// Disconnect while the agent is still signing ends that handshake: a Connect
// right after must not leave a second signing request waiting behind the
// first, and the old attempt must not report "failed" over the new state
// (#332).
func TestDisconnectEndsAHandshakeWaitingOnTheAgent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("SSH_AUTH_SOCK", "")

	_, userPriv, _ := ed25519.GenerateKey(rand.Reader)
	userSigner, _ := ssh.NewSignerFromKey(userPriv)
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)

	srvCfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if bytes.Equal(k.Marshal(), userSigner.PublicKey().Marshal()) {
			return nil, nil
		}
		return nil, errors.New("unknown key")
	}}
	srvCfg.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() }) //nolint:errcheck,gosec // test cleanup
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { _, _, _, _ = ssh.NewServerConn(c, srvCfg) }()
		}
	}()

	kr := agent.NewKeyring()
	if err := kr.Add(agent.AddedKey{PrivateKey: userPriv}); err != nil {
		t.Fatal(err)
	}
	ag := &stallAgent{Agent: kr, signing: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(ag.release) })
	oldDial := agentDial
	agentDial = func() (io.ReadWriteCloser, error) {
		app, end := net.Pipe()
		go func() { _ = agent.ServeAgent(ag, end) }()
		return app, nil
	}
	t.Cleanup(func() { agentDial = oldDial })

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	hk := NewHostKeys(filepath.Join(t.TempDir(), "known_hosts"))
	if err := hk.Add(net.JoinHostPort(host, portStr), hostSigner.PublicKey()); err != nil {
		t.Fatal(err)
	}
	emit := &statusLog{}
	m := NewManager(oneServer{ID: "s", Host: host, Port: port, User: "u", Auth: store.AuthAgent},
		passphraseOnly(""), hk, emit, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{})
	t.Cleanup(m.Close)

	done := make(chan error, 1)
	go func() { done <- m.Connect(context.Background(), "s") }()
	select {
	case <-ag.signing:
	case <-time.After(5 * time.Second):
		t.Fatal("the agent was never asked to sign")
	}
	m.Disconnect("s")

	select {
	case err := <-done:
		if code(err) != apperr.ConnNotConnected {
			t.Fatalf("Connect = %v; want CONN_NOT_CONNECTED", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Disconnect left the handshake waiting on the agent")
	}
	states := emit.states()
	if last := states[len(states)-1]; last != StateDisconnected {
		t.Fatalf("states %v: the canceled attempt reported after Disconnect", states)
	}
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}
