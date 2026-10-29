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
