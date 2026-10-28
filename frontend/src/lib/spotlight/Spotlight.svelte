<script lang="ts">
  import { untrack } from 'svelte'

  import { lastDir } from '../../apps/files/memory'
  import { dockApps, getApp } from '../../apps/registry'
  import { backend as dockerBackend } from '../../apps/docker/backend'
  import { backend as packagesBackend } from '../../apps/packages/backend'
  import { backend as servicesBackend } from '../../apps/services/backend'
  import { filesStat } from '../api/files'
  import { hasRuntime } from '../api/runtime'
  import { t } from '../i18n/vi'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import AppIcon from '../ui/AppIcon.svelte'
  import Icon from '../ui/Icon.svelte'
  import type { IconName } from '../ui/icons'
  import { portal } from '../ui/portal'
  import { openFileInEditor, openFilesAt, openSelected, openTerminalAt } from '../wm/intents.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import { type Candidate, intent, rank } from './search'

  interface Props {
    open: boolean
    onclose: () => void
  }

  let { open, onclose }: Props = $props()

  interface Item extends Candidate {
    icon: IconName | { app: string }
    run: () => void
  }

  let query = $state('')
  let index = $state(0)
  let input = $state<HTMLInputElement>()
  let units = $state.raw<{ name: string; description: string }[]>([])
  let containers = $state.raw<{ name: string; image: string }[]>([])
  let programs = $state.raw<{ id: string; label: string; version: string }[]>([])
  const listId = $props.id()

  const serverID = $derived(servers.activeId)
  const connected = $derived(!hasRuntime || servers.active?.state === 'connected')

  // Each open starts clean and fetches what this server runs; both lists
  // are commands in the audit log, so they are asked once per opening.
  $effect(() => {
    if (!open) return
    untrack(() => {
      query = ui.spotlightSeed
      ui.spotlightSeed = ''
      index = 0
      requestAnimationFrame(() => input?.focus())
      if (!connected) return
      const id = serverID
      servicesBackend.list(id).then(
        (u) => id === serverID && (units = u),
        () => (units = []),
      )
      dockerBackend.list(id).then(
        (c) => id === serverID && (containers = c),
        () => (containers = []),
      )
      packagesBackend.apps(id).then(
        (a) => id === serverID && (programs = a),
        () => (programs = []),
      )
    })
  })

  function appItems(): Item[] {
    const extra = ['audit', 'transfers'].map((id) => getApp(id)).filter((a) => a !== undefined)
    return [...dockApps, ...extra].map((a) => ({
      kind: 'app',
      id: a.id,
      label: a.label,
      icon: { app: a.id },
      run: () => wm.openOrFocus(a.id, serverID, { title: a.label, size: a.size }),
    }))
  }

  function select(appId: 'services' | 'docker' | 'packages', key: string) {
    const app = getApp(appId)
    if (app) openSelected(appId, serverID, key, { title: app.label, size: app.size })
  }

  async function openPath(path: string) {
    if (!hasRuntime) {
      if (/\.[a-z0-9]+$/i.test(path)) openFileInEditor(serverID, path)
      else openFilesAt(serverID, path)
      return
    }
    try {
      const e = await filesStat(serverID, path)
      if (e.dir) openFilesAt(serverID, path)
      else openFileInEditor(serverID, path)
    } catch {
      // Not there or not readable: Files shows why.
      openFilesAt(serverID, path)
    }
  }

  const items = $derived.by<Item[]>(() => {
    const all: Item[] = appItems()
    const want = intent(query)
    if (want?.kind === 'path' && connected)
      all.push({
        kind: 'path',
        id: want.path,
        label: t.spotlight.openPath(want.path),
        icon: 'folder',
        run: () => void openPath(want.path),
      })
    if (want?.kind === 'command' && connected)
      all.push({
        kind: 'command',
        id: want.command,
        label: t.spotlight.typeCommand(want.command),
        icon: 'terminal',
        run: () => openTerminalAt(serverID, undefined, want.command),
      })
    for (const u of units)
      all.push({
        kind: 'service',
        id: u.name,
        label: u.name.replace(/\.service$/, ''),
        detail: u.description,
        icon: 'services',
        run: () => select('services', u.name),
      })
    for (const c of containers)
      all.push({
        kind: 'container',
        id: c.name,
        label: c.name,
        detail: c.image,
        icon: 'docker',
        run: () => select('docker', c.name),
      })
    for (const p of programs)
      all.push({
        kind: 'program',
        id: p.id,
        label: p.label,
        detail: p.version,
        icon: 'package',
        run: () => select('packages', p.id),
      })
    const folders = [lastDir(serverID), '/etc', '/var/log', '/etc/nginx', '/srv', '/opt']
    for (const f of new Set(folders.filter((f): f is string => !!f)))
      all.push({
        kind: 'folder',
        id: f,
        label: f,
        icon: 'folder',
        run: () => openFilesAt(serverID, f),
      })
    // Help answers a question, so it shows only once something is typed.
    if (query.trim())
      for (const [appId, h] of Object.entries(t.help.apps)) {
        const app = getApp(appId)
        if (!app) continue
        all.push({
          kind: 'help',
          id: appId,
          label: t.help.spotlight(app.label),
          detail: [h.does, ...h.actions, ...h.runs].join(' · '),
          icon: 'help',
          run: () => {
            wm.openOrFocus(app.id, serverID, { title: app.label, size: app.size })
            ui.help = { kind: 'app', appId }
          },
        })
      }
    // A path or command typed in full is the point of the query: keep it
    // even when it matches nothing else.
    const q = want ? '' : query
    return rank(all, q).filter(
      (x) => !want || x.kind === want.kind || x.kind === 'folder' || x.kind === 'app',
    )
  })

  $effect(() => {
    void items.length
    untrack(() => (index = Math.min(index, Math.max(0, items.length - 1))))
  })

  function choose(item: Item | undefined) {
    if (!item) return
    onclose()
    item.run()
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      onclose()
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      index = Math.min(items.length - 1, index + 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      index = Math.max(0, index - 1)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      choose(items[index])
    }
  }

  const kindLabel = $derived(t.spotlight.kinds)
