package conn

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/apperr"
	"davish/internal/keyfile"
	"davish/internal/store"
)

// authMethods builds the ssh auth for srv. The returned cleanup releases the
// agent connection once the handshake is over. Errors never include the key
// path or a secret.
func (m *Manager) authMethods(srv store.Server, w *signWatch) ([]ssh.AuthMethod, func(), error) {
	noop := func() {}
	switch srv.Auth {
	case store.AuthAgent:
		// "Automatic", like the ssh command: agent keys first, then the
		// default key files in ~/.ssh. Most users never load the agent.
		signers, release := m.agentSigners(w)
		signers = appendUniqueSigners(signers, m.defaultKeySigners(srv))
		if len(signers) == 0 {
			release()
			return nil, noop, apperr.New(apperr.ConnAgentUnavailable, "no SSH agent keys and no default key in ~/.ssh")
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signers...)}, release, nil

	case store.AuthKey:
		signer, err := m.loadKey(srv)
		if err == nil {
			return []ssh.AuthMethod{ssh.PublicKeys(signer)}, noop, nil
		}
		// Like OpenSSH's `ssh -i`: if the file needs a passphrase or is a .pub
		// file (for example an imported IdentityFile or a key held only by an
		// agent), use the matching key from the SSH agent when loaded.
		var ae *apperr.Error
		if errors.As(err, &ae) && (ae.Code == apperr.ConnSecretMissing || ae.Code == apperr.ConnPassphraseWrong || ae.Code == apperr.ConnKeyUnreadable) {
			if agentSigner, release, ok := m.agentSignerForKey(srv.KeyPath, w); ok {
				return []ssh.AuthMethod{ssh.PublicKeys(agentSigner)}, release, nil
			}
		}
		return nil, noop, err

	case store.AuthPassword:
		pw, err := m.secrets.Get(srv.ID, store.SecretPassword)
		if errors.Is(err, store.ErrNoSecret) {
			return nil, noop, apperr.New(apperr.ConnSecretMissing, "no password saved")
		}
		if err != nil {
			return nil, noop, err
		}
		// Many servers only offer keyboard-interactive for passwords.
		answer := func(_, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = pw
			}
			return answers, nil
		}
		return []ssh.AuthMethod{ssh.Password(pw), ssh.KeyboardInteractive(answer)}, noop, nil
	}
	return nil, noop, apperr.New(apperr.InvalidInput, "unknown auth method")
}

// agentDial connects to the user's SSH agent; tests replace it.
var agentDial = dialAgent

// agentTimeout bounds asking the agent for its keys. An agent that never
// answers (a pipe nobody serves, an agent waiting on a locked vault) must
// not leave the connection stuck before it even dials (#168).
var agentTimeout = 5 * time.Second

// agentWaitOverride returns DAVISH_AGENT_WAIT when set to a valid number of
// seconds (1 to 300).
func agentWaitOverride() (time.Duration, bool) {
	if n, err := strconv.Atoi(os.Getenv("DAVISH_AGENT_WAIT")); err == nil && n >= 1 && n <= 300 {
		return time.Duration(n) * time.Second, true
	}
	return 0, false
}

// agentWait is how long to wait for the agent: agentTimeout, or DAVISH_AGENT_WAIT
// seconds (1 to 300) when that is set. An agent that waits for the person to
// unlock a vault or confirm a key, such as Bitwarden's, may need longer than
// agentTimeout, and the person can raise it without a new build (#168).
func agentWait() time.Duration {
	if d, ok := agentWaitOverride(); ok {
		return d
	}
	return agentTimeout
}

