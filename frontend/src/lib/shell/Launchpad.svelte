<script lang="ts">
  import { tick, untrack } from 'svelte'

  import { CATEGORY, CATEGORIES, gridMove, grouped, type Category } from '../../apps/categories'
  import { apps, isAvailable, platformName } from '../../apps/registry'
  import type { Platform } from '../api/servers'
  import type { AppDef } from '../../apps/types'
  import { t } from '../i18n'
  import { score } from '../spotlight/search'
  import { dockPins } from '../stores/dock.svelte'
  import AppIcon from '../ui/AppIcon.svelte'
  import Icon from '../ui/Icon.svelte'
  import { portal } from '../ui/portal'

  interface Props {
    open: boolean
    /** The active server's OS; apps that do not run there are shown disabled. */
    platform?: Platform
    onclose: () => void
    onlaunch: (app: AppDef) => void
  }

  let { open, platform, onclose, onlaunch }: Props = $props()

  /** Tiles per row; arrow keys move by this much up and down. */
  const COLS = 5

  let query = $state('')
  let index = $state(0)
  let tab = $state<'all' | Category>('all')
  let confirmReset = $state(false)
  let input = $state<HTMLInputElement>()
  let panel = $state<HTMLElement>()
  const gridId = $props.id()

  // A search looks in every group, best match first. Otherwise "All"
  // shows each group under its name, and a group tab shows only it.
  const sections = $derived.by((): { category: Category | null; apps: AppDef[] }[] => {
    if (query.trim()) {
      const found = apps
        .map((a) => ({ a, s: score(query, a.label) }))
        .filter((x) => x.s > 0)
        .sort((x, y) => y.s - x.s)
        .map((x) => x.a)
      return found.length ? [{ category: null, apps: found }] : []
    }
    if (tab === 'all') return grouped(apps)
    return [{ category: null, apps: apps.filter((a) => CATEGORY[a.id] === tab) }]
  })
  const shown = $derived(sections.flatMap((x) => x.apps))
  // Where each section starts in shown.
  const starts = $derived.by(() => {
    let n = 0
    return sections.map((x) => (n += x.apps.length) - x.apps.length)
  })

  $effect(() => {
    if (!open) return
    untrack(() => {
      query = ''
      index = 0
      requestAnimationFrame(() => input?.focus())
    })
  })

  $effect(() => {
    void shown.length
    untrack(() => (index = Math.min(index, Math.max(0, shown.length - 1))))
  })

  // Keep the selected tile in view when the arrows move past the edge.
  $effect(() => {
    void index
    void tick().then(() =>
      panel?.querySelector('.cell.active')?.scrollIntoView({ block: 'nearest' }),
    )
  })

  function pickTab(next: 'all' | Category) {
    tab = next
    query = ''
    index = 0
    input?.focus()
  }

  function resetDock() {
    if (!confirmReset) {
      confirmReset = true
      setTimeout(() => (confirmReset = false), 3000)
      return
    }
    confirmReset = false
    dockPins.reset()
  }

  function choose(app: AppDef | undefined) {
    if (!app || !isAvailable(app, platform)) return
    onclose()
    onlaunch(app)
  }

  const arrows = ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'] as const
  type Arrow = (typeof arrows)[number]
  const isArrow = (k: string): k is Arrow => (arrows as readonly string[]).includes(k)

  function onkeydown(e: KeyboardEvent) {
    if (isArrow(e.key)) {
      // Left and right stay in the text while there is a query to edit.
      if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && query) return
      e.preventDefault()
      if (shown.length)
        index = gridMove(
          sections.map((x) => x.apps.length),
          index,
          e.key,
          COLS,
        )
    } else if (e.key === 'Enter') {
      e.preventDefault()
      choose(shown[index])
    } else if (e.key === 'Escape') {
      e.preventDefault()
      if (query) query = ''
      else onclose()
    }
  }
</script>

