package git

import (
	"bytes"
	"context"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// readOnly keeps git from rewriting the index while it only looks, and
// its messages in English so they can be told apart.
const readOnly = `export GIT_OPTIONAL_LOCKS=0 LC_ALL=C; `

const (
	// maxChanges caps the changed files listed, for a tree full of
	// untracked build output.
	maxChanges = 1000
	// LogPage is how many commits Log returns at most.
	LogPage = 100
	// maxDiff caps a diff, in bytes.
	maxDiff = 1 << 20
)

// Commit is one commit of the history.
type Commit struct {
	Hash    string `json:"hash"`
	Short   string `json:"short"`
	Author  string `json:"author"`
	Time    int64  `json:"time"` // unix seconds
	Subject string `json:"subject"`
}

// Change is a file that differs from the last commit. Index and Worktree
// are git's two status letters: "M", "A", "D", "R", "C", "T" or "." when
// unchanged on that side; "?" for an untracked file.
type Change struct {
	Path     string `json:"path"`
	OrigPath string `json:"origPath"` // set for a rename or copy
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Conflict bool   `json:"conflict"`
}

// Status is where a working tree stands.
type Status struct {
	// Branch is empty on a detached HEAD.
	Branch string `json:"branch"`
	// Upstream is the remote branch tracked, empty when none.
	Upstream string `json:"upstream"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
	// Last is nil in a repository without commits.
	Last      *Commit  `json:"last"`
	Changes   []Change `json:"changes"`
	Truncated bool     `json:"truncated"`
	// Install installs the Node.js dependencies as the lockfile at the
	// root pins them, such as "npm ci"; empty without package.json.
	Install string `json:"install"`
}

// checkDir accepts an absolute folder, without line breaks.
func checkDir(dir string) (string, error) {
	if !path.IsAbs(dir) || strings.ContainsAny(dir, "\n\r\x00") {
		return "", apperr.New(apperr.InvalidInput, "folder must be an absolute path")
	}
	return path.Clean(dir), nil
}

// gitIn renders "cd -- dir && git args..." for a command line.
func gitIn(dir string, args ...string) string {
	return shellx.InDir(dir, shellx.Cmd("git", args...))
}

// lockfiles are what the root may hold, in the order the first match
// picks the install command.
var lockfiles = []struct{ file, install string }{
	{"pnpm-lock.yaml", "pnpm install --frozen-lockfile"},
	{"yarn.lock", "yarn install --frozen-lockfile"},
	{"package-lock.json", "npm ci"},
	{"npm-shrinkwrap.json", "npm ci"},
}

// statusLine prints the last commit on one line, which is empty without
// commits, then the Node.js files at the root on one line, then the
// porcelain status with NUL separated entries, so file names need no
// unquoting.
func statusLine(dir string) string {
	files := []string{"package.json"}
	for _, l := range lockfiles {
		files = append(files, l.file)
	}
	return readOnly + shellx.Cmd("cd", "--", dir).String() + ` && { ` +
		`printf '%s\n' "$(` + shellx.Cmd("git", "log", "-1", "--format=%H%x09%h%x09%an%x09%at%x09%s").String() + ` 2>/dev/null)"; ` +
		`for f in ` + strings.Join(files, " ") + `; do [ -f "$f" ] && printf '%s ' "$f"; done; echo; ` +
		shellx.Cmd("git", "status", "--porcelain=v2", "--branch", "-z").String() + `; }`
}

// Status reads the branch, upstream, last commit and changed files of the
// repository at dir.
func (s *Service) Status(ctx context.Context, serverID, dir string) (Status, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return Status{}, err
	}
	res, err := s.run.RunLine(ctx, serverID, app, statusLine(dir))
	if err != nil {
		return Status{}, err
	}
	if err := gitError(res); err != nil {
		return Status{}, err
	}
	head, rest, _ := bytes.Cut(res.Stdout, []byte("\n"))
	node, rest, _ := bytes.Cut(rest, []byte("\n"))
	st := parseStatus(rest)
	st.Last = parseCommit(string(head), "\t")
	st.Install = installFor(strings.Fields(string(node)))
	return st, nil
}

// installFor picks the install command from the files at the root. A
// package.json without a lockfile gets none: npm ci needs one, and a
// plain install would rewrite what the repository pins.
func installFor(files []string) string {
	if !slices.Contains(files, "package.json") {
		return ""
	}
	for _, l := range lockfiles {
		if slices.Contains(files, l.file) {
			return l.install
		}
	}
	return ""
}

// gitError maps a failed git run to an error the UI can explain.
func gitError(res audit.Result) error {
	if res.Exit == 0 {
		return nil
	}
	msg := string(res.Stderr)
	switch {
	case res.Exit == 127:
		return apperr.New(apperr.CmdMissing, "git is not installed on the server")
	// Since 2.35.2 git refuses a repository another user owns, whose
	// config could run programs as the login user. Davish does not
	// override that with safe.directory.
	case strings.Contains(msg, "dubious ownership"):
		return apperr.New(apperr.GitUnsafeOwner, "repository is owned by another user")
	case strings.Contains(msg, "not a git repository"), strings.Contains(msg, "can't cd"),
		strings.Contains(msg, "No such file"):
		return apperr.New(apperr.NotFound, "not a git repository")
	case strings.Contains(msg, "Permission denied"):
		return apperr.New(apperr.FSPermissionDenied, "cannot read the repository")
	}
	return apperr.New(apperr.CmdFailed, "git failed")
}

func parseCommit(line, sep string) *Commit {
	f := strings.SplitN(strings.TrimRight(line, "\r\n"), sep, 5)
	if len(f) < 5 || f[0] == "" {
		return nil
	}
	t, _ := strconv.ParseInt(f[3], 10, 64)
	return &Commit{Hash: f[0], Short: f[1], Author: f[2], Time: t, Subject: f[4]}
}

// parseStatus reads git status --porcelain=v2 --branch -z.
func parseStatus(out []byte) Status {
	st := Status{Changes: []Change{}}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		switch {
		case strings.HasPrefix(e, "# branch.head "):
			if b := strings.TrimPrefix(e, "# branch.head "); b != "(detached)" {
				st.Branch = b
			}
		case strings.HasPrefix(e, "# branch.upstream "):
			st.Upstream = strings.TrimPrefix(e, "# branch.upstream ")
		case strings.HasPrefix(e, "# branch.ab "):
			f := strings.Fields(strings.TrimPrefix(e, "# branch.ab "))
			if len(f) == 2 {
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(f[0], "+"))
				st.Behind, _ = strconv.Atoi(strings.TrimPrefix(f[1], "-"))
			}
		case len(e) < 3 || e[0] == '#' || e[0] == '!':
		case len(st.Changes) == maxChanges:
			st.Truncated = true
		case e[0] == '?':
			st.Changes = append(st.Changes, Change{Path: e[2:], Index: "?", Worktree: "?"})
		case e[0] == '1' || e[0] == 'u':
			// "1 XY sub mH mI mW hH hI path"; "u" has three more fields.
			n := 9
			if e[0] == 'u' {
				n = 11
			}
			if f := strings.SplitN(e, " ", n); len(f) == n {
				st.Changes = append(st.Changes, Change{Path: f[n-1], Index: f[1][:1], Worktree: f[1][1:], Conflict: e[0] == 'u'})
			}
		case e[0] == '2':
			// "2 XY sub mH mI mW hH hI Xscore path", then the old path.
			f := strings.SplitN(e, " ", 10)
			if len(f) == 10 && i+1 < len(entries) {
				i++
				st.Changes = append(st.Changes, Change{Path: f[9], OrigPath: entries[i], Index: f[1][:1], Worktree: f[1][1:]})
			}
		}
	}
	return st
}

// logLine lists commits from HEAD, fields split by US and records by RS.
func logLine(dir string, skip int) string {
	return readOnly + gitIn(dir, "log", "--format=%H%x1f%h%x1f%an%x1f%at%x1f%s%x1e",
		"--max-count="+strconv.Itoa(LogPage), "--skip="+strconv.Itoa(skip))
}

// Log lists up to LogPage commits from HEAD, after skipping skip of them.
func (s *Service) Log(ctx context.Context, serverID, dir string, skip int) ([]Commit, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return nil, err
	}
	if skip < 0 {
		return nil, apperr.New(apperr.InvalidInput, "skip must not be negative")
	}
	res, err := s.run.RunLine(ctx, serverID, app, logLine(dir, skip))
	if err != nil {
		return nil, err
	}
	// A repository without commits has no history yet.
	if res.Exit != 0 && strings.Contains(string(res.Stderr), "does not have any commits") {
		return []Commit{}, nil
	}
	if err := gitError(res); err != nil {
		return nil, err
	}
	commits := []Commit{}
	for _, r := range strings.Split(string(res.Stdout), "\x1e") {
		if c := parseCommit(strings.TrimLeft(r, "\n"), "\x1f"); c != nil {
			commits = append(commits, *c)
		}
	}
	return commits, nil
}

// DiffRequest picks what to diff: a commit against its parent when Commit
// is set, otherwise File in the working tree against HEAD. Untracked
// files are diffed against nothing.
type DiffRequest struct {
	Dir       string `json:"dir"`
	Commit    string `json:"commit"`
	File      string `json:"file"`
	Untracked bool   `json:"untracked"`
}

// Diff is a unified diff, cut at maxDiff bytes.
type Diff struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}

var hash = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// diffArgs never run programs the repository configures: no external
// diff, no textconv filter.
func diffArgs(r DiffRequest) ([]string, error) {
	base := []string{"--no-color", "--no-ext-diff", "--no-textconv"}
	switch {
	case r.Commit != "":
		if !hash.MatchString(r.Commit) {
			return nil, apperr.New(apperr.InvalidInput, "not a commit hash")
		}
		return append(append([]string{"show", "--format="}, base...), r.Commit, "--"), nil
	case r.File == "" || strings.ContainsAny(r.File, "\n\r\x00") || path.IsAbs(r.File):
		return nil, apperr.New(apperr.InvalidInput, "file must be a path inside the repository")
	case r.Untracked:
		return append(append([]string{"diff", "--no-index"}, base...), "--", "/dev/null", r.File), nil
	}
	return append(append([]string{"diff"}, base...), "HEAD", "--", r.File), nil
}

// Diff shows the changes of a commit or of one changed file.
func (s *Service) Diff(ctx context.Context, serverID string, r DiffRequest) (Diff, error) {
	dir, err := checkDir(r.Dir)
	if err != nil {
		return Diff{}, err
	}
	args, err := diffArgs(r)
	if err != nil {
		return Diff{}, err
	}
	// One byte over the cap tells a cut diff from one exactly that long.
	line := readOnly + gitIn(dir, args...) + ` | head -c ` + strconv.Itoa(maxDiff+1)
	res, err := s.run.RunLine(ctx, serverID, app, line)
	if err != nil {
		return Diff{}, err
	}
	// head hides git's exit status, so failures show on stderr only.
	if bytes.Contains(res.Stderr, []byte("fatal:")) || res.Exit != 0 {
		if res.Exit == 0 {
			res.Exit = 128
		}
		return Diff{}, gitError(res)
	}
	out := res.Stdout
	d := Diff{Truncated: len(out) > maxDiff}
	if d.Truncated {
		out = out[:maxDiff]
	}
	d.Text = strings.ToValidUTF8(string(out), "�")
	return d, nil
}
