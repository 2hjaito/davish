package docker

import "testing"

func TestPlanRun(t *testing.T) {
	p, err := PlanRun(RunSpec{
		Name:    "web",
		Image:   "nginx:1.27",
		Ports:   []PortMap{{HostPort: 8080, ContainerPort: 80}, {Host: "0.0.0.0", HostPort: 53, ContainerPort: 53, UDP: true}},
		Env:     []string{"A=1", "", "MSG=hello world"},
		EnvFile: "/srv/web/.env",
		Mounts:  []Mount{{Source: "webdata", Target: "/data"}, {Source: "/srv/web/conf", Target: "/etc/nginx/conf.d", ReadOnly: true}},
		Restart: "unless-stopped",
		Network: "app",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "docker run -d --name web --restart unless-stopped --network app -p 127.0.0.1:8080:80 -p 0.0.0.0:53:53/udp -e A=1 -e 'MSG=hello world' --env-file /srv/web/.env -v webdata:/data -v /srv/web/conf:/etc/nginx/conf.d:ro nginx:1.27"
	if p.Command != want {
		t.Errorf("command =\n%s\nwant\n%s", p.Command, want)
	}
	if len(p.Public) != 1 || p.Public[0] != "0.0.0.0:53:53/udp" {
		t.Errorf("public = %v", p.Public)
	}
}

func TestPlanRunRejects(t *testing.T) {
	for _, s := range []RunSpec{
		{Image: "x;rm"},
		{Image: "nginx", Name: "-x"},
		{Image: "nginx", Restart: "sometimes"},
		{Image: "nginx", Ports: []PortMap{{ContainerPort: 0}}},
		{Image: "nginx", Ports: []PortMap{{Host: "example.com", ContainerPort: 80}}},
		{Image: "nginx", Env: []string{"1A=x"}},
		{Image: "nginx", Mounts: []Mount{{Source: "data", Target: "rel"}}},
		{Image: "nginx", EnvFile: "rel/.env"},
	} {
		if _, err := PlanRun(s); err == nil {
			t.Errorf("%+v accepted", s)
		}
	}
}

func TestPullCommand(t *testing.T) {
	if c, err := PullCommand("ghcr.io/a/b:1"); err != nil || c != "docker pull ghcr.io/a/b:1" {
		t.Errorf("pull = %s, %v", c, err)
	}
}
