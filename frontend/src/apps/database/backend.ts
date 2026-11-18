// The database calls, or demo data in the plain browser preview.
import {
  databaseCreate,
  databaseCreateSQL,
  databaseDrop,
  databaseDropSQL,
  databaseDump,
  databaseDumps,
  databaseFind,
  databaseMongoConnect,
  databaseMongoForget,
  databaseRestore,
  databaseRestoreCommand,
  databaseList,
  databaseSQLite,
  databaseSQLiteBackup,
  databaseSQLiteRestoreLine,
  databaseSQLiteRows,
  type DBEngine,
  type DBKind,
  type DBLogin,
  type Found,
  type SQLiteInfo,
  type SQLiteRows,
} from '../../lib/api/database'
import { tunnelStart } from '../../lib/api/ports'
import { hasRuntime } from '../../lib/api/runtime'

let demo: DBEngine[] = [
  {
    kind: 'mysql',
    installed: true,
    needsRoot: false,
    client: 'sudo mysql',
    dbs: [
      { name: 'shop', size: 48_234_496 },
      { name: 'wordpress', size: 12_582_912 },
    ],
  },
  { kind: 'postgres', installed: true, needsRoot: true, client: 'sudo -u postgres psql', dbs: [] },
  { kind: 'mongo', installed: true, needsRoot: false, needsAuth: true, client: 'mongosh', dbs: [] },
]

const demoMongo = [
  { name: 'davitek', size: 86_016_000, collections: 14 },
  { name: 'analytics', size: 31_457_280, collections: 6 },
]

const demoFound: Found = {
  running: [
    { kind: 'mysql', port: 3306 },
    { kind: 'mongo', port: 27017, bind: '127.0.0.1', auth: 'on' },
  ],
  sqlite: [
    { path: '/opt/uptime-kuma/data/kuma.db', size: 5_242_880, root: false },
    { path: '/var/lib/vaultwarden/db.sqlite3', size: 1_048_576, root: true },
  ],
  sqlite3: true,
}

