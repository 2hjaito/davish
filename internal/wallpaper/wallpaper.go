// Package wallpaper keeps the one picture a person chose as the desktop
// background. It lives in its own file beside settings.json because a
// setting may hold only 64 KiB (internal/store) and a picture is far larger.
// The frontend has already scaled it down; this package only checks that the
// bytes are a picture it will draw, and stores them.
package wallpaper

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"davish/internal/apperr"
)

const (
	file = "wallpaper.img"
	// MaxBytes bounds what is read, decoded and kept. A 2560px JPEG is
	// around a megabyte; this leaves room for a PNG of a busy photo.
	MaxBytes = 16 << 20
)

// kinds are the picture types accepted. SVG is left out on purpose: it can
// carry scripts.
var kinds = []struct {
	mime  string
	magic func(b []byte) bool
}{
	{"image/jpeg", func(b []byte) bool { return bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}) }},
	{"image/png", func(b []byte) bool { return bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) }},
	{"image/webp", func(b []byte) bool {
		return len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP"
	}},
}

// Store holds the picture in dir.
type Store struct {
	dir string
	mu  sync.Mutex
}

// New returns a Store that keeps its file in dir.
func New(dir string) *Store { return &Store{dir: dir} }

func (s *Store) path() string { return filepath.Join(s.dir, file) }

// mimeOf names the picture type of b, or "" when it is none we accept.
func mimeOf(b []byte) string {
	for _, k := range kinds {
		if k.magic(b) {
			return k.mime
		}
	}
	return ""
}

// Set stores the picture in a data URL (data:image/jpeg;base64,...). The
// type the URL claims must match what the bytes are.
func (s *Store) Set(dataURL string) error {
	head, body, ok := strings.Cut(dataURL, ",")
	if !ok || !strings.HasPrefix(head, "data:") || !strings.HasSuffix(head, ";base64") {
		return apperr.New(apperr.InvalidInput, "wallpaper must be a base64 data URL")
	}
	// The encoded length bounds the decode before it allocates.
	if len(body) > base64.StdEncoding.EncodedLen(MaxBytes) {
		return apperr.New(apperr.InvalidInput, "wallpaper is too large")
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return apperr.Wrap(apperr.InvalidInput, "wallpaper is not valid base64", err)
	}
	mime := mimeOf(raw)
	if mime == "" || strings.TrimSuffix(strings.TrimPrefix(head, "data:"), ";base64") != mime {
		return apperr.New(apperr.InvalidInput, "wallpaper must be a JPEG, PNG or WebP picture")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeAtomic(s.dir, file, raw)
}

// Get returns the stored picture as a data URL, or "" when none was chosen.
func (s *Store) Get() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path())
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	// The file is ours, but a damaged or swapped one is not drawn.
	mime := mimeOf(raw)
	if mime == "" || len(raw) > MaxBytes {
		return "", nil
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

// Clear removes the picture. Clearing when there is none is not an error.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", file, err)
	}
	return nil
}

func writeAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, name+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp %s: %w", name, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	return nil
}
