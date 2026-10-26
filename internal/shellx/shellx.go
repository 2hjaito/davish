// Package shellx builds shell command lines that are safe to send to a
// remote POSIX shell. Never build commands with fmt.Sprintf.
package shellx

import (
	"regexp"
	"strings"
)

// safe matches words that need no quoting in a POSIX shell.
var safe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// Quote returns s as a single shell word.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if safe.MatchString(s) {
		return s
	}
	// Inside single quotes nothing is special except the quote itself,
	// which is closed, escaped, and reopened.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Command is a program and its arguments, quoted only when rendered.
type Command struct {
	args []string
}

// Cmd creates a Command. Every argument, including name, is quoted.
func Cmd(name string, args ...string) Command {
	return Command{args: append([]string{name}, args...)}
}

// Args returns a copy of the raw, unquoted arguments.
func (c Command) Args() []string {
	return append([]string(nil), c.args...)
}

// String renders the command line for the remote shell.
func (c Command) String() string {
	quoted := make([]string, len(c.args))
	for i, a := range c.args {
		quoted[i] = Quote(a)
	}
	return strings.Join(quoted, " ")
}
