// Package disk finds what fills a server's disk: the size of one folder's
// children at a time, and cleanups with the space they would free.
package disk

import (
	"bufio"
	"bytes"
	"context"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "disk"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Service measures folders and cleans up.
type Service struct {
	run Runner

	mu sync.Mutex
	// scans holds the cancel of each running measurement, by the token
	// the frontend chose, so its Cancel button can stop it.
	scans map[string]context.CancelFunc
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run, scans: map[string]context.CancelFunc{}}
}

// Mount is one mounted filesystem.
type Mount struct {
	Path  string `json:"path"`
	Total int64  `json:"total"` // bytes
	Used  int64  `json:"used"`
}

// mountsLine lists real filesystems in POSIX format (1024-byte blocks).
const mountsLine = `df -kP -x tmpfs -x devtmpfs -x overlay -x squashfs -x efivarfs 2>/dev/null; true`

// Mounts lists the server's filesystems.
func (s *Service) Mounts(ctx context.Context, serverID string) ([]Mount, error) {
	res, err := s.run.RunLine(ctx, serverID, app, mountsLine)
	if err != nil {
		return nil, err
	}
	return parseDF(res.Stdout), nil
}

func parseDF(out []byte) []Mount {
	var ms []Mount
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 6 || f[0] == "Filesystem" {
			continue
		}
		total, err1 := strconv.ParseInt(f[1], 10, 64)
		used, err2 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil || total == 0 {
			continue
		}
		ms = append(ms, Mount{Path: strings.Join(f[5:], " "), Total: total << 10, Used: used << 10})
	}
	return ms
}

// Entry is one child of a measured folder.
type Entry struct {
	Path string `json:"path"`
	Size int64  `json:"size"` // bytes
}

// Usage is a measured folder.
type Usage struct {
	Dir     string  `json:"dir"`
	Size    int64   `json:"size"`
	Entries []Entry `json:"entries"`
	// Partial is set when du ran out of time; sizes are what it got to.
	Partial bool `json:"partial"`
}

// scanTimeout bounds one measurement on the server.
const scanTimeout = 30 * time.Second

// usageLine measures dir's direct children on its own filesystem.
// Unreadable folders are skipped; the exit status tells a timeout apart.
func usageLine(dir string) string {
	return shellx.Cmd("timeout", strconv.Itoa(int(scanTimeout/time.Second)), "du", "-x", "-d", "1", "-k", "--", dir).String() +
		` 2>/dev/null; echo "exit:$?"`
}

// Usage measures dir's children. token names the scan for Cancel.
func (s *Service) Usage(ctx context.Context, serverID, dir, token string) (Usage, error) {
	if !path.IsAbs(dir) || strings.ContainsAny(dir, "\n\r") {
		return Usage{}, apperr.New(apperr.InvalidInput, "folder must be an absolute path")
	}
	dir = path.Clean(dir)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if token != "" {
		s.mu.Lock()
		s.scans[token] = cancel
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			delete(s.scans, token)
			s.mu.Unlock()
		}()
	}
	res, err := s.run.RunLine(ctx, serverID, app, usageLine(dir))
	if err != nil {
		return Usage{}, err
	}
	return parseDU(res.Stdout, dir), nil
}

// Cancel stops a running Usage; the server's du ends with its session.
func (s *Service) Cancel(token string) {
	s.mu.Lock()
	cancel := s.scans[token]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func parseDU(out []byte, dir string) Usage {
	u := Usage{Dir: dir}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if code, ok := strings.CutPrefix(line, "exit:"); ok {
			u.Partial = code == "124"
			continue
		}
		kb, p, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(kb, 10, 64)
		if err != nil {
			continue
		}
		if path.Clean(p) == dir {
			u.Size = n << 10
			continue
		}
		u.Entries = append(u.Entries, Entry{Path: p, Size: n << 10})
	}
	slices.SortFunc(u.Entries, func(a, b Entry) int {
		switch {
		case a.Size > b.Size:
			return -1
		case a.Size < b.Size:
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return u
}
