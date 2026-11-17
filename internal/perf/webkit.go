// Package perf applies Linux WebKit workarounds before the webview starts.
// The variables are read when WebKit creates its rendering, so a change in
// Settings takes effect on the next launch.
package perf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const settingsFile = "settings.json"

// Flags are the WebKit workarounds stored with the UI settings.
type Flags struct {
	NoDmabuf      bool
	NoCompositing bool
}

// Read loads the flags from settings.json in dir. A missing or damaged file
// means both stay off: startup must not fail over a preference.
func Read(dir string) Flags {
	data, err := os.ReadFile(filepath.Join(dir, settingsFile)) //nolint:gosec // our own config file
	if err != nil {
		return Flags{}
	}
	var doc struct {
		UI map[string]json.RawMessage `json:"ui"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Flags{}
	}
	raw, ok := doc.UI["settings"]
	if !ok {
		return Flags{}
	}
	var body struct {
		NoDmabuf      bool `json:"perfNoDmabuf"`
		NoCompositing bool `json:"perfNoCompositing"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Flags{}
	}
	return Flags{NoDmabuf: body.NoDmabuf, NoCompositing: body.NoCompositing}
}

// Apply sets the WebKit environment from dir on Linux. On other systems it
// does nothing. The in-app switches are the source of truth: a flag that is
// off clears the variable for this process.
func Apply(dir string) {
	if runtime.GOOS != "linux" {
		return
	}
	flags := Read(dir)
	set("WEBKIT_DISABLE_DMABUF_RENDERER", flags.NoDmabuf)
	set("WEBKIT_DISABLE_COMPOSITING_MODE", flags.NoCompositing)
}

func set(key string, on bool) {
	if on {
		_ = os.Setenv(key, "1")
		return
	}
	_ = os.Unsetenv(key)
}
