//go:build !windows

package conn

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"davish/internal/apperr"
)

func TestCheckControlSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "dv") // short path: unix socket paths are limited
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sock := filepath.Join(dir, "m.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	code := func(err error) apperr.Code {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return ae.Code
		}
		return ""
	}

	if err := os.Chmod(sock, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkControlSocket(sock); err != nil {
		t.Fatalf("own 0600 socket: %v", err)
	}
	if err := os.Chmod(sock, 0o660); err != nil { //nolint:gosec // the point of the test
		t.Fatal(err)
	}
	if err := checkControlSocket(sock); code(err) != apperr.ConnControlSocket {
		t.Fatalf("group-accessible socket: err = %v; want CONN_CONTROL_SOCKET", err)
	}
	if err := checkControlSocket(filepath.Join(dir, "missing")); code(err) != apperr.ConnControlSocket {
		t.Fatalf("missing socket: err = %v", err)
	}
	file := filepath.Join(dir, "plain")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkControlSocket(file); code(err) != apperr.ConnControlSocket {
		t.Fatalf("regular file: err = %v", err)
	}
}
