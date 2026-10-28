// Package sshkeys manages SSH keys around a server: the keys allowed to
// log in to it (authorized_keys) and the server's own keys, such as a
// deploy key for git. Every change is a shell command the user confirms
// first (docs/SECURITY.md).
package sshkeys

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "keys"

// Conns hands out a server's SFTP client and the keys Davish logs in with.
type Conns interface {
	SFTP(serverID string) (*sftp.Client, error)
	LoginKeys(ctx context.Context, serverID string) ([]string, error)
}

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Service manages keys on each server.
type Service struct {
	conns Conns
	run   Runner
	now   func() time.Time
}

// New creates the service.
func New(conns Conns, run Runner) *Service {
	return &Service{conns: conns, run: run, now: time.Now}
}

// AuthorizedKey is one key allowed to log in.
type AuthorizedKey struct {
	Type        string   `json:"type"`
	Fingerprint string   `json:"fingerprint"`
	Comment     string   `json:"comment"`
	Options     []string `json:"options"`
	// Login is set for a key Davish itself may log in with: removing it
	// can lock Davish out of the server.
	Login bool `json:"login"`
	// Text is the whole line, which removing matches exactly.
	Text string `json:"text"`
}

// Authorized is the content of the login user's authorized_keys.
type Authorized struct {
	Path string          `json:"path"`
	Keys []AuthorizedKey `json:"keys"`
}

// maxKeysFile caps what is read; a real authorized_keys is a few KB.
const maxKeysFile = 1 << 20

// Authorized lists ~/.ssh/authorized_keys of the user Davish logs in as.
// A missing file is an empty list.
func (s *Service) Authorized(ctx context.Context, serverID string) (Authorized, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return Authorized{}, err
	}
	home, err := sc.Getwd()
	if err != nil {
		return Authorized{}, apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	p := path.Join(home, ".ssh", "authorized_keys")
	out := Authorized{Path: p, Keys: []AuthorizedKey{}}
	data, err := readSmall(sc, p)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return Authorized{}, apperr.Wrap(apperr.FSPermissionDenied, "cannot read authorized_keys", err)
	}
	login, err := s.conns.LoginKeys(ctx, serverID)
	if err != nil {
		return Authorized{}, err
	}
	out.Keys = parseAuthorized(data)
	for i := range out.Keys {
		out.Keys[i].Login = slices.Contains(login, out.Keys[i].Fingerprint)
	}
	return out, nil
}

// parseAuthorized reads the key lines, skipping comments and lines that
// are not keys.
func parseAuthorized(data []byte) []AuthorizedKey {
	keys := []AuthorizedKey{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), maxKeysFile)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if k, ok := parseLine(line); ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func parseLine(line string) (AuthorizedKey, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return AuthorizedKey{}, false
	}
	pub, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(t))
	if err != nil || len(bytes.TrimSpace(rest)) > 0 {
		return AuthorizedKey{}, false
	}
	if options == nil {
		options = []string{}
	}
	return AuthorizedKey{
		Type:        pub.Type(),
		Fingerprint: ssh.FingerprintSHA256(pub),
		Comment:     comment,
		Options:     options,
		Text:        line,
	}, true
}

// KeyPlan is a change to authorized_keys, for the confirmation.
type KeyPlan struct {
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
	// Login warns that Davish logs in with the key being removed.
	Login bool `json:"login"`
	// Backup is the copy kept before removing, named when planned.
	Backup  string `json:"backup,omitempty"`
	Command string `json:"command"`
}

// PlanAdd describes adding the public key line text, unless it is there.
func (s *Service) PlanAdd(ctx context.Context, serverID, text string) (KeyPlan, error) {
	k, ok := parseLine(strings.TrimSpace(text))
	if !ok || strings.ContainsAny(strings.TrimSpace(text), "\r\n") {
		return KeyPlan{}, apperr.New(apperr.InvalidInput, "not one public key line")
	}
	a, err := s.Authorized(ctx, serverID)
	if err != nil {
		return KeyPlan{}, err
	}
	if slices.ContainsFunc(a.Keys, func(x AuthorizedKey) bool { return x.Fingerprint == k.Fingerprint }) {
		return KeyPlan{}, apperr.New(apperr.FSExists, "the key is already allowed")
	}
	return KeyPlan{Fingerprint: k.Fingerprint, Comment: k.Comment, Command: addCommand(a.Path, k.Text)}, nil
}

