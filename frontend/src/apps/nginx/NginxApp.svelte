<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type {
    NginxReload,
    NginxSite,
    NginxTest,
    RemovePlan,
    TogglePlan,
  } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ResultDialog from './ResultDialog.svelte'
  import NewSiteDialog from './NewSiteDialog.svelte'
  import SSLDialog from './SSLDialog.svelte'
  import EditSiteDialog from './EditSiteDialog.svelte'
  import RemoveDialog from './RemoveDialog.svelte'
  import SiteInspector from './SiteInspector.svelte'
  import { type Filter, FILTERS, mainName, matches, siteKey, visible, worstCert } from './sites'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('nginx', { filter: 'all' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'all')
  let query = $state('')
  let sites = $state.raw<NginxSite[] | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)
  let busy = $state(false)
  let toggling = $state.raw<TogglePlan | null>(null)
  let removing = $state.raw<RemovePlan | null>(null)
  let creating = $state(false)
  let sslNames = $state.raw<string[] | null>(null)
  let editing = $state.raw<NginxSite | null>(null)
  /** The server refused the link change: its sudo form. */
  let toggleSudo = $state('')
  let flash = $state('')
  let reloadCommand = $state<string | null>(null)
  let result = $state.raw<
    { kind: 'test'; test: NginxTest } | { kind: 'reload'; reload: NginxReload } | null
  >(null)

  $effect(() => save('nginx', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(sites ? visible(sites, filter, query) : [])
  const current = $derived(sites?.find((s) => siteKey(s) === selected) ?? null)

  const filters = $derived<FilterItem[]>(
    FILTERS.map((f) => ({
      id: f,
      label: t.nginx.filters[f],
      icon: f === 'all' ? 'globe2' : f === 'enabled' ? 'play' : 'pause',
      count: sites?.filter((s) => matches(s, f)).length,
    })),
  )

  const items = $derived<StatusItem[]>(
    shown.map((s) => {
      const cert = worstCert(s)
      return {
        key: siteKey(s),
        name: s.name,
        // An expiring certificate outweighs "on" in the dot.
        health: !s.enabled
          ? 'off'
          : cert === 'expired'
            ? 'failed'
            : cert === 'soon'
              ? 'busy'
              : 'ok',
        note: mainName(s),
        title: s.path,
      }
    }),
  )

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const current = attempt()
    loading = true
    try {
      sites = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      if (current()) error = toUserError(e).message
    } finally {
      if (current()) loading = false
    }
  }

  async function askToggle(site: NginxSite) {
    toggleSudo = ''
    flash = ''
    try {
      toggling = await backend.planToggle(win.serverID, site.name, !site.enabled)
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  async function toggle() {
    const plan = toggling
    if (!plan) return
    toggling = null
    busy = true
    try {
      await backend.toggle(win.serverID, plan)
      flash = t.nginx.toggled[plan.enable ? 'enable' : 'disable'](plan.name)
    } catch (e) {
      const ue = toUserError(e)
      if (ue.code === 'CMD_NEEDS_ROOT') toggleSudo = plan.sudo
      else flash = ue.message
    } finally {
      busy = false
    }
    await refresh()
  }

  async function askRemove(site: NginxSite) {
    toggleSudo = ''
    flash = ''
    try {
      removing = await backend.planRemove(win.serverID, site.path, false)
    } catch (e) {
      flash = toUserError(e).message
    }
  }

  async function remove(plan: RemovePlan) {
    removing = null
    busy = true
    try {
      await backend.remove(win.serverID, plan)
      // A site nginx was serving keeps being served until a reload.
      flash = plan.enabled ? t.nginx.removedEnabled(plan.name) : ''
    } catch (e) {
      const ue = toUserError(e)
      if (ue.code === 'CMD_NEEDS_ROOT') toggleSudo = plan.sudo
      else flash = ue.message
    } finally {
      busy = false
    }
    await refresh()
  }

  async function test() {
    busy = true
    try {
      result = { kind: 'test', test: await backend.test(win.serverID) }
    } catch (e) {
      flash = toUserError(e).message
    } finally {
      busy = false
    }
  }

  async function askReload() {
    reloadCommand = await backend.reloadCommand()
  }

  async function reload() {
    reloadCommand = null
    busy = true
    try {
      const r = await backend.reload(win.serverID)
      result = { kind: 'reload', reload: r }
      if (r.reloaded) flash = ''
    } catch (e) {
      flash = toUserError(e).message
    } finally {
      busy = false
    }
  }

  // Load when the server comes up and when the window regains focus.
  $effect(() => {
    if (connected) untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
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
  bind:filter={() => filter, (f) => (filter = f as Filter)}
  sidebarLabel={t.nginx.sections}
  searchLabel={t.nginx.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && sites !== null}
  footer={t.nginx.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={t.nginx.newSite.open}
      title={t.nginx.newSite.open}
      disabled={!connected || busy}
      onclick={() => (creating = true)}
    >
      <Icon name="plus" size={15} strokeWidth={2.4} />
    </button>
    <button
      class="tool"
      aria-label={t.nginx.test}
      title={t.nginx.test}
      disabled={!connected || busy}
      onclick={() => void test()}
    >
      <Icon name="check" size={15} strokeWidth={2.4} />
    </button>
    <button
      class="tool"
      aria-label={t.nginx.reload}
      title={t.nginx.reload}
      disabled={!connected || busy}
      onclick={() => void askReload()}
    >
      <Icon name="zap" size={15} strokeWidth={2.2} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.nginx.offline} hint={t.nginx.offlineHint} />
    {:else}
      <LoadState
        serverID={win.serverID}
        {error}
        icon="globe2"
        loading={t.nginx.loading}
        onretry={() => void refresh()}
      />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      {items}
      {selected}
      label={t.nginx.search}
      empty={t.nginx.noMatch}
      onselect={(k) => {
        selected = k
        flash = ''
        toggleSudo = ''
      }}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      {#if toggleSudo}
        <div class="root"><NeedsRoot serverID={win.serverID} command={toggleSudo} /></div>
      {/if}
      <SiteInspector
        serverID={win.serverID}
        site={current}
        {busy}
        {flash}
        ontoggle={(s) => void askToggle(s)}
        onremove={(s) => void askRemove(s)}
        onreload={() => void askReload()}
        onedit={(s) => (editing = s)}
        onssl={(s) => (sslNames = [...new Set(s.servers.flatMap((b) => b.names))])}
      />
    {:else}
      <Empty icon="globe2" title={t.nginx.noSelection} />
    {/if}
  {/snippet}
</FilterShell>

<ConfirmCommand
  command={toggling?.command ?? null}
  title={toggling
    ? t.nginx.confirmTitle[toggling.enable ? 'enable' : 'disable'](toggling.name)
    : ''}
  body={t.nginx.confirmBody[toggling?.enable ? 'enable' : 'disable']}
  action={toggling?.enable ? t.nginx.enable : t.nginx.disable}
  tone={toggling?.enable ? 'primary' : 'danger'}
  onclose={() => (toggling = null)}
  onconfirm={() => void toggle()}
/>
<ConfirmCommand
  command={reloadCommand}
  title={t.nginx.reloadTitle}
  body={t.nginx.reloadBody}
  action={t.nginx.reloadAction}
  tone="primary"
  onclose={() => (reloadCommand = null)}
  onconfirm={() => void reload()}
/>
<NewSiteDialog
  serverID={win.serverID}
  open={creating}
  onclose={() => (creating = false)}
  oncreated={(name, path) => {
    flash = t.nginx.newSite.created(name)
    selected = path
    void refresh()
  }}
/>
<EditSiteDialog
  serverID={win.serverID}
  site={editing}
  onclose={() => (editing = null)}
  ondone={(name) => {
    flash = t.nginx.edit.done(name)
    void refresh()
  }}
/>
<SSLDialog serverID={win.serverID} names={sslNames} onclose={() => (sslNames = null)} />
<ResultDialog serverID={win.serverID} {result} onclose={() => (result = null)} />
<RemoveDialog
  plan={removing}
  replan={(withCerts) => backend.planRemove(win.serverID, removing!.path, withCerts)}
  onclose={() => (removing = null)}
  onconfirm={(p) => void remove(p)}
/>

<style>
  .root {
    padding: 12px 16px 0;
  }
</style>
