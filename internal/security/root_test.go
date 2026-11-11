package security

import (
	"context"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/sudo"
)

type lineRunner struct {
	out string
	ran []string
}

func (r *lineRunner) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	r.ran = append(r.ran, line)
	return audit.Result{Stdout: []byte(r.out)}, nil
}

type fakeRoot struct {
	state string
	out   string
	err   error
	ran   []string
}

func (f *fakeRoot) Run(_ context.Context, _, _, line string, _ []byte) (audit.Result, error) {
	f.ran = append(f.ran, line)
	return audit.Result{Stdout: []byte(f.out)}, f.err
}

func (f *fakeRoot) State(context.Context, string) (string, error) { return f.state, nil }

func TestApplyGoesThroughSudoKeeper(t *testing.T) {
	ctx := context.Background()
	run, root := &lineRunner{}, &fakeRoot{state: sudo.Ready}
	s := New(run)
	s.UseRoot(root)
	user, _ := PlanAction(Block, "203.0.113.7", false)
	asRoot, _ := PlanAction(Block, "203.0.113.7", true)
	if err := s.Apply(ctx, "srv", Block, "203.0.113.7", user.Command); err != nil {
		t.Fatal(err)
	}
	// The plain line, as root through the keeper; never sudo -n inside sudo.
	if len(root.ran) != 1 || root.ran[0] != asRoot.Command || len(run.ran) != 0 {
		t.Fatalf("root ran %q, user ran %q", root.ran, run.ran)
	}
	// A root login runs it as it is.
	if err := s.Apply(ctx, "srv", Block, "203.0.113.7", asRoot.Command); err != nil || len(run.ran) != 1 {
		t.Fatalf("as root: %v, %q", err, run.ran)
	}
	// Anything else is refused before it runs.
	if err := s.Apply(ctx, "srv", Block, "203.0.113.7", "ufw disable"); codeOf(err) != apperr.FSStale || len(root.ran) != 1 {
		t.Fatalf("forged: %v", err)
	}
	root.err = apperr.New(apperr.SudoPasswordRequired, "x")
	if err := s.Apply(ctx, "srv", Ban, "203.0.113.7", mustPlan(t, Ban, "203.0.113.7").Command); codeOf(err) != apperr.SudoPasswordRequired {
		t.Fatalf("locked: %v", err)
	}
}

func mustPlan(t *testing.T, action, ip string) Plan {
	t.Helper()
	p, err := PlanAction(action, ip, false)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

const userReport = "1000\n--sudo--\n--ufw--\nINSTALLED\nERROR: You need to be root to run this script\n--f2b--\n"

func TestReportReadsWhatNeedsRootAsRoot(t *testing.T) {
	ctx := context.Background()
	root := &fakeRoot{state: sudo.Ready, out: "--ufw--\nINSTALLED\nStatus: active\nAnywhere DENY 203.0.113.7\n--f2b--\n"}
	s := New(&lineRunner{out: userReport})
	s.UseRoot(root)
	r, err := s.Report(ctx, "srv")
	if err != nil {
		t.Fatal(err)
	}
	if !r.AsRoot || !r.CanSudo || r.AskSudo || r.Root || !r.UFW.Active || r.UFW.NeedsRoot {
		t.Fatalf("report = root %v asRoot %v canSudo %v ask %v ufw %+v", r.Root, r.AsRoot, r.CanSudo, r.AskSudo, r.UFW)
	}
	if len(root.ran) != 1 || root.ran[0] != rootReportLine {
		t.Errorf("root ran %q", root.ran)
	}
}

func TestReportOffersThePromptOnlyToSudoers(t *testing.T) {
	for state, ask := range map[string]bool{sudo.Password: true, sudo.Unlikely: false, sudo.None: false} {
		root := &fakeRoot{state: state}
		s := New(&lineRunner{out: userReport})
		s.UseRoot(root)
		r, err := s.Report(context.Background(), "srv")
		if err != nil {
			t.Fatal(err)
		}
		if r.AskSudo != ask || r.CanSudo != ask || len(root.ran) != 0 {
			t.Errorf("%s: ask %v canSudo %v ran %q", state, r.AskSudo, r.CanSudo, root.ran)
		}
		if ask && !strings.HasPrefix(r.SudoRead, "sudo sh -c ") {
			t.Errorf("%s: sudoRead %q", state, r.SudoRead)
		}
	}
}
