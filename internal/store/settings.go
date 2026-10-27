package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"davish/internal/apperr"
)

const (
	settingsFile    = "settings.json"
	settingsVersion = 1
	// maxSettingValue bounds one key; UI state is small, and the whole file
	// is rewritten on every change.
	maxSettingValue = 64 << 10
)

var settingKey = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// settingsLayout is the on-disk layout of settings.json.
type settingsLayout struct {
	Version int                        `json:"version"`
	UI      map[string]json.RawMessage `json:"ui"`
}

// Settings keeps the frontend's own state in settings.json: window
// geometry, sort orders, toggles. Values are opaque JSON per key; Go only
// stores them. Nothing secret belongs here.
type Settings struct {
	dir string
	log *slog.Logger

	// mu guards ui, the file's content once read.
	mu sync.Mutex
	ui map[string]json.RawMessage
}

// NewSettings keeps settings.json in dir.
func NewSettings(dir string, log *slog.Logger) *Settings {
	return &Settings{dir: dir, log: log}
}

// All returns every stored value by key, each as JSON text.
func (s *Settings) All() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(s.ui))
	for k, v := range s.ui {
		out[k] = string(v)
	}
	return out, nil
}

// Set stores value, JSON text, under key and rewrites the file.
func (s *Settings) Set(key, value string) error {
	if !settingKey.MatchString(key) {
		return apperr.New(apperr.InvalidInput, "invalid setting key")
	}
	if len(value) > maxSettingValue || !json.Valid([]byte(value)) {
		return apperr.New(apperr.InvalidInput, "setting value must be JSON under 64 KiB")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(value)); err != nil {
		return apperr.Wrap(apperr.InvalidInput, "setting value must be JSON", err)
	}
	s.ui[key] = buf.Bytes()
	data, err := json.MarshalIndent(settingsLayout{Version: settingsVersion, UI: s.ui}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", settingsFile, err)
	}
	return writeAtomic(s.dir, settingsFile, data)
}

// loadLocked reads the file once. A damaged file only holds UI
// preferences, so it is set aside and the app starts from defaults
// rather than refusing to start.
func (s *Settings) loadLocked() error {
	if s.ui != nil {
		return nil
	}
	s.ui = map[string]json.RawMessage{}
	p := filepath.Join(s.dir, settingsFile)
	data, err := os.ReadFile(p) //nolint:gosec // our own config file
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		s.ui = nil
		return fmt.Errorf("read %s: %w", settingsFile, err)
	}
	var f settingsLayout
	if err := json.Unmarshal(data, &f); err != nil || f.Version > settingsVersion {
		aside := p + ".broken-" + strconv.FormatInt(time.Now().Unix(), 10)
		s.log.Warn("settings.json unreadable, starting from defaults", "op", "settings", "err", err)
		if rerr := os.Rename(p, aside); rerr != nil {
			s.log.Warn("could not set settings.json aside", "op", "settings", "err", rerr)
		}
		return nil
	}
	// The file is indented for people; hand values out compact.
	for k, v := range f.UI {
		var buf bytes.Buffer
		if err := json.Compact(&buf, v); err == nil {
			s.ui[k] = buf.Bytes()
		}
	}
	return nil
}
