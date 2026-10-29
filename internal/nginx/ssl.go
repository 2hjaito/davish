package nginx

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"time"

	"davish/internal/shellx"
)

// SSLPlan is what turning on HTTPS for a site with certbot takes.
type SSLPlan struct {
	// Names get a certificate; Skipped cannot (wildcards, IPs, "_").
	Names   []string `json:"names"`
	Skipped []string `json:"skipped"`
	// ServerIPs are the addresses the server's SSH host resolves to.
	ServerIPs []string   `json:"serverIPs"`
	DNS       []DNSCheck `json:"dns"`
	Redirect  bool       `json:"redirect"`
	// Sudo is the certbot command, for a terminal: certbot may ask for an
	// email and to accept its terms the first time.
	Sudo string `json:"sudo"`
}

// DNSCheck is where one domain points, as seen from this machine.
type DNSCheck struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
	// OK is set when one of IPs is the server's.
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Resolver looks up host names; net.DefaultResolver in the app.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// PlanSSL checks that names point at host, the server's SSH address, and
// gives the certbot command for them. DNS is looked up from this machine:
// nothing runs on the server.
func PlanSSL(ctx context.Context, r Resolver, host string, names []string, redirect bool) SSLPlan {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := SSLPlan{Names: []string{}, Skipped: []string{}, ServerIPs: []string{}, DNS: []DNSCheck{}, Redirect: redirect}
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		switch {
		case n == "" || slices.Contains(p.Names, n) || slices.Contains(p.Skipped, n):
		case n == "_" || strings.HasPrefix(n, "*") || net.ParseIP(n) != nil || !hostName.MatchString(n) || !strings.Contains(n, "."):
			p.Skipped = append(p.Skipped, n)
		default:
			p.Names = append(p.Names, n)
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		p.ServerIPs = []string{ip.String()}
	} else if ips, err := r.LookupHost(ctx, host); err == nil {
		p.ServerIPs = ips
	}
	for _, n := range p.Names {
		c := DNSCheck{Name: n, IPs: []string{}}
		ips, err := r.LookupHost(ctx, n)
		if err != nil {
			c.Error = "no DNS record"
		} else {
			c.IPs = ips
			c.OK = slices.ContainsFunc(ips, func(ip string) bool { return slices.Contains(p.ServerIPs, ip) })
		}
		p.DNS = append(p.DNS, c)
	}
	p.Sudo = CertbotCommand(p.Names, redirect)
	return p
}

// CertbotCommand is the certbot line for names, which PlanSSL has
// checked; empty without names.
func CertbotCommand(names []string, redirect bool) string {
	if len(names) == 0 {
		return ""
	}
	args := []string{"certbot", "--nginx"}
	for _, n := range names {
		args = append(args, "-d", n)
	}
	if redirect {
		args = append(args, "--redirect")
	} else {
		args = append(args, "--no-redirect")
	}
	return shellx.Cmd("sudo", args...).String()
}

// CertNames are the names of names a certificate can be issued for.
func CertNames(names []string) []string {
	return PlanSSL(context.Background(), noDNS{}, "127.0.0.1", names, false).Names
}

type noDNS struct{}

func (noDNS) LookupHost(context.Context, string) ([]string, error) { return nil, errNoDNS }

var errNoDNS = errors.New("no lookup")
