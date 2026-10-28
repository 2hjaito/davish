package app

import "davish/internal/git"

// GitFind lists the repositories in the usual folders and in extra.
func (a *App) GitFind(serverID string, extra []string) ([]git.Repo, error) {
	return a.git.Find(a.ctx, serverID, extra)
}

// GitStatus reads the branch, last commit and changed files of a repository.
func (a *App) GitStatus(serverID, dir string) (git.Status, error) {
	return a.git.Status(a.ctx, serverID, dir)
}

// GitLog lists a page of commits from HEAD.
func (a *App) GitLog(serverID, dir string, skip int) ([]git.Commit, error) {
	return a.git.Log(a.ctx, serverID, dir, skip)
}

// GitDiff shows the changes of a commit or of one changed file.
func (a *App) GitDiff(serverID string, req git.DiffRequest) (git.Diff, error) {
	return a.git.Diff(a.ctx, serverID, req)
}
