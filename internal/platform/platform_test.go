package platform

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func probe(uname, home string, unameErr, homeErr error) Probe {
	return Probe{
		Uname: func(context.Context) (string, error) { return uname, unameErr },
		Home:  func(context.Context) (string, error) { return home, homeErr },
	}
}

func TestDetect(t *testing.T) {
	fail := errors.New("exit 1")
	tests := []struct {
		name string
		p    Probe
		want OS
	}{
		{"linux", probe("Linux\n", "/home/ann", nil, nil), Linux},
		{"darwin", probe("Darwin\n", "/Users/ann", nil, nil), Darwin},
		{"windows cmd", probe("", "/C:/Users/ann", fail, nil), Windows},
		{"windows lower-case drive", probe("", "/d:/home", fail, nil), Windows},
		{"windows drive root", probe("", "/C:", fail, nil), Windows},
		{"git for windows", probe("MINGW64_NT-10.0-20348\n", "/C:/Users/ann", nil, nil), Windows},
		{"cygwin", probe("CYGWIN_NT-10.0\n", "", nil, fail), Windows},
		{"freebsd with a posix home", probe("FreeBSD\n", "/home/ann", nil, nil), Unknown},
		{"nothing answers", probe("", "", fail, fail), Unknown},
		{"uname fails, posix home", probe("", "/home/ann", fail, nil), Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Detect(context.Background(), tt.p); got != tt.want {
				t.Fatalf("Detect = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectStopsAtTheDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	p := Probe{
		Uname: func(ctx context.Context) (string, error) { return "", ctx.Err() },
		Home: func(context.Context) (string, error) {
			called = true
			return "/C:/Users/ann", nil
		},
	}
	if got := Detect(ctx, p); got != Unknown {
		t.Fatalf("Detect = %q, want unknown", got)
	}
	if called {
		t.Fatal("Home was asked after the context ended")
	}
}

func TestPowerShell(t *testing.T) {
	cmd, stdin := PowerShell("Write-Output 'hi'")
	if stdin != "Write-Output 'hi'" {
		t.Fatalf("stdin = %q", stdin)
	}
	const prefix = "powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand "
	enc, ok := strings.CutPrefix(cmd, prefix)
	if !ok {
		t.Fatalf("command = %q", cmd)
	}
	if len(cmd) > 1000 {
		t.Fatalf("command is %d characters; it must stay a small bootstrap", len(cmd))
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || len(raw)%2 != 0 {
		t.Fatalf("not UTF-16LE base64: %v", err)
	}
	u := make([]uint16, len(raw)/2)
	for i := range u {
		u[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	if got := string(utf16.Decode(u)); !strings.Contains(got, "[Console]::In.ReadToEnd()") {
		t.Fatalf("bootstrap = %q", got)
	}
}
