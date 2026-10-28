//go:build integration

package sshkeys_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"davish/internal/audit"
	"davish/internal/sshkeys"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestAddAndRemoveAuthorizedKey(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := sshkeys.New(m, audit.NewRunner(m, audit.New(discard{})))
	ctx := context.Background()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sp))) + " davish-test's key"
	fp := ssh.FingerprintSHA256(sp)

	before, err := s.Authorized(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	logins := 0
	for _, k := range before.Keys {
		if k.Login {
			logins++
		}
	}
	if logins == 0 {
		t.Fatalf("the test key Davish logs in with is not marked: %+v", before.Keys)
	}

	add, err := s.PlanAdd(ctx, id, line)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add(ctx, id, line, add.Command); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PlanAdd(ctx, id, line); err == nil {
		t.Error("the same key planned twice")
	}

	rm, err := s.PlanRemove(ctx, id, fp)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Login || rm.Comment != "davish-test's key" {
		t.Errorf("remove plan = %+v", rm)
	}
	if err := s.Remove(ctx, id, rm); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsshd.Run(t, m, id, "rm -f -- "+rm.Backup) })

	after, err := s.Authorized(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Keys) != len(before.Keys) {
		t.Errorf("keys after = %d, before = %d", len(after.Keys), len(before.Keys))
	}
	if out := testsshd.Run(t, m, id, "stat -c %a ~/.ssh/authorized_keys && test -f "+rm.Backup+" && echo backup"); !strings.Contains(out, "600") || !strings.Contains(out, "backup") {
		t.Errorf("mode/backup check = %q", out)
	}
}
