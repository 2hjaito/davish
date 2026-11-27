package conn

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchLoopSendsNoKeepalivesWhenTold(t *testing.T) {
	var pings atomic.Int32
	closed := make(chan struct{})
	done := make(chan struct{})
	go func() {
		watchLoop(context.Background(), closed, false, 10*time.Millisecond, 1, func() bool {
			pings.Add(1)
			return false
		})
		close(done)
	}()
	// Many intervals pass; a failing ping would have ended the loop.
	select {
	case <-done:
		t.Fatal("the loop ended without the client closing")
	case <-time.After(150 * time.Millisecond):
	}
	if n := pings.Load(); n != 0 {
		t.Fatalf("%d keepalives sent through a control socket", n)
	}
	close(closed)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the loop did not end when the client closed")
	}
}

func TestWatchLoopEndsAfterMissedKeepalives(t *testing.T) {
	var pings atomic.Int32
	done := make(chan struct{})
	go func() {
		watchLoop(context.Background(), make(chan struct{}), true, 10*time.Millisecond, 3, func() bool {
			pings.Add(1)
			return false
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the loop never gave up on a server that does not answer")
	}
	if n := pings.Load(); n != 3 {
		t.Fatalf("gave up after %d keepalives, want 3", n)
	}
}

func TestWatchLoopForgivesAMissThatIsFollowedByAnAnswer(t *testing.T) {
	var pings atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		// Every other keepalive fails, never two in a row: with misses = 2 it keeps going.
		watchLoop(ctx, make(chan struct{}), true, 5*time.Millisecond, 2, func() bool {
			return pings.Add(1)%2 == 0
		})
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("a single miss ended the loop")
	case <-time.After(120 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the loop did not end when its context was canceled")
	}
}
