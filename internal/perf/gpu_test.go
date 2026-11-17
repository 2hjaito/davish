package perf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeEnv is a process environment in a map.
type fakeEnv struct {
	vars map[string]string
	fail bool
}

func (e *fakeEnv) LookupEnv(k string) (string, bool) {
	v, ok := e.vars[k]
	return v, ok
}

func (e *fakeEnv) Setenv(k, v string) error {
	if e.fail {
		return errors.New("setenv failed")
	}
	e.vars[k] = v
	return nil
}

// card adds a DRM card with a PCI vendor and the driver it is bound to.
func card(t *testing.T, drm, name, vendor, driver string) {
	t.Helper()
	dev := filepath.Join(drm, name, "device")
	if err := os.MkdirAll(dev, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dev, "vendor"), []byte(vendor+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(drm, "..", "drivers", driver)
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dev, "driver")); err != nil {
		t.Fatal(err)
	}
}

func TestDecideGPU(t *testing.T) {
	type c struct{ name, vendor, driver string }
	tests := []struct {
		name       string
		cards      []c
		env        map[string]string
		failSet    bool
		wantPolicy string
		wantDmabuf string // "" when it must stay unset
		wantReason string
	}{
		{name: "no cards (VM, container)", wantPolicy: GPUAlways, wantReason: "default"},
		{name: "Intel", cards: []c{{"card0", "0x8086", "i915"}}, wantPolicy: GPUAlways, wantReason: "default"},
		{name: "AMD", cards: []c{{"card0", "0x1002", "amdgpu"}}, wantPolicy: GPUAlways, wantReason: "default"},
		{name: "NVIDIA on nouveau", cards: []c{{"card0", "0x10de", "nouveau"}}, wantPolicy: GPUAlways, wantReason: "default"},
		{
			name: "NVIDIA proprietary", cards: []c{{"card0", "0x10de", "nvidia"}},
			wantPolicy: GPUAlways, wantDmabuf: "1", wantReason: "=1 set",
		},
		{
			name: "hybrid laptop, NVIDIA second", cards: []c{{"card0", "0x8086", "i915"}, {"card1", "0x10de", "nvidia"}},
			wantPolicy: GPUAlways, wantDmabuf: "1", wantReason: "=1 set",
		},
		{
			name: "user's 0 is kept", cards: []c{{"card0", "0x10de", "nvidia"}},
			env: map[string]string{dmabufEnv: "0"}, wantPolicy: GPUAlways, wantDmabuf: "0", wantReason: "kept as set",
		},
		{
			name: "cannot set the workaround", cards: []c{{"card0", "0x10de", "nvidia"}},
			failSet: true, wantPolicy: GPUNever, wantReason: "could not set",
		},
		{name: "override never", env: map[string]string{GPUEnv: "never"}, wantPolicy: GPUNever, wantReason: "is set"},
		{name: "override ondemand", env: map[string]string{GPUEnv: " OnDemand "}, wantPolicy: GPUOnDemand, wantReason: "is set"},
		{
			name: "override always on NVIDIA leaves the env alone", cards: []c{{"card0", "0x10de", "nvidia"}},
			env: map[string]string{GPUEnv: "always"}, wantPolicy: GPUAlways, wantReason: "is set",
		},
		{
			name: "garbage override is ignored, said so", env: map[string]string{GPUEnv: "fast"},
			wantPolicy: GPUAlways, wantReason: "unknown value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drm := filepath.Join(t.TempDir(), "class", "drm")
			if err := os.MkdirAll(drm, 0o750); err != nil {
				t.Fatal(err)
			}
			// A connector entry is never read as a device.
			if err := os.MkdirAll(filepath.Join(drm, "card0-HDMI-A-1"), 0o750); err != nil {
				t.Fatal(err)
			}
			for _, cd := range tt.cards {
				card(t, drm, cd.name, cd.vendor, cd.driver)
			}
			env := &fakeEnv{vars: map[string]string{}, fail: tt.failSet}
			for k, v := range tt.env {
				env.vars[k] = v
			}
			got := DecideGPU(drm, env)
			if got.Policy != tt.wantPolicy || !strings.Contains(got.Reason, tt.wantReason) {
				t.Errorf("got %+v, want %s with %q", got, tt.wantPolicy, tt.wantReason)
			}
			if v := env.vars[dmabufEnv]; v != tt.wantDmabuf {
				t.Errorf("%s = %q, want %q", dmabufEnv, v, tt.wantDmabuf)
			}
		})
	}
}

func TestDecideGPUMissingSysfs(t *testing.T) {
	got := DecideGPU(filepath.Join(t.TempDir(), "nope"), &fakeEnv{vars: map[string]string{}})
	if got.Policy != GPUAlways {
		t.Errorf("got %+v", got)
	}
}
