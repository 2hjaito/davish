<script lang="ts">
  // The chosen Command log entries as a bash script: read it, then copy it
  // or save it (.claude/docs/adr/0022). Nothing runs from here.
  import { copyText } from '../../lib/api/clipboard'
  import { auditSaveScript, auditScript } from '../../lib/api/audit'
  import { toUserError } from '../../lib/api/errors'
  import type { AuditEntry } from '../../lib/api/events'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { mockScript } from './exportScript'

  interface Props {
    /** The entries to export; the dialog is closed while null. */
    entries: AuditEntry[] | null
    onclose: () => void
  }

  let { entries, onclose }: Props = $props()

  const x = t.audit.export
  let failed = $state(false)
  let script = $state('')
  let error = $state('')
  let note = $state('')

  const ids = $derived((entries ?? []).map((e) => e.id))
  const hasFailed = $derived((entries ?? []).some((e) => e.exit !== 0))

  $effect(() => {
    if (!entries) return
    const want = failed
    const list = entries
    note = ''
    error = ''
    if (!hasRuntime) {
      script = mockScript(
        list,
        (id) => {
          const s = servers.list.find((v) => v.id === id)
          return { name: s?.name ?? id, ssh: `ssh -t ${s?.user ?? 'user'}@${s?.host ?? 'host'}` }
        },
        want,
      )
      return
    }
    auditScript(
      list.map((e) => e.id),
      want,
    )
      .then((s) => (script = s))
      .catch((e: unknown) => (error = toUserError(e).message))
  })

  async function copy() {
    if (await copyText(script)) note = x.copied
  }

  async function save() {
    if (!hasRuntime) return
    try {
      const path = await auditSaveScript(ids, failed, x.saveTitle)
      if (path) note = x.saved(path)
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<Dialog open={entries !== null} title={x.title(entries?.length ?? 0)} width={640} {onclose}>
  <p class="lead">{x.lead}</p>
  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else}
    <pre class="script">{script}</pre>
  {/if}
  <p class="muted">{x.notIncluded}</p>
  {#if note}<p class="note" role="status">{note}</p>{/if}

  {#snippet footer()}
    {#if hasFailed}
      <label class="failed">
        <input type="checkbox" bind:checked={failed} />
        {x.failed}
      </label>
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{x.close}</button>
    <button type="button" class="btn" disabled={!script} onclick={() => void copy()}
      >{x.copy}</button
    >
    <button
      type="button"
      class="btn btn-primary"
      disabled={!script || !hasRuntime}
      onclick={() => void save()}>{x.save}</button
    >
  {/snippet}
</Dialog>

<style>
  .lead,
  .muted,
  .note,
  .error {
    margin: 0 0 10px;
    font-size: var(--text-sm);
  }

  .muted {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .note {
    color: var(--ok);
  }

  .error {
    color: var(--danger);
  }

  .script {
    margin: 0 0 10px;
    padding: 10px 12px;
    max-height: 46vh;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.5;
    white-space: pre;
    -webkit-user-select: text;
    user-select: text;
  }

  .failed {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-sm);
  }

  .spacer {
    flex: 1;
  }
</style>
