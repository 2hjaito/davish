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

func TestCreateServerKey(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := sshkeys.New(m, audit.NewRunner(m, audit.New(discard{})))
	ctx := context.Background()
	name := "davish-test-deploy"
	t.Cleanup(func() { testsshd.Run(t, m, id, "rm -f -- ~/.ssh/"+name+" ~/.ssh/"+name+".pub") })

	plan, err := s.PlanCreate(ctx, id, name, "deploy@dangth.dev")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, id, name, "deploy@dangth.dev", plan.Command); err != nil {
		t.Fatal(err)
	}
	keys, err := s.ServerKeys(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var found *sshkeys.ServerKey
	for i := range keys {
		if keys[i].Name == name {
			found = &keys[i]
		}
	}
	if found == nil || found.Type != "ssh-ed25519" || found.Comment != "deploy@dangth.dev" || !strings.HasPrefix(found.Public, "ssh-ed25519 ") {
		t.Fatalf("created key = %+v in %+v", found, keys)
	}
	if _, err := s.PlanCreate(ctx, id, name, ""); err == nil {
		t.Error("an existing key name was planned again")
	}
	if _, err := s.PlanGitTest(ctx, id, "example.com", name); err == nil {
		t.Error("an unknown git host was planned")
	}
}

// Runs the real test command from the container with a key GitHub has
// never seen: it must end as denied, or unreachable without internet.
func TestGitTestWithUnknownKey(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := sshkeys.New(m, audit.NewRunner(m, audit.New(discard{})))
	ctx := context.Background()
	name := "davish-test-git"
	t.Cleanup(func() { testsshd.Run(t, m, id, "rm -f -- ~/.ssh/"+name+" ~/.ssh/"+name+".pub") })
	plan, err := s.PlanCreate(ctx, id, name, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, id, name, "", plan.Command); err != nil {
		t.Fatal(err)
	}
	gp, err := s.PlanGitTest(ctx, id, "github.com", name)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.GitTest(ctx, id, "github.com", name, gp.Command)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != sshkeys.GitDenied && res.Outcome != sshkeys.GitUnreachable {
		t.Errorf("result = %+v", res)
	}
	t.Logf("outcome %s: %s", res.Outcome, res.Output)
}
