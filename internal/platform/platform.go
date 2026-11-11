// Package platform tells what kind of operating system a server runs, so
// each app can pick commands that exist there, and builds the command line
// that runs a PowerShell script on a Windows server.
package platform

import (
	"context"
	"encoding/base64"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

// OS is the operating system family of a server.
type OS string

// Platforms Detect can report.
const (
	Linux   OS = "linux"
	Windows OS = "windows"
	Darwin  OS = "darwin"
	Unknown OS = "unknown"
)

// DetectTimeout bounds the whole detection: a login shell that never
// returns must not keep the server from connecting.
const DetectTimeout = 5 * time.Second

// Probe is how Detect asks the server. Both functions may fail; a failure
// is an answer too (cmd.exe has no uname).
type Probe struct {
	// Uname returns the output of `uname -s`.
	Uname func(ctx context.Context) (string, error)
	// Home returns the SFTP path of the login directory, such as
	// /home/ann or /C:/Users/ann.
	Home func(ctx context.Context) (string, error)
}

// winPath matches the SFTP form of a Windows path, which Windows OpenSSH
// serves as /C:/Users/ann.
var winPath = regexp.MustCompile(`^/[A-Za-z]:([/\\]|$)`)

// Detect works out the server's OS. It does not depend on the login shell:
// uname works in every POSIX shell, and the SFTP path shape tells Windows
// apart when the shell is cmd.exe or PowerShell. Unknown means neither
// answered in time.
func Detect(ctx context.Context, p Probe) OS {
	ctx, cancel := context.WithTimeout(ctx, DetectTimeout)
	defer cancel()

	if p.Uname != nil {
		if out, err := p.Uname(ctx); err == nil {
			if os := fromUname(out); os != Unknown {
				return os
			}
		}
	}
	if ctx.Err() == nil && p.Home != nil {
		if home, err := p.Home(ctx); err == nil && winPath.MatchString(strings.TrimSpace(home)) {
			return Windows
		}
	}
	return Unknown
}

// fromUname maps `uname -s`. Git for Windows, MSYS2 and Cygwin report their
// own names, but the machine is still Windows: they have no real /proc.
func fromUname(out string) OS {
	s := strings.TrimSpace(out)
	switch {
	case s == "Linux":
		return Linux
	case s == "Darwin":
		return Darwin
	case strings.HasPrefix(s, "MINGW"), strings.HasPrefix(s, "MSYS"), strings.HasPrefix(s, "CYGWIN"):
		return Windows
	}
	return Unknown
}

// PowerShell returns the command line that runs script in Windows
// PowerShell 5.1 and the stdin to give it. The command is a tiny bootstrap
// that reads the script from stdin: it stays far below cmd.exe's 8191
// character limit, needs no quoting, and runs the same from cmd.exe,
// PowerShell or a POSIX default shell.
func PowerShell(script string) (command string, stdin string) {
	const bootstrap = `$s=[Console]::In.ReadToEnd();& ([scriptblock]::Create($s))`
	return "powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand " +
		encodeCommand(bootstrap), script
}

// encodeCommand is PowerShell's -EncodedCommand form: UTF-16LE in base64.
func encodeCommand(s string) string {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}
