//go:build integration

package security_test

import (
	"context"
	"testing"

	"davish/internal/audit"
	"davish/internal/security"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestReportSeesOwnLogin(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := security.New(audit.NewRunner(m, audit.New(discard{})))
	r, err := s.Report(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Source != "journal" || len(r.Accepted) == 0 || r.Accepted[0].User != testsshd.User {
		t.Fatalf("report = %+v", r)
	}
	if !r.LastbDenied || r.Root {
		t.Errorf("lastb denied = %v, root = %v", r.LastbDenied, r.Root)
	}
}
