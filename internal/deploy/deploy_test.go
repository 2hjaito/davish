package deploy

import (
	"strings"
	"testing"
)

func TestRenderUnit(t *testing.T) {
	got, err := RenderUnit(Unit{Name: "shop.service", ExecStart: "/usr/bin/node server.js", User: "www-data", WorkingDir: "/var/www/shop", Env: []string{"PORT=3000", `MSG=say "hi"`}, EnvFile: "/var/www/shop/.env"})
	if err != nil {
		t.Fatal(err)
	}
	want := `[Unit]
Description=shop (set up by Davish)
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/var/www/shop
ExecStart=/usr/bin/node server.js
Environment="PORT=3000"
Environment="MSG=say \"hi\""
EnvironmentFile=/var/www/shop/.env
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
`
	if got != want {
		t.Errorf("unit =\n%s", got)
	}
	for _, u := range []Unit{
		{Name: "a b", ExecStart: "/x", WorkingDir: "/a"},
		{Name: "a", ExecStart: "node x", WorkingDir: "/a"},
		{Name: "a", ExecStart: "/x\nExecStartPre=/bin/rm", WorkingDir: "/a"},
		{Name: "a", ExecStart: "/x", WorkingDir: "/a", User: "Root;"},
		{Name: "a", ExecStart: "/x", WorkingDir: "/a", Env: []string{"A"}},
	} {
		if _, err := RenderUnit(u); err == nil {
			t.Errorf("%+v accepted", u)
		}
	}
}

func TestMakeSystemd(t *testing.T) {
	p, err := Make(Spec{
		Source: SourceGit, RepoURL: "https://github.com/o/shop", Dir: "/var/www/shop", Run: RunSystemd, Port: 3000,
		Unit:    Unit{Name: "shop", ExecStart: "/usr/bin/npm start"},
		Domains: []string{"shop.example.com"}, SSL: true, Redirect: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	titles := []string{}
	for _, s := range p.Steps {
		titles = append(titles, s.Title)
	}
	if strings.Join(titles, ",") != "clone,systemd,nginx,https" {
		t.Errorf("steps = %v", titles)
	}
	if !strings.HasPrefix(p.Steps[0].Command, `sudo install -d -o "$USER" -- /var/www/shop && git clone -- https://github.com/o/shop /var/www/shop`) {
		t.Errorf("clone = %s", p.Steps[0].Command)
	}
	if !strings.Contains(p.Steps[1].Command, "| sudo tee -- /etc/systemd/system/shop.service > /dev/null && sudo systemctl daemon-reload && sudo systemctl enable --now shop") {
		t.Errorf("systemd = %s", p.Steps[1].Command)
	}
	if !strings.Contains(p.Steps[2].File, "proxy_pass http://127.0.0.1:3000;") || !strings.HasPrefix(p.Steps[2].Command, "printf %s ") {
		t.Errorf("nginx = %+v", p.Steps[2])
	}
	if p.Steps[3].Command != "sudo certbot --nginx -d shop.example.com --redirect" {
		t.Errorf("https = %s", p.Steps[3].Command)
	}
	if !strings.HasPrefix(p.Line, "echo '▶ 1/4  clone' && { sudo install") || !strings.HasSuffix(p.Line, "|| echo '✗ Stopped: the step above failed'") {
		t.Errorf("line = %s", p.Line)
	}
	want := "sudo rm -f -- /etc/nginx/sites-available/shop.example.com /etc/nginx/sites-enabled/shop.example.com && sudo systemctl reload nginx ; " +
		"sudo certbot delete --non-interactive --cert-name shop.example.com ; sudo systemctl disable --now shop ; " +
		"sudo rm -f -- /etc/systemd/system/shop.service ; sudo systemctl daemon-reload ; sudo rm -rf -- /var/www/shop"
	if p.Uninstall != want {
		t.Errorf("uninstall =\n%s\nwant\n%s", p.Uninstall, want)
	}
}

func TestMakeContainer(t *testing.T) {
	p, err := Make(Spec{Source: SourceImage, Image: "ghost:5", Run: RunContainer, Name: "blog", Port: 2368, ContainerPort: 2368, Domains: []string{"blog.example.com"}, ConfD: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Steps[0].Command != "docker run -d --name blog --restart unless-stopped -p 127.0.0.1:2368:2368 ghost:5" ||
		!strings.Contains(p.Uninstall, "docker rm -f blog") || !strings.Contains(p.Uninstall, "/etc/nginx/conf.d/blog.example.com.conf") {
		t.Errorf("plan = %+v", p)
	}
}

func TestMakeRejects(t *testing.T) {
	for _, s := range []Spec{
		{Source: SourceImage, Run: RunSystemd},
		{Source: SourceFolder, Dir: "rel", Run: RunStatic},
		{Source: SourceFolder, Dir: "/srv/a", Run: RunSystemd, Unit: Unit{Name: "a", ExecStart: "/x"}, Domains: []string{"a.com"}},
		{Source: SourceFolder, Dir: "/srv/a", Run: RunStatic},
	} {
		if _, err := Make(s); err == nil {
			t.Errorf("%+v accepted", s)
		}
	}
}
