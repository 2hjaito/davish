package ports

import "testing"

const ufwVerbose = `Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)
New profiles: skip

To                         Action      From
--                         ------      ----
Anywhere                   DENY IN     193.24.211.219
22/tcp                     ALLOW IN    Anywhere
Nginx Full                 ALLOW IN    Anywhere
3000:3005/tcp              ALLOW IN    Anywhere
5432                       ALLOW IN    10.0.0.5
53/udp                     ALLOW IN    Anywhere
22/tcp (v6)                ALLOW IN    Anywhere (v6)
`

func TestExposure(t *testing.T) {
	fw := parseUFW(ufwVerbose, true)
	if fw.State != FirewallActive || fw.DefaultAllow {
		t.Fatalf("firewall = %+v", fw)
	}
	for _, c := range []struct {
		p    Port
		want string
	}{
		{Port{Port: 22}, ExposedYes},
		{Port{Port: 443}, ExposedYes},
		{Port{Port: 3003}, ExposedYes},
		{Port{Port: 5432}, ExposedNo}, // only from one address
		{Port{Port: 6379}, ExposedNo},
		{Port{Port: 6379, Loopback: true}, ExposedNo},
		{Port{Port: 8080, Process: "docker-proxy"}, ExposedYes},
	} {
		if got := exposure(c.p, fw); got != c.want {
			t.Errorf("%d = %s, want %s", c.p.Port, got, c.want)
		}
	}
	if got := exposure(Port{Port: 6379}, parseUFW("", true)); got != ExposedUnknown {
		t.Errorf("unreadable ufw = %s", got)
	}
	if got := exposure(Port{Port: 6379}, parseUFW("Status: inactive\n", true)); got != ExposedYes {
		t.Errorf("inactive ufw = %s", got)
	}
	if got := exposure(Port{Port: 6379}, parseUFW("", false)); got != ExposedYes {
		t.Errorf("no ufw = %s", got)
	}
}
