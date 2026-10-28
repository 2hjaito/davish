<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import {
    type Access,
    type FileContent,
    filesAccess,
    filesRead,
    filesSave,
  } from '../../lib/api/files'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n/vi'
  import { servers } from '../../lib/stores/servers.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { type EditorRequest, editorRequests, openTerminalAt } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { wm } from '../../lib/wm/windowManager.svelte'
  import { mockRead } from '../files/mockFs'
  import type { AppProps } from '../types'
  import CodeEditor from './CodeEditor.svelte'
  import { decodeText } from './decode'
  import { detectLanguage, LANGUAGE_LABELS, type LanguageId } from './languages'
  import { settings } from '../../lib/stores/settings.svelte'
  import { backupFor } from './save'

  let { win }: AppProps = $props()

  interface Tab {
    id: number
    path: string
    name: string
    state: 'loading' | 'ready' | 'error'
    error?: string
    /** Text as read; edits live in the editor until saved. */
    doc: string
    language: LanguageId
    /** Stat at read time, to tell later whether the server copy changed. */
    file?: Omit<FileContent, 'data'>
    line?: number
    dirty: boolean
    /** Saving would fail (permissions); the file opens read-only. */
    access?: Access
    /** Bumped when the text is reloaded, so the editor starts over. */
    version: number
    saving?: boolean
    /** The outcome of the last save, shown in the status bar until the next edit. */
    notice?: { text: string; error?: boolean }
    cursor: { line: number; col: number }
  }

  let nextId = 1
  let tabs = $state<Tab[]>([])
  let activeId = $state<number | null>(null)
  const views: Record<number, CodeEditor> = {}
  /** Tabs with unsaved work waiting on "save or discard"; window means closing it all. */
  let closing = $state<{ ids: number[]; window: boolean } | null>(null)
  /** The tab whose file changed on the server while being edited. */
  let changedId = $state<number | null>(null)
  const changedTab = $derived(tabs.find((x) => x.id === changedId))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const active = $derived(tabs.find((x) => x.id === activeId))

  function open(req: EditorRequest) {
    const existing = tabs.find((x) => x.path === req.path)
    if (existing) {
      activeId = existing.id
      if (req.line) views[existing.id]?.goToLine(req.line)
      return
    }
    const name = req.path.slice(req.path.lastIndexOf('/') + 1)
    const tab: Tab = {
      id: nextId++,
      path: req.path,
      name,
      state: 'loading',
      doc: '',
      language: detectLanguage(req.path),
      line: req.line,
      dirty: false,
      version: 0,
      cursor: { line: 1, col: 1 },
    }
    tabs.push(tab)
    activeId = tab.id
    void load(tab.id)
  }

  async function load(id: number) {
    const tab = tabs.find((x) => x.id === id)
    if (!tab) return
    tab.state = 'loading'
    try {
      const [text, access] = hasRuntime
        ? await Promise.all([readText(tab), filesAccess(win.serverID, tab.path)])
        : [mockRead(tab.path), mockAccess(tab.path)]
      tab.access = access
      const nl = text.indexOf('\n')
      tab.language = detectLanguage(tab.path, nl === -1 ? text : text.slice(0, nl))
      tab.doc = text
      tab.dirty = false
      tab.version++
      tab.state = 'ready'
    } catch (e) {
      tab.error =
        e instanceof Error && e.message === 'binary' ? t.editor.binary : toUserError(e).message
      tab.state = 'error'
    }
  }

  // The preview pretends system files other than nginx's are root's.
  function mockAccess(path: string): Access {
    const writable = !path.startsWith('/etc/') || path.startsWith('/etc/nginx/')
    return { writable, sudoEdit: `sudoedit -- ${path}` }
  }

  function readOnly(tab: Tab): boolean {
    return tab.access ? !tab.access.writable : false
  }

  /** Opens a terminal in the file's folder with sudoedit typed, not run. */
  function editWithSudo(tab: Tab) {
    if (!tab.access) return
    openTerminalAt(
      win.serverID,
      tab.path.slice(0, tab.path.lastIndexOf('/')) || '/',
      tab.access.sudoEdit,
    )
  }

  async function readText(tab: Tab): Promise<string> {
    const { data, ...file } = await filesRead(win.serverID, tab.path)
    const text = decodeText(data)
    if (text === null) throw new Error('binary')
    tab.file = file
    return text
  }

  /**
   * Writes the tab's text; true when it is saved. Unless overwrite is set,
   * it refuses when the server copy changed since it was read.
   */
  async function save(tab: Tab, overwrite = false): Promise<boolean> {
    const view = views[tab.id]
    if (tab.state !== 'ready' || !view || tab.saving || readOnly(tab)) return false
    if (!hasRuntime) {
      tab.dirty = false
      tab.notice = { text: t.editor.saved }
      return true
    }
    tab.saving = true
    tab.notice = { text: t.editor.saving }
    try {
      const data = new TextEncoder().encode(view.getText())
      const expect =
        overwrite || !tab.file ? undefined : { modTime: tab.file.modTime, size: tab.file.size }
      const res = await filesSave(win.serverID, tab.path, data, {
        backup: backupFor(tab.path, settings.editorBackup),
        expect,
      })
      tab.file = res.entry
      tab.dirty = false
      tab.notice = {
        text: res.backup
          ? t.editor.savedWithBackup(res.backup.slice(res.backup.lastIndexOf('/') + 1))
          : t.editor.saved,
      }
      return true
    } catch (e) {
      const ue = toUserError(e)
      tab.notice = { text: t.editor.saveFailed(ue.message), error: true }
      if (ue.code === 'FS_CHANGED') changedId = tab.id
      // Permissions changed, or the owner could not be kept: stop editing.
      if (ue.code === 'FS_PERMISSION_DENIED' && tab.access) tab.access.writable = false
      return false
    } finally {
      tab.saving = false
    }
  }

  function requestCloseTab(id: number) {
    if (tabs.find((x) => x.id === id)?.dirty) closing = { ids: [id], window: false }
    else closeTab(id)
  }

  // Closing the window with unsaved tabs asks first, like a desktop editor.
  $effect(() =>
    wm.guardClose(win.id, () => {
      const dirty = tabs.filter((x) => x.dirty).map((x) => x.id)
      if (dirty.length === 0) return true
      closing = { ids: dirty, window: true }
      return false
    }),
  )

  async function resolveClosing(choice: 'save' | 'discard' | 'cancel') {
    const c = closing
    closing = null
    if (!c || choice === 'cancel') return
    if (choice === 'save') {
      for (const id of c.ids) {
        const tab = tabs.find((x) => x.id === id)
        // A failed save keeps everything open, with the reason on screen.
        if (tab && !(await save(tab))) {
          activeId = tab.id
          return
        }
      }
    }
    if (c.window) wm.close(win.id, { force: true })
    else for (const id of c.ids) closeTab(id)
  }

  async function resolveChanged(choice: 'overwrite' | 'reload' | 'cancel') {
    const tab = changedTab
    changedId = null
    if (!tab) return
    if (choice === 'overwrite') await save(tab, true)
    else if (choice === 'reload') {
      tab.notice = undefined
      await load(tab.id)
    }
  }

  const closingTabs = $derived(closing ? tabs.filter((x) => closing?.ids.includes(x.id)) : [])

  function closeTab(id: number) {
    const i = tabs.findIndex((x) => x.id === id)
    if (i === -1) return
    tabs.splice(i, 1)
    delete views[id]
    if (activeId === id) activeId = tabs[Math.min(i, tabs.length - 1)]?.id ?? null
  }

  // Files asked for before this window existed, then any asked for later.
  for (const req of untrack(() => editorRequests.take(win.serverID))) open(req)
  $effect(() => {
    if (!editorRequests.list.some((r) => r.serverID === win.serverID)) return
    untrack(() => {
      for (const req of editorRequests.take(win.serverID)) open(req)
    })
  })
