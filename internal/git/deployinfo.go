package git

import (
	"context"
	_ "embed"
	"encoding/json"
	"path"
	"slices"
	"sort"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

//go:embed deployinfo.sh
var deployInfoScript string

// Option is a deploy step Davish can offer, with its command ready to run
// in the repository folder.
type Option struct {
	// Name is what the user picks: a script name, a pm2 process, a unit.
	Name    string `json:"name"`
	Command string `json:"command"`
	// Sudo is set when the command asks for a password in the terminal.
	Sudo bool `json:"sudo"`
}

// AppRunner is something already running the app, found on the server.
type AppRunner struct {
	Option
	// Kind is "pm2", "systemd" or "compose".
	Kind string `json:"kind"`
}

// DeployInfo is what a deploy of a repository can be made of.
type DeployInfo struct {
	Pull Option `json:"pull"`
	// Install is nil without a lockfile to install from.
	Install *Option `json:"install"`
	// Builds are the package.json scripts, go build and make; "build"
	// first when there is one.
	Builds []Option `json:"builds"`
	// Runners run the app now: pm2 processes or systemd units started in
	// the folder, or a compose project of it. Compose is offered when the
	// folder has a compose file even if nothing runs yet.
	Runners []AppRunner `json:"runners"`
	// Compose is set when the folder has a compose file.
	Compose bool `json:"compose"`
}

// DeployInfo looks at the repository at dir and at what runs it, to offer
// deploy steps.
func (s *Service) DeployInfo(ctx context.Context, serverID, dir string) (DeployInfo, error) {
	dir, err := checkDir(dir)
	if err != nil {
		return DeployInfo{}, err
	}
	res, err := s.run.RunInput(ctx, serverID, app, shellx.Cmd("sh", "-s", "--", dir).String(), []byte(deployInfoScript))
	if err != nil {
		return DeployInfo{}, err
	}
	if res.Exit == 3 {
		return DeployInfo{}, apperr.New(apperr.NotFound, "cannot enter the repository folder")
	}
	return parseDeployInfo(dir, sections(string(res.Stdout))), nil
}

// sections splits the script's output at its @@name lines.
func sections(out string) map[string]string {
	m := map[string]string{}
	name := ""
	var b strings.Builder
	flush := func() {
		if name != "" {
			m[name] = b.String()
		}
		b.Reset()
	}
	for line := range strings.SplitSeq(out, "\n") {
		if n, ok := strings.CutPrefix(line, "@@"); ok && !strings.ContainsAny(n, " {") {
			flush()
			name = n
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	flush()
	return m
}

func parseDeployInfo(dir string, sec map[string]string) DeployInfo {
	files := strings.Fields(sec["root"])
	has := func(f string) bool { return slices.Contains(files, f) }
	root := strings.TrimSpace(sec["uid"]) == "0"
	info := DeployInfo{
		Pull:    Option{Name: "git pull", Command: shellx.Cmd("git", "pull", "--ff-only").String()},
		Builds:  []Option{},
		Runners: []AppRunner{},
	}

	install := installFor(files)
	if install != "" {
		info.Install = &Option{Name: strings.Fields(install)[0], Command: install}
	}
	if has("package.json") {
		info.Builds = append(info.Builds, scriptOptions(sec["package"], install)...)
	}
	if has("go.mod") {
		info.Builds = append(info.Builds, Option{Name: "go build", Command: "go build -o bin/ ./..."})
	}
	if has("Makefile") {
		info.Builds = append(info.Builds, Option{Name: "make", Command: "make"})
	}

	info.Runners = append(info.Runners, pm2Runners(dir, sec["pm2"])...)
	info.Runners = append(info.Runners, systemdRunners(dir, sec["systemd"], root)...)
	for _, f := range composeFiles {
		info.Compose = info.Compose || has(f)
	}
	projects := strings.Fields(sec["compose"])
	if info.Compose || len(projects) > 0 {
		name := ""
		if len(projects) > 0 {
			name = projects[0]
		}
		info.Runners = append(info.Runners, AppRunner{Kind: "compose", Option: Option{
			Name: name, Command: "docker compose up -d --build",
		}})
	}
	return info
}

// composeFiles are the names docker compose looks for.
var composeFiles = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

// scriptOptions offers each package.json script, run with the package
// manager that installs, "build" first.
func scriptOptions(pkg, install string) []Option {
	var p struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal([]byte(pkg), &p) != nil {
		return nil
	}
	names := make([]string, 0, len(p.Scripts))
	for n := range p.Scripts {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if (names[i] == "build") != (names[j] == "build") {
			return names[i] == "build"
		}
		return names[i] < names[j]
	})
	run := []string{"npm", "run"}
	switch install {
	case "pnpm install --frozen-lockfile":
		run = []string{"pnpm", "run"}
	case "yarn install --frozen-lockfile":
		run = []string{"yarn", "run"}
	}
	out := make([]Option, 0, len(names))
	for _, n := range names {
		out = append(out, Option{Name: n, Command: shellx.Cmd(run[0], append(run[1:], n)...).String()})
	}
	return out
}

// inside reports whether p is dir or a path under it.
func inside(p, dir string) bool {
	p = path.Clean(p)
	return p == dir || strings.HasPrefix(p, dir+"/")
}

// pm2Runners finds the pm2 processes started in dir, from pm2 jlist. The
// list also holds each process's environment, which is read past and
// never kept.
func pm2Runners(dir, jlist string) []AppRunner {
	var procs []struct {
		Name string `json:"name"`
		Env  struct {
			Cwd string `json:"pm_cwd"`
		} `json:"pm2_env"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(jlist)), &procs) != nil {
		return nil
	}
	var out []AppRunner
	seen := map[string]bool{}
	for _, p := range procs {
		if p.Name == "" || seen[p.Name] || !inside(p.Env.Cwd, dir) {
			continue
		}
		seen[p.Name] = true
		out = append(out, AppRunner{Kind: "pm2", Option: Option{
			Name: p.Name, Command: shellx.Cmd("pm2", "restart", p.Name).String(),
		}})
	}
	return out
}

// systemdRunners finds the running units whose working directory or
// program is in dir, from systemctl show. Not being root, restarting one
// goes through sudo.
func systemdRunners(dir, show string, root bool) []AppRunner {
	var out []AppRunner
	for block := range strings.SplitSeq(show, "\n\n") {
		var id, wd, exec string
		for line := range strings.SplitSeq(block, "\n") {
			k, v, _ := strings.Cut(line, "=")
			switch k {
			case "Id":
				id = v
			case "WorkingDirectory":
				wd = v
			case "ExecStart":
				exec = v
			}
		}
		inDir := wd != "" && inside(wd, dir) || strings.Contains(exec, dir+"/")
		if id == "" || !inDir {
			continue
		}
		cmd := shellx.Cmd("systemctl", "restart", id).String()
		if !root {
			cmd = "sudo " + cmd
		}
		out = append(out, AppRunner{Kind: "systemd", Option: Option{Name: id, Command: cmd, Sudo: !root}})
	}
	return out
}
