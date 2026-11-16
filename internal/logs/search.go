package logs

import (
	"context"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/docker"
	"davish/internal/shellx"
	"davish/internal/svc"
)

// Runner runs a shellx-built command line on a server and records it in
// the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// SearchLimit is how many of the latest matching lines a search returns.
const SearchLimit = 500

// AroundLimit is the most lines LogsAround returns at once.
const AroundLimit = 200

// searchTimeout bounds grep or journalctl on the server: a file of a few
// GB takes seconds, a stuck disk must not take for ever.
const searchTimeout = 60 * time.Second

// maxLineBytes cuts each returned line on the server, so one huge line
// cannot fill the result; the stream keeps 16 KiB, a result needs less.
const maxLineBytes = 4000

// SearchQuery is what to look for in a source's whole history.
type SearchQuery struct {
	Query     string `json:"query"`
	MatchCase bool   `json:"matchCase"`
}

// Line is one line of a file with its number; Number is 0 where the
// source has none (journal, container).
type Line struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// SearchResult is the latest matching lines, oldest first.
type SearchResult struct {
	Lines []Line `json:"lines"`
	// Truncated is set when SearchLimit lines came back: there may be
	// older matches.
	Truncated bool `json:"truncated"`
	TimedOut  bool `json:"timedOut"`
	// Restricted is set when the journal hid other users' entries.
	Restricted bool   `json:"restricted"`
	Command    string `json:"command"`
}

// Around is a page of a file.
type Around struct {
	Lines []Line `json:"lines"`
	// EOF is set when the file ended before the page did.
	EOF bool `json:"eof"`
}

// Searcher searches a source's history with one-shot commands.
type Searcher struct {
	run Runner

	mu sync.Mutex
	// searches holds the cancel of each running search by its token.
	searches map[string]context.CancelFunc
}

// NewSearcher creates a Searcher.
func NewSearcher(run Runner) *Searcher {
	return &Searcher{run: run, searches: map[string]context.CancelFunc{}}
}

// Search returns the latest lines of src that contain q.Query. token
// names it for Cancel.
func (s *Searcher) Search(ctx context.Context, serverID string, src Source, q SearchQuery, token string) (SearchResult, error) {
	line, err := searchLine(src, q, false)
	if err != nil {
		return SearchResult{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.track(token, cancel)()
	res, err := s.run.RunLine(ctx, serverID, "logs", line)
	if err != nil {
		return SearchResult{}, err
	}
	if src.Kind == KindJournal && noJournalGrep(res.Stderr) {
		// journalctl built without PCRE2: filter its output with grep.
		line, _ = searchLine(src, q, true)
		res, err = s.run.RunLine(ctx, serverID, "logs", line)
		if err != nil {
			return SearchResult{}, err
		}
	}
	out := parseSearch(src.Kind, res.Stdout)
	out.Command = line
	exit := exitOf(res.Stderr)
	out.TimedOut = exit == 124
	out.Restricted = src.Kind == KindJournal && journalRestricted(res.Stderr)
	if len(out.Lines) == 0 && exit != 0 && exit != 1 && exit != 124 {
		return SearchResult{}, failure(res.Stderr)
	}
	return out, nil
}

// Cancel stops a running Search.
func (s *Searcher) Cancel(token string) {
	s.mu.Lock()
	cancel := s.searches[token]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Searcher) track(token string, cancel context.CancelFunc) func() {
	if token == "" {
		return func() {}
	}
	s.mu.Lock()
	s.searches[token] = cancel
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.searches, token)
		s.mu.Unlock()
	}
}

// Around returns up to count lines of file starting at line from (1-based).
func (s *Searcher) Around(ctx context.Context, serverID, file string, from, count int) (Around, error) {
	line, err := aroundLine(file, from, count)
	if err != nil {
		return Around{}, err
	}
	res, err := s.run.RunLine(ctx, serverID, "logs", line)
	if err != nil {
		return Around{}, err
	}
	if res.Exit != 0 {
		return Around{}, failure(res.Stderr)
	}
	texts := splitLines(res.Stdout)
	out := Around{Lines: make([]Line, 0, len(texts)), EOF: len(texts) < count}
	for i, t := range texts {
		out.Lines = append(out.Lines, Line{Number: from + i, Text: t})
	}
	return out, nil
}

