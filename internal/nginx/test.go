package nginx

import (
	"regexp"
	"strconv"
	"strings"
)

// Issue is one message of nginx -t.
type Issue struct {
	Level   string `json:"level"` // emerg, alert, crit, error, warn
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// TestResult is what nginx -t said.
type TestResult struct {
	OK bool `json:"ok"`
	// NeedsRoot is set when the syntax is fine as far as nginx got, but it
	// stopped on files only root may open (the pid file, certificate keys).
	NeedsRoot bool    `json:"needsRoot"`
	Issues    []Issue `json:"issues"`
	// Output is nginx's own text, to show as is.
	Output string `json:"output"`
}

var issueLine = regexp.MustCompile(`\[(emerg|alert|crit|error|warn)\] \d+#\d+: (.*?)(?: in (\S+):(\d+))?$`)

// parseTest reads nginx -t's output (stderr) and exit status.
func parseTest(out string, exit int) TestResult {
	r := TestResult{OK: exit == 0, Issues: []Issue{}, Output: strings.TrimSpace(out)}
	errors, denied := 0, 0
	for line := range strings.Lines(out) {
		m := issueLine.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if m == nil {
			continue
		}
		// Expected whenever nginx -t runs without root; it says nothing
		// about the configuration.
		if m[1] == "warn" && strings.Contains(m[2], `the "user" directive makes sense only if the master process runs with super-user privileges`) {
			continue
		}
		is := Issue{Level: m[1], Message: m[2], File: m[3]}
		is.Line, _ = strconv.Atoi(m[4])
		r.Issues = append(r.Issues, is)
		if is.Level != "warn" {
			errors++
			if strings.Contains(is.Message, "Permission denied") {
				denied++
			}
		}
	}
	r.NeedsRoot = !r.OK && errors > 0 && errors == denied
	return r
}
