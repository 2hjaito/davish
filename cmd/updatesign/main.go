// Command updatesign makes the ed25519 key the self-updater trusts and signs
// release checksums with it (docs/adr/0009-self-update.md).
//
//	updatesign gen <private key file>   prints the public key to embed
//	updatesign sign <file>              writes <file>.sig; the key is read
//	                                    from DAVISH_UPDATE_KEY (base64 seed)
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "updatesign:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: updatesign gen <key file> | sign <file>")
	}
	switch args[0] {
	case "gen":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		seed := base64.StdEncoding.EncodeToString(priv.Seed())
		// O_EXCL: never overwrite a key releases already depend on.
		f, err := os.OpenFile(args[1], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // the maintainer names the file
		if err != nil {
			return err
		}
		if _, err := f.WriteString(seed + "\n"); err != nil {
			f.Close() //nolint:errcheck,gosec // already failing
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Println(base64.StdEncoding.EncodeToString(pub))
		return nil
	case "sign":
		seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("DAVISH_UPDATE_KEY")))
		if err != nil || len(seed) != ed25519.SeedSize {
			return fmt.Errorf("DAVISH_UPDATE_KEY is not a base64 ed25519 seed")
		}
		data, err := os.ReadFile(args[1]) //nolint:gosec // the maintainer names the file
		if err != nil {
			return err
		}
		sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), data)
		return os.WriteFile(args[1]+".sig", []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644) //nolint:gosec // a signature is public
	}
	return fmt.Errorf("unknown command %q", args[0])
}
