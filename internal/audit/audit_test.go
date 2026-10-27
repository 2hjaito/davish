package audit

import (
	"testing"

	"davish/internal/events"
)

type recorder struct{ names []string }

func (r *recorder) Emit(name string, _ any) { r.names = append(r.names, name) }

func TestLogKeepsLatest(t *testing.T) {
	tests := []struct {
		name      string
		added     int
		wantLen   int
		wantFirst uint64
	}{
		{"empty", 0, 0, 0},
		{"under capacity", 3, 3, 1},
		{"exactly full", Capacity, Capacity, 1},
		{"wrapped", Capacity + 5, Capacity, 6},
		{"wrapped twice", 2*Capacity + 1, Capacity, Capacity + 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			l := New(rec)
			for range tt.added {
				l.Add(Entry{Command: "true"})
			}
			got := l.Entries()
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if len(rec.names) != tt.added {
				t.Errorf("emitted %d events, want %d", len(rec.names), tt.added)
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0].ID != tt.wantFirst {
				t.Errorf("first ID = %d, want %d", got[0].ID, tt.wantFirst)
			}
			for i := 1; i < len(got); i++ {
				if got[i].ID != got[i-1].ID+1 {
					t.Fatalf("entries out of order at %d: %d after %d", i, got[i].ID, got[i-1].ID)
				}
			}
			if rec.names[0] != events.AuditEntry {
				t.Errorf("event = %q, want %q", rec.names[0], events.AuditEntry)
			}
		})
	}
}
