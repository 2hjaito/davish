package sudo

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/events"
	"davish/internal/lock"
)

const secret = "hunter2-sudo"

// fakeSudo plays a server whose sudo asks for secret.
type fakeSudo struct {
	uid    string
	policy string // "password", "nopassword", "none", "missing", "tty"
	groups string
	lines  []string
	inputs [][]byte
}

func (f *fakeSudo) RunInput(_ context.Context, _, _, line string, stdin []byte) (audit.Result, error) {
	f.lines = append(f.lines, line)
	f.inputs = append(f.inputs, append([]byte(nil), stdin...))
	if line == "id -u; id -Gn" {
		return audit.Result{Stdout: []byte(f.uid + "\n" + f.groups + "\n")}, nil
	}
	line, ok := strings.CutPrefix(line, "LC_ALL=C ")
	if !ok {
		return audit.Result{Stdout: []byte("ran as root: " + line)}, nil
	}
	switch f.policy {
	case "missing":
		return audit.Result{Exit: 127, Stderr: []byte("sh: 1: sudo: not found")}, nil
	case "none":
		return audit.Result{Exit: 1, Stderr: []byte("Sorry, user u may not run sudo on host.")}, nil
	case "tty":
		return audit.Result{Exit: 1, Stderr: []byte("sudo: sorry, you must have a tty to run sudo")}, nil
	case "nopassword":
		return audit.Result{Stdout: stdin}, nil
	}
	if strings.HasPrefix(line, "sudo -n ") {
		return audit.Result{Exit: 1, Stderr: []byte("sudo: a password is required")}, nil
	}
	got, rest, _ := bytes.Cut(stdin, []byte("\n"))
	if string(got) != secret {
		return audit.Result{Exit: 1, Stderr: []byte("sudo: 1 incorrect password attempt")}, nil
	}
	return audit.Result{Stdout: append([]byte(nil), rest...)}, nil
}

func code(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestRunAsks(t *testing.T) {
	ctx := context.Background()
	f := &fakeSudo{uid: "1000", policy: "password", groups: "u sudo"}
	k := New(f)

	if st, _ := k.State(ctx, "srv"); st != Password {
		t.Fatalf("state = %s", st)
	}
	if _, err := k.Run(ctx, "srv", "services", "systemctl restart nginx", nil); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("no password yet: %v", err)
	}
	if err := k.SetPassword(ctx, "srv", []byte("wrong")); code(err) != apperr.SudoPasswordWrong {
		t.Fatalf("wrong password: %v", err)
	}
	if err := k.SetPassword(ctx, "srv", []byte(secret)); err != nil {
		t.Fatal(err)
	}
	if st, _ := k.State(ctx, "srv"); st != Ready {
		t.Fatalf("state = %s", st)
	}
	res, err := k.Run(ctx, "srv", "crontab", "crontab -", []byte("* * * * * true\n"))
	if err != nil {
		t.Fatal(err)
	}
	// The command gets its own stdin and nothing of the password.
	if string(res.Stdout) != "* * * * * true\n" {
		t.Fatalf("command's stdin = %q", res.Stdout)
	}
	for _, l := range f.lines {
		if strings.Contains(l, secret) {
			t.Fatalf("password on a command line: %s", l)
		}
	}
	if last := f.lines[len(f.lines)-1]; last != `LC_ALL=C sudo -S -k -p '' -- sh -c 'crontab -'` {
		t.Errorf("line = %s", last)
	}

	k.Observe(events.ConnStatus, conn.Status{ServerID: "srv", State: conn.StateReconnecting})
	if _, err := k.Run(ctx, "srv", "services", "true", nil); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("after the connection dropped: %v", err)
	}
}

func TestPasswordThatStopsWorkingIsForgotten(t *testing.T) {
	ctx := context.Background()
	f := &fakeSudo{uid: "1000", policy: "password"}
	k := New(f)
	if err := k.SetPassword(ctx, "srv", []byte(secret)); err != nil {
		t.Fatal(err)
	}
	k.mu.Lock()
	k.pw["srv"] = []byte("changed-on-the-server")
	k.mu.Unlock()
	if _, err := k.Run(ctx, "srv", "services", "true", nil); code(err) != apperr.SudoPasswordWrong {
		t.Fatalf("stale password: %v", err)
	}
	if k.password("srv") != nil {
		t.Fatal("a wrong password was kept")
	}
}