// agentSigners returns the keys loaded in the SSH agent, if one is running
// and answers in time.
func (m *Manager) agentSigners(w *signWatch) ([]ssh.Signer, func()) {
	c, err := agentDial()
	if err != nil {
		// Without this line a pipe that cannot be opened (busy, no right to it,
		// nobody listening) looks like no agent at all, and the log of a person
		// whose agent does not work says nothing (#168).
		if m.log != nil {
			m.log.Info("ssh agent not reachable", "err", err)
		}
		return nil, func() {}
	}
	// Closing may block while a read is pending on a Windows pipe, so it
	// never runs on the caller's goroutine.
	release := func() { go c.Close() } //nolint:errcheck // best effort
	type listed struct {
		signers []ssh.Signer
		err     error
	}
	done := make(chan listed, 1)
	start := time.Now()
	go func() {
		s, err := agent.NewClient(c).Signers()
		done <- listed{s, err}
	}()
	wait := agentWait()
	select {
	case l := <-done:
		if m.log != nil && l.err == nil {
			// How long the agent took says whether waiting longer would help (#168).
			m.log.Info("ssh agent answered", "keys", len(l.signers), "took", time.Since(start).Round(time.Millisecond).String())
		}
		if l.err != nil {
			if m.log != nil {
				m.log.Warn("ssh agent did not list its keys", "err", l.err)
			}
			return nil, release
		}
		if len(l.signers) == 0 {
			if m.log != nil {
				// Some agents (authentik) list nothing until the client sends
				// session-bind@openssh.com, which x/crypto/ssh cannot do (#240).
				m.log.Info("ssh agent is running but lists no keys")
			}
			release()
			return nil, func() {}
		}
		return m.logSigning(l.signers, w), release
	case <-time.After(wait):
		if m.log != nil {
			m.log.Warn("ssh agent did not answer in time; trying key files", "waited", wait.String())
		}
		release()
		return nil, func() {}
	}
}

// logSigning wraps the agent's keys so that signing is logged and the
// handshake's clock knows when the agent is signing. A signer that is not an
// ssh.AlgorithmSigner is left as it is: wrapping it would change which
// signature algorithm is used.
func (m *Manager) logSigning(signers []ssh.Signer, w *signWatch) []ssh.Signer {
	out := make([]ssh.Signer, len(signers))
	for i, s := range signers {
		if as, ok := s.(ssh.AlgorithmSigner); ok {
			out[i] = loggedSigner{AlgorithmSigner: as, log: m.log, watch: w}
		} else {
			out[i] = s
		}
	}
	return out
}

// defaultKeyFiles are tried in the same order as OpenSSH.
var defaultKeyFiles = []string{"id_ed25519", "id_ecdsa", "id_rsa"}

// defaultKeySigners loads the default keys in ~/.ssh. An encrypted key is
// used only when its passphrase is saved for this server; others are skipped.
func (m *Manager) defaultKeySigners(srv store.Server) []ssh.Signer {
	var signers []ssh.Signer
	for _, name := range defaultKeyFiles {
		s := srv
		s.KeyPath = filepath.Join("~", ".ssh", name)
		signer, err := m.loadKey(s)
		if err == nil {
			signers = append(signers, signer)
		}
	}
	return signers
}

func (m *Manager) loadKey(srv store.Server) (ssh.Signer, error) {
	p := expandHome(srv.KeyPath)
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		// Reading a folder fails with "Incorrect function" on Windows,
		// which tells the user nothing (#168).
		return nil, apperr.New(apperr.ConnKeyUnreadable, "the key path is a folder; choose the private key file in it, such as id_ed25519")
	}
	data, err := os.ReadFile(p) //nolint:gosec // the key file the user chose
	if err != nil {
		return nil, apperr.Wrap(apperr.ConnKeyUnreadable, "cannot read the private key", err)
	}
	signer, err := keyfile.Parse(data, nil)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		pass, perr := m.secrets.Get(srv.ID, store.SecretPassphrase)
		if errors.Is(perr, store.ErrNoSecret) {
			return nil, apperr.New(apperr.ConnSecretMissing, "the key needs a passphrase")
		}
		if perr != nil {
			return nil, perr
		}
		signer, err = keyfile.Parse(data, []byte(pass))
	}
	if errors.Is(err, keyfile.ErrPPK) {
		return nil, apperr.New(apperr.ConnKeyPPK, "this PuTTY .ppk key is a kind Davish cannot read; in PuTTYgen use Conversions > Export OpenSSH key and choose that file")
	}
	if errors.Is(err, keyfile.ErrWrongPassphrase) {
		return nil, apperr.Wrap(apperr.ConnPassphraseWrong, "the saved passphrase does not open the key", err)
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.ConnKeyUnreadable, "cannot parse the private key", err)
	}
	return signer, nil
}

