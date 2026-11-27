package conn

import (
	"testing"

	"davish/internal/store"
)

func TestProxyConnRemoteAddrIsHostPort(t *testing.T) {
	// The host key check splits the reported address, so it has to read as
	// the server's host:port even though the transport is a command.
	pc, err := dialProxy(store.Server{Host: "sandbox.internal", User: "u", Port: 2222, ProxyCommand: "cat"})
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close() //nolint:errcheck // test cleanup
	if got := pc.RemoteAddr().String(); got != "sandbox.internal:2222" {
		t.Errorf("RemoteAddr = %q; want sandbox.internal:2222", got)
	}
}

func TestExpandProxyTokens(t *testing.T) {
	srv := store.Server{Host: "sandbox.internal", Port: 2222, User: "admin"}
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"plain command", "websocat -b - ws://relay/tunnel", "websocat -b - ws://relay/tunnel"},
		{"host and port", "nc %h %p", "nc sandbox.internal 2222"},
		{"user", "ssh -l %r %h", "ssh -l admin sandbox.internal"},
		{"original host", "ssh -W %n:%p jumphost", "ssh -W sandbox.internal:2222 jumphost"},
		{"literal percent", "echo 100%%", "echo 100%"},
		{"lone percent at end", "echo %", "echo %"},
		{"unknown token kept", "hash %C", "hash %C"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandProxyTokens(tt.cmd, srv); got != tt.want {
				t.Errorf("expandProxyTokens(%q) = %q; want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