const demoSQLite = (path: string): SQLiteInfo => ({
  path,
  size: path.includes('kuma') ? 5_242_880 : 1_048_576,
  journal: 'wal',
  root: path.startsWith('/var/lib'),
  backup: path.replace(/^.*\//, ''),
  tables: path.includes('kuma')
    ? [
        { name: 'heartbeat', rows: 48_210 },
        { name: 'monitor', rows: 12 },
        { name: 'user', rows: 1 },
      ]
    : [
        { name: 'ciphers', rows: 342 },
        { name: 'users', rows: 3 },
      ],
})

const demoRows = (table: string, offset: number): SQLiteRows => {
  const total = table === 'heartbeat' ? 48_210 : table === 'ciphers' ? 342 : 12
  const n = Math.max(0, Math.min(100, total - offset))
  return {
    columns: [
      { name: 'id', type: 'INTEGER', pk: true },
      { name: 'name', type: 'TEXT', pk: false },
      { name: 'status', type: 'INTEGER', pk: false },
      { name: 'note', type: 'TEXT', pk: false },
      { name: 'data', type: 'BLOB', pk: false },
    ],
    allColumns: 5,
    offset,
    more: offset + n < total,
    rows: Array.from({ length: n }, (_, i) => {
      const id = offset + i + 1
      return [
        { k: 'num', v: String(id) },
        { k: 'text', v: `${table}-${id}` },
        { k: 'num', v: String(id % 3 === 0 ? 0 : 1) },
        id % 4 === 0
          ? { k: 'null' }
          : id % 5 === 0
            ? { k: 'text', v: 'Lorem ipsum dolor sit amet, '.repeat(7).slice(0, 200), cut: true }
            : { k: 'text', v: 'ok' },
        id % 2 === 0 ? { k: 'blob', size: 1024 + id } : { k: 'null' },
      ]
    }),
  }
}

const q = (kind: DBKind, n: string) => (kind === 'mysql' ? `\`${n}\`` : `"${n}"`)

export const backend = hasRuntime
  ? {
      list: databaseList,
      createSQL: databaseCreateSQL,
      create: databaseCreate,
      dropSQL: databaseDropSQL,
      drop: databaseDrop,
      tunnel: (serverID: string, port: number) => tunnelStart(serverID, '127.0.0.1', port, ''),
      dump: databaseDump,
      dumps: databaseDumps,
      restoreCommand: databaseRestoreCommand,
      restore: databaseRestore,
      mongoConnect: databaseMongoConnect,
      mongoForget: databaseMongoForget,
      find: databaseFind,
      sqlite: databaseSQLite,
      sqliteBackup: databaseSQLiteBackup,
      sqliteRestoreLine: databaseSQLiteRestoreLine,
      sqliteRows: databaseSQLiteRows,
    }
  : {
      list: (_serverID: string) => Promise.resolve(demo),
      createSQL: (kind: DBKind, db: string, user: string) =>
        Promise.resolve(`CREATE DATABASE ${q(kind, db)};\nCREATE USER ${user} … '••••••••';\n`),
      create: (_serverID: string, kind: DBKind, db: string, user: string): Promise<DBLogin> => {
        demo = demo.map((e) =>
          e.kind === kind ? { ...e, dbs: [...e.dbs, { name: db, size: 0 }] } : e,
        )
        return Promise.resolve({ database: db, user, password: 'Xq7mK2pR9sTvW4yZ8aBcD3eF' })
      },
      dropSQL: (kind: DBKind, db: string) => Promise.resolve(`DROP DATABASE ${q(kind, db)};`),
      drop: (_serverID: string, kind: DBKind, db: string, _sql: string) => {
        demo = demo.map((e) =>
          e.kind === kind ? { ...e, dbs: e.dbs.filter((x) => x.name !== db) } : e,
        )
        return Promise.resolve()
      },
      tunnel: (_serverID: string, port: number) =>
        Promise.resolve({ localPort: port + 10000, moved: true }),
      dump: (_serverID: string, _kind: DBKind, db: string) =>
        Promise.resolve(`/root/db-backups/${db}-20261029T080000Z.sql.gz`),
      dumps: (_serverID: string, kind: DBKind | 'sqlite', db: string) =>
        Promise.resolve([
          `/root/db-backups/${db}-20261029T080000Z${kind === 'mongo' ? '.archive.gz' : kind === 'sqlite' ? '.sqlite.gz' : '.sql.gz'}`,
        ]),
      restoreCommand: (_serverID: string, _kind: DBKind, db: string, file: string) =>
        Promise.resolve(`gzip -t -- ${file} && gzip -dc -- ${file} | mysql --batch ${db}`),
      restore: (_serverID: string, _kind: DBKind, _db: string, _file: string, _command: string) =>
        Promise.resolve(),
      mongoConnect: (_serverID: string, uri: string) => {
        if (uri.includes(':wrong@'))
          return Promise.reject({ code: 'DB_AUTH_FAILED', message: 'Authentication failed.' })
        const login = uri.replace(/(mongodb(?:\+srv)?:\/\/[^:/@]*:)[^@]*@/, '$1••••@')
        demo = demo.map((e) =>
          e.kind === 'mongo' ? { ...e, needsAuth: false, login, dbs: demoMongo } : e,
        )
        return Promise.resolve()
      },
      mongoForget: (_serverID: string) => {
        demo = demo.map((e) =>
          e.kind === 'mongo' ? { ...e, needsAuth: true, login: undefined, dbs: [] } : e,
        )
        return Promise.resolve()
      },
      find: (_serverID: string) => new Promise<Found>((r) => setTimeout(() => r(demoFound), 600)),
      sqlite: (_serverID: string, file: string) => Promise.resolve(demoSQLite(file)),
      sqliteBackup: (_serverID: string, file: string) =>
        Promise.resolve(`/root/db-backups/${file.replace(/^.*\//, '')}-20261110T080000Z.sqlite.gz`),
      sqliteRestoreLine: (file: string, backup: string) =>
        Promise.resolve(
          `cp -p -- ${file} ${file}.before-restore && gzip -dc -- ${backup} > ${file}`,
        ),
      sqliteRows: (_serverID: string, _file: string, table: string, offset: number) =>
        new Promise<SQLiteRows>((r) => setTimeout(() => r(demoRows(table, offset)), 250)),
    }
