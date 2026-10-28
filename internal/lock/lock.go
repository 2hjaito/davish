// Package lock keeps Davish locked behind Touch ID or a password of its
// own while the user is away (docs/adr/0007-app-lock.md). It only guards
// the app's own UI and bindings: SSH sessions, log streams and transfers
// keep running while locked.
package lock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"davish/internal/apperr"
	"davish/internal/events"
	"davish/internal/store"
)

// Keychain accounts under the "davish" service; server secrets use
// "<serverID>:<kind>", so these never collide with them.
const (
	keychainService = "davish"
	hashAccount     = "app-lock:hash"
	failuresAccount = "app-lock:failures"
)

// MinPasswordLen is the shortest lock password accepted.
const MinPasswordLen = 6

// Wrong passwords allowed before each further one makes the next wait,
// starting at firstWait and doubling up to maxWait.
const (
	freeAttempts = 5
	firstWait    = 30 * time.Second
	maxWait      = time.Hour
)

// wakeTick is how often the clocks are compared to notice sleep.
const wakeTick = 5 * time.Second

// Keyring is where the password hash and the failure count live; Get
// returns store.ErrNoSecret for a missing entry.
type Keyring interface {
	Set(service, user, secret string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// Biometric is the OS fingerprint check: Touch ID on macOS, none elsewhere.
type Biometric interface {
	Available() bool
	Authenticate(ctx context.Context, reason string) error
}

// Emitter sends lock:state to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// State is the lock:state payload and what LockState returns.
type State struct {
	// Enabled is true once a lock password is set.
	Enabled bool `json:"enabled"`
	Locked  bool `json:"locked"`
	// Biometric tells whether Touch ID can unlock.
	Biometric bool `json:"biometric"`
	// RetryAt is when a password may be tried again, Unix ms; 0 means now.
	RetryAt int64 `json:"retryAt"`
}

// Options are the parts tests replace.
type Options struct {
	// Now is the clock for attempt delays; time.Now when nil.
	Now func() time.Time
}

// Service owns the lock state.
type Service struct {
	kr   Keyring
	bio  Biometric
	emit Emitter
	log  *slog.Logger
	// wipe removes every server secret Davish stored; Reset runs it first.
	wipe func() error
	now  func() time.Time

	// mu guards the fields below.
	mu       sync.Mutex
	enabled  bool
	locked   bool
	failures int
	retryAt  time.Time

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New reads whether a lock password is set; if so the app starts locked.
func New(kr Keyring, bio Biometric, emit Emitter, log *slog.Logger, wipe func() error, opts Options) *Service {
	s := &Service{kr: kr, bio: bio, emit: emit, log: log, wipe: wipe, now: opts.Now}
	if s.now == nil {
		s.now = time.Now
	}
	_, err := kr.Get(keychainService, hashAccount)
	switch {
	case err == nil:
		s.enabled, s.locked = true, true
	case !errors.Is(err, store.ErrNoSecret):
		// A password can only have been set through a working keychain.
		log.Warn("cannot read the lock password", "op", "lock", "err", err)
	}
	if v, err := kr.Get(keychainService, failuresAccount); err == nil {
		s.failures, s.retryAt = parseFailures(v)
	}
	return s
}

// Watch locks the app whenever the machine wakes from sleep, until Close.
func (s *Service) Watch() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		watchSleep(ctx, wakeTick, s.Lock)
	}()
}

// Close stops Watch.
func (s *Service) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// State returns the current state.
func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateLocked()
}

func (s *Service) stateLocked() State {
	st := State{Enabled: s.enabled, Locked: s.locked, Biometric: s.enabled && s.bio.Available()}
	if s.retryAt.After(s.now()) {
		st.RetryAt = s.retryAt.UnixMilli()
	}
	return st
}

// Check is the gate in front of every binding that reaches a server or a
// secret: it fails with APP_LOCKED while locked.
func (s *Service) Check() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return apperr.New(apperr.AppLocked, "Davish is locked")
	}
	return nil
}

// Lock locks the app if a password is set.
func (s *Service) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled || s.locked {
		return
	}
	s.locked = true
	s.publish()
}

// Unlock opens the app with the lock password.
func (s *Service) Unlock(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.locked {
		return nil
	}
	if err := s.checkPassword(password); err != nil {
		return err
	}
	s.locked = false
	s.publish()
	return nil
}

