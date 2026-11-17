// Package ansibleinv reads an Ansible inventory in INI format (never writes
// it) and turns its hosts into servers to import. Only the connection
// variables are used; groups only help the user choose and are not saved.
// Passwords in the inventory are never read: a host that has one is
// imported without it.
package ansibleinv

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Host is one inventory host that can be imported.
type Host struct {
	// Name is the inventory hostname, used as the server's display name.
	Name string `json:"name"`
	// Groups holds every group the host is in, through children too.
	Groups  []string `json:"groups"`
	Address string   `json:"address"`
	// User is "" when the inventory sets none; the caller picks the default.
	User      string  `json:"user"`
	Port      int     `json:"port"`
	KeyFile   string  `json:"keyFile,omitempty"`
	ProxyJump string  `json:"proxyJump,omitempty"`
	Notes     []Issue `json:"notes,omitempty"`
}

// Issue says why a host was skipped, or what of it was left out.
type Issue struct {
	// Code is one of the Skip* or Note* constants.
	Code string `json:"code"`
	// Detail names the variable or value concerned; never a secret.
	Detail string `json:"detail,omitempty"`
}

// Reasons a host is skipped.
const (
	SkipConnection = "connection" // ansible_connection is not ssh (Detail: its value)
	SkipJinja      = "jinja"      // a connection variable is a template (Detail: the variable)
	SkipVault      = "vault"      // a connection variable is encrypted (Detail: the variable)
	SkipPort       = "port"       // ansible_port is not a port (Detail: its value)
	SkipRange      = "range"      // a host range is too large or malformed (Detail: the pattern)
)

// What an imported host leaves out.
const (
	NotePassword = "password" // a password variable is set; it is not imported
	NoteSSHArgs  = "sshArgs"  // ssh options other than ProxyJump are ignored (Detail: their names)
)

// Skipped is a host that cannot be imported.
type Skipped struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
	Issue  Issue    `json:"issue"`
}

// Inventory is what an inventory file yields.
type Inventory struct {
	Hosts   []Host    `json:"hosts"`
	Skipped []Skipped `json:"skipped"`
	// Groups lists the groups that hold at least one host, in file order.
	Groups []string `json:"groups"`
}

// Limits on what one file may expand to.
const (
	maxRange = 256
	maxHosts = 10000
)

// Variable names, newest first; the old ansible_ssh_* forms still work.
var (
	varAddress  = []string{"ansible_host", "ansible_ssh_host"}
	varUser     = []string{"ansible_user", "ansible_ssh_user"}
	varPort     = []string{"ansible_port", "ansible_ssh_port"}
	varKey      = []string{"ansible_ssh_private_key_file", "ansible_private_key_file"}
	varConn     = []string{"ansible_connection"}
	varSSHArgs  = []string{"ansible_ssh_common_args", "ansible_ssh_extra_args"}
	varPassword = []string{
		"ansible_password", "ansible_ssh_pass", "ansible_ssh_password",
		"ansible_become_pass", "ansible_become_password", "ansible_sudo_pass",
	}
)

// redacted stands in for the value of a password variable, which is
// dropped as soon as it is read.
const redacted = "\x00"

type group struct {
	name     string
	hosts    []string
	vars     map[string]string
	children []string
	order    int
}

type parser struct {
	groups    map[string]*group
	hostVars  map[string]map[string]string
	hostOrder []string
	skipped   map[string]Issue // hosts whose line could not be expanded
}

func (p *parser) group(name string) *group {
	g, ok := p.groups[name]
	if !ok {
		g = &group{name: name, vars: map[string]string{}, order: len(p.groups)}
		p.groups[name] = g
	}
	return g
}

