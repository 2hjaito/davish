package packages

import (
	"context"
	"slices"
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
	ToolCertbot  = "certbot"
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
	// CanInstall tells whether the login user may run it: root, a member
	// of a group sudo usually trusts, or sudo without a password. Without
	// it the UI only says to ask whoever runs the server.
	CanInstall bool `json:"canInstall"`
	// Installed tells that the tool's program is already on the server.
	Installed bool `json:"installed"`
}

// tool is what installing one tool takes.
type tool struct {
	// bin is the program that tells the tool is installed.
	bin string
	// pkgs are the packages per manager.
	pkgs map[string][]string
	// extra are alternatives for one more package, per manager, tried in
	// turn; none being found does not stop the install.
	extra map[string][]string
	// unit is the service to enable and start; "" for none.
	unit string
	// init prepares the tool after install, per manager: the data
	// directory before the first start, or a timer the package leaves
	// off; most packages do it themselves.
	init map[string][]string
}

var tools = map[string]tool{
	ToolDocker: {
		bin:  "docker",
		pkgs: map[string][]string{Apt: {"docker.io"}, Dnf: {"moby-engine"}, Yum: {"docker"}, Apk: {"docker"}},
		// docker compose is a separate package. Ubuntu names it
		// docker-compose-v2; Debian 13 ships v2 as docker-compose.
		extra: map[string][]string{Apt: {"docker-compose-v2", "docker-compose"}, Dnf: {"docker-compose"}, Apk: {"docker-cli-compose"}},
		unit:  "docker",
	},
	ToolNginx: {
		bin:  "nginx",
		pkgs: map[string][]string{Apt: {"nginx"}, Dnf: {"nginx"}, Yum: {"nginx"}, Apk: {"nginx"}},
		unit: "nginx",
	},
	ToolGit: {
		bin:  "git",
		pkgs: map[string][]string{Apt: {"git"}, Dnf: {"git"}, Yum: {"git"}, Apk: {"git"}},
	},
	ToolMariaDB: {
		// Alpine has no mariadb-server; its init script sets up the data
		// directory on request.
		bin:  "mariadb",
		pkgs: map[string][]string{Apt: {"mariadb-server"}, Dnf: {"mariadb-server"}, Yum: {"mariadb-server"}, Apk: {"mariadb", "mariadb-client"}},
		unit: "mariadb",
		init: map[string][]string{Apk: {"rc-service", "mariadb", "setup"}},
	},
	ToolPostgres: {
		bin:  "psql",
		pkgs: map[string][]string{Apt: {"postgresql"}, Dnf: {"postgresql-server"}, Yum: {"postgresql-server"}, Apk: {"postgresql"}},
		unit: "postgresql",
		init: map[string][]string{Dnf: {"postgresql-setup", "--initdb"}, Yum: {"postgresql-setup", "--initdb"}},
	},
	ToolCertbot: {
		// Debian's package turns on renewal itself; Fedora's leaves its
		// timer off. Alpine has neither: renewal is the user's cron.
		bin:  "certbot",
		pkgs: map[string][]string{Apt: {"certbot", "python3-certbot-nginx"}, Dnf: {"certbot", "python3-certbot-nginx"}, Yum: {"certbot", "python3-certbot-nginx"}, Apk: {"certbot", "certbot-nginx"}},
		init: map[string][]string{Dnf: {"systemctl", "enable", "--now", "certbot-renew.timer"}, Yum: {"systemctl", "enable", "--now", "certbot-renew.timer"}},
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
	if alts := t.extra[manager]; len(alts) > 0 {
		var tries []string
		for _, p := range alts {
			line, err := PlanAction(manager, OpInstall, []string{p})
			if err != nil {
				return Setup{}, err
			}
			tries = append(tries, line)
		}
		steps = append(steps, "{ "+strings.Join(append(tries, "true"), " || ")+"; }")
	}
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

// privLine prints the user's id and groups, and sudo-ok when sudo needs
// no password. sudo -n never prompts, so it cannot hang.
const privLine = `id -u; id -nG; sudo -n true 2>/dev/null && echo sudo-ok; true`

// sudoGroups are the groups distributions give sudo rights by default.
var sudoGroups = []string{"sudo", "wheel", "admin"}

// parsePriv reads privLine's output.
func parsePriv(out []byte) bool {
	ls := lines(out)
	if len(ls) == 0 {
		return false
	}
	if strings.TrimSpace(ls[0]) == "0" || slices.Contains(ls, "sudo-ok") {
		return true
	}
	return len(ls) > 1 && slices.ContainsFunc(strings.Fields(ls[1]), func(g string) bool {
		return slices.Contains(sudoGroups, g)
	})
}

// Setup plans installing tool on the server, for its package manager, and
// tells whether it is there already and whether the login user may run
// the install.
func (s *Service) Setup(ctx context.Context, serverID, tool string) (Setup, error) {
	t, ok := tools[tool]
	if !ok {
		return Setup{}, apperr.New(apperr.InvalidInput, "unknown tool")
	}
	m, err := s.Detect(ctx, serverID)
	if err != nil {
		return Setup{}, err
	}
	plan, err := PlanSetup(m, tool)
	if err != nil {
		return Setup{}, err
	}
	// The installed mark comes last, after privLine's lines. dash's
	// command -v exits 127 for a missing program, which read takes for a
	// missing shell command: end with true.
	line := privLine + "; command -v " + shellx.Quote(t.bin) + " >/dev/null 2>&1 && echo " + installedMark + "; true"
	out, err := s.read(ctx, serverID, line)
	if err != nil {
		return Setup{}, err
	}
	plan.Installed = slices.Contains(lines(out), installedMark)
	plan.CanInstall = plan.Command != "" && parsePriv(out)
	return plan, nil
}

const installedMark = "davish-installed"
