// Package applog writes the app's log as JSON to rotating files under
// os.UserCacheDir()/davish/logs (docs/CONVENTIONS.md), so a user can send
// what happened to whoever supports them. Secrets are never logged.
package applog

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	fileName = "davish.log"
	maxSize  = 5 << 20
	keep     = 5 // davish.log plus davish.log.1 … .4
)

// Writer appends to dir/davish.log and rotates it past maxSize.
type Writer struct {
	dir string

	// mu guards f and size.
	mu   sync.Mutex
	f    *os.File
	size int64
}

// DefaultDir is os.UserCacheDir()/davish/logs.
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate cache dir: %w", err)
	}
	return filepath.Join(base, "davish", "logs"), nil
}

// Open starts writing to dir.
func Open(dir string) (*Writer, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	w := &Writer{dir: dir}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) path(n int) string {
	if n == 0 {
		return filepath.Join(w.dir, fileName)
	}
	return filepath.Join(w.dir, fileName+"."+strconv.Itoa(n))
}

func (w *Writer) open() error {
	f, err := os.OpenFile(w.path(0), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close() //nolint:errcheck,gosec // failing
		return err
	}
	w.f, w.size = f, fi.Size()
	return nil
}

// Write appends p, rotating first when the file is full.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+int64(len(p)) > maxSize {
		w.rotate()
	}
	n, err := w.f.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *Writer) rotate() {
	w.f.Close() //nolint:errcheck,gosec // replaced below
	for i := keep - 1; i > 0; i-- {
		os.Rename(w.path(i-1), w.path(i)) //nolint:errcheck,gosec // a missing one is fine
	}
	if err := w.open(); err != nil {
		// Keep logging somewhere rather than nowhere.
		w.f, w.size = os.Stderr, 0
	}
}

// Path is the current log file.
func (w *Writer) Path() string { return w.path(0) }

// Tail returns about the last n lines of the log, for a support message.
func (w *Writer) Tail(n int) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	f, err := os.Open(w.path(0))
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // read-only
	fi, err := f.Stat()
	if err != nil {
		return ""
	}
	// The last 256 KiB hold far more than n lines.
	if off := fi.Size() - 256<<10; off > 0 {
		f.Seek(off, io.SeekStart) //nolint:errcheck,gosec // reading from the start is fine too
	}
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// New returns a logger writing JSON to w and text to stderr.
func New(w io.Writer) *slog.Logger {
	return slog.New(fanout{
		slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		slog.NewTextHandler(os.Stderr, nil),
	})
}
