package conn

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"davish/internal/store"
)

type passphraseOnly string

func (p passphraseOnly) Get(_ string, kind store.SecretKind) (string, error) {
	if kind == store.SecretPassphrase && p != "" {
		return string(p), nil
	}
	return "", store.ErrNoSecret
}

// An ed25519 key with a passphrase, saved by Notepad with a byte order mark
// and CRLF line ends, as Windows users hand it over.
func TestLoadKeyEncryptedEd25519WithBOM(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const pass = "correct horse"
	block, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(pass))
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte("\xef\xbb\xbf"), pem.EncodeToMemory(block)...)
	var crlf []byte
	for _, b := range data {
		if b == '\n' {
			crlf = append(crlf, '\r')
		}
		crlf = append(crlf, b)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, crlf, 0o600); err != nil {
		t.Fatal(err)
	}

	m := &Manager{secrets: passphraseOnly(pass)}
	if _, err := m.loadKey(store.Server{ID: "s", KeyPath: path}); err != nil {
		t.Fatalf("loadKey: %v", err)
	}

	m = &Manager{secrets: passphraseOnly("")}
	if _, err := m.loadKey(store.Server{ID: "s", KeyPath: path}); err == nil {
		t.Fatal("want an error without a saved passphrase")
	}
}
