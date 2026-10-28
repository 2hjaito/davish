package docker

import (
	"testing"
)

func TestParseImages(t *testing.T) {
	out := `{"Containers":"2","CreatedAt":"2026-10-26 09:12:40 +0000 UTC","CreatedSince":"2 days ago","ID":"sha256:cf41f4080901aa","Repository":"e-office","Size":"431MB","Tag":"test"}
{"Containers":"0","CreatedAt":"2026-10-28 09:12:40 +0000 UTC","CreatedSince":"1 hour ago","ID":"sha256:9a68dc314d7abb","Repository":"<none>","Size":"2.14GB","Tag":"<none>"}
`
	imgs, counted, err := parseImages([]byte(out))
	if err != nil || !counted || len(imgs) != 2 {
		t.Fatalf("%+v %v %v", imgs, counted, err)
	}
	if imgs[0].ID != "sha256:9a68dc314d7abb" || !imgs[0].Dangling || imgs[0].Ref() != "sha256:9a68dc314d7abb" {
		t.Errorf("newest = %+v", imgs[0])
	}
	if imgs[1].Ref() != "e-office:test" || imgs[1].Containers != 2 || imgs[1].Size != 431_000_000 {
		t.Errorf("e-office = %+v", imgs[1])
	}
}

func TestCountUsesOnOldDocker(t *testing.T) {
	out := `{"Containers":"N/A","CreatedAt":"a","ID":"sha256:000f7ff032e26c","Repository":"nginx","Size":"1MB","Tag":"latest"}` + "\n"
	imgs, counted, _ := parseImages([]byte(out))
	if counted {
		t.Fatal("N/A read as a count")
	}
	countUses(imgs, []string{"nginx", "nginx:latest", "000f7ff032e2", "redis"})
	if imgs[0].Containers != 3 {
		t.Errorf("containers = %d", imgs[0].Containers)
	}
}

func TestParseImageDetail(t *testing.T) {
	out := `{"CreatedBy":"/bin/sh -c #(nop)  CMD [\"node\"]","Size":"0B"}
{"CreatedBy":"/bin/sh -c addgroup -g 1000 node","Size":"158MB"}
{"CreatedBy":"ADD alpine-minirootfs.tar.gz / # buildkit","Size":"9.31MB"}
` + inspectSep + `
{"Architecture":"arm64","Os":"linux","Config":{"Cmd":["node"],"Env":["NODE_VERSION=22"],"ExposedPorts":{"3000/tcp":{}},"WorkingDir":"/app"}}
`
	d, err := parseImageDetail([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ADD alpine-minirootfs.tar.gz / # buildkit", "RUN addgroup -g 1000 node", `CMD ["node"]`}
	for i, w := range want {
		if d.Layers[i].CreatedBy != w {
			t.Errorf("layer %d = %q", i, d.Layers[i].CreatedBy)
		}
	}
	if d.Architecture != "arm64" || d.ExposedPorts[0] != "3000/tcp" || d.WorkingDir != "/app" || d.Cmd[0] != "node" {
		t.Errorf("detail = %+v", d)
	}
}

func TestPlanImage(t *testing.T) {
	p, err := PlanImage("ghcr.io/davitek/web:1.2")
	if err != nil || p.Command != "docker rmi ghcr.io/davitek/web:1.2" || p.Run != "docker run --rm -it ghcr.io/davitek/web:1.2" {
		t.Fatalf("%+v %v", p, err)
	}
	for _, bad := range []string{"x; rm -rf /", "-f", "a b", "../x"} {
		if _, err := PlanImage(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
