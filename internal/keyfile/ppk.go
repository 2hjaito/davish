package keyfile

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // PPK version 2 derives its keys and MAC with SHA-1; the format is fixed
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/ssh"
)

// PuTTY's .ppk format, versions 2 (puttygen before 0.75) and 3 (since 2021):
//
//	PuTTY-User-Key-File-3: ssh-ed25519
//	Encryption: aes256-cbc | none
//	Comment: ...
//	Public-Lines: n        n lines of base64, an SSH public key blob
//	Key-Derivation: Argon2id, Argon2-Memory/Passes/Parallelism/Salt (v3, encrypted)
//	Private-Lines: n       n lines of base64, the private fields (AES-256-CBC when encrypted)
//	Private-MAC: hex       HMAC over the header values and both blobs
//
// The MAC is what tells a wrong passphrase from a good one: CBC has no tag.

const (
	// maxPPKLines and maxPPKBlob bound what a damaged file can make us read.
	maxPPKLines = 4096
	maxPPKBlob  = 64 << 10
	// The Argon2 cost comes from the file. puttygen writes 8 MiB and a few
	// dozen passes; these caps keep a hostile file from asking for gigabytes.
	maxArgonMemoryKiB = 512 << 10
	maxArgonPasses    = 4096
	maxArgonLanes     = 64
)

// ppkFile is one .ppk file before any decryption.
type ppkFile struct {
	version    int
	algorithm  string
	encryption string
	comment    string
	public     []byte
	private    []byte
	mac        []byte

	kdf           string
	argonMemory   uint32
	argonPasses   uint32
	argonLanes    uint32
	argonSalt     []byte
	argonMaterial []byte // cached: cipher key, IV and MAC key come from one derivation
}

func ppkErr(format string, args ...any) error {
	return fmt.Errorf("ppk: "+format, args...)
}

// parsePPK reads a PuTTY private key. Without a passphrase an encrypted key
// gives *ssh.PassphraseMissingError (carrying its public key); a wrong one
// gives ErrWrongPassphrase. A version or key derivation this reader does not
// know gives an error that wraps ErrPPK.
func parsePPK(data, passphrase []byte) (ssh.Signer, error) {
	f, err := readPPK(data)
	if err != nil {
		return nil, err
	}
	encrypted := f.encryption == "aes256-cbc"
	if encrypted && passphrase == nil {
		pub, _ := ssh.ParsePublicKey(f.public)
		return nil, &ssh.PassphraseMissingError{PublicKey: pub}
	}
	if !encrypted {
		// An unencrypted key is MACed with the empty passphrase.
		passphrase = nil
	}
	private := f.private
	if encrypted {
		if private, err = f.decrypt(passphrase); err != nil {
			return nil, err
		}
	}
	if err := f.verifyMAC(passphrase, private, encrypted); err != nil {
		return nil, err
	}
	return f.signer(private)
}

func readPPK(data []byte) (*ppkFile, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > maxPPKLines {
		return nil, ppkErr("file has too many lines")
	}

	f := &ppkFile{}
	seen := map[string]bool{}
	for i := 0; i < len(lines); {
		name, value, ok := strings.Cut(lines[i], ": ")
		if !ok {
			return nil, ppkErr("line %d is not a \"Name: value\" header", i+1)
		}
		i++
		if i == 1 {
			if !strings.HasPrefix(name, ppkHeader) {
				return nil, ppkErr("not a PuTTY key file")
			}
			switch name[len(ppkHeader):] {
			case "2":
				f.version = 2
			case "3":
				f.version = 3
			default:
				return nil, fmt.Errorf("%w: format version %q is not supported (2 and 3 are)", ErrPPK, name[len(ppkHeader):])
			}
			f.algorithm = value
			continue
		}
		if seen[name] {
			return nil, ppkErr("header %q appears twice", name)
		}
		seen[name] = true

		var err error
		switch name {
		case "Encryption":
			f.encryption = value
		case "Comment":
			f.comment = value
		case "Public-Lines", "Private-Lines":
			var blob []byte
			if blob, i, err = readBlob(lines, i, value); err != nil {
				return nil, err
			}
			if name == "Public-Lines" {
				f.public = blob
			} else {
				f.private = blob
			}
		case "Key-Derivation":
			f.kdf = value
		case "Argon2-Memory":
			f.argonMemory, err = uintHeader(value)
		case "Argon2-Passes":
			f.argonPasses, err = uintHeader(value)
		case "Argon2-Parallelism":
			f.argonLanes, err = uintHeader(value)
		case "Argon2-Salt":
			f.argonSalt, err = hex.DecodeString(value)
		case "Private-MAC":
			f.mac, err = hex.DecodeString(value)
		default:
			return nil, ppkErr("unknown header %q", name)
		}
		if err != nil {
			return nil, ppkErr("bad %s: %v", name, err)
		}
	}

	if f.public == nil || f.private == nil || f.mac == nil {
		return nil, ppkErr("the file ends before the keys or the MAC")
	}
	switch f.encryption {
	case "none":
	case "aes256-cbc":
		if f.version == 3 && (f.argonSalt == nil || f.argonMemory == 0 || f.argonPasses == 0 || f.argonLanes == 0) {
			return nil, ppkErr("an encrypted version 3 key needs its Argon2 settings")
		}
	default:
		return nil, ppkErr("unknown encryption %q", f.encryption)
	}
	return f, nil
}

