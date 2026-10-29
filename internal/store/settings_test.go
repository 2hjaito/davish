package store

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"davish/internal/apperr"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewSettings(dir, quietLog())
	all, err := s.All()
	if err != nil || len(all) != 0 {
		t.Fatalf("empty All = %v, %v", all, err)
	}
	if err := s.Set("windows", `{"files":{"x":1}}`); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("files", `{"order":"name"}`); err != nil {
		t.Fatal(err)
	}

	// A fresh instance reads what the first one wrote.
	all, err = NewSettings(dir, quietLog()).All()
	if err != nil {
		t.Fatal(err)
	}
	if all["windows"] != `{"files":{"x":1}}` || all["files"] != `{"order":"name"}` {
		t.Errorf("All = %s", all)
	}
	data, _ := os.ReadFile(filepath.Join(dir, settingsFile)) //nolint:gosec // test temp dir
	if !strings.Contains(string(data), `"version": 1`) {
		t.Errorf("file = %s", data)
	}
}

func TestSettingsRejectsBadInput(t *testing.T) {
	s := NewSettings(t.TempDir(), quietLog())
	tests := []struct {
		key   string
		value string
	}{
		{"", `1`},
		{"../x", `1`},
		{"ok", `{not json`},
		{"ok", `"` + strings.Repeat("a", maxSettingValue) + `"`},
	}
	for _, tt := range tests {
		var ae *apperr.Error
		if err := s.Set(tt.key, tt.value); !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
			t.Errorf("Set(%q, %.20q) = %v", tt.key, tt.value, err)
		}
	}
}

func TestSettingsSetsBrokenFileAside(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, settingsFile)
	if err := os.WriteFile(p, []byte("{oops"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewSettings(dir, quietLog())
	all, err := s.All()
	if err != nil || len(all) != 0 {
		t.Fatalf("All = %v, %v", all, err)
	}
	matches, _ := filepath.Glob(p + ".broken-*")
	if len(matches) != 1 {
		t.Errorf("broken file not kept aside: %v", matches)
	}
	if err := s.Set("k", `true`); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsKeys(t *testing.T) {
	s := NewSettings(t.TempDir(), quietLog())
	for _, k := range []string{"dock:pinned", "logs:srv-1", "update:skip"} {
		if err := s.Set(k, `1`); err != nil {
			t.Errorf("%s: %v", k, err)
		}
	}
	for _, k := range []string{"", "a/b", "a b", strings.Repeat("x", 65)} {
		if err := s.Set(k, `1`); err == nil {
			t.Errorf("%q accepted", k)
		}
	}
}
