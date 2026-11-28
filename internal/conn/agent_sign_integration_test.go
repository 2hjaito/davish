//go:build integration

package conn_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/apperr"
	"davish/internal/conn"
	"davish/internal/store"
	"davish/internal/testsshd"
)

// refusingAgent lists its keys but will not sign with any: an agent whose
// person said no, or whose prompt never reached them (#332).
type refusingAgent struct{ agent.Agent }

func (refusingAgent) Sign(ssh.PublicKey, []byte) (*ssh.Signature, error) {
	return nil, errors.New("agent: client error: EOF")
}

func (r refusingAgent) SignWithFlags(ssh.PublicKey, []byte, agent.SignatureFlags) (*ssh.Signature, error) {
	return r.Sign(nil, nil)
}

func serveAgent(t *testing.T, a agent.Agent) {
	t.Helper()
	dir, err := os.MkdirTemp("", "dvagent") // short path: unix socket paths are limited
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	ln, err := net.Listen("unix", filepath.Join(dir, "a"))
	if err != nil {
		t.Skipf("no unix sockets here: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(a, c) //nolint:errcheck // ends with the connection
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", ln.Addr().String())
	t.Setenv("HOME", t.TempDir()) // no default keys: the agent is the only way in
}

func agentServer() store.Server {
	srv := testServer()
	srv.Auth, srv.KeyPath = store.AuthAgent, ""
	return srv
}

// A login the agent would not sign is reported as that, not as an
// unreachable server (#332).
func TestAgentThatRefusesToSign(t *testing.T) {
	key, err := os.ReadFile(testsshd.KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ssh.ParseRawPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	ring := agent.NewKeyring()
	if err := ring.Add(agent.AddedKey{PrivateKey: raw}); err != nil {
		t.Fatal(err)
	}
	serveAgent(t, refusingAgent{ring})

	e := setup(t, agentServer(), conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))
	if err := e.m.Connect(context.Background(), e.id); code(err) != apperr.ConnAgentSignFailed {
		t.Fatalf("err = %v; want CONN_AGENT_SIGN_FAILED", err)
	}
}

// An agent that signs fine with a key the server does not know is still a
// failed login, not an agent problem.
func TestAgentWithAKeyTheServerRejects(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ring := agent.NewKeyring()
	if err := ring.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	serveAgent(t, ring)

	e := setup(t, agentServer(), conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))
	if err := e.m.Connect(context.Background(), e.id); code(err) != apperr.ConnAuthFailed {
		t.Fatalf("err = %v; want CONN_AUTH_FAILED", err)
	}
}