</script>

<TitleBar>
  <div class="tabs" role="tablist" aria-label={t.editor.tabs}>
    {#each tabs as tab (tab.id)}
      <div class="tab" class:active={tab.id === activeId}>
        <button
          role="tab"
          class="label"
          aria-selected={tab.id === activeId}
          title={tab.path}
          onclick={() => (activeId = tab.id)}
        >
          {tab.name}{#if tab.dirty}<span class="dot" aria-label={t.editor.unsaved}>●</span>{/if}
        </button>
        <button
          class="close"
          aria-label={t.editor.closeTab(tab.name)}
          onclick={() => requestCloseTab(tab.id)}
        >
          <Icon name="close" size={10} strokeWidth={2.5} />
        </button>
      </div>
    {/each}
  </div>
  <span class="spacer"></span>
</TitleBar>

<div class="body">
  {#if tabs.length === 0}
    <Empty icon="editor" title={t.editor.empty} hint={t.editor.emptyHint} />
  {/if}
  {#each tabs as tab (tab.id)}
    <div class="pane" class:hidden={tab.id !== activeId}>
      {#if tab.state === 'loading'}
        <Empty icon="file" title={t.editor.loading} />
      {:else if tab.state === 'error'}
        <div class="failed">
          <Empty icon="warning" title={tab.error ?? ''} />
          <button class="btn" onclick={() => void load(tab.id)}>{t.editor.retry}</button>
        </div>
      {:else}
        {#if readOnly(tab)}
          <div class="readonly" role="note">
            <Icon name="lock" size={14} strokeWidth={2.2} />
            <span>{t.editor.readOnlyBody}</span>
            <button class="btn" title={t.editor.sudoHint} onclick={() => editWithSudo(tab)}
              >{t.editor.sudo}</button
            >
          </div>
        {/if}
        {#key tab.version}
          <CodeEditor
            bind:this={views[tab.id]}
            doc={tab.doc}
            language={tab.language}
            readOnly={readOnly(tab)}
            line={tab.line}
            onchange={(c) => {
              tab.cursor = { line: c.line, col: c.col }
              if (c.edited) {
                tab.dirty = true
                tab.notice = undefined
              }
            }}
            onsave={() => void save(tab)}
          />
        {/key}
      {/if}
    </div>
  {/each}

  {#if active}
    <footer class="status">
      <span class="path" title={active.path}>{server?.name ?? ''}:{active.path}</span>
      {#if active.notice}
        <span class="notice" class:error={active.notice.error} role="status"
          >{active.notice.text}</span
        >
      {/if}
      {#if readOnly(active)}<span class="ro">{t.editor.readOnly}</span>{/if}
      {#if active.state === 'ready'}
        <span>{t.editor.position(active.cursor.line, active.cursor.col)}</span>
        <span>{LANGUAGE_LABELS[active.language]}</span>
      {/if}
    </footer>
  {/if}
</div>

<Dialog
  open={closing !== null}
  title={closingTabs.length === 1
    ? t.editor.unsavedTitle(closingTabs[0]?.name ?? '')
    : t.editor.unsavedMany(closingTabs.length)}
  onclose={() => void resolveClosing('cancel')}
>
  <p class="dialog-text">{t.editor.unsavedBody}</p>
  {#if closingTabs.length > 1}
    <ul class="dialog-list">
      {#each closingTabs as tab (tab.id)}<li>{tab.path}</li>{/each}
    </ul>
  {/if}
  {#snippet footer()}
    <button type="button" class="btn" onclick={() => void resolveClosing('discard')}
      >{t.editor.dontSave}</button
    >
    <span class="grow"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={() => void resolveClosing('cancel')}
      >{t.editor.cancel}</button
    >
    <button type="button" class="btn btn-primary" onclick={() => void resolveClosing('save')}
      >{t.editor.save}</button
    >
  {/snippet}
</Dialog>

<Dialog
  open={changedTab !== undefined}
  title={changedTab ? t.editor.changedTitle(changedTab.name) : ''}
  width={460}
  onclose={() => void resolveChanged('cancel')}
>
  <p class="dialog-text">{t.editor.changedBody}</p>
  {#snippet footer()}
    <button type="button" class="btn" onclick={() => void resolveChanged('reload')}
      >{t.editor.reload}</button
    >
    <span class="grow"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={() => void resolveChanged('cancel')}
      >{t.editor.cancel}</button
    >
    <button type="button" class="btn btn-danger" onclick={() => void resolveChanged('overwrite')}
      >{t.editor.overwrite}</button
    >
  {/snippet}
</Dialog>

<style>
  .readonly {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px 6px 12px;
    border-bottom: 1px solid var(--divider-soft);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .readonly :global(svg) {
    color: var(--warn);
    flex-shrink: 0;
  }

  .readonly span {
    flex: 1;
    min-width: 0;
  }

  .ro {
    color: var(--warn);
    font-weight: 600;
  }

  .notice {
    color: var(--ok);
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .notice.error {
    color: var(--danger);
  }

  .dialog-text {
    margin: 0 0 8px;
    line-height: 1.45;
  }

  .dialog-list {
    margin: 0;
    padding-left: 18px;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .grow {
    flex: 1;
  }

  .tabs {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    overflow: hidden;
  }

  .tab {
    position: relative;
    display: flex;
    align-items: center;
    min-width: 0;
    border-radius: var(--control-radius);
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .tab:hover {
    background: var(--row-hover);
  }

  .tab.active {
    background: var(--tab-active-light);
    color: var(--fg);
  }

  /* Room for the close button on both sides keeps the name centered. */
  .label {
    display: flex;
    align-items: center;
    gap: 5px;
    border: 0;
    background: none;
    padding: 4px 24px;
    color: inherit;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dot {
    font-size: 9px;
    color: var(--fg-muted);
  }

  .close {
    position: absolute;
    left: 5px;
    display: flex;
    border: 0;
    padding: 2px;
    border-radius: 4px;
    background: none;
    color: inherit;
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .tab:hover .close,
  .close:focus-visible {
    opacity: 1;
  }

  .close:hover {
    background: var(--fill-soft);
  }

  .spacer {
    width: 52px;
  }

  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
  }

  .pane {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .pane.hidden {
    display: none;
  }

  .failed {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-bottom: 24px;
  }

  .status {
    flex-shrink: 0;
    display: flex;
    gap: 16px;
    padding: 4px 12px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