func TestRunWithoutPassword(t *testing.T) {
	ctx := context.Background()
	for policy, want := range map[string]string{"nopassword": NoPassword, "none": None, "missing": None, "tty": None} {
		f := &fakeSudo{uid: "1000", policy: policy}
		k := New(f)
		if st, _ := k.State(ctx, "srv"); st != want {
			t.Errorf("%s: state %s, want %s", policy, st, want)
		}
		_, err := k.Run(ctx, "srv", "services", "systemctl stop x", nil)
		if want == NoPassword && err != nil || want == None && code(err) != apperr.CmdNeedsRoot {
			t.Errorf("%s: %v", policy, err)
		}
	}

	f := &fakeSudo{uid: "0"}
	k := New(f)
	res, err := k.Run(ctx, "srv", "services", "systemctl stop x", nil)
	if err != nil || string(res.Stdout) != "ran as root: systemctl stop x" {
		t.Fatalf("root: %q, %v", res.Stdout, err)
	}
	if st, _ := k.State(ctx, "srv"); st != Root {
		t.Errorf("root state = %s", st)
	}
}

func TestLockForgetsEveryPassword(t *testing.T) {
	ctx := context.Background()
	k := New(&fakeSudo{uid: "1000", policy: "password"})
	for _, id := range []string{"a", "b"} {
		if err := k.SetPassword(ctx, id, []byte(secret)); err != nil {
			t.Fatal(err)
		}
	}
	k.Observe(events.LockState, lock.State{Enabled: true, Locked: false})
	if k.password("a") == nil {
		t.Fatal("forgotten on unlock")
	}
	k.Observe(events.LockState, lock.State{Enabled: true, Locked: true})
	if k.password("a") != nil || k.password("b") != nil {
		t.Fatal("kept after locking")
	}
}

func TestSetPasswordRejectsLines(t *testing.T) {
	k := New(&fakeSudo{uid: "1000", policy: "password"})
	for _, pw := range []string{"", "a\nrm -rf /"} {
		if err := k.SetPassword(context.Background(), "srv", []byte(pw)); code(err) != apperr.InvalidInput {
			t.Errorf("%q: %v", pw, err)
		}
	}
}

// sudo asks a user outside sudo, wheel and admin for a password all the
// same; the state warns before the prompt.
func TestStateWarnsOutsideAdminGroups(t *testing.T) {
	ctx := context.Background()
	for groups, want := range map[string]string{"u docker": Unlikely, "u wheel": Password, "u admin users": Password} {
		k := New(&fakeSudo{uid: "1000", policy: "password", groups: groups})
		if st, _ := k.State(ctx, "srv"); st != want {
			t.Errorf("%s: state %s, want %s", groups, st, want)
		}
	}
}

func TestPrepareForAStream(t *testing.T) {
	ctx := context.Background()
	follow := "journalctl -f"

	root := New(&fakeSudo{uid: "0"})
	if line, in, err := root.Prepare(ctx, "srv", "logs", follow); err != nil || line != follow || in != nil {
		t.Fatalf("root: %q %q %v", line, in, err)
	}

	open := New(&fakeSudo{uid: "1000", policy: "nopassword", groups: "u sudo"})
	if line, in, err := open.Prepare(ctx, "srv", "logs", follow); err != nil || line != `LC_ALL=C sudo -n -- sh -c 'journalctl -f'` || in != nil {
		t.Fatalf("no password: %q %q %v", line, in, err)
	}

	none := New(&fakeSudo{uid: "1000", policy: "none"})
	if _, _, err := none.Prepare(ctx, "srv", "logs", follow); code(err) != apperr.CmdNeedsRoot {
		t.Fatalf("no sudo: %v", err)
	}

	f := &fakeSudo{uid: "1000", policy: "password", groups: "u sudo"}
	k := New(f)
	if _, _, err := k.Prepare(ctx, "srv", "logs", follow); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("before the password: %v", err)
	}
	if err := k.SetPassword(ctx, "srv", []byte(secret)); err != nil {
		t.Fatal(err)
	}
	line, in, err := k.Prepare(ctx, "srv", "logs", follow)
	if err != nil {
		t.Fatal(err)
	}
	if line != `LC_ALL=C sudo -S -k -p '' -- sh -c 'journalctl -f'` || strings.Contains(line, secret) {
		t.Errorf("line = %s", line)
	}
	if string(in) != secret+"\n" {
		t.Errorf("stdin = %q", in)
	}
	// What the caller is handed is its own: wiping it keeps the session's.
	wipe(in)
	if _, in2, _ := k.Prepare(ctx, "srv", "logs", follow); string(in2) != secret+"\n" {
		t.Errorf("after the caller wiped its copy: %q", in2)
	}
	if !RefusedPassword("sudo: 1 incorrect password attempt") || RefusedPassword("app: incorrect input") {
		t.Error("RefusedPassword")
	}
}
