package lock

import (
	"context"
	"time"
)

// sleepGap is how much longer than a tick must pass for the machine to
// count as having slept: long enough to ignore a busy CPU, short enough
// to catch closing the lid for a minute.
const sleepGap = 30 * time.Second

// slept reports whether the machine slept between two readings of
// time.Now. The monotonic clock stops during sleep on macOS and Linux
// while the wall clock does not; on Windows the monotonic clock keeps
// running, so a tick that came far too late gives it away instead.
func slept(prev, now time.Time, tick time.Duration) bool {
	mono := now.Sub(prev)
	wall := now.Round(0).Sub(prev.Round(0))
	return wall-mono > sleepGap || mono > tick+sleepGap
}

// watchSleep calls onWake after the machine wakes from sleep, until ctx ends.
func watchSleep(ctx context.Context, tick time.Duration, onWake func()) {
	t := time.NewTicker(tick)
	defer t.Stop()
	prev := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			if slept(prev, now, tick) {
				onWake()
			}
			prev = now
		}
	}
}
