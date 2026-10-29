package docker

import (
	"net"
	"regexp"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// PullCommand is the docker pull line for ref, typed into a terminal so
// the progress shows there.
func PullCommand(ref string) (string, error) {
	if err := CheckImage(ref); err != nil {
		return "", err
	}
	return shellx.Cmd("docker", "pull", ref).String(), nil
}

// PortMap publishes a container port.
type PortMap struct {
	// Host is the address to bind; empty means 127.0.0.1.
	Host          string `json:"host"`
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	UDP           bool   `json:"udp"`
}

// Mount is a volume or a server folder mounted into the container.
type Mount struct {
	// Source is a volume name or an absolute path on the server.
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// RunSpec is the form of a new container.
type RunSpec struct {
	Name    string    `json:"name"`
	Image   string    `json:"image"`
	Ports   []PortMap `json:"ports"`
	Env     []string  `json:"env"` // KEY=value
	EnvFile string    `json:"envFile"`
	Mounts  []Mount   `json:"mounts"`
	// Restart is no, always, unless-stopped or on-failure.
	Restart string `json:"restart"`
	Network string `json:"network"`
	// Args follow the image, as the container's command.
	Args []string `json:"args"`
}

// RunPlan is the docker run line for a RunSpec.
type RunPlan struct {
	Command string `json:"command"`
	// Public lists ports bound to every address, reachable from the
	// Internet unless a firewall stops them.
	Public []string `json:"public"`
}

var (
	envKey     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	objectName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	absPath    = regexp.MustCompile(`^/[^\x00\n:]*$`)
)

// PlanRun builds the docker run line for spec. Ports bind to 127.0.0.1
// unless another address is given.
func PlanRun(spec RunSpec) (RunPlan, error) {
	bad := func(msg string) (RunPlan, error) { return RunPlan{}, apperr.New(apperr.InvalidInput, msg) }
	if err := CheckImage(spec.Image); err != nil {
		return RunPlan{}, err
	}
	args := []string{"docker", "run", "-d"}
	if spec.Name != "" {
		if !objectName.MatchString(spec.Name) {
			return bad("not a container name")
		}
		args = append(args, "--name", spec.Name)
	}
	switch spec.Restart {
	case "", "no":
	case "always", "unless-stopped", "on-failure":
		args = append(args, "--restart", spec.Restart)
	default:
		return bad("unknown restart policy")
	}
	if spec.Network != "" {
		if !objectName.MatchString(spec.Network) {
			return bad("not a network name")
		}
		args = append(args, "--network", spec.Network)
	}
	plan := RunPlan{Public: []string{}}
	for _, p := range spec.Ports {
		if p.ContainerPort < 1 || p.ContainerPort > 65535 || p.HostPort < 0 || p.HostPort > 65535 {
			return bad("not a port")
		}
		host := strings.TrimSpace(p.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return bad("not an address to bind")
		}
		hp := ""
		if p.HostPort > 0 {
			hp = strconv.Itoa(p.HostPort)
		}
		h := ip.String()
		if ip.To4() == nil {
			h = "[" + h + "]"
		}
		v := h + ":" + hp + ":" + strconv.Itoa(p.ContainerPort)
		if p.UDP {
			v += "/udp"
		}
		if ip.IsUnspecified() {
			plan.Public = append(plan.Public, v)
		}
		args = append(args, "-p", v)
	}
	for _, e := range spec.Env {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if !envKey.MatchString(e) || strings.ContainsAny(e, "\x00\n") {
			return bad("environment lines are KEY=value")
		}
		args = append(args, "-e", e)
	}
	if spec.EnvFile != "" {
		if !absPath.MatchString(spec.EnvFile) {
			return bad("the env file is an absolute path on the server")
		}
		args = append(args, "--env-file", spec.EnvFile)
	}
	for _, m := range spec.Mounts {
		if !absPath.MatchString(m.Source) && !objectName.MatchString(m.Source) || !absPath.MatchString(m.Target) {
			return bad("a mount is a volume or absolute path, to an absolute path")
		}
		v := m.Source + ":" + m.Target
		if m.ReadOnly {
			v += ":ro"
		}
		args = append(args, "-v", v)
	}
	args = append(args, spec.Image)
	for _, a := range spec.Args {
		if strings.ContainsAny(a, "\x00") {
			return bad("not an argument")
		}
	}
	args = append(args, spec.Args...)
	plan.Command = shellx.Cmd(args[0], args[1:]...).String()
	return plan, nil
}