// searchLine builds the search command for src. Every tool's exit status
// goes to stderr as "exit:N", since sh has no pipefail to carry it through
// tail. viaGrep filters the journal with grep instead of journalctl -g.
func searchLine(src Source, q SearchQuery, viaGrep bool) (string, error) {
	query := strings.TrimSpace(q.Query)
	if query == "" || strings.ContainsAny(query, "\x00\n\r") {
		return "", apperr.New(apperr.InvalidInput, "search text must be one non-empty line")
	}
	secs := strconv.Itoa(int(searchTimeout / time.Second))
	grep := func(extra ...string) shellx.Command {
		args := []string{"-F"}
		if !q.MatchCase {
			args = append(args, "-i")
		}
		args = append(args, extra...)
		return shellx.Cmd("grep", append(args, "-e", query)...)
	}
	tail := shellx.Cmd("tail", "-n", strconv.Itoa(SearchLimit)).String() + " | " +
		shellx.Cmd("cut", "-b", "1-"+strconv.Itoa(maxLineBytes)).String()
	switch src.Kind {
	case KindFile:
		if err := checkFile(src.Target); err != nil {
			return "", err
		}
		g := grep("-n")
		cmd := shellx.Cmd("timeout", append([]string{secs}, append(g.Args(), "--", src.Target)...)...)
		return wrap(cmd.String()) + " | " + tail, nil
	case KindJournal:
		args := []string{secs, "journalctl"}
		if src.Target != "" {
			if err := svc.CheckUnit(src.Target); err != nil {
				return "", err
			}
			args = append(args, "-u", src.Target)
		}
		args = append(args, "--no-pager", "-o", "short-iso")
		if viaGrep {
			return wrap(shellx.Cmd("timeout", args...).String()) + " | " + grep().String() + " | " + tail, nil
		}
		// -g is a PCRE2 pattern: \Q…\E makes it literal. It is
		// case-insensitive unless asked, as journalctl only folds case
		// by itself when the pattern is all lowercase.
		args = append(args, "-g", pcreLiteral(query), "--case-sensitive="+strconv.FormatBool(q.MatchCase), "-n", strconv.Itoa(SearchLimit))
		return wrap(shellx.Cmd("timeout", args...).String()) + " | " + tail, nil
	case KindDocker:
		if err := docker.CheckName(src.Target); err != nil {
			return "", err
		}
		// A container's stderr is its output too, so both go through grep.
		logs := shellx.Cmd("timeout", secs, "docker", "logs", "--timestamps", src.Target).String() + " 2>&1"
		return wrap(logs) + " | " + grep().String() + " | " + tail, nil
	}
	return "", apperr.New(apperr.InvalidInput, "unknown log source")
}

// wrap runs cmd and reports its exit status on stderr.
func wrap(cmd string) string {
	return "{ " + cmd + `; echo "exit:$?" >&2; }`
}

// aroundLine prints lines from..from+count-1 of file and stops reading
// there, so a page near the start of a huge file is quick.
func aroundLine(file string, from, count int) (string, error) {
	if err := checkFile(file); err != nil {
		return "", err
	}
	if from < 1 || count < 1 || count > AroundLimit {
		return "", apperr.New(apperr.InvalidInput, "line range out of bounds")
	}
	last := strconv.Itoa(from + count - 1)
	script := strconv.Itoa(from) + "," + last + "p;" + last + "q"
	return shellx.Cmd("sed", "-n", "-e", script, "--", file).String() + " | " +
		shellx.Cmd("cut", "-b", "1-"+strconv.Itoa(maxLineBytes)).String(), nil
}

func checkFile(p string) error {
	if !path.IsAbs(p) || path.Clean(p) != p || strings.ContainsAny(p, "\x00\n\r") {
		return apperr.New(apperr.InvalidInput, "log path must be absolute and clean")
	}
	return nil
}

// pcreLiteral quotes s for PCRE2 with \Q…\E; a "\E" inside s is closed,
// written escaped and reopened.
func pcreLiteral(s string) string {
	return `\Q` + strings.ReplaceAll(s, `\E`, `\E\\E\Q`) + `\E`
}

func parseSearch(kind string, stdout []byte) SearchResult {
	texts := splitLines(stdout)
	out := SearchResult{Lines: make([]Line, 0, len(texts)), Truncated: len(texts) >= SearchLimit}
	for _, t := range texts {
		if kind == KindFile {
			if num, rest, ok := strings.Cut(t, ":"); ok {
				if n, err := strconv.Atoi(num); err == nil {
					out.Lines = append(out.Lines, Line{Number: n, Text: rest})
					continue
				}
			}
		}
		if kind == KindJournal && strings.HasPrefix(t, "-- ") {
			continue // "-- No entries --", "-- Boot … --"
		}
		out.Lines = append(out.Lines, Line{Text: t})
	}
	return out
}

// splitLines splits output into lines that are safe to show.
func splitLines(b []byte) []string {
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		l = strings.TrimSuffix(l, "\r")
		if !utf8.ValidString(l) {
			// cut -b may split a character at the cap.
			l = strings.ToValidUTF8(l, "�")
		}
		lines[i] = l
	}
	return lines
}

func exitOf(stderr []byte) int {
	for _, l := range strings.Split(string(stderr), "\n") {
		if code, ok := strings.CutPrefix(l, "exit:"); ok {
			if n, err := strconv.Atoi(code); err == nil {
				return n
			}
		}
	}
	return -1
}

// noJournalGrep tells whether journalctl was built without pattern
// matching, or is too old to know -g or --case-sensitive.
func noJournalGrep(stderr []byte) bool {
	s := string(stderr)
	return strings.Contains(s, "Compiled without pattern matching support") ||
		strings.Contains(s, "unrecognized option") ||
		strings.Contains(s, "invalid option")
}

func journalRestricted(stderr []byte) bool {
	s := string(stderr)
	return strings.Contains(s, "not seeing messages from other users") ||
		strings.Contains(s, "insufficient permissions")
}

// failure turns a tool's stderr into an error the UI can show.
func failure(stderr []byte) error {
	msg := "search failed"
	for _, l := range strings.Split(string(stderr), "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "exit:") {
			msg = strings.ToValidUTF8(l, "�")
			break
		}
	}
	if strings.Contains(msg, "Permission denied") || strings.Contains(msg, "permission denied") {
		return apperr.New(apperr.CmdNeedsRoot, msg)
	}
	return apperr.New(apperr.CmdFailed, msg)
}
