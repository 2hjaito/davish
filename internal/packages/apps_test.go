package packages

import (
	"strings"
	"testing"
)

func TestParseProbe(t *testing.T) {
	out := strings.Join([]string{
		"bin\tnginx\t/usr/sbin/nginx\tnginx version: nginx/1.24.0 (Ubuntu)",
		"bin\tmysqld\t/usr/sbin/mysqld\t/usr/sbin/mysqld  Ver 8.0.36-0ubuntu0.22.04.1 for Linux on x86_64 ((Ubuntu))",
		"bin\tpsql\t/usr/bin/psql\tpsql (PostgreSQL) 16.2 (Ubuntu 16.2-1ubuntu4)",
		"bin\tredis-server\t/usr/bin/redis-server\tRedis server v=7.0.15 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64 build=1",
		"bin\tdocker\t/usr/bin/docker\tDocker version 24.0.7, build 24.0.7-0ubuntu4",
		"bin\tnode\t/home/dangth/.nvm/versions/node/v20.11.1/bin/node\tv20.11.1",
		"bin\tphp\t/usr/bin/php\tPHP 8.3.6 (cli) (built: Apr 15 2024 19:21:47) (NTS)",
		"bin\tpython3\t/usr/bin/python3\tPython 3.12.3",
		"bin\tpip3\t/usr/bin/pip3\tpip 24.0 from /usr/lib/python3/dist-packages/pip (python 3.12)",
		"bin\tgo\t/usr/local/go/bin/go\tgo version go1.22.2 linux/amd64",
		"bin\tjava\t/usr/bin/java\topenjdk version \"17.0.10\" 2024-01-16",
		"cfg\t/etc/nginx/nginx.conf",
		"cfg\t/etc/mysql/my.cnf",
		"nvm\tv18.19.0",
		"nvm\tv20.11.1",
		"fpm\tphp-fpm8.3",
	}, "\n")
	apps := parseProbe([]byte(out))
	got := map[string]App{}
	for _, a := range apps {
		got[a.ID] = a
	}
	versions := map[string]string{
		"nginx": "1.24.0", "mysql": "8.0.36-0ubuntu0.22.04.1", "postgres": "16.2", "redis": "7.0.15",
		"docker": "24.0.7", "node": "20.11.1", "php": "8.3.6", "python": "3.12.3", "pip": "24.0",
		"go": "1.22.2", "java": "17.0.10",
	}
	for id, v := range versions {
		if got[id].Version != v {
			t.Errorf("%s version = %q, want %q", id, got[id].Version, v)
		}
	}
	if apps[0].ID != "nginx" || got["nginx"].Configs[0] != "/etc/nginx/nginx.conf" || got["nginx"].Units[0] != "nginx.service" {
		t.Errorf("nginx = %+v", got["nginx"])
	}
	if v := got["node"].Versions; len(v) != 2 || v[0] != "18.19.0" {
		t.Errorf("nvm versions = %v", v)
	}
	if u := got["php"].Units; len(u) != 1 || u[0] != "php8.3-fpm.service" {
		t.Errorf("php units = %v", u)
	}
	if _, ok := got["git"]; ok {
		t.Error("git reported without being found")
	}
}

func TestProbeLineIsConstant(t *testing.T) {
	for _, want := range []string{"nginx:-v", "java:-version", "/etc/nginx/nginx.conf", `.nvm/versions/node`, "timeout 5"} {
		if !strings.Contains(probeLine, want) {
			t.Errorf("probe lacks %q", want)
		}
	}
	if strings.Count(probeLine, "/etc/mysql/my.cnf") != 1 {
		t.Error("config paths are not deduplicated")
	}
}

func TestParseNpmGlobals(t *testing.T) {
	got, err := parseNpmGlobals([]byte(`{"dependencies":{"pm2":{"version":"5.3.1"},"corepack":{"version":"0.24.0"}}}`))
	if err != nil || len(got) != 2 || got[0].Name != "corepack" || got[1].Version != "5.3.1" {
		t.Errorf("got %+v, %v", got, err)
	}
	if got, err := parseNpmGlobals(nil); err != nil || len(got) != 0 {
		t.Errorf("no npm output = %+v, %v", got, err)
	}
}
