package sysmon

import (
	"bytes"
	"context"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// runScript runs a fixed script once and returns its @@sections. Like the
// collector, the script goes in on stdin so the login shell does not matter.
func runScript(ctx context.Context, client *ssh.Client, script string) (map[string][]string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer sess.Close() //nolint:errcheck // closed either way

	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks Output
	defer stop()

	sess.Stdin = bytes.NewReader([]byte(script))
	out, err := sess.Output("sh -s")
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("run script: %w", err)
	}
	return NewReader(bytes.NewReader(out)).sections()
}
