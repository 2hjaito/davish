<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity'
  import { toUserError } from '../api/errors'
  import { listConfigHosts, type ConfigHost } from '../api/servers'
  import { t } from '../i18n/vi'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import Dialog from '../ui/Dialog.svelte'

  let hosts = $state<ConfigHost[] | null>(null)
  const chosen = new SvelteSet<string>()
  let error = $state('')
  let busy = $state(false)

  const available = $derived((hosts ?? []).filter((h) => !h.exists))

  // Re-read the file each time the dialog opens; it may have been edited.
  $effect(() => {
    if (!ui.importDialog) return
    hosts = null
    error = ''
    chosen.clear()
    listConfigHosts()
      .then((list) => (hosts = list))
      .catch((err: unknown) => {
        hosts = []
        error = toUserError(err).message
      })
  })

  function toggle(alias: string, on: boolean) {
    if (on) chosen.add(alias)
    else chosen.delete(alias)
  }

  function close() {
    ui.importDialog = false
  }

  async function submit() {
    busy = true
    error = ''
    try {
      const added = await servers.importHosts([...chosen])
      close()
      // Open the last imported host the user picked; its jump hosts came first.
      const last = added.findLast((s) => chosen.has(s.name)) ?? added.at(-1)
      if (last) servers.select(last.id)
    } catch (err) {
      error = toUserError(err).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog open={ui.importDialog} title={t.importDialog.title} onclose={close} width={480}>
  <p class="lead">{t.importDialog.lead}</p>
  {#if hosts === null}
    <p class="muted">{t.importDialog.loading}</p>
  {:else if hosts.length === 0 && !error}
    <p class="muted">{t.importDialog.empty}</p>
  {:else if hosts.length > 0}
    <label class="row all">
      <input
        type="checkbox"
        checked={available.length > 0 && chosen.size === available.length}
        disabled={available.length === 0}
        onchange={(e) => {
          chosen.clear()
          if (e.currentTarget.checked) for (const h of available) chosen.add(h.alias)
        }}
      />
      <span>{t.importDialog.selectAll}</span>
    </label>
    <ul class="list">
      {#each hosts as h (h.alias)}
        <li>
          <label class="row" class:exists={h.exists}>
            <input
              type="checkbox"
              disabled={h.exists}
              checked={h.exists || chosen.has(h.alias)}
              onchange={(e) => toggle(h.alias, e.currentTarget.checked)}
            />
            <span class="name">{h.alias}</span>
            <span class="addr mono">{h.user}@{h.hostName}{h.port === 22 ? '' : `:${h.port}`}</span>
            {#if h.exists}
              <span class="tag">{t.importDialog.exists}</span>
            {:else if h.proxyJump}
              <span class="tag">{t.importDialog.via(h.proxyJump)}</span>
            {/if}
          </label>
        </li>
      {/each}
    </ul>
    {#if available.some((h) => h.proxyJump)}
      <p class="muted">{t.importDialog.jumpNote}</p>
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={close}>{t.importDialog.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={busy || chosen.size === 0}
      onclick={submit}>{t.importDialog.submit(chosen.size)}</button
    >
  {/snippet}
</Dialog>

<style>
  .lead,
  .muted {
    margin: 0 0 10px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .list {
    list-style: none;
    margin: 0 0 10px;
    padding: 4px;
    max-height: 320px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 6px;
    font-size: var(--text-sm);
  }

  .all {
    padding-inline: 10px;
  }

  .exists {
    opacity: 0.55;
  }

  .name {
    font-weight: 600;
  }

  .addr {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--fg-muted);
  }

  .tag {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .error {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .spacer {
    flex: 1;
  }
</style>
