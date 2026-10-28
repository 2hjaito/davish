//go:build integration

package docker_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/docker"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func setup(t *testing.T) (*docker.Service, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	return docker.New(audit.NewRunner(m, audit.New(discard{}))), id
}

func find(t *testing.T, s *docker.Service, id, name string) docker.Container {
	t.Helper()
	list, err := s.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("%s not listed in %+v", name, list)
	return docker.Container{}
}

func TestListAndStats(t *testing.T) {
	s, id := setup(t)
	if c := find(t, s, id, "davish-web"); c.State != "running" {
		t.Errorf("web = %+v", c)
	}
	// A restart loop is caught in any of its phases: restarting, exited, or
	// running for the instant before it dies again. That it keeps
	// restarting is checked through its restart count in TestInspectAndLogs.
	switch c := find(t, s, id, "davish-crashloop"); c.State {
	case "restarting", "exited", "running":
	default:
		t.Errorf("crashloop = %+v", c)
	}
	web := find(t, s, id, "davish-web")
	stats, err := s.Stats(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range stats {
		if st.ID == web.ID {
			if st.MemUsed <= 0 || st.MemLimit <= 0 || st.PIDs == 0 {
				t.Errorf("web stats = %+v", st)
			}
			return
		}
	}
	t.Errorf("no stats for web in %+v", stats)
}

func TestInspectAndLogs(t *testing.T) {
	s, id := setup(t)
	ctx := context.Background()
	d, err := s.Inspect(ctx, id, "davish-crashloop")
	if err != nil {
		t.Fatal(err)
	}
	if d.Policy != "always" || d.Restarts == 0 || !strings.Contains(d.JSON, `"RestartCount"`) {
		t.Errorf("inspect = %+v", d)
	}

	logs, err := s.Logs(ctx, id, "davish-crashloop")
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Lines) == 0 || !logs.Lines[0].Stderr ||
		logs.Lines[0].Text != "Error: cannot open /etc/nginx/nginx.conf: no such file" {
		t.Errorf("logs = %+v", logs.Lines)
	}

	if _, err := s.Inspect(ctx, id, "davish-nope"); code(err) != apperr.NotFound {
		t.Errorf("inspect missing = %v", err)
	}
}

func TestRestartAndStop(t *testing.T) {
	s, id := setup(t)
	ctx := context.Background()
	t.Cleanup(func() { _ = s.Act(ctx, id, "davish-web", docker.ActionStart) })

	before, err := s.Inspect(ctx, id, "davish-web")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Act(ctx, id, "davish-web", docker.ActionRestart); err != nil {
		t.Fatal(err)
	}
	after, err := s.Inspect(ctx, id, "davish-web")
	if err != nil {
		t.Fatal(err)
	}
	if after.State != "running" || after.StartedAt == before.StartedAt {
		t.Errorf("restart: before %s, after %+v", before.StartedAt, after)
	}

	if err := s.Act(ctx, id, "davish-web", docker.ActionStop); err != nil {
		t.Fatal(err)
	}
	if c := find(t, s, id, "davish-web"); c.State != "exited" {
		t.Errorf("after stop = %+v", c)
	}
	if err := s.Act(ctx, id, "davish-web", docker.ActionStart); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for find(t, s, id, "davish-web").State != "running" {
		if time.Now().After(deadline) {
			t.Fatal("web did not start again")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestImagesAndLayers(t *testing.T) {
	s, id := setup(t)
	imgs, err := s.Images(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var bb *docker.Image
	for i := range imgs {
		if imgs[i].Repository == "davish/busybox" {
			bb = &imgs[i]
		}
	}
	if bb == nil || bb.Containers < 2 || bb.Size == 0 {
		t.Fatalf("busybox = %+v in %+v", bb, imgs)
	}
	d, err := s.Image(context.Background(), id, bb.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Layers) == 0 || len(d.Cmd) == 0 || d.Cmd[0] != "/bin/sh" {
		t.Fatalf("detail = %+v", d)
	}
	// In use by davish-web: docker refuses, and says so.
	p, _ := docker.PlanImage(bb.Ref())
	if err := s.RemoveImage(context.Background(), id, bb.Ref(), p.Command); err == nil {
		t.Fatal("removed an image in use")
	}
}
