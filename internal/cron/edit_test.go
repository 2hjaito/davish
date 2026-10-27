package cron

import (
	"testing"
)

func TestJobLine(t *testing.T) {
	tests := []struct {
		sched, cmd, want string
		ok               bool
	}{
		{"*/5  *  * * *", " /usr/bin/backup.sh  --all ", "*/5 * * * * /usr/bin/backup.sh  --all", true},
		{"@daily", "date +\\%F >> /tmp/d", "@daily date +\\%F >> /tmp/d", true},
		{"61 * * * *", "x", "", false},
		{"* * * * *", "", "", false},
		{"* * * * *", "echo a\n* * * * * rm -rf ~", "", false},
		{"* * * * *", "echo a\rb", "", false},
	}
	for _, tt := range tests {
		got, err := jobLine(tt.sched, tt.cmd)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("jobLine(%q, %q) = %q, %v", tt.sched, tt.cmd, got, err)
		}
	}
}

func TestFindJob(t *testing.T) {
	cur := "MAILTO=x\n0 3 * * * a\n#5 * * * * b\n"
	id := lineID(SourceUser, 3, "#5 * * * * b")
	if i, err := findJob(cur, id); err != nil || i != 2 {
		t.Errorf("findJob = %d, %v", i, err)
	}
	// Same line number, other content: changed meanwhile.
	if _, err := findJob("MAILTO=x\n0 3 * * * a\n#5 * * * * c\n", id); err == nil {
		t.Error("changed line found")
	}
}

func TestParseRuns(t *testing.T) {
	out := "2026-10-27T16:00:01+00:00 h CRON[100]: (tester) CMD (/usr/bin/backup.sh)\n" +
		"2026-10-27T16:00:01+00:00 h CRON[101]: (root) CMD (cd / && run-parts --report /etc/cron.hourly)\n" +
		"2026-10-27T16:05:01+00:00 h CRON[102]: (tester) CMD (/usr/bin/backup.sh)\n" +
		"2026-10-27T16:05:01+00:00 h CRON[103]: (tester) CMD (/usr/bin/backup.sh --other)\n"
	r := parseRuns(out, "", " /usr/bin/backup.sh ")
	if len(r.Lines) != 2 || r.Restricted {
		t.Errorf("runs = %+v", r)
	}
	r = parseRuns("-- No entries --\n", "No journal files were opened due to insufficient permissions.\n", "x")
	if !r.Restricted || len(r.Lines) != 0 {
		t.Errorf("restricted = %+v", r)
	}
}
