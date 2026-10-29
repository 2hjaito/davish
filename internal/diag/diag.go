// Package diag finds out why a server does not answer Davish's commands,
// so a user can fix it, or send the findings to whoever can, without
// anyone having to log in to look.
package diag

import (
	"bufio"
	"context"
	"errors"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// app is the name commands are recorded under in the audit log.
const app = "diag"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// SFTPSource hands out a server's SFTP client.
type SFTPSource interface {
	SFTP(serverID string) (*sftp.Client, error)
}

// Step is one check, in the order they ran.
type Step struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Ms     int64  `json:"ms"`
}

// Finding is a line of a shell startup file that can keep commands from
// ever running: tmux, exec zsh, read…
type Finding struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Report is what Diagnose found; Text is a plain copy for support. Verdict names the likely cause:
// "ok", "no-connection", "shell-hangs", "shell-slow", "ss-hangs",
// "docker-hangs".
type Report struct {
	Verdict  string    `json:"verdict"`
	Steps    []Step    `json:"steps"`
	Findings []Finding `json:"findings,omitempty"`
	Text     string    `json:"text"`
}

// Service runs the checks.
type Service struct {
	run  Runner
	sftp SFTPSource
}

// New creates the service.
func New(run Runner, sftp SFTPSource) *Service {
	return &Service{run: run, sftp: sftp}
}

// Diagnose checks, in order: a bare command, then the tools the apps rely
// on that are known to hang. When the shell answers but slowly, or prints
// something other than asked, it reads the shell startup files for the
// usual culprits.
func (s *Service) Diagnose(ctx context.Context, serverID string) Report {
	r := s.diagnose(ctx, serverID)
	r.Text = Summary(r)
	return r
}

func (s *Service) diagnose(ctx context.Context, serverID string) Report {
	var r Report
	var firstErr error
	step := func(id string, limit time.Duration, line string) (audit.Result, bool) {
		c, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		start := time.Now()
		res, err := s.run.RunLine(c, serverID, app, line)
		st := Step{ID: id, OK: err == nil, Ms: time.Since(start).Milliseconds()}
		if err != nil {
			st.Detail = detail(err)
			if firstErr == nil {
				firstErr = err
			}
		}
		r.Steps = append(r.Steps, st)
		return res, err == nil
	}

	res, ok := step("echo", 15*time.Second, "echo davish-ok")
	if !ok {
		var ae *apperr.Error
		if errors.As(firstErr, &ae) && ae.Code == apperr.ConnNotConnected {
			r.Verdict = "no-connection"
			return r
		}
		// sshd starts sftp through the same shell, so the startup files
		// cannot be read either; the advice names where to look.
		r.Verdict = "shell-hangs"
		return r
	}
	if !strings.Contains(string(res.Stdout), "davish-ok") {
		r.Verdict = "shell-hangs"
		r.Steps[0].OK = false
		r.Steps[0].Detail = "echo printed something else: " + firstLine(res.Stdout)
		r.Findings = s.startupFiles(serverID)
		return r
	}
	if r.Steps[0].Ms > 5000 {
		r.Verdict = "shell-slow"
		r.Findings = s.startupFiles(serverID)
	}

	// The tools other apps run, each bounded on the server too; 124 is
	// timeout's "it did not finish".
	for _, t := range []struct{ id, verdict, line string }{
		{"ss", "ss-hangs", "command -v ss >/dev/null || exit 3; timeout 8 ss -tlnp >/dev/null 2>&1; echo $?"},
		{"docker", "docker-hangs", "command -v docker >/dev/null || exit 3; timeout 8 docker ps >/dev/null 2>&1; echo $?"},
	} {
		res, ok := step(t.id, 15*time.Second, t.line)
		code := strings.TrimSpace(string(res.Stdout))
		switch {
		case !ok:
		case res.Exit == 3:
			r.Steps[len(r.Steps)-1].Detail = "not installed"
		case code == "124":
			r.Steps[len(r.Steps)-1].OK = false
			r.Steps[len(r.Steps)-1].Detail = "did not finish in 8 seconds"
			if r.Verdict == "" {
				r.Verdict = t.verdict
			}
		case code != "0":
			r.Steps[len(r.Steps)-1].Detail = "exit " + code
		}
	}
	if r.Verdict == "" {
		r.Verdict = "ok"
	}
	return r
}

func detail(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return string(ae.Code) + ": " + ae.Message
	}
	return err.Error()
}

func firstLine(b []byte) string {
	line, _, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	if len(line) > 120 {
		line = line[:120]
	}
	return line
}

// startupFiles are read for every command sshd runs (bash reads .bashrc
// when started by sshd, even without a terminal), or at login.
var startupFiles = []string{".bashrc", ".bash_profile", ".profile", ".zshenv", ".zshrc", "/etc/bash.bashrc", "/etc/profile"}

// suspicious matches what keeps a non-interactive command from running:
// a terminal multiplexer, replacing the shell, waiting for input, a menu.
var suspicious = regexp.MustCompile(`^\s*[^#]*\b(tmux|screen|byobu|zellij|exec\s+\S*(zsh|fish|bash)|read\s|select\s|fish\b|zsh\b|neofetch|fastfetch|sudo\s)`)

func (s *Service) startupFiles(serverID string) []Finding {
	if s.sftp == nil {
		return nil
	}
	sc, err := s.sftp.SFTP(serverID)
	if err != nil {
		return nil
	}
	home, err := sc.Getwd()
	if err != nil {
		return nil
	}
	var out []Finding
	for _, name := range startupFiles {
		p := name
		if !path.IsAbs(p) {
			p = path.Join(home, name)
		}
		f, err := sc.Open(p)
		if err != nil {
			continue
		}
		out = append(out, scan(p, io.LimitReader(f, 256<<10))...)
		f.Close() //nolint:errcheck,gosec // read-only
	}
	return out
}

func scan(file string, r io.Reader) []Finding {
	var out []Finding
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		// An early return for non-interactive shells makes the rest harmless.
		if strings.Contains(line, "case $-") || strings.Contains(line, "[ -z \"$PS1\" ] && return") {
			return out
		}
		if suspicious.MatchString(line) {
			text := strings.TrimSpace(line)
			if len(text) > 160 {
				text = text[:160]
			}
			out = append(out, Finding{File: file, Line: n, Text: text})
			if len(out) >= 10 {
				return out
			}
		}
	}
	return out
}

// Summary is a plain-text copy of a report, for a support message.
func Summary(r Report) string {
	var b strings.Builder
	b.WriteString("verdict: " + r.Verdict + "\n")
	for _, st := range r.Steps {
		ok := "ok"
		if !st.OK {
			ok = "FAIL"
		}
		b.WriteString("  " + st.ID + ": " + ok + " (" + strconv.FormatInt(st.Ms, 10) + " ms)")
		if st.Detail != "" {
			b.WriteString(" " + st.Detail)
		}
		b.WriteString("\n")
	}
	for _, f := range r.Findings {
		b.WriteString("  " + f.File + ":" + strconv.Itoa(f.Line) + ": " + f.Text + "\n")
	}
	return b.String()
}