// Parse reads an INI inventory. dir is the inventory's folder, to resolve
// a relative key path; "" leaves it as written.
func Parse(r io.Reader, dir string) (Inventory, error) {
	p := &parser{groups: map[string]*group{}, hostVars: map[string]map[string]string{}, skipped: map[string]Issue{}}
	p.group("all")
	p.group("ungrouped")

	kind, cur := "hosts", "ungrouped"
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			kind, cur = "hosts", name
			if i := strings.LastIndex(name, ":"); i >= 0 {
				switch name[i+1:] {
				case "vars", "children":
					kind, cur = name[i+1:], name[:i]
				}
			}
			if cur == "" {
				return Inventory{}, fmt.Errorf("line %d: empty group name", n)
			}
			p.group(cur)
			continue
		}
		var err error
		switch kind {
		case "vars":
			err = p.groupVar(cur, line)
		case "children":
			child := strings.Fields(line)[0]
			g := p.group(cur)
			if !slices.Contains(g.children, child) {
				g.children = append(g.children, child)
			}
			p.group(child)
		default:
			err = p.hostLine(cur, line)
		}
		if err != nil {
			return Inventory{}, fmt.Errorf("line %d: %w", n, err)
		}
		if len(p.hostOrder) > maxHosts {
			return Inventory{}, fmt.Errorf("more than %d hosts", maxHosts)
		}
	}
	if err := sc.Err(); err != nil {
		return Inventory{}, fmt.Errorf("read inventory: %w", err)
	}
	return p.resolve(dir), nil
}

func (p *parser) groupVar(name, line string) error {
	key, val, ok := strings.Cut(line, "=")
	if !ok {
		// "key: value" is YAML style, which INI group vars do not accept.
		return fmt.Errorf("expected key=value in [%s:vars]", name)
	}
	key = strings.TrimSpace(key)
	toks, err := split(strings.TrimSpace(val))
	if err != nil {
		return err
	}
	p.group(name).vars[key] = keep(key, strings.Join(toks, " "))
	return nil
}

func (p *parser) hostLine(groupName, line string) error {
	toks, err := split(line)
	if err != nil {
		return err
	}
	if len(toks) == 0 {
		return nil
	}
	vars := map[string]string{}
	for _, tok := range toks[1:] {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			// The word itself is not quoted back: it may be a password.
			return fmt.Errorf("expected key=value after host %q", toks[0])
		}
		vars[k] = keep(k, v)
	}
	pattern, port := splitPort(toks[0])
	if port != "" {
		if _, set := vars["ansible_port"]; !set {
			vars["ansible_port"] = port
		}
	}
	names, err := expand(pattern)
	g := p.group(groupName)
	if err != nil {
		// Shown as one skipped entry, named by its pattern.
		names = []string{pattern}
		p.skipped[pattern] = Issue{Code: SkipRange, Detail: pattern}
	}
	for _, name := range names {
		hv, seen := p.hostVars[name]
		if !seen {
			hv = map[string]string{}
			p.hostVars[name] = hv
			p.hostOrder = append(p.hostOrder, name)
		}
		for k, v := range vars {
			hv[k] = v
		}
		if !slices.Contains(g.hosts, name) {
			g.hosts = append(g.hosts, name)
		}
	}
	return nil
}

// keep returns the value to store for key: a password is dropped at once.
func keep(key, val string) string {
	if slices.Contains(varPassword, key) {
		return redacted
	}
	return val
}

// splitPort splits "host:2222" (after any range); a range's own colons
// stay in the pattern.
func splitPort(tok string) (string, string) {
	i := strings.LastIndex(tok, ":")
	if i < 0 || i < strings.LastIndex(tok, "]") || strings.Count(tok, ":") > 1 && !strings.Contains(tok, "[") {
		return tok, "" // no port, or a bare IPv6 address
	}
	if _, err := strconv.Atoi(tok[i+1:]); err != nil {
		return tok, ""
	}
	return tok[:i], tok[i+1:]
}

