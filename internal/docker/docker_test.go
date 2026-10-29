package docker

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // fixed test fixtures
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParsePS(t *testing.T) {
	list, err := parsePS(read(t, "ps.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "davish-crashloop" || list[1].Name != "davish-web" {
		t.Fatalf("got %+v", list)
	}
	c := list[0]
	if c.State != "restarting" || c.Image != "davish/busybox" || len(c.ID) != 64 {
		t.Errorf("crashloop = %+v", c)
	}
	if c.Command == "" || c.Command[0] == '"' {
		t.Errorf("command keeps its quotes: %q", c.Command)
	}
	if list[1].State != "running" {
		t.Errorf("web = %+v", list[1])
	}

	if _, err := parsePS([]byte("not json\n")); err == nil {
		t.Error("bad output parsed")
	}
	if got, err := parsePS(nil); err != nil || len(got) != 0 {
		t.Errorf("empty = %v, %v", got, err)
	}
}

func TestParseStats(t *testing.T) {
	list, err := parseStats(read(t, "stats.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %+v", list)
	}
	web := list[1]
	if web.CPU != 0.12 || web.MemUsed != size("1.004MiB") || web.MemLimit != size("7.749GiB") ||
		web.MemUsed < 1<<20 || web.MemLimit < 7<<30 || web.PIDs != 2 {
		t.Errorf("web = %+v", web)
	}
}

func TestSize(t *testing.T) {
	tests := map[string]int64{
		"0B":        0,
		"512B":      512,
		"12kB":      12000,
		"1.5MiB":    1572864,
		" 2GiB ":    2 << 30,
		"1.2GB":     1200000000,
		"garbage":   0,
		"3 parsecs": 0,
	}
	for in, want := range tests {
		if got := size(in); got != want {
			t.Errorf("size(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseInspect(t *testing.T) {
	d, err := parseInspect(read(t, "inspect.json"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "davish-crashloop" || d.Image != "davish/busybox" || d.State != "restarting" ||
		d.ExitCode != 1 || d.Policy != "always" || d.Restarts == 0 || d.StartedAt == "" {
		t.Errorf("detail = %+v", d)
	}
	if d.JSON == "" || d.JSON[0] != '{' {
		t.Errorf("json = %.40q", d.JSON)
	}

	ports := []byte(`[{"Name":"/web","State":{"Status":"created","StartedAt":"0001-01-01T00:00:00Z","Health":{"Status":"starting"}},
	  "NetworkSettings":{"Ports":{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}],"443/tcp":null}},
	  "Mounts":[{"Source":"/srv/www","Destination":"/usr/share/nginx/html"}]}]`)
	d, err = parseInspect(ports)
	if err != nil {
		t.Fatal(err)
	}
	want := Detail{
		Name: "web", State: "created", Health: "starting",
		Ports:  []string{"0.0.0.0:8080→80/tcp", "443/tcp"},
		Mounts: []string{"/srv/www → /usr/share/nginx/html"},
	}
	d.JSON = ""
	if !reflect.DeepEqual(d, want) {
		t.Errorf("got  %+v\nwant %+v", d, want)
	}
}

func TestMergeLogs(t *testing.T) {
	stdout := "2026-10-27T15:54:01.000000001Z request 1 served\n2026-10-27T15:54:03.5Z request 2 served\n"
	stderr := "2026-10-27T15:54:02.25Z Error: cannot open /etc/nginx/nginx.conf\n"
	got := mergeLogs([]byte(stdout), []byte(stderr)).Lines
	want := []LogLine{
		{"2026-10-27T15:54:01.000000001Z", "request 1 served", false},
		{"2026-10-27T15:54:02.25Z", "Error: cannot open /etc/nginx/nginx.conf", true},
		{"2026-10-27T15:54:03.5Z", "request 2 served", false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	if got := mergeLogs(nil, nil).Lines; got == nil || len(got) != 0 {
		t.Errorf("empty = %#v", got)
	}
}

func TestPlanAction(t *testing.T) {
	p, err := PlanAction("davish-web", "restart")
	want := Plan{"davish-web", "restart", "docker restart davish-web", true, "sudo docker restart davish-web"}
	if err != nil || p != want {
		t.Errorf("got %+v, %v", p, err)
	}
	if p, _ := PlanAction("web.1_a", "start"); p.Confirm {
		t.Error("start asks for confirmation")
	}
	for _, bad := range [][2]string{
		{"-v", "start"}, {"a b", "stop"}, {"$(id)", "stop"}, {"", "stop"}, {"web", "kill"},
	} {
		if _, err := PlanAction(bad[0], bad[1]); err == nil {
			t.Errorf("PlanAction(%q, %q) accepted", bad[0], bad[1])
		}
	}
}

func TestCheckExit(t *testing.T) {
	tests := []struct {
		stderr string
		exit   int
		want   apperr.Code
	}{
		{"", 0, ""},
		{"permission denied while trying to connect to the docker API at unix:///var/run/docker.sock\n", 1, apperr.CmdNeedsRoot},
		{"Got permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock\n", 1, apperr.CmdNeedsRoot},
		{"Error response from daemon: No such container: nosuch\n", 1, apperr.NotFound},
		{"Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n", 1, apperr.DockerDown},
		{"sh: 1: docker: not found\n", 127, apperr.CmdMissing},
		{"Error response from daemon: something else\n", 1, apperr.CmdFailed},
	}
	for _, tt := range tests {
		err := checkExit(audit.Result{Exit: tt.exit, Stderr: []byte(tt.stderr)})
		var ae *apperr.Error
		got := apperr.Code("")
		if errors.As(err, &ae) {
			got = ae.Code
		}
		if got != tt.want {
			t.Errorf("%q: code %q, want %q", tt.stderr, got, tt.want)
		}
	}
}