func uintHeader(value string) (uint32, error) {
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", value)
	}
	return uint32(n), nil
}

// readBlob joins count base64 lines starting at lines[at] and returns the
// decoded bytes and the index after them.
func readBlob(lines []string, at int, count string) ([]byte, int, error) {
	n, err := strconv.Atoi(count)
	if err != nil || n < 0 || n > len(lines)-at {
		return nil, 0, ppkErr("bad line count %q", count)
	}
	joined := strings.Join(lines[at:at+n], "")
	if len(joined) > base64.StdEncoding.EncodedLen(maxPPKBlob) {
		return nil, 0, ppkErr("key data is too large")
	}
	blob, err := base64.StdEncoding.DecodeString(joined)
	if err != nil {
		return nil, 0, ppkErr("key data is not valid base64")
	}
	return blob, at + n, nil
}

// keys derives what the passphrase unlocks: the AES key and IV, and for
// version 3 the MAC key from the same Argon2 run.
func (f *ppkFile) keys(passphrase []byte) (key, iv, macKey []byte, err error) {
	if f.version == 2 {
		// Two SHA-1 blocks, counter first, give the 32-byte key; the IV is zero.
		var material []byte
		for counter := uint32(0); len(material) < 32; counter++ {
			var c [4]byte
			binary.BigEndian.PutUint32(c[:], counter)
			h := sha1.New() //nolint:gosec // see the import
			h.Write(c[:])
			h.Write(passphrase)
			material = h.Sum(material)
		}
		return material[:32], make([]byte, aes.BlockSize), nil, nil
	}

	if f.argonMaterial == nil {
		if f.argonMemory > maxArgonMemoryKiB || f.argonPasses > maxArgonPasses || f.argonLanes > maxArgonLanes {
			return nil, nil, nil, fmt.Errorf("%w: Argon2 cost (%d KiB, %d passes, %d lanes) is above what Davish reads", ErrPPK, f.argonMemory, f.argonPasses, f.argonLanes)
		}
		switch f.kdf {
		case "Argon2id":
			f.argonMaterial = argon2.IDKey(passphrase, f.argonSalt, f.argonPasses, f.argonMemory, uint8(f.argonLanes), 80)
		case "Argon2i":
			f.argonMaterial = argon2.Key(passphrase, f.argonSalt, f.argonPasses, f.argonMemory, uint8(f.argonLanes), 80)
		default:
			// Argon2d is in puttygen's menu but not in x/crypto.
			return nil, nil, nil, fmt.Errorf("%w: key derivation %q is not supported; save the key again in PuTTYgen with Argon2id", ErrPPK, f.kdf)
		}
	}
	m := f.argonMaterial
	return m[:32], m[32:48], m[48:80], nil
}

// decrypt opens the private blob. It keeps the random padding after the key
// fields, since the MAC covers it.
func (f *ppkFile) decrypt(passphrase []byte) ([]byte, error) {
	if len(f.private) == 0 || len(f.private)%aes.BlockSize != 0 {
		return nil, ppkErr("the encrypted key is not a whole number of AES blocks")
	}
	key, iv, _, err := f.keys(passphrase)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(f.private))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, f.private)
	return out, nil
}

