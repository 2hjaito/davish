<script lang="ts">
  import { blockedIPs, deniedIPs } from './ufw'
  import { untrack } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'

  import { copyText } from '../../lib/api/clipboard'
  import { toUserError } from '../../lib/api/errors'
  import type { SecurityAction, SecurityPlan, SecurityReport } from '../../lib/api/security'
  import { t } from '../../lib/i18n'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import { sudoPrompt, withSudo } from '../../lib/stores/sudoPrompt.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import BarList from './BarList.svelte'
  import HourBars from './HourBars.svelte'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const s = t.security

  type View =
    'overview' | 'failed' | 'blocked' | 'users' | 'accepted' | 'sessions' | 'firewall' | 'password'
  const icons: Record<View, IconName> = {
    overview: 'shield',
    failed: 'warning',
    blocked: 'lock',
    users: 'help',
    accepted: 'check',
    sessions: 'terminal',
    firewall: 'globe2',
    password: 'key',
  }

  let view = $state<View>('overview')
  let query = $state('')
  let selected = $state<string | null>(null)

  let report = $state.raw<SecurityReport | null>(null)
  let error = $state('')
  let loading = $state(false)

  const server = $derived(servers.list.find((x) => x.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const banned = $derived(new Set(report?.fail2ban.banned ?? []))
  const denied = $derived(deniedIPs(report?.ufw.lines ?? []))
  const blocked = $derived(blockedIPs(denied, banned))
  const attempts = $derived(new Map((report?.failed ?? []).map((a) => [a.ip, a])))
  const isBlocked = (ip: string) => denied.has(ip) || banned.has(ip)

  const filters = $derived.by((): FilterItem[] => {
    const r = report
    const n = (v: View): number | undefined => {
      if (!r) return undefined
      return {
        overview: undefined,
        failed: r.failedIPs,
        blocked: blocked.length,
        users: r.targets.length,
        accepted: r.acceptedTotal,
        sessions: r.sessions?.length ?? 0,
        firewall: undefined,
        password: undefined,
      }[v]
    }
    const item = (v: View, section?: string): FilterItem => ({
      id: v,
      label: s.views[v],
      icon: icons[v],
      count: n(v),
      section,
    })
    return [
      item('overview'),
      item('failed'),
      item('blocked'),
      item('users'),
      item('accepted'),
      item('sessions'),
      item('firewall', s.settings),
      item('password'),
    ]
  })

  const q = $derived(query.trim().toLowerCase())
  const has = (...fields: string[]) => !q || fields.some((f) => f.toLowerCase().includes(q))

  // The two tables of addresses, with a row picked to see in full below.
  const failedShown = $derived((report?.failed ?? []).filter((a) => has(a.ip, ...a.users)))
  const blockedShown = $derived(
    blocked.filter((b) => has(b.ip, ...(attempts.get(b.ip)?.users ?? []))),
  )
  const ipRows = $derived(
    view === 'failed'
      ? failedShown.map((a) => a.ip)
      : view === 'blocked'
        ? blockedShown.map((b) => b.ip)
        : [],
  )
  const ipView = $derived(view === 'failed' || view === 'blocked')
  const current = $derived(ipView && selected && ipRows.includes(selected) ? selected : null)

  /** "2026-11-09T20:27:28+0700" as "2026-11-09 20:27". */
  const shortTime = (v: string) => v.replace('T', ' ').slice(0, 16)

  // Rows ticked for one action on them all; only the shown ones count.
  const ticked = new SvelteSet<string>()
  const tickedShown = $derived(ipRows.filter((ip) => ticked.has(ip)))
  const allTicked = $derived(ipRows.length > 0 && tickedShown.length === ipRows.length)

  function tick(ip: string) {
    if (!ticked.delete(ip)) ticked.add(ip)
  }

  function tickAll() {
    const all = allTicked
    ticked.clear()
    if (!all) for (const ip of ipRows) ticked.add(ip)
  }

  function pick(ip: string) {
    selected = selected === ip ? null : ip
    flash = ''
    actionError = null
  }

  const footer = $derived.by(() => {
    const r = report
    if (!r) return ''
    switch (view) {
      case 'failed':
        return failedShown.length < r.failedIPs && !q
          ? s.ipsOf(failedShown.length, r.failedIPs)
          : s.ips(failedShown.length)
      case 'blocked':
        return s.ips(blockedShown.length)
      case 'users':
        return s.items(usersShown.length)
      case 'accepted':
        return s.items(acceptedShown.length)
      case 'sessions':
        return s.items(sessionsShown.length)
    }
    return ''
  })
  const stats = $derived(
    report
      ? ([
          { icon: 'warning', label: s.stats.failed, value: report.failedTotal, to: 'failed' },
          { icon: 'help', label: s.stats.invalid, value: report.invalidTotal, to: 'users' },
          { icon: 'globe', label: s.stats.ips, value: report.failedIPs, to: 'failed' },
          { icon: 'lock', label: s.stats.banned, value: blocked.length, to: 'blocked' },
          { icon: 'check', label: s.stats.accepted, value: report.acceptedTotal, to: 'accepted' },
        ] as const)
      : [],
  )
  const usersShown = $derived((report?.targets ?? []).filter((u) => has(u.user)))
  const acceptedShown = $derived((report?.accepted ?? []).filter((l) => has(l.ip, l.user)))
  const sessionsShown = $derived((report?.sessions ?? []).filter((x) => has(x.user, x.from)))

  /** Opens a section, with an address in it picked. */
  function open(v: View, ip: string | null = null) {
    view = v
    query = ''
    selected = ip
    ticked.clear()
  }

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const latest = attempt()
    loading = true
    try {
      report = await backend.report(win.serverID)
      error = ''
    } catch (e) {
      if (latest()) error = toUserError(e).message
    } finally {
      if (latest()) loading = false
    }
  }

  // What needs a password is typed into a terminal, for the user to run.
  const sudo = (cmd: string) => openTerminalAt(win.serverID, undefined, cmd)

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

  let copied = $state('')
  async function copy(ip: string) {
    if (await copyText(ip)) {
      copied = ip
      setTimeout(() => copied === ip && (copied = ''), 1500)
    }
  }

  // An action shows its exact command, then runs once confirmed: with
  // sudo -n when sudo needs no password, in Terminal otherwise (ADR 0016).
  // Several addresses share one dialog that lists every command.
  type Item = { action: SecurityAction; ip: string }
  type Step = Item & { plan: SecurityPlan }
  let confirm = $state.raw<[Step, ...Step[]] | null>(null)
  let running = $state(false)
  let flash = $state('')
  let actionError = $state.raw<{ message: string; terminal: string } | null>(null)
  const canRun = $derived(!!report && (report.root || report.sudo || report.canSudo))

  /** Reads what needs root once the user gives the sudo password. */
  async function unlock() {
    const read = report?.sudoRead
    if (read && (await sudoPrompt.ask(win.serverID, read))) await refresh()
  }

  /** What a row's action does to an address: ufw first, fail2ban otherwise. */
  function rowAction(r: SecurityReport, ip: string): SecurityAction | null {
    if (r.ufw.installed) return denied.has(ip) ? 'unblock' : 'block'
    if (r.fail2ban.installed) return banned.has(ip) ? 'unban' : 'ban'
    return null
  }

  // The ticked rows the section's action applies to: blocking in Failed,
  // lifting in Blocked.
  const bulk = $derived.by((): Item[] => {
    const r = report
    if (!r) return []
    const lift = view === 'blocked'
    return tickedShown.flatMap((ip) => {
      const a = rowAction(r, ip)
      return a && (a === 'unblock' || a === 'unban') === lift ? [{ action: a, ip }] : []
    })
  })

  async function ask(first: Item, ...more: Item[]) {
    actionError = null
    flash = ''
    const root = report?.root ?? false
    const plan = async (x: Item): Promise<Step> => ({
      ...x,
      plan: await backend.plan(x.action, x.ip, root),
    })
    try {
      const [head, tail] = await Promise.all([plan(first), Promise.all(more.map(plan))])
      confirm = [head, ...tail]
    } catch (e) {
      actionError = { message: toUserError(e).message, terminal: '' }
    }
  }

  const joined = (steps: Step[], k: keyof SecurityPlan, sep: string) =>
    steps.map((x) => x.plan[k]).join(sep)

  function toTerminal() {
    if (confirm) sudo(joined(confirm, 'terminal', ' && '))
    confirm = null
  }

  async function run() {
    const c = confirm
    if (!c) return
    if (!canRun) return toTerminal()
    running = true
    const done: string[] = []
    let message = ''
    let rest: Step[] = []
    for (const [i, x] of c.entries()) {
      try {
        // Through sudo with the session's password when sudo asks for one;
        // the prompt shows the command with sudo.
        const apply = () => backend.apply(win.serverID, x.action, x.ip, x.plan.command)
        if (!(await withSudo(win.serverID, x.plan.terminal, apply))) {
          rest = [...rest, ...c.slice(i)]
          break
        }
        done.push(x.ip)
      } catch (e) {
        const u = toUserError(e)
        message =
          u.code === 'CMD_NEEDS_ROOT'
            ? s.needsPassword
            : u.code === 'CMD_FAILED'
              ? `${u.message} ${u.detail}`
              : u.message
        // Without a password nothing else can run either.
        if (u.code === 'CMD_NEEDS_ROOT') {
          rest = [...rest, ...c.slice(i)]
          break
        }
        rest.push(x)
      }
    }
    confirm = null
    running = false
    if (done.length) {
      flash = c.length === 1 ? s.done[c[0].action](c[0].ip) : s.doneMany(done.length)
      for (const ip of done) ticked.delete(ip)
    }
    if (message)
      actionError = {
        message: c.length === 1 ? message : `${s.failedMany(rest.map((x) => x.ip))} ${message}`,
        terminal: joined(rest, 'terminal', ' && '),
      }
    if (done.length) await refresh()
  }

  const confirmNote = $derived.by(() => {
    const c = confirm
    if (!c || !report) return ''
    const blocking = c.filter((x) => x.action === 'block' || x.action === 'ban')
    if (blocking.some((x) => report?.sessions?.some((z) => z.from === x.ip)))
      return s.confirm.ownSession
    if (c.some((x) => x.action === 'block') && !report.ufw.active && !report.ufw.needsRoot)
      return s.confirm.ufwInactive
    return ''
  })

  const confirmTitle = $derived.by(() => {
    const c = confirm
    if (!c) return ''
    if (c.length === 1) return s.confirm.title[c[0].action](c[0].ip)
    const lift = c[0].action === 'unblock' || c[0].action === 'unban'
    return lift ? s.confirm.liftMany(c.length) : s.confirm.blockMany(c.length)
  })
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => view, (f) => open(f as View)}
  sidebarLabel={s.sections}
  searchLabel={s.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && report !== null}
  {footer}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={s.offline} hint={s.offlineHint} />
    {:else}
      <LoadState
        serverID={win.serverID}
        {error}
        icon="shield"
        loading={s.loading}
        onretry={() => void refresh()}
      />
    {/if}
  {/snippet}

  {#snippet list()}
    {@const r = report!}
    {#if ipView}
      {#if view === 'blocked' && r.ufw.installed && r.ufw.needsRoot}
        <div class="note">
          <p class="hint">{s.ufwHidden}</p>
          {#if r.askSudo}{@render unlockButton()}{/if}
          <button class="btn" onclick={() => sudo('sudo ufw status numbered')}>{s.ufwShow}</button>
        </div>
      {/if}
      {@render banner()}
      <div class="ips">
        {#if ipRows.length === 0}
          <p class="hint">
            {q ? s.noMatch : view === 'blocked' ? s.noBlocked : r.source ? s.failed.none : s.noLogs}
          </p>
        {:else}
          <table class="iptable">
            <thead>
              <tr>
                <th class="tick">
                  <input
                    type="checkbox"
                    aria-label={s.tickAll}
                    checked={allTicked}
                    indeterminate={tickedShown.length > 0 && !allTicked}
                    onchange={tickAll}
                  />
                </th>
                <th>{s.ip}</th>
                <th class="num">{s.count}</th>
                <th>{s.tried}</th>
                <th>{s.last}</th>
                <th>{s.status}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {#each ipRows as ip (ip)}
                {@const a = attempts.get(ip)}
                <tr class:sel={current === ip} onclick={() => pick(ip)}>
                  <td class="tick">
                    <input
                      type="checkbox"
                      aria-label={ip}
                      checked={ticked.has(ip)}
                      onclick={(e) => e.stopPropagation()}
                      onchange={() => tick(ip)}
                    />
                  </td>
                  <td class="mono ipcell">{@render ipCell(ip)}</td>
                  <td class="num">{a ? a.count.toLocaleString() : ''}</td>
                  <td class="mono users selectable" title={a?.users.join(', ')}
                    >{a?.users.join(', ') ?? ''}</td
                  >
                  <td class="muted nowrap selectable">{a ? shortTime(a.last) : ''}</td>
                  <td class="nowrap">{@render statusTags(ip)}</td>
                  <td class="act">{@render quickAction(r, ip)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
      {#if tickedShown.length}
        <div class="bulk">
          <span>{s.ticked(tickedShown.length)}</span>
          {#if bulk.length}
            <button
              class="link"
              class:danger={view === 'failed'}
              onclick={() => bulk[0] && void ask(bulk[0], ...bulk.slice(1))}
              >{view === 'failed'
                ? s.confirm.blockMany(bulk.length).replace(/\?$/, '')
                : s.confirm.liftMany(bulk.length).replace(/\?$/, '')}</button
            >
          {/if}
          <button class="link" onclick={() => ticked.clear()}>{s.untick}</button>
        </div>
      {/if}
      {#if current}
        <div class="below">{@render ipDetail(r, current)}</div>
      {/if}
    {:else}
      <div class="page">
        {#if view === 'overview'}
          {@render overview(r)}
        {:else if view === 'users'}
          {#if usersShown.length === 0}
            <p class="hint">{q ? s.noMatch : r.source ? s.failed.none : s.noLogs}</p>
          {:else}
            <table>
              <thead><tr><th>{s.user}</th><th class="num">{s.count}</th></tr></thead>
              <tbody>
                {#each usersShown as u (u.user)}
                  <tr>
                    <td class="mono selectable">{u.user}</td>
                    <td class="num">{u.count.toLocaleString()}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        {:else if view === 'accepted'}
          {@render accepted(r)}
        {:else if view === 'sessions'}
          {#if sessionsShown.length === 0}
            <p class="hint">{q ? s.noMatch : s.noSessions}</p>
          {:else}
            <table>
              <thead>
                <tr><th>{s.user}</th><th>{s.tty}</th><th>{s.from}</th><th>{s.since}</th></tr>
              </thead>
              <tbody>
                {#each sessionsShown as x, i (i)}
                  <tr>
                    <td>{x.user}</td>
                    <td class="mono muted">{x.tty}</td>
                    <td class="mono"
                      >{#if x.from}{@render ipCell(x.from)}{:else}{s.local}{/if}</td
                    >
                    <td class="muted">{x.since}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        {:else if view === 'firewall'}
          {@render firewall(r)}
        {:else if view === 'password'}
          {@render password(r)}
        {/if}
      </div>
    {/if}
  {/snippet}
</FilterShell>

{#snippet banner()}
  {#if flash || actionError}
    <div class="banner">
      {#if flash}<p class="ok" role="status"><Icon name="check" size={13} /> {flash}</p>{/if}
      {#if actionError}
        <p class="error" role="alert">{actionError.message}</p>
        {#if actionError.terminal}
          <button class="btn" onclick={() => actionError && sudo(actionError.terminal)}
            >{t.common.runInTerminal}</button
          >
        {/if}
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet statusTags(ip: string)}
  <span class="tags">
    {#if denied.has(ip)}<span class="tag bad">{s.blocked}</span>{/if}
    {#if banned.has(ip)}<span class="tag bad">{s.banned}</span>{/if}
    {#if !isBlocked(ip)}<span class="tag">{s.notBlocked}</span>{/if}
  </span>
{/snippet}

<!-- The one action a row offers: ufw when installed, fail2ban otherwise. -->
{#snippet quickAction(r: SecurityReport, ip: string)}
  {@const a = rowAction(r, ip)}
  {#if a}
    <button
      class="link"
      class:danger={a === 'block' || a === 'ban'}
      onclick={(e) => {
        e.stopPropagation()
        void ask({ action: a, ip })
      }}>{s.action[a]}</button
    >
  {/if}
{/snippet}

{#snippet ipDetail(r: SecurityReport, ip: string)}
  {@const a = attempts.get(ip)}
  {@const logins = r.accepted.filter((l) => l.ip === ip).length}
  <div class="detail">
    <header>
      <h3 class="mono selectable">{ip}</h3>
      {@render statusTags(ip)}
    </header>
    {#if a}
      <dl>
        <dt>{s.count}</dt>
        <dd>{a.count.toLocaleString()}</dd>
        <dt>{s.lastTry}</dt>
        <dd class="selectable">{a.last}</dd>
        <dt>{s.tried}</dt>
        <dd class="mono selectable">{a.users.join(', ')}</dd>
      </dl>
    {/if}
    {#if logins > 0}<p class="warn-text">{s.acceptedFrom(logins)}</p>{/if}

    <div class="row">
      {#if r.ufw.installed}
        {#if denied.has(ip)}
          <button class="btn" onclick={() => void ask({ action: 'unblock', ip })}
            >{s.action.unblock}</button
          >
        {:else}
          <button class="btn btn-danger" onclick={() => void ask({ action: 'block', ip })}
            >{s.action.block}</button
          >
        {/if}
      {/if}
      {#if r.fail2ban.installed}
        {#if banned.has(ip)}
          <button class="btn" onclick={() => void ask({ action: 'unban', ip })}
            >{s.action.unban}</button
          >
        {:else}
          <button
            class="btn"
            class:btn-danger={!r.ufw.installed}
            onclick={() => void ask({ action: 'ban', ip })}>{s.action.ban}</button
          >
        {/if}
      {/if}
    </div>
    {#if !r.ufw.installed && !r.fail2ban.installed}
      <p class="hint">ufw, fail2ban: {s.notInstalled}</p>
    {/if}
  </div>
{/snippet}

{#snippet ipCell(ip: string)}
  <span class="selectable">{ip}</span>
  <button
    class="copy"
    aria-label={s.copyIP}
    title={copied === ip ? s.copied : s.copyIP}
    onclick={(e) => {
      e.stopPropagation()
      void copy(ip)
    }}
  >
    <Icon name={copied === ip ? 'check' : 'copy'} size={12} />
  </button>
{/snippet}

{#snippet overview(r: SecurityReport)}
  {#if r.source}
    <div class="stats">
      {#each stats as st (st.label)}
        <button class="tile stat" onclick={() => open(st.to)}>
          <span class="tile-label"><Icon name={st.icon} size={13} />{st.label}</span>
          <span class="hero">{st.value.toLocaleString()}</span>
        </button>
      {/each}
    </div>
    {#if r.hourly?.length}
      <section class="tile">
        <HourBars hours={r.hourly} />
      </section>
    {/if}
    {#if r.failed.length}
      <div class="pair">
        <section class="tile">
          <BarList
            title={s.topIPs}
            rows={r.failed.slice(0, 8).map((a) => ({ key: a.ip, count: a.count }))}
          >
            {#snippet label(row)}
              <button class="iplink mono" onclick={() => open('failed', row.key)}>{row.key}</button>
              {#if denied.has(row.key)}<span class="tag bad">{s.blocked}</span>{/if}
              {#if banned.has(row.key)}<span class="tag bad">{s.banned}</span>{/if}
            {/snippet}
          </BarList>
        </section>
        <section class="tile">
          <BarList
            title={s.topUsers}
            rows={r.targets.slice(0, 8).map((u) => ({ key: u.user, count: u.count }))}
          >
            {#snippet label(row)}<span class="mono">{row.key}</span>{/snippet}
          </BarList>
        </section>
      </div>
    {/if}
  {:else}
    <p class="hint">{s.noLogs}</p>
  {/if}
  {#if r.passwordAuth !== 'no'}
    <button class="card warn" onclick={() => open('password')}>
      <strong>{s.password.title}</strong>
      <span>{r.passwordAuth === 'yes' ? s.password.on : s.password.unknown}</span>
    </button>
  {/if}
{/snippet}

{#snippet accepted(r: SecurityReport)}
  {#if acceptedShown.length === 0}
    <p class="hint">{q ? s.noMatch : r.source ? s.noAccepted : s.noLogs}</p>
  {:else}
    <table>
      <thead><tr><th>{s.time}</th><th>{s.user}</th><th>{s.ip}</th><th>{s.method}</th></tr></thead>
      <tbody>
        {#each acceptedShown as l, i (i)}
          <tr>
            <td class="muted">{l.time}</td>
            <td>{l.user}</td>
            <td class="mono">{@render ipCell(l.ip)}</td>
            <td class:warn-text={l.method === 'password'}>{l.method}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    {#if r.source}<p class="hint">{s.source(r.source)}</p>{/if}
  {/if}
  {#if r.last.length}
    <details>
      <summary>{s.lastTitle}</summary>
      <pre class="raw selectable">{r.last.join('\n')}</pre>
    </details>
  {/if}
  {#if r.lastbDenied}
    <p class="hint">{s.lastbRoot}</p>
    <button class="btn" onclick={() => sudo('sudo lastb -n 50 -i -w')}>{s.lastbRun}</button>
  {:else if r.lastb.length}
    <details>
      <summary>{s.lastbTitle}</summary>
      <pre class="raw selectable">{r.lastb.join('\n')}</pre>
    </details>
  {/if}
{/snippet}

{#snippet unlockButton()}
  <button class="btn btn-primary" onclick={() => void unlock()}>
    <Icon name="key" size={13} />
    {t.sudo.journal}
  </button>
{/snippet}

{#snippet firewall(r: SecurityReport)}
  <div class="fw">
    <section class="tile">
      <h3>ufw</h3>
      {#if !r.ufw.installed}
        <p class="hint">{s.notInstalled}</p>
      {:else if r.ufw.needsRoot}
        <p class="hint">{s.needsRoot}</p>
        {#if r.askSudo}{@render unlockButton()}{/if}
        <button class="btn" onclick={() => sudo('sudo ufw status verbose')}
          >{t.common.runInTerminal}</button
        >
      {:else}
        <p class:ok={r.ufw.active} class:warn-text={!r.ufw.active}>
          {r.ufw.active ? s.active : s.inactive}
        </p>
        {#if denied.size}
          <button class="link" onclick={() => open('blocked')}
            >{s.views.blocked}: {denied.size}</button
          >
        {/if}
      {/if}
    </section>
    <section class="tile">
      <h3>fail2ban</h3>
      {#if !r.fail2ban.installed}
        <p class="hint">{s.notInstalled}</p>
      {:else if r.fail2ban.needsRoot}
        <p class="hint">{s.needsRoot}</p>
        {#if r.askSudo}{@render unlockButton()}{/if}
        <button class="btn" onclick={() => sudo('sudo fail2ban-client status sshd')}
          >{t.common.runInTerminal}</button
        >
      {:else}
        <p class:ok={r.fail2ban.active}>
          {r.fail2ban.active ? s.jailActive(banned.size) : s.inactive}
        </p>
        {#if banned.size}
          <button class="link" onclick={() => open('blocked')}
            >{s.views.blocked}: {banned.size}</button
          >
        {/if}
      {/if}
    </section>
  </div>
  {#if r.ufw.lines.length}
    <h3>{s.ufwRules}</h3>
    <pre class="raw selectable">{r.ufw.lines.join('\n')}</pre>
  {/if}
  <p class="hint">{s.firewallHint}</p>
{/snippet}

{#snippet password(r: SecurityReport)}
  <section class="card" class:warn={r.passwordAuth !== 'no'}>
    <h3>{s.password.title}</h3>
    {#if r.passwordAuth === 'no'}
      <p>{s.password.off}</p>
    {:else}
      <p>{r.passwordAuth === 'yes' ? s.password.on : s.password.unknown}</p>
      {#if server?.auth === 'password'}
        <p class="hint">{s.password.useKeyFirst}</p>
      {:else}
        <p class="hint">{s.password.how}</p>
        <pre class="raw selectable">{r.disablePassword}</pre>
        <p class="hint strong">{s.password.keepOpen}</p>
        <button class="btn btn-primary" onclick={() => sudo(r.disablePassword)}
          >{t.common.runInTerminal}</button
        >
      {/if}
    {/if}
  </section>
{/snippet}

<ConfirmCommand
  command={confirm
    ? canRun
      ? // What runs: as it is for root, else through sudo (ADR 0018), which
        // the plain sudo form shows; never the sudo -n form.
        joined(confirm, report?.root ? 'command' : 'terminal', '\n')
      : joined(confirm, 'terminal', ' && ')
    : null}
  title={confirmTitle}
  body={!confirm
    ? ''
    : !canRun
      ? s.confirm.needsPassword
      : confirm.length === 1
        ? s.confirm.body[confirm[0].action]
        : s.confirm.bodyMany}
  action={!confirm
    ? ''
    : !canRun
      ? s.confirm.openTerminal
      : running
        ? s.working
        : confirm.length === 1
          ? s.action[confirm[0].action]
          : confirmTitle.replace(/\?$/, '')}
  tone={confirm?.[0].action === 'block' || confirm?.[0].action === 'ban' ? 'danger' : 'primary'}
  note={confirmNote}
  onterminal={canRun ? toTerminal : undefined}
  terminalLabel={s.confirm.inTerminal}
  busy={running}
  onclose={() => !running && (confirm = null)}
  onconfirm={() => void run()}
/>

<style>
  .page {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 14px 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .ips {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 6px 12px 8px;
  }

  .ips > .hint {
    padding: 10px 4px;
  }

  .iptable td {
    padding: 5px 6px;
  }

  .iptable tbody tr:hover {
    background: var(--fill-soft);
  }

  .iptable tr.sel {
    background: color-mix(in srgb, var(--accent) 16%, transparent);
  }

  .ipcell {
    white-space: nowrap;
  }

  .users {
    max-width: 0;
    width: 40%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--fg-muted);
  }

  .nowrap {
    white-space: nowrap;
  }

  .act {
    text-align: right;
    white-space: nowrap;
  }

  .tick {
    width: 1px;
    padding-right: 2px;
  }

  .tick input {
    margin: 0;
    vertical-align: middle;
    accent-color: var(--accent);
  }

  .bulk {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 7px 16px;
    border-top: 1px solid var(--divider);
    font-size: var(--text-sm);
  }

  .bulk span {
    flex: 1;
    color: var(--fg-muted);
  }

  .bulk .link {
    font-size: var(--text-sm);
  }

  .act .link {
    font-size: var(--text-sm);
  }

  .act .link:hover {
    text-decoration: underline;
  }

  .link.danger {
    color: var(--danger);
  }

  .below {
    flex: 0 1 auto;
    max-height: 45%;
    overflow: auto;
    border-top: 1px solid var(--divider);
  }

  .detail {
    padding: 12px 16px 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .detail header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .detail h3 {
    margin: 0;
    min-width: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    text-transform: none;
    color: var(--fg);
    overflow-wrap: anywhere;
  }

  .note,
  .banner {
    padding: 10px 16px;
    border-bottom: 1px solid var(--divider-soft);
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .note p,
  .banner p {
    margin: 0;
  }

  h3 {
    margin: 0 0 8px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  p {
    margin: 0 0 6px;
    font-size: var(--text-sm);
  }

  dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 4px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
    gap: 10px;
  }

  .tile {
    padding: 12px 14px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
    min-width: 0;
  }

  .stat {
    text-align: left;
    font: inherit;
    color: inherit;
    transition: border-color var(--dur-fast) var(--ease-out);
  }

  .stat:hover {
    border-color: var(--accent);
  }

  .tile-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .tile-label :global(svg) {
    color: var(--accent);
  }

  .hero {
    display: block;
    margin-top: 6px;
    font-size: 26px;
    font-weight: 700;
  }

  .pair {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 10px;
  }

  .card {
    padding: 12px 14px;
    border: 0;
    border-radius: 10px;
    background: var(--fill-soft);
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: var(--text-sm);
  }

  .card.warn {
    box-shadow: inset 3px 0 0 var(--warn);
  }

  .iplink {
    border: 0;
    background: none;
    padding: 0;
    color: var(--accent);
    font-size: inherit;
  }

  .copy {
    border: 0;
    background: none;
    padding: 2px;
    margin-left: 4px;
    vertical-align: middle;
    color: var(--fg-muted);
    opacity: 0;
  }

  tr:hover .copy,
  tr.sel .copy,
  .copy:focus-visible {
    opacity: 1;
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .strong {
    color: var(--fg);
    font-weight: 600;
  }

  .raw {
    margin: 0 0 8px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--text-sm);
  }

  th {
    text-align: left;
    font-weight: 500;
    font-size: var(--text-xs);
    color: var(--fg-label);
    padding: 4px 6px;
    border-bottom: 1px solid var(--divider);
  }

  td {
    padding: 4px 6px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .mono {
    font-family: var(--font-mono);
  }

  .muted {
    color: var(--fg-muted);
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin: 0;
  }

  .tag {
    font-size: var(--text-xxs);
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .tag.bad {
    background: color-mix(in srgb, var(--danger) 14%, transparent);
    color: var(--danger);
  }

  .link {
    border: 0;
    background: none;
    padding: 0;
    color: var(--accent);
    font-size: var(--text-xs);
  }

  .fw {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 10px;
  }

  .ok {
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--ok);
  }

  .warn-text {
    color: var(--warn);
  }

  .error {
    color: var(--danger);
  }

  summary {
    font-size: var(--text-sm);
    cursor: default;
    margin: 6px 0;
  }
</style>
