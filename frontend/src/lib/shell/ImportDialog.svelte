<script lang="ts">
  // Import servers from ~/.ssh/config or an Ansible inventory (INI). Both
  // files are only read; nothing on any server is touched.
  import { untrack } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'
  import { toUserError } from '../api/errors'
  import { hasRuntime, openURL } from '../api/runtime'
  import {
    defaultInventory,
    listConfigHosts,
    listInventory,
    pickInventory,
    type ConfigHost,
    type InventoryListing,
  } from '../api/servers'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import { address, groupCounts, issueText, mockInventory, visibleHosts } from './inventory'

  const ISSUES_URL = 'https://github.com/2hjaito/davish/issues/new'
  const d = t.importDialog

  type Source = 'ssh' | 'ansible'
  let source = $state<Source>('ssh')

  let hosts = $state<ConfigHost[] | null>(null)
  const chosen = new SvelteSet<string>()
  let error = $state('')
  let busy = $state(false)

  const available = $derived((hosts ?? []).filter((h) => !h.exists))

  // The inventory tab.
  let inv = $state.raw<InventoryListing | null>(null)
  let invPath = $state('')
  let invLoading = $state(false)
  let invError = $state('')
  /** The error is about YAML: offer to ask for it. */
  let invYAML = $state(false)
  let group = $state('')
  const picked = new SvelteSet<string>()

  const shown = $derived(visibleHosts(inv?.hosts ?? [], group))
  const shownFree = $derived(shown.filter((h) => !h.exists))
  const counts = $derived(groupCounts(inv?.hosts ?? []))

  // Re-read the files each time the dialog opens; they may have been edited.
  $effect(() => {
    if (!ui.importDialog) return
    hosts = null
    error = ''
    chosen.clear()
    picked.clear()
    untrack(() => invPath && void loadInventory(invPath))
    // The browser preview has no ~/.ssh/config to read.
    if (!hasRuntime) {
      hosts = []
      return
    }
    listConfigHosts()
      .then((list) => (hosts = list))
      .catch((err: unknown) => {
        hosts = []
        error = toUserError(err).message
      })
  })

  async function showAnsible() {
    source = 'ansible'
    error = ''
    if (invPath || inv) return
    if (!hasRuntime) {
      inv = mockInventory()
      invPath = inv.path
      return
    }
    const p = await defaultInventory().catch(() => '')
    if (p) await loadInventory(p)
  }

  async function choose() {
    if (!hasRuntime) return
    const p = await pickInventory(d.chooseTitle).catch(() => '')
    if (p) await loadInventory(p)
  }

  async function loadInventory(p: string) {
    if (!hasRuntime) return
    invPath = p
    invLoading = true
    invError = ''
    invYAML = false
    try {
      const next = await listInventory(p)
      inv = next
      if (group && !next.groups.includes(group)) group = ''
      for (const name of [...picked]) {
        if (!next.hosts.some((h) => h.name === name && !h.exists)) picked.delete(name)
      }
    } catch (err) {
      const ue = toUserError(err)
      inv = null
      picked.clear()
      invError = ue.message
      invYAML = ue.code === 'INVENTORY_YAML_UNSUPPORTED'
    } finally {
      invLoading = false
    }
  }

  function toggle(set: SvelteSet<string>, key: string, on: boolean) {
    if (on) set.add(key)
    else set.delete(key)
  }

  function close() {
    ui.importDialog = false
  }

  const count = $derived(source === 'ssh' ? chosen.size : picked.size)

  async function submit() {
    // The browser preview cannot save servers.
    if (!hasRuntime) return close()
    busy = true
    error = ''
    const want = new Set(source === 'ssh' ? chosen : picked)
    try {
      const added =
        source === 'ssh'
          ? await servers.importHosts([...want])
          : await servers.importInventory(invPath, [...want])
      close()
      // Open the last imported host the user picked; its jump hosts came first.
      const last = added.findLast((s) => want.has(s.name)) ?? added.at(-1)
      if (last) servers.select(last.id)
    } catch (err) {
      error = toUserError(err).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog open={ui.importDialog} title={d.title} onclose={close} width={560}>
  <div class="segmented" role="radiogroup" aria-label={d.source}>
    <button
      type="button"
      role="radio"
      aria-checked={source === 'ssh'}
      onclick={() => {
        source = 'ssh'
        error = ''
      }}>{d.tabSSH}</button
    >
    <button
      type="button"
      role="radio"
      aria-checked={source === 'ansible'}
      onclick={() => void showAnsible()}>{d.tabAnsible}</button
    >
  </div>

  {#if source === 'ssh'}
    <p class="lead">{d.lead}</p>
    {#if hosts === null}
      <p class="muted">{d.loading}</p>
    {:else if hosts.length === 0 && !error}
      <p class="muted">{d.empty}</p>
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
        <span>{d.selectAll}</span>
      </label>
      <ul class="list">
        {#each hosts as h (h.alias)}
          <li>
            <label class="row" class:exists={h.exists}>
              <input
                type="checkbox"
                disabled={h.exists}
                checked={h.exists || chosen.has(h.alias)}
                onchange={(e) => toggle(chosen, h.alias, e.currentTarget.checked)}
              />
              <span class="name">{h.alias}</span>
              <span class="addr mono">{h.user}@{h.hostName}{h.port === 22 ? '' : `:${h.port}`}</span
              >
              {#if h.exists}
                <span class="tag">{d.exists}</span>
              {:else if h.proxyJump}
                <span class="tag">{d.via(h.proxyJump)}</span>
              {/if}
            </label>
          </li>
        {/each}
      </ul>
      {#if available.some((h) => h.proxyJump)}
        <p class="muted">{d.jumpNote}</p>
      {/if}
    {/if}
  {:else}
    <p class="lead">{d.ansibleLead}</p>
    <div class="file">
      <span class="path mono" title={invPath}>{invPath || d.noFile}</span>
      <button type="button" class="btn" disabled={!hasRuntime} onclick={() => void choose()}
        >{d.choose}</button
      >
    </div>

    {#if invLoading}
      <p class="muted">{d.loadingInventory}</p>
    {:else if invError}
      <p class="error" role="alert">{invError}</p>
      {#if invYAML}
        <button type="button" class="btn link" onclick={() => openURL(ISSUES_URL)}
          >{d.askYAML}</button
        >
      {/if}
    {:else if inv && inv.hosts.length === 0 && inv.skipped.length === 0}
      <p class="muted">{d.emptyInventory}</p>
    {:else if inv}
      {#if inv.groups.length > 1}
        <div class="groups" role="radiogroup" aria-label={d.groups}>
          <button
            type="button"
            role="radio"
            class="chip"
            aria-checked={group === ''}
            onclick={() => (group = '')}>{d.allGroups}</button
          >
          {#each inv.groups as g (g)}
            <button
              type="button"
              role="radio"
              class="chip"
              aria-checked={group === g}
              onclick={() => (group = g)}>{g}<span class="count">{counts.get(g) ?? 0}</span></button
            >
          {/each}
        </div>
      {/if}
      {#if shown.length > 0}
        <label class="row all">
          <input
            type="checkbox"
            checked={shownFree.length > 0 && shownFree.every((h) => picked.has(h.name))}
            disabled={shownFree.length === 0}
            onchange={(e) => {
              const on = e.currentTarget.checked
              for (const h of shownFree) toggle(picked, h.name, on)
            }}
          />
          <span>{group ? d.selectGroup(group) : d.selectAll}</span>
        </label>
        <ul class="list">
          {#each shown as h (h.name)}
            <li>
              <label class="row" class:exists={h.exists}>
                <input
                  type="checkbox"
                  disabled={h.exists}
                  checked={h.exists || picked.has(h.name)}
                  onchange={(e) => toggle(picked, h.name, e.currentTarget.checked)}
                />
                <span class="name">{h.name}</span>
                <span class="addr mono">{address(h)}</span>
                {#if h.exists}
                  <span class="tag">{d.exists}</span>
                {:else if h.proxyJump}
                  <span class="tag">{d.via(h.proxyJump)}</span>
                {/if}
              </label>
              {#each h.notes ?? [] as n (n.code)}
                <p class="note">{issueText(n)}</p>
              {/each}
            </li>
          {/each}
        </ul>
      {/if}
      {#if shownFree.some((h) => h.proxyJump)}
        <p class="muted">{d.jumpNote}</p>
      {/if}
      {#if inv.skipped.length > 0}
        <details class="skipped">
          <summary>{d.skipped(inv.skipped.length)}</summary>
          <ul>
            {#each inv.skipped as s (s.name)}
              <li><span class="name">{s.name}</span> · {issueText(s.issue)}</li>
            {/each}
          </ul>
        </details>
      {/if}
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={close}>{d.cancel}</button>
    <button type="button" class="btn btn-primary" disabled={busy || count === 0} onclick={submit}
      >{d.submit(count)}</button
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

  .segmented {
    display: flex;
    margin-bottom: 12px;
    padding: 2px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .segmented button {
    flex: 1;
    border: 0;
    border-radius: 5px;
    padding: 5px 8px;
    background: none;
    font-size: var(--text-sm);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .segmented button[aria-checked='true'] {
    background: var(--content-bg);
    box-shadow: var(--switch-knob-shadow);
    font-weight: 600;
  }

  .file {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }

  .path {
    flex: 1;
    min-width: 0;
    padding: 5px 8px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .groups {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 8px;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    border: 0;
    border-radius: 999px;
    padding: 3px 10px;
    background: var(--fill-soft);
    font-size: var(--text-xs);
  }

  .chip[aria-checked='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .count {
    opacity: 0.7;
  }

  .list {
    list-style: none;
    margin: 0 0 10px;
    padding: 4px;
    max-height: 300px;
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

  .note {
    margin: -3px 0 4px 30px;
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .skipped {
    margin-bottom: 10px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .skipped summary {
    cursor: pointer;
  }

  .skipped ul {
    margin: 6px 0 0;
    padding-left: 18px;
  }

  .skipped li {
    padding: 2px 0;
  }

  .error {
    margin: 0 0 8px;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .link {
    margin-bottom: 8px;
  }

  .spacer {
    flex: 1;
  }
</style>
