package database

import (
	"reflect"
	"testing"
)

func TestParseFound(t *testing.T) {
	out := `#ps
cron
mariadbd
mongod
postgres
postgres
sshd
#mongod.conf
# mongod.conf
storage:
  dbPath: /var/lib/mongodb
net:
  port: 27018   # moved
  bindIp: 127.0.0.1,10.0.0.5
#security:
security:
  authorization: "enabled"
#sqlite3
/usr/bin/sqlite3
#files
ok	8192	/srv/app/app.db
root	4096	/var/lib/vault/db.sqlite3
weird line
`
	want := Found{
		Running: []Running{
			{Kind: MySQL, Port: 3306},
			{Kind: Mongo, Port: 27018, Bind: "127.0.0.1,10.0.0.5", Auth: "on"},
			{Kind: Postgres, Port: 5432},
		},
		SQLite: []SQLiteFile{
			{Path: "/srv/app/app.db", Size: 8192},
			{Path: "/var/lib/vault/db.sqlite3", Size: 4096, Root: true},
		},
		SQLite3: true,
	}
	if got := parseFound(out); !reflect.DeepEqual(got, want) {
		t.Errorf("parseFound =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseFoundMongoWithoutConfig(t *testing.T) {
	got := parseFound("#ps\nmongod\n#mongod.conf\n#sqlite3\n#files\n")
	if len(got.Running) != 1 || got.Running[0] != (Running{Kind: Mongo, Port: 27017}) || got.SQLite3 || len(got.SQLite) != 0 {
		t.Errorf("got %+v", got)
	}
	if conf := parseFound("#ps\nmongod\n#mongod.conf\nnet:\n  port: 27017\n#sqlite3\n#files\n"); conf.Running[0].Auth != "off" {
		t.Errorf("no security section: %+v", conf.Running[0])
	}
}
