<script lang="ts">
  import { untrack } from 'svelte'

  import { copyText } from '../../lib/api/clipboard'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import { snippets } from '../../lib/stores/snippets.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { looksSecret, type Snippet, visibleOn } from './snippets'

  let { win }: AppProps = $props()

  const s = t.snippets

  let query = $state('')
  // The form: null while closed, '' for a new command, else the id edited.
  let editing = $state<string | null>(null)
  let name = $state('')
  let command = $state('')
  let shared = $state(false)
  let removing = $state<string | null>(null)
  let copied = $state<string | null>(null)
  let nameInput = $state<HTMLInputElement>()

  const server = $derived(servers.list.find((x) => x.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const rows = $derived(visibleOn(snippets.list, win.serverID, query))
  const secret = $derived(looksSecret(command))
  const canSave = $derived(name.trim() !== '' && command.trim() !== '')

  function open(x?: Snippet, draft?: { command: string; serverID: string }) {
    editing = x?.id ?? ''
    name = x?.name ?? (draft ? draft.command.slice(0, 40) : '')
    command = x?.command ?? draft?.command ?? ''
    shared = x ? x.serverID === '' : false
    requestAnimationFrame(() => nameInput?.select())
  }

  function close() {
    editing = null
  }

  function submit(e: SubmitEvent) {
    e.preventDefault()
    if (!canSave || editing === null) return
    const value = {
      name: name.trim(),
      command: command.trim(),
      serverID: shared ? '' : win.serverID,
    }
    if (editing) snippets.update(editing, value)
    else snippets.add(value)
    close()
  }

  // Command log's "Save as a command" hands a draft over.
  $effect(() => {
    const d = snippets.draft
    if (!d) return
    untrack(() => {
      snippets.draft = null
      open(undefined, d)
    })
  })

  function remove(id: string) {
    if (removing !== id) {
      removing = id
      setTimeout(() => removing === id && (removing = null), 3000)
      return
    }
    removing = null
    snippets.remove(id)
  }

  async function copy(x: Snippet) {
    if (await copyText(x.command)) {
      copied = x.id
      setTimeout(() => copied === x.id && (copied = null), 1500)
    }
  }
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <button class="tool" aria-label={s.add} title={s.add} onclick={() => open()}>
    <Icon name="plus" size={15} strokeWidth={2.2} />
  </button>
</TitleBar>

<div class="body">
  <div class="bar">
    <label class="search">
      <Icon name="search" size={13} strokeWidth={2.2} />
      <input
        type="text"
        placeholder={s.search}
        aria-label={s.search}
        bind:value={query}
        onkeydown={(e) => e.key === 'Escape' && (query = '')}
      />
    </label>
  </div>

  {#if editing !== null}
    <form class="form" onsubmit={submit}>
      <label class="field">
        <span>{s.name}</span>
        <input
          bind:this={nameInput}
          class="input"
          type="text"
          bind:value={name}
          placeholder={s.namePlaceholder}
        />
      </label>
      <label class="field">
        <span>{s.command}</span>
        <textarea class="input mono" rows="3" bind:value={command} spellcheck="false"></textarea>
      </label>
      {#if secret}
        <p class="secret"><Icon name="warning" size={13} />{s.secret}</p>
      {/if}
      <fieldset class="scope">
        <legend>{s.scope}</legend>
        <label
          ><input type="radio" bind:group={shared} value={false} />
          {s.thisServer(server?.name ?? win.serverID)}</label
        >
        <label><input type="radio" bind:group={shared} value={true} /> {s.everyServer}</label>
      </fieldset>
      <div class="actions">
        <button type="button" class="btn" onclick={close}>{s.cancel}</button>
        <button type="submit" class="btn btn-primary" disabled={!canSave}>{s.save}</button>
      </div>
    </form>
  {/if}

  {#if snippets.list.length === 0 && editing === null}
    <Empty icon="terminal" title={s.empty} hint={s.emptyHint} />
  {:else if rows.length === 0 && editing === null}
    <Empty icon="search" title={s.noMatch} />
  {:else}
    <ul class="rows">
      {#each rows as x (x.id)}
        <li class="row">
          <div class="info">
            <span class="name"
              >{x.name}{#if x.serverID === ''}<span class="tag">{s.shared}</span>{/if}</span
            >
            <code class="cmd" title={x.command}>{x.command}</code>
          </div>
          <button class="icon" aria-label={s.edit} title={s.edit} onclick={() => open(x)}>
            <Icon name="rename" size={13} strokeWidth={2.2} />
          </button>
          <button
            class="icon"
            class:danger={removing === x.id}
            aria-label={removing === x.id ? s.removeConfirm : s.remove}
            title={removing === x.id ? s.removeConfirm : s.remove}
            onclick={() => remove(x.id)}
          >
            <Icon name="trash" size={13} strokeWidth={2.2} />
          </button>
          <button
            class="icon"
            aria-label={s.copy}
            title={copied === x.id ? s.copied : s.copy}
            onclick={() => void copy(x)}
          >
            <Icon name={copied === x.id ? 'check' : 'copy'} size={13} strokeWidth={2.2} />
          </button>
          <button
            class="btn type"
            disabled={!connected}
            title={connected ? s.typeHint : s.offline}
            onclick={() => openTerminalAt(win.serverID, undefined, x.command)}
          >
            <Icon name="terminal" size={13} strokeWidth={2.2} />{s.type}
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  <p class="footer">{connected ? s.typeHint : s.offline}</p>
</div>

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

  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
  }

  .bar {
    padding: 10px 14px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 9px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .search input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: none;
    background: transparent;
    font-size: var(--text-sm);
    color: var(--fg);
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin: 12px 14px 4px;
    padding: 12px 14px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
  }

  .secret {
    display: flex;
    gap: 6px;
    margin: 0;
    font-size: var(--text-xs);
    color: var(--fg);
  }

  .secret :global(svg) {
    flex-shrink: 0;
    margin-top: 1px;
    color: var(--warn);
  }

  .scope {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 16px;
    margin: 0;
    padding: 0;
    border: 0;
    font-size: var(--text-sm);
  }

  .scope legend {
    float: left;
    margin-right: 6px;
    color: var(--fg-label);
    font-weight: 500;
  }

  .scope label {
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .rows {
    flex: 1;
    min-height: 0;
    overflow: auto;
    margin: 0;
    padding: 6px 8px;
    list-style: none;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 7px 8px;
    border-radius: 7px;
  }

  .row:nth-child(even) {
    background: var(--row-stripe);
  }

  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .name {
    font-size: var(--text-sm);
    font-weight: 600;
  }

  .tag {
    margin-left: 6px;
    font-size: var(--text-xxs);
    font-weight: 400;
    padding: 0 5px;
    border-radius: 4px;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .cmd {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .icon {
    display: flex;
    padding: 5px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--fg-muted);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .row:hover .icon,
  .icon:focus-visible,
  .icon.danger {
    opacity: 1;
  }

  .icon.danger {
    color: var(--danger);
  }

  .type {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-left: 4px;
    padding: 4px 10px;
    font-size: var(--text-xs);
    white-space: nowrap;
  }

  .footer {
    margin: 0;
    padding: 6px 14px 8px;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
    border-top: 1px solid var(--divider-soft);
  }
</style>
