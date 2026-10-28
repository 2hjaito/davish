// Package packages reads what the server's package manager has installed
// and what it could upgrade, and plans the commands that change them.
// Changing packages needs root, so those commands are run by the user in
// a terminal (docs/SECURITY.md); this package only reads.
package packages

import (
	"bufio"
	"bytes"
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// app is the name commands are recorded under in the audit log.
const app = "packages"

// Package managers Davish knows.
const (
	Apt = "apt"
	Dnf = "dnf"
	Yum = "yum"
	Apk = "apk"
)

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Service reads packages on each server.
type Service struct {
	run  Runner
	emit Emitter

	// mu guards notified: the security fix count last notified per
	// server, so a check on every connection does not repeat an alert.
	mu       sync.Mutex
	notified map[string]int
}

// New creates the service.
func New(run Runner, emit Emitter) *Service {
	return &Service{run: run, emit: emit, notified: map[string]int{}}
}

// Package is one installed package.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Summary string `json:"summary"`
	// Size is the installed size in bytes; 0 when unknown.
	Size int64 `json:"size"`
}

// Update is a newer version the package manager knows of.
type Update struct {
	Name      string `json:"name"`
	Current   string `json:"current"`
	Available string `json:"available"`
	// Security is set for a fix from a security repository or advisory.
	Security bool `json:"security"`
}

// Inventory is what a server has and could upgrade.
type Inventory struct {
	Manager   string    `json:"manager"`
	Installed []Package `json:"installed"`
	Updates   []Update  `json:"updates"`
	// ListsAge is how many seconds ago apt last fetched its package lists;
	// -1 when it never did, so Updates cannot be trusted. Always 0 for
	// other managers, which read their cache with every check.
	ListsAge int64 `json:"listsAge"`
}

// detectLine prints the first package manager found.
const detectLine = `for m in apt-get dnf yum apk; do command -v "$m" >/dev/null 2>&1 && { echo "$m"; break; }; done`

// Detect tells which package manager the server uses; "" when none known.
func (s *Service) Detect(ctx context.Context, serverID string) (string, error) {
	res, err := s.run.RunLine(ctx, serverID, app, detectLine)
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(string(res.Stdout)) {
	case "apt-get":
		return Apt, nil
	case "dnf":
		return Dnf, nil
	case "yum":
		return Yum, nil
	case "apk":
		return Apk, nil
	}
	return "", nil
}

// Commands that read, by manager. None needs root; dnf and yum use only
// their cache (-C) so a check never downloads metadata as the user.
var (
	installedLine = map[string]string{
		Apt: `dpkg-query -W -f='${db:Status-Abbrev}\t${Package}\t${Version}\t${Installed-Size}\t${binary:Summary}\n'`,
		Dnf: `rpm -qa --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\t%{SUMMARY}\n'`,
		Yum: `rpm -qa --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\t%{SUMMARY}\n'`,
		Apk: `apk info -v 2>/dev/null`,
	}
	updatesLine = map[string]string{
		Apt: `apt list --upgradable 2>/dev/null`,
		Dnf: `dnf -q -C check-update 2>/dev/null; true`,
		Yum: `yum -q -C check-update 2>/dev/null; true`,
		Apk: `apk version -l '<' 2>/dev/null`,
	}
	securityLine = map[string]string{
		Dnf: `dnf -q -C updateinfo list --security 2>/dev/null; true`,
		Yum: `yum -q -C updateinfo list security 2>/dev/null; true`,
	}
)

// listsAgeLine prints now and the time apt last changed its lists, which
// is empty when it never fetched them.
const listsAgeLine = `date +%s; ls -A /var/lib/apt/lists 2>/dev/null | grep -qv '^partial$\|^lock$' && stat -c %Y /var/lib/apt/lists`

// Inventory reads the installed packages and the available updates.
func (s *Service) Inventory(ctx context.Context, serverID string) (Inventory, error) {
	m, err := s.Detect(ctx, serverID)
	if err != nil {
		return Inventory{}, err
	}
	if m == "" {
		return Inventory{}, apperr.New(apperr.CmdMissing, "no known package manager on the server")
	}
	inv := Inventory{Manager: m, Installed: []Package{}, Updates: []Update{}}
	out, err := s.read(ctx, serverID, installedLine[m])
	if err != nil {
		return Inventory{}, err
	}
	inv.Installed = parseInstalled(m, out)

	out, err = s.read(ctx, serverID, updatesLine[m])
	if err != nil {
		return Inventory{}, err
	}
	current := map[string]string{}
	for _, p := range inv.Installed {
		current[p.Name] = p.Version
	}
	inv.Updates = parseUpdates(m, out, current)

	if line, ok := securityLine[m]; ok {
		out, err := s.read(ctx, serverID, line)
		if err != nil {
			return Inventory{}, err
		}
		markSecurity(inv.Updates, parseAdvisories(out))
	}
	if m == Apt {
		out, err := s.read(ctx, serverID, listsAgeLine)
		if err != nil {
			return Inventory{}, err
		}
		inv.ListsAge = parseListsAge(out)
	}
	return inv, nil
}

