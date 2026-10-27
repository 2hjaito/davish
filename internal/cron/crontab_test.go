package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // fixed test fixtures
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func jobs(entries []Entry) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Kind == KindJob {
			out = append(out, e)
		}
	}
	return out
}

func TestParseUserCrontab(t *testing.T) {
	entries := ParseCrontab(fixture(t, "user-crontab"), false)
	got := jobs(entries)
	want := []struct {
		line           int
		sched, command string
		enabled        bool
	}{
		{9, "0 3 * * *", "/usr/local/bin/backup-db.sh >> /var/log/backup.log 2>&1", true},
		{10, "*/5 * * * *", "curl -fsS https://example.com/health > /dev/null", true},
		{11, "@reboot", "/home/deploy/start-worker.sh", true},
		{12, "@daily", "find /tmp -name '*.tmp' -mtime +7 -delete", true},
		{13, "30 2 * * 0", "/usr/local/bin/rotate-old.sh", false},
		{16, "0 9 1 * *", `echo "monthly at 9" | mail -s report ops@example.com`, true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d jobs: %+v", len(got), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Line != w.line || g.Schedule != w.sched || g.Command != w.command || g.Enabled != w.enabled || g.User != "" {
			t.Errorf("job %d = %+v, want %+v", i, g, w)
		}
	}
	env := map[string]string{}
	for _, e := range entries {
		if e.Kind == KindEnv {
			env[e.Name] = e.Value
		}
	}
	if env["MAILTO"] != "ops@example.com" || env["SHELL"] != "/bin/bash" || len(env) != 3 {
		t.Errorf("env = %v", env)
	}
	// The template's header and a commented example with a space are
	// comments, not jobs.
	for _, i := range []int{2, 13} {
		if entries[i].Kind != KindComment {
			t.Errorf("line %d = %+v", i+1, entries[i])
		}
	}
}

func TestParseSystemCrontabs(t *testing.T) {
	got := jobs(ParseCrontab(fixture(t, "etc-crontab"), true))
	if len(got) != 4 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Schedule != "17 * * * *" || got[0].User != "root" || got[0].Command != "cd / && run-parts --report /etc/cron.hourly" {
		t.Errorf("hourly = %+v", got[0])
	}
	// The example line in the comments names a user, not a schedule.
	for _, j := range got {
		if !j.Enabled {
			t.Errorf("disabled: %+v", j)
		}
	}

	got = jobs(ParseCrontab(fixture(t, "cron.d-e2scrub_all"), true))
	if len(got) != 2 || got[1].Schedule != "10 3 * * *" || got[1].User != "root" {
		t.Errorf("cron.d = %+v", got)
	}
}

func TestParseCrontabEdges(t *testing.T) {
	if ParseCrontab("", false) != nil {
		t.Error("empty crontab has entries")
	}
	e := ParseCrontab("61 * * * * nope\nFOO='a b'\n* * * * *\n", false)
	if e[0].Kind != KindComment || e[1].Value != "a b" || e[2].Kind != KindComment {
		t.Errorf("got %+v", e)
	}
}
