// Package deploy plans the "Deploy a new app" wizard: it puts together the
// clone, the way the app runs, the nginx site and the certificate as one
// line for a terminal, with a summary of what it makes and a line that
// removes it all again. It only builds text; nothing runs from here.
package deploy

import (
	"path"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/docker"
	"davish/internal/git"
	"davish/internal/nginx"
	"davish/internal/shellx"
)

// Sources and ways to run.
const (
	SourceGit    = "git"
	SourceImage  = "image"
	SourceFolder = "folder"

	RunCompose   = "compose"
	RunSystemd   = "systemd"
	RunStatic    = "static"
	RunContainer = "container"
)

// Spec is the wizard's answers.
type Spec struct {
	Source  string `json:"source"`
	RepoURL string `json:"repoURL"`
	Branch  string `json:"branch"`
	Image   string `json:"image"`
	// Dir is the app's folder: cloned into, already there, or served.
	Dir string `json:"dir"`
	Run string `json:"run"`
	// Port is where the app listens on 127.0.0.1, for nginx to proxy to.
	Port int `json:"port"`
	// ContainerPort is the image's port, for a container.
	ContainerPort int `json:"containerPort"`
	// Name names the service or container.
	Name     string   `json:"name"`
	Unit     Unit     `json:"unit"`
	Domains  []string `json:"domains"`
	ConfD    bool     `json:"confD"`
	SSL      bool     `json:"ssl"`
	Redirect bool     `json:"redirect"`
}

// Step is one part of the plan.
type Step struct {
	Title   string `json:"title"`
	Command string `json:"command"`
	// File is the content a step writes, to preview.
	File string `json:"file,omitempty"`
}

// Plan is the whole deployment.
type Plan struct {
	Steps []Step `json:"steps"`
	// Line runs the steps in order, stopping at the first that fails.
	Line string `json:"line"`
	// Created lists what the steps make, for the summary.
	Created []string `json:"created"`
	// Uninstall removes it all again, each part even when another fails.
	Uninstall string `json:"uninstall"`
}

func bad(msg string) (Plan, error) { return Plan{}, apperr.New(apperr.InvalidInput, msg) }

