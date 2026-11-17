package apperr

import (
	"errors"
	"os"
	"strings"
)

// maxDetail bounds Detail so a runaway error cannot flood the UI or log.
const maxDetail = 2000

// Detail is the technical cause of err, for a "copy error" box the user
// pastes into a bug report: the wrapped cause of an *Error, or the whole
// text of an unclassified one. The home directory becomes ~ so a report
// does not carry the user's account name.
//
// Only call it on errors whose causes are library or system errors (SSH
// handshakes, key parsing, file access). Never on errors that may wrap
// command output or anything a secret could reach.
func Detail(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	var ae *Error
	if errors.As(err, &ae) {
		if ae.cause == nil {
			return ""
		}
		text = ae.cause.Error()
	}
	if home, herr := os.UserHomeDir(); herr == nil && len(home) > 1 {
		text = hideHome(text, home)
	}
	if len(text) > maxDetail {
		text = text[:maxDetail] + "…"
	}
	return text
}

// hideHome replaces home, with backslashes or slashes, with ~.
func hideHome(text, home string) string {
	for _, h := range []string{home, strings.ReplaceAll(home, `\`, "/")} {
		text = strings.ReplaceAll(text, h, "~")
	}
	return text
}