</script>

{#if open}
  <!-- The backdrop only closes; every action has a key. -->
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="backdrop" use:portal onclick={onclose}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="panel glass glass-strong"
      role="dialog"
      aria-label={t.spotlight.label}
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="searchbox">
        <Icon name="search" size={18} strokeWidth={2.2} />
        <input
          bind:this={input}
          bind:value={query}
          {onkeydown}
          type="text"
          role="combobox"
          aria-expanded="true"
          aria-controls="{listId}-list"
          aria-activedescendant={items[index] ? `${listId}-${index}` : undefined}
          placeholder={t.spotlight.placeholder}
          spellcheck="false"
          autocomplete="off"
        />
      </div>
      {#if items.length > 0}
        <ul id="{listId}-list" role="listbox" aria-label={t.spotlight.results}>
          {#each items as item, i (item.kind + item.id)}
            {#if i === 0 || items[i - 1]?.kind !== item.kind}
              <li class="group" role="presentation">{kindLabel[item.kind]}</li>
            {/if}
            <li
              id="{listId}-{i}"
              role="option"
              aria-selected={i === index}
              class:active={i === index}
              onpointermove={() => (index = i)}
              onclick={() => choose(item)}
            >
              <span class="icon">
                {#if typeof item.icon === 'object'}
                  <AppIcon id={item.icon.app as never} size={22} />
                {:else}
                  <Icon name={item.icon} size={16} />
                {/if}
              </span>
              <span class="label">{item.label}</span>
              {#if item.detail}<span class="detail">{item.detail}</span>{/if}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="none">{t.spotlight.none}</p>
      {/if}
      <p class="hint">{t.spotlight.hint}</p>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 9000;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 16vh;
    background: var(--overlay);
  }

  .panel {
    width: min(620px, calc(100vw - 40px));
    max-height: 64vh;
    display: flex;
    flex-direction: column;
    border-radius: var(--menu-radius);
    overflow: hidden;
    animation: pop var(--dur) var(--ease-out);
  }

  @keyframes pop {
    from {
      opacity: 0;
      transform: scale(0.97) translateY(-4px);
    }
  }

  .searchbox {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    color: var(--fg-muted);
    border-bottom: 1px solid var(--divider-soft);
  }

  .searchbox input {
    flex: 1;
    border: 0;
    outline: none;
    background: transparent;
    font-size: 19px;
    color: var(--fg);
  }

  ul {
    margin: 0;
    padding: 6px;
    list-style: none;
    overflow-y: auto;
  }

  .group {
    padding: 8px 10px 3px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-subtle);
  }

  [role='option'] {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 34px;
    padding: 0 10px;
    border-radius: var(--item-radius);
    cursor: default;
  }

  [role='option'].active {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .icon {
    display: flex;
    width: 22px;
    justify-content: center;
    color: var(--accent);
    flex-shrink: 0;
  }

  .icon :global(svg) {
    border-radius: 5px;
  }

  .active .icon {
    color: var(--fg-on-accent);
  }

  .label {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .detail {
    margin-left: auto;
    padding-left: 12px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .active .detail {
    color: var(--fg-on-accent);
    opacity: 0.85;
  }

  .none {
    margin: 0;
    padding: 16px;
    color: var(--fg-muted);
  }

  .hint {
    margin: 0;
    padding: 6px 16px 8px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }
</style>
