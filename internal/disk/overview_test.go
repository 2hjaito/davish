package disk

import (
	"strings"
	"testing"
)

func TestOverviewLineQuotesMount(t *testing.T) {
	line := overviewLine("/mnt/it's $(reboot)")
	for _, want := range []string{
		`if sudo -n true 2>/dev/null; then echo as:root; sudo -n sh -c '`,
		` sh '/mnt/it'\''s $(reboot)'; else echo as:user; sh -c '`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("line lacks %q", want)
		}
	}
}

func TestParseOverview(t *testing.T) {
	out := strings.Join([]string{
		"as:root",
		"--df--",
		"Filesystem 1024-blocks Used Available Capacity Mounted on",
		"/dev/sda1 10485760 9437184 1048576 90% /",
		"--du--",
		"du: cannot read directory '/var/lib/docker/overlay2/x': Permission denied",
		"du: cannot open directory '/root': Permission denied",
		"6291456\t/var",
		"5242880\t/var/lib",
		"512\t/var/tiny",
		"1048576\t/home",
		"7340032\t/",
		"exit:1",
		"--files--",
		"2147483648\t/var/lib/mysql/ibdata1",
		"104857600\t/home/me/a b.tar",
		"--deleted--",
		"536870912\t812\tnode\t/var/log/app.log (deleted)",
		"bad line",
	}, "\n")
	o := parseOverview([]byte(out), "/")
	if !o.Root || o.Partial || o.Measured != 7340032<<10 || o.Mount.Used != 9437184<<10 {
		t.Fatalf("overview = %+v", o)
	}
	if len(o.Dirs) != 3 || o.Dirs[0].Path != "/var" || o.Dirs[2].Path != "/home" {
		t.Errorf("dirs = %+v", o.Dirs)
	}
	if len(o.Files) != 2 || o.Files[1].Path != "/home/me/a b.tar" {
		t.Errorf("files = %+v", o.Files)
	}
	if len(o.Unreadable) != 2 || o.Unreadable[0] != "/var/lib/docker/overlay2/x" || o.Unreadable[1] != "/root" {
		t.Errorf("unreadable = %q", o.Unreadable)
	}
	want := DeletedFile{Path: "/var/log/app.log", Size: 536870912, PID: 812, Command: "node"}
	if len(o.Deleted) != 1 || o.Deleted[0] != want {
		t.Errorf("deleted = %+v", o.Deleted)
	}
	// 9 GiB used - 7 GiB measured - 512 MiB held open.
	if o.Unmeasured != (9437184-7340032)<<10-536870912 {
		t.Errorf("unmeasured = %d", o.Unmeasured)
	}
}

func TestParseOverviewTimeoutAndEmpty(t *testing.T) {
	o := parseOverview([]byte("as:user\n--du--\n100\t/srv\nexit:124\n"), "/srv")
	if o.Root || !o.Partial || o.Measured != 100<<10 || o.Unmeasured != 0 {
		t.Errorf("overview = %+v", o)
	}
	if o.Dirs == nil || o.Files == nil || o.Unreadable == nil || o.Deleted == nil {
		t.Error("lists must be empty, not nil, for the frontend")
	}
}
