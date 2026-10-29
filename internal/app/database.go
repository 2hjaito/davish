package app

import "davish/internal/database"

// DatabaseList returns the MariaDB/MySQL and PostgreSQL databases.
func (a *App) DatabaseList(serverID string) ([]database.Engine, error) {
	return a.databases.List(a.ctx, serverID)
}

// DatabaseCreateSQL is the statement that makes a database and its user,
// with the password masked.
func (a *App) DatabaseCreateSQL(kind, db, user string) (string, error) {
	return database.CreateSQL(kind, db, user)
}

// DatabaseCreate makes a database and its user with a random password,
// returned once and never stored.
func (a *App) DatabaseCreate(serverID, kind, db, user string) (database.CreateResult, error) {
	if err := a.lock.Check(); err != nil {
		return database.CreateResult{}, err
	}
	return a.databases.Create(a.ctx, serverID, kind, db, user)
}

// DatabaseDropSQL is the statement that deletes a database.
func (a *App) DatabaseDropSQL(kind, db string) (string, error) {
	return database.DropSQL(kind, db)
}

// DatabaseDrop runs the confirmed statement of DatabaseDropSQL.
func (a *App) DatabaseDrop(serverID, kind, db, sql string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.databases.Drop(a.ctx, serverID, kind, db, sql)
}

// DatabaseDump writes a database to ~/db-backups and returns the file.
func (a *App) DatabaseDump(serverID, kind, db string) (string, error) {
	if err := a.lock.Check(); err != nil {
		return "", err
	}
	return a.databases.Dump(a.ctx, serverID, kind, db)
}

// DatabaseDumps lists a database's dumps in ~/db-backups, newest first.
func (a *App) DatabaseDumps(serverID, db string) ([]string, error) {
	return a.databases.Dumps(a.ctx, serverID, db)
}

// DatabaseRestoreCommand is the line that loads a dump into a database.
func (a *App) DatabaseRestoreCommand(serverID, kind, db, file string) (string, error) {
	return a.databases.RestoreCommand(a.ctx, serverID, kind, db, file)
}

// DatabaseRestore runs the confirmed line of DatabaseRestoreCommand.
func (a *App) DatabaseRestore(serverID, kind, db, file, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.databases.Restore(a.ctx, serverID, kind, db, file, command)
}
