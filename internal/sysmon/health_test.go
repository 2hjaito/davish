package sysmon

import (
	"reflect"
	"testing"
)

func TestParseHealth(t *testing.T) {
	h := parseHealth(
		[]string{"davish-crash.service", "", "garbage"},
		[]string{"/davish-web 0 running", "/davish-crashloop 12 restarting", "bad line", "/x nope running"},
	)
	want := &Health{
		Failed: []string{"davish-crash.service"},
		Containers: []ContainerState{
			{"davish-web", 0, "running"},
			{"davish-crashloop", 12, "restarting"},
		},
	}
	if !reflect.DeepEqual(h, want) {
		t.Errorf("got %+v", h)
	}
}

func TestHealthServices(t *testing.T) {
	w := newHealthWatch()
	kinds := func(as []Alert) (out []string) {
		for _, a := range as {
			out = append(out, a.Kind+" "+a.Target)
		}
		return out
	}
	steps := []struct {
		failed []string
		want   []string
	}{
		{[]string{"a.service"}, []string{"service-failed a.service"}},
		{[]string{"a.service", "b.service"}, []string{"service-failed b.service"}},
		{[]string{"a.service", "b.service"}, nil}, // still failed: once is enough
		{[]string{"b.service"}, nil},              // a recovered
		{[]string{"a.service", "b.service"}, []string{"service-failed a.service"}},
	}
	for i, st := range steps {
		got := kinds(w.check("s1", float64(i*30), &Health{Failed: st.failed}))
		if !reflect.DeepEqual(got, st.want) {
			t.Errorf("step %d: got %v, want %v", i, got, st.want)
		}
	}
}

func TestHealthContainers(t *testing.T) {
	type step struct {
		at       float64
		restarts int
		status   string
		alert    bool
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"restarting state alerts at once", []step{{0, 12, "restarting", true}, {30, 13, "restarting", false}}},
		{"steady container never alerts", []step{{0, 2, "running", false}, {30, 2, "running", false}, {400, 2, "running", false}}},
		{"burst seen while running", []step{{0, 0, "running", false}, {30, 1, "running", false}, {60, 3, "running", true}}},
		{"slow restarts are no loop", []step{{0, 0, "running", false}, {200, 1, "running", false}, {400, 2, "running", false}, {650, 3, "running", false}}},
		{
			"alerts again after a quiet window",
			[]step{
				{0, 5, "restarting", true},
				{30, 6, "running", false},
				{360, 6, "running", false}, // a whole window quiet: re-armed
				{390, 7, "restarting", true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newHealthWatch()
			for i, s := range tt.steps {
				got := w.check("s1", s.at, &Health{Containers: []ContainerState{{"web", s.restarts, s.status}}})
				if (len(got) == 1) != s.alert {
					t.Fatalf("step %d: alerts %+v, want alert=%v", i, got, s.alert)
				}
				if s.alert && (got[0].ID != "s1:ctr:web" || got[0].Value != float64(s.restarts)) {
					t.Errorf("step %d: alert %+v", i, got[0])
				}
			}
		})
	}
}

func TestHealthForgetsRemovedContainers(t *testing.T) {
	w := newHealthWatch()
	w.check("s1", 0, &Health{Containers: []ContainerState{{"web", 1, "restarting"}}})
	w.check("s1", 30, &Health{})
	if got := w.check("s1", 60, &Health{Containers: []ContainerState{{"web", 0, "restarting"}}}); len(got) != 1 {
		t.Errorf("a new container under the same name should alert: %+v", got)
	}
}
