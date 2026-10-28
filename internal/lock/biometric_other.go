//go:build !darwin

package lock

import (
	"context"
	"errors"
)

// NewBiometric returns the OS fingerprint check. Windows Hello is left for
// later (ADR 0007), so off macOS there is none and the password unlocks.
func NewBiometric() Biometric { return noBiometric{} }

type noBiometric struct{}

func (noBiometric) Available() bool { return false }

func (noBiometric) Authenticate(context.Context, string) error {
	return errors.New("no biometric unlock on this OS")
}
