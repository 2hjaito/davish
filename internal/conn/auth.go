package conn

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"davish/internal/apperr"
	"davish/internal/store"
)

// authMethods builds the ssh auth for srv. The returned cleanup releases the
// agent connection once the handshake is over. Errors never include the key
// path or a secret.
func (m *Manager) authMethods(srv store.Server) ([]ssh.AuthMethod, func(), error) {
	noop := func() {}
	switch srv.Auth {
	case store.AuthAgent:
		// "Automatic", like the ssh command: agent keys first, then the
		// default key files in ~/.ssh. Most users never load the agent.
		signers, release := m.agentSigners()
		signers = append(signers, m.defaultKeySigners(srv)...)
		if len(signers) == 0 {
			release()
			return nil, noop, apperr.New(apperr.ConnAgentUnavailable, "no SSH agent keys and no default key in ~/.ssh")
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signers...)}, release, nil

	case store.AuthKey:
		signer, err := m.loadKey(srv)
		if err != nil {
			return nil, noop, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, noop, nil

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

// agentSigners returns the keys loaded in the SSH agent, if one is running.
func (m *Manager) agentSigners() ([]ssh.Signer, func()) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, func() {}
	}
	c, err := net.Dial("unix", sock) //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
	if err != nil {
		return nil, func() {}
	}
	release := func() { c.Close() } //nolint:errcheck,gosec // best effort
	signers, err := agent.NewClient(c).Signers()
	if err != nil {
		return nil, release
	}
	return signers, release
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
	data, err := os.ReadFile(expandHome(srv.KeyPath))
	if err != nil {
		return nil, apperr.Wrap(apperr.ConnKeyUnreadable, "cannot read the private key", err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		pass, perr := m.secrets.Get(srv.ID, store.SecretPassphrase)
		if errors.Is(perr, store.ErrNoSecret) {
			return nil, apperr.New(apperr.ConnSecretMissing, "the key needs a passphrase")
		}
		if perr != nil {
			return nil, perr
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(pass))
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.ConnKeyUnreadable, "cannot parse the private key", err)
	}
	return signer, nil
}

// expandHome turns a leading ~ into the user's home directory.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
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
	var signers []ssh.Signer
	switch srv.Auth {
	case store.AuthAgent:
		agentSigners, release := m.agentSigners()
		defer release()
		signers = append(agentSigners, m.defaultKeySigners(srv)...)
	case store.AuthKey:
		if s, err := m.loadKey(srv); err == nil {
			signers = append(signers, s)
		}
	}
	fps := make([]string, 0, len(signers))
	for _, s := range signers {
		fps = append(fps, ssh.FingerprintSHA256(s.PublicKey()))
	}
	return fps, nil
}
