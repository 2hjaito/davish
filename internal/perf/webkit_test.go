package perf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFlags(t *testing.T) {
	dir := t.TempDir()
	body := `{
	  "version": 1,
	  "ui": {
	    "settings": {"perfNoDmabuf": true, "perfNoCompositing": false, "theme": "system"}
	  }
	}`
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Read(dir)
	if !got.NoDmabuf || got.NoCompositing {
		t.Fatalf("Read = %+v", got)
	}
}

func TestReadMissingOrDamaged(t *testing.T) {
	if got := Read(t.TempDir()); got != (Flags{}) {
		t.Fatalf("missing = %+v", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Read(dir); got != (Flags{}) {
		t.Fatalf("damaged = %+v", got)
	}
}

func TestApplySetsEnv(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "")
	t.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	// apply, not Apply: Apply only acts on Linux, and this runs everywhere.
	apply(Flags{NoDmabuf: true, NoCompositing: false})
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") != "1" {
		t.Fatalf("dmabuf = %q", os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"))
	}
	// Off leaves a variable the user exported as it was.
	if os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE") != "1" {
		t.Fatalf("compositing = %q, want the user's 1 kept", os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE"))
	}
}
