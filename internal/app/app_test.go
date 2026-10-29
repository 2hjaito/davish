package app

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"davish/internal/apperr"
)

func TestErrorFormatterLogsUnclassified(t *testing.T) {
	var buf bytes.Buffer
	format := ErrorFormatter(&App{log: slog.New(slog.NewTextHandler(&buf, nil))})

	got := format(errors.New("disk on fire"))
	if ae, ok := got.(*apperr.Error); !ok || ae.Code != apperr.Internal {
		t.Fatalf("got %#v, want an Internal apperr", got)
	}
	if !strings.Contains(buf.String(), "disk on fire") {
		t.Fatalf("log %q lacks the cause", buf.String())
	}

	buf.Reset()
	format(apperr.New(apperr.InvalidInput, "bad"))
	if buf.Len() != 0 {
		t.Fatalf("logged a classified error: %q", buf.String())
	}
}
