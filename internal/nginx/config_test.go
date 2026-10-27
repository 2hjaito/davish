package nginx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // fixed test fixtures
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseDefault(t *testing.T) {
	p := Parse(read(t, "default"))
	if len(p.Servers) != 1 {
		t.Fatalf("servers = %+v", p.Servers)
	}
	s := p.Servers[0]
	if !reflect.DeepEqual(s.Names, []string{"_"}) || !reflect.DeepEqual(s.Listen, []string{"80 default_server", "[::]:80 default_server"}) ||
		s.Root != "/var/www/html" || s.SSL {
		t.Errorf("default = %+v", s)
	}
}

func TestParseSiteWithUpstream(t *testing.T) {
	p := Parse(read(t, "davish-site"))
	if !reflect.DeepEqual(p.Upstreams, []string{"davish_backend"}) {
		t.Errorf("upstreams = %v", p.Upstreams)
	}
	// The upstream's own "server 127.0.0.1:9000;" is not a server block.
	if len(p.Servers) != 1 {
		t.Fatalf("servers = %+v", p.Servers)
	}
	s := p.Servers[0]
	if !reflect.DeepEqual(s.Names, []string{"davish.test", "www.davish.test"}) || s.Root != "/srv/davish-site" ||
		!reflect.DeepEqual(s.ProxyPass, []string{"http://davish_backend"}) || s.Line != 7 {
		t.Errorf("site = %+v", s)
	}
}

func TestParseTLS(t *testing.T) {
	p := Parse(read(t, "tls-site"))
	if len(p.Servers) != 2 {
		t.Fatalf("servers = %+v", p.Servers)
	}
	tls := p.Servers[0]
	if !tls.SSL || !reflect.DeepEqual(tls.Names, []string{"shop.example.com", "example.com"}) ||
		!reflect.DeepEqual(tls.Certs, []string{"/etc/letsencrypt/live/shop.example.com/fullchain.pem"}) ||
		!reflect.DeepEqual(tls.Listen, []string{"443 ssl http2", "[::]:443 ssl"}) ||
		!reflect.DeepEqual(tls.ProxyPass, []string{"http://127.0.0.1:3000"}) {
		t.Errorf("tls = %+v", tls)
	}
	// "return 404;" inside if is not a listen or a name.
	plain := p.Servers[1]
	if plain.SSL || !reflect.DeepEqual(plain.Listen, []string{"80"}) || plain.Line != 18 {
		t.Errorf("redirect = %+v", plain)
	}
}

func TestParseTest(t *testing.T) {
	r := parseTest(read(t, "t-errors.txt"), 1)
	if r.OK || r.NeedsRoot || len(r.Issues) == 0 {
		t.Fatalf("errors = %+v", r)
	}
	want := Issue{Level: "emerg", Message: `unknown directive "bogus_directive"`, File: "/etc/nginx/conf.d/broken.conf", Line: 3}
	if r.Issues[0] != want {
		t.Errorf("first issue = %+v", r.Issues[0])
	}

	r = parseTest(read(t, "t-permission.txt"), 1)
	if r.OK || !r.NeedsRoot || len(r.Issues) != 1 || r.Issues[0].File != "" {
		t.Errorf("permission = %+v", r)
	}

	ok := "2026/10/27 [warn] 1#1: the \"user\" directive makes sense only if the master process runs with super-user privileges, ignored in /etc/nginx/nginx.conf:1\n" +
		"nginx: the configuration file /etc/nginx/nginx.conf syntax is ok\nnginx: configuration file /etc/nginx/nginx.conf test is successful\n"
	if r = parseTest(ok, 0); !r.OK || len(r.Issues) != 0 {
		t.Errorf("ok = %+v", r)
	}
}
