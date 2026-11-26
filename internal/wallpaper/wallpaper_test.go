package wallpaper

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"davish/internal/apperr"
)

var (
	jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F'}
	pngBytes  = []byte("\x89PNG\r\n\x1a\nrest")
	webpBytes = []byte("RIFF\x10\x00\x00\x00WEBPVP8 ")
)

func url(mime string, b []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func TestSetGetClear(t *testing.T) {
	for _, c := range []struct {
		mime string
		b    []byte
	}{{"image/jpeg", jpegBytes}, {"image/png", pngBytes}, {"image/webp", webpBytes}} {
		s := New(t.TempDir())
		if got, err := s.Get(); err != nil || got != "" {
			t.Fatalf("empty store: %q, %v", got, err)
		}
		if err := s.Set(url(c.mime, c.b)); err != nil {
			t.Fatalf("%s: set: %v", c.mime, err)
		}
		got, err := s.Get()
		if err != nil || got != url(c.mime, c.b) {
			t.Fatalf("%s: get = %q, %v", c.mime, got, err)
		}
		if err := s.Clear(); err != nil {
			t.Fatal(err)
		}
		if got, _ := s.Get(); got != "" {
			t.Fatalf("%s: still there after clear", c.mime)
		}
		if err := s.Clear(); err != nil {
			t.Fatalf("clearing twice: %v", err)
		}
	}
}

func TestSetRejects(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	cases := map[string]string{
		"not a data url":          "https://example.com/a.png",
		"not base64":              "data:image/png,abc",
		"bad base64":              "data:image/png;base64,@@@",
		"svg":                     url("image/svg+xml", svg),
		"svg claiming png":        url("image/png", svg),
		"jpeg claiming png":       url("image/png", jpegBytes),
		"text":                    url("image/png", []byte("hello")),
		"empty":                   url("image/png", nil),
		"too large before decode": "data:image/png;base64," + strings.Repeat("A", MaxBytes*2),
	}
	for name, in := range cases {
		s := New(t.TempDir())
		var ae *apperr.Error
		err := s.Set(in)
		if !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
			t.Errorf("%s: want InvalidInput, got %v", name, err)
		}
		if got, _ := s.Get(); got != "" {
			t.Errorf("%s: stored anyway", name)
		}
	}
}

func TestSetReplacesAndKeepsNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Set(url("image/png", pngBytes)); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(url("image/jpeg", jpegBytes)); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(); got != url("image/jpeg", jpegBytes) {
		t.Fatalf("second picture not kept: %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != file {
		t.Fatalf("leftovers in dir: %v", entries)
	}
}

func TestGetIgnoresADamagedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, file), []byte("not a picture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := New(dir).Get(); err != nil || got != "" {
		t.Fatalf("damaged file drawn: %q, %v", got, err)
	}
}
