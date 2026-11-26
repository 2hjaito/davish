//go:build integration

package security_test

import (
	"net/netip"
	"strings"
	"testing"

	"davish/internal/testsshd"
)

// TestServerTellsWhereTheConnectionComesFrom checks what Apply relies on to
// never block the address Davish connects from: sshd sets SSH_CLIENT for
// a command run over a session, and its first word is an address.
func TestServerTellsWhereTheConnectionComesFrom(t *testing.T) {
	m, id := testsshd.Connect(t)
	out := strings.TrimSpace(testsshd.Run(t, m, id, `printf '%s' "${SSH_CLIENT%% *}"`))
	if _, err := netip.ParseAddr(out); err != nil {
		t.Fatalf("SSH_CLIENT gave %q: %v", out, err)
	}
}
