package packages

import (
	"context"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Tools an app can offer to install when the server lacks them.
const (
	ToolDocker   = "docker"
	ToolNginx    = "nginx"
	ToolGit      = "git"
	ToolMariaDB  = "mariadb"
	ToolPostgres = "postgres"
)

// Setup is how to install a tool on a server. Like the other package
// commands it needs root, so the user runs it in a terminal.
type Setup struct {
	Tool    string `json:"tool"`
	Manager string `json:"manager"`
	// Command is the whole line to run; "" when the server's package
	// manager is not one Davish knows, so the user installs it themselves.
	Command string `json:"command"`
	// Relogin is set when the command adds the user to a group, which
	// only takes effect on a new SSH connection.
	Relogin bool `json:"relogin"`
}

// tool is what installing one tool takes.
type tool struct {
	// pkgs are the packages per manager.
	pkgs map[string][]string
	// unit is the service to enable and start; "" for none.
	unit string
	// init prepares the data directory before the first start, per
	// manager; most packages do it themselves.
	init map[string][]string
}

var tools = map[string]tool{
	ToolDocker: {
		pkgs: map[string][]string{Apt: {"docker.io"}, Dnf: {"moby-engine"}, Yum: {"docker"}, Apk: {"docker"}},
		unit: "docker",
	},
	ToolNginx: {
		pkgs: map[string][]string{Apt: {"nginx"}, Dnf: {"nginx"}, Yum: {"nginx"}, Apk: {"nginx"}},
		unit: "nginx",
	},
	ToolGit: {
		pkgs: map[string][]string{Apt: {"git"}, Dnf: {"git"}, Yum: {"git"}, Apk: {"git"}},
	},
	ToolMariaDB: {
		// Alpine has no mariadb-server; its init script sets up the data
		// directory on request.
		pkgs: map[string][]string{Apt: {"mariadb-server"}, Dnf: {"mariadb-server"}, Yum: {"mariadb-server"}, Apk: {"mariadb", "mariadb-client"}},
		unit: "mariadb",
		init: map[string][]string{Apk: {"rc-service", "mariadb", "setup"}},
	},
	ToolPostgres: {
		pkgs: map[string][]string{Apt: {"postgresql"}, Dnf: {"postgresql-server"}, Yum: {"postgresql-server"}, Apk: {"postgresql"}},
		unit: "postgresql",
		init: map[string][]string{Dnf: {"postgresql-setup", "--initdb"}, Yum: {"postgresql-setup", "--initdb"}},
	},
}

// PlanSetup is the command that installs tool with manager, then starts
// its service at boot and now. For Docker it also adds the login user to
// the docker group, so Davish can run docker without sudo; that group is
// as good as root, which the UI says.
func PlanSetup(manager, name string) (Setup, error) {
	t, ok := tools[name]
	if !ok {
		return Setup{}, apperr.New(apperr.InvalidInput, "unknown tool")
	}
	s := Setup{Tool: name, Manager: manager}
	pkgs, ok := t.pkgs[manager]
	if !ok {
		return s, nil
	}
	install, err := PlanAction(manager, OpInstall, pkgs)
	if err != nil {
		return Setup{}, err
	}
	sudo := func(args ...string) string { return shellx.Cmd("sudo", args...).String() }

	var steps []string
	switch manager {
	case Apt, Apk:
		// A fresh server may never have fetched the package lists.
		refresh, _ := PlanAction(manager, OpRefresh, nil)
		steps = append(steps, refresh)
	}
	steps = append(steps, install)
	if init, ok := t.init[manager]; ok {
		steps = append(steps, sudo(init...))
	}
	if t.unit != "" {
		if manager == Apk {
			steps = append(steps, sudo("rc-update", "add", t.unit), sudo("rc-service", t.unit, "start"))
		} else {
			steps = append(steps, sudo("systemctl", "enable", "--now", t.unit))
		}
	}
	if name == ToolDocker {
		// "$USER" is left for the server's shell to expand.
		if manager == Apk {
			steps = append(steps, sudo("addgroup")+` "$USER" docker`)
		} else {
			steps = append(steps, sudo("usermod", "-aG", "docker")+` "$USER"`)
		}
		s.Relogin = true
	}
	s.Command = strings.Join(steps, " && ")
	return s, nil
}

// Setup plans installing tool on the server, for its package manager.
func (s *Service) Setup(ctx context.Context, serverID, tool string) (Setup, error) {
	if _, ok := tools[tool]; !ok {
		return Setup{}, apperr.New(apperr.InvalidInput, "unknown tool")
	}
	m, err := s.Detect(ctx, serverID)
	if err != nil {
		return Setup{}, err
	}
	return PlanSetup(m, tool)
}
