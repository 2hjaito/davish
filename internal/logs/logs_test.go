package logs

import (
	"reflect"
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		src  Source
		want string
	}{
		{Source{KindJournal, ""}, "journalctl -f -n 200 -o short-iso --no-pager"},
		{Source{KindJournal, "nginx.service"}, "journalctl -u nginx.service -f -n 200 -o short-iso --no-pager"},
		{Source{KindFile, "/var/log/nginx/error.log"}, "tail -F -n 200 -- /var/log/nginx/error.log"},
		{Source{KindFile, "/var/log/my app.log"}, "tail -F -n 200 -- '/var/log/my app.log'"},
		{Source{KindDocker, "davish-web"}, "docker logs -f --tail 200 davish-web"},
	}
	for _, tt := range tests {
		cmd, err := Command(tt.src)
		if err != nil || cmd.String() != tt.want {
			t.Errorf("Command(%+v) = %q, %v; want %q", tt.src, cmd.String(), err, tt.want)
		}
	}
	for _, bad := range []Source{
		{KindJournal, "-f.service"},
		{KindJournal, "nginx"},
		{KindFile, "var/log/syslog"},
		{KindFile, "/var/log/../../etc/shadow"},
		{KindDocker, "--help"},
		{"ssh", "x"},
	} {
		if _, err := Command(bad); err == nil {
			t.Errorf("Command(%+v) accepted", bad)
		}
	}
}

func TestReadLines(t *testing.T) {
	long := strings.Repeat("x", MaxLine+5000)
	in := "one\r\ntwo\n\n" + long + "\nlast without newline"
	var got []string
	if err := readLines(strings.NewReader(in), func(l string) { got = append(got, l) }); err != nil {
		t.Fatal(err)
	}
	want := []string{"one", "two", "", long[:MaxLine], "last without newline"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %d lines: %.40q", len(got), got)
	}
}

func TestNeedsRoot(t *testing.T) {
	tests := []struct {
		kind, line string
		want       bool
	}{
		{KindFile, "tail: cannot open '/var/log/auth.log' for reading: Permission denied", true},
		{KindFile, "tail: '/var/log/x' has become inaccessible: No such file or directory", false},
		{KindDocker, "permission denied while trying to connect to the docker API at unix:///var/run/docker.sock", true},
		{KindDocker, "Got permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock", true},
		// A container printing this on stderr is output, not a refusal.
		{KindDocker, "open /data/x: Permission denied", false},
		{KindJournal, "Hint: You are currently not seeing messages from other users and the system.", true},
		{KindJournal, "No journal files were opened due to insufficient permissions.", true},
		{KindJournal, "sshd[12]: error: Permission denied (publickey)", false},
	}
	for _, tt := range tests {
		if got := needsRoot(tt.kind, tt.line); got != tt.want {
			t.Errorf("needsRoot(%s, %q) = %v", tt.kind, tt.line, got)
		}
	}
}
