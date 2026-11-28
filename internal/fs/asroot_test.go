package fs

import (
	"strings"
	"testing"
)

func TestParseList(t *testing.T) {
	out := "d\td\t4096\t1700000000.5000000000\t700\t0\t0\tsecret dir\x00\x00" +
		"f\tf\t12\t1700000100.0\t600\t0\t0\ta\tb.txt\x00\x00" +
		"l\td\t9\t1700000200.0\t777\t0\t0\tlink\x00/etc\x00" +
		"f\tf\t1\t17"
	got := parseList("/root", []byte(out))
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3 (the cut record is dropped): %+v", len(got), got)
	}
	byName := map[string]Entry{}
	for _, e := range got {
		byName[e.Name] = e
	}
	if d := byName["secret dir"]; !d.Dir || d.Kind != KindDir || d.Mode != "drwx------" || d.Path != "/root/secret dir" {
		t.Errorf("dir: %+v", d)
	}
	if f := byName["a\tb.txt"]; f.Kind != KindFile || f.Mode != "-rw-------" || f.Size != 12 || f.ModTime != 1700000100 {
		t.Errorf("file with a tab in its name: %+v", f)
	}
	if l := byName["link"]; l.Kind != KindLink || !l.Dir || l.Target != "/etc" {
		t.Errorf("link to a dir: %+v", l)
	}
}

func TestParseListRootDir(t *testing.T) {
	got := parseList("/", []byte("f\tf\t1\t1.0\t644\t0\t0\tx\x00\x00"))
	if len(got) != 1 || got[0].Path != "/x" {
		t.Errorf("got %+v", got)
	}
}

func TestRootReadPlansQuoteThePath(t *testing.T) {
	s := &Service{}
	p, err := s.PlanReadAsRoot("/root/it's here; rm -rf x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Sudo, "sudo sh -c ") || strings.Count(p.Sudo, "rm -rf x") != 3 {
		t.Errorf("plan = %s", p.Sudo)
	}
	if _, err := s.PlanListAsRoot("relative"); err == nil {
		t.Error("a relative path must be refused")
	}
}
