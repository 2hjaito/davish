// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file of golang.org/x/crypto.
//
// Copied from golang.org/x/crypto/ssh/internal/bcrypt_pbkdf, which Go does
// not let other modules import.

package keyfile

import (
	"crypto/sha512"
	"errors"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // bcrypt_pbkdf is built on Blowfish; OpenSSH key files use it
)

const pbkdfBlockSize = 32

// bcryptPBKDF derives keyLen bytes from the password and salt, as
// bcrypt_pbkdf(3) from OpenBSD, the KDF of encrypted OpenSSH keys.
func bcryptPBKDF(password, salt []byte, rounds, keyLen int) ([]byte, error) {
	if rounds < 1 {
		return nil, errors.New("bcrypt_pbkdf: number of rounds is too small")
	}
	if len(password) == 0 {
		return nil, errors.New("bcrypt_pbkdf: empty password")
	}
	if len(salt) == 0 || len(salt) > 1<<20 {
		return nil, errors.New("bcrypt_pbkdf: bad salt length")
	}
	if keyLen > 1024 {
		return nil, errors.New("bcrypt_pbkdf: keyLen is too large")
	}

	numBlocks := (keyLen + pbkdfBlockSize - 1) / pbkdfBlockSize
	key := make([]byte, numBlocks*pbkdfBlockSize)

	h := sha512.New()
	h.Write(password)
	shapass := h.Sum(nil)

	shasalt := make([]byte, 0, sha512.Size)
	cnt, tmp := make([]byte, 4), make([]byte, pbkdfBlockSize)
	for block := 1; block <= numBlocks; block++ {
		h.Reset()
		h.Write(salt)
		cnt[0] = byte(block >> 24)
		cnt[1] = byte(block >> 16)
		cnt[2] = byte(block >> 8)
		cnt[3] = byte(block)
		h.Write(cnt)
		if err := bcryptHash(tmp, shapass, h.Sum(shasalt)); err != nil {
			return nil, err
		}

		out := make([]byte, pbkdfBlockSize)
		copy(out, tmp)
		for i := 2; i <= rounds; i++ {
			h.Reset()
			h.Write(tmp)
			if err := bcryptHash(tmp, shapass, h.Sum(shasalt)); err != nil {
				return nil, err
			}
			for j := range out {
				out[j] ^= tmp[j]
			}
		}

		for i, v := range out {
			key[i*numBlocks+(block-1)] = v
		}
	}
	return key[:keyLen], nil
}

var bcryptMagic = []byte("OxychromaticBlowfishSwatDynamite")

func bcryptHash(out, shapass, shasalt []byte) error {
	if len(out) < len(bcryptMagic) {
		return errors.New("bcrypt_pbkdf: short output buffer")
	}
	c, err := blowfish.NewSaltedCipher(shapass, shasalt)
	if err != nil {
		return err
	}
	for range 64 {
		blowfish.ExpandKey(shasalt, c)
		blowfish.ExpandKey(shapass, c)
	}
	copy(out, bcryptMagic)
	for i := 0; i < 32; i += 8 {
		for range 64 {
			c.Encrypt(out[i:i+8], out[i:i+8])
		}
	}
	// Swap bytes due to different endianness.
	for i := 0; i < 32; i += 4 {
		out[i+3], out[i+2], out[i+1], out[i] = out[i], out[i+1], out[i+2], out[i+3] //nolint:gosec // out holds 32 bytes, checked above
	}
	return nil
}
