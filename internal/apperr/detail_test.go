package apperr

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDetail(t *testing.T) {
	cause := errors.New("ssh: handshake failed: EOF")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"no cause", New(ConnNotConnected, "not connected"), ""},
		{"wrapped cause", Wrap(ConnUnreachable, "ssh handshake failed", cause), "ssh: handshake failed: EOF"},
		{"wrapped twice", fmt.Errorf("dial: %w", Wrap(ConnUnreachable, "x", cause)), "ssh: handshake failed: EOF"},
		{"unclassified", fmt.Errorf("load known_hosts: %w", cause), "load known_hosts: ssh: handshake failed: EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Detail(tt.err); got != tt.want {
				t.Errorf("Detail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHideHome(t *testing.T) {
	got := hideHome(`open C:\Users\kim\.ssh\id_ed25519 and C:/Users/kim/x`, `C:\Users\kim`)
	if strings.Contains(got, "kim") || !strings.Contains(got, `~\.ssh\id_ed25519`) || !strings.Contains(got, "~/x") {
		t.Errorf("hideHome = %q", got)
	}
}

func TestDetailIsBounded(t *testing.T) {
	got := Detail(errors.New(strings.Repeat("x", 5000)))
	if len(got) > maxDetail+len("…") {
		t.Errorf("len = %d", len(got))
	}
}