func (s *Service) read(ctx context.Context, serverID, line string) ([]byte, error) {
	res, err := s.run.RunLine(ctx, serverID, app, line)
	if err != nil {
		return nil, err
	}
	if res.Exit == 127 {
		return nil, apperr.New(apperr.CmdMissing, "program not installed on the server")
	}
	return res.Stdout, nil
}

func lines(out []byte) []string {
	var ls []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		if l := strings.TrimRight(sc.Text(), "\r"); strings.TrimSpace(l) != "" {
			ls = append(ls, l)
		}
	}
	return ls
}

func parseInstalled(m string, out []byte) []Package {
	pkgs := []Package{}
	for _, l := range lines(out) {
		var p Package
		switch m {
		case Apt:
			f := strings.SplitN(l, "\t", 5)
			// Only fully installed ones: "ii " (not removed with config left).
			if len(f) < 5 || !strings.HasPrefix(f[0], "ii") {
				continue
			}
			kb, _ := strconv.ParseInt(strings.TrimSpace(f[3]), 10, 64)
			p = Package{Name: f[1], Version: f[2], Size: kb * 1024, Summary: f[4]}
		case Dnf, Yum:
			f := strings.SplitN(l, "\t", 4)
			if len(f) < 4 || f[0] == "gpg-pubkey" {
				continue
			}
			b, _ := strconv.ParseInt(f[2], 10, 64)
			p = Package{Name: f[0], Version: f[1], Size: b, Summary: f[3]}
		case Apk:
			name, ver, ok := splitApk(strings.TrimSpace(l))
			if !ok {
				continue
			}
			p = Package{Name: name, Version: ver}
		}
		pkgs = append(pkgs, p)
	}
	slices.SortFunc(pkgs, func(a, b Package) int { return strings.Compare(a.Name, b.Name) })
	return pkgs
}

// splitApk splits "name-1.2.3-r0" into name and version: the version is
// the last two dash-separated parts.
func splitApk(s string) (string, string, bool) {
	i := strings.LastIndex(s, "-")
	if i <= 0 {
		return "", "", false
	}
	j := strings.LastIndex(s[:i], "-")
	if j <= 0 {
		return "", "", false
	}
	return s[:j], s[j+1:], true
}

// aptUpgradable matches "name/suite,suite version arch [upgradable from: old]".
var aptUpgradable = regexp.MustCompile(`^([^/\s]+)/(\S+)\s+(\S+)\s+\S+\s+\[upgradable from: ([^\]]+)\]`)

func parseUpdates(m string, out []byte, current map[string]string) []Update {
	ups := []Update{}
	for _, l := range lines(out) {
		switch m {
		case Apt:
			mm := aptUpgradable.FindStringSubmatch(l)
			if mm == nil {
				continue
			}
			ups = append(ups, Update{Name: mm[1], Available: mm[3], Current: mm[4], Security: strings.Contains(mm[2], "security")})
		case Dnf, Yum:
			// "name.arch  version  repo"; headers and "Obsoleting" blocks
			// have other shapes.
			f := strings.Fields(l)
			if len(f) != 3 || !strings.Contains(f[0], ".") || strings.HasPrefix(l, " ") {
				continue
			}
			name := f[0][:strings.LastIndex(f[0], ".")]
			ups = append(ups, Update{Name: name, Available: f[1], Current: current[name]})
		case Apk:
			// "name-1.0-r0 < 1.1-r0"
			f := strings.Fields(l)
			if len(f) != 3 || f[1] != "<" {
				continue
			}
			name, ver, ok := splitApk(f[0])
			if !ok {
				continue
			}
			ups = append(ups, Update{Name: name, Current: ver, Available: f[2]})
		}
	}
	slices.SortFunc(ups, func(a, b Update) int { return strings.Compare(a.Name, b.Name) })
	return slices.CompactFunc(ups, func(a, b Update) bool { return a.Name == b.Name })
}

// parseAdvisories reads "ID  Severity/Sec.  name-version-release.arch"
// lines into the package NEVRAs they fix.
func parseAdvisories(out []byte) []string {
	var nevras []string
	for _, l := range lines(out) {
		if f := strings.Fields(l); len(f) == 3 {
			nevras = append(nevras, f[2])
		}
	}
	return nevras
}

// markSecurity flags the updates an advisory names. An advisory gives
// name-version-release.arch, so a package matches when it is followed by
// a dash and a digit: "openssl" matches "openssl-3.0.7-1.el9.x86_64" but
// not "openssl-libs-…".
func markSecurity(ups []Update, nevras []string) {
	for i := range ups {
		prefix := ups[i].Name + "-"
		for _, n := range nevras {
			if rest, ok := strings.CutPrefix(n, prefix); ok && rest != "" && rest[0] >= '0' && rest[0] <= '9' {
				ups[i].Security = true
				break
			}
		}
	}
}

func parseListsAge(out []byte) int64 {
	ls := lines(out)
	if len(ls) < 2 {
		return -1
	}
	now, err1 := strconv.ParseInt(strings.TrimSpace(ls[0]), 10, 64)
	at, err2 := strconv.ParseInt(strings.TrimSpace(ls[1]), 10, 64)
	if err1 != nil || err2 != nil {
		return -1
	}
	return max(0, now-at)
}
