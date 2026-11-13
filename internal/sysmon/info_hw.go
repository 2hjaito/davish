package sysmon

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Disk is a physical or virtual block device, with what is on it.
type Disk struct {
	Name  string `json:"name"` // nvme0n1, sda, vda
	Model string `json:"model,omitempty"`
	Size  int64  `json:"size"` // bytes
	// Kind is nvme, ssd, hdd or virtual; empty when the kernel does not say.
	Kind  string `json:"kind,omitempty"`
	Parts []Part `json:"parts,omitempty"`
}

// Part is a partition, LVM volume or the disk itself when it holds a filesystem.
type Part struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	FSType string `json:"fstype,omitempty"`
	Mount  string `json:"mount,omitempty"`
}

// Iface is a network interface with its global addresses.
type Iface struct {
	Name  string   `json:"name"`
	Up    bool     `json:"up"`
	MAC   string   `json:"mac,omitempty"`
	Addrs []string `json:"addrs,omitempty"` // CIDR
	// Kind is lan, vpn or docker.
	Kind string `json:"kind"`
}

func joined(lines []string) []byte { return []byte(strings.Join(lines, "\n")) }

func parseHardware(info *SystemInfo, s map[string][]string) {
	if c := s["cpucount"]; len(c) >= 3 {
		info.Threads = atoi(c[0])
		sockets := atoi(c[1])
		perSocket := atoi(c[2])
		if sockets == 0 && perSocket > 0 {
			sockets = 1 // one socket that does not say so
		}
		info.Sockets, info.Cores = sockets, sockets*perSocket
	}
	for _, line := range s["mem"] {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		kb, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
		switch k {
		case "MemTotal":
			info.MemTotal = kb * 1024
		case "SwapTotal":
			info.SwapTotal = kb * 1024
		}
	}
	if b := strings.Fields(firstLine(s["boot"])); len(b) == 2 {
		info.BootTime, _ = strconv.ParseInt(b[1], 10, 64)
	}
	info.Timezone = strings.TrimPrefix(firstLine(s["tz"]), "/usr/share/zoneinfo/")
	info.Timezone = strings.TrimPrefix(info.Timezone, "../usr/share/zoneinfo/")
	for _, u := range s["users"] {
		if u = strings.TrimSpace(u); u != "" {
			info.Users = append(info.Users, u)
		}
	}
	// "systemd 255 (255.4-1ubuntu8.4)" → "255".
	if f := strings.Fields(firstLine(s["systemd"])); len(f) >= 2 && f[0] == "systemd" {
		info.Systemd = f[1]
	}
	info.Disks = parseLsblk(joined(s["lsblk"]))
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// lsblk's JSON changed over the years: SIZE and ROTA are numbers and booleans
// in new versions, strings in old ones.
type blockDev struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Size       json.RawMessage `json:"size"`
	Model      *string         `json:"model"`
	Rota       json.RawMessage `json:"rota"`
	Tran       *string         `json:"tran"`
	Mountpoint *string         `json:"mountpoint"`
	FSType     *string         `json:"fstype"`
	Children   []blockDev      `json:"children"`
}

func rawInt(r json.RawMessage) int64 {
	s := strings.Trim(string(r), `"`)
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func rawBool(r json.RawMessage) bool {
	s := strings.Trim(string(r), `"`)
	return s == "true" || s == "1"
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

var virtualDisk = regexp.MustCompile(`^(vd|xvd)[a-z]+$`)
var virtualModel = regexp.MustCompile(`(?i)qemu|virtual|vbox|vmware`)

func parseLsblk(b []byte) []Disk {
	var doc struct {
		Devices []blockDev `json:"blockdevices"`
	}
	if len(b) == 0 || json.Unmarshal(b, &doc) != nil {
		return nil
	}
	var out []Disk
	for _, d := range doc.Devices {
		if d.Type != "disk" {
			continue // loop (snaps), rom, zram
		}
		disk := Disk{Name: d.Name, Model: str(d.Model), Size: rawInt(d.Size)}
		switch tran := str(d.Tran); {
		case tran == "nvme" || strings.HasPrefix(d.Name, "nvme"):
			disk.Kind = "nvme"
		case virtualDisk.MatchString(d.Name) || virtualModel.MatchString(disk.Model):
			disk.Kind = "virtual"
		case len(d.Rota) > 0 && rawBool(d.Rota):
			disk.Kind = "hdd"
		case len(d.Rota) > 0:
			disk.Kind = "ssd"
		}
		var walk func(bd blockDev)
		walk = func(bd blockDev) {
			if str(bd.FSType) != "" || str(bd.Mountpoint) != "" {
				disk.Parts = append(disk.Parts, Part{
					Name: bd.Name, Size: rawInt(bd.Size), FSType: str(bd.FSType), Mount: str(bd.Mountpoint),
				})
			}
			for _, c := range bd.Children {
				walk(c)
			}
		}
		walk(d)
		out = append(out, disk)
	}
	return out
}

func parseNetwork(info *SystemInfo, s map[string][]string) {
	var links []struct {
		Name     string `json:"ifname"`
		State    string `json:"operstate"`
		Flags    []string
		MAC      string `json:"address"`
		LinkType string `json:"link_type"`
		Addrs    []struct {
			Family string `json:"family"`
			Local  string `json:"local"`
			Prefix int    `json:"prefixlen"`
			Scope  string `json:"scope"`
		} `json:"addr_info"`
	}
	if b := joined(s["ip"]); len(b) > 0 && json.Unmarshal(b, &links) == nil {
		for _, l := range links {
			kind := ifaceKind(l.Name)
			if kind == "" {
				continue
			}
			ifc := Iface{Name: l.Name, Up: l.State == "UP" || l.State == "UNKNOWN", Kind: kind}
			if l.LinkType == "ether" {
				ifc.MAC = l.MAC
			}
			for _, a := range l.Addrs {
				if a.Scope == "global" && a.Local != "" {
					ifc.Addrs = append(ifc.Addrs, a.Local+"/"+strconv.Itoa(a.Prefix))
				}
			}
			// A down interface without an address says nothing useful.
			if !ifc.Up && len(ifc.Addrs) == 0 {
				continue
			}
			info.Interfaces = append(info.Interfaces, ifc)
		}
		order := map[string]int{"lan": 0, "vpn": 1, "docker": 2}
		sort.SliceStable(info.Interfaces, func(i, j int) bool {
			return order[info.Interfaces[i].Kind] < order[info.Interfaces[j].Kind]
		})
	}
	var routes []struct {
		Gateway string `json:"gateway"`
		Dev     string `json:"dev"`
	}
	if b := joined(s["route"]); len(b) > 0 && json.Unmarshal(b, &routes) == nil && len(routes) > 0 {
		info.Gateway, info.GatewayDev = routes[0].Gateway, routes[0].Dev
	}
	for _, d := range s["dns"] {
		if d = strings.TrimSpace(d); d != "" {
			info.DNS = append(info.DNS, d)
		}
	}
}

// ifaceKind sorts an interface; "" leaves it out (loopback, a container's veth).
func ifaceKind(name string) string {
	switch {
	case name == "lo" || strings.HasPrefix(name, "veth"):
		return ""
	case name == "docker0" || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "cni") ||
		strings.HasPrefix(name, "flannel") || strings.HasPrefix(name, "virbr"):
		return "docker"
	case strings.HasPrefix(name, "tailscale") || strings.HasPrefix(name, "wg") ||
		strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "zt") || strings.HasPrefix(name, "nebula"):
		return "vpn"
	}
	return "lan"
}
