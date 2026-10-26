package conn

import (
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
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, noop, apperr.New(apperr.ConnAgentUnavailable, "SSH_AUTH_SOCK is not set")
		}
		c, err := net.Dial("unix", sock) //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
		if err != nil {
			return nil, noop, apperr.Wrap(apperr.ConnAgentUnavailable, "cannot reach the SSH agent", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(c).Signers)}, func() { c.Close() }, nil //nolint:errcheck,gosec // best effort

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
