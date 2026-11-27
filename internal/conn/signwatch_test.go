package conn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// closedWithin reports whether ch is closed within d.
func closedWithin(ch <-chan struct{}, d time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(d):
		return false
	}
}

func TestGuardClosesAfterTheBaseWhenNothingSigns(t *testing.T) {
	var w signWatch
	closed := make(chan struct{})
	w.guard(80*time.Millisecond, 5*time.Second, func() { close(closed) })
	if !closedWithin(closed, time.Second) {
		t.Fatal("the guard never closed a handshake that nobody was signing")
	}
	if w.timedOut() {
		t.Error("timed out is for a signing that ran out")
	}
}

// The agent is the person's time: the guard does not close while it signs,
// and gives the server a fresh base to answer once it has signed.
func TestGuardWaitsForTheAgentToSign(t *testing.T) {
	var w signWatch
	closed := make(chan struct{})
	w.guard(100*time.Millisecond, 10*time.Second, func() { close(closed) })
	w.begin()
	if closedWithin(closed, 600*time.Millisecond) {
		t.Fatal("closed while the agent was signing")
	}
	w.end()
	if closedWithin(closed, 40*time.Millisecond) {
		t.Fatal("closed the moment the agent had signed: the server got no time to answer")
	}
	if !closedWithin(closed, time.Second) {
		t.Fatal("never closed after the agent had signed")
	}
	if w.timedOut() {
		t.Error("the agent did sign")
	}
}

func TestGuardGivesUpOnAnAgentThatNeverSigns(t *testing.T) {
	var w signWatch
	closed := make(chan struct{})
	w.guard(50*time.Millisecond, 500*time.Millisecond, func() { close(closed) })
	w.begin() // and never ends
	if !closedWithin(closed, 3*time.Second) {
		t.Fatal("waited for ever on an agent that never signs")
	}
	if !w.timedOut() {
		t.Error("the cap ran out while signing, which timedOut must say")
	}
}

func TestGuardStops(t *testing.T) {
	var w signWatch
	closed := make(chan struct{})
	stop := w.guard(50*time.Millisecond, time.Second, func() { close(closed) })
	stop()
	if closedWithin(closed, 300*time.Millisecond) {
		t.Fatal("a stopped guard closed the connection")
	}
}

// A nil watch (an agent asked only for its keys) is not an error.
func TestNilSignWatch(t *testing.T) {
	var w *signWatch
	w.begin()
	w.end()
	if w.timedOut() {
		t.Error("nil watch timed out")
	}
}

// loggedSigner must not change how an agent's RSA key signs: wrapping it as
// a plain ssh.Signer would drop it to ssh-rsa (SHA-1), which servers refuse.
func TestLoggedSignerKeepsTheSignatureAlgorithm(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ring := agent.NewKeyring()
	if err := ring.Add(agent.AddedKey{PrivateKey: key}); err != nil {
		t.Fatal(err)
	}
	signers, err := ring.Signers()
	if err != nil {
		t.Fatal(err)
	}
	wrapped := (&Manager{}).logSigning(signers, nil)
	if _, ok := wrapped[0].(ssh.AlgorithmSigner); !ok {
		t.Fatal("the wrapped signer is not an AlgorithmSigner")
	}
	if _, ok := wrapped[0].(ssh.MultiAlgorithmSigner); ok {
		t.Fatal("the wrapped signer claims algorithms the agent's own does not")
	}
	data := []byte("login")
	for _, algo := range []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512} {
		sig, err := wrapped[0].(ssh.AlgorithmSigner).SignWithAlgorithm(rand.Reader, data, algo)
		if err != nil {
			t.Fatalf("%s: %v", algo, err)
		}
		if sig.Format != algo {
			t.Errorf("signed as %q; want %q", sig.Format, algo)
		}
		if err := wrapped[0].PublicKey().Verify(data, sig); err != nil {
			t.Errorf("%s: the signature does not verify: %v", algo, err)
		}
	}
}

type failingSigner struct{ ssh.Signer }

func (failingSigner) Sign(io.Reader, []byte) (*ssh.Signature, error) {
	return nil, errors.New("agent refused the operation")
}

func (f failingSigner) SignWithAlgorithm(io.Reader, []byte, string) (*ssh.Signature, error) {
	return f.Sign(nil, nil)
}

