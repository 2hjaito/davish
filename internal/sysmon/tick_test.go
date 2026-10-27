package sysmon

import (
	"math"
	"testing"
)

func TestRate(t *testing.T) {
	base := Snapshot{
		CPU:    CPUTimes{User: 100, System: 100, Idle: 800},
		Cores:  []CPUTimes{{User: 50, Idle: 450}, {User: 50, System: 100, Idle: 350}},
		Mem:    Memory{Total: 16 << 30, Available: 10 << 30, Free: 4 << 30, Cache: 5 << 30, SwapTotal: 2 << 30, SwapFree: 1 << 30},
		Load1:  1.5,
		Uptime: 100,
		Ifaces: []NetCounters{{"eth0", 1000, 500}, {"eth1", 0, 100}},
		Mounts: []Mount{{Device: "/dev/vda1", Path: "/", Pct: 65}},
		Disks:  []DiskCounters{{"vda", 4096, 0}},
	}
	tests := []struct {
		name string
		edit func(*Snapshot)
		want Tick
	}{
		{
			name: "quarter busy, 2s of traffic",
			edit: func(s *Snapshot) {
				s.CPU.User += 200
				s.CPU.System += 50
				s.CPU.IOWait += 50
				s.CPU.Idle += 100
				s.Uptime += 2
				s.Ifaces = []NetCounters{{"eth0", 1000 + 4096, 500 + 1024}, {"eth1", 0, 100 + 2}}
				s.Disks = []DiskCounters{{"vda", 4096 + 8192, 2048}}
			},
			want: Tick{
				CPU: 62.5, CPUSplit: CPUSplit{User: 50, System: 12.5, IOWait: 12.5},
				RxBps: 2048, TxBps: 513, ReadBps: 4096, WriteBps: 1024,
			},
		},
		{
			name: "counters reset by a reboot",
			edit: func(s *Snapshot) {
				s.CPU = CPUTimes{User: 5, Idle: 5}
				s.Uptime = 3
				s.Ifaces = []NetCounters{{"eth0", 0, 0}}
				s.Disks = []DiskCounters{{"vda", 0, 0}}
			},
			want: Tick{},
		},
		{
			name: "idle exceeding total after a counter glitch",
			edit: func(s *Snapshot) {
				s.CPU.User -= 50
				s.CPU.Idle += 150
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
			if got.CPU != tt.want.CPU || got.CPUSplit != tt.want.CPUSplit {
				t.Errorf("cpu = %v %+v; want %v %+v", got.CPU, got.CPUSplit, tt.want.CPU, tt.want.CPUSplit)
			}
			if got.RxBps != tt.want.RxBps || got.TxBps != tt.want.TxBps ||
				got.ReadBps != tt.want.ReadBps || got.WriteBps != tt.want.WriteBps {
				t.Errorf("rx, tx, read, write = %v, %v, %v, %v; want %v, %v, %v, %v",
					got.RxBps, got.TxBps, got.ReadBps, got.WriteBps,
					tt.want.RxBps, tt.want.TxBps, tt.want.ReadBps, tt.want.WriteBps)
			}
			if got.ServerID != "srv" || got.MemUsed != 6<<30 || got.MemTotal != 16<<30 ||
				got.MemCache != 5<<30 || got.SwapUsed != 1<<30 || got.Load1 != 1.5 || got.DiskPct != 65 {
				t.Errorf("tick = %+v", got)
			}
		})
	}
}

func TestRatePerCore(t *testing.T) {
	prev := Snapshot{
		Cores:  []CPUTimes{{Idle: 100}, {Idle: 100}},
		Mounts: []Mount{{Path: "/"}},
	}
	cur := Snapshot{
		Cores:  []CPUTimes{{User: 30, Idle: 170}, {System: 100, Idle: 100}, {User: 1}},
		Mounts: []Mount{{Path: "/"}},
	}
	// A core that came online since the last reading has no rate yet.
	want := []float64{30, 100, 0}
	got := Rate("srv", prev, cur).Cores
	if len(got) != len(want) {
		t.Fatalf("cores = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cores = %v; want %v", got, want)
		}
	}
}

// The real fixture must give what top and df showed on the test sshd.
func TestRateOnRealOutput(t *testing.T) {
	r := openFixture(t)
	a, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	got := Rate("test", a, b)

	// 6 busy jiffies of 3009 across 15 cores; core 0 was busy 2 of 202.
	if got.CPU != 0.2 || got.CPUSplit != (CPUSplit{User: 0.1, System: 0.1}) || got.Cores[0] != 1 {
		t.Errorf("cpu = %v %+v, core 0 = %v", got.CPU, got.CPUSplit, got.Cores[0])
	}
	if got.MemUsed != (8125796-7503800)*1024 || got.MemTotal != 8125796*1024 || got.SwapUsed != 0 {
		t.Errorf("mem = %d of %d, swap %d", got.MemUsed, got.MemTotal, got.SwapUsed)
	}
	if got.DiskPct != 4 || got.Load5 != 0.03 || got.Procs != 339 || got.Uptime != 5632.34 {
		t.Errorf("tick = %+v", got)
	}
	// eth0 moved 924 bytes in and 8438 out over the 2.01s between readings.
	if len(got.Ifaces) != 1 || math.Abs(got.RxBps-924/2.01) > 1e-6 || math.Abs(got.TxBps-8438/2.01) > 1e-6 {
		t.Errorf("rx, tx = %v, %v; ifaces %+v", got.RxBps, got.TxBps, got.Ifaces)
	}
	if got.Ifaces[0].RxBps != got.RxBps || got.Ifaces[0].TxTotal != 5300135 {
		t.Errorf("eth0 = %+v", got.Ifaces[0])
	}
	if len(got.Disks) != 2 || got.ReadBps != 0 || got.WriteBps != 0 {
		t.Errorf("disks = %+v", got.Disks)
	}
}