// resolve applies the variables to each host and sorts out which can be
// imported.
func (p *parser) resolve(dir string) Inventory {
	parents := map[string][]string{}
	for _, g := range p.groups {
		for _, c := range g.children {
			parents[c] = append(parents[c], g.name)
		}
	}
	depth := depths(p.groups)

	inv := Inventory{Hosts: []Host{}, Skipped: []Skipped{}, Groups: []string{}}
	used := map[string]bool{}
	for _, name := range p.hostOrder {
		groups := p.groupsOf(name, parents)
		for _, g := range groups {
			used[g] = true
		}
		if is, bad := p.skipped[name]; bad {
			inv.Skipped = append(inv.Skipped, Skipped{Name: name, Groups: groups, Issue: is})
			continue
		}
		vars := p.varsOf(name, groups, depth)
		h, issue := toHost(name, vars, dir)
		if issue != nil {
			inv.Skipped = append(inv.Skipped, Skipped{Name: name, Groups: groups, Issue: *issue})
			continue
		}
		h.Groups = groups
		inv.Hosts = append(inv.Hosts, h)
	}
	ordered := make([]*group, 0, len(p.groups))
	for _, g := range p.groups {
		if used[g.name] {
			ordered = append(ordered, g)
		}
	}
	slices.SortFunc(ordered, func(a, b *group) int { return a.order - b.order })
	for _, g := range ordered {
		inv.Groups = append(inv.Groups, g.name)
	}
	return inv
}

// groupsOf lists the groups a host is in, directly or as a child's
// member, in file order; "all" is left out, and "ungrouped" only applies
// to a host in no other group.
func (p *parser) groupsOf(host string, parents map[string][]string) []string {
	in := map[string]bool{}
	var walk func(string)
	walk = func(g string) {
		if in[g] {
			return
		}
		in[g] = true
		for _, up := range parents[g] {
			walk(up)
		}
	}
	for _, g := range p.groups {
		if g.name != "all" && g.name != "ungrouped" && slices.Contains(g.hosts, host) {
			walk(g.name)
		}
	}
	delete(in, "all")
	if len(in) == 0 {
		in["ungrouped"] = true
	}
	out := make([]string, 0, len(in))
	for g := range in {
		out = append(out, g)
	}
	slices.SortFunc(out, func(a, b string) int { return p.groups[a].order - p.groups[b].order })
	return out
}

// depths gives each group's distance from "all", the longest path through
// children, as Ansible ranks group variables. A loop is cut where it closes.
func depths(groups map[string]*group) map[string]int {
	d := map[string]int{}
	var visit func(name string, at int, path map[string]bool)
	visit = func(name string, at int, path map[string]bool) {
		if path[name] {
			return
		}
		if at > d[name] {
			d[name] = at
		}
		g, ok := groups[name]
		if !ok {
			return
		}
		path[name] = true
		for _, c := range g.children {
			visit(c, at+1, path)
		}
		delete(path, name)
	}
	isChild := map[string]bool{}
	for _, g := range groups {
		for _, c := range g.children {
			isChild[c] = true
		}
	}
	for _, g := range groups {
		if g.name != "all" && !isChild[g.name] {
			visit(g.name, 1, map[string]bool{})
		}
	}
	return d
}

// varsOf merges the host's variables: "all", then its groups from the
// shallowest (ties by name), then the host's own line, each over the last.
func (p *parser) varsOf(host string, groups []string, depth map[string]int) map[string]string {
	vars := map[string]string{}
	for k, v := range p.groups["all"].vars {
		vars[k] = v
	}
	ranked := slices.Clone(groups)
	slices.SortStableFunc(ranked, func(a, b string) int {
		if depth[a] != depth[b] {
			return depth[a] - depth[b]
		}
		return strings.Compare(a, b)
	})
	for _, g := range ranked {
		for k, v := range p.groups[g].vars {
			vars[k] = v
		}
	}
	for k, v := range p.hostVars[host] {
		vars[k] = v
	}
	return vars
}

