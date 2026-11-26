package platform

import "strings"

// DistroCommand prints the ID and ID_LIKE lines of os-release, which every
// mainstream Linux has. It reads a file and changes nothing.
const DistroCommand = `grep -E '^(ID|ID_LIKE)=' /etc/os-release 2>/dev/null`

// Logo keys for servers whose system is not a Linux distribution Davish
// has a logo for.
const (
	DistroLinux   = "linux"
	DistroWindows = "windows"
	DistroMacOS   = "macos"
)

// logos maps an os-release ID to the logo it gets. A key that is not here
// takes the next step in Distro.
var logos = map[string]string{
	"ubuntu":              "ubuntu",
	"debian":              "debian",
	"fedora":              "fedora",
	"centos":              "centos",
	"rocky":               "rocky",
	"almalinux":           "almalinux",
	"rhel":                "rhel",
	"arch":                "arch",
	"alpine":              "alpine",
	"opensuse":            "opensuse",
	"opensuse-leap":       "opensuse",
	"opensuse-tumbleweed": "opensuse",
	"sles":                "opensuse",
	"raspbian":            "raspberrypi",
	"kali":                "kali",
	"linuxmint":           "linuxmint",
}

// families are the ID_LIKE values worth a logo for a derivative: Pop!_OS,
// Zorin or elementary look like Ubuntu, Manjaro like Arch. Other families
// stay plain Linux, since Amazon Linux or Oracle Linux are not Fedora.
var families = map[string]bool{"ubuntu": true, "debian": true, "arch": true}

// Distro gives the logo key for the output of DistroCommand: the ID when
// it has a logo, else the first ID_LIKE family that has one, else "linux".
func Distro(out string) string {
	var id string
	var like []string
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		v = strings.ToLower(strings.Trim(strings.TrimSpace(v), `"'`))
		switch k {
		case "ID":
			id = v
		case "ID_LIKE":
			like = strings.Fields(v)
		}
	}
	if key, ok := logos[id]; ok {
		return key
	}
	for _, l := range like {
		if families[l] {
			return logos[l]
		}
	}
	return DistroLinux
}
