package svc

import (
	"context"
	"path"
	"regexp"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// How a unit is removed, chosen from where its file lives.
const (
	// RemoveDelete is for a unit an admin put in /etc/systemd/system:
	// stop and disable it, forget its failure, delete the file, reload.
	RemoveDelete = "delete"
	// RemoveMask is for a unit a package installed: its file belongs to
	// the package, so it is masked, with what triggers it, and never runs
	// again; systemctl unmask undoes it.
	RemoveMask = "mask"
	// RemoveReset is for a unit whose file is gone or already masked:
	// only its failed state is left to clear.
	RemoveReset = "reset"
)

// triggerName is a unit that can start a service: a timer, socket, path
// or another service, named as CheckUnit requires.
var triggerName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_.@\\-]*\.(timer|socket|path|service)$`)

// adminDir holds the units an admin wrote, as opposed to a package's.
const adminDir = "/etc/systemd/system"

// RemovePlan is what removing a unit runs, for the confirmation.
type RemovePlan struct {
	Unit string `json:"unit"`
	Mode string `json:"mode"`
	// File is the unit file deleted in RemoveDelete mode.
	File string `json:"file,omitempty"`
	// Triggers are the timers or sockets stopped along with the unit.
	Triggers []string `json:"triggers"`
	// Command is exactly what Remove runs.
	Command string `json:"command"`
	// Sudo is the same steps as root, for a terminal.
	Sudo string `json:"sudo"`
}

// PlanRemove reads unit's file and triggers and describes removing it.
func (s *Service) PlanRemove(ctx context.Context, serverID, unit string) (RemovePlan, error) {
	st, err := s.Status(ctx, serverID, unit)
	if err != nil {
		return RemovePlan{}, err
	}
	return planRemove(st)
}

func planRemove(st Status) (RemovePlan, error) {
	if err := CheckUnit(st.Unit); err != nil {
		return RemovePlan{}, err
	}
	var triggers []string
	for _, t := range strings.Fields(st.TriggeredBy) {
		if triggerName.MatchString(t) {
			triggers = append(triggers, t)
		}
	}
	units := append([]string{st.Unit}, triggers...)
	resetFailed := shellx.Cmd("systemctl", "reset-failed", st.Unit)

	plan := RemovePlan{Unit: st.Unit, Triggers: triggers}
	var steps []shellx.Command
	switch {
	case st.Load != "loaded" || st.File == "" || st.File == "/dev/null" || st.Enabled == "masked":
		plan.Mode = RemoveReset
		steps = []shellx.Command{resetFailed}
	case path.Dir(st.File) == adminDir && path.Clean(st.File) == st.File:
		plan.Mode, plan.File = RemoveDelete, st.File
		steps = []shellx.Command{
			shellx.Cmd("systemctl", append([]string{"disable", "--now"}, units...)...),
			resetFailed,
			shellx.Cmd("rm", "--", st.File),
			shellx.Cmd("systemctl", "daemon-reload"),
		}
	default:
		plan.Mode = RemoveMask
		steps = []shellx.Command{
			shellx.Cmd("systemctl", append([]string{"mask", "--now"}, units...)...),
			resetFailed,
		}
	}
	plan.Command, plan.Sudo = chain(steps, false), chain(steps, true)
	return plan, nil
}

// chain joins steps with &&, each run as root when sudo is set.
func chain(steps []shellx.Command, sudo bool) string {
	parts := make([]string, len(steps))
	for i, c := range steps {
		if sudo {
			c = shellx.Cmd("sudo", c.Args()...)
		}
		parts[i] = c.String()
	}
	return strings.Join(parts, " && ")
}

// Remove runs command, which must be what PlanRemove gives now: if the
// unit changed since the user confirmed, it fails with FS_STALE and the
// user is asked again.
func (s *Service) Remove(ctx context.Context, serverID, unit, command string) error {
	plan, err := s.PlanRemove(ctx, serverID, unit)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "the unit changed since the removal was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return err
	}
	return checkExit(res)
}
