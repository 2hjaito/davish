package git

import (
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// DeployLine runs steps one after another in dir, stopping at the first
// that fails, with a heading before each. Steps are shell commands the
// user wrote in their deploy recipe; the line is typed into a terminal
// for them to read and run, never run by Davish.
func DeployLine(dir string, steps []string) (string, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return "", err
	}
	var kept []string
	for _, s := range steps {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.ContainsAny(s, "\n\r\x00") {
			return "", apperr.New(apperr.InvalidInput, "a step is one line")
		}
		kept = append(kept, s)
	}
	if len(kept) == 0 {
		return "", apperr.New(apperr.InvalidInput, "the recipe has no steps")
	}
	parts := []string{shellx.Cmd("cd", "--", dir).String()}
	for i, s := range kept {
		head := "▶ " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(kept)) + "  " + s
		parts = append(parts, shellx.Cmd("echo", head).String(), "{ "+s+"; }")
	}
	parts = append(parts, shellx.Cmd("echo", "✓ Deploy finished").String())
	return strings.Join(parts, " && ") + " || " + shellx.Cmd("echo", "✗ Deploy stopped: the step above failed").String(), nil
}
