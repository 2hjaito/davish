//go:build !windows

package conn

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh/agent"
)

func TestAgentSignersFromSocket(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kr := agent.NewKeyring()
	if err := kr.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	// A short path: macOS limits Unix socket paths to 104 bytes.
	dir, err := os.MkdirTemp("", "ag")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck,gosec // test cleanup
	sock := filepath.Join(dir, "s")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() }) //nolint:errcheck,gosec // test cleanup
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(kr, c) //nolint:errcheck // ends with the connection
		}
	}()

	t.Setenv("SSH_AUTH_SOCK", sock)
	signers, release := (&Manager{}).agentSigners()
	defer release()
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(signers))
	}

	t.Setenv("SSH_AUTH_SOCK", "")
	if signers, release := (&Manager{}).agentSigners(); len(signers) != 0 {
		release()
		t.Fatalf("no agent: got %d signers", len(signers))
	}
}
