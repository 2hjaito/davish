package deploy

import (
	"bytes"
	_ "embed"
	"path"
	"regexp"
	"strings"
	"text/template"

	"davish/internal/apperr"
)

//go:embed unit.service.tmpl
var unitTmpl string

var unitTemplate = template.Must(template.New("unit").Parse(unitTmpl))

// Unit is the systemd service form.
type Unit struct {
	Name       string   `json:"name"`
	ExecStart  string   `json:"execStart"`
	User       string   `json:"user"`
	WorkingDir string   `json:"workingDir"`
	Env        []string `json:"env"` // KEY=value
	EnvFile    string   `json:"envFile"`
	// Restart is always, on-failure or no.
	Restart string `json:"restart"`
}

var (
	unitName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]{0,63}$`)
	userName = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	envKV    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	absPath  = regexp.MustCompile(`^/[^\x00\n\r"]*$`)
)

// UnitPath is where a unit named name is written.
func UnitPath(name string) string {
	return path.Join("/etc/systemd/system", name+".service")
}

func (u Unit) check() (Unit, error) {
	bad := func(msg string) (Unit, error) { return u, apperr.New(apperr.InvalidInput, msg) }
	u.Name = strings.TrimSuffix(strings.TrimSpace(u.Name), ".service")
	if !unitName.MatchString(u.Name) {
		return bad("not a service name")
	}
	u.ExecStart = strings.TrimSpace(u.ExecStart)
	if !strings.HasPrefix(u.ExecStart, "/") || strings.ContainsAny(u.ExecStart, "\n\r\x00") {
		return bad("ExecStart starts with the program's absolute path, such as /usr/bin/node")
	}
	if u.User = strings.TrimSpace(u.User); u.User != "" && !userName.MatchString(u.User) {
		return bad("not a user name")
	}
	if !absPath.MatchString(u.WorkingDir) {
		return bad("the working folder is an absolute path")
	}
	if u.EnvFile = strings.TrimSpace(u.EnvFile); u.EnvFile != "" && !absPath.MatchString(u.EnvFile) {
		return bad("the env file is an absolute path")
	}
	var env []string
	for _, e := range u.Env {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if !envKV.MatchString(e) || strings.ContainsAny(e, "\n\r\x00") {
			return bad("environment lines are KEY=value")
		}
		// systemd unquotes "…" and its \" and \\ escapes.
		env = append(env, `"`+strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(e)+`"`)
	}
	u.Env = env
	switch u.Restart {
	case "":
		u.Restart = "always"
	case "always", "on-failure", "no":
	default:
		return bad("unknown restart policy")
	}
	return u, nil
}

// RenderUnit returns the unit file for u.
func RenderUnit(u Unit) (string, error) {
	u, err := u.check()
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = unitTemplate.Execute(&b, struct {
		Unit
		Description string
	}{u, u.Name + " (set up by Davish)"})
	return b.String(), err
}
