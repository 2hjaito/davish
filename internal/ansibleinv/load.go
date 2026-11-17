package ansibleinv

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"davish/internal/apperr"
	"davish/internal/sshconfig"
	"davish/internal/store"
)

// maxFile bounds the inventory read; a real one is a few KB.
const maxFile = 4 << 20

// Load reads the inventory at path. localUser fills in hosts with no
// ansible_user, as ssh would. A YAML inventory or a script returns its
// own error code; neither is read or run.
func Load(path, localUser string) (Inventory, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yml", ".yaml", ".json":
		return Inventory{}, apperr.New(apperr.InventoryYAMLUnsupported, "YAML inventories are not supported yet")
	}
	f, err := os.Open(path) //nolint:gosec // a file the user chose to import
	if err != nil {
		return Inventory{}, apperr.Wrap(apperr.NotFound, "cannot open the inventory", err)
	}
	defer f.Close() //nolint:errcheck // read-only
	st, err := f.Stat()
	if err != nil {
		return Inventory{}, apperr.Wrap(apperr.NotFound, "cannot open the inventory", err)
	}
	if st.IsDir() {
		return Inventory{}, apperr.New(apperr.InvalidInput, "choose the inventory file, not its folder")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFile+1))
	if err != nil {
		return Inventory{}, apperr.Wrap(apperr.Internal, "cannot read the inventory", err)
	}
	if len(data) > maxFile {
		return Inventory{}, apperr.New(apperr.InvalidInput, "the inventory is larger than 4 MB")
	}
	if code := detect(data); code != "" {
		return Inventory{}, apperr.New(code, "only INI inventories can be imported")
	}
	inv, err := Parse(bytes.NewReader(data), filepath.Dir(path))
	if err != nil {
		return Inventory{}, apperr.Wrap(apperr.InvalidInput, "cannot read the inventory: "+err.Error(), err)
	}
	for i := range inv.Hosts {
		if inv.Hosts[i].User == "" {
			inv.Hosts[i].User = localUser
		}
	}
	return inv, nil
}

// yamlKey is a first line such as "all:" or "webservers:", which INI
// never has: there a host's port follows the colon.
var yamlKey = regexp.MustCompile(`^[A-Za-z0-9_.-]+:\s*$`)

// detect tells YAML (or JSON) and scripts from INI by the first line
// that is not blank or a comment.
func detect(data []byte) apperr.Code {
	if bytes.HasPrefix(data, []byte("#!")) {
		return apperr.InventoryScriptUnsupported
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line == "---" || line[0] == '{' || yamlKey.MatchString(line) {
			return apperr.InventoryYAMLUnsupported
		}
		return ""
	}
	return ""
}

// Default returns the inventory Ansible itself would use, when it is a
// file: $ANSIBLE_INVENTORY, then "inventory" in ~/.ansible.cfg, then
// /etc/ansible/hosts. "" when none exists.
func Default() string {
	var tries []string
	if env := os.Getenv("ANSIBLE_INVENTORY"); env != "" {
		tries = append(tries, strings.Split(env, ",")...)
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		if p := cfgInventory(filepath.Join(home, ".ansible.cfg")); p != "" {
			tries = append(tries, p)
		}
	}
	tries = append(tries, "/etc/ansible/hosts")
	for _, p := range tries {
		p = strings.TrimSpace(p)
		if home != "" && strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
		// Only checked to exist; the user still confirms it in the dialog.
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() { //nolint:gosec // the user's own Ansible settings

			return p
		}
	}
	return ""
}

// cfgInventory reads "inventory =" from the [defaults] section of an
// ansible.cfg; the first path when it lists several.
func cfgInventory(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // the user's own ansible.cfg
	if err != nil {
		return ""
	}
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if ok && section == "defaults" && strings.TrimSpace(key) == "inventory" {
			first, _, _ := strings.Cut(strings.TrimSpace(val), ",")
			return strings.TrimSpace(first)
		}
	}
	return ""
}

// Candidate is an inventory host as the import dialog lists it.
type Candidate struct {
	Host
	// Exists is set when a saved server already has this user@host:port.
	Exists bool `json:"exists"`
}

// Listing is an inventory as the import dialog shows it.
type Listing struct {
	Path    string      `json:"path"`
	Hosts   []Candidate `json:"hosts"`
	Skipped []Skipped   `json:"skipped"`
	Groups  []string    `json:"groups"`
}

// List marks which hosts are already saved.
func List(path string, inv Inventory, saved []store.Server) Listing {
	cands := sshconfig.Candidates(SSHHosts(inv), saved)
	out := Listing{Path: path, Hosts: make([]Candidate, 0, len(inv.Hosts)), Skipped: inv.Skipped, Groups: inv.Groups}
	for i, h := range inv.Hosts {
		out.Hosts = append(out.Hosts, Candidate{Host: h, Exists: cands[i].Exists})
	}
	return out
}

// SSHHosts turns the hosts into ~/.ssh/config hosts, so they import the
// same way; a ProxyJump naming another inventory host uses its settings.
func SSHHosts(inv Inventory) []sshconfig.Host {
	out := make([]sshconfig.Host, 0, len(inv.Hosts))
	for _, h := range inv.Hosts {
		out = append(out, sshconfig.Host{
			Alias:        h.Name,
			HostName:     h.Address,
			User:         h.User,
			Port:         h.Port,
			IdentityFile: h.KeyFile,
			ProxyJump:    h.ProxyJump,
		})
	}
	return out
}

// Import saves the chosen hosts, and the jump hosts they need, as
// servers; a host already saved is reused. It returns the servers added.
func Import(ctx context.Context, saver sshconfig.Saver, inv Inventory, names []string, localUser string) ([]store.Server, error) {
	for _, n := range names {
		if !slices.ContainsFunc(inv.Hosts, func(h Host) bool { return h.Name == n }) {
			return nil, apperr.New(apperr.NotFound, fmt.Sprintf("host %q is not in the inventory, or cannot be imported", n))
		}
	}
	return sshconfig.Import(ctx, saver, SSHHosts(inv), names, localUser)
}
