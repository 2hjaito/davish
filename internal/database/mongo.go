package database

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/store"
)

// Mongo is MongoDB, reached with mongosh on the server (ADR 0017).
const Mongo = "mongo"

// defaultMongoURI reaches a MongoDB on the server without a login, which
// works when authorization is off or no user exists yet.
const defaultMongoURI = "mongodb://127.0.0.1:27017/admin"

// mongoLine runs a script read from stdin. The script carries the
// connection string, so it never shows on a command line, in the audit log
// or in ps.
const mongoLine = "mongosh --quiet --nodb --file /dev/stdin"

// mongoSystem are MongoDB's own databases, left out as MySQL's are.
var mongoSystem = map[string]bool{"admin": true, "config": true, "local": true}

// SetMongoURI sets the connection string for the server's MongoDB in
// memory; an empty uri goes back to connecting without a login.
func (s *Service) SetMongoURI(serverID, uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mongo[serverID] = uri
}

// mongoURI is the connection string to use, read from the keychain once.
func (s *Service) mongoURI(serverID string) string {
	s.mu.Lock()
	u, ok := s.mongo[serverID]
	l := s.logins
	s.mu.Unlock()
	if !ok && l != nil {
		// A keychain that cannot be read counts as no login: MongoDB then
		// asks for one again.
		u, _ = l.Get(serverID, store.SecretMongo)
		s.SetMongoURI(serverID, u)
	}
	if u == "" {
		return defaultMongoURI
	}
	return u
}

// ConnectMongo logs in with uri and, once it works, keeps it in the
// keychain for the next runs.
func (s *Service) ConnectMongo(ctx context.Context, serverID, uri string) error {
	if err := s.TryMongo(ctx, serverID, uri); err != nil {
		return err
	}
	s.mu.Lock()
	l := s.logins
	s.mu.Unlock()
	if l != nil {
		if err := l.Set(serverID, store.SecretMongo, uri); err != nil {
			return err
		}
	}
	s.SetMongoURI(serverID, uri)
	return nil
}

// ForgetMongo removes the server's MongoDB login from the keychain and
// from memory.
func (s *Service) ForgetMongo(serverID string) error {
	s.mu.Lock()
	l := s.logins
	s.mu.Unlock()
	if l != nil {
		if err := l.Delete(serverID, store.SecretMongo); err != nil {
			return err
		}
	}
	s.SetMongoURI(serverID, "")
	return nil
}

// ForgetLogins drops every login kept in memory, after the keychain was
// cleared (the app lock's reset).
func (s *Service) ForgetLogins() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mongo = map[string]string{}
}

var mongoScheme = regexp.MustCompile(`^mongodb(\+srv)?://[^\s]+$`)

// CheckMongoURI tells a connection string Davish can use from one it cannot.
func CheckMongoURI(uri string) error {
	if len(uri) > 2048 || !mongoScheme.MatchString(uri) {
		return apperr.New(apperr.InvalidInput, "a connection string starts with mongodb:// or mongodb+srv://")
	}
	return nil
}

// mongoPassword is the password in a connection string.
var mongoPassword = regexp.MustCompile(`(mongodb(?:\+srv)?://[^:/@\s]*:)[^@\s]*@`)

// MaskMongoURI hides the password of uri, for showing and for errors.
func MaskMongoURI(uri string) string {
	return mongoPassword.ReplaceAllString(uri, "${1}••••@")
}

// mongoScript connects to uri as d, then runs body.
func mongoScript(uri, body string) []byte {
	q, _ := json.Marshal(uri)
	return []byte("const d = connect(" + string(q) + ");\n" + body + "\n")
}

// runMongo runs body against uri and returns what it printed.
func (s *Service) runMongo(ctx context.Context, serverID, uri, body string) (string, error) {
	res, err := s.run.RunInput(ctx, serverID, app, mongoLine, mongoScript(uri, body))
	if err != nil {
		return "", err
	}
	if res.Exit == 127 {
		return "", apperr.New(apperr.CmdMissing, "mongosh is not installed")
	}
	if res.Exit != 0 {
		return "", mongoError(string(res.Stderr) + string(res.Stdout))
	}
	return string(res.Stdout), nil
}

