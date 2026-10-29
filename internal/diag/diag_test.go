package diag

import (
	"context"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// fakeRunner answers by the first word of the command; "hang" waits for
// the deadline like a stuck shell would.
type fakeRunner map[string]string

func (f fakeRunner) RunLine(ctx context.Context, _, _, line string) (audit.Result, error) {
	for prefix, answer := range f {
		if !strings.Contains(line, prefix) {
			continue
		}
		switch answer {
		case "hang":
			<-ctx.Done()
			return audit.Result{}, apperr.Wrap(apperr.CmdTimeout, "timeout", ctx.Err())
		case "down":
			return audit.Result{}, apperr.New(apperr.ConnNotConnected, "not connected")
		}
		return audit.Result{Stdout: []byte(answer)}, nil
	}
	return audit.Result{Stdout: []byte("0\n")}, nil
}

func TestVerdicts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cases := map[string]fakeRunner{
		"ok":            {"echo davish-ok": "davish-ok\n"},
		"no-connection": {"echo davish-ok": "down"},
		"docker-hangs":  {"echo davish-ok": "davish-ok\n", "timeout 8 docker": "124\n"},
		"ss-hangs":      {"echo davish-ok": "davish-ok\n", "timeout 8 ss": "124\n"},
	}
	for want, r := range cases {
		got := New(r, nil).Diagnose(ctx, "s")
		if got.Verdict != want {
			t.Errorf("%s: verdict = %s (%s)", want, got.Verdict, Summary(got))
		}
	}
}

func TestScanFindsWhatHangs(t *testing.T) {
	rc := "export PATH=$PATH:~/bin\n# tmux in a comment\nif [ -z \"$TMUX\" ]; then tmux attach || tmux new; fi\nexec zsh\n"
	got := scan("/home/u/.bashrc", strings.NewReader(rc))
	if len(got) != 2 || got[0].Line != 3 || got[1].Text != "exec zsh" {
		t.Fatalf("findings = %+v", got)
	}
	guarded := "case $- in *i*) ;; *) return;; esac\ntmux attach\n"
	if f := scan("x", strings.NewReader(guarded)); len(f) != 0 {
		t.Errorf("guarded rc flagged: %+v", f)
	}
}
