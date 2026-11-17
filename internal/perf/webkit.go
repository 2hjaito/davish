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
// does nothing. A switch that is on sets its variable; one that is off
// leaves the environment alone, so a variable the user exported (a common
// workaround for a blank window on some GPUs) still applies.
func Apply(dir string) {
	if runtime.GOOS != "linux" {
		return
	}
	apply(Read(dir))
}

// apply sets the variable of each switch that is on.
func apply(flags Flags) {
	if flags.NoDmabuf {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	if flags.NoCompositing {
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
}
