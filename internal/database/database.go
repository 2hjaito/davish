// Package database lists, creates and drops MariaDB/MySQL and PostgreSQL
// databases through their command line clients on the server. Statements
// that carry a password go on stdin, so they are neither in the audit log
// nor in a shell history.
package database

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "database"

// Engines.
const (
	MySQL    = "mysql" // MariaDB or MySQL
	Postgres = "postgres"
)

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunInput(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// Service talks to the database servers of each server.
type Service struct {
	run Runner
	mu  sync.Mutex
	// access is the prefix that worked last, per server and engine.
	access map[string][]string
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run, access: map[string][]string{}}
}

// DB is one database.
type DB struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Engine is what one database server holds.
type Engine struct {
	Kind      string `json:"kind"`
	Installed bool   `json:"installed"`
	// NeedsRoot is set when only root may connect and sudo asks for a
	// password; Terminal then opens the client.
	NeedsRoot bool   `json:"needsRoot"`
	Client    string `json:"client"`
	DBs       []DB   `json:"dbs"`
	Error     string `json:"error,omitempty"`
}

// How each engine's client is reached: as the user, then through sudo
// without a password. PostgreSQL's admin is the postgres system user.
var prefixes = map[string][][]string{
	MySQL:    {{}, {"sudo", "-n"}},
	Postgres: {{"runuser", "-u", "postgres", "--"}, {"sudo", "-n", "-u", "postgres"}},
}

var clients = map[string][]string{
	MySQL:    {"mysql", "--batch", "--skip-column-names"},
	Postgres: {"psql", "-X", "-q", "-At", "-v", "ON_ERROR_STOP=1", "-d", "postgres"},
}

var listSQL = map[string]string{
	MySQL: `SELECT s.schema_name, COALESCE(SUM(t.data_length + t.index_length), 0)
FROM information_schema.schemata s LEFT JOIN information_schema.tables t ON t.table_schema = s.schema_name
WHERE s.schema_name NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys')
GROUP BY s.schema_name;`,
	Postgres: `SELECT datname || '|' || pg_database_size(datname) FROM pg_database
WHERE NOT datistemplate AND datname <> 'postgres';`,
}

func line(prefix []string, kind string) string {
	args := append(append([]string{}, prefix...), clients[kind]...)
	return shellx.Cmd(args[0], args[1:]...).String()
}

// exec runs sql with kind's client, trying each way in until one gets in.
func (s *Service) exec(ctx context.Context, serverID, kind, sql string) (audit.Result, error) {
	key := serverID + "/" + kind
	s.mu.Lock()
	known := s.access[key]
	s.mu.Unlock()
	tries := prefixes[kind]
	if known != nil {
		tries = [][]string{known}
	}
	var last audit.Result
	missing := true
	for _, p := range tries {
		res, err := s.run.RunInput(ctx, serverID, app, line(p, kind), []byte(sql))
		if err != nil {
			return res, err
		}
		if res.Exit == 127 {
			continue
		}
		missing = false
		if res.Exit == 0 {
			s.mu.Lock()
			s.access[key] = p
			s.mu.Unlock()
			return res, nil
		}
		last = res
		if !denied(string(res.Stderr)) {
			return res, sqlError(res)
		}
	}
	s.mu.Lock()
	delete(s.access, key)
	s.mu.Unlock()
	if missing {
		return last, apperr.New(apperr.CmdMissing, kind+" is not installed")
	}
	return last, apperr.New(apperr.CmdNeedsRoot, "only root may connect")
}

// denied tells a refused login or sudo from an error in the statement.
func denied(out string) bool {
	for _, s := range []string{"Access denied", "password is required", "non-root", "runuser:", "sudo:", "Permission denied", "not allowed", "must be root", "Peer authentication failed", "role \"", "Operation not permitted", "can't connect", "Can't connect", "No such file"} {
		if strings.Contains(out, s) {
			return true
		}
	}
	return false
}

