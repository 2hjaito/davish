//go:build windows

package conn

import "testing"

func TestIsPipe(t *testing.T) {
	for path, want := range map[string]bool{
		`\\.\pipe\openssh-ssh-agent`: true,
		`//./pipe/openssh-ssh-agent`: true,
		`C:\Users\x\.ssh\agent.sock`: false,
		`/tmp/ssh-x/agent.1`:         false,
	} {
		if got := isPipe(path); got != want {
			t.Errorf("isPipe(%q) = %v, want %v", path, got, want)
		}
	}
}

// With no agent behind the pipe, there are no keys and no hang.
func TestAgentSignersMissingPipe(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", `\\.\pipe\davish-test-no-agent`)
	signers, release := (&Manager{}).agentSigners()
	defer release()
	if len(signers) != 0 {
		t.Fatalf("got %d signers", len(signers))
	}
}
