// Package ports lists the TCP ports a server listens on and forwards them
// to this machine over the existing SSH connection.
package ports

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "ports"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Lister reads listening ports and probes them from the server.
type Lister struct {
	run Runner
}

// NewLister creates the lister.
func NewLister(run Runner) *Lister {
	return &Lister{run: run}
}

// Port is one listening TCP socket.
type Port struct {
	Addr string `json:"addr"`
	Port int    `json:"port"`
	// Process and PID are empty when the socket belongs to another user
	// and Davish is not root.
	Process string `json:"process,omitempty"`
	PID     int    `json:"pid,omitempty"`
	// Container is set when docker-proxy publishes the port for it.
	Container string `json:"container,omitempty"`
	// Loopback is set when only the server itself can reach the port.
	Loopback bool `json:"loopback"`
}

// List is the ports and whether Davish could see every process.
type List struct {
	Ports []Port `json:"ports"`
	Root  bool   `json:"root"`
}

const sep = "--davish--"

// listLine reads the uid, then ss, then docker's published ports, which
// may be missing or refused.
var listLine = `id -u; command -v ss >/dev/null || echo NO_SS; ss -tlnp 2>/dev/null; echo ` + sep +
	`; docker ps --format '{{.Names}}\t{{.Ports}}' 2>/dev/null; true`

// List reads the server's listening TCP ports.
func (l *Lister) List(ctx context.Context, serverID string) (List, error) {
	res, err := l.run.RunLine(ctx, serverID, app, listLine)
	if err != nil {
		return List{}, err
	}
	return parse(res.Stdout)
}

func parse(out []byte) (List, error) {
	head, docker, _ := bytes.Cut(out, []byte(sep+"\n"))
	sc := bufio.NewScanner(bytes.NewReader(head))
	var list List
	if sc.Scan() {
		list.Root = strings.TrimSpace(sc.Text()) == "0"
	}
	containers := parseDocker(docker)
	for sc.Scan() {
		line := sc.Text()
		if line == "NO_SS" {
			return List{}, apperr.New(apperr.CmdMissing, "ss is not installed on the server")
		}
		if p, ok := parseSS(line); ok {
			if p.Process == "docker-proxy" {
				p.Container = containers[p.Port]
			}
			list.Ports = append(list.Ports, p)
		}
	}
	slices.SortStableFunc(list.Ports, func(a, b Port) int { return a.Port - b.Port })
	return list, nil
}

var procRe = regexp.MustCompile(`\(\("([^"]+)",pid=(\d+)`)

// parseSS reads one line of `ss -tlnp`:
// LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:(("nginx",pid=12,fd=6))
func parseSS(line string) (Port, bool) {
	f := strings.Fields(line)
	if len(f) < 5 || f[0] != "LISTEN" {
		return Port{}, false
	}
	local := f[3]
	i := strings.LastIndex(local, ":")
	if i < 0 {
		return Port{}, false
	}
	port, err := strconv.Atoi(local[i+1:])
	if err != nil {
		return Port{}, false
	}
	addr := strings.Trim(local[:i], "[]")
	if j := strings.Index(addr, "%"); j >= 0 { // 127.0.0.53%lo
		addr = addr[:j]
	}
	p := Port{Addr: addr, Port: port}
	if ip := net.ParseIP(addr); ip != nil {
		p.Loopback = ip.IsLoopback()
	}
	if m := procRe.FindStringSubmatch(line); m != nil {
		p.Process = m[1]
		p.PID, _ = strconv.Atoi(m[2])
	}
	return p, true
}

var publishRe = regexp.MustCompile(`:(\d+)->`)

// parseDocker maps host ports to the container publishing them, from
// lines like "web\t0.0.0.0:8080->80/tcp, :::8080->80/tcp".
func parseDocker(out []byte) map[int]string {
	m := map[int]string{}
	for line := range strings.SplitSeq(string(out), "\n") {
		name, ports, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		for _, g := range publishRe.FindAllStringSubmatch(ports, -1) {
			if n, err := strconv.Atoi(g[1]); err == nil {
				m[n] = name
			}
		}
	}
	return m
}

// maxProbe is how much of a response "try from the server" shows.
const maxProbe = 16 << 10

// ProbeLine is the curl that "try from the server" runs.
func ProbeLine(host string, port int, hostHeader string) (string, error) {
	host = dialHost(host)
	if err := checkHost(host); err != nil {
		return "", err
	}
	if port < 1 || port > 65535 {
		return "", apperr.New(apperr.InvalidInput, "port must be 1-65535")
	}
	args := []string{"curl", "-sS", "-i", "--max-time", "10"}
	if hostHeader != "" {
		if err := checkHostHeader(hostHeader); err != nil {
			return "", err
		}
		args = append(args, "-H", "Host: "+hostHeader)
	}
	args = append(args, "http://"+net.JoinHostPort(host, strconv.Itoa(port))+"/")
	return shellx.Cmd(args[0], args[1:]...).String() + " 2>&1 | head -c " + strconv.Itoa(maxProbe), nil
}

// Probe fetches http://host:port/ from the server itself and returns the
// status line, headers and the start of the body as text.
func (l *Lister) Probe(ctx context.Context, serverID, host string, port int, hostHeader string) (string, error) {
	line, err := ProbeLine(host, port, hostHeader)
	if err != nil {
		return "", err
	}
	res, err := l.run.RunLine(ctx, serverID, app, "command -v curl >/dev/null || { echo NO_CURL; exit 0; }; "+line)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(res.Stdout)) == "NO_CURL" {
		return "", apperr.New(apperr.CmdMissing, "curl is not installed on the server")
	}
	return strings.ToValidUTF8(string(res.Stdout), "�"), nil
}

// checkHost accepts what a listening socket reports: an IP literal, a
// wildcard, or localhost.
func checkHost(h string) error {
	if h == "localhost" || net.ParseIP(h) != nil {
		return nil
	}
	return apperr.New(apperr.InvalidInput, "host must be an IP address or localhost")
}

var hostHeaderRe = regexp.MustCompile(`^[A-Za-z0-9.-]{1,253}(:\d{1,5})?$`)

func checkHostHeader(h string) error {
	if !hostHeaderRe.MatchString(h) {
		return apperr.New(apperr.InvalidInput, "Host header must be a domain name")
	}
	return nil
}

// dialHost is where the server reaches a socket listening on addr: a
// wildcard address is reached on loopback.
func dialHost(addr string) string {
	switch addr {
	case "", "*", "0.0.0.0", "::":
		return "127.0.0.1"
	}
	return addr
}
