package audit

import (
	"context"
	"strings"
	"testing"
	"time"

	"davish/internal/store"
)

func TestContextMarks(t *testing.T) {
	ctx := context.Background()
	if IsChanging(ctx) || passwordStdin(ctx) {
		t.Fatal("plain context is marked")
	}
	if !IsChanging(Changing(ctx)) || !passwordStdin(PasswordStdin(ctx)) {
		t.Fatal("marks lost")
	}
}

func TestTargets(t *testing.T) {
	got := Targets([]store.Server{
		{ID: "b", Name: "bastion", User: "jump", Host: "203.0.113.1", Port: 2200},
		{ID: "w", Name: "web", User: "deploy", Host: "10.0.0.5", Port: 22, JumpID: "b"},
	})
	if got["w"].Jump != "jump@203.0.113.1:2200" || got["b"].Jump != "" {
		t.Errorf("targets = %+v", got)
	}
}

func TestScript(t *testing.T) {
	at := time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC).UnixMilli()
	entries := []Entry{
		{ID: 3, Time: at + 2000, ServerID: "db", App: "services", Command: "LC_ALL=C sudo -n -- sh -c 'systemctl restart postgresql'"},
		{ID: 1, Time: at, ServerID: "web", App: "services", Command: "LC_ALL=C sudo -S -k -p '' -- sh -c 'systemctl restart nginx'"},
		{ID: 2, Time: at + 1000, ServerID: "web", App: "nginx", Command: "tee /etc/nginx/sites-available/x >/dev/null", Input: true},
		{ID: 4, Time: at + 3000, ServerID: "web", App: "docker", Command: "docker rm web", Exit: 1},
	}
	targets := map[string]Target{
		"web": {Name: "web", User: "deploy", Host: "10.0.0.5", Port: 22, Jump: "jump@203.0.113.1:2200"},
		"db":  {Name: "db", User: "root", Host: "db.example.com", Port: 2222},
	}
	got := Script(entries, targets, ScriptOptions{Now: time.Unix(0, 0)})
	want := `#!/usr/bin/env bash
# Exported from the Davish Command log at 1970-01-01 00:00 UTC.
# Each command runs over its own ssh connection, in the server's login
# shell (Davish ran them with sh -c). Read it before you run it.
set -euo pipefail

# --- web (deploy@10.0.0.5 via jump@203.0.113.1:2200) ---

# 2026-11-16 08:00:00 services
ssh -t -J jump@203.0.113.1:2200 deploy@10.0.0.5 -- 'sudo sh -c '\''systemctl restart nginx'\'''

# 2026-11-16 08:00:01 nginx
# content sent on stdin was not recorded; the command was:
#   tee /etc/nginx/sites-available/x >/dev/null

# --- db (root@db.example.com:2222) ---

# 2026-11-16 08:00:02 services
ssh -t -p 2222 root@db.example.com -- 'sudo sh -c '\''systemctl restart postgresql'\'''
`
	if got != want {
		t.Errorf("script:\n%s\nwant:\n%s", got, want)
	}

	withFailed := Script(entries, targets, ScriptOptions{Failed: true, Now: time.Unix(0, 0)})
	if !strings.Contains(withFailed, "# 2026-11-16 08:00:03 docker, exit 1 then\nssh -t -J jump@203.0.113.1:2200 deploy@10.0.0.5 -- 'docker rm web'\n") {
		t.Errorf("failed command missing:\n%s", withFailed)
	}
	if !strings.Contains(Script(nil, nil, ScriptOptions{}), "No commands to export") {
		t.Error("empty script has no note")
	}
}

func TestPlain(t *testing.T) {
	for in, want := range map[string]string{
		"LC_ALL=C sudo -n -- sh -c 'ls /root'":          "sudo sh -c 'ls /root'",
		"LC_ALL=C sudo -S -k -p '' -- sh -c 'ls /root'": "sudo sh -c 'ls /root'",
		"systemctl status nginx":                        "systemctl status nginx",
	} {
		if got := Plain(in); got != want {
			t.Errorf("Plain(%q) = %q, want %q", in, got, want)
		}
	}
}
