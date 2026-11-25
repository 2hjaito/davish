package conn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/store"
)

type passphraseOnly string

func (p passphraseOnly) Get(_ string, kind store.SecretKind) (string, error) {
	if kind == store.SecretPassphrase && p != "" {
		return string(p), nil
	}
	return "", store.ErrNoSecret
}

// An ed25519 key with a passphrase, saved by Notepad with a byte order mark
// and CRLF line ends, as Windows users hand it over.
func TestLoadKeyEncryptedEd25519WithBOM(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const pass = "correct horse"
	block, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(pass))
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte("\xef\xbb\xbf"), pem.EncodeToMemory(block)...)
	var crlf []byte
	for _, b := range data {
		if b == '\n' {
			crlf = append(crlf, '\r')
		}
		crlf = append(crlf, b)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, crlf, 0o600); err != nil {
		t.Fatal(err)
	}

	m := &Manager{secrets: passphraseOnly(pass)}
	if _, err := m.loadKey(store.Server{ID: "s", KeyPath: path}); err != nil {
		t.Fatalf("loadKey: %v", err)
	}

	m = &Manager{secrets: passphraseOnly("")}
	if _, err := m.loadKey(store.Server{ID: "s", KeyPath: path}); err == nil {
		t.Fatal("want an error without a saved passphrase")
	}
}

// stuckAgent accepts the connection and never answers.
type stuckAgent struct{ closed chan struct{} }

func (s stuckAgent) Read([]byte) (int, error) {
	<-s.closed
	return 0, io.EOF
}
func (s stuckAgent) Write(b []byte) (int, error) { return len(b), nil }
func (s stuckAgent) Close() error {
	select {
	case <-s.closed:
	default:
		close(s.closed)
	}
	return nil
}

// An agent that never answers costs agentTimeout, not the whole connection.
func TestAgentSignersTimeout(t *testing.T) {
	agent := stuckAgent{closed: make(chan struct{})}
	oldDial, oldTimeout := agentDial, agentTimeout
	agentDial = func() (io.ReadWriteCloser, error) { return agent, nil }
	agentTimeout = 100 * time.Millisecond
	t.Cleanup(func() { agentDial, agentTimeout = oldDial, oldTimeout })

	start := time.Now()
	signers, release := (&Manager{}).agentSigners()
	release()
	if len(signers) != 0 || time.Since(start) > 2*time.Second {
		t.Fatalf("signers = %d after %v", len(signers), time.Since(start))
	}
	select {
	case <-agent.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the stuck agent was not closed")
	}
}

// An agent that cannot be reached leaves a line in the log, with why, so a
// support log can tell "no agent" from "an agent we could not open" (#168).
func TestAgentSignersLogsWhenUnreachable(t *testing.T) {
	oldDial := agentDial
	agentDial = func() (io.ReadWriteCloser, error) {
		return nil, errors.New(`open \\.\pipe\openssh-ssh-agent: All pipe instances are busy`)
	}
	t.Cleanup(func() { agentDial = oldDial })

	var buf bytes.Buffer
	m := &Manager{log: slog.New(slog.NewJSONHandler(&buf, nil))}
	signers, release := m.agentSigners()
	release()

	if len(signers) != 0 {
		t.Fatalf("signers = %d; want none", len(signers))
	}
	out := buf.String()
	if !strings.Contains(out, "ssh agent not reachable") || !strings.Contains(out, "All pipe instances are busy") {
		t.Errorf("log = %q; want the line and the cause", out)
	}
	// And with no logger, as in other tests, it must not panic.
	if _, release := (&Manager{}).agentSigners(); release != nil {
		release()
	}
}

func TestLoadKeyFolder(t *testing.T) {
	dir := t.TempDir()
	_, err := (&Manager{secrets: passphraseOnly("")}).loadKey(store.Server{KeyPath: dir})
	if err == nil || !strings.Contains(err.Error(), "folder") {
		t.Fatalf("err = %v", err)
	}
}

func TestExpandHomeSeparator(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	want := filepath.Join(home, ".ssh", "id_ed25519")
	if got := expandHome(filepath.Join("~", ".ssh", "id_ed25519")); got != want {
		t.Errorf("expandHome = %q; want %q", got, want)
	}
}
