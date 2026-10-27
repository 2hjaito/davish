package sysmon

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// testdata/ubuntu.txt is two snapshots of collect.sh on the Ubuntu test sshd.
func TestReaderParsesRealOutput(t *testing.T) {
	f, err := os.Open("testdata/ubuntu.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r := NewReader(f)

	want := []Snapshot{
		{
			CPUTotal: 578224, CPUIdle: 576807,
			MemTotal: 8125796 * 1024, MemAvailable: 7483280 * 1024,
			Load1: 0.06, Uptime: 385.85,
			RxBytes: 62826, TxBytes: 1677222,
			DiskSize: 954925484 * 1024, DiskUsed: 28681832 * 1024, DiskAvail: 877662380 * 1024,
		},
		{
			CPUTotal: 581236, CPUIdle: 579816,
			MemTotal: 8125796 * 1024, MemAvailable: 7496792 * 1024,
			Load1: 0.06, Uptime: 387.86,
			RxBytes: 62826, TxBytes: 1677222,
			DiskSize: 954925484 * 1024, DiskUsed: 28681832 * 1024, DiskAvail: 877662380 * 1024,
		},
	}
	for i, w := range want {
		got, err := r.Next()
		if err != nil {
			t.Fatalf("snapshot %d: %v", i, err)
		}
		if got != w {
			t.Errorf("snapshot %d:\n got %+v\nwant %+v", i, got, w)
		}
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
	if snap.Load1 != 0.5 || snap.MemAvailable != 400*1024 {
		t.Errorf("snapshot = %+v", snap)
	}
}

func TestParseMeminfoWithoutMemAvailable(t *testing.T) {
	// Kernels before 3.14 (CentOS 6 and friends) have no MemAvailable.
	lines := []string{"MemTotal: 2000 kB", "MemFree: 500 kB", "Buffers: 100 kB", "Cached: 300 kB"}
	total, avail, err := parseMeminfo(lines)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2000*1024 || avail != 900*1024 {
		t.Errorf("total, avail = %d, %d; want %d, %d", total, avail, 2000*1024, 900*1024)
	}
}

func TestParseNetDev(t *testing.T) {
	header := []string{
		"Inter-|   Receive                                                |  Transmit",
		" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed",
	}
	tests := []struct {
		name   string
		lines  []string
		rx, tx uint64
	}{
		{
			name: "sums physical interfaces",
			lines: []string{
				"  eth0: 100 1 0 0 0 0 0 0 200 1 0 0 0 0 0 0",
				"  eth1: 10 1 0 0 0 0 0 0 20 1 0 0 0 0 0 0",
			},
			rx: 110, tx: 220,
		},
		{
			name: "name touching a large counter",
			lines: []string{
				"enp0s3:4294967296 9 0 0 0 0 0 0 7 1 0 0 0 0 0 0",
			},
			rx: 4294967296, tx: 7,
		},
		{
			name: "skips loopback and container links",
			lines: []string{
				"    lo: 999 1 0 0 0 0 0 0 999 1 0 0 0 0 0 0",
				"docker0: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				"veth1a2b: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				"br-3f9c: 50 1 0 0 0 0 0 0 50 1 0 0 0 0 0 0",
				"  eth0: 5 1 0 0 0 0 0 0 6 1 0 0 0 0 0 0",
			},
			rx: 5, tx: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rx, tx, err := parseNetDev(append(header, tt.lines...))
			if err != nil {
				t.Fatal(err)
			}
			if rx != tt.rx || tx != tt.tx {
				t.Errorf("rx, tx = %d, %d; want %d, %d", rx, tx, tt.rx, tt.tx)
			}
		})
	}
}

func TestParseDf(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		used  uint64
	}{
		{
			name:  "plain device",
			lines: []string{"Filesystem 1024-blocks Used Available Capacity Mounted on", "/dev/vda1 1000 250 750 25% /"},
			used:  250 * 1024,
		},
		{
			name:  "device name with a space",
			lines: []string{"Filesystem 1024-blocks Used Available Capacity Mounted on", "my disk 1000 300 700 30% /"},
			used:  300 * 1024,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, used, _, err := parseDf(tt.lines)
			if err != nil {
				t.Fatal(err)
			}
			if used != tt.used {
				t.Errorf("used = %d; want %d", used, tt.used)
			}
		})
	}
	if _, _, _, err := parseDf([]string{"Filesystem 1024-blocks Used Available Capacity Mounted on"}); err == nil {
		t.Error("df without a data line parsed")
	}
}
