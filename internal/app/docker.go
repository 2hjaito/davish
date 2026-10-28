package app

import "davish/internal/docker"

// DockerList returns every container, running or not.
func (a *App) DockerList(serverID string) ([]docker.Container, error) {
	return a.docker.List(a.ctx, serverID)
}

// DockerStats samples CPU and memory of running containers; it takes about
// two seconds.
func (a *App) DockerStats(serverID string) ([]docker.Stats, error) {
	return a.docker.Stats(a.ctx, serverID)
}

// DockerInspect returns the detail of one container.
func (a *App) DockerInspect(serverID, name string) (docker.Detail, error) {
	return a.docker.Inspect(a.ctx, serverID, name)
}

// DockerLogs returns a container's latest output.
func (a *App) DockerLogs(serverID, name string) (docker.Logs, error) {
	return a.docker.Logs(a.ctx, serverID, name)
}

// DockerPlan returns the exact command an action would run, and whether
// the UI must confirm it first.
func (a *App) DockerPlan(name, action string) (docker.Plan, error) {
	return docker.PlanAction(name, action)
}

// DockerAct starts, stops or restarts a container. Stop and restart must
// have been confirmed with the command from DockerPlan.
func (a *App) DockerAct(serverID, name, action string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.docker.Act(a.ctx, serverID, name, action)
}
