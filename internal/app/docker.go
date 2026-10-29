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

// DockerImages lists the server's images and how many containers use each.
func (a *App) DockerImages(serverID string) ([]docker.Image, error) {
	return a.docker.Images(a.ctx, serverID)
}

// DockerImage reads an image's layers and config.
func (a *App) DockerImage(serverID, ref string) (docker.ImageDetail, error) {
	return a.docker.Image(a.ctx, serverID, ref)
}

// DockerPlanImage gives the remove and run commands for an image.
func (a *App) DockerPlanImage(ref string) (docker.ImagePlan, error) {
	return docker.PlanImage(ref)
}

// DockerRemoveImage runs docker rmi after the user confirmed command.
func (a *App) DockerRemoveImage(serverID, ref, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.docker.RemoveImage(a.ctx, serverID, ref, command)
}

// DockerVolumes lists the server's volumes with their size and use.
func (a *App) DockerVolumes(serverID string) ([]docker.Volume, error) {
	return a.docker.Volumes(a.ctx, serverID)
}

// DockerPlanVolume gives the command that removes a volume.
func (a *App) DockerPlanVolume(name string) (docker.VolumePlan, error) {
	return docker.PlanVolume(name)
}

// DockerRemoveVolume runs docker volume rm after the user confirmed command.
func (a *App) DockerRemoveVolume(serverID, name, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.docker.RemoveVolume(a.ctx, serverID, name, command)
}

// DockerPullCommand is the docker pull line for a terminal.
func (a *App) DockerPullCommand(ref string) (string, error) {
	return docker.PullCommand(ref)
}

// DockerPlanRun builds the docker run line for a new container, for the
// preview; it is typed into a terminal.
func (a *App) DockerPlanRun(spec docker.RunSpec) (docker.RunPlan, error) {
	return docker.PlanRun(spec)
}

// DockerComposeCommand is the docker compose line for an action on a
// project, for a terminal.
func (a *App) DockerComposeCommand(project, dir, file, action string) (string, error) {
	return docker.ComposeCommand(project, dir, file, action)
}

// DockerPlanStack describes writing a template as a new compose.yaml.
func (a *App) DockerPlanStack(dir, template string) (docker.StackPlan, error) {
	return docker.PlanStack(dir, template)
}

// DockerCreateStack runs the confirmed command of DockerPlanStack.
func (a *App) DockerCreateStack(serverID, dir, template, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.docker.CreateStack(a.ctx, serverID, dir, template, command)
}

// DockerCreate makes a network or a volume.
func (a *App) DockerCreate(serverID, kind, name string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.docker.CreateObject(a.ctx, serverID, kind, name)
}

// DockerRunCommand rebuilds the docker run line of a container.
func (a *App) DockerRunCommand(serverID, name string) (docker.RunPlan, error) {
	return a.docker.RunCommand(a.ctx, serverID, name)
}
