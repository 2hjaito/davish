<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type {
    Authorized,
    AuthorizedKey,
    GitHost,
    GitTestResult,
    LocalKey,
    ServerKey,
  } from '../../lib/api/keys'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import KeyForm, { type FormKind, type FormValue } from './KeyForm.svelte'
  import KeyInspector, { type Selected } from './KeyInspector.svelte'

  type Filter = FormKind
  const FILTERS: Filter[] = ['authorized', 'server', 'local']

  let { win }: AppProps = $props()
  const ctx = getWindowContext()

  const saved = load('keys', { filter: 'authorized' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'authorized')
  let query = $state('')
  let authorized = $state.raw<Authorized | null>(null)
  let serverKeys = $state.raw<ServerKey[] | null>(null)
  let localKeys = $state.raw<LocalKey[]>([])
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)
  let flash = $state<{ text: string; ok: boolean } | null>(null)
  let busy = $state(false)
  /** The last git test per server key path. */
  let gitTests = $state.raw<Record<string, { host: GitHost; result: GitTestResult }>>({})

  let form = $state<FormKind | null>(null)
  let formError = $state('')
  /** A command waiting for the user's yes, with what running it does. */
  let pending = $state.raw<{
    title: string
    body: string
    action: string
    tone: 'danger' | 'primary'
    command: string
    run: () => Promise<void>
  } | null>(null)
  /** A local key about to be used for this server, asking its passphrase. */
  let using = $state.raw<LocalKey | null>(null)
  let passphrase = $state('')

  $effect(() => save('keys', { filter }))

  // Another app asked for a section, such as Git for a deploy key.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'keys' && r.serverID === win.serverID)) return
    untrack(() => {
      const req = selectRequests.take('keys', win.serverID)
      const f = FILTERS.find((x) => x === req?.key)
      if (!f) return
      filter = f
      query = ''
      selected = null
    })
  })

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  function label(k: { comment: string }, fallback: string) {
    return k.comment || fallback
  }

  const rows = $derived.by<(StatusItem & { sel: Selected; text: string })[]>(() => {
    if (filter === 'authorized')
      return (authorized?.keys ?? []).map((k) => ({
        key: k.fingerprint,
        name: label(k, t.keys.noComment),
        health: k.login ? 'ok' : 'off',
        note: k.type.replace('ssh-', ''),
        title: k.fingerprint,
        sel: { kind: 'authorized', key: k },
        text: `${k.comment} ${k.fingerprint}`,
      }))
    const list = filter === 'server' ? (serverKeys ?? []) : localKeys
    return list.map((k) => ({
      key: k.path,
      name: k.name,
      health: 'off',
      note: k.comment || k.type.replace('ssh-', ''),
      title: k.fingerprint,
      sel: (filter === 'server'
        ? { kind: 'server', key: k }
        : { kind: 'local', key: k as LocalKey }) as Selected,
      text: `${k.name} ${k.comment} ${k.fingerprint}`,
    }))
  })

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return rows.filter((r) => !q || r.text.toLowerCase().includes(q))
  })
  const current = $derived(shown.find((r) => r.key === selected)?.sel ?? null)

  const filters = $derived<FilterItem[]>(
    FILTERS.map((f) => ({
      id: f,
      label: t.keys.filters[f],
      icon: f === 'authorized' ? 'lock' : f === 'server' ? 'server' : 'key',
      count:
        f === 'authorized'
          ? authorized?.keys.length
          : f === 'server'
            ? serverKeys?.length
            : localKeys.length,
    })),
  )

  const needsServer = $derived(filter !== 'local')
  const ready = $derived(
    filter === 'local'
      ? true
      : connected && (filter === 'authorized' ? !!authorized : !!serverKeys),
  )

  async function refresh() {
    if (loading) return
    loading = true
    try {
      const [l, a, s] = await Promise.all([
        backend.local(),
        connected ? backend.authorized(win.serverID) : Promise.resolve(null),
        connected ? backend.server(win.serverID) : Promise.resolve(null),
      ])
      localKeys = l
      if (a) authorized = a
      if (s) serverKeys = s
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  function ask(p: NonNullable<typeof pending>) {
    flash = null
    pending = p
  }

  async function confirm() {
    const p = pending
    if (!p) return
    pending = null
    busy = true
    try {
      await p.run()
    } catch (e) {
      const ue = toUserError(e)
      flash = { text: ue.message, ok: false }
      if (ue.code === 'FS_STALE') void refresh()
    } finally {
      busy = false
    }
  }

  async function planAdd(text: string) {
    const plan = await backend.planAdd(win.serverID, text)
    ask({
      title: t.keys.addTitle,
      body: t.keys.addBody(plan.comment || plan.fingerprint),
      action: t.keys.addAction,
      tone: 'primary',
      command: plan.command,
      run: async () => {
        await backend.add(win.serverID, text, plan.command)
        flash = { text: t.keys.done, ok: true }
        await refresh()
      },
    })
  }

  async function submitForm(v: FormValue) {
    busy = true
    formError = ''
    try {
      if (form === 'authorized') {
        await planAdd(v.text)
      } else if (form === 'server') {
        const plan = await backend.planCreate(win.serverID, v.name, v.comment)
        ask({
          title: t.keys.createTitle(plan.name),
          body: t.keys.createBody,
          action: t.keys.createAction,
          tone: 'primary',
          command: plan.command,
          run: async () => {
            await backend.create(win.serverID, plan.name, plan.comment, plan.command)
            await refresh()
            selected = plan.path
          },
        })
      } else {
        const k = await backend.generateLocal(v.name, v.comment, v.passphrase)
        await refresh()
        selected = k.path
        flash = { text: t.keys.generated(k.name), ok: true }
      }
      form = null
    } catch (e) {
      formError = toUserError(e).message
    } finally {
      busy = false
    }
  }

  async function remove(k: AuthorizedKey) {
    try {
      const plan = await backend.planRemove(win.serverID, k.fingerprint)
      ask({
        title: t.keys.removeTitle(plan.comment || plan.fingerprint),
        body: (plan.login ? t.keys.removeLogin : '') + t.keys.removeBody(plan.backup ?? ''),
        action: t.keys.removeAction,
        tone: 'danger',
        command: plan.command,
        run: async () => {
          await backend.remove(win.serverID, plan)
          selected = null
          flash = { text: t.keys.done, ok: true }
          await refresh()
        },
      })
    } catch (e) {
      flash = { text: toUserError(e).message, ok: false }
    }
  }

  async function gitTest(k: ServerKey, host: GitHost) {
    try {
      const plan = await backend.planGitTest(win.serverID, host, k.name)
      ask({
        title: t.keys.gitTestTitle(host),
        body: t.keys.gitTestBody,
        action: t.keys.gitTestAction,
        tone: 'primary',
        command: plan.command,
        run: async () => {
          const result = await backend.gitTest(win.serverID, host, k.name, plan.command)
          gitTests = { ...gitTests, [k.path]: { host, result } }
        },
      })
    } catch (e) {
      flash = { text: toUserError(e).message, ok: false }
    }
  }

  async function push(k: LocalKey) {
    try {
      await planAdd(k.public)
    } catch (e) {
      flash = { text: toUserError(e).message, ok: false }
    }
  }

  // Switches this server to log in with a local key. The key is on the
  // server already (the button is off otherwise), so the next connection
  // works; an encrypted key's passphrase goes to the keychain.
  async function use(k: LocalKey, pass: string) {
    if (!server) return
    using = null
    busy = true
    try {
      await servers.save({
        id: server.id,
        name: server.name,
        host: server.host,
        port: server.port,
        user: server.user,
        auth: 'key',
        keyPath: k.path,
        color: server.color ?? '',
        jumpID: server.jumpID ?? '',
        password: '',
        passphrase: pass,
      })
      flash = { text: t.keys.used(server.name), ok: true }
    } catch (e) {
      flash = { text: toUserError(e).message, ok: false }
    } finally {
      busy = false
      passphrase = ''
    }
  }

  // Load when the window opens or the server comes up, and when the window
  // regains focus; no polling, as every read is a command or SFTP call.
  $effect(() => {
    void connected
    untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused) untrack(() => void refresh())
    wasFocused = f
  })

  useAppMenu(
    () => [],
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => filter, (f) => ((filter = f as Filter), (selected = null), (flash = null))}
  sidebarLabel={t.keys.sections}
  searchLabel={t.keys.search}
  bind:query
  refreshing={loading}
  {ready}
  footer={t.keys.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={t.keys.add[filter]}
      title={t.keys.add[filter]}
      disabled={needsServer && !connected}
      onclick={() => {
        formError = ''
        form = filter
      }}
    >
      <Icon name="plus" size={15} strokeWidth={2.2} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if needsServer && !connected}
      <Empty icon="server" title={t.keys.offline} hint={t.keys.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'key'} title={error || t.keys.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      items={shown}
      {selected}
      label={t.keys.search}
      empty={query ? t.keys.noMatch : t.keys.empty[filter]}
      onselect={(k) => ((selected = k), (flash = null))}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <KeyInspector
        sel={current}
        {busy}
        {flash}
        git={current.kind === 'server' ? (gitTests[current.key.path] ?? null) : null}
        onServer={current.kind === 'local' &&
          !!authorized?.keys.some((a) => a.fingerprint === current.key.fingerprint)}
        inUse={current.kind === 'local' &&
          server?.auth === 'key' &&
          server.keyPath === current.key.path}
        {connected}
        onremove={(k) => void remove(k)}
        ongittest={(k, h) => void gitTest(k, h)}
        onpush={(k) => void push(k)}
        onuse={(k) => (k.encrypted ? (using = k) : void use(k, ''))}
      />
    {:else}
      <Empty icon="key" title={t.keys.noSelection} />
    {/if}
  {/snippet}
</FilterShell>

<KeyForm
  kind={form}
  {busy}
  error={formError}
  local={localKeys}
  defaultComment={form === 'server' && server ? `${server.user}@${server.name}` : 'davish'}
  onclose={() => (form = null)}
  onsubmit={(v) => void submitForm(v)}
/>

<ConfirmCommand
  command={pending?.command ?? null}
  title={pending?.title ?? ''}
  body={pending?.body ?? ''}
  action={pending?.action ?? ''}
  tone={pending?.tone ?? 'danger'}
  onclose={() => (pending = null)}
  onconfirm={() => void confirm()}
/>

<Dialog
  open={using !== null}
  title={t.keys.useForServer}
  width={400}
  onclose={() => ((using = null), (passphrase = ''))}
>
  <form id="use-key" onsubmit={(e) => (e.preventDefault(), using && void use(using, passphrase))}>
    <label class="field">
      <span>{t.keys.usePassphrase}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" type="password" autocomplete="off" autofocus bind:value={passphrase} />
    </label>
  </form>
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => ((using = null), (passphrase = ''))}
      >{t.keys.cancel}</button
    >
    <button type="submit" form="use-key" class="btn btn-primary" disabled={!passphrase}
      >{t.keys.useForServer}</button
    >
  {/snippet}
</Dialog>

<style>
  .spacer {
    flex: 1;
  }
</style>
