package perf

import (
	"os"
	"path/filepath"
	"strings"
)

// GPU policies for the Linux webview, as WebKitGTK names them. The caller
// maps them to Wails' linux.WebviewGpuPolicy; this package does not import
// Wails so it stays testable without cgo.
const (
	GPUAlways   = "always"
	GPUOnDemand = "ondemand"
	GPUNever    = "never"
)

// GPUEnv is the variable a user sets to pick the policy by hand, for a
// machine the automatic choice gets wrong (docs/adr/0023).
const GPUEnv = "DAVISH_WEBVIEW_GPU"

// dmabufEnv is WebKitGTK's switch for the DMA-BUF renderer, which leaves
// the window blank on the proprietary NVIDIA driver (wails#2977).
const dmabufEnv = "WEBKIT_DISABLE_DMABUF_RENDERER"

// nvidiaVendor is NVIDIA's PCI vendor ID, as sysfs writes it.
const nvidiaVendor = "0x10de"

// GPUDecision is the policy chosen for this launch and why, for the log.
type GPUDecision struct {
	Policy string
	Reason string
}

// Env is the part of the process environment DecideGPU reads and writes.
type Env interface {
	LookupEnv(key string) (string, bool)
	Setenv(key, value string) error
}

type osEnv struct{}

func (osEnv) LookupEnv(k string) (string, bool) { return os.LookupEnv(k) }
func (osEnv) Setenv(k, v string) error          { return os.Setenv(k, v) }

// OSEnv is the real process environment.
var OSEnv Env = osEnv{}

// DecideGPU picks the webview's GPU policy on Linux, before GTK starts
// (docs/adr/0023):
//
//   - DAVISH_WEBVIEW_GPU=never|ondemand|always wins, as the way out for a
//     machine this gets wrong.
//   - Otherwise Always, WebKitGTK's own default: GPU compositing, which
//     WebKitGTK drops to software by itself when it cannot get GL.
//   - On the proprietary NVIDIA driver, Never, as before: that is where
//     the blank window behind wails#2977 comes from, and nobody has tried
//     the usual fix (WEBKIT_DISABLE_DMABUF_RENDERER=1 with the GPU on) on
//     Davish yet. DAVISH_WEBVIEW_GPU=always tries it, and then the
//     workaround is set unless the variable is already set.
//
// drm is the sysfs DRM folder, /sys/class/drm on a real system.
func DecideGPU(drm string, env Env) GPUDecision {
	if v, ok := env.LookupEnv(GPUEnv); ok {
		switch p := strings.ToLower(strings.TrimSpace(v)); p {
		case GPUAlways, GPUOnDemand:
			d := GPUDecision{Policy: p, Reason: GPUEnv + " is set"}
			if nvidiaProprietary(drm) {
				d.Reason += "; " + dmabufWorkaround(env)
			}
			return d
		case GPUNever:
			return GPUDecision{Policy: p, Reason: GPUEnv + " is set"}
		}
		// A typo must not take the GPU away silently; say so and go on.
		d := DecideGPU(drm, withoutKey{env, GPUEnv})
		d.Reason = GPUEnv + " has an unknown value, ignored; " + d.Reason
		return d
	}
	if !nvidiaProprietary(drm) {
		return GPUDecision{Policy: GPUAlways, Reason: "default"}
	}
	return GPUDecision{
		Policy: GPUNever,
		Reason: "NVIDIA driver: GPU off until it is tested (" + GPUEnv + "=always to try it)",
	}
}

// dmabufWorkaround sets WEBKIT_DISABLE_DMABUF_RENDERER=1 for the NVIDIA
// driver unless the variable is set already (to anything, 0 included), and
// says what it did.
func dmabufWorkaround(env Env) string {
	if v, ok := env.LookupEnv(dmabufEnv); ok {
		return "NVIDIA driver, " + dmabufEnv + "=" + v + " kept as set"
	}
	if err := env.Setenv(dmabufEnv, "1"); err != nil {
		return "NVIDIA driver, could not set " + dmabufEnv + ": " + err.Error()
	}
	return "NVIDIA driver, " + dmabufEnv + "=1 set"
}

// withoutKey hides one variable, to decide again as if it were unset.
type withoutKey struct {
	Env
	key string
}

func (w withoutKey) LookupEnv(k string) (string, bool) {
	if k == w.key {
		return "", false
	}
	return w.Env.LookupEnv(k)
}

// nvidiaProprietary reports whether a DRM card is an NVIDIA device bound
// to the proprietary nvidia driver (not nouveau, which does not have the
// problem). Any read error counts as no.
func nvidiaProprietary(drm string) bool {
	cards, err := filepath.Glob(filepath.Join(drm, "card[0-9]*"))
	if err != nil {
		return false
	}
	for _, c := range cards {
		if strings.Contains(filepath.Base(c), "-") {
			continue // a connector such as card0-HDMI-A-1, not a device
		}
		vendor, err := os.ReadFile(filepath.Join(c, "device", "vendor")) //nolint:gosec // sysfs, read only
		if err != nil || strings.TrimSpace(string(vendor)) != nvidiaVendor {
			continue
		}
		driver, err := filepath.EvalSymlinks(filepath.Join(c, "device", "driver"))
		if err == nil && filepath.Base(driver) == "nvidia" {
			return true
		}
	}
	return false
}
