package disk

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Overview is where a filesystem's space goes, measured in one pass.
type Overview struct {
	Mount Mount `json:"mount"`
	// Measured is what du counted under the mount.
	Measured int64 `json:"measured"`
	// Dirs are the folders down to overviewDepth below the mount, largest
	// first, without the mount itself.
	Dirs []Entry `json:"dirs"`
	// Files are the largest files, largest first.
	Files []Entry `json:"files"`
	// Unreadable are the folders du could not enter; what they hold is in
	// Unmeasured.
	Unreadable []string `json:"unreadable"`
	// Deleted are files removed while a process still holds them open:
	// df counts them, du cannot see them.
	Deleted []DeletedFile `json:"deleted"`
	// Unmeasured is used space neither du nor Deleted account for.
	Unmeasured int64 `json:"unmeasured"`
	// Root is set when it was measured as root, through sudo -n.
	Root bool `json:"root"`
	// Partial is set when the measurement ran out of time.
	Partial bool `json:"partial"`
	// SudoMeasure is a read-only command for the user to run in a terminal
	// with their password, measuring what the user could not read.
	SudoMeasure string `json:"sudoMeasure"`
}

// DeletedFile is a deleted file a process keeps open.
type DeletedFile struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	PID     int    `json:"pid"`
	Command string `json:"command"`
}

const (
	// overviewDepth is how deep below the mount folders are listed.
	overviewDepth = 3
	// minDir and minFile keep the lists to what matters.
	minDir  = 1 << 20
	minFile = 50 << 20
	// maxDirs and maxFiles cap the lists.
	maxDirs  = 3000
	maxFiles = 50
	// overviewTimeout bounds each of du and find on the server.
	overviewTimeout = 90 * time.Second
)

// overviewScript measures the mount in $1. Fixed text: the mount is an
// argument, never part of the script. The C locale keeps du's messages in
// the plain quotes duUnreadable reads.
var overviewScript = `export LC_ALL=C; m="$1"; dev=$(stat -c %d -- "$m")
echo --df--; df -kP -- "$m"
echo --du--; timeout ` + strconv.Itoa(int(overviewTimeout/time.Second)) + ` du -x -d ` + strconv.Itoa(overviewDepth) + ` -k -- "$m" 2>&1; echo "exit:$?"
echo --files--; timeout ` + strconv.Itoa(int(overviewTimeout/time.Second)) + ` find "$m" -xdev -type f -size +` + strconv.Itoa(minFile>>20) + `M -printf '%s\t%p\n' 2>/dev/null | sort -rn | head -n ` + strconv.Itoa(maxFiles) + `
echo --deleted--; find /proc/[0-9]*/fd -lname '*(deleted)' 2>/dev/null | while read -r f; do
  set -- $(stat -L -c '%s %d' -- "$f" 2>/dev/null) || continue
  [ "$2" = "$dev" ] || continue
  p=${f#/proc/}; p=${p%%/*}
  printf '%s\t%s\t%s\t%s\n' "$1" "$p" "$(cat /proc/$p/comm 2>/dev/null)" "$(readlink -- "$f")"
done | sort -rn | head -n 50; true`

// overviewLine runs overviewScript as root when sudo needs no password,
// as the user otherwise, and says which on the first line.
func overviewLine(mount string) string {
	script := shellx.Quote(overviewScript)
	m := shellx.Quote(mount)
	return `if sudo -n true 2>/dev/null; then echo as:root; sudo -n sh -c ` + script + ` sh ` + m +
		`; else echo as:user; sh -c ` + script + ` sh ` + m + `; fi`
}

