// Package shortcut opens what a desktop icon points at. It decides what
// kind of thing a path is, and builds the line that runs a script in a
// terminal, refusing a script that changed since the user saw it
// (docs/adr/0013).
package shortcut

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/fs"
	"davish/internal/runmark"
	"davish/internal/shellx"
)

// Kinds of shortcut.
const (
	KindFolder = "folder"
	KindScript = "script"
	KindFile   = "file"
)

// Files reads what a shortcut points at.
type Files interface {
	Stat(ctx context.Context, serverID, p string) (fs.Entry, error)
	Read(ctx context.Context, serverID, p string) (fs.File, error)
}

// Recorder is where runs are noted for the user to review.
type Recorder interface {
	Add(e audit.Entry) audit.Entry
}

// Target is what a path is right now.
type Target struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// Hash is the SHA-256 of a script; empty for anything else and for a
	// script too big to read, which then can never be trusted.
	Hash string `json:"hash"`
	// Command is what a script's run does, for the confirmation dialog.
	Command string `json:"command"`
	// perm decides how the script is invoked.
	perm uint32
}

// Run is a script's run made into one terminal line.
type Run struct {
	// Line is typed into a terminal. It holds no sudo of its own, so
	// Davish presses Enter; a sudo inside the script asks in the terminal.
	Line string `json:"line"`
	// Nonce tags the progress marks of this run.
	Nonce string `json:"nonce"`
}

// Service inspects and runs shortcuts.
type Service struct {
	files Files
	log   Recorder
}

// New returns a Service.
func New(files Files, log Recorder) *Service {
	return &Service{files: files, log: log}
}

// Inspect reports what p is now.
func (s *Service) Inspect(ctx context.Context, serverID, p string) (Target, error) {
	e, err := s.files.Stat(ctx, serverID, p)
	if err != nil {
		return Target{}, err
	}
	t := Target{Name: path.Base(p)}
	if e.Dir {
		t.Kind = KindFolder
		return t, nil
	}
	if e.Kind != fs.KindFile && e.Kind != fs.KindLink {
		return Target{}, apperr.New(apperr.InvalidInput, "not a file or folder")
	}
	perm := e.Perm
	f, err := s.files.Read(ctx, serverID, p)
	var ae *apperr.Error
	switch {
	case err == nil:
		// The stat of a link says nothing about the file it points to.
		perm = f.Perm
	case errors.As(err, &ae) && ae.Code == apperr.FSTooLarge:
	default:
		return Target{}, err
	}
	if !isScript(t.Name, perm) {
		t.Kind = KindFile
		return t, nil
	}
	t.Kind = KindScript
	t.Command = display(p, perm)
	t.perm = perm
	if err == nil {
		t.Hash = hash(f.Data)
	}
	return t, nil
}

// Run returns the line that runs the script at p, provided its content
// is still what the user saw. expected is the Hash of that Target, empty
// when the script was too big to have one.
func (s *Service) Run(ctx context.Context, serverID, p, expected string) (Run, error) {
	t, err := s.Inspect(ctx, serverID, p)
	if err != nil {
		return Run{}, err
	}
	if t.Kind != KindScript {
		return Run{}, apperr.New(apperr.InvalidInput, "not a script")
	}
	if t.Hash != expected {
		return Run{}, apperr.New(apperr.ShortcutChanged, "the script changed")
	}
	nonce, err := runmark.Nonce()
	if err != nil {
		return Run{}, err
	}
	// The exit code arrives as a progress mark, not here: the run goes
	// on in a terminal, so the entry says it has none.
	s.log.Add(audit.Entry{
		Time: time.Now().UnixMilli(), ServerID: serverID, App: "shortcut", Command: t.Command, Exit: -1,
	})
	return Run{Line: line(p, t.perm, nonce), Nonce: nonce}, nil
}

func isScript(name string, perm uint32) bool {
	if perm&0o111 != 0 {
		return true
	}
	ext := strings.ToLower(path.Ext(name))
	return ext == ".sh" || ext == ".bash"
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// display is the command as the user reads it. ./ keeps a name starting
// with - from being read as an option.
func display(p string, perm uint32) string {
	return cdTo(p) + " && " + invocation(p, perm)
}

func cdTo(p string) string {
	return shellx.Cmd("cd", "--", path.Dir(p)).String()
}

func invocation(p string, perm uint32) string {
	if perm&0o111 == 0 {
		return shellx.Cmd("bash", "./"+path.Base(p)).String()
	}
	return shellx.Cmd("./" + path.Base(p)).String()
}

// line wraps the run in progress marks as one step, so a failing cd reads
// as failing before that step.
func line(p string, perm uint32, nonce string) string {
	name := path.Base(p)
	ok := strings.Join([]string{
		cdTo(p),
		runmark.Mark(nonce, "begin;1"),
		shellx.Cmd("echo", "▶ "+name).String(),
		runmark.Mark(nonce, "step;1"),
		"{ " + invocation(p, perm) + "; }",
		runmark.Mark(nonce, "end;0"),
		shellx.Cmd("echo", "✓ "+name+" finished").String(),
	}, " && ")
	failed := "{ " + runmark.Expand(nonce, "end;$?") + "; " +
		shellx.Cmd("echo", "✗ "+name+" failed").String() + "; }"
	return runmark.Define + "; " + runmark.ClearScreen + "; " + ok + " || " + failed + "; " + runmark.Undefine
}
