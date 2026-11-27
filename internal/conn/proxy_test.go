package conn

import (
	"testing"

	"davish/internal/store"
)

func TestProxyConnRemoteAddrIsHostPort(t *testing.T) {
	// The host key check splits the reported address, so it has to read as
	// the server's host:port even though the transport is a command.
	pc, err := dialProxy(typedProxy(store.Server{Host: "sandbox.internal", User: "u", Port: 2222, ProxyCommand: "cat"}))
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
			got, err := expandProxyTokens(tt.cmd, srv)
			if err != nil || got != tt.want {
				t.Errorf("expandProxyTokens(%q) = %q, %v; want %q", tt.cmd, got, err, tt.want)
			}
		})
	}
}

// The line is built for a shell, so a host or login with a character the
// shell reads is refused where it is put in, not left to store.validate.
func TestExpandProxyTokensRefusesUnsafeValues(t *testing.T) {
	bad := []string{"a;b", "$(id)", "`id`", "a b", "a|b", "a&b", "a\nb", "a'b", "a\"b", "a>b", "a*", "-x;y", ""}
	for _, v := range bad {
		for _, tok := range []string{"%h", "%n", "%r"} {
			srv := store.Server{Host: "ok.example", User: "admin", Port: 22}
			if tok == "%r" {
				srv.User = v
			} else {
				srv.Host = v
			}
			if got, err := expandProxyTokens("nc "+tok, srv); err == nil {
				t.Errorf("%s with %q went into the shell line: %q", tok, v, got)
			}
		}
	}
	// A value that is not used cannot hurt: only what is substituted is checked.
	if got, err := expandProxyTokens("websocat -b - ws://relay/tunnel", store.Server{Host: "a;b", User: "x y"}); err != nil || got != "websocat -b - ws://relay/tunnel" {
		t.Errorf("unused values were checked: %q, %v", got, err)
	}
	for _, ok := range []string{"sandbox.internal", "10.0.0.5", "::1", "fe80::1", "jane.doe@example.com", "_svc-1"} {
		if _, err := expandProxyTokens("nc %h %r", store.Server{Host: ok, User: ok, Port: 22}); err != nil {
			t.Errorf("%q was refused: %v", ok, err)
		}
	}
}

func TestProxyConfirmed(t *testing.T) {
	cmd := "websocat -b - ws://relay/tunnel"
	for name, tt := range map[string]struct {
		srv  store.Server
		want bool
	}{
		"no proxy command":   {store.Server{}, true},
		"typed or confirmed": {store.Server{ProxyCommand: cmd, ProxyTrust: store.ProxyHash(cmd)}, true},
		"never confirmed":    {store.Server{ProxyCommand: cmd}, false},
		"another command":    {store.Server{ProxyCommand: cmd, ProxyTrust: store.ProxyHash("nc evil 1")}, false},
		"changed afterwards": {store.Server{ProxyCommand: cmd + " --x", ProxyTrust: store.ProxyHash(cmd)}, false},
	} {
		if got := ProxyConfirmed(tt.srv); got != tt.want {
			t.Errorf("%s: ProxyConfirmed = %v; want %v", name, got, tt.want)
		}
	}
}

func TestPlanProxy(t *testing.T) {
	srv := store.Server{Host: "sandbox.internal", Port: 2222, User: "admin", ProxyCommand: "nc %h %p"}
	plan, err := PlanProxy(srv)
	if err != nil || plan.Command != "nc sandbox.internal 2222" || plan.Hash != store.ProxyHash("nc %h %p") {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	srv.Host = "a;b"
	if _, err := PlanProxy(srv); err == nil {
		t.Error("a plan was made for an unsafe host")
	}
}

// typedProxy is srv as the dialog saves it: its proxy command was typed by
// the user, so it is confirmed (ADR 0031).
func typedProxy(srv store.Server) store.Server {
	srv.ProxyTrust = store.ProxyHash(srv.ProxyCommand)
	return srv
}