func TestLoggedSignerLogsHowSigningEnded(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	good, _ := ssh.NewSignerFromKey(priv)
	var buf bytes.Buffer
	m := &Manager{log: slog.New(slog.NewTextHandler(&buf, nil))}

	ok := m.logSigning([]ssh.Signer{good}, nil)[0]
	if _, err := ok.Sign(rand.Reader, []byte("x")); err != nil {
		t.Fatal(err)
	}
	bad := m.logSigning([]ssh.Signer{failingSigner{good}}, nil)[0]
	if _, err := bad.Sign(rand.Reader, []byte("x")); err == nil {
		t.Fatal("the failing signer signed")
	}
	log := buf.String()
	for _, want := range []string{"ssh agent is signing the login", "confirm the key", "ssh agent signed", "ssh agent did not sign", "agent refused the operation", "key=SHA256:", "took="} {
		if !strings.Contains(log, want) {
			t.Errorf("the log lacks %q:\n%s", want, log)
		}
	}
	// A fingerprint is public; nothing of the key itself may reach the log.
	if strings.Contains(log, string(priv)) {
		t.Error("key material in the log")
	}
}

// slowSigner is an agent that asks the person first: it signs after delay.
type slowSigner struct {
	ssh.Signer
	delay time.Duration
}

func (s slowSigner) Sign(r io.Reader, data []byte) (*ssh.Signature, error) {
	time.Sleep(s.delay)
	return s.Signer.Sign(r, data)
}

func (s slowSigner) SignWithAlgorithm(r io.Reader, data []byte, algo string) (*ssh.Signature, error) {
	time.Sleep(s.delay)
	if as, ok := s.Signer.(ssh.AlgorithmSigner); ok {
		return as.SignWithAlgorithm(r, data, algo)
	}
	return s.Signer.Sign(r, data)
}

// handshake runs a real SSH handshake in this process: a server that accepts
// the key, and a client whose agent takes signDelay to sign, under the guard.
func handshake(t *testing.T, signDelay, base, limit time.Duration) (*signWatch, error) {
	t.Helper()
	_, userPriv, _ := ed25519.GenerateKey(rand.Reader)
	userSigner, _ := ssh.NewSignerFromKey(userPriv)
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)

	srvCfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if bytes.Equal(k.Marshal(), userSigner.PublicKey().Marshal()) {
			return nil, nil
		}
		return nil, errors.New("unknown key")
	}}
	srvCfg.AddHostKey(hostSigner)
	// A real loopback connection: net.Pipe has no buffer, and both ends of an
	// SSH handshake write their version before they read.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() }) //nolint:errcheck,gosec // test cleanup
	go func() {
		srvEnd, err := ln.Accept()
		if err != nil {
			return
		}
		if c, chans, reqs, err := ssh.NewServerConn(srvEnd, srvCfg); err == nil {
			go ssh.DiscardRequests(reqs)
			go func() {
				for range chans {
				}
			}()
			_ = c
		}
	}()
	cliEnd, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	w := &signWatch{}
	signers := (&Manager{log: slog.New(slog.NewTextHandler(io.Discard, nil))}).
		logSigning([]ssh.Signer{slowSigner{userSigner, signDelay}}, w)
	cfg := &ssh.ClientConfig{
		User:            "u",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signers...)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // an in-process test server
		Timeout:         5 * time.Second,
	}
	stop := w.guard(base, limit, func() { cliEnd.Close() }) //nolint:errcheck,gosec // aborting
	c, _, _, err := ssh.NewClientConn(cliEnd, "test", cfg)
	stop()
	if err == nil {
		c.Close() //nolint:errcheck,gosec // test cleanup
	}
	return w, err
}

// The bug of #168 end to end: an agent that takes longer than the handshake
// budget (the person has to confirm the key) used to be cut off.
func TestHandshakeWaitsForASlowAgent(t *testing.T) {
	// Before: the budget is the cap, so the agent is cut off while signing.
	w, err := handshake(t, 600*time.Millisecond, 150*time.Millisecond, 150*time.Millisecond)
	if err == nil {
		t.Fatal("expected the old budget to cut the slow agent off")
	}
	if !w.timedOut() {
		t.Error("the cut-off during signing must be reported as such")
	}

	// After: the clock waits while the agent signs, and the login succeeds.
	w, err = handshake(t, 600*time.Millisecond, 150*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("a slow agent was cut off: %v", err)
	}
	if w.timedOut() {
		t.Error("no time out expected")
	}
}
