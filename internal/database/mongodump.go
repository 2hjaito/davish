package database

import (
	"context"
	"encoding/json"
	"path"
	"regexp"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// mongoDumpExt ends the archives mongodump writes to ~/db-backups.
const mongoDumpExt = ".archive.gz"

// withMongoConfig starts a line that reads the connection string from
// stdin into a file only the user can read, removed when the line ends
// however it ends: mongodump and mongorestore take no password on stdin,
// and one on their command line would show in ps and the audit log.
const withMongoConfig = `umask 077 && mkdir -p "$HOME/.cache/davish" && ` +
	`c=$(mktemp "$HOME/.cache/davish/mongo.XXXXXX") && trap 'rm -f "$c"' EXIT && cat > "$c" && `

// mongoToolURI is uri as mongodump and mongorestore take it. They refuse a
// database in the URI beside --db, so it moves to authSource, the
// database a login belongs to.
func mongoToolURI(uri string) string {
	scheme, rest, _ := strings.Cut(uri, "://")
	at := strings.LastIndexByte(rest, '@')
	slash := strings.IndexByte(rest[at+1:], '/')
	if slash < 0 {
		return uri
	}
	slash += at + 1
	hosts, tail := rest[:slash], rest[slash+1:]
	db, query, _ := strings.Cut(tail, "?")
	if db != "" && !strings.Contains(query, "authSource=") {
		if query != "" {
			query += "&"
		}
		query += "authSource=" + db
	}
	out := scheme + "://" + hosts + "/"
	if query != "" {
		out += "?" + query
	}
	return out
}

// mongoConfig is the YAML file mongodump and mongorestore read: JSON
// strings are YAML strings.
func mongoConfig(uri string) []byte {
	q, _ := json.Marshal(mongoToolURI(uri))
	return []byte("uri: " + string(q) + "\n")
}

func mongoDumpLine(db, stamp string) string {
	f := `"$HOME/db-backups/"` + shellx.Quote(db+"-"+stamp+mongoDumpExt)
	return withMongoConfig + `mkdir -p "$HOME/db-backups" && mongodump --config="$c" --db=` +
		shellx.Quote(db) + " --archive=" + f + " --gzip && echo " + f
}

var mongoDumpFile = regexp.MustCompile(`^/[^\x00\n\r]*\.archive\.gz$`)

// mongoRestoreLine loads file into db, replacing its collections.
func mongoRestoreLine(db, file string) (string, error) {
	if !mongoDumpFile.MatchString(file) || path.Clean(file) != file {
		return "", apperr.New(apperr.InvalidInput, "a MongoDB dump is an .archive.gz file")
	}
	return withMongoConfig + "mongorestore --config=\"$c\" --archive=" + shellx.Quote(file) +
		" --gzip --nsInclude=" + shellx.Quote(db+".*") + " --drop", nil
}

// runMongoTool runs a line made by mongoDumpLine or mongoRestoreLine, the
// connection string on its stdin.
func (s *Service) runMongoTool(ctx context.Context, serverID, line string) (string, error) {
	res, err := s.run.RunInput(ctx, serverID, app, line, mongoConfig(s.mongoURI(serverID)))
	if err != nil {
		return "", err
	}
	if res.Exit == 127 {
		return "", apperr.New(apperr.CmdMissing, "mongodump or mongorestore is not installed (mongodb-database-tools)")
	}
	if res.Exit != 0 {
		return "", mongoError(string(res.Stderr) + string(res.Stdout))
	}
	return string(res.Stdout), nil
}

func (s *Service) dumpMongo(ctx context.Context, serverID, db string) (string, error) {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	out, err := s.runMongoTool(ctx, serverID, mongoDumpLine(db, stamp))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
