package sshkeys

import (
	"context"
	"errors"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// ServerKey is a key pair in the server user's ~/.ssh, such as a deploy
// key git pulls with.
type ServerKey struct {
	Name string `json:"name"`
	// Path is the private key; Path + ".pub" the public one.
	Path        string `json:"path"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
	// Public is the line to paste into GitHub or GitLab.
	Public string `json:"public"`
}

// ServerKeys lists the key pairs in ~/.ssh: every *.pub that parses.
func (s *Service) ServerKeys(_ context.Context, serverID string) ([]ServerKey, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	home, err := sc.Getwd()
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	dir := path.Join(home, ".ssh")
	keys := []ServerKey{}
	entries, err := sc.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return keys, nil
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.FSPermissionDenied, "cannot list ~/.ssh", err)
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".pub")
		if !ok || e.IsDir() {
			continue
		}
		data, err := readSmall(sc, path.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		line := strings.TrimSpace(string(data))
		pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			continue
		}
		keys = append(keys, ServerKey{
			Name: name, Path: path.Join(dir, name), Type: pub.Type(),
			Fingerprint: ssh.FingerprintSHA256(pub), Comment: comment, Public: line,
		})
	}
	slices.SortFunc(keys, func(a, b ServerKey) int { return strings.Compare(a.Name, b.Name) })
	return keys, nil
}

// keyName is a file name in ~/.ssh that cannot be read as an option.
var keyName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// CreatePlan is what creating a server key runs, for the confirmation.
type CreatePlan struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Comment string `json:"comment"`
	Command string `json:"command"`
}

// PlanCreate describes making an ed25519 key pair ~/.ssh/name without a
// passphrase, as a deploy key needs to work unattended.
func (s *Service) PlanCreate(_ context.Context, serverID, name, comment string) (CreatePlan, error) {
	if !keyName.MatchString(name) || strings.HasSuffix(name, ".pub") || name == "authorized_keys" || name == "known_hosts" || name == "config" {
		return CreatePlan{}, apperr.New(apperr.InvalidInput, "not a key file name")
	}
	if len(comment) > 100 || strings.ContainsAny(comment, "\r\n\x00") {
		return CreatePlan{}, apperr.New(apperr.InvalidInput, "comment must be one short line")
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return CreatePlan{}, err
	}
	home, err := sc.Getwd()
	if err != nil {
		return CreatePlan{}, apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	dir := path.Join(home, ".ssh")
	p := path.Join(dir, name)
	for _, f := range []string{p, p + ".pub"} {
		if _, err := sc.Lstat(f); err == nil {
			return CreatePlan{}, apperr.New(apperr.FSExists, "a key with that name exists")
		}
	}
	cmd := shellx.Cmd("mkdir", "-p", "-m", "700", "--", dir).String() + " && " +
		shellx.Cmd("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", comment, "-f", p).String()
	return CreatePlan{Name: name, Path: p, Comment: comment, Command: cmd}, nil
}

// Create runs the confirmed command of PlanCreate.
func (s *Service) Create(ctx context.Context, serverID, name, comment, command string) error {
	plan, err := s.PlanCreate(ctx, serverID, name, comment)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "~/.ssh changed since the key was confirmed")
	}
	return s.shell(ctx, serverID, command)
}

// GitHosts are the git hosts a deploy key can be tested against, with the
// ed25519 host key fingerprints they publish, so the test never trusts
// whatever answers.
var GitHosts = map[string]string{
	"github.com": "SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU",
	"gitlab.com": "SHA256:eUXGGm1YGsMAS7vkcx6JOJdOGHPem5gQp4taiCfCLB8",
}

// GitTestPlan is the test command, shown before it runs.
type GitTestPlan struct {
	Host    string `json:"host"`
	Key     string `json:"key"`
	Command string `json:"command"`
}

// Outcomes of a git test.
const (
	GitOK          = "ok"          // the host knows the key
	GitDenied      = "denied"      // the host does not know the key
	GitHostKey     = "hostkey"     // the host's key is not the published one
	GitUnreachable = "unreachable" // no answer from the host
	GitOther       = "other"
)

// GitTestResult is how a test went; Account is who the host says the key
// belongs to, when it says.
type GitTestResult struct {
	Outcome string `json:"outcome"`
	Account string `json:"account,omitempty"`
	// Output is the host's own words, shown as text.
	Output string `json:"output"`
}

// PlanGitTest describes logging in to host as git with the server key
// name. The host key is fetched into a throwaway known_hosts and must
// match the published fingerprint, so the server's own known_hosts is
// never written.
func (s *Service) PlanGitTest(ctx context.Context, serverID, host, name string) (GitTestPlan, error) {
	if _, ok := GitHosts[host]; !ok {
		return GitTestPlan{}, apperr.New(apperr.InvalidInput, "unknown git host")
	}
	keys, err := s.ServerKeys(ctx, serverID)
	if err != nil {
		return GitTestPlan{}, err
	}
	i := slices.IndexFunc(keys, func(k ServerKey) bool { return k.Name == name })
	if i < 0 {
		return GitTestPlan{}, apperr.New(apperr.NotFound, "no such key in ~/.ssh")
	}
	return GitTestPlan{Host: host, Key: name, Command: gitTestCommand(host, keys[i].Path)}, nil
}

func gitTestCommand(host, key string) string {
	q := shellx.Quote
	return `f="$(mktemp)" && ` + shellx.Cmd("ssh-keyscan", "-t", "ed25519", host).String() + ` > "$f" 2>/dev/null; ` +
		`if [ ! -s "$f" ]; then echo davish:unreachable >&2; r=255; ` +
		`elif ssh-keygen -lf "$f" | grep -qF -- ` + q(GitHosts[host]) + `; then ` +
		shellx.Cmd("ssh", "-T", "-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes").String() +
		` -o UserKnownHostsFile="$f" ` + shellx.Cmd("-i", key, "git@"+host).String() + `; r=$?; ` +
		`else echo davish:hostkey >&2; r=255; fi; rm -f -- "$f"; exit $r`
}

// GitTest runs the confirmed command of PlanGitTest.
func (s *Service) GitTest(ctx context.Context, serverID, host, name, command string) (GitTestResult, error) {
	plan, err := s.PlanGitTest(ctx, serverID, host, name)
	if err != nil {
		return GitTestResult{}, err
	}
	if plan.Command != command {
		return GitTestResult{}, apperr.New(apperr.FSStale, "the key changed since the test was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return GitTestResult{}, err
	}
	return parseGitTest(string(res.Stdout) + string(res.Stderr)), nil
}

var (
	githubHi = regexp.MustCompile(`Hi ([^!\s]+)!`)
	gitlabHi = regexp.MustCompile(`Welcome to GitLab, @([^!\s]+)!`)
)

// parseGitTest reads what the host answered. GitHub and GitLab both end
// the session with a non-zero exit even when the key works, since they
// give no shell, so the words decide.
func parseGitTest(out string) GitTestResult {
	r := GitTestResult{Outcome: GitOther, Output: strings.TrimSpace(out)}
	switch {
	case strings.Contains(out, "davish:hostkey"):
		r.Outcome = GitHostKey
	case strings.Contains(out, "davish:unreachable"), strings.Contains(out, "Could not resolve hostname"),
		strings.Contains(out, "Connection timed out"), strings.Contains(out, "Network is unreachable"):
		r.Outcome = GitUnreachable
	case strings.Contains(out, "successfully authenticated"):
		r.Outcome = GitOK
		if m := githubHi.FindStringSubmatch(out); m != nil {
			r.Account = m[1]
		}
	case strings.Contains(out, "Welcome to GitLab"):
		r.Outcome = GitOK
		if m := gitlabHi.FindStringSubmatch(out); m != nil {
			r.Account = m[1]
		}
	case strings.Contains(out, "Permission denied (publickey)"):
		r.Outcome = GitDenied
	}
	return r
}
