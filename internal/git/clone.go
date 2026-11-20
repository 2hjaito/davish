package git

import (
	"context"
	"path"
	"regexp"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// knownHosts are the SSH host keys GitHub and GitLab publish
// (docs.github.com "GitHub's SSH key fingerprints", docs.gitlab.com
// "SSH host keys fingerprints"). A clone over SSH adds the host's key to
// ~/.ssh/known_hosts first, so ssh never asks to trust an unknown key.
var knownHosts = map[string]string{
	"github.com": "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
	"gitlab.com": "gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf",
}

// CloneSpec is the clone form.
type CloneSpec struct {
	URL string `json:"url"`
	// Dir is the folder to clone into; it must not exist or be empty.
	Dir    string `json:"dir"`
	Branch string `json:"branch"`
}

// ClonePlan is the clone line, typed into a terminal so ssh or https can
// ask for what they need.
type ClonePlan struct {
	Command string `json:"command"`
	// SSH is set for an SSH URL, which needs a key the host accepts.
	SSH bool `json:"ssh"`
	// Host is the URL's host; Pinned tells that its published key is added.
	Host   string `json:"host"`
	Pinned bool   `json:"pinned"`
	// Name is the repository's name, for a default folder.
	Name string `json:"name"`
}

var (
	sshURL   = regexp.MustCompile(`^(?:ssh://)?(?:[A-Za-z0-9._-]+@)?([A-Za-z0-9.-]+)(?::\d+)?[:/]([A-Za-z0-9._/~-]+?)(?:\.git)?/?$`)
	httpsURL = regexp.MustCompile(`^https://(?:[A-Za-z0-9._%-]+@)?([A-Za-z0-9.-]+)(?::\d+)?/([A-Za-z0-9._/~-]+?)(?:\.git)?/?$`)
)

// parseURL returns a git URL's host and repository name.
func parseURL(u string) (host, name string, ssh bool, err error) {
	if m := httpsURL.FindStringSubmatch(u); m != nil {
		return strings.ToLower(m[1]), path.Base(m[2]), false, nil
	}
	if !strings.Contains(u, "://") || strings.HasPrefix(u, "ssh://") {
		if m := sshURL.FindStringSubmatch(u); m != nil && (strings.HasPrefix(u, "ssh://") || strings.Contains(u, "@")) {
			return strings.ToLower(m[1]), path.Base(m[2]), true, nil
		}
	}
	return "", "", false, apperr.New(apperr.InvalidInput, "not a git URL over SSH or HTTPS")
}

// remoteHTTPS is httpsURL that also takes user:token@ and http, as an
// existing remote may; only the host and path are kept from it.
var remoteHTTPS = regexp.MustCompile(`^https?://(?:[^/@\s]+@)?([A-Za-z0-9.-]+)(?::\d+)?/([A-Za-z0-9._/~-]+?)(?:\.git)?/?$`)

// remoteOwner returns a remote URL's host and owner: the user or org, or
// a GitLab group with its subgroups. Anything else gives empty strings.
func remoteOwner(u string) (host, owner string) {
	u = strings.TrimSpace(u)
	m := remoteHTTPS.FindStringSubmatch(u)
	if m == nil && (strings.HasPrefix(u, "ssh://") || (!strings.Contains(u, "://") && strings.Contains(u, "@"))) {
		m = sshURL.FindStringSubmatch(u)
	}
	if m == nil {
		return "", ""
	}
	dir := path.Dir(strings.Trim(m[2], "/"))
	if dir == "." || dir == "/" {
		return "", ""
	}
	return strings.ToLower(m[1]), dir
}

// PlanClone builds the clone line for spec. An empty Dir only parses the
// URL, to suggest a folder.
func PlanClone(spec CloneSpec) (ClonePlan, error) {
	return planClone(spec, false)
}

// planClone builds the clone line; quiet makes it one Davish runs itself,
// where git can ask nothing and a stuck remote is cut off.
func planClone(spec CloneSpec, quiet bool) (ClonePlan, error) {
	u := strings.TrimSpace(spec.URL)
	host, name, ssh, err := parseURL(u)
	if err != nil {
		return ClonePlan{}, err
	}
	p := ClonePlan{SSH: ssh, Host: host, Name: name}
	if strings.TrimSpace(spec.Dir) == "" {
		return p, nil
	}
	dir, err := checkDir(strings.TrimSpace(spec.Dir))
	if err != nil {
		return ClonePlan{}, err
	}
	args := []string{"clone"}
	if b := strings.TrimSpace(spec.Branch); b != "" {
		if !validBranch(b) {
			return ClonePlan{}, apperr.New(apperr.InvalidInput, "not a branch name")
		}
		args = append(args, "--branch", b)
	}
	args = append(args, "--", u, dir)
	clone := shellx.Cmd("git", args...).String()
	if quiet {
		clone = shellx.Cmd("env", append([]string{"GIT_TERMINAL_PROMPT=0", "timeout", "900", "git"}, args...)...).String()
	}
	if key, ok := knownHosts[host]; ssh && ok {
		p.Pinned = true
		kh := "~/.ssh/known_hosts"
		clone = "mkdir -p ~/.ssh && { " + shellx.Cmd("ssh-keygen", "-F", host).String() + " > /dev/null 2>&1 || " +
			shellx.Cmd("printf", "%s\n", key).String() + " >> " + kh + "; } && " + clone
	}
	p.Command = clone
	return p, nil
}

// Clone runs the clone of spec on the server, for a URL the server can
// reach without a prompt: an SSH key the host accepts, or a public HTTPS
// repository. A remote that wants a password fails with GIT_AUTH_FAILED,
// and the clone is then typed into a terminal instead. git refuses a
// folder that is not empty, so nothing on the server is overwritten.
func (s *Service) Clone(ctx context.Context, serverID string, spec CloneSpec) (Result, error) {
	if strings.TrimSpace(spec.Dir) == "" {
		return Result{}, apperr.New(apperr.InvalidInput, "folder must be an absolute path")
	}
	p, err := planClone(spec, true)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 16*time.Minute)
	defer cancel()
	res, err := s.run.RunLine(ctx, serverID, app, p.Command)
	if err != nil {
		return Result{}, err
	}
	out := string(res.Stdout) + string(res.Stderr)
	if len(out) > maxOutput {
		out = out[len(out)-maxOutput:]
	}
	r := Result{Output: redact(strings.TrimSpace(strings.ToValidUTF8(out, "\uFFFD")))}
	if res.Exit != 0 {
		r.Error = cloneError(res.Exit, r.Output)
	}
	return r, nil
}

