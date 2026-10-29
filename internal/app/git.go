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

// GitBranches lists the branches a repository can switch to.
func (a *App) GitBranches(serverID, dir string) ([]git.Branch, error) {
	return a.git.Branches(a.ctx, serverID, dir)
}

// GitPlan is the exact command of fetch, pull or switch, to confirm.
func (a *App) GitPlan(serverID, dir, op, branch string) (git.Plan, error) {
	return a.git.Plan(a.ctx, serverID, dir, op, branch)
}

// GitRun runs a confirmed fetch, pull or switch.
func (a *App) GitRun(serverID, dir, op, branch, command string) (git.Result, error) {
	if err := a.lock.Check(); err != nil {
		return git.Result{}, err
	}
	return a.git.Run(a.ctx, serverID, dir, op, branch, command)
}

// GitPlanClone builds the clone line for a terminal; an empty folder only
// reads the URL.
func (a *App) GitPlanClone(spec git.CloneSpec) (git.ClonePlan, error) {
	return git.PlanClone(spec)
}

// GitInitCommand is the line that makes a folder a repository, with a
// remote when url is given.
func (a *App) GitInitCommand(dir, remote, url string) (string, error) {
	return git.InitCommand(dir, remote, url)
}

// GitRemoteCommand sets or adds a remote in a repository.
func (a *App) GitRemoteCommand(dir, remote, url string) (string, error) {
	return git.RemoteCommand(dir, remote, url)
}

// GitDeployLine joins a deploy recipe's steps into one line for a
// terminal, stopping at the first that fails.
func (a *App) GitDeployLine(dir string, steps []string) (string, error) {
	return git.DeployLine(dir, steps)
}
