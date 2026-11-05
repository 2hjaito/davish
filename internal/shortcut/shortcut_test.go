package shortcut

import (
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/fs"
)

type fakeFiles struct {
	stat    fs.Entry
	statErr error
	file    fs.File
	readErr error
}

func (f *fakeFiles) Stat(context.Context, string, string) (fs.Entry, error) {
	return f.stat, f.statErr
}

func (f *fakeFiles) Read(context.Context, string, string) (fs.File, error) {
	return f.file, f.readErr
}

type fakeLog struct{ entries []audit.Entry }

func (l *fakeLog) Add(e audit.Entry) audit.Entry {
	l.entries = append(l.entries, e)
	return e
}

func script(perm uint32, data string) *fakeFiles {
	return &fakeFiles{
		stat: fs.Entry{Kind: fs.KindFile, Perm: perm},
		file: fs.File{Entry: fs.Entry{Kind: fs.KindFile, Perm: perm}, Data: []byte(data)},
	}
}

func TestInspectKinds(t *testing.T) {
	link := script(0o644, "text")
	link.stat = fs.Entry{Kind: fs.KindLink, Perm: 0o777}
	cases := []struct {
		name  string
		path  string
		files *fakeFiles
		want  string
	}{
		{"folder", "/srv/app", &fakeFiles{stat: fs.Entry{Kind: fs.KindDir, Dir: true}}, KindFolder},
		{"link to folder", "/srv/cur", &fakeFiles{stat: fs.Entry{Kind: fs.KindLink, Dir: true}}, KindFolder},
		{"executable", "/srv/deploy", script(0o755, "x"), KindScript},
		{"sh without the bit", "/srv/deploy.sh", script(0o644, "x"), KindScript},
		{"upper case extension", "/srv/DEPLOY.SH", script(0o644, "x"), KindScript},
		{"plain file", "/srv/notes.txt", script(0o644, "x"), KindFile},
		{"link judged by its target", "/srv/link.txt", link, KindFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := New(c.files, &fakeLog{}).Inspect(context.Background(), "s", c.path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != c.want {
				t.Fatalf("kind = %q, want %q", got.Kind, c.want)
			}
		})
	}
}

func TestInspectRefusesDevices(t *testing.T) {
	f := &fakeFiles{stat: fs.Entry{Kind: "other"}}
	if _, err := New(f, &fakeLog{}).Inspect(context.Background(), "s", "/dev/zero"); err == nil {
		t.Fatal("want an error: reading a device would never end")
	}
}

func TestCommandQuotesPaths(t *testing.T) {
	cases := []struct {
		name string
		path string
		perm uint32
		want string
	}{
		{"executable", "/srv/app/deploy.sh", 0o755, "cd -- /srv/app && ./deploy.sh"},
		{"no exec bit", "/srv/app/deploy.sh", 0o644, "cd -- /srv/app && bash ./deploy.sh"},
		{"space", "/srv/my app/de ploy.sh", 0o755, `cd -- '/srv/my app' && './de ploy.sh'`},
		{"substitution", "/srv/$(id)/x.sh", 0o755, `cd -- '/srv/$(id)' && ./x.sh`},
		{"quote", "/srv/it's/x.sh", 0o755, `cd -- '/srv/it'\''s' && ./x.sh`},
		{"dash", "/srv/-rf.sh", 0o755, "cd -- /srv && ./-rf.sh"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := New(script(c.perm, "x"), &fakeLog{}).Inspect(context.Background(), "s", c.path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Command != c.want {
				t.Fatalf("command = %s, want %s", got.Command, c.want)
			}
		})
	}
}

func TestRunRefusesChangedContent(t *testing.T) {
	log := &fakeLog{}
	s := New(script(0o755, "new content"), log)
	_, err := s.Run(context.Background(), "s", "/srv/deploy.sh", hash([]byte("old content")))
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ShortcutChanged {
		t.Fatalf("err = %v, want %s", err, apperr.ShortcutChanged)
	}
	if len(log.entries) != 0 {
		t.Fatal("a refused run must not be recorded")
	}
}

func TestRunRefusesNonScript(t *testing.T) {
	s := New(script(0o644, "x"), &fakeLog{})
	if _, err := s.Run(context.Background(), "s", "/srv/notes.txt", ""); err == nil {
		t.Fatal("want an error")
	}
}

func TestRunRecordsAndMarks(t *testing.T) {
	log := &fakeLog{}
	s := New(script(0o755, "echo hi"), log)
	run, err := s.Run(context.Background(), "s", "/srv/app/deploy.sh", hash([]byte("echo hi")))
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Nonce) != 32 {
		t.Fatalf("nonce = %q", run.Nonce)
	}
	for _, want := range []string{
		"cd -- /srv/app && __davish_mark '" + run.Nonce + ";begin;1'",
		"__davish_mark '" + run.Nonce + ";step;1' && { ./deploy.sh; } && __davish_mark '" + run.Nonce + ";end;0'",
		`|| { __davish_mark "` + run.Nonce + `;end;$?"`,
		"unset -f __davish_mark",
	} {
		if !strings.Contains(run.Line, want) {
			t.Errorf("line lacks %q:\n%s", want, run.Line)
		}
	}
	if strings.Contains(run.Line, "sudo") {
		t.Error("the line must hold no sudo: Davish presses Enter on it")
	}
	if len(log.entries) != 1 || log.entries[0].Command != "cd -- /srv/app && ./deploy.sh" ||
		log.entries[0].App != "shortcut" || log.entries[0].Exit != -1 {
		t.Fatalf("audit = %+v", log.entries)
	}
}

func TestRunKeepsSeparatorsInsidePaths(t *testing.T) {
	data := "echo hi"
	run, err := New(script(0o755, data), &fakeLog{}).
		Run(context.Background(), "s", "/srv/a && b/x.sh", hash([]byte(data)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(run.Line, `cd -- '/srv/a && b' && __davish_mark`) ||
		!strings.Contains(run.Line, "{ ./x.sh; }") {
		t.Fatalf("line = %s", run.Line)
	}
}

func TestTooBigScriptHasNoHash(t *testing.T) {
	f := script(0o755, "")
	f.readErr = apperr.New(apperr.FSTooLarge, "big")
	got, err := New(f, &fakeLog{}).Inspect(context.Background(), "s", "/srv/big.sh")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != KindScript || got.Hash != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestPermissionErrorPropagates(t *testing.T) {
	f := script(0o755, "")
	f.readErr = apperr.New(apperr.FSPermissionDenied, "no")
	if _, err := New(f, &fakeLog{}).Inspect(context.Background(), "s", "/srv/x.sh"); err == nil {
		t.Fatal("want an error")
	}
}
