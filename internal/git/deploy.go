package git

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Deploy is a recipe made into one terminal line.
type Deploy struct {
	// Line is typed into a terminal for the user to read and run; it is
	// never run by Davish.
	Line string `json:"line"`
	// Nonce tags the progress marks of this run (docs/adr/0012).
	Nonce string `json:"nonce"`
	// Steps are the recipe's commands that run, numbered from 1 in the marks.
	Steps []string `json:"steps"`
}

// markFn prints an OSC 7761 sequence that the terminal swallows and the
// frontend reads as progress. Defined for the line and removed after it,
// so the marks do not repeat the printf on every step.
const markFn = `__davish_mark() { printf '\033]7761;davish-deploy;%s\007' "$1"; }`

// DeployLine runs steps one after another in dir, stopping at the first
// that fails, with a heading and progress marks around each. Steps are
// shell commands the user wrote in their deploy recipe.
func DeployLine(dir string, steps []string) (Deploy, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return Deploy{}, fmt.Errorf("deploy nonce: %w", err)
	}
	return deployLine(dir, hex.EncodeToString(b), steps)
}

func deployLine(dir, nonce string, steps []string) (Deploy, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return Deploy{}, err
	}
	var kept []string
	for _, s := range steps {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.ContainsAny(s, "\n\r\x00") {
			return Deploy{}, apperr.New(apperr.InvalidInput, "a step is one line")
		}
		kept = append(kept, s)
	}
	if len(kept) == 0 {
		return Deploy{}, apperr.New(apperr.InvalidInput, "the recipe has no steps")
	}
	n := strconv.Itoa(len(kept))
	// The ; inside a mark must be quoted. The end marks are in double
	// quotes for $? and the commit; the nonce is hex, safe there.
	mark := func(ev string) string { return "__davish_mark " + shellx.Quote(nonce+";"+ev) }
	head := shellx.Cmd("git", "-C", dir, "rev-parse", "--short", "HEAD").String()

	parts := []string{shellx.Cmd("cd", "--", dir).String(), mark("begin;" + n)}
	for i, s := range kept {
		at := strconv.Itoa(i + 1)
		parts = append(parts,
			shellx.Cmd("echo", "▶ "+at+"/"+n+"  "+s).String(),
			mark("step;"+at),
			"{ "+s+"; }")
	}
	parts = append(parts,
		`__davish_mark "`+nonce+`;end;0;$(`+head+` 2>/dev/null)"`,
		shellx.Cmd("echo", "✓ Deploy finished").String())
	failed := `{ __davish_mark "` + nonce + `;end;$?"; ` +
		shellx.Cmd("echo", "✗ Deploy stopped: the step above failed").String() + "; }"
	line := markFn + "; " + strings.Join(parts, " && ") + " || " + failed + "; unset -f __davish_mark"
	return Deploy{Line: line, Nonce: nonce, Steps: kept}, nil
}
