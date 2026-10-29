package nginx

import (
	"context"
	"errors"
	"testing"
)

type fakeDNS map[string][]string

func (f fakeDNS) LookupHost(_ context.Context, host string) ([]string, error) {
	if ips, ok := f[host]; ok {
		return ips, nil
	}
	return nil, errors.New("no such host")
}

func TestPlanSSL(t *testing.T) {
	dns := fakeDNS{"vps.example.net": {"203.0.113.5"}, "a.com": {"203.0.113.5"}, "www.a.com": {"198.51.100.1"}}
	p := PlanSSL(context.Background(), dns, "vps.example.net", []string{"a.com", "www.a.com", "new.a.com", "_", "*.a.com", "localhost", "A.com"}, true)
	if want := "sudo certbot --nginx -d a.com -d www.a.com -d new.a.com --redirect"; p.Sudo != want {
		t.Errorf("sudo = %s", p.Sudo)
	}
	if len(p.Skipped) != 3 {
		t.Errorf("skipped = %v", p.Skipped)
	}
	if !p.DNS[0].OK || p.DNS[1].OK || p.DNS[2].Error == "" {
		t.Errorf("dns = %+v", p.DNS)
	}
	p = PlanSSL(context.Background(), dns, "203.0.113.5", []string{"a.com"}, false)
	if !p.DNS[0].OK || p.Sudo != "sudo certbot --nginx -d a.com --no-redirect" {
		t.Errorf("by ip = %+v", p)
	}
	if p := PlanSSL(context.Background(), dns, "x", []string{"_"}, false); p.Sudo != "" {
		t.Errorf("no names gave %s", p.Sudo)
	}
}
