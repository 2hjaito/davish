package ports

import (
	"testing"
)

func TestParse(t *testing.T) {
	out := `1000
State  Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
LISTEN 0      511          0.0.0.0:80         0.0.0.0:*     users:(("nginx",pid=1234,fd=6),("nginx",pid=1235,fd=6))
LISTEN 0      4096   127.0.0.53%lo:53         0.0.0.0:*     users:(("systemd-resolve",pid=600,fd=14))
LISTEN 0      128             [::]:22            [::]:*
LISTEN 0      4096         0.0.0.0:8080       0.0.0.0:*     users:(("docker-proxy",pid=900,fd=4))
` + sep + `
web	0.0.0.0:8080->80/tcp, :::8080->80/tcp
db	3306/tcp
`
	l, err := parse([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if l.Root {
		t.Error("uid 1000 reported as root")
	}
	want := []Port{
		{Addr: "::", Port: 22},
		{Addr: "127.0.0.53", Port: 53, Process: "systemd-resolve", PID: 600, Loopback: true},
		{Addr: "0.0.0.0", Port: 80, Process: "nginx", PID: 1234},
		{Addr: "0.0.0.0", Port: 8080, Process: "docker-proxy", PID: 900, Container: "web"},
	}
	if len(l.Ports) != len(want) {
		t.Fatalf("ports = %+v", l.Ports)
	}
	for i := range want {
		if l.Ports[i] != want[i] {
			t.Errorf("port %d = %+v, want %+v", i, l.Ports[i], want[i])
		}
	}
}

func TestParseNoSS(t *testing.T) {
	if _, err := parse([]byte("0\nNO_SS\n" + sep + "\n")); err == nil {
		t.Fatal("missing ss not reported")
	}
}

func TestProbeLine(t *testing.T) {
	line, err := ProbeLine("0.0.0.0", 80, "shop.davitek.com")
	if err != nil {
		t.Fatal(err)
	}
	want := `curl -sS -i --max-time 10 -H 'Host: shop.davitek.com' http://127.0.0.1:80/ 2>&1 | head -c 16384`
	if line != want {
		t.Errorf("line = %s", line)
	}
	for _, bad := range []struct {
		host, header string
	}{{"evil;rm", ""}, {"127.0.0.1", "a b"}, {"127.0.0.1", "x\r\nInjected: 1"}} {
		if _, err := ProbeLine(bad.host, 80, bad.header); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}
