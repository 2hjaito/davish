<script lang="ts">
  import { untrack } from 'svelte'

  import { copyText } from '../../lib/api/clipboard'
  import type { DBEngine, DBKind, DBLogin, Found, SQLiteInfo } from '../../lib/api/database'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import { sqliteFiles } from '../../lib/stores/sqliteFiles.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import NotInstalled from '../../lib/ui/NotInstalled.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openFilesAt, openTerminalAt } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'

  let { win }: AppProps = $props()
  const ctx = getWindowContext()
  const d = t.database

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  let engines = $state.raw<DBEngine[] | null>(null)
  let loading = $state(false)
  let error = $state('')
  let flash = $state('')

  // New database form, then its login shown once.
  let creating = $state<DBKind | null>(null)
  let dbName = $state('')
  let userName = $state('')
  let createSQL = $state('')
  let login = $state.raw<DBLogin | null>(null)
  let copied = $state(false)

  // Drop: the user types the name to confirm.
  let dropping = $state.raw<{ kind: DBKind; name: string; sql: string } | null>(null)
  let typed = $state('')

  const ports: Record<DBKind, number> = { mysql: 3306, postgres: 5432, mongo: 27017 }
  // MongoDB only shows where it is installed: most servers have none.
  const shown = $derived((engines ?? []).filter((e) => e.kind !== 'mongo' || e.installed))

  async function refresh() {
    loading = true
    for (const p of sqliteFiles.of(win.serverID)) void loadSQLite(p)
    try {
      engines = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  $effect(() => {
    if (connected) untrack(() => void refresh())
  })
  // Coming back after installing an engine shows it.
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected && engines?.some((e) => !e.installed))
      untrack(() => void refresh())
    wasFocused = f
  })

  async function previewCreate() {
    if (!creating) return
    error = ''
    try {
      createSQL = await backend.createSQL(creating, dbName.trim(), userName.trim() || dbName.trim())
    } catch (e) {
      createSQL = ''
      error = toUserError(e).message
    }
  }

  async function create() {
    if (!creating) return
    try {
      login = await backend.create(
        win.serverID,
        creating,
        dbName.trim(),
        userName.trim() || dbName.trim(),
      )
      creating = null
      void refresh()
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function askDrop(kind: DBKind, name: string) {
    typed = ''
    try {
      dropping = { kind, name, sql: await backend.dropSQL(kind, name) }
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  async function drop() {
    const p = dropping
    if (!p || typed !== p.name) return
    dropping = null
    try {
      await backend.drop(win.serverID, p.kind, p.name, p.sql)
      flash = d.dropped(p.name)
    } catch (e) {
      flash = toUserError(e).message
    }
    void refresh()
  }

  let busy = $state('')
  let lastDump = $state('')

  async function dump(kind: DBKind, name: string) {
    busy = name
    flash = ''
    try {
      lastDump = await backend.dump(win.serverID, kind, name)
      flash = d.dumped(lastDump)
    } catch (e) {
      flash = toUserError(e).message
    } finally {
      busy = ''
    }
  }

  // Restore: pick a dump, see the exact command, type the name.
  let restoring = $state.raw<{ kind: DBKind; name: string; dumps: string[] } | null>(null)
  let restoreFile = $state('')
  let restoreCmd = $state('')
  let restoreTyped = $state('')

  async function askRestore(kind: DBKind, name: string) {
    restoreCmd = ''
    restoreTyped = ''
    try {
      const dumps = await backend.dumps(win.serverID, kind, name)
      restoring = { kind, name, dumps }
      restoreFile = dumps[0] ?? ''
      if (restoreFile) await planRestore()
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  async function planRestore() {
    if (!restoring) return
    try {
      restoreCmd = await backend.restoreCommand(
        win.serverID,
        restoring.kind,
        restoring.name,
        restoreFile.trim(),
      )
      error = ''
    } catch (e) {
      restoreCmd = ''
      error = toUserError(e).message
    }
  }

  async function restore() {
    const r = restoring
    if (!r || restoreTyped !== r.name || !restoreCmd) return
    restoring = null
    busy = r.name
    try {
      await backend.restore(win.serverID, r.kind, r.name, restoreFile.trim(), restoreCmd)
      flash = d.restored(r.name)
    } catch (e) {
      flash = toUserError(e).message
    } finally {
      busy = ''
    }
    void refresh()
  }

  // MongoDB login: a connection string, or one built from its fields.
  let mongoOpen = $state(false)
  let mongoMode = $state<'fields' | 'uri'>('fields')
  let mUri = $state('')
  let mHost = $state('127.0.0.1')
  let mPort = $state('27017')
  let mUser = $state('')
  let mPass = $state('')
  let mAuth = $state('admin')
  let mongoBusy = $state(false)
  let mongoError = $state('')
  const mongoURI = $derived(
    mongoMode === 'uri'
      ? mUri.trim()
      : `mongodb://${encodeURIComponent(mUser.trim())}:${encodeURIComponent(mPass)}@${mHost.trim()}:${mPort.trim()}/?authSource=${encodeURIComponent(mAuth.trim() || 'admin')}`,
  )
  const mongoReady = $derived(
    mongoMode === 'uri'
      ? mUri.trim() !== ''
      : mUser.trim() !== '' && mPass !== '' && mHost.trim() !== '',
  )

  function askMongo(port?: number) {
    findOpen = false
    mongoOpen = true
    mongoError = ''
    mPass = ''
    mUri = ''
    if (port) mPort = String(port)
  }

  function closeMongo() {
    mongoOpen = false
    mPass = ''
    mUri = ''
  }

  async function connectMongo() {
    mongoBusy = true
    mongoError = ''
    try {
      await backend.mongoConnect(win.serverID, mongoURI)
      closeMongo()
      flash = d.mongo.connected
      await refresh()
    } catch (e) {
      mongoError = toUserError(e).message
    } finally {
      mongoBusy = false
    }
  }

  async function forgetMongo() {
    try {
      await backend.mongoForget(win.serverID)
      flash = d.mongo.forgotten
      await refresh()
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  // Find: read-only, shown in a dialog to pick from.
  let findOpen = $state(false)
  let finding = $state(false)
  let found = $state.raw<Found | null>(null)
  let findError = $state('')

  async function find() {
    findOpen = true
    finding = true
    found = null
    findError = ''
    try {
      found = await backend.find(win.serverID)
    } catch (e) {
      findError = toUserError(e).message
    } finally {
      finding = false
    }
  }

  // SQLite files, kept per server on this computer.
  const files = $derived(sqliteFiles.of(win.serverID))
  /** Each file's tables, or why it could not be read. */
  let infos = $state.raw<Record<string, SQLiteInfo | string>>({})

  async function loadSQLite(path: string) {
    try {
      const info = await backend.sqlite(win.serverID, path)
      infos = { ...infos, [path]: info }
    } catch (e) {
      infos = { ...infos, [path]: toUserError(e).message }
    }
  }

  $effect(() => {
    if (!connected) return
    for (const p of files) if (!(p in untrack(() => infos))) untrack(() => void loadSQLite(p))
  })

  function addSQLite(path: string) {
    sqliteFiles.add(win.serverID, path)
    void loadSQLite(path)
  }

  let addOpen = $state(false)
  let addPath = $state('')

  async function backupSQLite(path: string) {
    busy = path
    flash = ''
    try {
      lastDump = await backend.sqliteBackup(win.serverID, path)
      flash = d.sqlite.backedUp(lastDump)
    } catch (e) {
      flash = toUserError(e).message
    } finally {
      busy = ''
    }
  }

  // Restoring SQLite only types its line into a terminal.
  let sqlRestore = $state.raw<{ path: string; backups: string[] } | null>(null)
  let sqlBackup = $state('')
  let sqlLine = $state('')

  async function askSQLiteRestore(info: SQLiteInfo) {
    sqlLine = ''
    try {
      const backups = await backend.dumps(win.serverID, 'sqlite', info.backup)
      sqlRestore = { path: info.path, backups }
      sqlBackup = backups[0] ?? ''
      if (sqlBackup) await planSQLiteRestore()
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  async function planSQLiteRestore() {
    if (!sqlRestore) return
    try {
      sqlLine = await backend.sqliteRestoreLine(sqlRestore.path, sqlBackup)
    } catch (e) {
      sqlLine = ''
      flash = toUserError(e).message
    }
  }

  const baseName = (p: string) => p.replace(/^.*\//, '')
  const dir = (p: string) => p.replace(/\/[^/]*$/, '') || '/'

  async function tunnel(kind: DBKind) {
    try {
      const tn = await backend.tunnel(win.serverID, ports[kind])
      flash = d.tunnelOpen(ports[kind], tn.localPort)
    } catch (e) {
      flash = toUserError(e).message
    }
  }
</script>

<TitleBar>
  <span class="title">{d.title}</span>
  <button
    class="tool find"
    disabled={!connected || finding}
    title={d.found.title}
    onclick={() => void find()}
  >
    <Icon name="search" size={13} strokeWidth={2.2} />
    {d.find}
  </button>
  <button
    class="tool"
    aria-label={d.refresh}
    class:spin={loading}
    disabled={!connected || loading}
    onclick={() => void refresh()}
  >
    <Icon name="refresh" size={14} strokeWidth={2.2} />
  </button>
</TitleBar>

{#if !connected}
  <Empty icon="server" title={d.offline} />
{:else if engines === null}
  <Empty icon="disk" title={error || d.loading} />
{:else}
  <div class="db">
    {#if flash}
      <p class="flash" role="status">
        {flash}
        {#if lastDump && flash.includes(lastDump)}
          <button class="link" onclick={() => openFilesAt(win.serverID, dir(lastDump))}
            >{d.showInFiles}</button
          >
        {/if}
      </p>
    {/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <details class="how">
      <summary><Icon name="info" size={13} /> {d.how.title}</summary>
      <ul>
        {#each d.how.items as item (item)}<li>{item}</li>{/each}
      </ul>
    </details>

    {#each shown as e (e.kind)}
      <section>
        <div class="head">
          <h4>{d.engines[e.kind]}</h4>
          {#if e.installed && !e.needsRoot && !e.needsAuth && !e.error}
            <button type="button" class="btn" onclick={() => void tunnel(e.kind)}>
              <Icon name="link" size={13} />
              {d.tunnel}
            </button>
            <button
              type="button"
              class="btn btn-primary"
              onclick={() => {
                creating = e.kind
                dbName = ''
                userName = ''
                createSQL = ''
                error = ''
              }}
            >
              <Icon name="plus" size={13} />
              {d.create}
            </button>
          {/if}
        </div>
        {#if !e.installed}
          <p class="muted">{d.notInstalled}</p>
          <NotInstalled
            compact
            serverID={win.serverID}
            tool={e.kind === 'mysql' ? 'mariadb' : 'postgres'}
            name={d.engines[e.kind]}
            onretry={() => void refresh()}
          />
        {:else if e.needsRoot}
          <p class="muted">{d.needsRoot}</p>
          <button
            type="button"
            class="btn"
            onclick={() => openTerminalAt(win.serverID, undefined, e.client)}
          >
            <Icon name="terminal" size={13} />
            {d.openClient(e.client)}
          </button>
        {:else if e.needsAuth}
          <p class="muted">{d.mongo.needsAuth}</p>
          {#if e.error}<p class="error">{e.error}</p>{/if}
          <button type="button" class="btn btn-primary" onclick={() => askMongo()}>
            <Icon name="key" size={13} />
            {d.mongo.login}
          </button>
        {:else if e.error}
          <p class="error">{e.error}</p>
        {:else if e.dbs.length === 0}
          <p class="muted">{d.none}</p>
        {:else}
          <table>
            <tbody>
              {#each e.dbs as db (db.name)}
                <tr>
                  <td class="mono">{db.name}</td>
                  {#if e.kind === 'mongo'}
                    <td class="num muted">{d.collections(db.collections ?? 0)}</td>
                  {/if}
                  <td class="num muted">{formatBytes(db.size)}</td>
                  <td class="act">
                    <button
                      class="link"
                      disabled={busy === db.name}
                      onclick={() => void dump(e.kind, db.name)}
                      >{busy === db.name ? d.working : d.dump}</button
                    >
                    <button
                      class="link"
                      disabled={busy === db.name}
                      onclick={() => void askRestore(e.kind, db.name)}>{d.restore}</button
                    >
                    <button class="link danger" onclick={() => void askDrop(e.kind, db.name)}
                      >{d.drop}</button
                    >
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
        {#if e.login}
          <p class="muted small">
            {d.mongo.loggedIn(e.login)}
            <button class="link" onclick={() => void forgetMongo()}>{d.mongo.forget}</button>
          </p>
        {/if}
      </section>
    {/each}

    <section>
      <div class="head">
        <h4>{d.sqlite.title}</h4>
        <button
          type="button"
          class="btn"
          onclick={() => {
            addOpen = true
            addPath = ''
          }}
        >
          <Icon name="plus" size={13} />
          {d.sqlite.add}
        </button>
      </div>
      {#if files.length === 0}
        <p class="muted">{d.sqlite.none}</p>
      {:else}
        <table>
          <tbody>
            {#each files as path (path)}
              {@const info = infos[path]}
              <tr>
                <td class="file">
                  <span class="mono" title={path}>{baseName(path)}</span>
                  <small class="mono">{dir(path)}</small>
                  {#if typeof info === 'string'}
                    <small class="error">{info}</small>
                  {:else if info}
                    <details class="tables">
                      <summary>
                        {d.sqlite.tables(info.tables.length)} · {formatBytes(info.size)} · {info.journal}
                        {#if info.root}· {d.sqlite.rootOnly}{/if}
                      </summary>
                      <ul>
                        {#each info.tables as tb (tb.name)}
                          <li>
                            <span class="mono">{tb.name}</span>
                            <small>{d.sqlite.rows(tb.rows)}</small>
                          </li>
                        {/each}
                      </ul>
                    </details>
                  {:else}
                    <small>{d.loading}</small>
                  {/if}
                </td>
                <td class="act">
                  {#if info && typeof info !== 'string'}
                    <button
                      class="link"
                      disabled={busy === path}
                      onclick={() => void backupSQLite(path)}
                      >{busy === path ? d.working : d.sqlite.backup}</button
                    >
                    <button class="link" onclick={() => void askSQLiteRestore(info)}
                      >{d.restore}</button
                    >
                  {/if}
                  <button
                    class="link"
                    title={d.showInFiles}
                    aria-label={d.showInFiles}
                    onclick={() => openFilesAt(win.serverID, dir(path))}
                  >
                    <Icon name="folder" size={13} />
                  </button>
                  <button
                    class="link"
                    title={d.sqlite.remove}
                    aria-label={d.sqlite.remove}
                    onclick={() => sqliteFiles.remove(win.serverID, path)}
                  >
                    <Icon name="close" size={13} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </section>
  </div>
{/if}

<Dialog open={mongoOpen} title={d.mongo.title} width={500} onclose={closeMongo}>
  <div class="tabs" role="tablist">
    <button
      type="button"
      role="tab"
      aria-selected={mongoMode === 'fields'}
      class:on={mongoMode === 'fields'}
      onclick={() => (mongoMode = 'fields')}>{d.mongo.byFields}</button
    >
    <button
      type="button"
      role="tab"
      aria-selected={mongoMode === 'uri'}
      class:on={mongoMode === 'uri'}
      onclick={() => (mongoMode = 'uri')}>{d.mongo.byUri}</button
    >
  </div>
  {#if mongoMode === 'uri'}
    <label class="field">
      <span>{d.mongo.uri}</span>
      <input
        class="input mono"
        type="password"
        bind:value={mUri}
        placeholder="mongodb://user:password@127.0.0.1:27017/?authSource=admin"
        spellcheck="false"
        autocomplete="off"
      />
    </label>
  {:else}
    <div class="row">
      <label class="field grow">
        <span>{d.mongo.host}</span>
        <input class="input" bind:value={mHost} spellcheck="false" />
      </label>
      <label class="field port">
        <span>{d.mongo.port}</span>
        <input class="input" bind:value={mPort} inputmode="numeric" />
      </label>
    </div>
    <div class="row">
      <label class="field">
        <span>{d.mongo.user}</span>
        <input class="input" bind:value={mUser} spellcheck="false" autocomplete="off" />
      </label>
      <label class="field">
        <span>{d.mongo.password}</span>
        <input class="input" type="password" bind:value={mPass} autocomplete="off" />
      </label>
    </div>
    <label class="field">
      <span>{d.mongo.authSource}</span>
      <input class="input" bind:value={mAuth} spellcheck="false" />
    </label>
  {/if}
  <p class="note"><Icon name="lock" size={13} /> {d.mongo.keep}</p>
  <details class="how">
    <summary>{d.how.title}</summary>
    <ul>
      {#each d.how.items.slice(0, 3) as item (item)}<li>{item}</li>{/each}
    </ul>
  </details>
  {#if mongoError}<p class="error" role="alert">{mongoError}</p>{/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={closeMongo}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!mongoReady || mongoBusy}
      onclick={() => void connectMongo()}>{mongoBusy ? d.mongo.connecting : d.mongo.connect}</button
    >
  {/snippet}
</Dialog>

<Dialog open={findOpen} title={d.found.title} width={560} onclose={() => (findOpen = false)}>
  <p class="muted small">{d.found.hint}</p>
  {#if finding}
    <p class="muted">{d.found.scanning}</p>
  {:else if findError}
    <p class="error">{findError}</p>
  {:else if found}
    <h5>{d.found.running}</h5>
    {#if found.running.length === 0}
      <p class="muted">{d.found.noRunning}</p>
    {:else}
      <ul class="found">
        {#each found.running as r (r.kind)}
          {@const e = engines?.find((x) => x.kind === r.kind)}
          <li>
            <span>
              <strong>{d.engines[r.kind]}</strong>
              <small>
                {#if r.port}{d.found.port(r.port)}{/if}
                {#if r.bind}· {r.bind}{/if}
                {#if r.auth}· {r.auth === 'on' ? d.found.authOn : d.found.authOff}{/if}
              </small>
            </span>
            {#if r.kind === 'mongo' && e?.needsAuth}
              <button class="btn" onclick={() => askMongo(r.port)}>{d.mongo.login}</button>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
    <h5>{d.found.files}</h5>
    {#if !found.sqlite3}<p class="warn">{d.found.noSqlite3}</p>{/if}
    {#if found.sqlite.length === 0}
      <p class="muted">{d.found.noFiles}</p>
    {:else}
      <ul class="found">
        {#each found.sqlite as f (f.path)}
          <li>
            <span>
              <span class="mono">{f.path}</span>
              <small
                >{f.root
                  ? `${formatBytes(f.size)} · ${d.found.rootOnly}`
                  : formatBytes(f.size)}</small
              >
            </span>
            {#if files.includes(f.path)}
              <small class="muted">{d.found.added}</small>
            {:else}
              <button class="btn" disabled={!found.sqlite3} onclick={() => addSQLite(f.path)}
                >{d.found.add}</button
              >
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (findOpen = false)}>{d.done}</button>
  {/snippet}
</Dialog>

<Dialog open={addOpen} title={d.sqlite.addTitle} width={480} onclose={() => (addOpen = false)}>
  <label class="field">
    <span>{d.sqlite.path}</span>
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input mono"
      bind:value={addPath}
      placeholder="/var/lib/app/data.db"
      spellcheck="false"
      autofocus
    />
  </label>
  <small>{d.sqlite.pathHint}</small>
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (addOpen = false)}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!addPath.trim().startsWith('/')}
      onclick={() => {
        addSQLite(addPath.trim())
        addOpen = false
      }}>{d.found.add}</button
    >
  {/snippet}
</Dialog>

<Dialog
  open={sqlRestore !== null}
  title={sqlRestore ? d.sqlite.restoreTitle(baseName(sqlRestore.path)) : ''}
  width={540}
  onclose={() => (sqlRestore = null)}
>
  {#if sqlRestore}
    {#if sqlRestore.backups.length === 0}
      <p class="muted">{d.sqlite.noBackups}</p>
    {:else}
      <label class="field">
        <span>{d.restoreFile}</span>
        <select class="input" bind:value={sqlBackup} onchange={() => void planSQLiteRestore()}>
          {#each sqlRestore.backups as f (f)}<option value={f}>{f}</option>{/each}
        </select>
      </label>
      <p class="warn"><Icon name="warning" size={14} /> {d.sqlite.restoreBody}</p>
      {#if sqlLine}<pre class="code">{sqlLine}</pre>{/if}
    {/if}
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (sqlRestore = null)}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!sqlLine}
      onclick={() => {
        openTerminalAt(win.serverID, undefined, sqlLine)
        sqlRestore = null
      }}
    >
      <Icon name="terminal" size={13} />
      {d.sqlite.toTerminal}
    </button>
  {/snippet}
</Dialog>

<Dialog
  open={creating !== null}
  title={creating ? d.createTitle(d.engines[creating]) : ''}
  width={500}
  onclose={() => (creating = null)}
>
  <div class="row">
    <label class="field">
      <span>{d.dbName}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input"
        bind:value={dbName}
        oninput={() => (createSQL = '')}
        autofocus
        spellcheck="false"
      />
    </label>
    <label class="field">
      <span>{d.userName}</span>
      <input
        class="input"
        bind:value={userName}
        oninput={() => (createSQL = '')}
        placeholder={dbName}
        spellcheck="false"
      />
    </label>
  </div>
  <small>{d.namesHint}</small>
  {#if createSQL}
    <p>{d.createBody}</p>
    <pre class="code">{createSQL}</pre>
  {/if}
  {#if error}<p class="error">{error}</p>{/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (creating = null)}>{t.common.cancel}</button>
    {#if createSQL}
      <button type="button" class="btn btn-primary" onclick={() => void create()}>{d.create}</button
      >
    {:else}
      <button
        type="button"
        class="btn btn-primary"
        disabled={!dbName.trim()}
        onclick={() => void previewCreate()}>{d.preview}</button
      >
    {/if}
  {/snippet}
</Dialog>

<Dialog
  open={login !== null}
  title={d.loginTitle}
  width={480}
  onclose={() => ((login = null), (copied = false))}
>
  {#if login}
    <p class="warn"><Icon name="warning" size={14} /> {d.loginOnce}</p>
    <dl>
      <dt>{d.dbName}</dt>
      <dd class="mono">{login.database}</dd>
      <dt>{d.userName}</dt>
      <dd class="mono">{login.user}</dd>
      <dt>{d.password}</dt>
      <dd class="mono">{login.password}</dd>
    </dl>
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button
      type="button"
      class="btn"
      onclick={async () => {
        if (login) copied = await copyText(login.password)
      }}>{copied ? d.copied : d.copyPassword}</button
    >
    <button type="button" class="btn btn-primary" onclick={() => ((login = null), (copied = false))}
      >{d.done}</button
    >
  {/snippet}
</Dialog>

<Dialog
  open={restoring !== null}
  title={restoring ? d.restoreTitle(restoring.name) : ''}
  width={540}
  onclose={() => (restoring = null)}
>
  {#if restoring}
    <label class="field">
      <span>{d.restoreFile}</span>
      {#if restoring.dumps.length > 0}
        <select class="input" bind:value={restoreFile} onchange={() => void planRestore()}>
          {#each restoring.dumps as f (f)}<option value={f}>{f}</option>{/each}
        </select>
      {:else}
        <input
          class="input"
          bind:value={restoreFile}
          onchange={() => void planRestore()}
          placeholder="/root/db-backups/{restoring.name}{restoring.kind === 'mongo'
            ? '.archive.gz'
            : '.sql.gz'}"
          spellcheck="false"
        />
        <small>{restoring.kind === 'mongo' ? d.noDumpsMongo : d.noDumps}</small>
      {/if}
    </label>
    {#if restoreCmd}
      <p class="warn"><Icon name="warning" size={14} /> {d.restoreBody}</p>
      <pre class="code">{restoreCmd}</pre>
      <label class="field">
        <span>{d.typeName(restoring.name)}</span>
        <input class="input" bind:value={restoreTyped} spellcheck="false" />
      </label>
    {/if}
    {#if error}<p class="error">{error}</p>{/if}
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (restoring = null)}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-danger"
      disabled={!restoreCmd || restoreTyped !== restoring?.name}
      onclick={() => void restore()}>{d.restore}</button
    >
  {/snippet}
</Dialog>

<Dialog
  open={dropping !== null}
  title={dropping ? d.dropTitle(dropping.name) : ''}
  width={480}
  onclose={() => (dropping = null)}
>
  {#if dropping}
    <p>{d.dropBody}</p>
    <pre class="code">{dropping.sql}</pre>
    <label class="field">
      <span>{d.typeName(dropping.name)}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={typed} autofocus spellcheck="false" />
    </label>
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (dropping = null)}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-danger"
      disabled={typed !== dropping?.name}
      onclick={() => void drop()}>{d.drop}</button
    >
  {/snippet}
</Dialog>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .tool {
    border: 0;
    background: none;
    padding: 3px;
    display: flex;
    color: var(--fg-muted);
  }

  .tool.spin :global(svg) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      rotate: 360deg;
    }
  }

  .db {
    flex: 1;
    min-height: 0;
    padding: 12px 16px;
    overflow: auto;
    box-sizing: border-box;
    background: var(--content-bg);
  }

  .link {
    border: 0;
    background: none;
    padding: 0 4px;
    color: var(--accent);
    font-size: var(--text-sm);
  }

  .link:disabled {
    opacity: 0.5;
  }

  .link.danger {
    color: var(--danger);
  }

  .head {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  h4 {
    margin: 0;
  }

  section {
    margin-top: 18px;
  }

  .head h4 {
    flex: 1;
  }

  .spacer {
    flex: 1;
  }

  table {
    width: 100%;
    margin-top: 8px;
    border-collapse: collapse;
  }

  td {
    padding: 6px 4px;
    border-bottom: 1px solid var(--border);
  }

  .num,
  .act {
    text-align: right;
    white-space: nowrap;
  }

  .mono {
    font-family: var(--font-mono);
    user-select: text;
  }

  .muted,
  small {
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  p {
    margin: 8px 0;
    line-height: 1.45;
  }

  .flash {
    padding: 6px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .error,
  .danger {
    color: var(--danger);
  }

  .warn {
    display: flex;
    gap: 6px;
    color: var(--warn);
  }

  .row {
    display: flex;
    gap: 12px;
  }

  .field {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 4px;
    margin: 0 0 8px;
  }

  .code {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    user-select: text;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 6px 12px;
    margin: 10px 0 0;
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
  }

  .tool.find {
    gap: 4px;
    align-items: center;
    font-size: var(--text-sm);
    padding: 3px 6px;
  }

  .how {
    margin-top: 10px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .how summary {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    cursor: default;
    color: var(--accent);
  }

  .how ul {
    margin: 6px 0 0;
    padding-left: 18px;
    line-height: 1.5;
  }

  .how li + li {
    margin-top: 4px;
  }

  .small {
    font-size: var(--text-sm);
  }

  .file {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .file small.mono {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tables summary {
    cursor: default;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .tables ul {
    margin: 4px 0 2px;
    padding-left: 16px;
    font-size: var(--text-sm);
  }

  .act {
    vertical-align: top;
  }

  .tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 10px;
  }

  .tabs button {
    border: 0;
    padding: 4px 10px;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .tabs button.on {
    background: var(--fill-soft);
    color: var(--fg);
  }

  .grow {
    flex: 2;
  }

  .port {
    flex: 0 0 90px;
  }

  .note {
    display: flex;
    gap: 6px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  h5 {
    margin: 12px 0 6px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  .found {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .found li {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 0;
    border-bottom: 1px solid var(--border);
  }

  .found li > span {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow-wrap: anywhere;
  }
</style>
