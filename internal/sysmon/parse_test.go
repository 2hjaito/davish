package sysmon

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func openFixture(t *testing.T) *Reader {
	t.Helper()
	f, err := os.Open("testdata/ubuntu.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return NewReader(f)
}

// testdata/ubuntu.txt is two snapshots of collect.sh on the Ubuntu test sshd:
// a container with 15 cores whose / is an overlay on the host's /dev/vda1.
func TestReaderParsesRealOutput(t *testing.T) {
	r := openFixture(t)
	snap, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}

	if want := (CPUTimes{User: 8599, System: 7140, Idle: 8422285, IOWait: 117, SoftIRQ: 728}); snap.CPU != want {
		t.Errorf("cpu = %+v; want %+v", snap.CPU, want)
	}
	if len(snap.Cores) != 15 {
		t.Errorf("%d cores; want 15", len(snap.Cores))
	}
	if want := (CPUTimes{User: 587, System: 502, Idle: 561058, IOWait: 12, SoftIRQ: 571}); snap.Cores[0] != want {
		t.Errorf("cpu0 = %+v; want %+v", snap.Cores[0], want)
	}
	wantMem := Memory{
		Total:     8125796 * 1024,
		Free:      7250720 * 1024,
		Available: 7500692 * 1024,
		Cache:     (38468 + 351208 + 15520) * 1024,
		SwapTotal: 1048572 * 1024,
		SwapFree:  1048572 * 1024,
	}
	if snap.Mem != wantMem {
		t.Errorf("mem = %+v; want %+v", snap.Mem, wantMem)
	}
	if snap.Load1 != 0.02 || snap.Load5 != 0.04 || snap.Load15 != 0 || snap.Procs != 339 || snap.Uptime != 5630.33 {
		t.Errorf("load, procs, uptime = %v %v %v, %d, %v", snap.Load1, snap.Load5, snap.Load15, snap.Procs, snap.Uptime)
	}
	// lo and the kernel's idle tunnel devices are left out.
	if want := []NetCounters{{Name: "eth0", Rx: 307018, Tx: 5291697}}; !reflect.DeepEqual(snap.Ifaces, want) {
		t.Errorf("ifaces = %+v; want %+v", snap.Ifaces, want)
	}
	// /dev/vda1 on /etc/hosts is the same filesystem as the overlay on /.
	wantMounts := []Mount{{
		Device: "overlay", Path: "/",
		Size: 954925484 * 1024, Used: 28681852 * 1024, Avail: 877662360 * 1024, Pct: 4,
	}}
	if !reflect.DeepEqual(snap.Mounts, wantMounts) {
		t.Errorf("mounts = %+v; want %+v", snap.Mounts, wantMounts)
	}
	wantDisks := []DiskCounters{
		{Name: "vda", Read: 66378 * 512, Written: 22960 * 512},
		{Name: "vdb", Read: 533256 * 512},
	}
	if !reflect.DeepEqual(snap.Disks, wantDisks) {
		t.Errorf("disks = %+v; want %+v", snap.Disks, wantDisks)
	}
	if len(snap.GPUs) != 0 {
		t.Errorf("gpus = %+v; want none", snap.GPUs)
	}

	if _, err := r.Next(); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("after the last snapshot: err = %v; want io.EOF", err)
	}
}

func TestReaderSkipsMalformedSnapshot(t *testing.T) {
	good := "@@stat\ncpu  10 0 10 80 0 0 0 0 0 0\n" +
		"@@meminfo\nMemTotal: 1000 kB\nMemAvailable: 400 kB\n" +
		"@@loadavg\n0.50 0.40 0.30 1/100 42\n" +
		"@@uptime\n100.00 300.00\n" +
		"@@netdev\nInter-| Receive\n face |bytes\n  eth0: 1 0 0 0 0 0 0 0 2 0 0 0 0 0 0 0\n" +
		"@@df\nFilesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 100 40 60 40% /\n" +
		"@@end\n"
	bad := "@@stat\nintr 1 2 3\n@@end\n"
	r := NewReader(strings.NewReader(bad + good))

	if _, err := r.Next(); !errors.Is(err, ErrMalformed) {
		t.Fatalf("bad snapshot: err = %v; want ErrMalformed", err)
	}
	snap, err := r.Next()
	if err != nil {
		t.Fatalf("good snapshot after a bad one: %v", err)
	}
	// No diskstats or gpu sections: older or locked-down servers still work.
	if snap.Load1 != 0.5 || snap.Mem.Available != 400*1024 || len(snap.Disks) != 0 || snap.GPUs == nil {
		t.Errorf("snapshot = %+v", snap)
	}
}

func TestParseMeminfoWithoutMemAvailable(t *testing.T) {
	// Kernels before 3.14 (CentOS 6 and friends) have no MemAvailable.
	lines := []string{"MemTotal: 2000 kB", "MemFree: 500 kB", "Buffers: 100 kB", "Cached: 300 kB"}
	m, err := parseMeminfo(lines)
	if err != nil {
		t.Fatal(err)
	}
	if m.Total != 2000*1024 || m.Available != 900*1024 || m.Cache != 400*1024 {
		t.Errorf("mem = %+v", m)
	}
}

