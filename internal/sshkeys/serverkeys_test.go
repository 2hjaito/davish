package sshkeys

import (
	"strings"
	"testing"
)

func TestParseGitTest(t *testing.T) {
	tests := []struct {
		out, outcome, account string
	}{
		{"Hi theptaydo-deploy! You've successfully authenticated, but GitHub does not provide shell access.", GitOK, "theptaydo-deploy"},
		{"Welcome to GitLab, @dang!", GitOK, "dang"},
		{"git@github.com: Permission denied (publickey).", GitDenied, ""},
		{"davish:hostkey", GitHostKey, ""},
		{"davish:unreachable", GitUnreachable, ""},
		{"ssh: Could not resolve hostname github.com: Temporary failure in name resolution", GitUnreachable, ""},
		{"Load key \"/home/x/.ssh/k\": invalid format", GitOther, ""},
	}
	for _, tt := range tests {
		got := parseGitTest(tt.out)
		if got.Outcome != tt.outcome || got.Account != tt.account {
			t.Errorf("%q: got %+v", tt.out, got)
		}
	}
}

func TestGitTestCommand(t *testing.T) {
	cmd := gitTestCommand("github.com", "/home/dang/.ssh/deploy key")
	for _, want := range []string{
		"ssh-keyscan -t ed25519 github.com",
		"grep -qF -- SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU;",
		`-o UserKnownHostsFile="$f"`,
		"-o StrictHostKeyChecking=yes",
		"-i '/home/dang/.ssh/deploy key' git@github.com",
		`rm -f -- "$f"`,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("command lacks %q:\n%s", want, cmd)
		}
	}
}

func TestKeyName(t *testing.T) {
	for name, ok := range map[string]bool{
		"deploy_theptaydo": true, "id_ed25519.github": true,
		"-rf": false, ".hidden": false, "a/b": false, "": false, "x y": false,
	} {
		if keyName.MatchString(name) != ok {
			t.Errorf("keyName(%q) = %v", name, !ok)
		}
	}
}
