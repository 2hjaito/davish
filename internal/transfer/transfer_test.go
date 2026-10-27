package transfer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/events"
)

func TestNumbered(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"a.txt", 1, "a (1).txt"},
		{"backup.tar.gz", 2, "backup (2).tar.gz"},
		{"BACKUP.TAR.XZ", 1, "BACKUP (1).TAR.XZ"},
		{".bashrc", 1, ".bashrc (1)"},
		{".env.local", 1, ".env (1).local"},
		{"Makefile", 3, "Makefile (3)"},
		{"dir.d", 1, "dir (1).d"},
		{".tar.gz", 1, ".tar (1).gz"},
	}
	for _, tt := range tests {
		if got := numbered(tt.name, tt.n); got != tt.want {
			t.Errorf("numbered(%q, %d) = %q; want %q", tt.name, tt.n, got, tt.want)
		}
	}
}

func TestFreeName(t *testing.T) {
	taken := map[string]bool{"a.txt": true, "a (1).txt": true}
	got, err := freeName("a.txt", func(n string) (bool, error) { return taken[n], nil })
	if err != nil || got != "a (2).txt" {
		t.Errorf("freeName = %q, %v", got, err)
	}
	got, err = freeName("b.txt", func(n string) (bool, error) { return taken[n], nil })
	if err != nil || got != "b.txt" {
		t.Errorf("freeName = %q, %v", got, err)
	}
}

func TestSafeName(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "a/b", `a\b`, "a\x00b"} {
		if safeName(bad) == nil {
			t.Errorf("safeName(%q) accepted", bad)
		}
	}
	for _, good := range []string{"a", ".hidden", "..x", "a b", "tên.txt"} {
		if err := safeName(good); err != nil {
			t.Errorf("safeName(%q) = %v", good, err)
		}
	}
}

type progressLog struct {
	mu  sync.Mutex
	all []Progress
}

func (p *progressLog) Emit(name string, payload any) {
	if pr, ok := payload.(Progress); ok && name == events.TransferProgress {
		p.mu.Lock()
		p.all = append(p.all, pr)
		p.mu.Unlock()
	}
}

func TestMeterThrottlesAndFlushes(t *testing.T) {
	log := &progressLog{}
	m := newMeter(context.Background(), "j", log, time.Hour)
	m.setTotal(300)
	for range 3 {
		if err := m.add(100); err != nil {
			t.Fatal(err)
		}
	}
	if len(log.all) != 1 {
		t.Fatalf("events before flush = %d; want only the total", len(log.all))
	}
	m.flush()
	last := log.all[len(log.all)-1]
	if last.Done != 300 || last.Total != 300 || last.JobID != "j" {
		t.Errorf("final progress = %+v", last)
	}
}

func TestMeterStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := newMeter(ctx, "j", &progressLog{}, time.Hour)
	cancel()
	if m.add(1) == nil {
		t.Error("add after cancel succeeded")
	}
}

func TestRevealDownloadStaysInDownloads(t *testing.T) {
	s, err := New(nil, &progressLog{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{DownloadDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"relative",
		s.opts.DownloadDir,
		filepath.Join(s.opts.DownloadDir, ".."),
		filepath.Join(s.opts.DownloadDir, "..", "elsewhere"),
		"/etc/passwd",
	} {
		var ae *apperr.Error
		if err := s.RevealDownload(context.Background(), p); !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
			t.Errorf("RevealDownload(%q) = %v; want INVALID_INPUT", p, err)
		}
	}
	var ae *apperr.Error
	missing := filepath.Join(s.opts.DownloadDir, "missing.txt")
	if err := s.RevealDownload(context.Background(), missing); !errors.As(err, &ae) || ae.Code != apperr.NotFound {
		t.Errorf("RevealDownload(missing) = %v; want NOT_FOUND", err)
	}
}
