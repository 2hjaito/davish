// Package git finds the git repositories on a server and reads their
// state. It runs the server's own git over SSH; nothing is installed.
package git

import (
	"bufio"
	"bytes"
	"context"
	"path"
	"slices"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "git"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Service reads repositories on each server.
type Service struct {
	run Runner
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run}
}

// Roots are where sites and apps usually live, looked into besides the
// login user's home.
var Roots = []string{"/var/www", "/srv", "/opt"}

const (
	// maxDepth is how deep under a root a .git may be: /var/www/a/b/c/.git.
	maxDepth = 4
	// maxRepos caps the list on a server holding a mirror of everything.
	maxRepos = 500
	// maxExtra caps the folders a user adds by hand.
	maxExtra = 32
)

// Repo is a working tree found on the server.
type Repo struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// findLine builds the search: every .git (a folder, or a file for a
// worktree or submodule) at most maxDepth below a root, not looking into
// node_modules, vendor or hidden folders such as ~/.nvm, which is itself a
// clone. -mindepth 1 keeps a root that is hidden, or a repo itself, from
// being pruned. Missing roots are fine, so errors are dropped.
func findLine(extra []string) string {
	roots := []string{`"$HOME"`}
	for _, r := range Roots {
		roots = append(roots, shellx.Quote(r))
	}
	for _, r := range extra {
		roots = append(roots, shellx.Quote(r))
	}
	return `timeout 15 find ` + strings.Join(roots, " ") +
		` -mindepth 1 -maxdepth ` + strconv.Itoa(maxDepth) +
		` \( -name .git -print -prune \) -o \( -name node_modules -o -name vendor -o -name '.?*' \) -prune` +
		` 2>/dev/null | head -n ` + strconv.Itoa(maxRepos) + `; true`
}

// checkExtra accepts absolute folders only, so a relative one cannot be
// read as a find option, and no line breaks, which would split the output.
func checkExtra(extra []string) ([]string, error) {
	if len(extra) > maxExtra {
		return nil, apperr.New(apperr.InvalidInput, "too many folders")
	}
	clean := make([]string, 0, len(extra))
	for _, d := range extra {
		if !path.IsAbs(d) || strings.ContainsAny(d, "\n\r\x00") {
			return nil, apperr.New(apperr.InvalidInput, "folder must be an absolute path")
		}
		clean = append(clean, path.Clean(d))
	}
	return clean, nil
}

// Find lists the repositories in the usual folders and in extra, the
// folders the user added, sorted by path.
func (s *Service) Find(ctx context.Context, serverID string, extra []string) ([]Repo, error) {
	extra, err := checkExtra(extra)
	if err != nil {
		return nil, err
	}
	res, err := s.run.RunLine(ctx, serverID, app, findLine(extra))
	if err != nil {
		return nil, err
	}
	return parseFind(res.Stdout), nil
}

func parseFind(out []byte) []Repo {
	repos := []Repo{}
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		dir, ok := strings.CutSuffix(strings.TrimRight(sc.Text(), "\r"), "/.git")
		// $HOME may sit under a root, or a user folder overlap one.
		if !ok || dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		repos = append(repos, Repo{Path: dir, Name: path.Base(dir)})
	}
	slices.SortFunc(repos, func(a, b Repo) int { return strings.Compare(a.Path, b.Path) })
	return repos
}
