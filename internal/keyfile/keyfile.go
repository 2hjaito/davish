// Package keyfile reads SSH private key files. It adds to x/crypto/ssh the
// ciphers OpenSSH may encrypt a key with that x/crypto refuses: it only
// decrypts aes256-ctr and aes256-cbc, while ssh-keygen -Z, Termix and other
// tools also write aes128/192-ctr, aes128/192-cbc, aes128/256-gcm and
// chacha20-poly1305 keys (issue #162).
package keyfile

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/poly1305" //nolint:staticcheck // OpenSSH's chacha20-poly1305 needs the raw one-time MAC, not an AEAD
	"golang.org/x/crypto/ssh"
)

const magic = "openssh-key-v1\x00"

// ErrWrongPassphrase means the passphrase does not open the key.
var ErrWrongPassphrase = x509.IncorrectPasswordError

// maxRounds caps the bcrypt rounds read from the file, as x/crypto does,
// so a crafted key cannot keep the CPU busy for hours.
const maxRounds = 1 << 11

type encrypted struct {
	CipherName   string
	KdfName      string
	KdfOpts      string
	NumKeys      uint32
	PubKey       []byte
	PrivKeyBlock []byte
	Rest         []byte `ssh:"rest"`
}

// spec is one OpenSSH key cipher: key and IV lengths from the KDF, the
// tag after the encrypted block, and how to decrypt it in place.
type spec struct {
	keyLen, ivLen, tagLen int
	open                  func(key, iv, block, tag []byte) error
}

var ciphers = map[string]spec{
	"aes128-ctr":                    {16, 16, 0, aesCTR},
	"aes192-ctr":                    {24, 16, 0, aesCTR},
	"aes256-ctr":                    {32, 16, 0, aesCTR},
	"aes128-cbc":                    {16, 16, 0, aesCBC},
	"aes192-cbc":                    {24, 16, 0, aesCBC},
	"aes256-cbc":                    {32, 16, 0, aesCBC},
	"aes128-gcm@openssh.com":        {16, 12, 16, aesGCM},
	"aes256-gcm@openssh.com":        {32, 12, 16, aesGCM},
	"chacha20-poly1305@openssh.com": {64, 0, 16, chachaPoly},
}

// Parse reads a private key, with passphrase when it is encrypted. With
// no passphrase, an encrypted key gives *ssh.PassphraseMissingError; a
// passphrase that does not open it gives ErrWrongPassphrase.
func Parse(data, passphrase []byte) (ssh.Signer, error) {
	// Notepad saves with a byte order mark, which the PEM reader rejects.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if passphrase == nil {
		return ssh.ParsePrivateKey(data)
	}
	plain, ok, err := decryptOpenSSH(data, passphrase)
	if err != nil {
		return nil, err
	}
	if ok {
		return ssh.ParsePrivateKey(plain)
	}
	return ssh.ParsePrivateKeyWithPassphrase(data, passphrase)
}

// decryptOpenSSH turns an encrypted OpenSSH key into the same key without
// encryption, in PEM. ok is false for anything else (PEM keys, unencrypted
// keys), which x/crypto reads by itself.
func decryptOpenSSH(data, passphrase []byte) ([]byte, bool, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "OPENSSH PRIVATE KEY" {
		return nil, false, nil
	}
	raw := block.Bytes
	if !bytes.HasPrefix(raw, []byte(magic)) {
		return nil, false, nil
	}
	var w encrypted
	if err := ssh.Unmarshal(raw[len(magic):], &w); err != nil {
		return nil, false, err
	}
	if w.CipherName == "none" {
		return nil, false, nil
	}
	if w.KdfName != "bcrypt" {
		return nil, false, fmt.Errorf("ssh: unknown KDF %q, only supports %q", w.KdfName, "bcrypt")
	}
	sp, known := ciphers[w.CipherName]
	if !known {
		return nil, false, fmt.Errorf("ssh: unknown cipher %q", w.CipherName)
	}
	if len(w.Rest) < sp.tagLen {
		return nil, false, errors.New("ssh: encrypted private key is truncated")
	}

	var opts struct {
		Salt   string
		Rounds uint32
	}
	if err := ssh.Unmarshal([]byte(w.KdfOpts), &opts); err != nil {
		return nil, false, err
	}
	if opts.Rounds > maxRounds {
		return nil, false, fmt.Errorf("ssh: bcrypt KDF rounds %d exceed maximum %d", opts.Rounds, maxRounds)
	}
	k, err := bcryptPBKDF(passphrase, []byte(opts.Salt), int(opts.Rounds), sp.keyLen+sp.ivLen)
	if err != nil {
		return nil, false, err
	}
	priv := append([]byte(nil), w.PrivKeyBlock...)
	if err := sp.open(k[:sp.keyLen], k[sp.keyLen:], priv, w.Rest[:sp.tagLen]); err != nil {
		return nil, false, err
	}
	// The block starts with the same random number twice; a wrong
	// passphrase leaves two different ones (CTR and CBC have no tag).
	if len(priv) < 8 || subtle.ConstantTimeCompare(priv[:4], priv[4:8]) != 1 {
		return nil, false, ErrWrongPassphrase
	}

	plain := ssh.Marshal(encrypted{
		CipherName:   "none",
		KdfName:      "none",
		NumKeys:      w.NumKeys,
		PubKey:       w.PubKey,
		PrivKeyBlock: priv,
	})
	return pem.EncodeToMemory(&pem.Block{Type: block.Type, Bytes: append([]byte(magic), plain...)}), true, nil
}

func aesCTR(key, iv, block, _ []byte) error {
	c, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	cipher.NewCTR(c, iv).XORKeyStream(block, block)
	return nil
}

func aesCBC(key, iv, block, _ []byte) error {
	c, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	if len(block)%c.BlockSize() != 0 {
		return errors.New("ssh: invalid encrypted private key length, not a multiple of the block size")
	}
	cipher.NewCBCDecrypter(c, iv).CryptBlocks(block, block)
	return nil
}

// aesGCM opens the block with the 16-byte tag OpenSSH writes after it.
func aesGCM(key, iv, block, tag []byte) error {
	c, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	g, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}
	sealed := append(append([]byte(nil), block...), tag...)
	if _, err := g.Open(block[:0], iv, sealed, nil); err != nil {
		return ErrWrongPassphrase
	}
	return nil
}

// chachaPoly is OpenSSH's chacha20-poly1305@openssh.com with sequence
// number 0: the first 32 bytes of the key are the main key, block 0 of its
// stream keys Poly1305 over the ciphertext, and the text starts at block 1.
func chachaPoly(key, _, block, tag []byte) error {
	nonce := make([]byte, chacha20.NonceSize)
	c, err := chacha20.NewUnauthenticatedCipher(key[:32], nonce)
	if err != nil {
		return err
	}
	var polyKey [32]byte
	c.XORKeyStream(polyKey[:], polyKey[:])
	var mac [16]byte
	copy(mac[:], tag)
	if !poly1305.Verify(&mac, block, &polyKey) {
		return ErrWrongPassphrase
	}
	c.SetCounter(1)
	c.XORKeyStream(block, block)
	return nil
}
