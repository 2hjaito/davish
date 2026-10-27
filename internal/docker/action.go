package docker

import (
	"context"
	"regexp"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Actions on a container.
const (
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionRestart = "restart"
)

// Plan is what the UI shows before acting on a container.
type Plan struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	// Command is exactly what Act runs.
	Command string `json:"command"`
	// Confirm is set for stop and restart (docs/SECURITY.md).
	Confirm bool `json:"confirm"`
	// Sudo is the same command with sudo, for a terminal.
	Sudo string `json:"sudo"`
}

// containerName matches docker's rule for names; IDs match it too.
var containerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,254}$`)

// CheckName rejects anything that is not a container name or ID.
func CheckName(name string) error {
	if !containerName.MatchString(name) {
		return apperr.New(apperr.InvalidInput, "not a container name")
	}
	return nil
}

// PlanAction describes action on the container. The command depends only
// on its arguments, so nothing is asked of the server.
func PlanAction(name, action string) (Plan, error) {
	if err := CheckName(name); err != nil {
		return Plan{}, err
	}
	switch action {
	case ActionStart, ActionStop, ActionRestart:
	default:
		return Plan{}, apperr.New(apperr.InvalidInput, "unknown action")
	}
	cmd := shellx.Cmd("docker", action, name)
	return Plan{
		Name:    name,
		Action:  action,
		Command: cmd.String(),
		Confirm: action != ActionStart,
		Sudo:    shellx.Cmd("sudo", cmd.Args()...).String(),
	}, nil
}

// Act runs action on the container.
func (s *Service) Act(ctx context.Context, serverID, name, action string) error {
	if _, err := PlanAction(name, action); err != nil {
		return err
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("docker", action, name))
	if err != nil {
		return err
	}
	return checkExit(res)
}
