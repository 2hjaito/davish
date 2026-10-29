package docker

import "testing"

func TestSpecFromInspect(t *testing.T) {
	out := `[{"Name":"/web","Config":{"Image":"nginx:1.27","Env":["PATH=/usr/bin","APP=1"],"Cmd":["nginx","-g","daemon off;"]},
"HostConfig":{"RestartPolicy":{"Name":"unless-stopped"},"NetworkMode":"app","PortBindings":{"80/tcp":[{"HostIp":"","HostPort":"8080"}],"443/tcp":[{"HostIp":"127.0.0.1","HostPort":"8443"}]}},
"Mounts":[{"Type":"volume","Name":"webdata","Source":"/var/lib/docker/volumes/webdata/_data","Destination":"/data","RW":true},{"Type":"bind","Source":"/srv/conf","Destination":"/etc/nginx/conf.d","RW":false}]}]
--davish-image--
{"Env":["PATH=/usr/bin"],"Cmd":["nginx","-g","daemon off;"]}
`
	spec, err := specFromInspect([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	p, err := PlanRun(spec)
	if err != nil {
		t.Fatal(err)
	}
	want := "docker run -d --name web --restart unless-stopped --network app -p 127.0.0.1:8443:443 -p 0.0.0.0:8080:80 -e APP=1 -v webdata:/data -v /srv/conf:/etc/nginx/conf.d:ro nginx:1.27"
	if p.Command != want {
		t.Errorf("command =\n%s\nwant\n%s", p.Command, want)
	}
	if len(p.Public) != 1 {
		t.Errorf("public = %v", p.Public)
	}
}
