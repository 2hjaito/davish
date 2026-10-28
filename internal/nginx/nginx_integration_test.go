//go:build integration

package nginx_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/nginx"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func setup(t *testing.T) (*nginx.Service, *conn.Manager, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	s := nginx.New(audit.NewRunner(m, audit.New(discard{})), m, discard{})
	t.Cleanup(func() {
		testsshd.Run(t, m, id, "ln -sf /etc/nginx/sites-available/davish-site /etc/nginx/sites-enabled/davish-site && systemctl reload nginx")
	})
	return s, m, id
}

func status(t *testing.T, m *conn.Manager, id string) string {
	t.Helper()
	return testsshd.Run(t, m, id, "curl -s -o /dev/null -w '%{http_code}' -H 'Host: davish.test' http://localhost/davish.html")
}

// eventually waits for the page to answer want: a reload only signals the
// master process, which swaps workers a moment later.
func eventually(t *testing.T, m *conn.Manager, id, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := status(t, m, id)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("page answers %s, want %s", got, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func site(t *testing.T, s *nginx.Service, id, name string) nginx.Site {
	t.Helper()
	sites, err := s.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range sites {
		if x.Name == name && x.Kind == nginx.KindSite {
			return x
		}
	}
	t.Fatalf("%s not in %+v", name, sites)
	return nginx.Site{}
}

func TestListsSites(t *testing.T) {
	s, _, id := setup(t)
	on := site(t, s, id, "davish-site")
	if !on.Enabled || !on.Toggleable || on.Servers[0].Names[0] != "davish.test" || on.Upstreams[0] != "davish_backend" {
		t.Errorf("davish-site = %+v", on)
	}
	if off := site(t, s, id, "davish-off"); off.Enabled {
		t.Errorf("davish-off = %+v", off)
	}
	if def := site(t, s, id, "default"); !def.Enabled {
		t.Errorf("default = %+v", def)
	}
}

// The M8 acceptance: turn a site off, nginx -t runs before the reload,
// and the site answers 404 after it.
func TestDisableTestReload(t *testing.T) {
	s, m, id := setup(t)
	ctx := context.Background()
	eventually(t, m, id, "200")

	plan, err := s.PlanToggle(ctx, id, "davish-site", false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != "rm -- /etc/nginx/sites-enabled/davish-site" {
		t.Errorf("plan = %+v", plan)
	}
	if err := s.Toggle(ctx, id, "davish-site", false, plan.Command); err != nil {
		t.Fatal(err)
	}
	// Running the same plan again: the link is gone, so it is stale.
	if err := s.Toggle(ctx, id, "davish-site", false, plan.Command); err == nil {
		t.Error("second disable accepted")
	}

	r, err := s.Reload(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Test.OK || !r.Reloaded {
		t.Fatalf("reload = %+v", r)
	}
	eventually(t, m, id, "404")

	plan, err = s.PlanToggle(ctx, id, "davish-site", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Toggle(ctx, id, "davish-site", true, plan.Command); err != nil {
		t.Fatal(err)
	}
	if r, err := s.Reload(ctx, id); err != nil || !r.Reloaded {
		t.Fatalf("reload = %+v, %v", r, err)
	}
	eventually(t, m, id, "200")
}

func TestBrokenConfigIsNotReloaded(t *testing.T) {
	s, m, id := setup(t)
	eventually(t, m, id, "200")
	testsshd.Run(t, m, id, "printf 'server {\\n  bogus_directive on;\\n}\\n' > /etc/nginx/conf.d/davish-broken.conf")
	t.Cleanup(func() { testsshd.Run(t, m, id, "rm -f /etc/nginx/conf.d/davish-broken.conf") })

	r, err := s.Reload(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Reloaded || r.Test.OK || r.NeedsRoot || len(r.Test.Issues) == 0 {
		t.Fatalf("reload = %+v", r)
	}
	is := r.Test.Issues[0]
	if is.File != "/etc/nginx/conf.d/davish-broken.conf" || is.Line != 2 {
		t.Errorf("issue = %+v", is)
	}
	time.Sleep(time.Second)
	if got := status(t, m, id); got != "200" {
		t.Errorf("site broke: %s", got)
	}
}

func TestRefusesBadNames(t *testing.T) {
	s, _, id := setup(t)
	for _, name := range []string{"../nginx.conf", "-rf", "", "nope"} {
		_, err := s.PlanToggle(context.Background(), id, name, true)
		var ae *apperr.Error
		if !errors.As(err, &ae) {
			t.Errorf("PlanToggle(%q) = %v", name, err)
		}
	}
}

func TestPlanRemove(t *testing.T) {
	s, _, id := setup(t)
	ctx := context.Background()
	p, err := s.PlanRemove(ctx, id, "/etc/nginx/sites-available/davish-site", false)
	if err != nil {
		t.Fatal(err)
	}
	if want := "rm -- /etc/nginx/sites-enabled/davish-site && rm -- /etc/nginx/sites-available/davish-site"; !p.Enabled || p.Command != want {
		t.Errorf("plan = %+v", p)
	}
	if _, err := s.PlanRemove(ctx, id, "/etc/nginx/nginx.conf", false); err == nil {
		t.Error("nginx.conf planned for removal")
	}
	if err := s.Remove(ctx, id, "/etc/nginx/sites-available/davish-site", false, "rm -- /etc/nginx/sites-available/davish-site"); err == nil {
		t.Error("a command other than the plan's was run")
	}
}
