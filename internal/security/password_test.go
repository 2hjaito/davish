package security

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func statusOut(eff, include, ours, cfg string) []byte {
	return []byte("--eff--\n" + eff + "--include--\n" + include + "--ours--\n" + ours + "--cfg--\n" + cfg)
}

const plainEff = "passwordauthentication yes\nkbdinteractiveauthentication no\nauthenticationmethods any\n"

func TestParsePasswordStatus(t *testing.T) {
	cases := []struct {
		name    string
		out     []byte
		pw, kbd string
		ours    bool
		blocked []string
	}{
		{"plain", statusOut(plainEff, "yes\n", "", "PasswordAuthentication yes\n"), "yes", "no", false, nil},
		{"already ours", statusOut("passwordauthentication no\nkbdinteractiveauthentication no\nauthenticationmethods any\n", "yes\n", "yes\n", ""), "no", "no", true, nil},
		{"no include", statusOut(plainEff, "", "", ""), "yes", "no", false, []string{BlockNoInclude}},
		{"methods ask a password", statusOut("passwordauthentication yes\nauthenticationmethods publickey,password\n", "yes\n", "", ""), "yes", "", false, []string{BlockMethods}},
		{"methods ask keyboard-interactive", statusOut("authenticationmethods publickey,keyboard-interactive\n", "yes\n", "", ""), "", "", false, []string{BlockMethods}},
		{"methods publickey only", statusOut("passwordauthentication yes\nauthenticationmethods publickey\n", "yes\n", "", ""), "yes", "", false, nil},
		{"match turns it on", statusOut(plainEff, "yes\n", "", "Match User backup\n  PasswordAuthentication yes\n"), "yes", "no", false, []string{BlockMatch}},
	}
	for _, c := range cases {
		st := ParsePasswordStatus(c.out)
		if st.Password != c.pw || st.KbdInteractive != c.kbd || st.Ours != c.ours {
			t.Errorf("%s: %+v", c.name, st)
		}
		want := c.blocked
		if want == nil {
			want = []string{}
		}
		if !slices.Equal(st.Blocked, want) {
			t.Errorf("%s: blocked = %v, want %v", c.name, st.Blocked, want)
		}
	}
}

