package sysmon

import (
	"reflect"
	"testing"
)

func TestParseInfoFull(t *testing.T) {
	got := parseInfo(readSections(t, "info-full.txt"))
	if got.Threads != 16 || got.Cores != 8 || got.Sockets != 1 {
		t.Errorf("cpu = %d threads, %d cores, %d sockets", got.Threads, got.Cores, got.Sockets)
	}
	if got.MemTotal != 65748312*1024 || got.SwapTotal != 1003516*1024 {
		t.Errorf("mem = %d, swap = %d", got.MemTotal, got.SwapTotal)
	}
	if got.BootTime != 1794000000 || got.Timezone != "Europe/Paris" || got.Systemd != "252" {
		t.Errorf("boot %d, tz %q, systemd %q", got.BootTime, got.Timezone, got.Systemd)
	}
	if !reflect.DeepEqual(got.Users, []string{"dang", "root"}) {
		t.Errorf("users = %v", got.Users)
	}
	wantDisks := []Disk{
		{Name: "sda", Model: "WDC WD120EFBX-68B0EN0", Size: 12000138625024, Kind: "hdd", Parts: []Part{
			{Name: "sda1", Size: 12000136527872, FSType: "ext4", Mount: "/srv/media"},
		}},
		{Name: "nvme0n1", Model: "Samsung SSD 980 PRO 1TB", Size: 1000204886016, Kind: "nvme", Parts: []Part{
			{Name: "nvme0n1p1", Size: 536870912, FSType: "vfat", Mount: "/boot/efi"},
			{Name: "nvme0n1p2", Size: 999666294784, FSType: "LVM2_member"},
			{Name: "vg-root", Size: 999662100480, FSType: "ext4", Mount: "/"},
		}},
	}
	if !reflect.DeepEqual(got.Disks, wantDisks) {
		t.Errorf("disks = %+v", got.Disks)
	}
	wantIfaces := []Iface{
		{Name: "enp5s0", Up: true, MAC: "a8:a1:59:00:00:01", Addrs: []string{"192.168.1.20/24"}, Kind: "lan"},
		{Name: "tailscale0", Up: true, Addrs: []string{"100.64.0.3/32"}, Kind: "vpn"},
		{Name: "docker0", Up: true, MAC: "02:42:00:00:00:01", Addrs: []string{"172.17.0.1/16"}, Kind: "docker"},
	}
	if !reflect.DeepEqual(got.Interfaces, wantIfaces) {
		t.Errorf("interfaces = %+v", got.Interfaces)
	}
	if got.Gateway != "192.168.1.1" || got.GatewayDev != "enp5s0" {
		t.Errorf("gateway = %q on %q", got.Gateway, got.GatewayDev)
	}
	if !reflect.DeepEqual(got.DNS, []string{"192.168.1.1", "1.1.1.1"}) {
		t.Errorf("dns = %v", got.DNS)
	}
}

func TestParseLsblkOldAndVirtual(t *testing.T) {
	// util-linux before 2.33 printed every value as a string.
	old := []byte(`{"blockdevices": [
	  {"name": "vda", "type": "disk", "size": "42949672960", "model": null, "rota": "1", "tran": null, "mountpoint": null, "fstype": null,
	   "children": [{"name": "vda1", "type": "part", "size": "42948624384", "rota": "1", "mountpoint": "/", "fstype": "ext4"}]},
	  {"name": "sdb", "type": "disk", "size": "500107862016", "model": "Crucial MX500", "rota": "0", "tran": "sata", "mountpoint": "/data", "fstype": "xfs"},
	  {"name": "sr0", "type": "rom", "size": "1073741312", "model": "QEMU DVD-ROM", "rota": "1"}
	]}`)
	got := parseLsblk(old)
	want := []Disk{
		{Name: "vda", Size: 42949672960, Kind: "virtual", Parts: []Part{{Name: "vda1", Size: 42948624384, FSType: "ext4", Mount: "/"}}},
		{Name: "sdb", Model: "Crucial MX500", Size: 500107862016, Kind: "ssd", Parts: []Part{{Name: "sdb", Size: 500107862016, FSType: "xfs", Mount: "/data"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	if parseLsblk(nil) != nil || parseLsblk([]byte("lsblk: not found")) != nil {
		t.Error("garbage should give nothing")
	}
}

func TestParseInfoWithoutNewSections(t *testing.T) {
	// An old ip without -j, no lsblk: the new fields stay empty, nothing breaks.
	got := parseInfo(map[string][]string{"ip": {"Usage: ip [ OPTIONS ] OBJECT"}, "lsblk": nil})
	if got.Interfaces != nil || got.Disks != nil || got.Threads != 0 {
		t.Errorf("got %+v", got)
	}
}
