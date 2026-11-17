<script lang="ts">
  // Confirms a run on several servers: every server by name and the exact
  // command (docs/SECURITY.md, docs/adr/0021). A destructive-looking
  // command, or three servers or more, asks for the count to be typed.
  import type { FleetCheck } from '../../lib/api/fleet'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'

  interface Props {
    /** The servers and what runs on each; the dialog is closed while null. */
    rows: { name: string; command: string }[] | null
    check: FleetCheck
    onclose: () => void
    onconfirm: () => void
  }

  let { rows, check, onclose, onconfirm }: Props = $props()

  const r = t.servers.run
  let typed = $state('')

  const n = $derived(rows?.length ?? 0)
  /** One command for all: shown once, above the names. */
  const same = $derived(rows !== null && rows.every((x) => x.command === rows[0]?.command))
  const ok = $derived(!check.typed || typed.trim() === String(n))

  $effect(() => {
    if (rows) typed = ''
  })
</script>

<Dialog open={rows !== null} title={r.confirmTitle(n)} width={520} {onclose}>
  {#if rows}
    <p>{r.confirmBody}</p>
    {#if same}
      <pre class="command">{rows[0]?.command}</pre>
      <ul class="names">
        {#each rows as row, i (i)}<li>{row.name}</li>{/each}
      </ul>
    {:else}
      <ul class="per">
        {#each rows as row, i (i)}
          <li><span class="name">{row.name}</span><code>{row.command}</code></li>
        {/each}
      </ul>
    {/if}
    {#if check.destructive}
      <p class="warn"><Icon name="warning" size={13} />{r.destructive}</p>
    {/if}
    <p class="note">{r.sudoNote}</p>
    {#if check.typed}
      <label class="type">
        <span>{r.typeCount(n)}</span>
        <input
          class="input"
          inputmode="numeric"
          autocomplete="off"
          bind:value={typed}
          onkeydown={(e) => e.key === 'Enter' && ok && onconfirm()}
        />
      </label>
    {/if}
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus={!check.typed} onclick={onclose}
      >{t.common.cancel}</button
    >
    <button
      type="button"
      class="btn {check.destructive ? 'btn-danger' : 'btn-primary'}"
      disabled={!ok}
      onclick={onconfirm}>{r.confirmRun(n)}</button
    >
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    font-size: var(--text-sm);
  }

  .command,
  code {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    -webkit-user-select: text;
    user-select: text;
  }

  .command {
    margin: 0 0 10px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    max-height: 30vh;
    overflow: auto;
  }

  .names,
  .per {
    list-style: none;
    margin: 0 0 10px;
    padding: 6px 10px;
    max-height: 180px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .names {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
  }

  .per li {
    display: flex;
    gap: 10px;
    padding: 3px 0;
  }

  .name {
    flex-shrink: 0;
    font-weight: 600;
  }

  .warn {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--danger);
  }

  .note {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .type {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: var(--text-sm);
  }

  .type input {
    width: 80px;
  }

  .spacer {
    flex: 1;
  }
</style>