{#if open}
  <!-- The backdrop only closes; every action has a key. -->
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="backdrop" use:portal onclick={onclose}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      bind:this={panel}
      class="panel glass glass-strong"
      role="dialog"
      aria-label={t.launchpad.label}
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="searchbox">
        <Icon name="search" size={14} strokeWidth={2.2} />
        <input
          bind:this={input}
          bind:value={query}
          {onkeydown}
          type="text"
          placeholder={t.launchpad.search}
          aria-label={t.launchpad.search}
          aria-controls={gridId}
          spellcheck="false"
          autocomplete="off"
        />
      </div>

      <div class="tabs" role="tablist" aria-label={t.launchpad.groups}>
        {#each ['all', ...CATEGORIES] as const as c (c)}
          <button
            role="tab"
            aria-selected={tab === c && !query}
            class:on={tab === c && !query}
            onclick={() => pickTab(c)}
            >{c === 'all' ? t.launchpad.all : t.launchpad.categories[c]}</button
          >
        {/each}
      </div>

      {#if shown.length > 0}
        <div class="scroll" id={gridId}>
          {#each sections as sec, k (sec.category ?? k)}
            {#if sec.category}<h4>{t.launchpad.categories[sec.category]}</h4>{/if}
            <ul class="grid" style:--cols={COLS}>
              {#each sec.apps as app, j (app.id)}
                {@const i = (starts[k] ?? 0) + j}
                {@const pinned = dockPins.has(app.id)}
                {@const usable = isAvailable(app, platform)}
                <li class="cell" class:active={i === index}>
                  <button
                    class="app"
                    class:unavailable={!usable}
                    aria-disabled={!usable}
                    title={usable
                      ? undefined
                      : t.launchpad.unavailable(platformName[platform ?? 'unknown'])}
                    data-launchpad-app={app.id}
                    onclick={() => choose(app)}
                    onpointerenter={() => (index = i)}
                  >
                    <span class="art"><AppIcon id={app.id} size={60} /></span>
                    <span class="name">{app.label}</span>
                  </button>
                  <button
                    class="pin"
                    class:pinned
                    aria-label={pinned ? t.launchpad.unpin(app.label) : t.launchpad.pin(app.label)}
                    title={pinned ? t.launchpad.unpin(app.label) : t.launchpad.pin(app.label)}
                    aria-pressed={pinned}
                    onclick={() => dockPins.toggle(app.id)}
                  >
                    <Icon name={pinned ? 'minus' : 'plus'} size={11} strokeWidth={2.8} />
                  </button>
                </li>
              {/each}
            </ul>
          {/each}
        </div>
      {:else}
        <p class="none">{t.launchpad.none}</p>
      {/if}
      <div class="foot">
        <p class="hint">{t.launchpad.hint}</p>
        <button class="reset" disabled={dockPins.isDefault} onclick={resetDock}
          >{confirmReset ? t.launchpad.resetConfirm : t.launchpad.resetDock}</button
        >
      </div>
    </div>
  </div>
{/if}

<style>
  .app.unavailable {
    opacity: 0.38;
    filter: grayscale(1);
    cursor: not-allowed;
  }

  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 9000;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 12vh;
    background: var(--overlay);
  }

  .panel {
    width: min(720px, calc(100vw - 40px));
    max-height: 76vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 18px 18px 10px;
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
    width: 240px;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .searchbox input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: none;
    background: transparent;
    font-size: var(--text-sm);
    color: var(--fg);
  }

  .tabs {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 2px;
    margin-top: 12px;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .tabs button {
    padding: 2px 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .tabs button.on {
    background: var(--chart-surface);
    color: var(--fg);
    font-weight: 600;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
  }

  .scroll {
    width: 100%;
    margin-top: 10px;
    overflow-y: auto;
  }

  h4 {
    margin: 10px 6px 2px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  .grid {
    width: 100%;
    margin: 0;
    padding: 0;
    list-style: none;
    display: grid;
    grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
    gap: 6px;
  }

  .cell {
    position: relative;
  }

  .app {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 10px 4px 8px;
    border: 0;
    border-radius: var(--item-radius);
    background: none;
    color: var(--fg);
  }

  .active .app {
    background: var(--accent-soft);
  }

  .art {
    display: flex;
    border-radius: var(--dock-tile-radius);
    overflow: hidden;
    box-shadow: var(--tile-shadow);
  }

  .name {
    max-width: 100%;
    font-size: var(--text-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Pin toggle: shows on hover or focus, like the jiggle badge on macOS. */
  .pin {
    position: absolute;
    top: 4px;
    right: calc(50% - 40px);
    display: flex;
    padding: 3px;
    border: 0;
    border-radius: 50%;
    background: var(--fg-muted);
    color: var(--fg-on-accent);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .cell:hover .pin,
  .cell.active .pin,
  .pin:focus-visible {
    opacity: 1;
  }

  /* A pinned app shows its mark all the time, so the Dock's apps stand out. */
  .pin.pinned {
    opacity: 1;
    background: var(--accent);
  }

  .none {
    margin: 28px 0 22px;
    color: var(--fg-subtle);
  }

  .foot {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 10px;
    padding: 0 6px;
  }

  .hint {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .reset {
    padding: 3px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-xs);
    color: var(--fg);
  }

  .reset:disabled {
    opacity: 0.5;
  }
</style>