// addCommand appends line to file with the folder and file modes sshd
// insists on, and does nothing if the exact line is already there.
func addCommand(file, line string) string {
	dir := path.Dir(file)
	q := shellx.Quote
	return shellx.Cmd("mkdir", "-p", "-m", "700", "--", dir).String() +
		" && " + shellx.Cmd("touch", "--", file).String() +
		" && " + shellx.Cmd("chmod", "600", "--", file).String() +
		" && { " + shellx.Cmd("grep", "-qxF", "--", line, file).String() +
		" || printf '%s\\n' " + q(line) + " >> " + q(file) + "; }"
}

// Add runs the confirmed command of PlanAdd.
func (s *Service) Add(ctx context.Context, serverID, text, command string) error {
	plan, err := s.PlanAdd(ctx, serverID, text)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "authorized_keys changed since the change was confirmed")
	}
	return s.shell(ctx, serverID, command)
}

// backupStamp is the UTC time in a backup's name.
const backupStamp = "20060102T150405Z"

// PlanRemove describes removing every line of the key with fingerprint,
// after copying the file aside.
func (s *Service) PlanRemove(ctx context.Context, serverID, fingerprint string) (KeyPlan, error) {
	return s.planRemove(ctx, serverID, fingerprint, s.now())
}

func (s *Service) planRemove(ctx context.Context, serverID, fingerprint string, stamp time.Time) (KeyPlan, error) {
	a, err := s.Authorized(ctx, serverID)
	if err != nil {
		return KeyPlan{}, err
	}
	var lines []string
	plan := KeyPlan{Fingerprint: fingerprint}
	for _, k := range a.Keys {
		if k.Fingerprint == fingerprint {
			lines = append(lines, k.Text)
			plan.Comment, plan.Login = k.Comment, plan.Login || k.Login
		}
	}
	if len(lines) == 0 {
		return KeyPlan{}, apperr.New(apperr.NotFound, "no such key in authorized_keys")
	}
	plan.Backup = a.Path + ".bak-" + stamp.UTC().Format(backupStamp)
	plan.Command = removeCommand(a.Path, plan.Backup, lines)
	return plan, nil
}

// removeCommand keeps a copy, writes the file without lines to a temporary
// file with sshd's mode, then renames it over, so a failure never leaves
// a half-written authorized_keys.
func removeCommand(file, backup string, lines []string) string {
	tmp := file + ".davish-tmp"
	q := shellx.Quote
	grep := []string{"-vxF"}
	for _, l := range lines {
		grep = append(grep, "-e", l)
	}
	grep = append(grep, "--", file)
	return shellx.Cmd("cp", "-p", "--", file, backup).String() +
		// grep exits 1 when no line is left, which is still a success here.
		" && { " + shellx.Cmd("grep", grep...).String() + " > " + q(tmp) + " || [ $? -eq 1 ]; }" +
		" && " + shellx.Cmd("chmod", "600", "--", tmp).String() +
		" && " + shellx.Cmd("mv", "--", tmp, file).String()
}

// Remove runs the confirmed plan of PlanRemove. The backup is named when
// the plan is shown, so the plan is rebuilt with that name and must still
// give the same command.
func (s *Service) Remove(ctx context.Context, serverID string, confirmed KeyPlan) error {
	i := strings.LastIndex(confirmed.Backup, ".bak-")
	if i < 0 {
		return apperr.New(apperr.InvalidInput, "not a backup name")
	}
	stamp, err := time.Parse(backupStamp, confirmed.Backup[i+len(".bak-"):])
	if err != nil {
		return apperr.New(apperr.InvalidInput, "not a backup name")
	}
	plan, err := s.planRemove(ctx, serverID, confirmed.Fingerprint, stamp)
	if err != nil {
		return err
	}
	if plan.Command != confirmed.Command {
		return apperr.New(apperr.FSStale, "authorized_keys changed since the change was confirmed")
	}
	return s.shell(ctx, serverID, plan.Command)
}

// shell runs a confirmed command and maps a failure to an error code.
func (s *Service) shell(ctx context.Context, serverID, command string) error {
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return err
	}
	if res.Exit == 0 {
		return nil
	}
	if res.Exit == 127 {
		return apperr.New(apperr.CmdMissing, "program not installed on the server")
	}
	msg := string(res.Stderr)
	if strings.Contains(msg, "Permission denied") || strings.Contains(msg, "Operation not permitted") {
		return apperr.New(apperr.FSPermissionDenied, "permission denied")
	}
	return apperr.New(apperr.CmdFailed, "command failed")
}

func readSmall(sc *sftp.Client, p string) ([]byte, error) {
	f, err := sc.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read only
	return io.ReadAll(io.LimitReader(f, maxKeysFile))
}
