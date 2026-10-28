// Package cron reads and edits crontabs on a server: the user's own
// (crontab -l / crontab -) and, read-only, /etc/crontab and /etc/cron.d.
package cron

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Schedule is when a job runs: five time fields expanded to the values
// they allow, or @reboot.
type Schedule struct {
	// Raw is the schedule as written: "*/5 * * * *", "@daily".
	Raw string `json:"raw"`
	// Reboot is @reboot: at startup, never on a clock.
	Reboot bool `json:"reboot"`
	// The values each field allows, sorted. Dow uses 0 for Sunday.
	Minute []int `json:"minute"`
	Hour   []int `json:"hour"`
	Dom    []int `json:"dom"`
	Month  []int `json:"month"`
	Dow    []int `json:"dow"`
	// DomAny and DowAny mark a field written as "*": when both day fields
	// are restricted, a day matching either one runs, as in Vixie cron.
	DomAny bool `json:"domAny"`
	DowAny bool `json:"dowAny"`
}

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

var monthNames = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
var dowNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// ErrSchedule marks a schedule cron would not accept.
var ErrSchedule = errors.New("invalid schedule")

// ParseSchedule reads a schedule: five fields or one @macro.
func ParseSchedule(raw string) (Schedule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "@reboot" {
		return Schedule{Raw: raw, Reboot: true}, nil
	}
	expr := raw
	if strings.HasPrefix(raw, "@") {
		m, ok := macros[strings.ToLower(raw)]
		if !ok {
			return Schedule{}, fmt.Errorf("%w: unknown %s", ErrSchedule, raw)
		}
		expr = m
	}
	f := strings.Fields(expr)
	if len(f) != 5 {
		return Schedule{}, fmt.Errorf("%w: want 5 fields, got %d", ErrSchedule, len(f))
	}
	s := Schedule{Raw: raw, DomAny: f[2] == "*", DowAny: f[4] == "*"}
	var err error
	if s.Minute, err = parseField(f[0], 0, 59, nil); err != nil {
		return Schedule{}, err
	}
	if s.Hour, err = parseField(f[1], 0, 23, nil); err != nil {
		return Schedule{}, err
	}
	if s.Dom, err = parseField(f[2], 1, 31, nil); err != nil {
		return Schedule{}, err
	}
	if s.Month, err = parseField(f[3], 1, 12, monthNames); err != nil {
		return Schedule{}, err
	}
	// 7 is Sunday too.
	if s.Dow, err = parseField(f[4], 0, 7, dowNames); err != nil {
		return Schedule{}, err
	}
	if slices.Contains(s.Dow, 7) {
		s.Dow = slices.DeleteFunc(s.Dow, func(v int) bool { return v == 7 })
		if !slices.Contains(s.Dow, 0) {
			s.Dow = append([]int{0}, s.Dow...)
		}
	}
	return s, nil
}

// parseField expands one field: "*", "5", "1-5", "*/15", "1-30/2",
// lists of those, and names when given (index+min is the value).
func parseField(field string, lo, hi int, names []string) ([]int, error) {
	set := map[int]bool{}
	for part := range strings.SplitSeq(field, ",") {
		rng, stepS, hasStep := strings.Cut(part, "/")
		step := 1
		if hasStep {
			n, err := strconv.Atoi(stepS)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("%w: step %q", ErrSchedule, part)
			}
			step = n
		}
		from, to := lo, hi
		switch {
		case rng == "*":
		case strings.Contains(rng, "-"):
			a, b, _ := strings.Cut(rng, "-")
			var err error
			if from, err = value(a, lo, hi, names); err != nil {
				return nil, err
			}
			if to, err = value(b, lo, hi, names); err != nil {
				return nil, err
			}
			if from > to {
				return nil, fmt.Errorf("%w: range %q", ErrSchedule, rng)
			}
		default:
			v, err := value(rng, lo, hi, names)
			if err != nil {
				return nil, err
			}
			from = v
			// "5/10" means from 5 to the end, every 10.
			if !hasStep {
				to = v
			}
		}
		for v := from; v <= to; v += step {
			set[v] = true
		}
	}
	out := make([]int, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	slices.Sort(out)
	return out, nil
}

func value(s string, lo, hi int, names []string) (int, error) {
	if i := slices.Index(names, strings.ToLower(s)); i >= 0 {
		return i + lo, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < lo || v > hi {
		return 0, fmt.Errorf("%w: value %q", ErrSchedule, s)
	}
	return v, nil
}

// Next returns the first run strictly after t, in t's location, or the
// zero time for @reboot or a date that never comes (Feb 30).
func (s Schedule) Next(t time.Time) time.Time {
	if s.Reboot {
		return time.Time{}
	}
	loc := t.Location()
	t = t.Truncate(time.Minute).Add(time.Minute)
	// Eight years covers every leap-day schedule; past that, never.
	limit := t.AddDate(8, 0, 0)
	for t.Before(limit) {
		if !slices.Contains(s.Month, int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !s.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			continue
		}
		if !slices.Contains(s.Hour, t.Hour()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, loc)
			continue
		}
		if !slices.Contains(s.Minute, t.Minute()) {
			t = t.Add(time.Minute)
			continue
		}
		return t
	}
	return time.Time{}
}

func (s Schedule) dayMatches(t time.Time) bool {
	dom := slices.Contains(s.Dom, t.Day())
	dow := slices.Contains(s.Dow, int(t.Weekday()))
	switch {
	case s.DomAny && s.DowAny:
		return true
	case s.DomAny:
		return dow
	case s.DowAny:
		return dom
	}
	return dom || dow
}

// Preview is a schedule being typed, parsed, with its next runs.
type Preview struct {
	Schedule Schedule `json:"schedule"`
	Next     []int64  `json:"next"` // unix seconds, up to three
}

// PreviewSchedule parses raw and lists its next runs in zone (the
// server's, from Table.Zone), from now.
func PreviewSchedule(raw, zone string, now time.Time) (Preview, error) {
	s, err := ParseSchedule(raw)
	if err != nil {
		return Preview{}, err
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc = time.UTC
	}
	p := Preview{Schedule: s, Next: []int64{}}
	for at := now.In(loc); len(p.Next) < 3; {
		if at = s.Next(at); at.IsZero() {
			break
		}
		p.Next = append(p.Next, at.Unix())
	}
	return p, nil
}
