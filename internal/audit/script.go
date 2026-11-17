package audit

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
	"davish/internal/store"
)

// Target is how the script reaches a server: what `ssh` needs.
type Target struct {
	Name string
	User string
	Host string
	Port int
	// Jump is "user@host:port" of the jump host, "" when direct.
	Jump string
}

// Targets maps saved servers by ID, with their jump host resolved.
func Targets(servers []store.Server) map[string]Target {
	byID := map[string]store.Server{}
	for _, s := range servers {
		byID[s.ID] = s
	}
	out := map[string]Target{}
	for _, s := range servers {
		t := Target{Name: s.Name, User: s.User, Host: s.Host, Port: s.Port}
		if j, ok := byID[s.JumpID]; ok && s.JumpID != "" {
			t.Jump = j.User + "@" + j.Host
			if j.Port != 22 && j.Port != 0 {
				t.Jump += ":" + strconv.Itoa(j.Port)
			}
		}
		out[s.ID] = t
	}
	return out
}

// ScriptOptions tunes Script.
type ScriptOptions struct {
	// Failed keeps commands that exited non-zero, with their exit code.
	Failed bool
	// Now is when the script is made, for its header.
	Now time.Time
}

// Script turns entries into a bash script that runs them again, each
// over its own `ssh -t` so sudo and other prompts still work. Entries
// are grouped by server, in time order. sudo as Davish runs it becomes a
// plain `sudo`, and content sent on stdin, never recorded, is left as a
// comment (docs/adr/0022).
func Script(entries []Entry, targets map[string]Target, opt ScriptOptions) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	fmt.Fprintf(&b, "# Exported from the Davish Command log at %s.\n", opt.Now.UTC().Format("2006-01-02 15:04 UTC"))
	b.WriteString("# Each command runs over its own ssh connection, in the server's login\n")
	b.WriteString("# shell (Davish ran them with sh -c). Read it before you run it.\n")
	b.WriteString("set -euo pipefail\n")

	kept := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Exit != 0 && !opt.Failed {
			continue
		}
		kept = append(kept, e)
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].ID < kept[j].ID })

	// Servers in the order of their first command.
	var order []string
	groups := map[string][]Entry{}
	for _, e := range kept {
		if _, ok := groups[e.ServerID]; !ok {
			order = append(order, e.ServerID)
		}
		groups[e.ServerID] = append(groups[e.ServerID], e)
	}
	if len(order) == 0 {
		b.WriteString("\n# No commands to export.\n")
		return b.String()
	}
	for _, id := range order {
		t, ok := targets[id]
		if !ok {
			t = Target{Name: id, User: "USER", Host: "HOST", Port: 22}
		}
		fmt.Fprintf(&b, "\n# --- %s (%s) ---\n", oneLine(t.Name), address(t))
		ssh := sshPrefix(t)
		for _, e := range groups[id] {
			fmt.Fprintf(&b, "\n# %s %s", time.UnixMilli(e.Time).UTC().Format("2006-01-02 15:04:05"), oneLine(e.App))
			if e.Exit != 0 {
				fmt.Fprintf(&b, ", exit %d then", e.Exit)
			}
			b.WriteString("\n")
			line := Plain(e.Command)
			if e.Input {
				b.WriteString("# content sent on stdin was not recorded; the command was:\n")
				for _, l := range strings.Split(line, "\n") {
					b.WriteString("#   " + l + "\n")
				}
				continue
			}
			b.WriteString(ssh + " " + shellx.Quote(line) + "\n")
		}
	}
	return b.String()
}

func address(t Target) string {
	a := t.User + "@" + t.Host
	if t.Port != 22 && t.Port != 0 {
		a += ":" + strconv.Itoa(t.Port)
	}
	if t.Jump != "" {
		a += " via " + t.Jump
	}
	return a
}

func sshPrefix(t Target) string {
	args := []string{"-t"}
	if t.Port != 22 && t.Port != 0 {
		args = append(args, "-p", strconv.Itoa(t.Port))
	}
	if t.Jump != "" {
		args = append(args, "-J", t.Jump)
	}
	args = append(args, t.User+"@"+t.Host, "--")
	return shellx.Cmd("ssh", args...).String()
}

// The forms sudo.Keeper gives a line run as root (internal/sudo).
var sudoForms = []string{
	"LC_ALL=C " + shellx.Cmd("sudo", "-n", "--", "sh", "-c").String() + " ",
	"LC_ALL=C " + shellx.Cmd("sudo", "-S", "-k", "-p", "", "--", "sh", "-c").String() + " ",
}

// Plain is the command as a person would type it: Davish's sudo forms
// (no prompt, or the password on stdin) become a plain sudo.
func Plain(line string) string {
	for _, f := range sudoForms {
		if rest, ok := strings.CutPrefix(line, f); ok {
			return "sudo sh -c " + rest
		}
	}
	return line
}

func oneLine(s string) string { return strings.ReplaceAll(s, "\n", " ") }

// SaveScript writes script to path on this computer, executable by its
// owner only.
func SaveScript(path, script string) error {
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil { //nolint:gosec // a path the user chose in the save dialog
		return apperr.Wrap(apperr.Internal, "cannot save the script", err)
	}
	// WriteFile keeps the mode of a file it overwrites.
	if err := os.Chmod(path, 0o700); err != nil { //nolint:gosec // as above
		return apperr.Wrap(apperr.Internal, "cannot save the script", err)
	}
	return nil
}
