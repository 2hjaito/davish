package shortcut

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"davish/internal/fs"
)

// runLine runs the line for a script in dir with a real bash and returns
// the progress events it printed.
func runLine(t *testing.T, dir, name, content string, perm uint32) ([]string, string) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	p := filepath.ToSlash(filepath.Join(dir, name))
	if err := os.WriteFile(p, []byte(content), os.FileMode(perm)); err != nil { //nolint:gosec // a test script
		t.Fatal(err)
	}
	files := &fakeFiles{
		stat: fs.Entry{Kind: fs.KindFile, Perm: perm},
		file: fs.File{Entry: fs.Entry{Kind: fs.KindFile, Perm: perm}, Data: []byte(content)},
	}
	run, err := New(files, &fakeLog{}).Run(context.Background(), "s", p, hash([]byte(content)))
	if err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command(bash, "-c", run.Line).CombinedOutput() //nolint:gosec // the line under test
	marks := regexp.MustCompile("\x1b]7761;davish-deploy;"+run.Nonce+";([^\a]*)\a").
		FindAllStringSubmatch(string(out), -1)
	events := make([]string, 0, len(marks))
	for _, m := range marks {
		events = append(events, m[1])
	}
	return events, string(out)
}

func TestLineReportsResult(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		file    string
		content string
		perm    uint32
		want    string
		output  string
	}{
		{"success", "a", "ok.sh", "#!/bin/sh\necho hello\n", 0o755, "begin;1 step;1 end;0", "hello"},
		{"failure keeps the exit code", "a", "bad.sh", "#!/bin/sh\nexit 3\n", 0o755, "begin;1 step;1 end;3", "bad.sh failed"},
		{"no exec bit runs through bash", "a", "plain.sh", "echo plain\n", 0o644, "begin;1 step;1 end;0", "plain"},
		{"odd names", "my && dir", "it's $(x).sh", "#!/bin/sh\necho odd\n", 0o755, "begin;1 step;1 end;0", "odd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			events, out := runLine(t, filepath.Join(t.TempDir(), c.dir), c.file, c.content, c.perm)
			if got := strings.Join(events, " "); got != c.want {
				t.Fatalf("events = %q, want %q\n%s", got, c.want, out)
			}
			if !strings.Contains(out, c.output) {
				t.Fatalf("output lacks %q:\n%s", c.output, out)
			}
		})
	}
}
