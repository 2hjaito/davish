package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestFormat(t *testing.T) {
	cause := errors.New("open /home/u/.ssh/id_ed25519: permission denied")
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{"app error", New(CmdNeedsRoot, "needs root"), CmdNeedsRoot},
		{"wrapped app error", fmt.Errorf("ctx: %w", Wrap(FSPermissionDenied, "denied", cause)), FSPermissionDenied},
		{"plain error", cause, Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Format(tt.err).(*Error)
			if !ok {
				t.Fatalf("Format returned %T", Format(tt.err))
			}
			if got.Code != tt.want {
				t.Errorf("code = %s, want %s", got.Code, tt.want)
			}
		})
	}
}

func TestWrapKeepsCause(t *testing.T) {
	cause := errors.New("boom")
	if err := Wrap(Internal, "x", cause); !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}
