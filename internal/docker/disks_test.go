package docker

import (
	"reflect"
	"testing"
)

func TestParseMounts(t *testing.T) {
	out := "/web\tshop\t[{\"Type\":\"bind\",\"Source\":\"/srv/shop/uploads\",\"Destination\":\"/app/uploads\",\"Mode\":\"\",\"RW\":true}]\n" +
		"/db\tshop\t[{\"Type\":\"volume\",\"Name\":\"shop_pg\",\"Source\":\"/var/lib/docker/volumes/shop_pg/_data\",\"Destination\":\"/var/lib/postgresql/data\",\"Driver\":\"local\"}]\n" +
		"/cache\t<no value>\t[{\"Type\":\"tmpfs\",\"Source\":\"\",\"Destination\":\"/tmp\"}]\n" +
		"/lone\t<no value>\t[]\n" +
		"Error: No such object: 0123\n"
	got := parseMounts([]byte(out))
	want := []ContainerMount{
		{Container: "db", Project: "shop", Type: "volume", Name: "shop_pg", Source: "/var/lib/docker/volumes/shop_pg/_data", Dest: "/var/lib/postgresql/data"},
		{Container: "web", Project: "shop", Type: "bind", Source: "/srv/shop/uploads", Dest: "/app/uploads"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if len(parseMounts(nil)) != 0 {
		t.Fatal("empty output should give an empty list")
	}
}
