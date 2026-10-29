package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
)

func TestDumpLine(t *testing.T) {
	got := dumpLine([]string{"sudo", "-n"}, MySQL, "shop", "20261029T000000Z")
	want := `mkdir -p "$HOME/db-backups" && sudo -n mysqldump --single-transaction --routines --triggers shop > "$HOME/db-backups/shop-20261029T000000Z.sql" && gzip -- "$HOME/db-backups/shop-20261029T000000Z.sql" && echo "$HOME/db-backups/shop-20261029T000000Z.sql.gz"`
	if got != want {
		t.Errorf("dump =\n%s\nwant\n%s", got, want)
	}
}

func TestRestoreLine(t *testing.T) {
	got := restoreLine([]string{"runuser", "-u", "postgres", "--"}, restoreTools[Postgres], "shop", "/root/db-backups/shop 1.sql.gz")
	want := "gzip -t -- '/root/db-backups/shop 1.sql.gz' && gzip -dc -- '/root/db-backups/shop 1.sql.gz' | runuser -u postgres -- psql -X -q -v ON_ERROR_STOP=1 -d shop"
	if got != want {
		t.Errorf("restore =\n%s\nwant\n%s", got, want)
	}
	if got := restoreLine(nil, restoreTools[MySQL], "shop", "/a.sql"); got != "mysql --batch shop < /a.sql" {
		t.Errorf("plain = %s", got)
	}
}

func TestRestoreCommandChecks(t *testing.T) {
	s := New(&fakeRun{reply: func(string) audit.Result { return audit.Result{} }})
	for _, f := range []string{"rel.sql", "/a.txt", "/a/../b.sql"} {
		if _, err := s.RestoreCommand(context.Background(), "srv", MySQL, "shop", f); err == nil {
			t.Errorf("%q accepted", f)
		}
	}
	c, err := s.RestoreCommand(context.Background(), "srv", MySQL, "shop", "/x.sql.gz")
	if err != nil || !strings.HasPrefix(c, "gzip -t") {
		t.Errorf("command = %s, %v", c, err)
	}
}
