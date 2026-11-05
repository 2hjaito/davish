package git

import (
	"regexp"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/runmark"
	"davish/internal/shellx"
)

// Step is one deploy step: a command, and the words shown for it.
type Step struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

// Deploy is a recipe made into one terminal line.
type Deploy struct {
	// Line is typed into a terminal; Davish presses Enter only when Sudo
	// is false (docs/SECURITY.md).
	Line string `json:"line"`
	// Nonce tags the progress marks of this run (docs/adr/0012).
	Nonce string `json:"nonce"`
	// Steps are the labels of the steps that run, numbered from 1 in the marks.
	Steps []string `json:"steps"`
	// Sudo is set when a step runs sudo, which asks for a password.
	Sudo bool `json:"sudo"`
}

var sudoWord = regexp.MustCompile(`(^|[\s;&|(])sudo\s`)

// DeployLine runs steps one after another in dir, stopping at the first
// that fails, with a heading and progress marks around each.
func DeployLine(dir string, steps []Step) (Deploy, error) {
	nonce, err := runmark.Nonce()
	if err != nil {
		return Deploy{}, err
	}
	return deployLine(dir, nonce, steps)
}

func deployLine(dir, nonce string, steps []Step) (Deploy, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return Deploy{}, err
	}
	var kept []Step
	for _, s := range steps {
		s.Command = strings.TrimSpace(s.Command)
		s.Label = strings.TrimSpace(s.Label)
		if s.Command == "" || strings.HasPrefix(s.Command, "#") {
			continue
		}
		if strings.ContainsAny(s.Command+s.Label, "\n\r\x00") {
			return Deploy{}, apperr.New(apperr.InvalidInput, "a step is one line")
		}
		if s.Label == "" {
			s.Label = s.Command
		}
		kept = append(kept, s)
	}
	if len(kept) == 0 {
		return Deploy{}, apperr.New(apperr.InvalidInput, "the recipe has no steps")
	}
	n := strconv.Itoa(len(kept))
	mark := func(ev string) string { return runmark.Mark(nonce, ev) }
	head := shellx.Cmd("git", "-C", dir, "rev-parse", "--short", "HEAD").String()

	d := Deploy{Nonce: nonce}
	parts := []string{shellx.Cmd("cd", "--", dir).String(), mark("begin;" + n)}
	for i, s := range kept {
		at := strconv.Itoa(i + 1)
		heading := "▶ " + at + "/" + n + "  " + s.Label
		if s.Label != s.Command {
			heading += "  (" + s.Command + ")"
		}
		parts = append(parts, shellx.Cmd("echo", heading).String(), mark("step;"+at), "{ "+s.Command+"; }")
		d.Steps = append(d.Steps, s.Label)
		d.Sudo = d.Sudo || sudoWord.MatchString(s.Command)
	}
	parts = append(parts,
		runmark.Expand(nonce, `end;0;$(`+head+` 2>/dev/null)`),
		shellx.Cmd("echo", "✓ Deploy finished").String())
	failed := "{ " + runmark.Expand(nonce, "end;$?") + "; " +
		shellx.Cmd("echo", "✗ Deploy stopped: the step above failed").String() + "; }"
	d.Line = runmark.Define + "; " + runmark.ClearScreen + "; " + strings.Join(parts, " && ") + " || " + failed + "; " + runmark.Undefine
	return d, nil
}
