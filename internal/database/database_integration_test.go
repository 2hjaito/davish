//go:build integration

package database_test

import (
	"context"
	"errors"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/database"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

// The test sshd's MongoDB has authorization on and a throwaway admin
// (test/sshd/Dockerfile).
const mongoAdmin = "mongodb://davish-admin:davish-test-only@127.0.0.1:27017/admin"

func mongoEngine(t *testing.T, s *database.Service, id string) database.Engine {
	t.Helper()
	es, err := s.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if e.Kind == database.Mongo {
			return e
		}
	}
	t.Fatal("no MongoDB in the list")
	return database.Engine{}
}

func TestMongoLoginCreateDrop(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.Connect(t)
	s := database.New(audit.NewRunner(m, audit.New(discard{})))

	if e := mongoEngine(t, s, id); !e.Installed || !e.NeedsAuth {
		t.Fatalf("without a login: %+v", e)
	}
	var ae *apperr.Error
	err := s.TryMongo(ctx, id, "mongodb://davish-admin:wrong@127.0.0.1:27017/admin")
	if !errors.As(err, &ae) || ae.Code != apperr.DBAuthFailed {
		t.Fatalf("wrong password: %v", err)
	}
	if err := s.TryMongo(ctx, id, mongoAdmin); err != nil {
		t.Fatal(err)
	}
	s.SetMongoURI(id, mongoAdmin)

	e := mongoEngine(t, s, id)
	if e.NeedsAuth || len(e.DBs) == 0 || e.DBs[0].Name != "shop" || e.DBs[0].Collections != 1 {
		t.Fatalf("logged in: %+v", e)
	}

	r, err := s.Create(ctx, id, database.Mongo, "davish_it", "davish_it")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sql, _ := database.DropSQL(database.Mongo, "davish_it")
		_ = s.Drop(context.Background(), id, database.Mongo, "davish_it", sql)
	})
	// The new user logs in to its own database.
	own := "mongodb://davish_it:" + r.Password + "@127.0.0.1:27017/davish_it"
	if err := s.TryMongo(ctx, id, own); err != nil {
		t.Fatalf("new user: %v", err)
	}
	found := false
	for _, db := range mongoEngine(t, s, id).DBs {
		found = found || db.Name == "davish_it"
	}
	if !found {
		t.Fatal("created database not listed")
	}

	sql, _ := database.DropSQL(database.Mongo, "davish_it")
	if err := s.Drop(ctx, id, database.Mongo, "davish_it", sql); err != nil {
		t.Fatal(err)
	}
	for _, db := range mongoEngine(t, s, id).DBs {
		if db.Name == "davish_it" {
			t.Fatal("dropped database still listed")
		}
	}
}
