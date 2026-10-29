package ports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEmptyListIsAnArray(t *testing.T) {
	l, err := parse([]byte("1000\n" + sep + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(l)
	if !strings.Contains(string(b), `"ports":[]`) {
		t.Errorf("json = %s", b)
	}
}

func TestNetidColumn(t *testing.T) {
	l, _ := parse([]byte("1000\ntcp   LISTEN 0      128      0.0.0.0:22      0.0.0.0:*    users:((\"sshd\",pid=812,fd=3))\n" + sep + "\n"))
	if len(l.Ports) != 1 || l.Ports[0].Port != 22 || l.Ports[0].Process != "sshd" {
		t.Fatalf("ports = %+v", l.Ports)
	}
}

// Two sockets on one address and port (one per bridge interface, or
// SO_REUSEPORT) are one port: the frontend keys its list on addr:port.
func TestDuplicateSocketsAreOnePort(t *testing.T) {
	out := "1000\n" +
		"LISTEN 0 4096 172.27.0.1%br-1a2b:53 0.0.0.0:*\n" +
		"LISTEN 0 4096 172.27.0.1%br-3c4d:53 0.0.0.0:* users:((\"dnsmasq\",pid=77,fd=5))\n" +
		"LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:((\"nginx\",pid=12,fd=6))\n" +
		"LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:((\"nginx\",pid=13,fd=6))\n" +
		sep + "\n"
	l, err := parse([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	want := []Port{
		{Addr: "172.27.0.1", Port: 53, Process: "dnsmasq", PID: 77, Exposed: ExposedYes},
		{Addr: "0.0.0.0", Port: 80, Process: "nginx", PID: 12, Exposed: ExposedYes},
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
