package cron

import (
	"regexp"
	"strings"
)

// Entry kinds of a crontab line.
const (
	KindJob     = "job"
	KindEnv     = "env"
	KindComment = "comment"
	KindBlank   = "blank"
)

// Entry is one line of a crontab file.
type Entry struct {
	Line int    // 1-based
	Text string // as written
	Kind string
	// Job lines, enabled or commented out.
	Schedule string
	User     string // system crontabs only
	Command  string
	Enabled  bool
	// Env lines.
	Name, Value string
}

var envLine = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)

// ParseCrontab reads a crontab. System crontabs (/etc/crontab,
// /etc/cron.d) have a user field before the command. A "#" right against
// a job line ("#30 2 * * 0 cmd") is a disabled job, which is how Davish
// turns jobs off; "# 0 5 * * 1 tar …" with a space is a comment, as in
// the examples of crontab templates.
func ParseCrontab(text string, system bool) []Entry {
	var out []Entry
	for i, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		e := Entry{Line: i + 1, Text: line}
		trimmed := strings.TrimSpace(line)
		switch {
		case text == "":
			return nil
		case trimmed == "":
			e.Kind = KindBlank
		case strings.HasPrefix(trimmed, "#"):
			e.Kind = KindComment
			body := trimmed[1:]
			if body == "" || body[0] == ' ' || body[0] == '\t' || body[0] == '#' {
				break
			}
			if j, ok := parseJob(body, system); ok {
				j.Line, j.Text, j.Enabled = e.Line, line, false
				e = j
			}
		default:
			if m := envLine.FindStringSubmatch(trimmed); m != nil {
				e.Kind, e.Name, e.Value = KindEnv, m[1], unquote(m[2])
			} else if j, ok := parseJob(trimmed, system); ok {
				j.Line, j.Text, j.Enabled = e.Line, line, true
				e = j
			} else {
				// cron would reject it; show it as it is.
				e.Kind = KindComment
			}
		}
		out = append(out, e)
	}
	return out
}

// parseJob splits "schedule [user] command", keeping the command's own
// spacing.
func parseJob(line string, system bool) (Entry, bool) {
	n := 5
	if strings.HasPrefix(line, "@") {
		n = 1
	}
	if system {
		n++
	}
	fields, rest := splitFields(line, n)
	if len(fields) < n || rest == "" {
		return Entry{}, false
	}
	schedFields := fields
	e := Entry{Kind: KindJob, Command: rest}
	if system {
		schedFields, e.User = fields[:n-1], fields[n-1]
	}
	e.Schedule = strings.Join(schedFields, " ")
	if _, err := ParseSchedule(e.Schedule); err != nil {
		return Entry{}, false
	}
	return e, true
}

// splitFields takes the first n whitespace-separated fields of s and
// returns the rest of the line after them, untouched but trimmed.
func splitFields(s string, n int) ([]string, string) {
	var fields []string
	rest := s
	for len(fields) < n {
		rest = strings.TrimLeft(rest, " \t")
		if rest == "" {
			break
		}
		end := strings.IndexAny(rest, " \t")
		if end < 0 {
			end = len(rest)
		}
		fields = append(fields, rest[:end])
		rest = rest[end:]
	}
	return fields, strings.TrimSpace(rest)
}

// unquote drops the quotes cron itself drops around an env value.
func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}
