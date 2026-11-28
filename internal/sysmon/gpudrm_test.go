package sysmon

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseDRMGPUs(t *testing.T) {
	lines := []string{
		"0000:03:00.0|0x1002|amdgpu|37|1073741824|8589934592|52000|75000000|",
		"0000:00:02.0|0x8086|i915||||||",
		"0000:05:00.0|0x1002|amdgpu|||||| Radeon Pro W6800 ",
		"broken line",
	}
	names := []string{
		`0000:03:00.0 "VGA compatible controller" "Advanced Micro Devices, Inc. [AMD/ATI]" "Ellesmere [Radeon RX 470/480/570/580]" -ref1 "Sapphire" "Nitro"`,
		`0000:00:02.0 "VGA compatible controller" "Intel Corporation" "CometLake-U GT2 [UHD Graphics]" -p00`,
	}
	got := parseDRMGPUs(lines, names, 2)
	if len(got) != 3 {
		t.Fatalf("got %d GPUs: %+v", len(got), got)
	}
	amd := got[0]
	if amd.Index != 2 || amd.Name != "AMD Radeon RX 470/480/570/580" || amd.Util != 37 ||
		amd.MemUsed != 1<<30 || amd.MemTotal != 8<<30 || amd.Temp != 52 || amd.Power != 75 {
		t.Errorf("amd = %+v", amd)
	}
	intel := got[1]
	if intel.Index != 3 || intel.Name != "Intel UHD Graphics" || intel.Util != -1 || intel.MemTotal != -1 || intel.Temp != -1 || intel.Power != -1 {
		t.Errorf("intel = %+v", intel)
	}
	if got[2].Name != "Radeon Pro W6800" {
		t.Errorf("a product name from sysfs wins over a guess: %q", got[2].Name)
	}
}

func TestParseDRMGPUsWithoutNames(t *testing.T) {
	got := parseDRMGPUs([]string{"0000:03:00.0|0x1002|amdgpu|||||| "}, nil, 0)
	if len(got) != 1 || got[0].Name != "AMD GPU (amdgpu)" {
		t.Errorf("got %+v", got)
	}
}

func TestNextGPUIndex(t *testing.T) {
	if nextGPUIndex(nil) != 0 || nextGPUIndex([]GPU{{Index: 0}, {Index: 3}}) != 4 {
		t.Error("next index must follow the highest one")
	}
}

// fakeDRM builds the files an amdgpu card and an Intel card show, plus a
// connector and a render node that the script must skip.
func fakeDRM(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil { //nolint:gosec // fake sysfs
			t.Fatal(err)
		}
	}
	write("card0/device/vendor", "0x1002\n")
	write("card0/device/uevent", "DRIVER=amdgpu\nPCI_ID=1002:6FDF\nPCI_SLOT_NAME=0000:03:00.0\n")
	write("card0/device/gpu_busy_percent", "37\n")
	write("card0/device/mem_info_vram_used", "1073741824\n")
	write("card0/device/mem_info_vram_total", "8589934592\n")
	write("card0/device/hwmon/hwmon3/temp1_input", "52000\n")
	write("card0/device/hwmon/hwmon3/power1_average", "75000000\n")
	write("card1/device/vendor", "0x8086\n")
	write("card1/device/uevent", "DRIVER=i915\nPCI_SLOT_NAME=0000:00:02.0\n")
	write("card0-DP-1/device/vendor", "0x1002\n")
	write("renderD128/device/vendor", "0x1002\n")
	return root
}

func TestCollectScriptReadsSysfsGPUs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("collect.sh needs sh")
	}
	root := fakeDRM(t)
	// One snapshot only: the script loops until it is stopped.
	script := strings.Replace(collectScript, `sleep "$every"`, "exit 0", 1)
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = strings.NewReader(posixVars(1, 30) + script)
	cmd.Env = append(os.Environ(), "DAVISH_DRM="+root)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sections, err := NewReader(strings.NewReader(string(out))).sections()
	if err != nil {
		t.Fatal(err)
	}
	got := parseDRMGPUs(sections["gpudrm"], sections["gpuname"], 0)
	if len(got) != 2 {
		t.Fatalf("want the two real cards, got %+v from %q", got, sections["gpudrm"])
	}
	if got[0].Util != 37 || got[0].MemTotal != 8<<30 || got[0].Temp != 52 || got[0].Power != 75 {
		t.Errorf("amd = %+v", got[0])
	}
	if got[1].Util != -1 || got[1].MemTotal != -1 {
		t.Errorf("intel has no readings in sysfs: %+v", got[1])
	}
}
