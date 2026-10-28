//go:build integration

package packages_test

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
	"davish/internal/packages"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestInventoryMatchesDpkg(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := packages.New(audit.NewRunner(m, audit.New(discard{})), discard{})
	ctx := context.Background()

	inv, err := s.Inventory(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Manager != packages.Apt || len(inv.Installed) < 50 {
		t.Fatalf("manager %q with %d packages", inv.Manager, len(inv.Installed))
	}
	want := ""
	for _, l := range strings.Split(testsshd.Run(t, m, id, "dpkg -s nginx"), "\n") {
		if v, ok := strings.CutPrefix(l, "Version: "); ok {
			want = strings.TrimSpace(v)
		}
	}
	var got string
	for _, p := range inv.Installed {
		if p.Name == "nginx" {
			got = p.Version
		}
	}
	if got == "" || got != want {
		t.Errorf("nginx version = %q, dpkg -s says %q", got, want)
	}
	// The image drops apt's lists, so it never fetched them.
	if inv.ListsAge != -1 {
		t.Errorf("listsAge = %d", inv.ListsAge)
	}
	if err := s.CheckSecurity(ctx, id); err != nil {
		t.Error(err)
	}
}
