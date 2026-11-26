package disk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
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
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	line := strings.Replace(commands["logs"], "/var/log", dir, 1)
	line = strings.TrimSuffix(line, " -delete") + " -print"
	out, err := exec.Command("sh", "-c", line).Output() //nolint:gosec // a fixed selector run on a temp tree
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

func TestPreviewListsWhatTheCommandSelects(t *testing.T) {
	// The delete command and the listing share their selector.
	for id, sel := range map[string]string{
		"logs": selLogs, "docker-logs": selDockerLogs, "coredumps": selCoredumps, "tmp": selTmp,
	} {
		if !strings.HasPrefix(commands[id], sel) {
			t.Errorf("%s: the command does not start from its selector", id)
		}
		if !strings.Contains(previews[id], sel) {
			t.Errorf("%s: the listing does not use its selector", id)
		}
		if strings.Contains(previews[id], "-delete") || strings.Contains(previews[id], "-exec") {
			t.Errorf("%s: the listing must not change anything: %s", id, previews[id])
		}
	}
}

func TestParsePreview(t *testing.T) {
	got := parsePreview([]byte("2048\t/var/log/syslog.1\n10\t/var/log/a b.gz\nbad line\n\n"))
	if len(got) != 2 || got[0].Path != "/var/log/syslog.1" || got[0].Size != 2048 || got[1].Path != "/var/log/a b.gz" {
		t.Errorf("got %+v", got)
	}
}

func TestPreviewRunsReadOnlyAsRootWhenItNeedsNoPrompt(t *testing.T) {
	run, root := &lineRunner{}, &fakeRoot{res: audit.Result{Stdout: []byte("5\t/var/log/x.1\n")}}
	s := New(run, nil)
	s.UseRoot(root)
	p, err := s.Preview(context.Background(), "srv", "logs")
	if err != nil || !p.Root || len(p.Files) != 1 || len(run.ran) != 0 {
		t.Fatalf("preview = %+v, %v, ran as user %q", p, err, run.ran)
	}
	// Asking for a password falls back to what the user can read.
	run2 := &lineRunner{res: audit.Result{Stdout: []byte("5\t/var/log/y.1\n")}}
	s2 := New(run2, nil)
	s2.UseRoot(&fakeRoot{err: apperr.New(apperr.SudoPasswordRequired, "x")})
	p, err = s2.Preview(context.Background(), "srv", "logs")
	if err != nil || p.Root || len(p.Files) != 1 {
		t.Errorf("fallback = %+v, %v", p, err)
	}
	if _, err := s.Preview(context.Background(), "srv", "docker-images"); err == nil {
		t.Error("a prune has no listing")
	}
}

// TestTmpSelectorSparesWhatIsInUse runs the tmp selector on a temp tree:
// old, untouched files match; recent ones, files only read long ago but
// written lately, and a service's private /tmp do not.
func TestTmpSelectorSparesWhatIsInUse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX find")
	}
	dir := t.TempDir()
	old := time.Now().Add(-30 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	// name -> (atime, mtime, matches)
	cases := map[string]struct {
		atime, mtime time.Time
		want         bool
	}{
		"tmp/old.dat":                          {old, old, true},
		"var/tmp/old.dat":                      {old, old, true},
		"tmp/fresh.dat":                        {recent, recent, false},
		"tmp/written-lately.db":                {old, recent, false},
		"tmp/systemd-private-x/tmp/old.sock.f": {old, old, false},
	}
	var want []string
	for name, c := range cases {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, c.atime, c.mtime); err != nil {
			t.Fatal(err)
		}
		if c.want {
			want = append(want, name)
		}
	}
	line := strings.ReplaceAll(selTmp, "/var/tmp", "@VARTMP@")
	line = strings.ReplaceAll(line, "/tmp", dir+"/tmp")
	line = strings.ReplaceAll(line, "@VARTMP@", dir+"/var/tmp") + " -print"
	out, err := exec.Command("sh", "-c", line).Output() //nolint:gosec // a fixed selector run on a temp tree
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	var got []string
	for _, l := range strings.Fields(string(out)) {
		rel, _ := filepath.Rel(dir, l)
		got = append(got, rel)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("matched %v, want %v", got, want)
	}
}

func TestParsePreviewKeepsUnknownSizes(t *testing.T) {
	got := parsePreview([]byte("-1\tdb_data\n"))
	if len(got) != 1 || got[0].Path != "db_data" || got[0].Size != -1 {
		t.Errorf("got %+v", got)
	}
}