func cloneError(exit int, out string) apperr.Code {
	switch {
	case strings.Contains(out, "already exists and is not an empty directory"):
		return apperr.FSExists
	case strings.Contains(out, "Permission denied") && !strings.Contains(out, "publickey"),
		strings.Contains(out, "could not create work tree"), strings.Contains(out, "could not create leading directories"):
		return apperr.FSPermissionDenied
	case strings.Contains(out, "Remote branch") && strings.Contains(out, "not found"):
		return apperr.NotFound
	}
	return opError(exit, out)
}

// remoteName accepts a plain remote name.
var remoteName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// InitCommand is the line that makes dir a repository, with remote set to
// url when one is given.
func InitCommand(dir, remote, url string) (string, error) {
	dir, err := checkDir(strings.TrimSpace(dir))
	if err != nil {
		return "", err
	}
	line := shellx.Cmd("mkdir", "-p", "--", dir).String() + " && " + gitIn(dir, "init")
	if url != "" {
		r, err := RemoteCommand(dir, remote, url)
		if err != nil {
			return "", err
		}
		line += " && " + r
	}
	return line, nil
}

// RemoteCommand sets remote to url in dir, adding it when missing.
func RemoteCommand(dir, remote, url string) (string, error) {
	dir, err := checkDir(strings.TrimSpace(dir))
	if err != nil {
		return "", err
	}
	if remote == "" {
		remote = "origin"
	}
	if !remoteName.MatchString(remote) {
		return "", apperr.New(apperr.InvalidInput, "not a remote name")
	}
	url = strings.TrimSpace(url)
	if _, _, _, err := parseURL(url); err != nil {
		return "", err
	}
	return "{ " + shellx.InDir(dir, shellx.Cmd("git", "remote", "set-url", remote, url)) + " 2>/dev/null || " +
		gitIn(dir, "remote", "add", remote, url) + "; }", nil
}
