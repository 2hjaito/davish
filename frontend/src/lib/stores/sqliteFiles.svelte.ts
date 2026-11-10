import { load, save } from './persist'

const KEY = 'databaseSQLite'

/**
 * The SQLite files added to Database, per server. Paths only, in
 * settings.json on this computer (ADR 0017); nothing secret.
 */
class SQLiteFiles {
  byServer = $state<Record<string, string[]>>(load(KEY, {}))

  of(serverID: string): string[] {
    return this.byServer[serverID] ?? []
  }

  add(serverID: string, path: string) {
    if (this.of(serverID).includes(path)) return
    this.byServer = { ...this.byServer, [serverID]: [...this.of(serverID), path] }
    save(KEY, this.byServer)
  }

  remove(serverID: string, path: string) {
    this.byServer = { ...this.byServer, [serverID]: this.of(serverID).filter((p) => p !== path) }
    save(KEY, this.byServer)
  }
}

export const sqliteFiles = new SQLiteFiles()