// Make plans spec.
func Make(spec Spec) (Plan, error) {
	var p Plan
	var undo []string
	dir := path.Clean(strings.TrimSpace(spec.Dir))
	if spec.Source != SourceImage || spec.Run == RunStatic {
		if !strings.HasPrefix(dir, "/") || dir == "/" {
			return bad("the app's folder is an absolute path")
		}
	}

	switch spec.Source {
	case SourceGit:
		clone, err := git.PlanClone(git.CloneSpec{URL: spec.RepoURL, Dir: dir, Branch: spec.Branch})
		if err != nil {
			return Plan{}, err
		}
		// The folder is made for the user, so the clone needs no root.
		own := shellx.Cmd("sudo", "install", "-d", "-o").String() + ` "$USER" -- ` + shellx.Quote(dir)
		p.Steps = append(p.Steps, Step{Title: "clone", Command: own + " && " + clone.Command})
		p.Created = append(p.Created, dir)
		undo = append(undo, shellx.Cmd("sudo", "rm", "-rf", "--", dir).String())
	case SourceFolder:
	case SourceImage:
		if spec.Run != RunContainer {
			return bad("an image runs as a container")
		}
	default:
		return bad("unknown source")
	}

	switch spec.Run {
	case RunCompose:
		if spec.Source == SourceImage {
			return bad("compose runs from a folder")
		}
		p.Steps = append(p.Steps, Step{Title: "compose", Command: shellx.InDir(dir, shellx.Cmd("docker", "compose", "up", "-d", "--build"))})
		p.Created = append(p.Created, "docker compose: "+dir)
		undo = append([]string{shellx.InDir(dir, shellx.Cmd("docker", "compose", "down"))}, undo...)
	case RunSystemd:
		u := spec.Unit
		u.WorkingDir = dir
		file, err := RenderUnit(u)
		if err != nil {
			return Plan{}, err
		}
		u, _ = u.check()
		unit := UnitPath(u.Name)
		p.Steps = append(p.Steps, Step{
			Title: "systemd",
			Command: shellx.Cmd("printf", "%s", file).String() + " | " + shellx.Cmd("sudo", "tee", "--", unit).String() + " > /dev/null && " +
				shellx.Cmd("sudo", "systemctl", "daemon-reload").String() + " && " +
				shellx.Cmd("sudo", "systemctl", "enable", "--now", u.Name).String(),
			File: file,
		})
		p.Created = append(p.Created, unit, "systemd: "+u.Name)
		undo = append([]string{
			shellx.Cmd("sudo", "systemctl", "disable", "--now", u.Name).String(),
			shellx.Cmd("sudo", "rm", "-f", "--", unit).String(),
			shellx.Cmd("sudo", "systemctl", "daemon-reload").String(),
		}, undo...)
	case RunContainer:
		run, err := docker.PlanRun(docker.RunSpec{
			Name:    spec.Name,
			Image:   spec.Image,
			Restart: "unless-stopped",
			Ports:   []docker.PortMap{{Host: "127.0.0.1", HostPort: spec.Port, ContainerPort: spec.ContainerPort}},
		})
		if err != nil {
			return Plan{}, err
		}
		if spec.Name == "" {
			return bad("the container needs a name")
		}
		p.Steps = append(p.Steps, Step{Title: "container", Command: run.Command})
		p.Created = append(p.Created, "container: "+spec.Name)
		undo = append([]string{shellx.Cmd("docker", "rm", "-f", spec.Name).String()}, undo...)
	case RunStatic:
	default:
		return bad("unknown way to run the app")
	}

	if len(spec.Domains) > 0 {
		site := nginx.NewSite{Names: spec.Domains, Type: nginx.TypeStatic, Root: dir}
		if spec.Run != RunStatic {
			if spec.Port < 1 || spec.Port > 65535 {
				return bad("the app's port is 1 to 65535")
			}
			site = nginx.NewSite{Names: spec.Domains, Type: nginx.TypeProxy, Upstream: "127.0.0.1:" + strconv.Itoa(spec.Port), Websocket: true}
		}
		config, err := nginx.RenderSite(site)
		if err != nil {
			return Plan{}, err
		}
		cp, err := nginx.PlanCreateFor(site, config, spec.ConfD)
		if err != nil {
			return Plan{}, err
		}
		p.Steps = append(p.Steps, Step{Title: "nginx", Command: cp.Sudo, File: cp.Config})
		p.Created = append(p.Created, cp.Path)
		rm := []string{cp.Path}
		if !spec.ConfD {
			rm = append(rm, path.Join(nginx.DirEnabled, cp.Name))
			p.Created = append(p.Created, path.Join(nginx.DirEnabled, cp.Name))
		}
		siteUndo := shellx.Cmd("sudo", append([]string{"rm", "-f", "--"}, rm...)...).String() + " && " +
			shellx.Cmd("sudo", "systemctl", "reload", "nginx").String()
		if spec.SSL {
			if names := nginx.CertNames(spec.Domains); len(names) > 0 {
				p.Steps = append(p.Steps, Step{Title: "https", Command: nginx.CertbotCommand(names, spec.Redirect)})
				p.Created = append(p.Created, "certificate: "+names[0])
				undo = append([]string{shellx.Cmd("sudo", "certbot", "delete", "--non-interactive", "--cert-name", names[0]).String()}, undo...)
			}
		}
		undo = append([]string{siteUndo}, undo...)
	}

	if len(p.Steps) == 0 {
		return bad("nothing to set up")
	}
	parts := make([]string, 0, 2*len(p.Steps)+1)
	for i, s := range p.Steps {
		parts = append(parts, shellx.Cmd("echo", "▶ "+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(p.Steps))+"  "+s.Title).String(), "{ "+s.Command+"; }")
	}
	parts = append(parts, shellx.Cmd("echo", "✓ Deployed").String())
	p.Line = strings.Join(parts, " && ") + " || " + shellx.Cmd("echo", "✗ Stopped: the step above failed").String()
	p.Uninstall = strings.Join(undo, " ; ")
	return p, nil
}
