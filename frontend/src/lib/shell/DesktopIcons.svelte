<script lang="ts">
  import { toUserError } from '../api/errors'
  import { shortcutInspect, shortcutRun, type ShortcutTarget } from '../api/shortcuts'
  import { t } from '../i18n'
  import { deploys } from '../stores/deploys.svelte'
  import { notifications } from '../stores/notifications.svelte'
  import { servers } from '../stores/servers.svelte'
  import { type Shortcut, shortcuts } from '../stores/shortcuts.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import Icon from '../ui/Icon.svelte'
  import PopupMenu from '../ui/PopupMenu.svelte'
  import type { IconName } from '../ui/icons'
  import {
    focusTerminal,
    openFileInEditor,
    openFilesAt,
    openTerminalAt,
  } from '../wm/intents.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import { cellAt, cellPosition, gridOf, layout } from './iconGrid'
  import { hasChanged, needsConfirm, simulateRun } from './shortcutActions'
  import ShortcutRunDialog from './ShortcutRunDialog.svelte'

  const s = t.shortcuts
  const ICONS: Record<Shortcut['kind'], IconName> = {
    script: 'terminal',
    file: 'file',
    folder: 'folder',
  }

  // A press becomes a drag only after it moves this far, so clicks and
  // double-clicks stay what they are.
  const DRAG_START = 6

  const serverID = $derived(servers.activeId)
  const server = $derived(servers.list.find((x) => x.id === serverID))
  const online = $derived(server?.state === 'connected')
  const items = $derived(shortcuts.of(serverID))
  const grid = $derived(gridOf(wm.area))
  const cells = $derived(layout(items, grid))

  let selected = $state<string | null>(null)
  let drag = $state<{ path: string; dx: number; dy: number } | null>(null)
  let press: { path: string; x: number; y: number; pointer: number } | null = null
  let dialog = $state<{ item: Shortcut; target: ShortcutTarget; serverID: string } | null>(null)
  let missing = $state<Shortcut | null>(null)
  let menu = $state<{ item: Shortcut; x: number; y: number } | null>(null)

  function fail(e: unknown, title: string) {
    notifications.add({
      id: `shortcut-${Date.now()}`,
      level: 'error',
      title,
      body: toUserError(e).message,
      serverID,
      at: Date.now(),
    })
  }

  async function open(item: Shortcut) {
    if (!online) return
    if (deploys.active(serverID, item.path)) {
      focusTerminal(serverID)
      return
    }
    const id = serverID
    try {
      const target = await shortcutInspect(id, item.path)
      if (target.kind !== item.kind) shortcuts.setKind(id, item.path, target.kind)
      if (target.kind === 'folder') openFilesAt(id, item.path)
      else if (target.kind === 'file') openFileInEditor(id, item.path)
      else if (needsConfirm(item, target)) dialog = { item, target, serverID: id }
      else await run(item, target, id)
    } catch (e) {
      if (toUserError(e).code === 'NOT_FOUND') missing = item
      else fail(e, s.openFailed)
    }
  }

  async function run(item: Shortcut, target: ShortcutTarget, id: string) {
    try {
      const r = await shortcutRun(id, item.path, target.hash)
      deploys.watch(id, item.path, r.nonce, [item.name], true, 'script')
      openTerminalAt(id, undefined, r.line, r.nonce, true)
      simulateRun(r.nonce)
    } catch (e) {
      // Changed between looking and running: show what is there now.
      if (toUserError(e).code !== 'SHORTCUT_CHANGED') throw e
      const fresh = await shortcutInspect(id, item.path)
      dialog = {
        item: { ...item, trusted: item.trusted ?? target.hash },
        target: fresh,
        serverID: id,
      }
    }
  }

  function confirmRun(remember: boolean) {
    if (!dialog) return
    const { item, target, serverID: id } = dialog
    dialog = null
    if (remember) shortcuts.trust(id, item.path, target.hash)
    run(item, target, id).catch((e: unknown) => fail(e, s.openFailed))
  }

  function review() {
    if (!dialog) return
    openFileInEditor(dialog.serverID, dialog.item.path)
    dialog = null
  }

  function parentOf(path: string) {
    return path.slice(0, path.lastIndexOf('/')) || '/'
  }

  function actions(item: Shortcut) {
    return [
      { label: s.open, run: () => void open(item), disabled: !online },
      {
        label: s.showInFiles,
        run: () => openFilesAt(serverID, parentOf(item.path)),
        disabled: !online,
      },
      item.trusted
        ? { label: s.untrust, run: () => shortcuts.trust(serverID, item.path, undefined) }
        : null,
      { label: s.remove, run: () => shortcuts.remove(serverID, item.path), danger: true },
    ]
  }

  /** The result of the last run, as a dot on the icon. */
  function lastRun(item: Shortcut) {
    return deploys.historyOf(serverID, item.path)[0]?.state
  }

  function hint(item: Shortcut) {
    const notes = [item.path]
    if (!online) notes.push(s.offline)
    else if (item.trusted) notes.push(s.trustedHint)
    const last = lastRun(item)
    if (last === 'ok') notes.push(s.lastOk)
    else if (last === 'failed') notes.push(s.lastFailed)
    return notes.join('\n')
  }

  function onPointerDown(ev: PointerEvent, item: Shortcut) {
    if (ev.button !== 0) return
    press = { path: item.path, x: ev.clientX, y: ev.clientY, pointer: ev.pointerId }
  }

  function onPointerMove(ev: PointerEvent) {
    if (!press || ev.pointerId !== press.pointer) return
    const dx = ev.clientX - press.x
    const dy = ev.clientY - press.y
    if (!drag && Math.hypot(dx, dy) < DRAG_START) return
    ;(ev.currentTarget as HTMLElement).setPointerCapture(ev.pointerId)
    drag = { path: press.path, dx, dy }
  }

  function onPointerUp(ev: PointerEvent) {
    if (drag) {
      shortcuts.move(serverID, drag.path, cellAt(ev.clientX, ev.clientY, wm.area, grid), grid)
    }
    drag = null
    press = null
  }

  function onKeyDown(ev: KeyboardEvent, item: Shortcut) {
    if (ev.key === 'Enter') {
      ev.preventDefault()
      void open(item)
    } else if (ev.key === 'Delete') {
      shortcuts.remove(serverID, item.path)
    } else if (ev.key === 'ContextMenu' || (ev.key === 'F10' && ev.shiftKey)) {
      // The menu has no button: this is its keyboard way in.
      ev.preventDefault()
      const r = (ev.currentTarget as HTMLElement).getBoundingClientRect()
      selected = item.path
      menu = { item, x: r.right, y: r.top }
    }
  }

  function onContextMenu(ev: MouseEvent, item: Shortcut) {
    ev.preventDefault()
    selected = item.path
    menu = { item, x: ev.clientX, y: ev.clientY }
  }
