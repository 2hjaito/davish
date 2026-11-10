//go:build integration

package database_test

import (
	"context"
	"errors"
	"strings"
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

func TestMongoDumpRestore(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.Connect(t)
	s := database.New(audit.NewRunner(m, audit.New(discard{})))
	s.SetMongoURI(id, mongoAdmin)

	file, err := s.Dump(ctx, id, database.Mongo, "shop")
	if err != nil {
		t.Fatal(err)
	}
	dumps, err := s.Dumps(ctx, id, database.Mongo, "shop")
	if err != nil || len(dumps) == 0 || dumps[0] != file {
		t.Fatalf("dumps = %v, %v (want %s first)", dumps, err, file)
	}
	if sql, _ := s.Dumps(ctx, id, database.MySQL, "shop"); len(sql) != 0 {
		t.Errorf("MySQL dumps list a MongoDB archive: %v", sql)
	}
	cmd, err := s.RestoreCommand(ctx, id, database.Mongo, "shop", file)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(ctx, id, database.Mongo, "shop", file, cmd); err != nil {
		t.Fatal(err)
	}
	if e := mongoEngine(t, s, id); len(e.DBs) == 0 || e.DBs[0].Name != "shop" || e.DBs[0].Collections != 1 {
		t.Fatalf("after restore: %+v", e)
	}
	// No login file stays behind, whatever happened.
	res, err := audit.NewRunner(m, audit.New(discard{})).RunInput(ctx, id, "test", `ls -A "$HOME/.cache/davish"`, nil)
	if err != nil || strings.TrimSpace(string(res.Stdout)) != "" {
		t.Fatalf("left in ~/.cache/davish: %q, %v", res.Stdout, err)
	}

	s.SetMongoURI(id, "mongodb://davish-admin:wrong@127.0.0.1:27017/admin")
	var ae *apperr.Error
	if _, err := s.Dump(ctx, id, database.Mongo, "shop"); !errors.As(err, &ae) || ae.Code != apperr.DBAuthFailed {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestFind(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := database.New(audit.NewRunner(m, audit.New(discard{})))
	f, err := s.Find(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !f.SQLite3 || len(f.Running) != 1 || f.Running[0] != (database.Running{Kind: database.Mongo, Port: 27017, Bind: "127.0.0.1", Auth: "on"}) {
		t.Fatalf("found %+v", f)
	}
	// tester cannot read /var/lib/davish-private and has no sudo; notes.db
	// is not SQLite.
	if len(f.SQLite) != 1 || f.SQLite[0].Path != "/srv/davish-app/app.db" || f.SQLite[0].Root || f.SQLite[0].Size == 0 {
		t.Fatalf("sqlite %+v", f.SQLite)
	}
}
