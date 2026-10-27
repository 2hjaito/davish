package sysmon

import (
	"reflect"
	"testing"
	"time"
)

func TestAlerter(t *testing.T) {
	// reading is one tick: memory percent, / percent, uptime.
	type reading struct{ mem, disk, at float64 }
	restart := reading{mem: -1} // a new collector starts
	memAlert := func(v float64) Alert {
		return Alert{ID: "s1:mem", Level: "warn", ServerID: "s1", Kind: AlertMemHigh, Value: v, Threshold: 90}
	}
	diskAlert := func(v float64) Alert {
		return Alert{ID: "s1:disk:/", Level: "warn", ServerID: "s1", Kind: AlertDiskHigh, Value: v, Threshold: 85, Mount: "/"}
	}
	tests := []struct {
		name  string
		ticks []reading
		want  []Alert
	}{
		{
			name:  "under thresholds",
			ticks: []reading{{50, 60, 0}, {89, 84, 120}},
		},
		{
			name:  "over for less than a minute",
			ticks: []reading{{95, 90, 0}, {95, 90, 58}},
		},
		{
			name:  "over for a minute, notified once",
			ticks: []reading{{95, 88, 0}, {95, 88, 30}, {96, 88, 60}, {96, 88, 62}, {96, 88, 300}},
			want:  []Alert{memAlert(96), diskAlert(88)},
		},
		{
			name:  "spike that drops back resets the clock",
			ticks: []reading{{95, 0, 0}, {80, 0, 30}, {95, 0, 40}, {95, 0, 90}},
		},
		{
			name:  "hovering just under the threshold does not repeat",
			ticks: []reading{{95, 0, 0}, {95, 0, 60}, {88, 0, 62}, {95, 0, 64}, {95, 0, 200}},
			want:  []Alert{memAlert(95)},
		},
		{
			name:  "falling well under re-arms",
			ticks: []reading{{95, 0, 0}, {95, 0, 60}, {80, 0, 62}, {92, 0, 64}, {92, 0, 124}},
			want:  []Alert{memAlert(95), memAlert(92)},
		},
		{
			name:  "time unwatched does not count",
			ticks: []reading{{95, 0, 0}, restart, {95, 0, 30}, {95, 0, 70}, {95, 0, 90}},
			want:  []Alert{memAlert(95)},
		},
		{
			name:  "a new collector does not repeat a given alert",
			ticks: []reading{{95, 0, 0}, {95, 0, 60}, restart, {95, 0, 600}, {95, 0, 700}},
			want:  []Alert{memAlert(95)},
		},
		{
			name:  "reboot restarts the clock",
			ticks: []reading{{95, 0, 1000}, {95, 0, 10}, {95, 0, 40}, {95, 0, 70}},
			want:  []Alert{memAlert(95)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAlerter(Thresholds{MemPct: 90, DiskPct: 85, For: time.Minute})
			var got []Alert
			for _, r := range tt.ticks {
				if r == restart {
					a.resume()
					continue
				}
				got = append(got, a.check(Tick{
					ServerID: "s1", Uptime: r.at,
					MemUsed: uint64(r.mem * 10), MemTotal: 1000,
					Mounts: []Mount{{Path: "/", Pct: r.disk}},
				})...)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("alerts = %+v\nwant     %+v", got, tt.want)
			}
		})
	}
}

func TestAlerterPerMount(t *testing.T) {
	a := newAlerter(Thresholds{DiskPct: 85})
	got := a.check(Tick{ServerID: "s1", Mounts: []Mount{{Path: "/", Pct: 40}, {Path: "/data", Pct: 97}}})
	if len(got) != 1 || got[0].ID != "s1:disk:/data" || got[0].Mount != "/data" {
		t.Errorf("alerts = %+v, want one for /data", got)
	}
}

func TestSetThresholds(t *testing.T) {
	s := New(nil, nil, nil, Options{})
	s.resumeAlerts("s1")
	tick := func(mem, at float64) []Alert {
		return s.checkAlerts(Tick{ServerID: "s1", MemUsed: uint64(mem), MemTotal: 100, Uptime: at})
	}
	// 80% is fine by default…
	if got := append(tick(80, 0), tick(80, 120)...); len(got) != 0 {
		t.Fatalf("default thresholds alerted: %+v", got)
	}
	// …and over a lowered threshold for its duration.
	s.SetThresholds(Thresholds{MemPct: 75, DiskPct: 85, For: 30 * time.Second})
	tick(80, 200)
	if got := tick(80, 231); len(got) != 1 || got[0].Threshold != 75 {
		t.Errorf("after lowering: %+v", got)
	}
}
