package store

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"davish/internal/apperr"
)

// SecretKind names what a secret unlocks.
type SecretKind string

// Secrets a server can have.
const (
	SecretPassword   SecretKind = "password"
	SecretPassphrase SecretKind = "passphrase"
)

var secretKinds = []SecretKind{SecretPassword, SecretPassphrase}

// keychainService groups every Davish entry in the OS keychain.
const keychainService = "davish"

// ErrNoSecret means the keychain holds no secret of that kind for the server.
var ErrNoSecret = errors.New("secret not found")

// Keyring is the OS keychain. OSKeyring is the real one; tests use a fake so
// they never touch the developer's keychain.
type Keyring interface {
	Set(service, user, secret string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// OSKeyring stores secrets in the macOS Keychain, Windows Credential
// Manager or the Secret Service on Linux.
type OSKeyring struct{}

// Set stores secret.
func (OSKeyring) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

// Get returns the stored secret or ErrNoSecret.
func (OSKeyring) Get(service, user string) (string, error) {
	s, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNoSecret
	}
	return s, err
}

// Delete removes the secret; a missing one is ErrNoSecret.
func (OSKeyring) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNoSecret
	}
	return err
}

// Secrets keeps server passwords and key passphrases in a Keyring under the
// key "davish:<serverID>:<kind>". Secret values never appear in errors.
type Secrets struct {
	kr Keyring
}

// NewSecrets wraps kr.
func NewSecrets(kr Keyring) *Secrets {
	return &Secrets{kr: kr}
}

func account(serverID string, kind SecretKind) string {
	return serverID + ":" + string(kind)
}

func checkKind(kind SecretKind) error {
	for _, k := range secretKinds {
		if k == kind {
			return nil
		}
	}
	return apperr.New(apperr.InvalidInput, "unknown secret kind")
}

// Set stores the secret, replacing any previous one.
func (s *Secrets) Set(serverID string, kind SecretKind, value string) error {
	if err := checkKind(kind); err != nil {
		return err
	}
	if value == "" {
		return apperr.New(apperr.InvalidInput, "secret must not be empty")
	}
	if err := s.kr.Set(keychainService, account(serverID, kind), value); err != nil {
		return apperr.Wrap(apperr.KeychainUnavailable, "cannot save to the keychain", err)
	}
	return nil
}

// Get returns the secret, or ErrNoSecret.
func (s *Secrets) Get(serverID string, kind SecretKind) (string, error) {
	if err := checkKind(kind); err != nil {
		return "", err
	}
	v, err := s.kr.Get(keychainService, account(serverID, kind))
	if errors.Is(err, ErrNoSecret) {
		return "", ErrNoSecret
	}
	if err != nil {
		return "", apperr.Wrap(apperr.KeychainUnavailable, "cannot read the keychain", err)
	}
	return v, nil
}

// Has reports whether a secret is stored, for the UI's hasSecret flag.
func (s *Secrets) Has(serverID string, kind SecretKind) (bool, error) {
	_, err := s.Get(serverID, kind)
	if errors.Is(err, ErrNoSecret) {
		return false, nil
	}
	return err == nil, err
}

// DeleteAll removes every kind of secret stored for the server.
func (s *Secrets) DeleteAll(serverID string) error {
	var errs []error
	for _, kind := range secretKinds {
		err := s.kr.Delete(keychainService, account(serverID, kind))
		if err != nil && !errors.Is(err, ErrNoSecret) {
			errs = append(errs, fmt.Errorf("delete %s: %w", kind, err))
		}
	}
	if len(errs) > 0 {
		return apperr.Wrap(apperr.KeychainUnavailable, "cannot remove secrets from the keychain", errors.Join(errs...))
	}
	return nil
}
