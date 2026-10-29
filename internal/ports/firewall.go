package ports

import (
	"strconv"
	"strings"
)

// Firewall states.
const (
	FirewallNone     = "none"     // ufw is not installed
	FirewallInactive = "inactive" // installed, turned off
	FirewallActive   = "active"
	FirewallUnknown  = "unknown" // installed, its rules need root
)

// Exposure of a port to the Internet.
const (
	ExposedYes     = "yes"
	ExposedNo      = "no"
	ExposedUnknown = "unknown"
)

// Firewall is what ufw lets in.
type Firewall struct {
	State string `json:"state"`
	// DefaultAllow is set when ufw lets in what no rule names.
	DefaultAllow bool `json:"defaultAllow"`
	// Open are the TCP ports a rule allows from anywhere.
	Open []portRange `json:"-"`
}

type portRange struct{ lo, hi int }

// profiles are ufw's common application profiles.
var profiles = map[string][]int{
	"OpenSSH": {22}, "Nginx Full": {80, 443}, "Nginx HTTP": {80}, "Nginx HTTPS": {443},
	"Apache Full": {80, 443}, "Apache": {80}, "Apache Secure": {443}, "WWW Full": {80, 443},
}

// parseUFW reads `ufw status verbose`; installed tells that ufw exists.
func parseUFW(out string, installed bool) Firewall {
	fw := Firewall{State: FirewallNone}
	if !installed {
		return fw
	}
	fw.State = FirewallUnknown
	for l := range strings.Lines(out) {
		l = strings.TrimSpace(l)
		switch {
		case l == "Status: active":
			fw.State = FirewallActive
		case l == "Status: inactive":
			fw.State = FirewallInactive
		case strings.HasPrefix(l, "Default:"):
			fw.DefaultAllow = strings.Contains(l, "allow (incoming)")
		}
		i := strings.Index(l, " ALLOW")
		if i < 0 || !strings.HasSuffix(strings.TrimSpace(strings.TrimSuffix(l, "(v6)")), "Anywhere") {
			continue
		}
		to := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l[:i]), "(v6)"))
		if ports, ok := profiles[to]; ok {
			for _, p := range ports {
				fw.Open = append(fw.Open, portRange{p, p})
			}
			continue
		}
		spec, proto, _ := strings.Cut(to, "/")
		if proto != "" && proto != "tcp" {
			continue
		}
		for part := range strings.SplitSeq(spec, ",") {
			lo, hi, isRange := strings.Cut(part, ":")
			a, err := strconv.Atoi(lo)
			if err != nil {
				continue
			}
			b := a
			if isRange {
				if b, err = strconv.Atoi(hi); err != nil {
					continue
				}
			}
			fw.Open = append(fw.Open, portRange{a, b})
		}
	}
	return fw
}

// exposure tells whether p is reachable from the Internet through fw.
// Docker publishes ports with its own iptables rules, before ufw's, so a
// container's public port is open whatever ufw says.
func exposure(p Port, fw Firewall) string {
	switch {
	case p.Loopback:
		return ExposedNo
	case p.Process == "docker-proxy":
		return ExposedYes
	case fw.State == FirewallNone || fw.State == FirewallInactive:
		return ExposedYes
	case fw.State == FirewallUnknown:
		return ExposedUnknown
	case fw.DefaultAllow:
		return ExposedYes
	}
	for _, r := range fw.Open {
		if p.Port >= r.lo && p.Port <= r.hi {
			return ExposedYes
		}
	}
	return ExposedNo
}
