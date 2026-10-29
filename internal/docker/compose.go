package docker

import (
	"context"
	"embed"
	"path"
	"regexp"
	"slices"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Compose actions on a project.
var composeActions = map[string][]string{
	"up":      {"up", "-d"},
	"down":    {"down"},
	"pull":    {"pull"},
	"restart": {"restart"},
}

var projectName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ComposeCommand is the docker compose line for action on a project,
// typed into a terminal. Without the project's folder only down and
// restart work, by project name.
func ComposeCommand(project, dir, file, action string) (string, error) {
	verb, ok := composeActions[action]
	if !ok {
		return "", apperr.New(apperr.InvalidInput, "unknown compose action")
	}
	if !projectName.MatchString(project) {
		return "", apperr.New(apperr.InvalidInput, "not a compose project")
	}
	args := []string{"compose"}
	switch {
	case dir != "" && absPath.MatchString(dir):
		args = append(args, "--project-directory", dir)
		if file != "" {
			if !absPath.MatchString(file) {
				return "", apperr.New(apperr.InvalidInput, "not a compose file")
			}
			args = append(args, "-f", file)
		}
		args = append(args, "-p", project)
	case action == "down" || action == "restart":
		args = append(args, "-p", project)
	default:
		return "", apperr.New(apperr.InvalidInput, "the project's folder is unknown")
	}
	return shellx.Cmd("docker", append(args, verb...)...).String(), nil
}

//go:embed stacks/*.yaml
var stacks embed.FS

// StackTemplates are the names of the new stack templates.
var StackTemplates = []string{"blank", "web-db", "app-redis"}

// StackPlan is what creating a stack writes, for the preview.
type StackPlan struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Command string `json:"command"`
}

// PlanStack describes writing template as compose.yaml in dir. The shell
// refuses to replace a file that exists (set -C).
func PlanStack(dir, template string) (StackPlan, error) {
	dir = path.Clean(strings.TrimSpace(dir))
	if !absPath.MatchString(dir) || dir == "/" || strings.ContainsAny(dir, "'\"$`\\") {
		return StackPlan{}, apperr.New(apperr.InvalidInput, "not a folder for the stack")
	}
	if !slices.Contains(StackTemplates, template) {
		return StackPlan{}, apperr.New(apperr.InvalidInput, "unknown template")
	}
	b, err := stacks.ReadFile("stacks/" + template + ".yaml")
	if err != nil {
		return StackPlan{}, err
	}
	p := StackPlan{Path: path.Join(dir, "compose.yaml"), Content: string(b)}
	p.Command = shellx.Cmd("mkdir", "-p", "--", dir).String() + " && (set -C && " +
		shellx.Cmd("printf", "%s", p.Content).String() + " > " + shellx.Quote(p.Path) + ")"
	return p, nil
}

// CreateStack runs command, which must be what PlanStack gives.
func (s *Service) CreateStack(ctx context.Context, serverID, dir, template, command string) error {
	p, err := PlanStack(dir, template)
	if err != nil {
		return err
	}
	if p.Command != command {
		return apperr.New(apperr.FSStale, "the stack changed since it was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return err
	}
	if res.Exit == 0 {
		return nil
	}
	out := string(res.Stderr)
	switch {
	case strings.Contains(out, "exists") || strings.Contains(out, "cannot overwrite"):
		return apperr.New(apperr.FSExists, "the folder already has a compose.yaml")
	case strings.Contains(out, "Permission denied"):
		return apperr.New(apperr.CmdNeedsRoot, "the folder is not writable")
	}
	return apperr.New(apperr.CmdFailed, "could not write the compose file")
}
