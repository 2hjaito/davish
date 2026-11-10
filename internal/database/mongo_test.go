package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestMaskMongoURI(t *testing.T) {
	for in, want := range map[string]string{
		"mongodb://admin:s3cr%40t@127.0.0.1:27017/admin":          "mongodb://admin:••••@127.0.0.1:27017/admin",
		"mongodb+srv://u:p@cluster.example.com/?authSource=admin": "mongodb+srv://u:••••@cluster.example.com/?authSource=admin",
		"mongodb://127.0.0.1:27017/admin":                         "mongodb://127.0.0.1:27017/admin",
		`connect("mongodb://a:b@h/x") failed; mongodb://c:d@h2/y`: `connect("mongodb://a:••••@h/x") failed; mongodb://c:••••@h2/y`,
	} {
		if got := MaskMongoURI(in); got != want {
			t.Errorf("MaskMongoURI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckMongoURI(t *testing.T) {
	for _, ok := range []string{"mongodb://127.0.0.1", "mongodb+srv://u:p@x.example.com/db"} {
		if err := CheckMongoURI(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "http://x", "mongodb://a b", "mongodb://x\nrm -rf /"} {
		if err := CheckMongoURI(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// The connection string goes on stdin, never on the command line that the
// audit log records.
func TestMongoURIStaysOffTheCommandLine(t *testing.T) {
	f := &fakeRun{reply: func(string) audit.Result { return audit.Result{} }}
	s := New(f)
	s.SetMongoURI("srv", "mongodb://admin:hunter2@127.0.0.1:27017/admin")
	if err := s.TryMongo(context.Background(), "srv", "mongodb://admin:hunter2@127.0.0.1:27017/admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), "srv", Mongo, "shop", "shop"); err != nil {
		t.Fatal(err)
	}
	for i, l := range f.lines {
		if l != mongoLine {
			t.Errorf("line %d = %q", i, l)
		}
		if !strings.Contains(f.inputs[i], `connect("mongodb://admin:hunter2@127.0.0.1:27017/admin")`) {
			t.Errorf("stdin %d = %q", i, f.inputs[i])
		}
	}
}

func TestMongoErrors(t *testing.T) {
	for out, want := range map[string]apperr.Code{
		"MongoServerError: Command listDatabases requires authentication": apperr.DBAuthRequired,
		"MongoServerError: Authentication failed.":                        apperr.DBAuthFailed,
		"MongoServerError: not authorized on admin to execute command":    apperr.DBAuthRequired,
		"MongoNetworkError: connect ECONNREFUSED 127.0.0.1:27017":         apperr.CmdFailed,
	} {
		if got := codeOf(mongoError(out)); got != want {
			t.Errorf("%q: %s, want %s", out, got, want)
		}
	}
	err := mongoError(`SyntaxError: const d = connect("mongodb://u:hunter2@h/a");`)
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("password in %v", err)
	}
}

func TestListMongo(t *testing.T) {
	reply := audit.Result{Exit: 1, Stderr: []byte("MongoServerError: Command listDatabases requires authentication")}
	f := &fakeRun{reply: func(line string) audit.Result {
		if line == mongoLine {
			return reply
		}
		return audit.Result{Exit: 127}
	}}
	s := New(f)
	es, err := s.List(context.Background(), "srv")
	if err != nil {
		t.Fatal(err)
	}
	m := es[len(es)-1]
	if m.Kind != Mongo || !m.Installed || !m.NeedsAuth || m.Error != "" || m.Login != "" {
		t.Fatalf("needs a login: %+v", m)
	}

	reply = audit.Result{Stdout: []byte("admin\t102400\t2\nshop\t40960\t1\nblog\t0\t0\nlocal\t77824\t1\n")}
	s.SetMongoURI("srv", "mongodb://admin:pw@127.0.0.1/admin")
	es, _ = s.List(context.Background(), "srv")
	m = es[len(es)-1]
	if m.NeedsAuth || m.Login != "mongodb://admin:••••@127.0.0.1/admin" || len(m.DBs) != 2 ||
		m.DBs[0] != (DB{Name: "blog"}) || m.DBs[1] != (DB{Name: "shop", Size: 40960, Collections: 1}) {
		t.Fatalf("logged in: %+v", m)
	}
}

func TestMongoStatements(t *testing.T) {
	c, err := CreateSQL(Mongo, "shop", "shop_app")
	if err != nil || !strings.Contains(c, `pwd: "••••••••"`) || !strings.HasPrefix(c, "use shop\n") {
		t.Errorf("create = %q, %v", c, err)
	}
	d, err := DropSQL(Mongo, "shop")
	if err != nil || d != "use shop\ndb.dropDatabase()\ndb.dropAllUsers()" {
		t.Errorf("drop = %q, %v", d, err)
	}
	if _, err := DropSQL(Mongo, `x"); db.shutdownServer(); ("`); err == nil {
		t.Error("unsafe name accepted")
	}
}
