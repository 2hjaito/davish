package sshkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
)

// LocalKey is a key pair in ~/.ssh on this machine. Only the public half
// ever leaves this package; the private key stays in its file.
type LocalKey struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
	Public      string `json:"public"`
	// Encrypted keys need their passphrase saved for a server to use them.
	Encrypted bool `json:"encrypted"`
}

// LocalDir is ~/.ssh of the user running Davish.
func LocalDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	return filepath.Join(home, ".ssh"), nil
}

// LocalKeys lists the pairs in dir: a public key file with its private key
// beside it.
func LocalKeys(dir string) ([]LocalKey, error) {
	keys := []LocalKey{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return keys, nil
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.FSPermissionDenied, "cannot list ~/.ssh", err)
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".pub")
		if !ok || e.IsDir() {
			continue
		}
		if k, err := readLocal(dir, name); err == nil {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b LocalKey) int { return strings.Compare(a.Name, b.Name) })
	return keys, nil
}

func readLocal(dir, name string) (LocalKey, error) {
	p := filepath.Join(dir, name)
	data, err := os.ReadFile(p + ".pub") //nolint:gosec // a .pub listed in ~/.ssh
	if err != nil {
		return LocalKey{}, err
	}
	line := strings.TrimSpace(string(data))
	pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return LocalKey{}, err
	}
	priv, err := os.ReadFile(p) //nolint:gosec // its private half, only to tell if it is encrypted
	if err != nil {
		return LocalKey{}, err
	}
	_, perr := ssh.ParseRawPrivateKey(priv)
	var missing *ssh.PassphraseMissingError
	return LocalKey{
		Name: name, Path: p, Type: pub.Type(), Fingerprint: ssh.FingerprintSHA256(pub),
		Comment: comment, Public: line, Encrypted: errors.As(perr, &missing),
	}, nil
}

// GenerateLocal makes an ed25519 pair dir/name and dir/name.pub, never
// over an existing file, encrypted with passphrase unless it is empty.
// The private key file is 0600 and ~/.ssh 0700, as ssh requires.
func GenerateLocal(dir, name, comment, passphrase string) (LocalKey, error) {
	if !keyName.MatchString(name) || strings.HasSuffix(name, ".pub") || name == "authorized_keys" || name == "known_hosts" || name == "config" {
		return LocalKey{}, apperr.New(apperr.InvalidInput, "not a key file name")
	}
	if len(comment) > 100 || strings.ContainsAny(comment, "\r\n\x00") {
		return LocalKey{}, apperr.New(apperr.InvalidInput, "comment must be one short line")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return LocalKey{}, apperr.Wrap(apperr.Internal, "cannot generate a key", err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, comment)
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, comment, []byte(passphrase))
	}
	if err != nil {
		return LocalKey{}, apperr.Wrap(apperr.Internal, "cannot encode the key", err)
	}
	sp, err := ssh.NewPublicKey(pub)
	if err != nil {
		return LocalKey{}, apperr.Wrap(apperr.Internal, "cannot encode the key", err)
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sp)))
	if comment != "" {
		line += " " + comment
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return LocalKey{}, apperr.Wrap(apperr.FSPermissionDenied, "cannot create ~/.ssh", err)
	}
	p := filepath.Join(dir, name)
	if err := writeNew(p, pem.EncodeToMemory(block), 0o600); err != nil {
		return LocalKey{}, err
	}
	if err := writeNew(p+".pub", []byte(line+"\n"), 0o644); err != nil {
		os.Remove(p) //nolint:errcheck,gosec // undo the half-made pair
		return LocalKey{}, err
	}
	return readLocal(dir, name)
}

// writeNew creates path with data, failing if it exists.
func writeNew(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode) //nolint:gosec // a name checked by keyName in ~/.ssh
	if errors.Is(err, os.ErrExist) {
		return apperr.New(apperr.FSExists, "a key with that name exists")
	}
	if err != nil {
		return apperr.Wrap(apperr.FSPermissionDenied, "cannot write the key", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close() //nolint:errcheck,gosec // already failing
		return apperr.Wrap(apperr.FSPermissionDenied, "cannot write the key", err)
	}
	if err := f.Close(); err != nil {
		return apperr.Wrap(apperr.FSPermissionDenied, "cannot write the key", err)
	}
	return nil
}
