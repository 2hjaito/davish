package packages

import (
	"context"
	"errors"
	"regexp"

	"davish/internal/apperr"
	"davish/internal/events"
	"davish/internal/shellx"
)

// Operations on packages. All of them need root, so they are typed into a
// terminal for the user to run (docs/SECURITY.md), never run by Davish.
const (
	OpInstall         = "install"
	OpRemove          = "remove"
	OpUpgrade         = "upgrade"          // the named packages
	OpUpgradeAll      = "upgrade-all"      // everything
	OpUpgradeSecurity = "upgrade-security" // the security fixes only
	OpRefresh         = "refresh"          // fetch the package lists
)

// packageName starts with a letter or digit, so it is never an option.
var packageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._:@~-]{0,127}$`)

// PlanAction is the sudo command for op on the named packages with manager.
// Install, remove and upgrade need names; the others take none, except
// upgrade-security on apt and apk, which have no such switch and upgrade
// the names given.
func PlanAction(manager, op string, names []string) (string, error) {
	for _, n := range names {
		if !packageName.MatchString(n) {
			return "", apperr.New(apperr.InvalidInput, "not a package name")
		}
	}
	needsNames := op == OpInstall || op == OpRemove || op == OpUpgrade ||
		(op == OpUpgradeSecurity && (manager == Apt || manager == Apk))
	if needsNames != (len(names) > 0) {
		return "", apperr.New(apperr.InvalidInput, "wrong packages for this action")
	}
	sudo := func(args ...string) string { return shellx.Cmd("sudo", args...).String() }
	with := func(args ...string) []string { return append(args, names...) }

	switch manager {
	case Apt:
		switch op {
		case OpInstall:
			return sudo(with("apt-get", "install")...), nil
		case OpRemove:
			return sudo(with("apt-get", "remove")...), nil
		case OpUpgrade, OpUpgradeSecurity:
			return sudo(with("apt-get", "install", "--only-upgrade")...), nil
		case OpUpgradeAll:
			return sudo("apt-get", "update") + " && " + sudo("apt-get", "upgrade"), nil
		case OpRefresh:
			return sudo("apt-get", "update"), nil
		}
	case Dnf, Yum:
		switch op {
		case OpInstall:
			return sudo(with(manager, "install")...), nil
		case OpRemove:
			return sudo(with(manager, "remove")...), nil
		case OpUpgrade:
			return sudo(with(manager, "upgrade")...), nil
		case OpUpgradeAll:
			return sudo(manager, "upgrade"), nil
		case OpUpgradeSecurity:
			return sudo(manager, "upgrade", "--security"), nil
		case OpRefresh:
			return sudo(manager, "makecache"), nil
		}
	case Apk:
		switch op {
		case OpInstall:
			return sudo(with("apk", "add")...), nil
		case OpRemove:
			return sudo(with("apk", "del")...), nil
		case OpUpgrade, OpUpgradeSecurity:
			return sudo(with("apk", "add", "-u")...), nil
		case OpUpgradeAll:
			return sudo("apk", "update") + " && " + sudo("apk", "upgrade"), nil
		case OpRefresh:
			return sudo("apk", "update"), nil
		}
	}
	return "", apperr.New(apperr.InvalidInput, "unknown package manager or action")
}

// Alert is the notify:new payload for pending security fixes; the
// frontend writes the text from kind, as for the other alerts.
type Alert struct {
	ID       string  `json:"id"`
	Level    string  `json:"level"`
	ServerID string  `json:"serverID"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"value"` // number of security fixes
}

// AlertSecurityUpdates is the kind of the alert.
const AlertSecurityUpdates = "security-updates"

// CheckSecurity reads the updates and notifies when security fixes are
// pending, once per count per app run. It is quiet on a server without a
// known package manager.
func (s *Service) CheckSecurity(ctx context.Context, serverID string) error {
	inv, err := s.Inventory(ctx, serverID)
	var ae *apperr.Error
	if errors.As(err, &ae) && ae.Code == apperr.CmdMissing {
		return nil
	}
	if err != nil {
		return err
	}
	n := 0
	for _, u := range inv.Updates {
		if u.Security {
			n++
		}
	}
	s.mu.Lock()
	same := s.notified[serverID] == n
	s.notified[serverID] = n
	s.mu.Unlock()
	if n == 0 || same {
		return nil
	}
	s.emit.Emit(events.NotifyNew, Alert{
		ID: serverID + ":security-updates", Level: "warn", ServerID: serverID,
		Kind: AlertSecurityUpdates, Value: float64(n),
	})
	return nil
}
