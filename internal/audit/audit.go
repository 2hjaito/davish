// Package audit keeps an in-memory log of commands the app ran on servers,
// so the user can see exactly what Davish did (docs/ARCHITECTURE.md §3.7).
package audit

import (
	"sync"

	"davish/internal/events"
)

// Capacity is how many entries the log keeps; older ones are dropped.
const Capacity = 1000

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Entry is one command run on a server.
type Entry struct {
	ID       uint64 `json:"id"`
	Time     int64  `json:"time"` // unix milliseconds, when it started
	ServerID string `json:"serverID"`
	App      string `json:"app"`
	// Command is the command line as sent to the server. Commands are
	// built by shellx from names and paths, never from secrets.
	Command string `json:"command"`
	// Exit is the exit status, or -1 when the command did not finish
	// (connection lost, canceled).
	Exit  int   `json:"exit"`
	DurMs int64 `json:"durMs"`
	// Mutating marks a command run for an action that changes the server,
	// as opposed to a read or a poll (docs/adr/0022).
	Mutating bool `json:"mutating,omitempty"`
	// Input marks a command that was given content on stdin, such as a
	// file written with tee. The content is never recorded.
	Input bool `json:"input,omitempty"`
}

// Log is a ring buffer of the latest Capacity entries.
type Log struct {
	emit Emitter

	mu sync.Mutex
	// ring holds up to Capacity entries; next is where the next one goes
	// once it is full. Guarded by mu, as is seq.
	ring []Entry
	next int
	seq  uint64
}

// New creates an empty log.
func New(emit Emitter) *Log {
	return &Log{emit: emit}
}

// Add records e with the next ID, emits it as audit:entry and returns it.
func (l *Log) Add(e Entry) Entry {
	l.mu.Lock()
	l.seq++
	e.ID = l.seq
	if len(l.ring) < Capacity {
		l.ring = append(l.ring, e)
	} else {
		l.ring[l.next] = e
		l.next = (l.next + 1) % Capacity
	}
	l.mu.Unlock()
	l.emit.Emit(events.AuditEntry, e)
	return e
}

// Entries returns the kept entries, oldest first.
func (l *Log) Entries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, 0, len(l.ring))
	out = append(out, l.ring[l.next:]...)
	return append(out, l.ring[:l.next]...)
}

// Pick returns the kept entries with the given IDs, oldest first; IDs
// already dropped from the log are skipped.
func (l *Log) Pick(ids []uint64) []Entry {
	want := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []Entry
	for _, e := range l.Entries() {
		if want[e.ID] {
			out = append(out, e)
		}
	}
	return out
}
