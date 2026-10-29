package nginx

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name, got string) {
	t.Helper()
	file := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(file) // #nosec G304 -- test fixture
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs:\n%s", name, got)
	}
}

func TestRenderSite(t *testing.T) {
	cases := map[string]NewSite{
		"static.conf": {Names: []string{"Example.com", " www.example.com", "example.com"}, Type: TypeStatic, Root: "/var/www/example/"},
		"proxy.conf":  {Names: []string{"app.example.com"}, Type: TypeProxy, Upstream: "127.0.0.1:3000"},
		"ws.conf":     {Names: []string{"*.example.com"}, Type: TypeProxy, Upstream: "http://127.0.0.1:8080/", Websocket: true},
		"php.conf":    {Names: []string{"blog.example.com"}, Type: TypePHP, Root: "/var/www/blog", PHPSocket: "/run/php/php8.3-fpm.sock"},
	}
	for name, n := range cases {
		got, err := RenderSite(n)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		golden(t, name, got)
	}
}

func TestRenderSiteRejects(t *testing.T) {
	for _, n := range []NewSite{
		{Type: TypeStatic, Root: "/var/www"},
		{Names: []string{"a.com;"}, Type: TypeStatic, Root: "/var/www"},
		{Names: []string{"a.com"}, Type: TypeStatic, Root: "/"},
		{Names: []string{"a.com"}, Type: TypeStatic, Root: "/var/www; rm -rf /"},
		{Names: []string{"a.com"}, Type: TypeProxy, Upstream: "127.0.0.1:3000; }"},
		{Names: []string{"a.com"}, Type: TypePHP, Root: "/var/www", PHPSocket: "x"},
		{Names: []string{"a.com"}, Type: "other"},
	} {
		if _, err := RenderSite(n); err == nil {
			t.Errorf("%+v rendered", n)
		}
	}
}

func TestPlanCreate(t *testing.T) {
	n, _ := NewSite{Names: []string{"a.com"}, Type: TypeStatic, Root: "/var/www/a b"}.check()
	p := planCreate(n, "server {}", false)
	want := "mkdir -p -- '/var/www/a b' && printf %s 'server {}\n' | tee -- /etc/nginx/sites-available/a.com > /dev/null && ln -s -- /etc/nginx/sites-available/a.com /etc/nginx/sites-enabled/a.com && { nginx -t && systemctl reload nginx || { rm -- /etc/nginx/sites-enabled/a.com; false; }; }"
	if p.Command != want {
		t.Errorf("command = %s", p.Command)
	}
	if !strings.HasPrefix(p.Sudo, "sudo mkdir") || !strings.Contains(p.Sudo, "sudo nginx -t && sudo systemctl reload nginx || { sudo rm") {
		t.Errorf("sudo = %s", p.Sudo)
	}

	n, _ = NewSite{Names: []string{"*.a.com"}, Type: TypeProxy, Upstream: "127.0.0.1:1"}.check()
	p = planCreate(n, "x\n", true)
	if p.Path != "/etc/nginx/conf.d/wildcard.a.com.conf" || !strings.Contains(p.Command, "|| { rm -- /etc/nginx/conf.d/wildcard.a.com.conf; false; }") {
		t.Errorf("conf.d plan = %+v", p)
	}
}

func TestCreateResult(t *testing.T) {
	r, err := createResult("nginx: [emerg] 12#12: unknown directive \"x\" in /etc/nginx/sites-enabled/a.com:3\nnginx: configuration file /etc/nginx/nginx.conf test failed\n", 1, "s")
	if err != nil || r.Created || r.Test == nil || r.Test.OK || len(r.Test.Issues) != 1 {
		t.Errorf("failed test = %+v, %v", r, err)
	}
	r, err = createResult("tee: /etc/nginx/sites-available/a.com: Permission denied\n", 1, "s")
	if err != nil || !r.NeedsRoot || r.Sudo != "s" {
		t.Errorf("denied = %+v, %v", r, err)
	}
	r, err = createResult("nginx: configuration file /etc/nginx/nginx.conf test is successful\n", 0, "s")
	if err != nil || !r.Created {
		t.Errorf("ok = %+v, %v", r, err)
	}
}
