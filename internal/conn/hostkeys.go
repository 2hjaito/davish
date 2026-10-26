package conn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeys is the app's own known_hosts file. The user's ~/.ssh/known_hosts
// is never written.
type HostKeys struct {
	path string

	// mu serializes appends so concurrent approvals do not interleave lines.
	mu sync.Mutex
}

// NewHostKeys uses the known_hosts file at path, creating it on first write.
func NewHostKeys(path string) *HostKeys {
	return &HostKeys{path: path}
}

// hostKeyResult is what the callback saw, read back after the handshake
// because the ssh package may not wrap the callback's error.
type hostKeyResult struct {
	unknown ssh.PublicKey // set when the host is not in the file
	changed bool          // set when the host is known with a different key
}

// callback checks the presented key against the file. An unknown host and a
// changed key both fail the handshake; the caller decides what to tell the
// user from res.
func (h *HostKeys) callback(res *hostKeyResult) (ssh.HostKeyCallback, error) {
	if err := h.ensureFile(); err != nil {
		return nil, err
	}
	check, err := knownhosts.New(h.path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var ke *knownhosts.KeyError
		switch {
		case errors.As(err, &ke) && len(ke.Want) == 0:
			res.unknown = key
			return errHostKeyUnknown
		case errors.As(err, &ke):
			res.changed = true
			return errHostKeyChanged
		}
		return err
	}, nil
}

var (
	errHostKeyUnknown = errors.New("host key unknown")
	errHostKeyChanged = errors.New("host key changed")
)

// algorithms lists the key types already pinned for addr, so the handshake
// asks for those first. Without it a server offering another key type would
// look like a changed key.
func (h *HostKeys) algorithms(addr string) ([]string, error) {
	data, err := os.ReadFile(h.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read known_hosts: %w", err)
	}
	want := knownhosts.Normalize(addr)
	var algos []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		_, hosts, key, _, _, err := ssh.ParseKnownHosts(sc.Bytes())
		if err != nil || !slices.Contains(hosts, want) {
			continue
		}
		for _, a := range algosFor(key.Type()) {
			if !slices.Contains(algos, a) {
				algos = append(algos, a)
			}
		}
	}
	return algos, sc.Err()
}

// algosFor maps a key type to the signature algorithms that prove it.
func algosFor(keyType string) []string {
	if keyType == ssh.KeyAlgoRSA {
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
	}
	return []string{keyType}
}

// Add pins key for addr (host:port).
func (h *HostKeys) Add(addr string, key ssh.PublicKey) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.ensureFile(); err != nil {
		return err
	}
	f, err := os.OpenFile(h.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	line := knownhosts.Line([]string{knownhosts.Normalize(addr)}, key)
	if _, err := f.WriteString(line + "\n"); err != nil {
		f.Close() //nolint:errcheck,gosec // already failing
		return fmt.Errorf("write known_hosts: %w", err)
	}
	return f.Close()
}

func (h *HostKeys) ensureFile() error {
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	f, err := os.OpenFile(h.path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create known_hosts: %w", err)
	}
	return f.Close()
}

// Fingerprint is the SHA256 fingerprint shown to the user, as ssh-keygen -l.
func Fingerprint(key ssh.PublicKey) string {
	return ssh.FingerprintSHA256(key)
}
