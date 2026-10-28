<script lang="ts">
  import { untrack } from 'svelte'

  import { apps } from '../../apps/registry'
  import type { AppDef } from '../../apps/types'
  import { t } from '../i18n/vi'
  import { score } from '../spotlight/search'
  import { dockPins } from '../stores/dock.svelte'
  import AppIcon from '../ui/AppIcon.svelte'
  import Icon from '../ui/Icon.svelte'
  import { portal } from '../ui/portal'

  interface Props {
    open: boolean
    onclose: () => void
    onlaunch: (app: AppDef) => void
  }

  let { open, onclose, onlaunch }: Props = $props()

  /** Tiles per row; arrow keys move by this much up and down. */
  const COLS = 5

  let query = $state('')
  let index = $state(0)
  let input = $state<HTMLInputElement>()
  const gridId = $props.id()

  // Registry order when nothing is typed, best match first otherwise.
  const shown = $derived(
    query.trim()
      ? apps
          .map((a) => ({ a, s: score(query, a.label) }))
          .filter((x) => x.s > 0)
          .sort((x, y) => y.s - x.s)
          .map((x) => x.a)
      : apps,
  )

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

  function choose(app: AppDef | undefined) {
    if (!app) return
    onclose()
    onlaunch(app)
  }

  const moves: Record<string, number> = {
    ArrowRight: 1,
    ArrowLeft: -1,
    ArrowDown: COLS,
    ArrowUp: -COLS,
  }

  function onkeydown(e: KeyboardEvent) {
    const step = moves[e.key]
    if (step !== undefined) {
      // Left and right stay in the text while there is a query to edit.
      if (Math.abs(step) === 1 && query) return
      e.preventDefault()
      const next = index + step
      if (next >= 0 && next < shown.length) index = next
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

      {#if shown.length > 0}
        <ul id={gridId} class="grid" style:--cols={COLS}>
          {#each shown as app, i (app.id)}
            {@const pinned = dockPins.has(app.id)}
            <li class="cell" class:active={i === index}>
              <button
                class="app"
                data-launchpad-app={app.id}
                onclick={() => choose(app)}
                onpointerenter={() => (index = i)}
              >
                <span class="art"><AppIcon id={app.id} size={60} /></span>
                <span class="name">{app.label}</span>
              </button>
              <button
                class="pin"
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
      {:else}
        <p class="none">{t.launchpad.none}</p>
      {/if}
      <p class="hint">{t.launchpad.hint}</p>
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
    padding-top: 12vh;
    background: var(--overlay);
  }

  .panel {
    width: min(660px, calc(100vw - 40px));
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

  .grid {
    width: 100%;
    margin: 14px 0 0;
    padding: 0;
    list-style: none;
    display: grid;
    grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
    gap: 6px;
    overflow-y: auto;
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

  .none {
    margin: 28px 0 22px;
    color: var(--fg-subtle);
  }

  .hint {
    margin: 10px 0 0;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }
</style>
