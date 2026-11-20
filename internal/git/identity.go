package git

import (
	"context"
	_ "embed"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"davish/internal/shellx"
)

//go:embed identity.sh
var identityScript string

// Identity is who the SSH session is on the server, and what git and the
// code hosts know it as. Everything is read; nothing on the server changes.
type Identity struct {
	User string `json:"user"`
	Home string `json:"home"`
	Host string `json:"host"`
	// GitName and GitEmail are git's global user.name and user.email,
	// which commits made on the server are signed with.
	GitName  string `json:"gitName"`
	GitEmail string `json:"gitEmail"`
	// Helper is git's credential.helper, which keeps HTTPS tokens.
	Helper string   `json:"helper"`
	Keys   []SSHKey `json:"keys"`
	// Hosts are GitHub and GitLab as reached over SSH with the server's keys.
	Hosts []HostLogin `json:"hosts"`
}

// SSHKey is a public key in ~/.ssh.
type SSHKey struct {
	Type        string `json:"type"`
	Bits        int    `json:"bits"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
}

// Login states of a code host.
const (
	// LoginOK: a key is accepted; Account says as whom.
	LoginOK = "ok"
	// LoginDenied: the host answered but took no key.
	LoginDenied = "denied"
	// LoginUnreachable: no answer, or ssh is missing.
	LoginUnreachable = "unreachable"
)

// HostLogin is what a code host answered to ssh -T.
type HostLogin struct {
	Host  string `json:"host"`
	State string `json:"state"`
	// Account is the user, or owner/repo for a deploy key.
	Account string `json:"account"`
	// Deploy is set when the key is a deploy key of one repository.
	Deploy bool `json:"deploy"`
}

// identityHosts are the hosts asked, in order: those whose key is pinned.
var identityHosts = []string{"github.com", "gitlab.com"}

// Identity reads who the session is and which code hosts accept its keys.
func (s *Service) Identity(ctx context.Context, serverID string) (Identity, error) {
	args := []string{"-s", "--"}
	for _, h := range identityHosts {
		args = append(args, knownHosts[h])
	}
	res, err := s.run.RunInput(ctx, serverID, app, shellx.Cmd("sh", args...).String(), []byte(identityScript))
	if err != nil {
		return Identity{}, err
	}
	return parseIdentity(sections(string(res.Stdout))), nil
}

func parseIdentity(sec map[string]string) Identity {
	one := func(k string) string { return strings.TrimSpace(sec[k]) }
	id := Identity{
		User:     one("user"),
		Home:     one("home"),
		Host:     one("host"),
		GitName:  one("gitname"),
		GitEmail: one("gitemail"),
		Helper:   one("helper"),
		Keys:     parseKeys(sec["keys"]),
		Hosts:    []HostLogin{},
	}
	for _, h := range identityHosts {
		out, ok := sec["host:"+h]
		if !ok {
			continue
		}
		id.Hosts = append(id.Hosts, parseLogin(h, out))
	}
	return id
}

// keyLine is ssh-keygen -l: "256 SHA256:abc comment words (ED25519)".
var keyLine = regexp.MustCompile(`^(\d+) (\S+) ?(.*?) \(([A-Z0-9-]+)\)$`)

func parseKeys(out string) []SSHKey {
	keys := []SSHKey{}
	for line := range strings.SplitSeq(out, "\n") {
		m := keyLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		bits, _ := strconv.Atoi(m[1])
		comment := m[3]
		if comment == "no comment" {
			comment = ""
		}
		k := SSHKey{Type: m[4], Bits: bits, Fingerprint: m[2], Comment: comment}
		if !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	return keys
}

var (
	// GitHub: "Hi octocat! You've successfully authenticated..."; a deploy
	// key names its repository: "Hi octocat/app! ...".
	githubHi = regexp.MustCompile(`Hi ([A-Za-z0-9._/-]+)! You've successfully authenticated`)
	// GitLab: "Welcome to GitLab, @octocat!"
	gitlabHi = regexp.MustCompile(`Welcome to GitLab, @?([A-Za-z0-9._/-]+)!`)
)

func parseLogin(host, out string) HostLogin {
	l := HostLogin{Host: host, State: LoginUnreachable}
	for _, re := range []*regexp.Regexp{githubHi, gitlabHi} {
		if m := re.FindStringSubmatch(out); m != nil {
			l.State, l.Account = LoginOK, m[1]
			l.Deploy = strings.Contains(m[1], "/")
			return l
		}
	}
	if strings.Contains(out, "Permission denied") {
		l.State = LoginDenied
	}
	return l
}
