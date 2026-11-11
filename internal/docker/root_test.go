package docker

import (
	"context"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// plainRunner answers every command with res.
type plainRunner struct {
	res audit.Result
	ran []string
}

func (p *plainRunner) Run(_ context.Context, _, _ string, cmd shellx.Command) (audit.Result, error) {
	p.ran = append(p.ran, cmd.String())
	return p.res, nil
}

func (p *plainRunner) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	p.ran = append(p.ran, line)
	return p.res, nil
}

// fakeRoot answers as root with res or err.
type fakeRoot struct {
	res audit.Result
	err error
	ran []string
}

func (f *fakeRoot) Run(_ context.Context, _, _, line string, stdin []byte) (audit.Result, error) {
	if stdin != nil {
		panic("docker gives root no stdin")
	}
	f.ran = append(f.ran, line)
	return f.res, f.err
}

var denied = audit.Result{
	Exit:   1,
	Stderr: []byte("permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock"),
}

func TestExecAsRoot(t *testing.T) {
	ok := audit.Result{Stdout: []byte("{}\n")}
	cases := []struct {
		name     string
		plain    audit.Result
		root     *fakeRoot
		wantRoot bool
		wantCode apperr.Code
		wantOut  string
	}{
		{name: "allowed, root never asked", plain: ok, root: &fakeRoot{}, wantOut: "{}\n"},
		{name: "other failure stays plain", plain: audit.Result{Exit: 1, Stderr: []byte("No such container: x")}, root: &fakeRoot{}, wantCode: apperr.NotFound},
		{name: "denied, no root wired", plain: denied, wantCode: apperr.CmdNeedsRoot},
		{name: "denied, sudo runs it", plain: denied, root: &fakeRoot{res: ok}, wantRoot: true, wantOut: "{}\n"},
		{name: "denied, password needed", plain: denied, root: &fakeRoot{err: apperr.New(apperr.SudoPasswordRequired, "x")}, wantRoot: true, wantCode: apperr.SudoPasswordRequired},
		{name: "denied, no sudo either", plain: denied, root: &fakeRoot{err: apperr.New(apperr.CmdNeedsRoot, "x")}, wantRoot: true, wantCode: apperr.CmdNeedsRoot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			run := &plainRunner{res: c.plain}
			s := New(run)
			if c.root != nil {
				s.UseRoot(c.root)
			}
			cmd := shellx.Cmd("docker", "ps", "-a", "--format", "{{json .}}")
			res, err := s.exec(context.Background(), "srv", cmd)
			if err == nil {
				err = checkExit(res)
			}
			if got := codeOf(err); got != c.wantCode {
				t.Fatalf("code = %q, want %q (err %v)", got, c.wantCode, err)
			}
			if c.wantCode == "" && string(res.Stdout) != c.wantOut {
				t.Errorf("stdout = %q", res.Stdout)
			}
			if len(run.ran) != 1 || run.ran[0] != cmd.String() {
				t.Errorf("plain ran %q", run.ran)
			}
			ranRoot := c.root != nil && len(c.root.ran) > 0
			if ranRoot != c.wantRoot {
				t.Fatalf("root ran = %v, want %v", ranRoot, c.wantRoot)
			}
			// The same command, not another one.
			if ranRoot && c.root.ran[0] != cmd.String() {
				t.Errorf("root ran %q, want %q", c.root.ran[0], cmd.String())
			}
		})
	}
}

func TestActAsRoot(t *testing.T) {
	root := &fakeRoot{}
	s := New(&plainRunner{res: denied})
	s.UseRoot(root)
	if err := s.Act(context.Background(), "srv", "web", ActionRestart); err != nil {
		t.Fatal(err)
	}
	p, _ := PlanAction("web", ActionRestart)
	if len(root.ran) != 1 || root.ran[0] != p.Command {
		t.Errorf("root ran %q, want the confirmed %q", root.ran, p.Command)
	}
}
