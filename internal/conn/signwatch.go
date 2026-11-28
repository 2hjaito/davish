package conn

import (
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// agentSignWait is the longest the handshake waits while the SSH agent is
// signing the login. An agent such as Bitwarden's asks the person to confirm
// using the key before it signs, and a person needs more than the seconds a
// silent server is given: with the plain handshake budget Davish closed the
// connection under a prompt nobody had answered yet, and logged nothing
// (#168). DAVISH_AGENT_WAIT raises it further.
var agentSignWait = 60 * time.Second

// signCheckEvery is how often the guard looks again while the agent signs.
const signCheckEvery = 250 * time.Millisecond

// signWatch tells a handshake's clock that the SSH agent is signing, which
// is the person's time and not the server's.
type signWatch struct {
	base    time.Duration
	pending atomic.Int32
	until   atomic.Int64 // unix nano: when the handshake runs out while nothing signs
	expired atomic.Bool  // the cap ran out while the agent was still signing
	failed  atomic.Bool  // the agent answered a signing request with an error
}

func (w *signWatch) begin() {
	if w != nil {
		w.pending.Add(1)
	}
}

// end gives the server a fresh base budget to answer what was signed.
func (w *signWatch) end() {
	if w != nil {
		w.pending.Add(-1)
		w.until.Store(time.Now().Add(w.base).UnixNano())
	}
}

// timedOut reports whether the handshake was ended because the agent had
// not signed within the cap.
func (w *signWatch) timedOut() bool { return w != nil && w.expired.Load() }

// signFailed reports whether the agent was asked to sign and did not: it
// refused, or it closed the connection while waiting for the person.
func (w *signWatch) signFailed() bool { return w != nil && w.failed.Load() }

// guard calls closeConn when the handshake takes too long. The budget is
// base, for dialing, the key exchange and a server that says nothing. While
// the agent is signing the clock waits for the person, up to limit from the
// start; when it has signed, the server gets base again to answer. The
// returned function stops the guard, like Timer.Stop.
func (w *signWatch) guard(base, limit time.Duration, closeConn func()) func() bool {
	start := time.Now()
	w.base = base
	w.until.Store(start.Add(base).UnixNano())
	var timer *time.Timer
	onTimer := func() {
		now := time.Now()
		if w.pending.Load() > 0 {
			if now.Sub(start) < limit {
				timer.Reset(signCheckEvery)
				return
			}
			w.expired.Store(true)
		} else if left := time.Unix(0, w.until.Load()).Sub(now); left > 0 {
			timer.Reset(left)
			return
		}
		closeConn()
	}
	timer = time.AfterFunc(base, onTimer)
	return timer.Stop
}

// loggedSigner is a key held by the SSH agent, with its signing logged: how
// long it took and how it ended, since that is the step an agent that asks
// the person for confirmation blocks on, and without these lines a support
// log stops at "ssh agent answered" (#168). It is an ssh.AlgorithmSigner and
// nothing more, as the agent's own signers are, so the signature algorithm
// is chosen as before (an RSA key keeps rsa-sha2-256 and -512).
type loggedSigner struct {
	ssh.AlgorithmSigner
	log   *slog.Logger
	watch *signWatch
}

func (s loggedSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	return s.sign(func() (*ssh.Signature, error) { return s.AlgorithmSigner.Sign(rand, data) })
}

func (s loggedSigner) SignWithAlgorithm(rand io.Reader, data []byte, algorithm string) (*ssh.Signature, error) {
	return s.sign(func() (*ssh.Signature, error) {
		return s.AlgorithmSigner.SignWithAlgorithm(rand, data, algorithm)
	})
}

func (s loggedSigner) sign(do func() (*ssh.Signature, error)) (*ssh.Signature, error) {
	s.watch.begin()
	defer s.watch.end()
	key := ssh.FingerprintSHA256(s.PublicKey())
	if s.log != nil {
		s.log.Info("ssh agent is signing the login; it may be waiting for you to confirm the key", "key", key)
	}
	start := time.Now()
	sig, err := do()
	took := time.Since(start).Round(time.Millisecond).String()
	if err != nil && s.watch != nil {
		s.watch.failed.Store(true)
	}
	if s.log != nil {
		if err != nil {
			s.log.Warn("ssh agent did not sign", "key", key, "took", took, "err", err)
		} else {
			s.log.Info("ssh agent signed", "key", key, "took", took)
		}
	}
	return sig, err
}
