package git

import (
	"path"
	"regexp"
	"strings"

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

// PlanClone builds the clone line for spec. An empty Dir only parses the
// URL, to suggest a folder.
func PlanClone(spec CloneSpec) (ClonePlan, error) {
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
	if key, ok := knownHosts[host]; ssh && ok {
		p.Pinned = true
		kh := "~/.ssh/known_hosts"
		clone = "mkdir -p ~/.ssh && { " + shellx.Cmd("ssh-keygen", "-F", host).String() + " > /dev/null 2>&1 || " +
			shellx.Cmd("printf", "%s\n", key).String() + " >> " + kh + "; } && " + clone
	}
	p.Command = clone
	return p, nil
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
