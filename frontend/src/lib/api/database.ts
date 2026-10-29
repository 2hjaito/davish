// Typed wrappers for the database bindings (internal/app/database.go).
import * as App from '../../../wailsjs/go/app/App'

export type DBKind = 'mysql' | 'postgres'

export interface DB {
  name: string
  /** Bytes. */
  size: number
}

export interface DBEngine {
  kind: DBKind
  installed: boolean
  /** Only root may connect; client opens it in a terminal. */
  needsRoot: boolean
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