// mongoError reads mongosh's error, with any password masked: a mistake in
// the script can echo the line that holds the connection string.
func mongoError(out string) error {
	msg := MaskMongoURI(strings.TrimSpace(out))
	// The tools log a line per step: the error is the one that says Failed.
	if i := strings.Index(msg, "Failed: "); i >= 0 {
		msg = msg[i:]
	}
	if i := strings.IndexByte(msg, '\n'); i > 0 {
		msg = msg[:i]
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	switch {
	case strings.Contains(out, "Authentication failed"):
		return apperr.New(apperr.DBAuthFailed, msg)
	case strings.Contains(out, "requires authentication"), strings.Contains(out, "not authorized"):
		return apperr.New(apperr.DBAuthRequired, msg)
	}
	return apperr.New(apperr.CmdFailed, msg)
}

// TryMongo checks that uri logs in and may list databases, before it is
// kept.
func (s *Service) TryMongo(ctx context.Context, serverID, uri string) error {
	if err := CheckMongoURI(uri); err != nil {
		return err
	}
	_, err := s.runMongo(ctx, serverID, uri, "d.adminCommand({listDatabases: 1, nameOnly: true});")
	return err
}

// Each database as "name<TAB>bytes on disk<TAB>collections". A database
// Davish created has a user but no data yet, and MongoDB lists it only once
// something is written, so the databases users belong to are added too.
const mongoListJS = `const seen = new Set();
for (const x of d.adminCommand({listDatabases: 1}).databases) {
  seen.add(x.name);
  print(x.name + "\t" + x.sizeOnDisk + "\t" + d.getSiblingDB(x.name).getCollectionNames().length);
}
try {
  for (const u of d.getSiblingDB("admin").system.users.find({}, {db: 1}).toArray()) {
    if (!seen.has(u.db)) { seen.add(u.db); print(u.db + "\t0\t0"); }
  }
} catch (e) {}`

func (s *Service) listMongo(ctx context.Context, serverID string) Engine {
	uri := s.mongoURI(serverID)
	e := Engine{Kind: Mongo, DBs: []DB{}, Client: "mongosh", Login: MaskMongoURI(uri)}
	if uri == defaultMongoURI {
		e.Login = ""
	}
	out, err := s.runMongo(ctx, serverID, uri, mongoListJS)
	switch code := codeOf(err); {
	case err == nil:
		e.Installed = true
		e.DBs = parseMongoList(out)
	case code == apperr.CmdMissing:
	case code == apperr.DBAuthRequired, code == apperr.DBAuthFailed:
		e.Installed, e.NeedsAuth = true, true
		if code == apperr.DBAuthFailed {
			e.Error = err.Error()
		}
	default:
		e.Installed, e.Error = true, err.Error()
	}
	return e
}

func parseMongoList(out string) []DB {
	dbs := []DB{}
	for l := range strings.Lines(out) {
		f := strings.Split(strings.TrimRight(l, "\r\n"), "\t")
		if len(f) != 3 || f[0] == "" || mongoSystem[f[0]] {
			continue
		}
		size, _ := strconv.ParseInt(f[1], 10, 64)
		n, _ := strconv.Atoi(f[2])
		dbs = append(dbs, DB{Name: f[0], Size: size, Collections: n})
	}
	return sortDBs(dbs)
}

// mongoCreate is the script that makes db's user, shown with pw masked.
// MongoDB makes the database itself on its first write.
func mongoCreate(db, user, pw string) string {
	return `use ` + db + "\n" +
		`db.createUser({user: "` + user + `", pwd: "` + pw + `", roles: [{role: "readWrite", db: "` + db + `"}]})`
}

// mongoCreateJS runs mongoCreate through the connection d.
func mongoCreateJS(db, user, pw string) string {
	q := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	return "d.getSiblingDB(" + q(db) + ").createUser({user: " + q(user) + ", pwd: " + q(pw) +
		", roles: [{role: \"readWrite\", db: " + q(db) + "}]});"
}

// mongoDrop is the script that deletes db and the users defined in it.
func mongoDrop(db string) string {
	return "use " + db + "\ndb.dropDatabase()\ndb.dropAllUsers()"
}

func mongoDropJS(db string) string {
	b, _ := json.Marshal(db)
	return "const t = d.getSiblingDB(" + string(b) + ");\nt.dropDatabase();\nt.dropAllUsers();"
}
