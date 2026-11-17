package ansibleinv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// split breaks a line into words like Python's shlex.split(comments=True),
// which Ansible uses for INI host lines: quotes group, a backslash escapes
// outside single quotes, and "#" at the start of a word ends the line.
func split(line string) ([]string, error) {
	var (
		words []string
		cur   strings.Builder
		in    bool // inside a word
		quote rune
	)
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\'):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, in = r, true
		case r == '\\' && i+1 < len(rs):
			i++
			cur.WriteRune(rs[i])
			in = true
		case r == ' ' || r == '\t':
			if in {
				words = append(words, cur.String())
				cur.Reset()
				in = false
			}
		case r == '#' && !in:
			i = len(rs)
		default:
			cur.WriteRune(r)
			in = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unclosed quote")
	}
	if in {
		words = append(words, cur.String())
	}
	return words, nil
}

// expand turns a host pattern with ranges, such as web[01:03].example.com
// or db-[a:c], into its names. A range may have a step ([1:9:2]) and
// keeps the leading zeros of its start.
func expand(pattern string) ([]string, error) {
	open := strings.Index(pattern, "[")
	if open < 0 {
		return []string{pattern}, nil
	}
	end := strings.Index(pattern[open:], "]")
	if end < 0 {
		return nil, fmt.Errorf("unclosed range in %q", pattern)
	}
	end += open
	head, spec, tail := pattern[:open], pattern[open+1:end], pattern[end+1:]
	parts := strings.Split(spec, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, fmt.Errorf("bad range in %q", pattern)
	}
	step := 1
	if len(parts) == 3 {
		s, err := strconv.Atoi(parts[2])
		if err != nil || s < 1 {
			return nil, fmt.Errorf("bad range step in %q", pattern)
		}
		step = s
	}
	items, err := rangeItems(parts[0], parts[1], step)
	if err != nil {
		return nil, fmt.Errorf("%w in %q", err, pattern)
	}
	rest, err := expand(tail)
	if err != nil {
		return nil, err
	}
	if len(items)*len(rest) > maxRange {
		return nil, fmt.Errorf("range %q makes more than %d hosts", pattern, maxRange)
	}
	out := make([]string, 0, len(items)*len(rest))
	for _, it := range items {
		for _, r := range rest {
			out = append(out, head+it+r)
		}
	}
	return out, nil
}

func rangeItems(beg, end string, step int) ([]string, error) {
	if beg == "" {
		beg = "0"
	}
	if b, err := strconv.Atoi(beg); err == nil {
		e, err := strconv.Atoi(end)
		if err != nil || e < b {
			return nil, errors.New("bad range end")
		}
		if (e-b)/step+1 > maxRange {
			return nil, errors.New("range too large")
		}
		width := 0
		if len(beg) > 1 && beg[0] == '0' {
			width = len(beg)
		}
		var out []string
		for i := b; i <= e; i += step {
			out = append(out, fmt.Sprintf("%0*d", width, i))
		}
		return out, nil
	}
	if len(beg) != 1 || len(end) != 1 || !isLetter(beg[0]) || !isLetter(end[0]) || end[0] < beg[0] {
		return nil, errors.New("bad range")
	}
	var out []string
	for c := int(beg[0]); c <= int(end[0]); c += step {
		out = append(out, string(rune(c))) //nolint:gosec // c is an ASCII letter
	}
	return out, nil
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isAbs(p string) bool { return filepath.IsAbs(p) || strings.HasPrefix(p, "/") }

func joinPath(dir, p string) string { return filepath.Join(dir, p) }