func TestMatchSetsPassword(t *testing.T) {
	cases := []struct {
		name string
		cfg  string
		want bool
	}{
		{"none", "PasswordAuthentication yes\nUsePAM yes\n", false},
		{"before any match", "PasswordAuthentication yes\nMatch User x\n  X11Forwarding no\n", false},
		{"match with no", "Match User x\n  PasswordAuthentication no\n", false},
		{"match with yes", "Match Address 10.0.0.0/8\n    PasswordAuthentication yes\n", true},
		{"keyboard-interactive in match", "Match User x\n KbdInteractiveAuthentication yes\n", true},
		{"methods in match", "Match User x\n AuthenticationMethods publickey,password\n", true},
		{"comment", "Match User x\n # PasswordAuthentication yes\n", false},
		{"case", "match user x\n  passwordauthentication YES\n", true},
		{"next file after a match", "Match User x\n  X11Forwarding no\n", false},
	}
	for _, c := range cases {
		if got := matchSetsPassword(strings.Split(c.cfg, "\n")); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestPlanPassword(t *testing.T) {
	p, r := PlanPassword(false), PlanPassword(true)
	if !strings.HasPrefix(p.Disable.Command, "sudo -n sh -c '") || !strings.HasPrefix(p.Disable.Terminal, "sudo sh -c '") {
		t.Errorf("sudo forms: %q / %q", p.Disable.Command, p.Disable.Terminal)
	}
	if !strings.HasPrefix(r.Disable.Command, "sh -c '") || r.Disable.Terminal != p.Disable.Terminal {
		t.Errorf("root form: %q", r.Disable.Command)
	}
	// 00- sorts before 50-cloud-init.conf, so sshd, which keeps the first
	// value it reads, uses ours.
	for _, want := range []string{"00-davish-no-password.conf", "sshd -t", "sshd -T", "cp -a /etc/ssh/sshd_config", "DAVISH_OVERRIDDEN"} {
		if !strings.Contains(p.Disable.Terminal, want) {
			t.Errorf("the change lacks %q", want)
		}
	}
	for _, want := range []string{"rm -f /etc/ssh/sshd_config.d/00-davish-no-password.conf", "sshd -t", "systemctl reload"} {
		if !strings.Contains(p.Enable.Terminal, want) {
			t.Errorf("the undo lacks %q", want)
		}
	}
}

type fakeKeys struct {
	errs  []error
	calls int
}

// CheckKeyLogin answers errs in turn, and the last one for ever after.
func (f *fakeKeys) CheckKeyLogin(context.Context, string) error {
	f.calls++
	if len(f.errs) == 0 {
		return nil
	}
	return f.errs[min(f.calls, len(f.errs))-1]
}

// stepRun answers each command in turn and remembers them.
type stepRun struct {
	res   []audit.Result
	lines []string
}

func (f *stepRun) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	f.lines = append(f.lines, line)
	if len(f.lines) > len(f.res) {
		return audit.Result{}, nil
	}
	return f.res[len(f.lines)-1], nil
}

func init() { afterReloadWait = 0 }

func newDisable(run *stepRun, keys *fakeKeys) *Service {
	s := New(run)
	s.UseKeys(keys)
	return s
}

func TestDisablePasswordChecksTheKeyFirst(t *testing.T) {
	run, keys := &stepRun{}, &fakeKeys{errs: []error{errors.New("denied")}}
	err := newDisable(run, keys).DisablePassword(context.Background(), "s", PlanPassword(true).Disable.Command)
	if !isCode(err, apperr.SecKeyLoginFailed) {
		t.Fatalf("err = %v", err)
	}
	if len(run.lines) != 0 {
		t.Errorf("ran %d commands although the key does not log in", len(run.lines))
	}
}

func TestDisablePasswordRefusesAnotherCommand(t *testing.T) {
	run, keys := &stepRun{}, &fakeKeys{}
	err := newDisable(run, keys).DisablePassword(context.Background(), "s", "rm -rf /")
	if !isCode(err, apperr.FSStale) || len(run.lines) != 0 {
		t.Fatalf("err = %v, ran %v", err, run.lines)
	}
}

func TestDisablePasswordWorks(t *testing.T) {
	run, keys := &stepRun{}, &fakeKeys{}
	cmd := PlanPassword(true).Disable.Command
	if err := newDisable(run, keys).DisablePassword(context.Background(), "s", cmd); err != nil {
		t.Fatal(err)
	}
	if keys.calls != 2 || len(run.lines) != 1 || run.lines[0] != cmd {
		t.Errorf("key checks %d, commands %q", keys.calls, run.lines)
	}
}

func TestDisablePasswordUndoesItWhenTheKeyIsLost(t *testing.T) {
	run, keys := &stepRun{}, &fakeKeys{errs: []error{nil, errors.New("denied")}}
	cmd := PlanPassword(true).Disable.Command
	err := newDisable(run, keys).DisablePassword(context.Background(), "s", cmd)
	if !isCode(err, apperr.SecKeyLoginLost) || !strings.Contains(err.Error(), "was undone") {
		t.Fatalf("err = %v", err)
	}
	if len(run.lines) != 2 || run.lines[1] != PlanPassword(true).Enable.Command {
		t.Errorf("commands = %q", run.lines)
	}
}

func TestDisablePasswordReportsWhatTheScriptUndid(t *testing.T) {
	for marker, want := range map[string]string{
		"DAVISH_SSHD_T":     "rejected",
		"DAVISH_OVERRIDDEN": "keeps password logins on",
		"DAVISH_NO_INCLUDE": "sshd_config.d",
		"DAVISH_RELOAD":     "reloaded",
	} {
		run := &stepRun{res: []audit.Result{{Exit: 4, Stdout: []byte(marker + "\n")}}}
		keys := &fakeKeys{}
		err := newDisable(run, keys).DisablePassword(context.Background(), "s", PlanPassword(true).Disable.Command)
		if !isCode(err, apperr.CmdFailed) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", marker, err)
		}
		if keys.calls != 1 {
			t.Errorf("%s: tried the key %d times, want only before", marker, keys.calls)
		}
	}
}

func TestEnablePassword(t *testing.T) {
	run := &stepRun{}
	cmd := PlanPassword(true).Enable.Command
	if err := New(run).EnablePassword(context.Background(), "s", cmd); err != nil || len(run.lines) != 1 {
		t.Fatalf("err = %v, ran %q", err, run.lines)
	}
	if err := New(run).EnablePassword(context.Background(), "s", "id"); !isCode(err, apperr.FSStale) {
		t.Errorf("another command: %v", err)
	}
}

func TestReportSeesOurDropIn(t *testing.T) {
	r := parse([]byte("0\n--ours--\nyes\n--sshd--\npasswordauthentication no\n"))
	if !r.PasswordOurs || r.PasswordAuth != "no" {
		t.Errorf("ours %v, auth %q", r.PasswordOurs, r.PasswordAuth)
	}
	if parse([]byte("0\n--sshd--\npasswordauthentication no\n")).PasswordOurs {
		t.Error("ours without the file")
	}
}

// sshd restarts on a reload: a key that is refused for a moment and then
// logs in is not a lost key.
func TestDisablePasswordWaitsForSshdToRestart(t *testing.T) {
	run, keys := &stepRun{}, &fakeKeys{errs: []error{nil, errors.New("refused"), errors.New("refused"), nil}}
	if err := newDisable(run, keys).DisablePassword(context.Background(), "s", PlanPassword(true).Disable.Command); err != nil {
		t.Fatal(err)
	}
	if keys.calls != 4 || len(run.lines) != 1 {
		t.Errorf("key checks %d, commands %q", keys.calls, run.lines)
	}
}
