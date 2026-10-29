//go:build integration

// Package testsshd points integration tests at the throwaway SSH server in
// test/sshd (make sshd-up). It never targets a real server.
package testsshd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/crypto/ssh"
)

// Container address, user and ports of servers A and B, as in
// test/sshd/compose.yml.
const (
	Host  = "127.0.0.1"
	Port  = 2222
	PortB = 2223
	User  = "tester"
	// InnerHost is the server only reachable through the one at Port.
	InnerHost = "sshd-inner"
)

// Root is the repository root.
func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// KeyPath is the test client's private key.
func KeyPath() string {
	return filepath.Join(Root(), "test", "sshd", "keys", "client_ed25519")
}

// HostKey is the container's fixed host key.
func HostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(Root(), "test", "sshd", "keys", "host_ed25519.pub"))
	if err != nil {
		t.Fatalf("read test host key (run make sshd-up): %v", err)
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// Restart restarts the containers (all when services is empty), dropping
// every connection to them.
func Restart(t *testing.T, services ...string) {
	t.Helper()
	compose := filepath.Join(Root(), "test", "sshd", "compose.yml")
	out, err := exec.Command("docker", append([]string{"compose", "-f", compose, "restart"}, services...)...).CombinedOutput() //nolint:gosec // fixed test command
	if err != nil {
		t.Fatalf("restart test sshd: %v\n%s", err, out)
	}
}

// Exec runs a command inside the main test container as root, bypassing
// SSH: for putting things back that a test broke on purpose.
func Exec(t *testing.T, args ...string) {
	t.Helper()
	compose := filepath.Join(Root(), "test", "sshd", "compose.yml")
	cmd := append([]string{"compose", "-f", compose, "exec", "-T", "sshd"}, args...)
	if out, err := exec.Command("docker", cmd...).CombinedOutput(); err != nil { //nolint:gosec // fixed test command
		t.Errorf("exec in test sshd: %v\n%s", err, out)
	}
}