// Overview measures the filesystem mounted at mount. token names the scan
// for Cancel.
func (s *Service) Overview(ctx context.Context, serverID, mount, token string) (Overview, error) {
	if !path.IsAbs(mount) || strings.ContainsAny(mount, "\n\r") {
		return Overview{}, apperr.New(apperr.InvalidInput, "mount must be an absolute path")
	}
	mount = path.Clean(mount)
	ctx, cancel := context.WithTimeout(ctx, 3*overviewTimeout)
	defer cancel()
	defer s.track(token, cancel)()
	res, err := s.run.RunLine(ctx, serverID, app, overviewLine(mount))
	if err != nil {
		return Overview{}, err
	}
	o := parseOverview(res.Stdout, mount)
	o.SudoMeasure = shellx.Cmd("sudo", "du", "-xh", "-d", "2", "--", mount).String() + ` 2>/dev/null | sort -h | tail -n 25`
	return o, nil
}

// track lets Cancel stop the scan named token until the returned func runs.
func (s *Service) track(token string, cancel context.CancelFunc) func() {
	if token == "" {
		return func() {}
	}
	s.mu.Lock()
	s.scans[token] = cancel
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.scans, token)
		s.mu.Unlock()
	}
}

func parseOverview(out []byte, mount string) Overview {
	o := Overview{Dirs: []Entry{}, Files: []Entry{}, Unreadable: []string{}, Deleted: []DeletedFile{}}
	cur := ""
	var df []byte
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		switch line {
		case "as:root":
			o.Root = true
			continue
		case "--df--", "--du--", "--files--", "--deleted--":
			cur = strings.Trim(line, "-")
			continue
		}
		switch cur {
		case "df":
			df = append(append(df, line...), '\n')
		case "du":
			if code, ok := strings.CutPrefix(line, "exit:"); ok {
				o.Partial = code == "124"
				continue
			}
			if dir, ok := duUnreadable(line); ok {
				o.Unreadable = append(o.Unreadable, dir)
				continue
			}
			kb, p, ok := strings.Cut(line, "\t")
			n, err := strconv.ParseInt(kb, 10, 64)
			if !ok || err != nil {
				continue
			}
			if path.Clean(p) == mount {
				o.Measured = n << 10
			} else if n<<10 >= minDir && len(o.Dirs) < maxDirs*4 {
				o.Dirs = append(o.Dirs, Entry{Path: p, Size: n << 10})
			}
		case "files":
			n, p, ok := strings.Cut(line, "\t")
			b, err := strconv.ParseInt(n, 10, 64)
			if ok && err == nil {
				o.Files = append(o.Files, Entry{Path: p, Size: b})
			}
		case "deleted":
			f := strings.SplitN(line, "\t", 4)
			if len(f) != 4 {
				continue
			}
			b, err1 := strconv.ParseInt(f[0], 10, 64)
			pid, err2 := strconv.Atoi(f[1])
			if err1 != nil || err2 != nil {
				continue
			}
			o.Deleted = append(o.Deleted, DeletedFile{Size: b, PID: pid, Command: f[2], Path: strings.TrimSuffix(f[3], " (deleted)")})
		}
	}
	if ms := parseDF(df); len(ms) > 0 {
		o.Mount = ms[0]
	}
	bySize := func(a, b Entry) int {
		if c := cmp.Compare(b.Size, a.Size); c != 0 {
			return c
		}
		return strings.Compare(a.Path, b.Path)
	}
	slices.SortFunc(o.Dirs, bySize)
	o.Dirs = o.Dirs[:min(len(o.Dirs), maxDirs)]
	slices.SortFunc(o.Files, bySize)
	var held int64
	for _, d := range o.Deleted {
		held += d.Size
	}
	o.Unmeasured = max(0, o.Mount.Used-o.Measured-held)
	return o
}

// duUnreadable reads du's "cannot read directory '/x': Permission denied"
// (or "cannot open"), as GNU du prints them.
func duUnreadable(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "du: cannot ")
	if !ok {
		return "", false
	}
	_, rest, ok = strings.Cut(rest, " '")
	if !ok {
		return "", false
	}
	dir, _, ok := strings.Cut(rest, "': ")
	return dir, ok
}
