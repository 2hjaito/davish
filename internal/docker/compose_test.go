package docker

import (
	"strings"
	"testing"
)

func TestComposeCommand(t *testing.T) {
	c, err := ComposeCommand("shop", "/srv/shop app", "/srv/shop app/compose.yaml", "up")
	if want := "docker compose --project-directory '/srv/shop app' -f '/srv/shop app/compose.yaml' -p shop up -d"; err != nil || c != want {
		t.Errorf("up = %s, %v", c, err)
	}
	if c, _ := ComposeCommand("shop", "", "", "down"); c != "docker compose -p shop down" {
		t.Errorf("down = %s", c)
	}
	for _, args := range [][4]string{{"shop", "", "", "up"}, {"Shop;", "/a", "", "up"}, {"shop", "/a", "", "rm"}} {
		if _, err := ComposeCommand(args[0], args[1], args[2], args[3]); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}

func TestPlanStack(t *testing.T) {
	for _, tmpl := range StackTemplates {
		p, err := PlanStack("/srv/app/", tmpl)
		if err != nil {
			t.Fatal(err)
		}
		if p.Path != "/srv/app/compose.yaml" || !strings.HasPrefix(p.Content, "services:") ||
			!strings.HasPrefix(p.Command, "mkdir -p -- /srv/app && (set -C && printf %s ") ||
			!strings.HasSuffix(p.Command, " > /srv/app/compose.yaml)") {
			t.Errorf("%s: %+v", tmpl, p)
		}
	}
	for _, dir := range []string{"rel", "/", "/a'b"} {
		if _, err := PlanStack(dir, "blank"); err == nil {
			t.Errorf("%q accepted", dir)
		}
	}
}
