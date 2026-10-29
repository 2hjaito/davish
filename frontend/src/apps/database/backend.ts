// The database calls, or demo data in the plain browser preview.
import {
  databaseCreate,
  databaseCreateSQL,
  databaseDrop,
  databaseDropSQL,
  databaseList,
  type DBEngine,
  type DBKind,
  type DBLogin,
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
]

const q = (kind: DBKind, n: string) => (kind === 'mysql' ? `\`${n}\`` : `"${n}"`)

export const backend = hasRuntime
  ? {
      list: databaseList,
      createSQL: databaseCreateSQL,
      create: databaseCreate,
      dropSQL: databaseDropSQL,
      drop: databaseDrop,
      tunnel: (serverID: string, port: number) => tunnelStart(serverID, '127.0.0.1', port, ''),
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
    }
