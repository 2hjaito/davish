package applog

import (
	"os"
	"strings"
	"testing"
)

func TestRotateAndTail(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("x", 1023) + "\n"
	for range 6 << 10 { // 6 MiB: one rotation
		w.Write([]byte(line)) //nolint:errcheck,gosec // test
	}
	w.Write([]byte("last\n")) //nolint:errcheck,gosec // test
	if _, err := os.Stat(w.path(1)); err != nil {
		t.Fatal("no rotated file")
	}
	if fi, _ := os.Stat(w.Path()); fi.Size() > maxSize {
		t.Errorf("log grew to %d", fi.Size())
	}
	tail := w.Tail(3)
	if !strings.HasSuffix(tail, "last\n") || strings.Count(tail, "\n") != 3 {
		t.Errorf("tail = %q", tail[len(tail)-20:])
	}
}
