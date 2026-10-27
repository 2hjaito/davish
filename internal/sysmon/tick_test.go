package sysmon

import (
	"os"
	"testing"
)

func TestRate(t *testing.T) {
	base := Snapshot{
		CPUTotal: 1000, CPUIdle: 800,
		MemTotal: 16 << 30, MemAvailable: 10 << 30,
		Load1: 1.5, Uptime: 100,
		RxBytes: 1000, TxBytes: 500,
		DiskSize: 100 << 30, DiskUsed: 61 << 30, DiskAvail: 34 << 30,
	}
	tests := []struct {
		name string
		edit func(*Snapshot)
		want Tick
	}{
		{
			name: "quarter busy, 2s of traffic",
			edit: func(s *Snapshot) {
				s.CPUTotal += 400
				s.CPUIdle += 300
				s.Uptime += 2
				s.RxBytes += 4096
				s.TxBytes += 1024
			},
			want: Tick{CPU: 25, RxBps: 2048, TxBps: 512},
		},
		{
			name: "counters reset by a reboot",
			edit: func(s *Snapshot) {
				s.CPUTotal = 10
				s.CPUIdle = 5
				s.Uptime = 3
				s.RxBytes = 0
				s.TxBytes = 0
			},
			want: Tick{},
		},
		{
			name: "idle exceeding total after a counter glitch",
			edit: func(s *Snapshot) {
				s.CPUTotal += 100
				s.CPUIdle += 150
				s.Uptime += 2
			},
			want: Tick{CPU: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cur := base
			tt.edit(&cur)
			got := Rate("srv", base, cur)
			if got.CPU != tt.want.CPU || got.RxBps != tt.want.RxBps || got.TxBps != tt.want.TxBps {
				t.Errorf("cpu, rx, tx = %v, %v, %v; want %v, %v, %v",
					got.CPU, got.RxBps, got.TxBps, tt.want.CPU, tt.want.RxBps, tt.want.TxBps)
			}
			if got.ServerID != "srv" || got.MemUsed != 6<<30 || got.MemTotal != 16<<30 || got.Load1 != 1.5 {
				t.Errorf("tick = %+v", got)
			}
			// 61 of 95 usable GiB, rounded up like df.
			if got.DiskPct != 65 {
				t.Errorf("diskPct = %v; want 65", got.DiskPct)
			}
		})
	}
}

// The real fixture must give what top and df showed on the test sshd.
func TestRateOnRealOutput(t *testing.T) {
	f, err := os.Open("testdata/ubuntu.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r := NewReader(f)
	a, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	got := Rate("test", a, b)
	want := Tick{
		ServerID: "test",
		CPU:      0.1, // 3 busy jiffies of 3012
		MemUsed:  (8125796 - 7496792) * 1024,
		MemTotal: 8125796 * 1024,
		DiskPct:  4, // df printed 4%
		Load1:    0.06,
		Uptime:   387.86,
	}
	if got != want {
		t.Errorf("tick:\n got %+v\nwant %+v", got, want)
	}
}
