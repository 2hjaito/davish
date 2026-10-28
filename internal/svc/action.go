package svc

import (
	"context"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Actions on a unit.
const (
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionRestart = "restart"
)

// Plan is what the UI shows before acting on a unit.
type Plan struct {
	Unit   string `json:"unit"`
	Action string `json:"action"`
	// Command is exactly what Act runs.
	Command string `json:"command"`
	// Confirm is set for actions that interrupt a running service; the UI
	// must show Command and wait for the user (docs/SECURITY.md).
	Confirm bool `json:"confirm"`
	// Sudo is the same command with sudo, for the user to run in a
	// terminal when the server requires root.
	Sudo string `json:"sudo"`
}

// PlanAction describes action on unit. It does not touch the server: the
// command depends only on its arguments.
func PlanAction(unit, action string) (Plan, error) {
	if err := CheckUnit(unit); err != nil {
		return Plan{}, err
	}
	switch action {
	case ActionStart, ActionStop, ActionRestart:
	default:
		return Plan{}, apperr.New(apperr.InvalidInput, "unknown action")
	}
	cmd := shellx.Cmd("systemctl", action, unit)
	return Plan{
		Unit:    unit,
		Action:  action,
		Command: cmd.String(),
		Confirm: action != ActionStart,
		Sudo:    shellx.Cmd("sudo", cmd.Args()...).String(),
	}, nil
}

// Act runs action on unit. Without root, the server usually refuses with
// CMD_NEEDS_ROOT; the UI then offers Plan.Sudo in a terminal. A unit that
// fails to start gives CMD_FAILED, and its log tells why.
func (s *Service) Act(ctx context.Context, serverID, unit, action string) error {
	if _, err := PlanAction(unit, action); err != nil {
		return err
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("systemctl", action, unit))
	if err != nil {
		return err
	}
	return checkExit(res)
}
