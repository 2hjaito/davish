package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Bindings that act on a server must mark their commands as changing
// it, or the Command log cannot export them (.claude/docs/adr/0022). A new
// binding whose name says it acts fails here until it uses a.change(),
// or is listed as an exception with the reason.
func TestActionBindingsMarkChanges(t *testing.T) {
	acts := regexp.MustCompile(`(Act|Apply|Remove|Create|Edit|Toggle|Reload|Clean|Truncate|Drop|Dump|Restore|Backup|Add|Rename|Mkdir|Compress|Save|SaveAsRoot|Write)$|^GitRun$`)
	except := map[string]string{
		"SaveServer":        "local settings, nothing runs on a server",
		"SettingsSave":      "local settings",
		"KeysPlanAdd":       "plan only",
		"DockerPlanRun":     "plan only",
		"TerminalWrite":     "keystrokes, not a command",
		"TransferUpload":    "SFTP, not in the Command log",
		"DatabaseCreateSQL": "builds the statement only",
	}
	fn := regexp.MustCompile(`(?s)\nfunc \(a \*App\) ([A-Z]\w*)\(.*?\n}\n`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f) //nolint:gosec // this package's own source
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range fn.FindAllStringSubmatch(string(src), -1) {
			name, body := m[1], m[0]
			if !acts.MatchString(name) || strings.HasPrefix(name, "Plan") || strings.Contains(name, "Plan") {
				continue
			}
			if _, ok := except[name]; ok {
				continue
			}
			if !strings.Contains(body, "a.ctx") && !strings.Contains(body, "a.change()") {
				continue // runs nothing with a context
			}
			if !strings.Contains(body, "a.change()") {
				t.Errorf("%s (%s) acts on a server but does not use a.change()", name, f)
			}
		}
	}
}