func sqlError(res audit.Result) error {
	msg := strings.TrimSpace(string(res.Stderr))
	if i := strings.IndexByte(msg, '\n'); i > 0 {
		msg = msg[:i]
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return apperr.New(apperr.CmdFailed, msg)
}

// List reads every engine the server has.
func (s *Service) List(ctx context.Context, serverID string) ([]Engine, error) {
	var out []Engine
	for _, kind := range []string{MySQL, Postgres} {
		e := Engine{Kind: kind, DBs: []DB{}, Client: rootClient(kind)}
		res, err := s.exec(ctx, serverID, kind, listSQL[kind])
		switch {
		case err == nil:
			e.Installed = true
			e.DBs = parseList(string(res.Stdout))
		case codeOf(err) == apperr.CmdMissing:
		case codeOf(err) == apperr.CmdNeedsRoot:
			e.Installed, e.NeedsRoot = true, true
		default:
			if ctx.Err() != nil {
				return nil, err
			}
			e.Installed, e.Error = true, err.Error()
		}
		out = append(out, e)
	}
	return out, nil
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// rootClient opens kind's client as its admin, for a terminal.
func rootClient(kind string) string {
	if kind == Postgres {
		return "sudo -u postgres psql"
	}
	return "sudo mysql"
}

// parseList reads "name<TAB or |>size" lines.
func parseList(out string) []DB {
	dbs := []DB{}
	for l := range strings.Lines(out) {
		l = strings.TrimRight(l, "\r\n")
		name, size, ok := strings.Cut(l, "\t")
		if !ok {
			name, size, ok = strings.Cut(l, "|")
		}
		if !ok || name == "" {
			continue
		}
		n, _ := strconv.ParseInt(strings.TrimSpace(size), 10, 64)
		dbs = append(dbs, DB{Name: name, Size: n})
	}
	sort.Slice(dbs, func(i, j int) bool { return dbs[i].Name < dbs[j].Name })
	return dbs
}

// ident is a database or user name Davish creates: plain enough to need no
// quoting in either engine.
var ident = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// CreateResult is a new database's login, shown once and never stored.
type CreateResult struct {
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// CreateSQL is the statement that makes database and its user, with the
// password masked, for the preview.
func CreateSQL(kind, db, user string) (string, error) {
	return createSQL(kind, db, user, "••••••••")
}

func createSQL(kind, db, user, pw string) (string, error) {
	if !ident.MatchString(db) || !ident.MatchString(user) {
		return "", apperr.New(apperr.InvalidInput, "names are lowercase letters, digits and _")
	}
	switch kind {
	case MySQL:
		return "CREATE DATABASE `" + db + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\n" +
			"CREATE USER '" + user + "'@'localhost' IDENTIFIED BY '" + pw + "';\n" +
			"GRANT ALL PRIVILEGES ON `" + db + "`.* TO '" + user + "'@'localhost';\n" +
			"FLUSH PRIVILEGES;\n", nil
	case Postgres:
		return `CREATE USER "` + user + `" WITH PASSWORD '` + pw + "';\n" +
			`CREATE DATABASE "` + db + `" OWNER "` + user + "\";\n", nil
	}
	return "", apperr.New(apperr.InvalidInput, "unknown engine")
}

const pwChars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func password() (string, error) {
	b := make([]byte, 24)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pwChars))))
		if err != nil {
			return "", err
		}
		b[i] = pwChars[n.Int64()]
	}
	return string(b), nil
}

// Create makes db and user with a random password, returned only here.
func (s *Service) Create(ctx context.Context, serverID, kind, db, user string) (CreateResult, error) {
	pw, err := password()
	if err != nil {
		return CreateResult{}, err
	}
	sql, err := createSQL(kind, db, user, pw)
	if err != nil {
		return CreateResult{}, err
	}
	if _, err := s.exec(ctx, serverID, kind, sql); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Database: db, User: user, Password: pw}, nil
}

// safeName accepts an existing database's name that can be quoted
// safely: Davish may not have created it, so it is less strict than ident.
var safeName = regexp.MustCompile(`^[A-Za-z0-9_$-][A-Za-z0-9_$.-]{0,63}$`)

// DropSQL is the statement that deletes db, for the confirmation.
func DropSQL(kind, db string) (string, error) {
	if !safeName.MatchString(db) {
		return "", apperr.New(apperr.InvalidInput, "not a database name")
	}
	switch kind {
	case MySQL:
		return "DROP DATABASE `" + db + "`;", nil
	case Postgres:
		return `DROP DATABASE "` + db + `";`, nil
	}
	return "", apperr.New(apperr.InvalidInput, "unknown engine")
}

// Drop runs sql, which must be what DropSQL gives for db.
func (s *Service) Drop(ctx context.Context, serverID, kind, db, sql string) error {
	want, err := DropSQL(kind, db)
	if err != nil {
		return err
	}
	if want != sql {
		return apperr.New(apperr.FSStale, "the statement changed since it was confirmed")
	}
	_, err = s.exec(ctx, serverID, kind, sql)
	return err
}
