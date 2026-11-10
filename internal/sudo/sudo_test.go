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
	policy string // "password", "nopassword", "none", "missing"
	lines  []string
	inputs [][]byte
}

func (f *fakeSudo) RunInput(_ context.Context, _, _, line string, stdin []byte) (audit.Result, error) {
	f.lines = append(f.lines, line)
	f.inputs = append(f.inputs, append([]byte(nil), stdin...))
	if line == "id -u" {
		return audit.Result{Stdout: []byte(f.uid + "\n")}, nil
	}
	if !strings.HasPrefix(line, "sudo") {
		return audit.Result{Stdout: []byte("ran as root: " + line)}, nil
	}
	switch f.policy {
	case "missing":
		return audit.Result{Exit: 127, Stderr: []byte("sh: 1: sudo: not found")}, nil
	case "none":
		return audit.Result{Exit: 1, Stderr: []byte("Sorry, user u may not run sudo on host.")}, nil
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
	f := &fakeSudo{uid: "1000", policy: "password"}
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
	if last := f.lines[len(f.lines)-1]; last != `sudo -S -k -p '' -- sh -c 'crontab -'` {
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
	for policy, want := range map[string]string{"nopassword": NoPassword, "none": None, "missing": None} {
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
