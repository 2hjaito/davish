//go:build !windows

package conn

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/store"
)

func startTestAgent(t *testing.T, dir string, keys ...any) string {
	t.Helper()
	kr := agent.NewKeyring()
	for _, priv := range keys {
		if err := kr.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
			t.Fatal(err)
		}
	}
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
	return sock
}

func TestAgentSignersFromSocket(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// A short path: macOS limits Unix socket paths to 104 bytes.
	dir, err := os.MkdirTemp("", "ag")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck,gosec // test cleanup
	sock := startTestAgent(t, dir, priv)

	t.Setenv("SSH_AUTH_SOCK", sock)
	signers, release := (&Manager{}).agentSigners(nil)
	defer release()
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(signers))
	}

	t.Setenv("SSH_AUTH_SOCK", "")
	if signers, release := (&Manager{}).agentSigners(nil); len(signers) != 0 {
		release()
		t.Fatalf("no agent: got %d signers", len(signers))
	}
}

func TestAgentSignersFromTildeSocket(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "agh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck,gosec // test cleanup
	startTestAgent(t, dir, priv)

	t.Setenv("HOME", dir)
	t.Setenv("SSH_AUTH_SOCK", "~/s")
	signers, release := (&Manager{}).agentSigners(nil)
	defer release()
	if len(signers) != 1 {
		t.Fatalf("got %d signers from ~/s, want 1", len(signers))
	}
}

type staticServers map[string]store.Server

func (s staticServers) Server(_ context.Context, id string) (store.Server, error) {
	srv, ok := s[id]
	if !ok {
		return store.Server{}, errors.New("not found")
	}
	return srv, nil
}

type discardEmitter struct{}

func (discardEmitter) Emit(string, any) {}

func startTestSSHServer(t *testing.T, allowed ssh.PublicKey) (string, int, ssh.PublicKey) {
	t.Helper()
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	wantBlob := allowed.Marshal()
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), wantBlob) {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("rejected")
		},
	}
	cfg.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() }) //nolint:errcheck,gosec // test cleanup

	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				sc, chans, reqs, err := ssh.NewServerConn(nc, cfg)
				if err != nil {
					nc.Close() //nolint:errcheck,gosec // failed handshake
					return
				}
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					ch.Reject(ssh.UnknownChannelType, "test") //nolint:errcheck,gosec // test server
				}
				sc.Wait() //nolint:errcheck,gosec // test server
			}()
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, hostSigner.PublicKey()
}

