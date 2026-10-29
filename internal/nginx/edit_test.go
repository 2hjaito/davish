package nginx

import (
	"strings"
	"testing"
)

const editSrc = `server {
    listen 80;
    server_name a.com www.a.com;
    gzip off;

    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}
server {
    listen 443 ssl;
    server_name a.com www.a.com; # tls
    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}`

func TestApplyEdit(t *testing.T) {
	e, err := SiteEdit{AddNames: []string{"b.com", "a.com"}, Upstream: "127.0.0.1:4000", Gzip: "on", MaxBody: "50m"}.check()
	if err != nil {
		t.Fatal(err)
	}
	got, err := applyEdit(editSrc, e)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"    server_name a.com www.a.com b.com;\n    gzip off;", // gzip replaced below, not added
		"    server_name a.com www.a.com b.com; # tls\n    client_max_body_size 50m;",
		"proxy_pass http://127.0.0.1:4000;",
	} {
		want = strings.Replace(want, "gzip off", "client_max_body_size 50m;\n    gzip on", 1)
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Count(got, "gzip on;") != 1 || strings.Contains(got, "3000") {
		t.Errorf("got\n%s", got)
	}
}

func TestApplyEditRejects(t *testing.T) {
	two := editSrc + "\nserver {\n    server_name c.com;\n    proxy_pass http://x:1;\n}"
	if _, err := applyEdit(two, SiteEdit{Upstream: "http://y:2"}); err == nil {
		t.Error("changed one of two upstreams")
	}
	for _, e := range []SiteEdit{{}, {MaxBody: "50 m"}, {Gzip: "yes"}, {AddNames: []string{"a;b"}}} {
		if _, err := e.check(); err == nil {
			t.Errorf("%+v accepted", e)
		}
	}
}

func TestEditLine(t *testing.T) {
	p := EditPlan{Path: "/etc/nginx/sites-available/a", Backup: "/etc/nginx/sites-available/a.bak-20261029T000000Z", After: "x"}
	want := "cp -p -- /etc/nginx/sites-available/a /etc/nginx/sites-available/a.bak-20261029T000000Z && printf %s x | tee -- /etc/nginx/sites-available/a > /dev/null && { nginx -t && systemctl reload nginx || { cp -p -- /etc/nginx/sites-available/a.bak-20261029T000000Z /etc/nginx/sites-available/a; false; }; }"
	if got := editLine(p, false); got != want {
		t.Errorf("line = %s", got)
	}
}