// expandHome turns a leading ~ into the user's home directory.
func expandHome(path string) string {
	// "~\" on Windows too: filepath.Join("~", ".ssh", …) writes it, and
	// so do Windows users (#168).
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// appendUniqueSigners appends extra signers whose public key is not already
// present in base, so a key loaded in the agent and also in ~/.ssh is not
// offered twice against sshd's MaxAuthTries.
func appendUniqueSigners(base, extra []ssh.Signer) []ssh.Signer {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, s := range base {
		seen[string(s.PublicKey().Marshal())] = struct{}{}
	}
	for _, s := range extra {
		k := string(s.PublicKey().Marshal())
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		base = append(base, s)
	}
	return base
}

// agentSignerForKey looks up keyPath's public key in the SSH agent. It checks
// keyPath itself (when pointing at a .pub file) or keyPath+".pub", matching
// OpenSSH's IdentityFile behavior.
func (m *Manager) agentSignerForKey(keyPath string, w *signWatch) (ssh.Signer, func(), bool) {
	pub := readPublicKey(expandHome(keyPath))
	if pub == nil {
		return nil, nil, false
	}
	want := pub.Marshal()
	signers, release := m.agentSigners(w)
	for _, s := range signers {
		if bytes.Equal(s.PublicKey().Marshal(), want) {
			return s, release, true
		}
	}
	release()
	return nil, nil, false
}

// readPublicKey extracts the SSH public key for path: from path itself (as an
// authorized public key or an encrypted OpenSSH private key's unencrypted
// header), or from path+".pub". Returns nil if none yields a public key.
func readPublicKey(path string) ssh.PublicKey {
	for _, candidate := range []string{path, path + ".pub"} {
		fi, err := os.Stat(candidate)
		if err != nil || fi.IsDir() || fi.Size() > 64<<10 {
			continue
		}
		data, err := os.ReadFile(candidate) //nolint:gosec // user-configured key path or its .pub sibling
		if err != nil {
			continue
		}
		if pub, _, _, _, err := ssh.ParseAuthorizedKey(data); err == nil {
			return pub
		}
		if candidate == path {
			_, perr := keyfile.Parse(data, nil)
			var missing *ssh.PassphraseMissingError
			if errors.As(perr, &missing) && missing.PublicKey != nil {
				return missing.PublicKey
			}
		}
	}
	return nil
}

// LoginKeys are the SHA256 fingerprints of every public key Davish may log
// in to the server with: the agent's, the default files and the chosen
// key. Removing one of them from the server's authorized_keys may lock
// Davish out, so the UI warns about those. Password logins have none.
func (m *Manager) LoginKeys(ctx context.Context, id string) ([]string, error) {
	srv, err := m.servers.Server(ctx, id)
	if err != nil {
		return nil, err
	}
	var pubs []ssh.PublicKey
	switch srv.Auth {
	case store.AuthAgent:
		agentSigners, release := m.agentSigners(nil)
		defer release()
		for _, s := range appendUniqueSigners(agentSigners, m.defaultKeySigners(srv)) {
			pubs = append(pubs, s.PublicKey())
		}
	case store.AuthKey:
		if s, err := m.loadKey(srv); err == nil {
			pubs = append(pubs, s.PublicKey())
		} else if pub := readPublicKey(expandHome(srv.KeyPath)); pub != nil {
			pubs = append(pubs, pub)
		}
	}
	fps := make([]string, 0, len(pubs))
	for _, p := range pubs {
		fps = append(fps, ssh.FingerprintSHA256(p))
	}
	return fps, nil
}
