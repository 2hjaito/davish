package keyfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// The keys in testdata were written by puttygen 0.8x (version 2 and 3), the
// encrypted ones with the passphrase "test pass". The .pub next to each is
// puttygen's own export of its public half. They are throwaway keys.
const ppkPass = "test pass"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParsePPKKeys(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.ppk.txt")
	if len(files) < 20 {
		t.Fatalf("only %d .ppk fixtures", len(files))
	}
	for _, path := range files {
		file := filepath.Base(path)
		var pass []byte
		if strings.Contains(file, "-enc") {
			pass = []byte(ppkPass)
		}
		t.Run(file, func(t *testing.T) {
			want, _, _, _, err := ssh.ParseAuthorizedKey(readFixture(t, strings.TrimSuffix(file, ".ppk.txt")+".pub"))
			if err != nil {
				t.Fatal(err)
			}
			signer, err := Parse(readFixture(t, file), pass)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(signer.PublicKey().Marshal(), want.Marshal()) {
				t.Fatal("public key differs from puttygen's export")
			}
			// A signature the public key accepts shows the private half is right.
			sig, err := signer.Sign(nil, []byte("hello"))
			if err != nil {
				t.Fatal(err)
			}
			if err := signer.PublicKey().Verify([]byte("hello"), sig); err != nil {
				t.Fatalf("signature does not verify: %v", err)
			}
		})
	}
}

func TestParsePPKPassphrase(t *testing.T) {
	for _, name := range []string{"ed25519-v3-enc", "rsa-v2-enc"} {
		data := readFixture(t, name+".ppk.txt")

		var missing *ssh.PassphraseMissingError
		if _, err := Parse(data, nil); !errors.As(err, &missing) || missing.PublicKey == nil {
			t.Errorf("%s without a passphrase: err = %v; want PassphraseMissingError with the public key", name, err)
		}
		if _, err := Parse(data, []byte("wrong")); !errors.Is(err, ErrWrongPassphrase) {
			t.Errorf("%s with a wrong passphrase: err = %v; want ErrWrongPassphrase", name, err)
		}
		// An empty passphrase is a passphrase, not a missing one.
		if _, err := Parse(data, []byte{}); !errors.Is(err, ErrWrongPassphrase) {
			t.Errorf("%s with an empty passphrase: err = %v; want ErrWrongPassphrase", name, err)
		}
	}
	// A passphrase offered for a key that has none changes nothing.
	if _, err := Parse(readFixture(t, "ed25519-v3.ppk.txt"), []byte("anything")); err != nil {
		t.Errorf("plain key with a passphrase: %v", err)
	}
}

func TestParsePPKTolerates(t *testing.T) {
	data := readFixture(t, "ed25519-v3.ppk.txt")
	for name, in := range map[string][]byte{
		"BOM":        append([]byte("\xef\xbb\xbf"), data...),
		"indented":   append([]byte("\n  "), data...),
		"CRLF":       bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n")),
		"no newline": bytes.TrimRight(data, "\n"),
	} {
		if _, err := Parse(in, nil); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestParsePPKRefuses(t *testing.T) {
	plain := string(readFixture(t, "ed25519-v3.ppk.txt"))
	enc := string(readFixture(t, "ed25519-v3-enc.ppk.txt"))

	unsupported := map[string]string{
		"version 1":          strings.Replace(plain, "-3:", "-1:", 1),
		"Argon2d":            strings.Replace(enc, "Argon2id", "Argon2d", 1),
		"huge Argon2 memory": strings.Replace(enc, "Argon2-Memory: 8192", "Argon2-Memory: 4000000000", 1),
		"huge Argon2 passes": strings.Replace(enc, "Argon2-Passes: ", "Argon2-Passes: 4000000", 1),
	}
	for name, in := range unsupported {
		if _, err := Parse([]byte(in), []byte(ppkPass)); !errors.Is(err, ErrPPK) {
			t.Errorf("%s: err = %v; want ErrPPK", name, err)
		}
	}

	damaged := map[string]string{
		"comment edited":     strings.Replace(plain, "Comment: test-ed25519", "Comment: other", 1),
		"MAC edited":         strings.Replace(plain, "Private-MAC: ", "Private-MAC: 00", 1),
		"duplicate header":   strings.Replace(plain, "Encryption: none\n", "Encryption: none\nEncryption: none\n", 1),
		"unknown header":     strings.Replace(plain, "Comment:", "Colour: red\nComment:", 1),
		"bad line count":     strings.Replace(plain, "Public-Lines: 2", "Public-Lines: 99", 1),
		"negative count":     strings.Replace(plain, "Public-Lines: 2", "Public-Lines: -1", 1),
		"bad base64":         strings.Replace(plain, "AAAAC3Nza", "!!!!C3Nza", 1),
		"no private key":     plain[:strings.Index(plain, "Private-Lines")],
		"unknown encryption": strings.Replace(plain, "Encryption: none", "Encryption: rot13", 1),
	}
	for name, in := range damaged {
		_, err := Parse([]byte(in), nil)
		if err == nil || errors.Is(err, ErrWrongPassphrase) {
			t.Errorf("%s: err = %v; want a parse error", name, err)
		}
	}
}

// A file cut anywhere must give an error, never a panic.
func TestParsePPKTruncated(t *testing.T) {
	for _, name := range []string{"rsa-v2.ppk.txt", "ecdsa256-v3.ppk.txt"} {
		data := bytes.TrimRight(readFixture(t, name), "\n")
		for n := 0; n < len(data); n += 7 {
			if _, err := Parse(data[:n], []byte(ppkPass)); err == nil {
				t.Fatalf("%s cut at %d bytes parsed", name, n)
			}
		}
	}
}
