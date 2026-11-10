package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
)

func TestMongoToolURI(t *testing.T) {
	for in, want := range map[string]string{ //nolint:gosec // made-up logins
		"mongodb://u:p@127.0.0.1:27017/admin":                  "mongodb://u:p@127.0.0.1:27017/?authSource=admin",
		"mongodb://u:p@127.0.0.1:27017/shop?authSource=admin":  "mongodb://u:p@127.0.0.1:27017/?authSource=admin",
		"mongodb://u:p@h1:1,h2:2/app?replicaSet=rs0":           "mongodb://u:p@h1:1,h2:2/?replicaSet=rs0&authSource=app",
		"mongodb://127.0.0.1:27017/admin":                      "mongodb://127.0.0.1:27017/?authSource=admin",
		"mongodb://127.0.0.1:27017":                            "mongodb://127.0.0.1:27017",
		"mongodb+srv://u:p%2F@x.example.com/?retryWrites=true": "mongodb+srv://u:p%2F@x.example.com/?retryWrites=true",
	} {
		if got := mongoToolURI(in); got != want {
			t.Errorf("mongoToolURI(%q) = %q, want %q", in, got, want)
		}
	}
}

// The tools read the connection string from a file made from stdin, so
// neither the line nor the audit log holds a password.
func TestMongoDumpKeepsTheLoginOffTheLine(t *testing.T) {
	f := &fakeRun{reply: func(string) audit.Result {
		return audit.Result{Stdout: []byte("/home/u/db-backups/shop-x.archive.gz\n")}
	}}
	s := New(f)
	s.SetMongoURI("srv", "mongodb://admin:hunter2@127.0.0.1:27017/admin") //nolint:gosec // a made-up login
	p, err := s.Dump(context.Background(), "srv", Mongo, "shop")
	if err != nil || p != "/home/u/db-backups/shop-x.archive.gz" {
		t.Fatalf("dump = %q, %v", p, err)
	}
	if strings.Contains(f.lines[0], "hunter2") || !strings.Contains(f.lines[0], `trap 'rm -f "$c"' EXIT`) ||
		!strings.Contains(f.lines[0], "mongodump --config=\"$c\" --db=shop") {
		t.Errorf("line = %s", f.lines[0])
	}
	if f.inputs[0] != "uri: \"mongodb://admin:hunter2@127.0.0.1:27017/?authSource=admin\"\n" { //nolint:gosec // a made-up login
		t.Errorf("stdin = %q", f.inputs[0])
	}
}

func TestMongoRestoreCommand(t *testing.T) {
	s := New(&fakeRun{reply: func(string) audit.Result { return audit.Result{} }})
	for _, f := range []string{"/a.sql.gz", "rel.archive.gz", "/a/../b.archive.gz"} {
		if _, err := s.RestoreCommand(context.Background(), "srv", Mongo, "shop", f); err == nil {
			t.Errorf("%q accepted", f)
		}
	}
	c, err := s.RestoreCommand(context.Background(), "srv", Mongo, "shop", "/home/u/db-backups/shop 1.archive.gz")
	if err != nil || !strings.HasSuffix(c, "mongorestore --config=\"$c\" --archive='/home/u/db-backups/shop 1.archive.gz' --gzip --nsInclude='shop.*' --drop") {
		t.Errorf("command = %s, %v", c, err)
	}
}