// toHost reads the connection variables, or says why the host is skipped.
func toHost(name string, vars map[string]string, dir string) (Host, *Issue) {
	get := func(keys []string) (string, string) {
		for _, k := range keys {
			if v, ok := vars[k]; ok {
				return k, v
			}
		}
		return "", ""
	}
	for _, keys := range [][]string{varAddress, varUser, varPort, varKey, varConn, varSSHArgs} {
		for _, k := range keys {
			v, ok := vars[k]
			if !ok {
				continue
			}
			if strings.Contains(v, "{{") || strings.Contains(v, "{%") {
				return Host{}, &Issue{Code: SkipJinja, Detail: k}
			}
			if strings.HasPrefix(v, "$ANSIBLE_VAULT") || strings.HasPrefix(v, "!vault") {
				return Host{}, &Issue{Code: SkipVault, Detail: k}
			}
		}
	}
	if _, conn := get(varConn); !sshConnection(conn) {
		return Host{}, &Issue{Code: SkipConnection, Detail: conn}
	}

	h := Host{Name: name, Address: name, Port: 22}
	if _, v := get(varAddress); v != "" {
		h.Address = v
	}
	_, h.User = get(varUser)
	if _, v := get(varPort); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return Host{}, &Issue{Code: SkipPort, Detail: v}
		}
		h.Port = port
	}
	if _, v := get(varKey); v != "" {
		h.KeyFile = keyPath(v, dir)
	}
	var ignored []string
	for _, k := range varSSHArgs {
		jump, other := sshArgs(vars[k])
		if jump != "" {
			h.ProxyJump = jump
		}
		ignored = append(ignored, other...)
	}
	if len(ignored) > 0 {
		h.Notes = append(h.Notes, Issue{Code: NoteSSHArgs, Detail: strings.Join(ignored, ", ")})
	}
	for _, k := range varPassword {
		if _, ok := vars[k]; ok {
			h.Notes = append(h.Notes, Issue{Code: NotePassword})
			break
		}
	}
	return h, nil
}

func sshConnection(c string) bool {
	switch strings.TrimPrefix(c, "ansible.builtin.") {
	case "", "ssh", "smart", "paramiko", "paramiko_ssh":
		return true
	}
	return false
}

// keyPath resolves a key file relative to the inventory; "~" is kept, as
// conn expands it.
func keyPath(p, dir string) string {
	if dir == "" || p == "~" || strings.HasPrefix(p, "~/") || isAbs(p) {
		return p
	}
	return joinPath(dir, p)
}

// sshArgs finds a ProxyJump in ansible_ssh_common_args, and names the
// other options, which are not imported.
func sshArgs(v string) (string, []string) {
	toks, err := split(v)
	if err != nil || len(toks) == 0 {
		return "", nil
	}
	jump := ""
	var other []string
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		next := func() string {
			if i+1 < len(toks) {
				i++
				return toks[i]
			}
			return ""
		}
		switch {
		case tok == "-J":
			jump = next()
		case strings.HasPrefix(tok, "-J"):
			jump = tok[2:]
		case tok == "-o" || strings.HasPrefix(tok, "-o"):
			opt := strings.TrimPrefix(tok, "-o")
			if opt == "" {
				opt = next()
			}
			key, val, _ := strings.Cut(opt, "=")
			if !strings.Contains(opt, "=") {
				key, val, _ = strings.Cut(opt, " ")
			}
			if strings.EqualFold(strings.TrimSpace(key), "ProxyJump") {
				jump = strings.TrimSpace(val)
			} else if key = strings.TrimSpace(key); key != "" && !slices.Contains(other, key) {
				other = append(other, key)
			}
		case strings.HasPrefix(tok, "-"):
			if !slices.Contains(other, tok) {
				other = append(other, tok)
			}
		}
	}
	if strings.EqualFold(jump, "none") {
		jump = ""
	}
	return jump, other
}
