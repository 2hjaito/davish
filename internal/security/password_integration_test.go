//go:build integration

package security_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/security"
	"davish/internal/sudo"
	"davish/internal/testsshd"
)

const dropInFile = "/etc/ssh/sshd_config.d/00-davish-no-password.conf"

// ready is the Security service for the sudoer, with the sudo password
// given, and the real key check.
func ready(t *testing.T) (*security.Service, string, *conn.Manager) {
	t.Helper()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	run := audit.NewRunner(m, audit.New(discard{}))
	k := sudo.New(run)
	s := security.New(run)
	s.UseRoot(k)
	s.UseKeys(m)
	if err := k.SetPassword(context.Background(), id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	// Whatever a test leaves behind is removed, so the others start clean.
	t.Cleanup(func() {
		testsshd.Exec(t, "sh", "-c", "rm -f "+dropInFile+" /etc/ssh/sshd_config.d/00-aaa-yes.conf; kill -HUP $(cat /run/sshd.pid)")
	})
	return s, id, m
}

// failSecond lets the first key check through and fails every later one,
// as if the change had locked the key out.
type failSecond struct {
	inner security.KeyChecker
	calls int
}

func (f *failSecond) CheckKeyLogin(ctx context.Context, id string) error {
	f.calls++
	if f.calls >= 2 {
		return errors.New("denied")
	}
	return f.inner.CheckKeyLogin(ctx, id)
}

func TestDisableAndEnablePassword(t *testing.T) {
	ctx := context.Background()
	s, id, m := ready(t)

	st, err := s.ReadPasswordStatus(ctx, id, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.Ours || len(st.Blocked) != 0 {
		t.Fatalf("before: %+v", st)
	}

	if err := s.DisablePassword(ctx, id, security.PlanPassword(false).Disable.Command); err != nil {
		t.Fatal(err)
	}
	if st, err = s.ReadPasswordStatus(ctx, id, true, false); err != nil || !st.Ours || st.Password != "no" || st.KbdInteractive != "no" {
		t.Fatalf("after: %+v, %v", st, err)
	}
	if out := testsshd.Run(t, m, id, "ls -d /var/backups/davish-sshd-*"); !strings.Contains(out, "/var/backups/davish-sshd-") {
		t.Errorf("no backup: %q", out)
	}
	if err := m.CheckKeyLogin(ctx, id); err != nil {
		t.Errorf("the key no longer logs in: %v", err)
	}

	if err := s.EnablePassword(ctx, id, security.PlanPassword(false).Enable.Command); err != nil {
		t.Fatal(err)
	}
	if st, err = s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Ours {
		t.Fatalf("after undo: %+v, %v", st, err)
	}
}

// A file that sorts before ours and turns passwords on wins in sshd: the
// change is undone, not reported as done.
func TestDisablePasswordUndoesWhenAnotherFileWins(t *testing.T) {
	ctx := context.Background()
	s, id, _ := ready(t)
	testsshd.Exec(t, "sh", "-c", "printf 'PasswordAuthentication yes\\n' > /etc/ssh/sshd_config.d/00-aaa-yes.conf; kill -HUP $(cat /run/sshd.pid)")

	err := s.DisablePassword(ctx, id, security.PlanPassword(false).Disable.Command)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CmdFailed || !strings.Contains(ae.Message, "keeps password logins on") {
		t.Fatalf("err = %v", err)
	}
	if st, err := s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Ours {
		t.Errorf("the drop-in stayed: %+v, %v", st, err)
	}
}

// The key stops logging in after the change: it is undone over the open
// session.
func TestDisablePasswordUndoesWhenTheKeyIsLost(t *testing.T) {
	ctx := context.Background()
	s, id, m := ready(t)
	s.UseKeys(&failSecond{inner: m})

	err := s.DisablePassword(ctx, id, security.PlanPassword(false).Disable.Command)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.SecKeyLoginLost || !strings.Contains(ae.Message, "was undone") {
		t.Fatalf("err = %v", err)
	}
	if st, err := s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Ours {
		t.Errorf("the drop-in stayed: %+v, %v", st, err)
	}
}

// Without a key, a server that logs in with a password cannot be checked.
func TestCheckKeyLoginNeedsAKey(t *testing.T) {
	ctx := context.Background()
	s, id, _ := ready(t)
	st, err := s.ReadPasswordStatus(ctx, id, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Blocked) != 2 || st.Blocked[0] != security.BlockPassword && st.Blocked[1] != security.BlockPassword {
		t.Errorf("blocked = %v", st.Blocked)
	}
}

// With password logins really on, the change turns them off, ahead of the
// file that turned them on, and undoing it brings them back.
func TestDisablePasswordWhenItIsOn(t *testing.T) {
	ctx := context.Background()
	s, id, _ := ready(t)
	const conf = "/etc/ssh/sshd_config.d/davish-test.conf"
	testsshd.Exec(t, "sh", "-c", "sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/' "+conf+"; kill -HUP $(cat /run/sshd.pid)")
	t.Cleanup(func() {
		testsshd.Exec(t, "sh", "-c", "sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' "+conf+"; kill -HUP $(cat /run/sshd.pid)")
	})

	if st, err := s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Password != "yes" {
		t.Fatalf("before: %+v, %v", st, err)
	}
	if err := s.DisablePassword(ctx, id, security.PlanPassword(false).Disable.Command); err != nil {
		t.Fatal(err)
	}
	if st, err := s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Password != "no" || !st.Ours {
		t.Fatalf("after: %+v, %v", st, err)
	}
	if err := s.EnablePassword(ctx, id, security.PlanPassword(false).Enable.Command); err != nil {
		t.Fatal(err)
	}
	if st, err := s.ReadPasswordStatus(ctx, id, true, false); err != nil || st.Password != "yes" || st.Ours {
		t.Fatalf("after undo: %+v, %v", st, err)
	}
}
