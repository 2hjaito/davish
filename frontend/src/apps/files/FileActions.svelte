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
  import { t } from '../../lib/i18n/vi'
  import { notifications } from '../../lib/stores/notifications.svelte'
  import { hitTest, type Hit, NO_HIT } from '../../lib/ui/radial/radial'
  import RadialMenu, { type RadialItem } from '../../lib/ui/radial/RadialMenu.svelte'
  import { openFileInEditor, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { ACTIONS, type ActionId, FORMATS, petalsFor } from './actions'
  import CompressDialog from './CompressDialog.svelte'
  import DeleteDialog from './DeleteDialog.svelte'
  import type { DragPhase } from './FileList.svelte'
  import { baseName } from './listing'
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
  }

  let { serverID, cwd, bounds, onchanged, onflash }: Props = $props()

  let dialog = $state<{ kind: 'rename' | 'delete' | 'compress'; entry: Entry } | null>(null)
  let menu = $state<{ entry: Entry; x: number; y: number; hit: Hit } | null>(null)

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
      menu = { entry, x, y, hit: NO_HIT }
      return
    }
    if (!menu) return
    if (phase === 'cancel') {
      menu = null
      return
    }
    if (phase === 'move') {
      const b = bounds()
      // Leaving the window is for dragging files elsewhere (M9), not this menu.
      if (b && (x < b.left || x > b.right || y < b.top || y > b.bottom)) {
        menu = null
        return
      }
      menu.hit = hitTest(
        x - menu.x,
        y - menu.y,
        items.length,
        (i) => items[i]?.children?.length ?? 0,
      )
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

  /**
   * Runs an action on entry, from the menu or a shortcut. Destructive ones
   * open their confirmation dialog first.
   */
  export function act(id: ActionId, entry: Entry | null, format?: ArchiveFormat) {
    if (id === 'editor') {
      if (entry && !entry.dir) openFileInEditor(serverID, entry.path)
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

{#if menu}
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