// UnlockBiometric opens the app with Touch ID. The prompt is shown by the
// OS and may take as long as the user does, so mu is not held meanwhile.
func (s *Service) UnlockBiometric(ctx context.Context, reason string) error {
	if !s.State().Biometric {
		return apperr.New(apperr.InvalidInput, "no biometric unlock on this machine")
	}
	if err := s.bio.Authenticate(ctx, reason); err != nil {
		return apperr.Wrap(apperr.LockBiometricFailed, "biometric check did not pass", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearFailures()
	if s.locked {
		s.locked = false
		s.publish()
	}
	return nil
}

// SetPassword sets the lock password, or changes it given the current one.
// It turns the lock on but does not lock right away.
func (s *Service) SetPassword(current, next string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return apperr.New(apperr.AppLocked, "Davish is locked")
	}
	if len([]rune(next)) < MinPasswordLen {
		return apperr.New(apperr.InvalidInput, "lock password too short")
	}
	if s.enabled {
		if err := s.checkPassword(current); err != nil {
			return err
		}
	}
	hash, err := hashPassword(next)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "cannot hash the lock password", err)
	}
	if err := s.kr.Set(keychainService, hashAccount, hash); err != nil {
		return apperr.Wrap(apperr.KeychainUnavailable, "cannot save the lock password", err)
	}
	s.enabled = true
	s.publish()
	return nil
}

// Disable turns the lock off given the current password.
func (s *Service) Disable(current string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return apperr.New(apperr.AppLocked, "Davish is locked")
	}
	if !s.enabled {
		return nil
	}
	if err := s.checkPassword(current); err != nil {
		return err
	}
	if err := s.forget(); err != nil {
		return err
	}
	s.publish()
	return nil
}

// Reset is "forgot the password": it removes every server secret Davish
// stored, then the lock password, and unlocks. Whoever resets gets into
// Davish but not into a server without its password.
func (s *Service) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.wipe(); err != nil {
		return err
	}
	if err := s.forget(); err != nil {
		return err
	}
	s.locked = false
	s.publish()
	s.log.Info("lock reset, server secrets removed", "op", "lock")
	return nil
}

// forget removes the password and the failure count; mu must be held.
func (s *Service) forget() error {
	if err := s.kr.Delete(keychainService, hashAccount); err != nil && !errors.Is(err, store.ErrNoSecret) {
		return apperr.Wrap(apperr.KeychainUnavailable, "cannot remove the lock password", err)
	}
	s.enabled = false
	s.clearFailures()
	return nil
}

// checkPassword verifies password, counting failures and making them wait
// after freeAttempts; mu must be held.
func (s *Service) checkPassword(password string) error {
	now := s.now()
	if s.retryAt.After(now) {
		return apperr.New(apperr.LockTooManyAttempts, "too many wrong passwords, wait before trying again")
	}
	hash, err := s.kr.Get(keychainService, hashAccount)
	if err != nil {
		return apperr.Wrap(apperr.KeychainUnavailable, "cannot read the lock password", err)
	}
	ok, err := verifyPassword(password, hash)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "cannot check the lock password", err)
	}
	if ok {
		s.clearFailures()
		return nil
	}
	s.failures++
	if s.failures >= freeAttempts {
		s.retryAt = now.Add(waitAfter(s.failures))
	}
	s.saveFailures()
	s.publish()
	s.log.Info("wrong lock password", "op", "lock", "failures", s.failures)
	return apperr.New(apperr.LockWrongPassword, "wrong lock password")
}

// waitAfter is the delay after the given number of wrong passwords.
func waitAfter(failures int) time.Duration {
	d := firstWait
	for i := freeAttempts; i < failures && d < maxWait; i++ {
		d *= 2
	}
	return min(d, maxWait)
}

func (s *Service) clearFailures() {
	if s.failures == 0 && s.retryAt.IsZero() {
		return
	}
	s.failures, s.retryAt = 0, time.Time{}
	if err := s.kr.Delete(keychainService, failuresAccount); err != nil && !errors.Is(err, store.ErrNoSecret) {
		s.log.Warn("cannot clear lock failures", "op", "lock", "err", err)
	}
}

// saveFailures keeps the count across restarts, so quitting does not
// reset the wait.
func (s *Service) saveFailures() {
	v := fmt.Sprintf("%d:%d", s.failures, s.retryAt.UnixMilli())
	if err := s.kr.Set(keychainService, failuresAccount, v); err != nil {
		s.log.Warn("cannot save lock failures", "op", "lock", "err", err)
	}
}

func parseFailures(v string) (int, time.Time) {
	n, at, ok := strings.Cut(v, ":")
	if !ok {
		return 0, time.Time{}
	}
	count, err1 := strconv.Atoi(n)
	ms, err2 := strconv.ParseInt(at, 10, 64)
	if err1 != nil || err2 != nil || count < 0 {
		return 0, time.Time{}
	}
	var retry time.Time
	if ms > 0 {
		retry = time.UnixMilli(ms)
	}
	return count, retry
}

// publish sends the state; mu must be held.
func (s *Service) publish() {
	s.emit.Emit(events.LockState, s.stateLocked())
}