</script>

<!-- Pressing the bare wallpaper clears the selection, as on any desktop. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="surface"
  data-desktop-surface
  onpointerdown={(e) => {
    if (e.target === e.currentTarget) selected = null
  }}
>
  <ul aria-label={s.label}>
    {#each items as item, i (item.path)}
      {@const cell = cells[i] ?? item}
      {@const pos = cellPosition(cell, wm.area)}
      {@const active = deploys.active(serverID, item.path)}
      {@const moving = drag?.path === item.path}
      {@const last = lastRun(item)}
      <li
        class:selected={selected === item.path}
        class:off={!online}
        class:moving
        style:transform="translate({pos.x + (moving ? (drag?.dx ?? 0) : 0)}px, {pos.y +
          (moving ? (drag?.dy ?? 0) : 0)}px)"
      >
        <button
          type="button"
          class="icon"
          title={hint(item)}
          aria-disabled={!online}
          onclick={() => (selected = item.path)}
          ondblclick={() => void open(item)}
          onkeydown={(e) => onKeyDown(e, item)}
          oncontextmenu={(e) => onContextMenu(e, item)}
          onpointerdown={(e) => onPointerDown(e, item)}
          onpointermove={onPointerMove}
          onpointerup={onPointerUp}
          onpointercancel={onPointerUp}
        >
          <span class="tile glass" class:running={!!active}>
            <Icon name={ICONS[item.kind]} size={24} />
            {#if !active && item.kind === 'script' && last && last !== 'stopped'}
              <span class="dot {last}"></span>
            {/if}
          </span>
          <span class="name">{item.name}</span>
        </button>
        {#if active}
          <button
            type="button"
            class="stop"
            aria-label={s.stop(item.name)}
            data-own-tooltip
            title={s.stop(item.name)}
            onclick={() => deploys.interrupt(active.nonce)}
          >
            <Icon name="close" size={11} strokeWidth={3} />
          </button>
        {/if}
      </li>
    {/each}
  </ul>
</div>

{#if menu}
  <PopupMenu
    x={menu.x}
    y={menu.y}
    label={s.actions(menu.item.name)}
    actions={actions(menu.item)}
    onclose={() => (menu = null)}
  />
{/if}

<ShortcutRunDialog
  target={dialog?.target ?? null}
  server={servers.list.find((x) => x.id === dialog?.serverID)?.name ?? ''}
  changed={dialog ? hasChanged(dialog.item, dialog.target) : false}
  onclose={() => (dialog = null)}
  onreview={review}
  onrun={confirmRun}
/>

<Dialog
  open={missing !== null}
  title={s.missingTitle(missing?.name ?? '')}
  onclose={() => (missing = null)}
>
  <p>{s.missingBody(missing?.path ?? '')}</p>
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (missing = null)}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-danger"
      onclick={() => {
        if (missing) shortcuts.remove(serverID, missing.path)
        missing = null
      }}
    >
      {s.remove2}
    </button>
  {/snippet}
</Dialog>

<style>
  .surface {
    position: absolute;
    inset: 0;
    z-index: 0;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    position: absolute;
    top: 0;
    left: 0;
    width: 96px;
    height: 100px;
  }

  li.moving {
    z-index: 1;
    opacity: 0.8;
  }

  li.off .tile,
  li.off .name {
    opacity: 0.5;
  }

  .icon {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
    width: 100%;
    padding: 6px 4px 4px;
    border: none;
    border-radius: 12px;
    background: transparent;
    color: var(--fg);
    font: inherit;
    cursor: default;
    touch-action: none;
  }

  .selected .icon,
  .icon:focus-visible {
    background: var(--fill-soft);
    outline: none;
  }

  .tile {
    position: relative;
    display: grid;
    place-items: center;
    width: 52px;
    height: 52px;
    border-radius: var(--dock-tile-radius);
  }

  .tile.running {
    box-shadow: 0 0 0 2px var(--accent);
    animation: pulse 1.6s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      box-shadow: 0 0 0 2px transparent;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .tile.running {
      animation: none;
    }
  }

  .dot {
    position: absolute;
    right: -2px;
    bottom: -2px;
    width: 10px;
    height: 10px;
    border: 2px solid var(--wallpaper-base);
    border-radius: 50%;
  }

  .dot.ok {
    background: var(--ok);
  }

  .dot.failed {
    background: var(--danger);
  }

  .name {
    display: -webkit-box;
    max-width: 100%;
    overflow: hidden;
    overflow-wrap: anywhere;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    font-size: var(--text-xs);
    font-weight: 500;
    line-height: 1.25;
    text-align: center;
  }

  .stop {
    position: absolute;
    top: 0;
    left: 14px;
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: none;
    border-radius: 50%;
    background: var(--danger);
    color: var(--fg-on-accent);
    cursor: pointer;
  }

  p {
    margin: 0;
    line-height: 1.45;
    overflow-wrap: anywhere;
  }

  .spacer {
    flex: 1;
  }
</style>
