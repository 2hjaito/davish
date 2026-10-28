package lock

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/store"
)

type fakeKeyring struct {
	mu sync.Mutex
	m  map[string]string
}

func newFakeKeyring() *fakeKeyring { return &fakeKeyring{m: map[string]string{}} }

func (k *fakeKeyring) Set(service, user, secret string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.m[service+"/"+user] = secret
	return nil
}

func (k *fakeKeyring) Get(service, user string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	v, ok := k.m[service+"/"+user]
	if !ok {
		return "", store.ErrNoSecret
	}
	return v, nil
}

func (k *fakeKeyring) Delete(service, user string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, ok := k.m[service+"/"+user]; !ok {
		return store.ErrNoSecret
	}
	delete(k.m, service+"/"+user)
	return nil
}

type fakeBio struct {
	available bool
	pass      bool
}

func (b fakeBio) Available() bool { return b.available }

func (b fakeBio) Authenticate(context.Context, string) error {
	if !b.pass {
		return errors.New("cancelled")
	}
	return nil
}

type recorder struct {
	mu     sync.Mutex
	states []State
}

func (r *recorder) Emit(_ string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = append(r.states, payload.(State))
}

func (r *recorder) last() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.states[len(r.states)-1]
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newService(t *testing.T, kr *fakeKeyring, bio Biometric, wipe func() error) (*Service, *recorder, *clock) {
	t.Helper()
	rec := &recorder{}
	c := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	if wipe == nil {
		wipe = func() error { return nil }
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(kr, bio, rec, log, wipe, Options{Now: c.now}), rec, c
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestStartsUnlockedWithoutPassword(t *testing.T) {
	s, _, _ := newService(t, newFakeKeyring(), fakeBio{}, nil)
	if st := s.State(); st.Enabled || st.Locked {
		t.Fatalf("state = %+v", st)
	}
	s.Lock()
	if err := s.Check(); err != nil {
		t.Errorf("lock without a password: %v", err)
	}
}

func TestPasswordLocksAndUnlocks(t *testing.T) {
	s, rec, _ := newService(t, newFakeKeyring(), fakeBio{}, nil)
	if err := s.SetPassword("", "short"); code(err) != apperr.InvalidInput {
		t.Fatalf("short password: %v", err)
	}
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	if st := rec.last(); !st.Enabled || st.Locked {
		t.Fatalf("after set: %+v", st)
	}
	s.Lock()
	if err := s.Check(); code(err) != apperr.AppLocked {
		t.Fatalf("check while locked: %v", err)
	}
	if err := s.Unlock("wrong horse"); code(err) != apperr.LockWrongPassword {
		t.Fatalf("wrong password: %v", err)
	}
	if err := s.Unlock("correct horse"); err != nil {
		t.Fatal(err)
	}
	if err := s.Check(); err != nil || rec.last().Locked {
		t.Fatalf("after unlock: %v %+v", err, rec.last())
	}
}

func TestStartsLockedWhenPasswordIsSet(t *testing.T) {
	kr := newFakeKeyring()
	s, _, _ := newService(t, kr, fakeBio{}, nil)
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	again, _, _ := newService(t, kr, fakeBio{}, nil)
	if st := again.State(); !st.Enabled || !st.Locked {
		t.Fatalf("restart: %+v", st)
	}
}

func TestWrongPasswordsWaitLongerEachTime(t *testing.T) {
	kr := newFakeKeyring()
	s, _, c := newService(t, kr, fakeBio{}, nil)
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	s.Lock()
	for i := range freeAttempts {
		if err := s.Unlock("nope"); code(err) != apperr.LockWrongPassword {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	st := s.State()
	if want := c.t.Add(firstWait).UnixMilli(); st.RetryAt != want {
		t.Fatalf("retryAt = %d, want %d", st.RetryAt, want)
	}
	// Even the right password waits.
	if err := s.Unlock("correct horse"); code(err) != apperr.LockTooManyAttempts {
		t.Fatalf("during the wait: %v", err)
	}

	// The count survives a restart.
	restarted, _, rc := newService(t, kr, fakeBio{}, nil)
	rc.t = c.t
	if err := restarted.Unlock("correct horse"); code(err) != apperr.LockTooManyAttempts {
		t.Fatalf("after restart: %v", err)
	}

	c.t = c.t.Add(firstWait + time.Second)
	if err := s.Unlock("nope"); code(err) != apperr.LockWrongPassword {
		t.Fatal(err)
	}
	if want := c.t.Add(2 * firstWait).UnixMilli(); s.State().RetryAt != want {
		t.Fatalf("second wait: retryAt = %d, want %d", s.State().RetryAt, want)
	}
	c.t = c.t.Add(2*firstWait + time.Second)
	if err := s.Unlock("correct horse"); err != nil {
		t.Fatal(err)
	}
	if _, err := kr.Get(keychainService, failuresAccount); !errors.Is(err, store.ErrNoSecret) {
		t.Errorf("failures kept after unlocking: %v", err)
	}
}

func TestWaitAfterIsCapped(t *testing.T) {
	if got := waitAfter(freeAttempts); got != firstWait {
		t.Errorf("first wait = %v", got)
	}
	if got := waitAfter(freeAttempts + 50); got != maxWait {
		t.Errorf("long run = %v", got)
	}
}

func TestChangeAndDisableNeedCurrentPassword(t *testing.T) {
	s, _, _ := newService(t, newFakeKeyring(), fakeBio{}, nil)
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassword("guess", "battery staple"); code(err) != apperr.LockWrongPassword {
		t.Fatalf("change with a wrong current: %v", err)
	}
	if err := s.SetPassword("correct horse", "battery staple"); err != nil {
		t.Fatal(err)
	}
	if err := s.Disable("correct horse"); code(err) != apperr.LockWrongPassword {
		t.Fatalf("disable with the old password: %v", err)
	}
	if err := s.Disable("battery staple"); err != nil {
		t.Fatal(err)
	}
	if s.State().Enabled {
		t.Error("still enabled")
	}
}

func TestResetWipesSecretsFirst(t *testing.T) {
	kr := newFakeKeyring()
	failWipe := errors.New("keychain busy")
	wipeErr := failWipe
	wiped := 0
	s, _, _ := newService(t, kr, fakeBio{}, func() error { wiped++; return wipeErr })
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	s.Lock()
	if err := s.Reset(); !errors.Is(err, failWipe) {
		t.Fatalf("reset with a failing wipe: %v", err)
	}
	if st := s.State(); !st.Locked || !st.Enabled {
		t.Fatalf("a failed wipe must keep the lock: %+v", st)
	}
	wipeErr = nil
	if err := s.Reset(); err != nil {
		t.Fatal(err)
	}
	if st := s.State(); st.Locked || st.Enabled || wiped != 2 {
		t.Fatalf("after reset: %+v, wiped %d", st, wiped)
	}
	if _, err := kr.Get(keychainService, hashAccount); !errors.Is(err, store.ErrNoSecret) {
		t.Errorf("hash kept: %v", err)
	}
}

func TestBiometricUnlock(t *testing.T) {
	s, _, _ := newService(t, newFakeKeyring(), fakeBio{available: true, pass: false}, nil)
	if err := s.SetPassword("", "correct horse"); err != nil {
		t.Fatal(err)
	}
	if !s.State().Biometric {
		t.Fatal("biometric not offered")
	}
	s.Lock()
	if err := s.UnlockBiometric(context.Background(), "unlock"); code(err) != apperr.LockBiometricFailed {
		t.Fatalf("failed touch id: %v", err)
	}
	s.bio = fakeBio{available: true, pass: true}
	if err := s.UnlockBiometric(context.Background(), "unlock"); err != nil {
		t.Fatal(err)
	}
	if s.State().Locked {
		t.Error("still locked")
	}
}

func TestBiometricNeedsAPassword(t *testing.T) {
	s, _, _ := newService(t, newFakeKeyring(), fakeBio{available: true, pass: true}, nil)
	if s.State().Biometric {
		t.Error("touch id offered with no lock set")
	}
}

func TestHashRoundTrip(t *testing.T) {
	h, err := hashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "argon2id$v=19$m=65536,t=3,p=2$") || strings.Contains(h, "correct") {
		t.Fatalf("hash = %q", h)
	}
	for pw, want := range map[string]bool{"correct horse": true, "correct hors": false, "": false} {
		if ok, err := verifyPassword(pw, h); err != nil || ok != want {
			t.Errorf("verify(%q) = %v, %v", pw, ok, err)
		}
	}
	if _, err := verifyPassword("x", "md5$abc"); err == nil {
		t.Error("unknown format accepted")
	}
}

func TestSlept(t *testing.T) {
	prev := time.Now()
	tick := wakeTick
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"on time", prev.Add(tick), false},
		{"a little late", prev.Add(tick + 10*time.Second), false},
		// A tick that came minutes late: Windows, whose monotonic clock runs during sleep.
		{"tick far too late", prev.Add(tick + 2*time.Minute), true},
		// No monotonic reading: the wall clock jumped while the monotonic one stood still.
		{"wall clock jumped", time.Unix(0, prev.Add(10*time.Minute).UnixNano()), true},
	}
	for _, tc := range tests {
		if got := slept(prev, tc.now, tick); got != tc.want {
			t.Errorf("%s: slept = %v, want %v", tc.name, got, tc.want)
		}
	}
}
