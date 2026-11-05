// Package runmark builds the OSC 7761 progress marks a line typed into a
// terminal prints, which the terminal swallows and the frontend draws as
// progress (docs/adr/0012).
package runmark

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"davish/internal/shellx"
)

const (
	// Define defines the mark function, so the marks do not repeat the
	// printf on every step.
	Define = `__davish_mark() { printf '\033]7761;davish-deploy;%s\007' "$1"; }`
	// Undefine removes it after the line.
	Undefine = "unset -f __davish_mark"
	// ClearScreen leaves the run's own output on screen, not the long line
	// that was typed; a dialog showed the command before.
	ClearScreen = `printf '\033[H\033[2J'`
)

// Nonce returns the tag of one run's marks.
func Nonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("run nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Mark prints the literal event ev. The ; inside a mark must be quoted.
func Mark(nonce, ev string) string {
	return "__davish_mark " + shellx.Quote(nonce+";"+ev)
}

// Expand prints an event holding shell expansions such as $?, in double
// quotes; the nonce is hex, safe there.
func Expand(nonce, ev string) string {
	return `__davish_mark "` + nonce + ";" + ev + `"`
}
