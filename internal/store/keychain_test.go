package store

import (
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
)

func TestSecretsRoundTrip(t *testing.T) {
	kr := newFakeKeyring()
	s := NewSecrets(kr)

	if has, err := s.Has("srv1", SecretPassword); err != nil || has {
		t.Fatalf("Has before Set = %v, %v; want false, nil", has, err)
	}
	if err := s.Set("srv1", SecretPassword, "pw"); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get("srv1", SecretPassword); err != nil || got != "pw" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if _, ok := kr.m["davish|srv1:password"]; !ok {
		t.Errorf("keychain keys = %v; want davish|srv1:password", kr.m)
	}
	if _, err := s.Get("srv1", SecretPassphrase); !errors.Is(err, ErrNoSecret) {
		t.Errorf("Get passphrase err = %v; want ErrNoSecret", err)
	}
}

func TestSecretsRejectBadInput(t *testing.T) {
	s := NewSecrets(newFakeKeyring())
	for name, err := range map[string]error{
		"unknown kind": s.Set("srv1", "token", "x"),
		"empty value":  s.Set("srv1", SecretPassword, ""),
	} {
		if codeOf(err) != apperr.InvalidInput {
			t.Errorf("%s: err = %v; want INVALID_INPUT", name, err)
		}
	}
}

func TestKeychainErrorsHideSecret(t *testing.T) {
	kr := newFakeKeyring()
	kr.err = errors.New("dbus: no secret service")
	s := NewSecrets(kr)

	err := s.Set("srv1", SecretPassword, "hunter2-very-secret")
	if codeOf(err) != apperr.KeychainUnavailable {
		t.Fatalf("err = %v; want KEYCHAIN_UNAVAILABLE", err)
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Error("error message contains the secret")
	}
	if _, err := s.Has("srv1", SecretPassword); codeOf(err) != apperr.KeychainUnavailable {
		t.Errorf("Has err = %v; want KEYCHAIN_UNAVAILABLE", err)
	}
}
