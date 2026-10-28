package disk

import (
	"testing"
)

func TestParseDU(t *testing.T) {
	out := "4\t/var/empty\n2048\t/var/log\n1048576\t/var/lib\n1052680\t/var\nexit:124\n"
	u := parseDU([]byte(out), "/var")
	if u.Size != 1052680<<10 || !u.Partial {
		t.Fatalf("usage = %+v", u)
	}
	want := []string{"/var/lib", "/var/log", "/var/empty"}
	for i, e := range u.Entries {
		if e.Path != want[i] {
			t.Errorf("entry %d = %s, want %s", i, e.Path, want[i])
		}
	}
	if parseDU([]byte("8\t/\nexit:1\n"), "/").Partial {
		t.Error("a permission error is not a timeout")
	}
}

func TestUsageLine(t *testing.T) {
	got := usageLine("/srv/my site")
	want := `timeout 30 du -x -d 1 -k -- '/srv/my site' 2>/dev/null; echo "exit:$?"`
	if got != want {
		t.Errorf("line = %s", got)
	}
}

func TestParseDF(t *testing.T) {
	out := "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 1000 600 400 60% /\n/dev/sdb1 2000 100 1900 5% /mnt/my data\n"
	ms := parseDF([]byte(out))
	if len(ms) != 2 || ms[1].Path != "/mnt/my data" || ms[0].Used != 600<<10 {
		t.Fatalf("mounts = %+v", ms)
	}
}

func TestParseCleanups(t *testing.T) {
	out := `1000
--docker--
Images	1.5GB (60%)
Containers	0B (0%)
Local Volumes	0B
Build Cache	500MB
--journal--
Archived and active journals take up 1.1G in the file system.
--logs--
52428800 37
--apt--
204800	/var/cache/apt/archives
--dnf--
`
	cs := parseCleanups([]byte(out))
	got := map[string]Cleanup{}
	for _, c := range cs {
		got[c.ID] = c
	}
	if got["docker"].Bytes != 2_000_000_000 || got["docker"].Root {
		t.Errorf("docker = %+v", got["docker"])
	}
	if j := got["journal"]; j.Bytes != parseSize("1.1G")-journalKeep || j.Bytes < 1<<30 || !j.Root {
		t.Errorf("journal = %+v", j)
	}
	if l := got["logs"]; l.Bytes != 52428800 || l.Count != 37 {
		t.Errorf("logs = %+v", l)
	}
	if got["apt"].Bytes != 204800<<10 {
		t.Errorf("apt = %+v", got["apt"])
	}
	if _, ok := got["dnf"]; ok {
		t.Error("dnf listed without a cache")
	}
	if cs[0].ID != "docker" {
		t.Errorf("not largest first: %+v", cs)
	}
	// As root nothing needs sudo.
	for _, c := range parseCleanups([]byte("0\n--logs--\n100 1\n")) {
		if c.Root {
			t.Errorf("%s needs root for root", c.ID)
		}
	}
}

func TestCleanRefusesOtherCommand(t *testing.T) {
	s := New(nil)
	if err := s.Clean(t.Context(), "x", "docker", "rm -rf /"); err == nil {
		t.Fatal("ran a command that was not confirmed")
	}
}
