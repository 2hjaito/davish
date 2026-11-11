// Typed wrappers for the database bindings (internal/app/database.go).
import * as App from '../../../wailsjs/go/app/App'

export type DBKind = 'mysql' | 'postgres' | 'mongo'

export interface DB {
  name: string
  /** Bytes. */
  size: number
  /** MongoDB's collections. */
  collections?: number
}

export interface DBEngine {
  kind: DBKind
  installed: boolean
  /** Only root may connect; client opens it in a terminal. */
  needsRoot: boolean
  /** sudo would let the user in once given the session's password (ADR 0018). */
  askSudo?: boolean
  /** MongoDB wants a login Davish does not have. */
  needsAuth?: boolean
  /** The connection string in use, its password masked. */
  login?: string
  client: string
  dbs: DB[]
  error?: string
}

/** A new database's login: shown once, never stored. */
export interface DBLogin {
  database: string
  user: string
  password: string
}

export const databaseList = (serverID: string): Promise<DBEngine[]> =>
  App.DatabaseList(serverID) as Promise<DBEngine[]>

export const databaseCreateSQL = (kind: DBKind, db: string, user: string): Promise<string> =>
  App.DatabaseCreateSQL(kind, db, user)

export const databaseCreate = (
  serverID: string,
  kind: DBKind,
  db: string,
  user: string,
): Promise<DBLogin> => App.DatabaseCreate(serverID, kind, db, user)

export const databaseDropSQL = (kind: DBKind, db: string): Promise<string> =>
  App.DatabaseDropSQL(kind, db)

export const databaseDrop = (
  serverID: string,
  kind: DBKind,
  db: string,
  sql: string,
): Promise<void> => App.DatabaseDrop(serverID, kind, db, sql)

/** Writes a database to ~/db-backups; resolves to the file's path. */
export const databaseDump = (serverID: string, kind: DBKind, db: string): Promise<string> =>
  App.DatabaseDump(serverID, kind, db)

/** Dumps of a database, or backups of a SQLite file by its backup name. */
export const databaseDumps = (
  serverID: string,
  kind: DBKind | 'sqlite',
  db: string,
): Promise<string[]> => App.DatabaseDumps(serverID, kind, db)

export const databaseRestoreCommand = (
  serverID: string,
  kind: DBKind,
  db: string,
  file: string,
): Promise<string> => App.DatabaseRestoreCommand(serverID, kind, db, file)

export const databaseRestore = (
  serverID: string,
  kind: DBKind,
  db: string,
  file: string,
  command: string,
): Promise<void> => App.DatabaseRestore(serverID, kind, db, file, command)

/** Logs in to the server's MongoDB; kept in the keychain once it works. */
export const databaseMongoConnect = (serverID: string, uri: string): Promise<void> =>
  App.DatabaseMongoConnect(serverID, uri)

/** Removes the server's MongoDB login from the keychain. */
export const databaseMongoForget = (serverID: string): Promise<void> =>
  App.DatabaseMongoForget(serverID)

/** A database server Find saw running. */
export interface Running {
  kind: DBKind
  /** 0 when unknown. */
  port: number
  bind?: string
  /** MongoDB's authorization. */
  auth?: 'on' | 'off' | ''
}

export interface SQLiteFile {
  path: string
  size: number
  /** Only root can read it, through sudo -n. */
  root: boolean
}

/** What Find saw on the server (internal/database.Found). */
export interface Found {
  running: Running[]
  sqlite: SQLiteFile[]
  /** The sqlite3 client is installed. */
  sqlite3: boolean
}

/** Looks for database servers and SQLite files, changing nothing. */
export const databaseFind = (serverID: string): Promise<Found> =>
  App.DatabaseFind(serverID) as Promise<Found>

export interface SQLiteInfo {
  path: string
  size: number
  journal: string
  tables: { name: string; rows: number }[]
  root: boolean
  /** The name its backups go under in ~/db-backups. */
  backup: string
}

export const databaseSQLite = (serverID: string, file: string): Promise<SQLiteInfo> =>
  App.DatabaseSQLite(serverID, file) as Promise<SQLiteInfo>

/** Copies a SQLite file to ~/db-backups; resolves to the backup's path. */
export const databaseSQLiteBackup = (serverID: string, file: string): Promise<string> =>
  App.DatabaseSQLiteBackup(serverID, file)

/** The line that puts a backup back, for a terminal. */
export const databaseSQLiteRestoreLine = (file: string, backup: string): Promise<string> =>
  App.DatabaseSQLiteRestoreLine(file, backup)

/** A file name SQLite databases are often saved under. */
export const looksLikeSQLite = (name: string): boolean => /\.(db|sqlite3?)$/i.test(name)
