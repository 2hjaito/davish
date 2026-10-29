<script lang="ts">
  import { untrack } from 'svelte'

  import { copyText } from '../../lib/api/clipboard'
  import type { DBEngine, DBKind, DBLogin } from '../../lib/api/database'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'

  let { win }: AppProps = $props()
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

  const ports: Record<DBKind, number> = { mysql: 3306, postgres: 5432 }

  async function refresh() {
    loading = true
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

  async function tunnel(kind: DBKind) {
    try {
      const tn = await backend.tunnel(win.serverID, ports[kind])
      flash = d.tunnelOpen(ports[kind], tn.localPort)
    } catch (e) {
      flash = toUserError(e).message
    }
  }
</script>

{#if !connected}
  <Empty icon="server" title={d.offline} />
{:else if engines === null}
  <Empty icon="disk" title={error || d.loading} />
{:else}
  <div class="db">
    <header>
      <h3>{d.title}</h3>
      <span class="spacer"></span>
      <button class="tool" aria-label={d.refresh} disabled={loading} onclick={() => void refresh()}>
        <Icon name="refresh" size={15} />
      </button>
    </header>
    {#if flash}<p class="flash" role="status">{flash}</p>{/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}

    {#each engines as e (e.kind)}
      <section>
        <div class="head">
          <h4>{d.engines[e.kind]}</h4>
          {#if e.installed && !e.needsRoot && !e.error}
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
                  <td class="num muted">{formatBytes(db.size)}</td>
                  <td class="act">
                    <button class="link danger" onclick={() => void askDrop(e.kind, db.name)}
                      >{d.drop}</button
                    >
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </section>
    {/each}
  </div>
{/if}

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
  .db {
    height: 100%;
    padding: 12px 16px;
    overflow: auto;
    box-sizing: border-box;
  }

  header,
  .head {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  h3,
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
</style>
