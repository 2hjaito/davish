package svc

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestParseListFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/list-units.json")
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := parseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("testdata/list-units.plain")
	if err != nil {
		t.Fatal(err)
	}
	fromPlain := parsePlain(data)

	// Both outputs were taken from the same server at the same time.
	if !reflect.DeepEqual(tidy(fromJSON), tidy(fromPlain)) {
		t.Errorf("JSON and plain parse differently:\n%v\n%v", tidy(fromJSON), tidy(fromPlain))
	}
	units := tidy(fromJSON)
	want := map[string]Unit{
		"davish-ok.service":    {"davish-ok.service", "loaded", "active", "running", "Davish test service that keeps running"},
		"davish-crash.service": {"davish-crash.service", "loaded", "failed", "failed", "Davish test service that always fails"},
	}
	for _, u := range units {
		if w, ok := want[u.Name]; ok {
			if u != w {
				t.Errorf("%s = %+v, want %+v", u.Name, u, w)
			}
			delete(want, u.Name)
		}
		if u.Load == "not-found" {
			t.Errorf("%s is only referenced and should be dropped", u.Name)
		}
	}
	if len(want) > 0 {
		t.Errorf("missing units: %v", want)
	}
}

func TestParsePlain(t *testing.T) {
	out := "● nginx.service  loaded failed failed A high performance web server\n" +
		"cron.service loaded active running\n" +
		"\n" +
		"broken line\n"
	got := parsePlain([]byte(out))
	want := []Unit{
		{"nginx.service", "loaded", "failed", "failed", "A high performance web server"},
		{"cron.service", "loaded", "active", "running", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTidy(t *testing.T) {
	in := []Unit{
		{Name: "z.service", Load: "loaded", Active: "active"},
		{Name: "gone.service", Load: "not-found", Active: "inactive"},
		{Name: "deleted-but-running.service", Load: "not-found", Active: "active"},
		{Name: "a.socket", Load: "loaded", Active: "active"},
		{Name: "a.service", Load: "masked", Active: "inactive"},
	}
	var names []string
	for _, u := range tidy(in) {
		names = append(names, u.Name)
	}
	want := []string{"a.service", "deleted-but-running.service", "z.service"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestCheckUnit(t *testing.T) {
	tests := []struct {
		unit string
		ok   bool
	}{
		{"nginx.service", true},
		{"getty@tty1.service", true},
		{"systemd-fsck@dev-disk-by\\x2duuid-1.service", true},
		{"user-runtime-dir@1000.service", true},
		{"nginx", false},
		{"nginx.socket", false},
		{"-H.service", false},
		{"--help.service", false},
		{"a b.service", false},
		{"a;rm -rf /.service", false},
		{"$(id).service", false},
		{"", false},
	}
	for _, tt := range tests {
		err := CheckUnit(tt.unit)
		if (err == nil) != tt.ok {
			t.Errorf("CheckUnit(%q) = %v, want ok=%v", tt.unit, err, tt.ok)
		}
	}
}

func TestPlanAction(t *testing.T) {
	tests := []struct {
		unit, action string
		want         Plan
		wantErr      bool
	}{
		{
			unit: "nginx.service", action: "restart",
			want: Plan{"nginx.service", "restart", "systemctl restart nginx.service", true, "sudo systemctl restart nginx.service"},
		},
		{
			unit: "getty@tty1.service", action: "stop",
			want: Plan{"getty@tty1.service", "stop", "systemctl stop getty@tty1.service", true, "sudo systemctl stop getty@tty1.service"},
		},
		{
			unit: `a\x2db.service`, action: "start",
			want: Plan{`a\x2db.service`, "start", `systemctl start 'a\x2db.service'`, false, `sudo systemctl start 'a\x2db.service'`},
		},
		{unit: "nginx.service", action: "disable", wantErr: true},
		{unit: "nginx.service", action: "", wantErr: true},
		{unit: "-x.service", action: "start", wantErr: true},
	}
	for _, tt := range tests {
		got, err := PlanAction(tt.unit, tt.action)
		if tt.wantErr {
			if err == nil {
				t.Errorf("PlanAction(%q, %q) = %+v, want error", tt.unit, tt.action, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("PlanAction(%q, %q) = %+v, %v; want %+v", tt.unit, tt.action, got, err, tt.want)
		}
	}
}

func TestCheckExit(t *testing.T) {
	tests := []struct {
		name string
		res  audit.Result
		want apperr.Code
	}{
		{"ok", audit.Result{}, ""},
		{"polkit", audit.Result{Exit: 1, Stderr: []byte("Failed to stop nginx.service: Interactive authentication required.\n")}, apperr.CmdNeedsRoot},
		{"old dbus", audit.Result{Exit: 1, Stderr: []byte("Failed to restart x.service: Access denied\n")}, apperr.CmdNeedsRoot},
		{"no unit", audit.Result{Exit: 5, Stderr: []byte("Failed to start x.service: Unit x.service not found.\n")}, apperr.NotFound},
		{"no systemd", audit.Result{Exit: 127, Stderr: []byte("sh: systemctl: not found\n")}, apperr.CmdMissing},
		{"start failed", audit.Result{Exit: 1, Stderr: []byte("Job for x.service failed because the control process exited with error code.\n")}, apperr.CmdFailed},
	}
	for _, tt := range tests {
		err := checkExit(tt.res)
		var ae *apperr.Error
		got := apperr.Code("")
		if errors.As(err, &ae) {
			got = ae.Code
		}
		if got != tt.want {
			t.Errorf("%s: code = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestParseStatus(t *testing.T) {
	show := "Description=A high performance web server\n" +
		"LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\n" +
		"ActiveEnterTimestampMonotonic=12000000\nMainPID=812\nMemoryCurrent=4423680\n" +
		"NRestarts=2\nResult=success\nFragmentPath=/usr/lib/systemd/system/nginx.service\n"
	got := parseStatus("nginx.service", []byte(show), []byte("3612.47 7000.10\n"))
	want := Status{
		Unit: "nginx.service", Description: "A high performance web server",
		Load: "loaded", Active: "active", Sub: "running", Enabled: "enabled",
		ActiveFor: 3600, MainPID: 812, Memory: 4423680, Restarts: 2, Result: "success",
		File: "/usr/lib/systemd/system/nginx.service",
	}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	stopped := "ActiveState=failed\nMemoryCurrent=[not set]\nActiveEnterTimestampMonotonic=12000000\nMainPID=0\n"
	got = parseStatus("x.service", []byte(stopped), []byte("3612.47 1\n"))
	if got.ActiveFor != 0 || got.Memory != -1 {
		t.Errorf("stopped unit: ActiveFor %d, Memory %d; want 0, -1", got.ActiveFor, got.Memory)
	}
	untracked := "ActiveState=active\nMemoryCurrent=18446744073709551615\n"
	if got := parseStatus("x.service", []byte(untracked), nil); got.Memory != -1 {
		t.Errorf("untracked memory = %d, want -1", got.Memory)
	}
}

func TestParseJournal(t *testing.T) {
	out := "2026-10-27T15:35:04+00:00 host sh[34]: error: cannot open /etc/nginx/nginx.conf\n" +
		"-- Boot 1d2c --\n" +
		"2026-10-27T15:35:04+00:00 host systemd[1]: x.service: Failed with result 'exit-code'.\n"
	got := parseJournal([]byte(out), nil, "sudo journalctl")
	if len(got.Lines) != 3 || got.Restricted {
		t.Errorf("got %+v", got)
	}

	hint := "Hint: You are currently not seeing messages from other users and the system.\n" +
		"      Users in groups 'adm', 'systemd-journal' can see all messages.\n"
	got = parseJournal([]byte("-- No entries --\n"), []byte(hint), "sudo journalctl")
	if len(got.Lines) != 0 || !got.Restricted || got.Lines == nil {
		t.Errorf("restricted: got %+v", got)
	}

	none := "Failed to search journal ACL: Operation not supported\n" +
		"No journal files were opened due to insufficient permissions.\n"
	if got = parseJournal(nil, []byte(none), "sudo journalctl"); !got.Restricted {
		t.Errorf("no journal: got %+v", got)
	}
}

func TestPlanRemove(t *testing.T) {
	tests := []struct {
		name string
		st   Status
		mode string
		cmd  string
		sudo string
	}{
		{
			name: "package unit run by a timer",
			st: Status{Unit: "certbot.service", Load: "loaded", Enabled: "static",
				File: "/lib/systemd/system/certbot.service", TriggeredBy: "certbot.timer"},
			mode: RemoveMask,
			cmd:  "systemctl mask --now certbot.service certbot.timer && systemctl reset-failed certbot.service",
			sudo: "sudo systemctl mask --now certbot.service certbot.timer && sudo systemctl reset-failed certbot.service",
		},
		{
			name: "admin unit",
			st: Status{Unit: "btpanel.service", Load: "loaded", Enabled: "enabled",
				File: "/etc/systemd/system/btpanel.service"},
			mode: RemoveDelete,
			cmd: "systemctl disable --now btpanel.service && systemctl reset-failed btpanel.service && " +
				"rm -- /etc/systemd/system/btpanel.service && systemctl daemon-reload",
		},
		{
			name: "file already gone",
			st:   Status{Unit: "old.service", Load: "not-found"},
			mode: RemoveReset,
			cmd:  "systemctl reset-failed old.service",
		},
		{
			name: "already masked",
			st:   Status{Unit: "x.service", Load: "masked", File: "/dev/null", Enabled: "masked"},
			mode: RemoveReset,
			cmd:  "systemctl reset-failed x.service",
		},
		{
			name: "unit in a subfolder of /etc is a package's drop-in, not deleted",
			st:   Status{Unit: "y.service", Load: "loaded", File: "/etc/systemd/system/multi-user.target.wants/y.service"},
			mode: RemoveMask,
			cmd:  "systemctl mask --now y.service && systemctl reset-failed y.service",
		},
		{
			name: "odd trigger names are dropped",
			st:   Status{Unit: "z.service", Load: "loaded", File: "/usr/lib/systemd/system/z.service", TriggeredBy: "z.timer -bad;rm"},
			mode: RemoveMask,
			cmd:  "systemctl mask --now z.service z.timer && systemctl reset-failed z.service",
		},
	}
	for _, tt := range tests {
		got, err := planRemove(tt.st)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got.Mode != tt.mode || got.Command != tt.cmd {
			t.Errorf("%s:\n got %s: %s\nwant %s: %s", tt.name, got.Mode, got.Command, tt.mode, tt.cmd)
		}
		if tt.sudo != "" && got.Sudo != tt.sudo {
			t.Errorf("%s: sudo = %s", tt.name, got.Sudo)
		}
	}
}
