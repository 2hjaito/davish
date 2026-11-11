package disk

import (
	"context"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// lineRunner answers every line with res, recording what ran.
type lineRunner struct {
	res audit.Result
	ran []string
}

func (r *lineRunner) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	r.ran = append(r.ran, line)
	return r.res, nil
}

type fakeRoot struct {
	res audit.Result
	err error
	ran []string
}

func (f *fakeRoot) Run(_ context.Context, _, _, line string, _ []byte) (audit.Result, error) {
	f.ran = append(f.ran, line)
	return f.res, f.err
}

func TestCleanRunsOnlyRootCleanupsAsRoot(t *testing.T) {
	ctx := context.Background()
	for id, want := range commands {
		run, root := &lineRunner{}, &fakeRoot{}
		s := New(run, nil)
		s.UseRoot(root)
		if err := s.Clean(ctx, "srv", id, want); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		asRoot := len(root.ran) == 1 && root.ran[0] == want && len(run.ran) == 0
		asUser := len(run.ran) == 1 && run.ran[0] == want && len(root.ran) == 0
		needs := id == "docker-logs" || id == "journal" || id == "logs" || id == "apt" ||
			id == "dnf" || id == "coredumps" || id == "tmp" || id == "snap"
		// The user's own caches stay the user's: as root, $HOME is root's.
		if needs && !asRoot || !needs && !asUser {
			t.Errorf("%s ran as user %q, as root %q", id, run.ran, root.ran)
		}
	}
}

func TestCleanPassesSudoErrorsOn(t *testing.T) {
	for _, code := range []apperr.Code{apperr.SudoPasswordRequired, apperr.CmdNeedsRoot} {
		s := New(&lineRunner{}, nil)
		s.UseRoot(&fakeRoot{err: apperr.New(code, "x")})
		if err := s.Clean(context.Background(), "srv", "journal", commands["journal"]); codeOf(err) != code {
			t.Errorf("%s: got %v", code, err)
		}
	}
}

// checkRunner refuses the truncate and answers the check with check.
type checkRunner struct{ check string }

func (r checkRunner) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	if strings.HasPrefix(line, "id -u; readlink") {
		return audit.Result{Stdout: []byte(r.check)}, nil
	}
	return audit.Result{Exit: 1, Stderr: []byte("truncate: cannot open for writing: Permission denied")}, nil
}

func TestTruncateAsRootAfterRefusal(t *testing.T) {
	ctx := context.Background()
	p := "/var/log/app/big.log"
	plan, _ := PlanTruncate(p)
	dirs := "0 755 directory\n0 755 directory\n0 755 directory\n0 755 directory\n"
	cases := map[string]struct {
		check string
		ok    bool
	}{
		"root's file":                {"1000\n" + p + "\n33 640 regular file\n" + dirs, true},
		"a link":                     {"1000\n/etc/shadow\n1001 777 symbolic link\n" + dirs, false},
		"really elsewhere":           {"1000\n/etc/x/big.log\n0 640 regular file\n" + dirs, false},
		"another user's folder":      {"1000\n" + p + "\n0 640 regular file\n1001 755 directory\n0 755 directory\n0 755 directory\n0 755 directory\n", false},
		"a log folder syslog writes": {"1000\n" + p + "\n104 640 regular file\n0 775 directory\n0 755 directory\n0 755 directory\n0 755 directory\n", true},
		"a shared folder":            {"1000\n" + p + "\n0 640 regular file\n0 1777 directory\n0 755 directory\n0 755 directory\n0 755 directory\n", false},
		"check failed":               {"1000\n", false},
	}
	for name, c := range cases {
		root := &fakeRoot{}
		s := New(checkRunner{c.check}, nil)
		s.UseRoot(root)
		err := s.Truncate(ctx, "srv", p, plan.Command)
		if c.ok && (err != nil || len(root.ran) != 1 || root.ran[0] != plan.Command) {
			t.Errorf("%s: %v, root ran %q", name, err, root.ran)
		}
		if !c.ok && (codeOf(err) != apperr.InvalidInput || len(root.ran) != 0) {
			t.Errorf("%s: %v, root ran %q", name, err, root.ran)
		}
	}
	// Disk's refusals come first: a system file is never emptied as root.
	root := &fakeRoot{}
	s := New(checkRunner{}, nil)
	s.UseRoot(root)
	if err := s.Truncate(ctx, "srv", "/etc/passwd", "truncate -s 0 -- /etc/passwd"); err == nil || len(root.ran) != 0 {
		t.Errorf("/etc/passwd: %v, root ran %q", err, root.ran)
	}
}

func TestMeasureFallsBackWhenLocked(t *testing.T) {
	run := &lineRunner{res: audit.Result{Stdout: []byte("as:user\n")}}
	s := New(run, nil)
	s.UseRoot(&fakeRoot{err: apperr.New(apperr.SudoPasswordRequired, "x")})
	o, err := s.Overview(context.Background(), "srv", "/", "")
	if err != nil {
		t.Fatal(err)
	}
	if !o.AskSudo || len(run.ran) != 1 || run.ran[0] != overviewLine("/") {
		t.Errorf("askSudo %v, ran %q", o.AskSudo, run.ran)
	}
}