// End-to-end SSH authentication through $SSH_AUTH_SOCK for Automatic auth,
// an encrypted private key unlocked in the agent, and a .pub key path.
func TestConnectWithSSHAuthSock(t *testing.T) {
	pubRaw, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ssh.NewPublicKey(pubRaw)
	if err != nil {
		t.Fatal(err)
	}

	sockDir, err := os.MkdirTemp("", "ags")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sockDir) }) //nolint:errcheck,gosec // test cleanup
	sock := startTestAgent(t, sockDir, priv)
	t.Setenv("SSH_AUTH_SOCK", sock)

	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)

	host, port, hostPub := startTestSSHServer(t, pub)
	hkFile := filepath.Join(t.TempDir(), "known_hosts")
	hk := NewHostKeys(hkFile)
	if err := hk.Add(net.JoinHostPort(host, strconv.Itoa(port)), hostPub); err != nil {
		t.Fatal(err)
	}

	// Write an encrypted private key with NO .pub file and NO saved passphrase,
	// plus a standalone .pub file with no private key beside it.
	keyDir := t.TempDir()
	encBlock, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte("secret-pass"))
	if err != nil {
		t.Fatal(err)
	}
	encPath := filepath.Join(keyDir, "id_enc")
	if err := os.WriteFile(encPath, pem.EncodeToMemory(encBlock), 0o600); err != nil {
		t.Fatal(err)
	}
	pubPath := filepath.Join(keyDir, "agent_only.pub")
	if err := os.WriteFile(pubPath, ssh.MarshalAuthorizedKey(pub), 0o600); err != nil {
		t.Fatal(err)
	}

	servers := staticServers{
		"auto": {ID: "auto", Host: host, Port: port, User: "tester", Auth: store.AuthAgent},
		"enc":  {ID: "enc", Host: host, Port: port, User: "tester", Auth: store.AuthKey, KeyPath: encPath},
		"pub":  {ID: "pub", Host: host, Port: port, User: "tester", Auth: store.AuthKey, KeyPath: pubPath},
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := NewManager(servers, passphraseOnly(""), hk, discardEmitter{}, log, Options{})
	t.Cleanup(m.Close)

	for _, id := range []string{"auto", "enc", "pub"} {
		t.Run(id, func(t *testing.T) {
			if err := m.Connect(context.Background(), id); err != nil {
				t.Fatalf("Connect(%s): %v", id, err)
			}
			defer m.Disconnect(id)

			fps, err := m.LoginKeys(context.Background(), id)
			if err != nil {
				t.Fatalf("LoginKeys(%s): %v", id, err)
			}
			wantFP := ssh.FingerprintSHA256(pub)
			if len(fps) != 1 || fps[0] != wantFP {
				t.Fatalf("LoginKeys(%s) = %v; want [%s]", id, fps, wantFP)
			}
		})
	}

	// Deduplication: same key in both $SSH_AUTH_SOCK and ~/.ssh/id_ed25519
	// should only appear once.
	plainBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	sshDir := filepath.Join(emptyHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519"), pem.EncodeToMemory(plainBlock), 0o600); err != nil {
		t.Fatal(err)
	}
	fps, err := m.LoginKeys(context.Background(), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if len(fps) != 1 {
		t.Fatalf("deduped LoginKeys = %v; want 1 unique key", fps)
	}
}

func TestShouldProbeShellAgent(t *testing.T) {
	for _, tt := range []struct {
		name    string
		goos    string
		ppid    int
		term    string
		bench   string
		display string
		wayland string
		sock    string
		want    bool
	}{
		{
			name: "darwin GUI with launchd default socket",
			goos: "darwin", ppid: 1, sock: "/private/tmp/com.apple.launchd.abc123/Listeners",
			want: true,
		},
		{
			name: "darwin GUI with empty socket",
			goos: "darwin", ppid: 1, sock: "",
			want: true,
		},
		{
			name: "darwin GUI with explicit launchctl custom socket",
			goos: "darwin", ppid: 1, sock: "/Users/me/.1password/agent.sock",
			want: false,
		},
		{
			name: "darwin terminal launch",
			goos: "darwin", ppid: 4242, term: "xterm-256color", sock: "/private/tmp/com.apple.launchd.abc123/Listeners",
			want: false,
		},
		{
			name: "darwin bench",
			goos: "darwin", ppid: 1, bench: "1", sock: "",
			want: false,
		},
		{
			name: "linux GUI X11",
			goos: "linux", ppid: 900, display: ":0", sock: "",
			want: true,
		},
		{
			name: "linux GUI Wayland",
			goos: "linux", ppid: 900, wayland: "wayland-0", sock: "/run/user/1000/keyring/ssh",
			want: true,
		},
		{
			name: "linux terminal launch",
			goos: "linux", ppid: 900, term: "xterm-256color", display: ":0", sock: "",
			want: false,
		},
		{
			name: "linux headless test",
			goos: "linux", ppid: 900, sock: "",
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldProbeShellAgent(tt.goos, tt.ppid, tt.term, tt.bench, tt.display, tt.wayland, tt.sock)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseShellAgentOutput(t *testing.T) {
	for _, tt := range []struct {
		name string
		out  string
		want string
	}{
		{
			name: "noisy profile with spaces in path",
			out:  "welcome!\n\x1b[32mok\x1b[0m\n" + shellAgentStart + "/Users/me/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock" + shellAgentEnd + "\nbye\n",
			want: "/Users/me/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock",
		},
		{
			name: "empty socket",
			out:  shellAgentStart + "" + shellAgentEnd + "\n",
			want: "",
		},
		{
			name: "missing markers",
			out:  "/tmp/ssh/agent.sock\n",
			want: "",
		},
		{
			name: "takes last pair",
			out:  shellAgentStart + "/old" + shellAgentEnd + "\n" + shellAgentStart + "/new/sock" + shellAgentEnd + "\n",
			want: "/new/sock",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseShellAgentOutput([]byte(tt.out)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProbeShellAgentSock(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "fake-sh")
	script := `#!/bin/sh
echo "noisy login banner"
export SSH_AUTH_SOCK="/tmp/my agent dir/ssh-agent.sock"
# Execute the command passed after -c
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-c" ]; then
    shift
    exec /bin/sh -c "$1"
  fi
  shift
done
`
	if err := os.WriteFile(fakeShell, []byte(script), 0o700); err != nil { //nolint:gosec // test script in TempDir
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := probeShellAgentSock(ctx, fakeShell)
	if err != nil {
		t.Fatalf("probeShellAgentSock: %v", err)
	}
	if got != "/tmp/my agent dir/ssh-agent.sock" {
		t.Errorf("got %q, want %q", got, "/tmp/my agent dir/ssh-agent.sock")
	}

	// Fallback to -l -c when -i fails.
	fallbackShell := filepath.Join(dir, "fallback-sh")
	fallbackScript := `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = "-i" ]; then
    exit 1
  fi
done
export SSH_AUTH_SOCK="/tmp/from-login-only.sock"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-c" ]; then
    shift
    exec /bin/sh -c "$1"
  fi
  shift
done
`
	if err := os.WriteFile(fallbackShell, []byte(fallbackScript), 0o700); err != nil { //nolint:gosec // test script in TempDir
		t.Fatal(err)
	}
	got, err = probeShellAgentSock(ctx, fallbackShell)
	if err != nil {
		t.Fatalf("fallback probeShellAgentSock: %v", err)
	}
	if !strings.EqualFold(got, "/tmp/from-login-only.sock") {
		t.Errorf("fallback got %q", got)
	}
}
