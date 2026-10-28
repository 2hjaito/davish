package nginx

import "testing"

func site(p, name string, enabled, toggle bool, certs ...string) Site {
	return Site{Path: p, Name: name, Enabled: enabled, Toggleable: toggle, Servers: []Server{{Certs: certs}}}
}

func TestPlanRemove(t *testing.T) {
	inv := "/etc/nginx/sites-available/inventory.davitek.com"
	sites := []Site{
		site(inv, "inventory.davitek.com", true, true,
			"/etc/letsencrypt/live/inventory.davitek.com/fullchain.pem",
			"/etc/letsencrypt/live/davitek.com/fullchain.pem"),
		site("/etc/nginx/sites-available/davitek.com", "davitek.com", true, true,
			"/etc/letsencrypt/live/davitek.com/fullchain.pem"),
		site("/etc/nginx/conf.d/fixssl.conf", "fixssl.conf", true, false, "/etc/ssl/own.pem"),
		site("/etc/nginx/sites-available/old", "old", false, true),
	}

	p, err := planRemove(sites, inv, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := "rm -- /etc/nginx/sites-enabled/inventory.davitek.com && rm -- " + inv; p.Command != want {
		t.Errorf("command = %s", p.Command)
	}
	if len(p.Certs) != 1 || p.Certs[0] != "inventory.davitek.com" || len(p.SharedCerts) != 1 || p.SharedCerts[0] != "davitek.com" {
		t.Errorf("certs = %v, shared = %v", p.Certs, p.SharedCerts)
	}

	p, _ = planRemove(sites, inv, true)
	if want := "rm -- /etc/nginx/sites-enabled/inventory.davitek.com && rm -- " + inv +
		" && certbot delete --non-interactive --cert-name inventory.davitek.com"; p.Command != want || !p.WithCerts {
		t.Errorf("with certs = %s", p.Command)
	}
	if want := "sudo rm -- /etc/nginx/sites-enabled/inventory.davitek.com && sudo rm -- " + inv +
		" && sudo certbot delete --non-interactive --cert-name inventory.davitek.com"; p.Sudo != want {
		t.Errorf("sudo = %s", p.Sudo)
	}

	// A conf.d file has no link; an own certificate is not certbot's.
	p, _ = planRemove(sites, "/etc/nginx/conf.d/fixssl.conf", true)
	if p.Command != "rm -- /etc/nginx/conf.d/fixssl.conf" || p.WithCerts {
		t.Errorf("conf.d = %+v", p)
	}

	// A disabled site has no link to remove.
	if p, _ = planRemove(sites, "/etc/nginx/sites-available/old", false); p.Command != "rm -- /etc/nginx/sites-available/old" {
		t.Errorf("disabled = %s", p.Command)
	}

	if _, err := planRemove(sites, "/etc/nginx/sites-available/missing", false); err == nil {
		t.Error("missing site planned")
	}
}

func TestCheckSitePath(t *testing.T) {
	for p, ok := range map[string]bool{
		"/etc/nginx/sites-available/a":    true,
		"/etc/nginx/conf.d/a.conf":        true,
		"/etc/nginx/nginx.conf":           false,
		"/etc/nginx/sites-available/../x": false,
		"/etc/passwd":                     false,
		"/etc/nginx/sites-available/-rf":  false,
	} {
		if err := checkSitePath(p); (err == nil) != ok {
			t.Errorf("checkSitePath(%q) = %v", p, err)
		}
	}
}
