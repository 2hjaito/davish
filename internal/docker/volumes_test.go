package docker

import "testing"

func TestParseVolumes(t *testing.T) {
	out := `{"Images":[],"Volumes":[{"Driver":"local","Labels":"com.docker.compose.project=shop,x=y","Links":"1","Mountpoint":"/v/db","Name":"shop_db","Size":"438.6MB"},{"Driver":"local","Labels":"","Links":"0","Mountpoint":"/v/a","Name":"0a1b5a07404e73f21101ddc3cdadd630d36cc80f04883250ec280fa0a0a0a0a0","Size":"227.4kB"},{"Driver":"nfs","Labels":"","Links":"0","Name":"remote","Size":"N/A"}]}
` + volSep + `
{"CreatedAt":"2026-10-29T01:37:55Z","Name":"shop_db"}
`
	vs, err := parseVolumes([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 3 || vs[0].Name != "shop_db" || vs[0].Project != "shop" || vs[0].Containers != 1 || vs[0].Created != "2026-10-29T01:37:55Z" {
		t.Fatalf("volumes = %+v", vs)
	}
	if !vs[1].Anonymous || vs[1].Size != 227_400 {
		t.Errorf("anonymous = %+v", vs[1])
	}
	if vs[2].Size != -1 || vs[2].Anonymous {
		t.Errorf("remote = %+v", vs[2])
	}
}

func TestPlanVolumeAndRemoveContainer(t *testing.T) {
	p, err := PlanVolume("shop_db")
	if err != nil || p.Command != "docker volume rm shop_db" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := PlanVolume("a;b"); err == nil {
		t.Error("bad name accepted")
	}
	rm, err := PlanAction("web", ActionRemove)
	if err != nil || rm.Command != "docker rm web" || !rm.Confirm {
		t.Fatalf("%+v %v", rm, err)
	}
}

func TestComposeLabels(t *testing.T) {
	out := `{"ID":"1","Names":"shop-web-1","Labels":"com.docker.compose.project=shop,com.docker.compose.service=web","State":"running","RunningFor":"6 days ago"}` + "\n"
	cs, err := parsePS([]byte(out))
	if err != nil || cs[0].Project != "shop" || cs[0].Service != "web" || cs[0].RunningFor != "6 days ago" {
		t.Fatalf("%+v %v", cs, err)
	}
}
