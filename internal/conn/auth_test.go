package conn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/apperr"
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
	signers, release := (&Manager{}).agentSigners(nil)
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

// An agent that lists no keys is asked once more: 1Password on Windows lists
// none the first time while its vault is locked, then its keys (#332).
func TestAgentSignersAsksAgainWhenEmpty(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	full := agent.NewKeyring()
	if err := full.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	asked := 0
	oldDial, oldRetry := agentDial, agentEmptyRetry
	agentDial = func() (io.ReadWriteCloser, error) {
		asked++
		kr := agent.NewKeyring() // empty the first time
		if asked > 1 {
			kr = full
		}
		app, ag := net.Pipe()
		go func() { _ = agent.ServeAgent(kr, ag) }()
		return app, nil
	}
	agentEmptyRetry = time.Millisecond
	t.Cleanup(func() { agentDial, agentEmptyRetry = oldDial, oldRetry })

	signers, release := (&Manager{}).agentSigners(nil)
	defer release()
	if asked != 2 || len(signers) != 1 {
		t.Fatalf("asked %d times, got %d keys; want 2 and 1", asked, len(signers))
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
	signers, release := m.agentSigners(nil)
	release()

	if len(signers) != 0 {
		t.Fatalf("signers = %d; want none", len(signers))
	}
	out := buf.String()
	if !strings.Contains(out, "ssh agent not reachable") || !strings.Contains(out, "All pipe instances are busy") {
		t.Errorf("log = %q; want the line and the cause", out)
	}
	// And with no logger, as in other tests, it must not panic.
	if _, release := (&Manager{}).agentSigners(nil); release != nil {
		release()
	}
}

// DAVISH_AGENT_WAIT raises how long the agent is waited for; a bad value is
// ignored (#168).
func TestAgentWait(t *testing.T) {
	for _, tt := range []struct {
		env  string
		want time.Duration
	}{
		{"", agentTimeout},
		{"60", 60 * time.Second},
		{"1", time.Second},
		{"300", 300 * time.Second},
		{"0", agentTimeout},
		{"301", agentTimeout},
		{"-5", agentTimeout},
		{"soon", agentTimeout},
	} {
		t.Setenv("DAVISH_AGENT_WAIT", tt.env)
		if got := agentWait(); got != tt.want {
			t.Errorf("DAVISH_AGENT_WAIT=%q: wait = %v; want %v", tt.env, got, tt.want)
		}
	}
}

// A PuTTY key file Davish reads (versions 2 and 3) logs in like any other key.
func TestLoadKeyPPKReadable(t *testing.T) {
	for file, pass := range map[string]string{"ed25519-v3.ppk.txt": "", "rsa-v2-enc.ppk.txt": "test pass"} {
		path := filepath.Join("..", "keyfile", "testdata", file)
		signer, err := (&Manager{secrets: passphraseOnly(pass)}).loadKey(store.Server{KeyPath: path})
		if err != nil || signer == nil {
			t.Errorf("%s: signer = %v, err = %v", file, signer, err)
		}
	}
}

// A PuTTY key file of a kind Davish does not read gets its own error code and
// says how to convert it (#264).
func TestLoadKeyPPK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.ppk")
	if err := os.WriteFile(path, []byte("PuTTY-User-Key-File-1: ssh-rsa\nEncryption: none\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (&Manager{secrets: passphraseOnly("")}).loadKey(store.Server{KeyPath: path})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ConnKeyPPK || !strings.Contains(ae.Message, "Export OpenSSH key") {
		t.Fatalf("err = %v; want CONN_KEY_PPK with the way to convert", err)
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
