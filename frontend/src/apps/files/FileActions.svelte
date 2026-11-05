<script lang="ts">
  import { copyText } from '../../lib/api/clipboard'
  import { toUserError } from '../../lib/api/errors'
  import {
    type ArchiveFormat,
    type CompressPlan,
    type Entry,
    filesCompress,
    filesPlanCompress,
  } from '../../lib/api/files'
  import { t } from '../../lib/i18n'
  import { addShortcut, onSurface } from '../../lib/shell/shortcutActions'
  import { notifications } from '../../lib/stores/notifications.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { portal } from '../../lib/ui/portal'
  import { hitTest, type Hit, NO_HIT } from '../../lib/ui/radial/radial'
  import RadialMenu, { type RadialItem } from '../../lib/ui/radial/RadialMenu.svelte'
  import { openFileInEditor, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { ACTIONS, type ActionId, FORMATS, petalsFor } from './actions'
  import CompressDialog from './CompressDialog.svelte'
  import DeleteDialog from './DeleteDialog.svelte'
  import type { DragPhase } from './FileList.svelte'
  import { baseName } from './listing'
  import type { DropTarget } from './remotes.svelte'
  import RenameDialog from './RenameDialog.svelte'

  interface Props {
    serverID: string
    /** The folder on screen, for "Open Terminal here" with nothing selected. */
    cwd: string | null
    /** The window: dragging out of it cancels the menu. */
    bounds: () => DOMRect | undefined
    /** Something changed on the server; reload and select path if given. */
    onchanged: (select: string | null) => void
    /** A short confirmation for the path bar. */
    onflash: (text: string) => void
    /** Another server's folder under the pointer, where a drag can copy to. */
    dropAt?: (x: number, y: number) => DropTarget | null
    /** The drop target the drag is over, to highlight; null when none. */
    onhover?: (target: DropTarget | null) => void
    /** The entry was dropped on another server's folder. */
    ondropcopy?: (entry: Entry, target: DropTarget) => void
  }

  let { serverID, cwd, bounds, onchanged, onflash, dropAt, onhover, ondropcopy }: Props = $props()

  let dialog = $state<{ kind: 'rename' | 'delete' | 'compress'; entry: Entry } | null>(null)
  let menu = $state<{
    entry: Entry
    x: number
    y: number
    hit: Hit
    /** Over another server's folder: the menu steps aside for the drop. */
    target: DropTarget | null
    /** Out of the window: the file follows the pointer, to the wallpaper. */
    away: { x: number; y: number; over: boolean } | null
  } | null>(null)

  const items = $derived<RadialItem[]>(
    menu
      ? petalsFor(menu.entry).map((id) => ({
          id,
          label: t.files.actions[id],
          icon: ACTIONS[id].icon,
          children:
            id === 'compress' ? FORMATS.map((f) => ({ id: f.id, label: f.label })) : undefined,
        }))
      : [],
  )

  /** Follows a drag from the file list (FileList's ondrag). */
  export function drag(entry: Entry, phase: DragPhase, x: number, y: number) {
    if (phase === 'start') {
      menu = { entry, x, y, hit: NO_HIT, target: null, away: null }
      return
    }
    if (!menu) return
    if (phase === 'cancel') {
      menu = null
      onhover?.(null)
      return
    }
    const target = dropAt?.(x, y) ?? null
    if (phase === 'move' && target) {
      if (menu.target?.serverID !== target.serverID || menu.target.path !== target.path) {
        menu.target = target
        onhover?.(target)
      }
      return
    }
    if (menu.target && !target) {
      menu.target = null
      onhover?.(null)
    }
    if (phase === 'end' && target) {
      menu = null
      onhover?.(null)
      ondropcopy?.(entry, target)
      return
    }
    if (phase === 'move') {
      const b = bounds()
      // Out of the window is the way to the wallpaper; other servers are
      // reached through the sidebar.
      if (b && (x < b.left || x > b.right || y < b.top || y > b.bottom)) {
        menu.hit = NO_HIT
        menu.away = { x, y, over: onSurface(x, y) }
        return
      }
      menu.away = null
      menu.hit = hitTest(
        x - menu.x,
        y - menu.y,
        items.length,
        (i) => items[i]?.children?.length ?? 0,
      )
      return
    }
    if (menu.away) {
      menu = null
      if (onSurface(x, y)) void pin(entry, { x, y })
      return
    }
    // Released: run what is under the pointer; the center or nothing cancels.
    const { hit } = menu
    const item = hit.petal === null ? undefined : items[hit.petal]
    menu = null
    if (!item || item.disabled) return
    const sub = hit.sub === null ? undefined : item.children?.[hit.sub]
    act(item.id as ActionId, entry, sub?.id as ArchiveFormat | undefined)
  }

  /** Puts entry on the desktop, at the drop point when dragged there. */
  async function pin(entry: Entry, at?: { x: number; y: number }) {
    try {
      const added = await addShortcut(serverID, entry.path, at)
      onflash(added ? t.files.pinned : t.files.alreadyPinned)
    } catch (e) {
      notifications.add({
        id: `pin-${entry.path}`,
        level: 'error',
        title: t.shortcuts.addFailed,
        body: toUserError(e).message,
        serverID,
        at: Date.now(),
      })
    }
  }

  /**
   * Runs an action on entry, from the menu or a shortcut. Destructive ones
   * open their confirmation dialog first.
   */
  export function act(id: ActionId, entry: Entry | null, format?: ArchiveFormat) {
    if (id === 'editor') {
      if (entry && !entry.dir) openFileInEditor(serverID, entry.path)
      return
    }
    if (id === 'desktop') {
      if (entry) void pin(entry)
      return
    }
    if (id === 'terminal') {
      const dir = entry?.dir ? entry.path : cwd
      if (dir) openTerminalAt(serverID, dir)
      return
    }
    if (!entry) return
    switch (id) {
      case 'copyPath':
        void copyText(entry.path).then((ok) => onflash(ok ? t.files.copied : t.files.copyFailed))
        return
      case 'compress':
        if (format) void compressNow(entry, format)
        else dialog = { kind: 'compress', entry }
        return
      case 'rename':
      case 'delete':
        dialog = { kind: id, entry }
    }
  }

  async function compressNow(entry: Entry, format: ArchiveFormat) {
    try {
      await runCompress(entry, format, await filesPlanCompress(serverID, entry.path, format))
    } catch (e) {
      failed(entry, e)
    }
  }

  async function runCompress(entry: Entry, format: ArchiveFormat, plan: CompressPlan) {
    dialog = null
    const id = `compress-${serverID}-${entry.path}`
    const note = { id, serverID, at: Date.now() }
    notifications.add({
      ...note,
      level: 'info',
      title: t.files.compressing,
      body: baseName(plan.output),
    })
    try {
      const done = await filesCompress(serverID, entry.path, format, plan)
      notifications.add({
        ...note,
        level: 'info',
        title: t.files.compressed(baseName(done.output)),
        body: '',
        at: Date.now(),
      })
      onchanged(done.output)
    } catch (e) {
      failed(entry, e, plan)
    }
  }

  function failed(entry: Entry, e: unknown, plan?: CompressPlan) {
    const ue = toUserError(e)
    // A folder we may not write to: offer the same archiver with sudo.
    const sudo = ue.code === 'FS_PERMISSION_DENIED' ? plan?.sudo : undefined
    notifications.add({
      id: `compress-${serverID}-${entry.path}`,
      level: 'error',
      title: t.files.compressFailed(entry.name),
      body: ue.message,
      serverID,
      at: Date.now(),
      action: sudo
        ? { label: t.common.runInTerminal, run: () => openTerminalAt(serverID, undefined, sudo) }
        : undefined,
    })
  }
</script>

{#if menu?.away}
  <div
    use:portal
    class="away glass glass-strong"
    class:over={menu.away.over}
    style:transform="translate({menu.away.x + 14}px, {menu.away.y + 14}px)"
  >
    <Icon name={menu.entry.dir ? 'folder' : 'file'} size={14} />
    <span class="name">{menu.entry.name}</span>
    <span class="hint">{menu.away.over ? t.files.dropDesktopHere : t.files.dropDesktop}</span>
  </div>
{/if}

{#if menu && !menu.target && !menu.away}
  <RadialMenu
    x={menu.x}
    y={menu.y}
    {items}
    hit={menu.hit}
    title={menu.entry.name}
    icon={menu.entry.dir ? 'folder' : 'file'}
    hint={t.files.dragHint}
  />
{/if}

<DeleteDialog
  {serverID}
  entry={dialog?.kind === 'delete' ? dialog.entry : null}
  onclose={() => (dialog = null)}
  ondone={() => {
    dialog = null
    onchanged(null)
  }}
/>
<RenameDialog
  {serverID}
  entry={dialog?.kind === 'rename' ? dialog.entry : null}
  onclose={() => (dialog = null)}
  ondone={(path) => {
    dialog = null
    onchanged(path)
  }}
/>
<CompressDialog
  {serverID}
  entry={dialog?.kind === 'compress' ? dialog.entry : null}
  onclose={() => (dialog = null)}
  onconfirm={(e, format, plan) => void runCompress(e, format, plan)}
/>

<style>
  /* Above every window, dialogs aside; the wallpaper must stay hit-testable. */
  .away {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 9000;
    display: flex;
    align-items: center;
    gap: 6px;
    max-width: 320px;
    padding: 6px 10px;
    border-radius: var(--control-radius);
    font-size: var(--text-sm);
    white-space: nowrap;
    pointer-events: none;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 600;
  }

  .hint {
    color: var(--fg-muted);
  }

  .over .hint {
    color: var(--accent);
  }
</style>