func TestParseNetDev(t *testing.T) {
	header := []string{
		"Inter-|   Receive                                                |  Transmit",
		" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed",
	}
	tests := []struct {
		name  string
		lines []string
		want  []NetCounters
	}{
		{
			name: "every physical interface",
			lines: []string{
				"  eth0: 100 1 0 0 0 0 0 0 200 1 0 0 0 0 0 0",
				"  eth1: 10 1 0 0 0 0 0 0 20 1 0 0 0 0 0 0",
			},
			want: []NetCounters{{"eth0", 100, 200}, {"eth1", 10, 20}},
		},
		{
			name:  "name touching a large counter",
			lines: []string{"enp0s3:4294967296 9 0 0 0 0 0 0 7 1 0 0 0 0 0 0"},
			want:  []NetCounters{{"enp0s3", 4294967296, 7}},
		},
		{
			name: "skips loopback, container links and idle devices",
			lines: []string{
				"    lo: 999 1 0 0 0 0 0 0 999 1 0 0 0 0 0 0",
				"docker0: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				"veth1a2b: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				"br-3f9c: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				" tunl0: 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
				"  eth0: 5 1 0 0 0 0 0 0 6 1 0 0 0 0 0 0",
			},
			want: []NetCounters{{"eth0", 5, 6}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNetDev(append(header, tt.lines...))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v; want %+v", got, tt.want)
			}
		})
	}
}

func TestParseDf(t *testing.T) {
	const head = "Filesystem 1024-blocks Used Available Capacity Mounted on"
	tests := []struct {
		name  string
		lines []string
		want  []string // device on path
	}{
		{
			name:  "plain device",
			lines: []string{head, "/dev/vda1 1000 250 750 25% /"},
			want:  []string{"/dev/vda1 on /"},
		},
		{
			name:  "device name with a space",
			lines: []string{head, "my disk 1000 300 700 30% /"},
			want:  []string{"my disk on /"},
		},
		{
			name: "real disks after /, no pseudo or repeated filesystems",
			lines: []string{
				head,
				"udev 100 0 100 0% /dev",
				"tmpfs 100 1 99 1% /run",
				"/dev/sda2 1000 400 600 40% /",
				"/dev/loop3 50 50 0 100% /snap/core/1",
				"/dev/sda1 500 10 490 3% /boot/efi",
				"/dev/sdb1 9000 10 8990 1% /data",
				"/dev/sdb1 9000 10 8990 1% /srv/data",
			},
			want: []string{"/dev/sda2 on /", "/dev/sda1 on /boot/efi", "/dev/sdb1 on /data"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts, err := parseDf(tt.lines)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, m := range mounts {
				got = append(got, m.Device+" on "+m.Path)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q; want %q", got, tt.want)
			}
		})
	}
	if _, err := parseDf([]string{head}); err == nil {
		t.Error("df without a data line parsed")
	}
	if _, err := parseDf([]string{head, "/dev/sdb1 9000 10 8990 1% /data"}); err == nil {
		t.Error("df without / parsed")
	}
}

func TestParseDiskstatsCountsWholeDisksOnly(t *testing.T) {
	stats := []string{
		"   8       0 sda 100 0 10 0 50 0 20 0 0 0 0",
		"   8       1 sda1 100 0 10 0 50 0 20 0 0 0 0",
		" 259       0 nvme0n1 1 0 4 0 1 0 8 0 0 0 0",
		" 259       1 nvme0n1p1 1 0 4 0 1 0 8 0 0 0 0",
		" 253       0 dm-0 9 0 99 0 9 0 99 0 0 0 0",
		"   7       0 loop0 9 0 99 0 9 0 99 0 0 0 0",
	}
	block := []string{"sda", "nvme0n1", "dm-0", "loop0"}
	want := []DiskCounters{{"sda", 10 * 512, 20 * 512}, {"nvme0n1", 4 * 512, 8 * 512}}
	if got := parseDiskstats(stats, block); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

func TestParseGPUs(t *testing.T) {
	lines := []string{
		"0, NVIDIA GeForce RTX 4090, 37, 2048, 24564, 51, 112.35",
		"1, Tesla T4, [N/A], 0, 15360, 40, [Not Supported]",
		"garbage",
	}
	const mib = 1024 * 1024
	want := []GPU{
		{Index: 0, Name: "NVIDIA GeForce RTX 4090", Util: 37, MemUsed: 2048 * mib, MemTotal: 24564 * mib, Temp: 51, Power: 112.35},
		{Index: 1, Name: "Tesla T4", Util: -1, MemUsed: 0, MemTotal: 15360 * mib, Temp: 40, Power: -1},
	}
	if got := parseGPUs(lines); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}
