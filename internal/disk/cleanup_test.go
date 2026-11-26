package disk

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestLogsPatternSparesDataVolumes runs the real find expression on a temp
// tree: rotated logs match, a live PCP archive volume (#296) does not.
func TestLogsPatternSparesDataVolumes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX find")
	}
	dir := t.TempDir()
	files := map[string]bool{
		"syslog.1":                            true,
		"auth.log.2":                          true,
		"nginx/access.log.10":                 true,
		"kern.log.3.gz":                       true,
		"dpkg.log.old":                        true,
		"syslog":                              false,
		"auth.log":                            false,
		"pcp/pmlogger/h/20261125.00.10.0":     false,
		"pcp/pmlogger/h/20261125.00.10.index": false,
	}
	for name := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	line := strings.Replace(commands["logs"], "/var/log", dir, 1)
	line = strings.TrimSuffix(line, " -delete") + " -print"
	out, err := exec.Command("sh", "-c", line).Output()
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	var got []string
	for _, l := range strings.Fields(string(out)) {
		rel, _ := filepath.Rel(dir, l)
		got = append(got, rel)
	}
	var want []string
	for name, rotated := range files {
		if rotated {
			want = append(want, name)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("matched %v, want %v", got, want)
	}
}
