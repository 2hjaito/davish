package git

import (
	"context"
	"regexp"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Operations that change a working tree; each is confirmed first.
const (
	OpFetch  = "fetch"
	OpPull   = "pull"
	OpSwitch = "switch"
)

// Plan is the exact command an operation runs, for the user to confirm.
type Plan struct {
	Command string `json:"command"`
	// Dirty counts tracked files with uncommitted changes. A pull is
	// refused while it is not zero; a switch only warns, as git keeps the
	// changes or refuses itself when they would be lost.
	Dirty int `json:"dirty"`
}

// Branch is a local branch, or a remote one to check out as local.
type Branch struct {
	Name    string `json:"name"`
	Remote  bool   `json:"remote"`
	Current bool   `json:"current"`
}

// noPrompt keeps git from asking an https password nobody can type; ssh
// cannot ask either, as the session has no terminal. GIT_SSH_COMMAND is
// left alone, since core.sshCommand often picks the deploy key. timeout
// bounds a remote that hangs.
var noPrompt = []string{"env", "GIT_TERMINAL_PROMPT=0", "timeout", "120", "git"}

// branchName accepts what git check-ref-format --branch would, roughly,
// and never a leading dash, so a name cannot pass as an option.
var branchName = regexp.MustCompile(`^[A-Za-z0-9_.+@][A-Za-z0-9_./+@-]*$`)

func validBranch(b string) bool {
	return branchName.MatchString(b) && !strings.Contains(b, "..") && !strings.HasSuffix(b, ".lock") &&
		!strings.HasSuffix(b, "/") && !strings.Contains(b, "//") && !strings.Contains(b, "@{")
}

// command renders an operation inside dir.
func command(dir, op, branch string) (string, error) {
	args := append([]string{}, noPrompt[1:]...)
	switch op {
	case OpFetch:
		args = append(args, "fetch", "--prune")
	case OpPull:
		args = append(args, "pull", "--ff-only")
	case OpSwitch:
		if !validBranch(branch) {
			return "", apperr.New(apperr.InvalidInput, "not a branch name")
		}
		// Switching is local; no remote is asked, so no env is needed.
		return gitIn(dir, "switch", branch), nil
	default:
		return "", apperr.New(apperr.InvalidInput, "unknown git operation")
	}
	return shellx.InDir(dir, shellx.Cmd(noPrompt[0], args...)), nil
}

func dirty(st Status) int {
	n := 0
	for _, c := range st.Changes {
		if c.Index != "?" {
			n++
		}
	}
	return n
}

// Plan builds the command for op on the repository at dir.
func (s *Service) Plan(ctx context.Context, serverID, dir, op, branch string) (Plan, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return Plan{}, err
	}
	cmd, err := command(dir, op, branch)
	if err != nil {
		return Plan{}, err
	}
	st, err := s.Status(ctx, serverID, dir)
	if err != nil {
		return Plan{}, err
	}
	return Plan{Command: cmd, Dirty: dirty(st)}, nil
}

// Result is what git printed, both streams, with the reason it failed:
// an apperr code, empty on success. The output says which file or remote
// is at fault, which the user needs to act on a failure.
type Result struct {
	Output string      `json:"output"`
	Error  apperr.Code `json:"error"`
}

// maxOutput caps what is kept of git's output.
const maxOutput = 16 << 10

// Run runs a confirmed operation. The plan is built again so a command
// confirmed for another state never runs, and a pull is refused on a
// dirty tree even if the tree became dirty after the confirmation.
func (s *Service) Run(ctx context.Context, serverID, dir, op, branch, confirmed string) (Result, error) {
	plan, err := s.Plan(ctx, serverID, dir, op, branch)
	if err != nil {
		return Result{}, err
	}
	if plan.Command != confirmed {
		return Result{}, apperr.New(apperr.FSStale, "the command changed since it was confirmed")
	}
	if op == OpPull && plan.Dirty > 0 {
		return Result{}, apperr.New(apperr.GitDirty, "uncommitted changes")
	}
	res, err := s.run.RunLine(ctx, serverID, app, plan.Command)
	if err != nil {
		return Result{}, err
	}
	out := string(res.Stdout) + string(res.Stderr)
	if len(out) > maxOutput {
		out = out[len(out)-maxOutput:]
	}
	r := Result{Output: redact(strings.TrimSpace(strings.ToValidUTF8(out, "\uFFFD")))}
	if res.Exit != 0 {
		r.Error = opError(res.Exit, r.Output)
	}
	return r, nil
}

// opError tells why fetch, pull or switch failed.
func opError(exit int, out string) apperr.Code {
	switch {
	case exit == 124:
		return apperr.GitRemoteFailed
	case strings.Contains(out, "Permission denied (publickey"), strings.Contains(out, "Host key verification failed"),
		strings.Contains(out, "Authentication failed"), strings.Contains(out, "terminal prompts disabled"),
		strings.Contains(out, "could not read Username"):
		return apperr.GitAuthFailed
	case strings.Contains(out, "Not possible to fast-forward"), strings.Contains(out, "Diverging branches"),
		strings.Contains(out, "not possible because you have unmerged"):
		return apperr.GitNotFastForward
	case strings.Contains(out, "no tracking information"):
		return apperr.GitNoUpstream
	case strings.Contains(out, "would be overwritten"), strings.Contains(out, "Please commit your changes"):
		return apperr.GitDirty
	case strings.Contains(out, "dubious ownership"):
		return apperr.GitUnsafeOwner
	case strings.Contains(out, "Could not resolve host"), strings.Contains(out, "Connection refused"),
		strings.Contains(out, "Could not read from remote"), strings.Contains(out, "unable to access"):
		return apperr.GitRemoteFailed
	}
	return apperr.CmdFailed
}

// credentials matches the user:token part of an https remote URL.
var credentials = regexp.MustCompile(`(://)[^/@\s]+@`)

// redact hides credentials a remote URL may carry, before the output
// reaches the UI or an error message.
func redact(out string) string {
	return credentials.ReplaceAllString(out, "${1}***@")
}

// branchesLine lists local then remote-tracking branches, marking HEAD.
func branchesLine(dir string) string {
	return readOnly + gitIn(dir, "for-each-ref", "--format=%(HEAD)%09%(refname)", "refs/heads", "refs/remotes")
}

// Branches lists the branches to switch to: local ones, then remote ones
// with no local branch of the same name, which git switch creates.
func (s *Service) Branches(ctx context.Context, serverID, dir string) ([]Branch, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return nil, err
	}
	res, err := s.run.RunLine(ctx, serverID, app, branchesLine(dir))
	if err != nil {
		return nil, err
	}
	if err := gitError(res); err != nil {
		return nil, err
	}
	return parseBranches(string(res.Stdout)), nil
}

func parseBranches(out string) []Branch {
	branches := []Branch{}
	local := map[string]bool{}
	var remote []string
	for _, l := range strings.Split(out, "\n") {
		head, ref, ok := strings.Cut(strings.TrimRight(l, "\r"), "\t")
		if !ok {
			continue
		}
		if name, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			local[name] = true
			branches = append(branches, Branch{Name: name, Current: head == "*"})
			continue
		}
		// refs/remotes/origin/feature → feature; origin/HEAD is an alias.
		if rest, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
			if _, name, ok := strings.Cut(rest, "/"); ok && name != "HEAD" {
				remote = append(remote, name)
			}
		}
	}
	seen := map[string]bool{}
	for _, name := range remote {
		if !local[name] && !seen[name] {
			seen[name] = true
			branches = append(branches, Branch{Name: name, Remote: true})
		}
	}
	return branches
}