func (f *ppkFile) verifyMAC(passphrase, private []byte, encrypted bool) error {
	var input []byte
	for _, part := range [][]byte{[]byte(f.algorithm), []byte(f.encryption), []byte(f.comment), f.public, private} {
		input = binary.BigEndian.AppendUint32(input, uint32(len(part))) //nolint:gosec // each part is a blob read under maxPPKBlob
		input = append(input, part...)
	}

	var sum []byte
	if f.version == 2 {
		seed := sha1.New() //nolint:gosec // see the import
		seed.Write([]byte("putty-private-key-file-mac-key"))
		seed.Write(passphrase)
		mac := hmac.New(sha1.New, seed.Sum(nil)) //nolint:gosec // see the import
		mac.Write(input)
		sum = mac.Sum(nil)
	} else {
		var macKey []byte // empty for a key without encryption
		if encrypted {
			var err error
			if _, _, macKey, err = f.keys(passphrase); err != nil {
				return err
			}
		}
		mac := hmac.New(sha256.New, macKey)
		mac.Write(input)
		sum = mac.Sum(nil)
	}
	if hmac.Equal(sum, f.mac) {
		return nil
	}
	if encrypted {
		return ErrWrongPassphrase
	}
	return ppkErr("the key file is damaged (its MAC does not match)")
}

// wire reads SSH wire fields (uint32 length, then bytes) from a blob.
type wire struct {
	b   []byte
	err error
}

func (w *wire) string() []byte {
	if w.err != nil {
		return nil
	}
	if len(w.b) < 4 {
		w.err = errors.New("short key data")
		return nil
	}
	n := binary.BigEndian.Uint32(w.b)
	if int64(n) > int64(len(w.b)-4) {
		w.err = errors.New("short key data")
		return nil
	}
	s := w.b[4 : 4+n]
	w.b = w.b[4+n:]
	return s
}

// mpint is the big-endian number in the next field. PuTTY only writes
// non-negative ones.
func (w *wire) mpint() *big.Int {
	return new(big.Int).SetBytes(w.string())
}

// signer builds the key from the decrypted private fields and refuses one
// whose public half is not the blob in the file.
func (f *ppkFile) signer(private []byte) (ssh.Signer, error) {
	pub := wire{b: f.public}
	if string(pub.string()) != f.algorithm || pub.err != nil {
		return nil, ppkErr("the public key does not match the %q header", f.algorithm)
	}
	priv := wire{b: private}

	var key any
	switch f.algorithm {
	case "ssh-rsa":
		e, n := pub.mpint(), pub.mpint()
		d, p, q := priv.mpint(), priv.mpint(), priv.mpint()
		if pub.err != nil || priv.err != nil || !e.IsInt64() || e.Int64() > 1<<31-1 {
			return nil, ppkErr("the RSA key data is damaged")
		}
		k := &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n, E: int(e.Int64())}, D: d, Primes: []*big.Int{p, q}}
		if err := k.Validate(); err != nil {
			return nil, ppkErr("the RSA key is not consistent: %v", err)
		}
		k.Precompute()
		key = k
	case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		curve := map[string]elliptic.Curve{
			"ecdsa-sha2-nistp256": elliptic.P256(),
			"ecdsa-sha2-nistp384": elliptic.P384(),
			"ecdsa-sha2-nistp521": elliptic.P521(),
		}[f.algorithm]
		d := priv.mpint()
		if priv.err != nil {
			return nil, ppkErr("the ECDSA key data is damaged")
		}
		raw := make([]byte, (curve.Params().BitSize+7)/8)
		if d.BitLen() > len(raw)*8 {
			return nil, ppkErr("the ECDSA key data is damaged")
		}
		d.FillBytes(raw)
		k, err := ecdsa.ParseRawPrivateKey(curve, raw)
		if err != nil {
			return nil, ppkErr("the ECDSA key is not valid: %v", err)
		}
		key = k
	case "ssh-ed25519":
		// The private field is the 32-byte seed as an SSH string.
		seed := priv.string()
		if priv.err != nil || len(seed) != ed25519.SeedSize {
			return nil, ppkErr("the Ed25519 key data is damaged")
		}
		key = ed25519.NewKeyFromSeed(seed)
	default:
		return nil, fmt.Errorf("%w: key type %q is not supported", ErrPPK, f.algorithm)
	}

	s, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(s.PublicKey().Marshal(), f.public) {
		return nil, ppkErr("the private key is not the one the public key describes")
	}
	return s, nil
}
