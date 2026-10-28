package sshkeys

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
)

func TestGenerateLocal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ssh")
	plain, err := GenerateLocal(dir, "davish_tsc", "dang@laptop", "")
	if err != nil {
		t.Fatal(err)
	}
	locked, err := GenerateLocal(dir, "davish_locked", "", "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Encrypted || !locked.Encrypted || plain.Type != "ssh-ed25519" || plain.Comment != "dang@laptop" {
		t.Errorf("plain = %+v\nlocked = %+v", plain, locked)
	}

	for p, want := range map[string]os.FileMode{dir: 0o700, plain.Path: 0o600, plain.Path + ".pub": 0o644} {
		fi, err := os.Stat(p)
		if err != nil || fi.Mode().Perm() != want {
			t.Errorf("%s mode = %v, want %v", p, fi.Mode().Perm(), want)
		}
	}

	data, _ := os.ReadFile(locked.Path)
	signer, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte("correct horse"))
	if err != nil || ssh.FingerprintSHA256(signer.PublicKey()) != locked.Fingerprint {
		t.Errorf("encrypted key does not open with its passphrase: %v", err)
	}

	if _, err := GenerateLocal(dir, "davish_tsc", "", ""); !isCode(err, apperr.FSExists) {
		t.Errorf("overwrote an existing key: %v", err)
	}
	if _, err := GenerateLocal(dir, "../evil", "", ""); !isCode(err, apperr.InvalidInput) {
		t.Errorf("bad name accepted: %v", err)
	}

	keys, err := LocalKeys(dir)
	if err != nil || len(keys) != 2 || keys[0].Name != "davish_locked" || keys[1].Public != plain.Public {
		t.Errorf("LocalKeys = %+v, %v", keys, err)
	}
	if keys, err := LocalKeys(filepath.Join(dir, "missing")); err != nil || len(keys) != 0 {
		t.Errorf("missing dir = %+v, %v", keys, err)
	}
}

func isCode(err error, code apperr.Code) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == code
}
