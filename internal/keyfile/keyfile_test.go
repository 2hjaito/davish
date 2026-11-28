package keyfile

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

const pass = "correct horse"

// keygen makes a key with ssh-keygen, the way users make theirs. Tests skip
// where ssh-keygen is missing or too old for the cipher.
func keygen(t *testing.T, typ, cipherName string) []byte {
	t.Helper()
	bin, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen not found")
	}
	path := filepath.Join(t.TempDir(), "id")
	args := []string{"-q", "-t", typ, "-N", pass, "-C", "", "-f", path}
	if cipherName != "" {
		args = append(args, "-Z", cipherName)
	}
	if out, err := exec.Command(bin, args...).CombinedOutput(); err != nil { //nolint:gosec // test-only, fixed arguments
		t.Skipf("ssh-keygen %s %s: %v: %s", typ, cipherName, err, out)
	}
	data, err := os.ReadFile(path) //nolint:gosec // the test's own temp file
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParseEveryCipher(t *testing.T) {
	for name := range ciphers {
		for _, typ := range []string{"ed25519", "ecdsa"} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				data := keygen(t, typ, name)

				var missing *ssh.PassphraseMissingError
				if _, err := Parse(data, nil); !errors.As(err, &missing) {
					t.Fatalf("no passphrase: err = %v, want PassphraseMissingError", err)
				}
				s, err := Parse(data, []byte(pass))
				if err != nil {
					t.Fatalf("Parse: %v", err)
				}
				if !bytesEqual(s.PublicKey().Marshal(), missing.PublicKey.Marshal()) {
					t.Error("decrypted key does not match the file's public key")
				}
				if _, err := Parse(data, []byte("wrong")); !errors.Is(err, ErrWrongPassphrase) {
					t.Errorf("wrong passphrase: err = %v, want ErrWrongPassphrase", err)
				}
			})
		}
	}
}

func TestParseDefaultRSA(t *testing.T) {
	data := keygen(t, "rsa", "")
	if _, err := Parse(data, []byte(pass)); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestParseBOM(t *testing.T) {
	data := append([]byte("\xef\xbb\xbf"), keygen(t, "ed25519", "aes256-gcm@openssh.com")...)
	if _, err := Parse(data, []byte(pass)); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func bytesEqual(a, b []byte) bool { return string(a) == string(b) }
