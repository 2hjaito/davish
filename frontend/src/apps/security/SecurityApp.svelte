<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { SecurityReport } from '../../lib/api/security'
  import { t } from '../../lib/i18n'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const s = t.security

  let report = $state.raw<SecurityReport | null>(null)
  let error = $state('')
  let loading = $state(false)

  const server = $derived(servers.list.find((x) => x.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const banned = $derived(new Set(report?.fail2ban.banned ?? []))

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const current = attempt()
    loading = true
    try {
      report = await backend.report(win.serverID)
      error = ''
    } catch (e) {
      if (current()) error = toUserError(e).message
    } finally {
      if (current()) loading = false
    }
  }

  // Root commands are only ever typed into a terminal (docs/SECURITY.md).
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
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <button
    class="tool"
    aria-label={t.common.refresh}
    class:spin={loading}
    disabled={!connected}
    onclick={() => void refresh()}
  >
    <Icon name="refresh" size={14} strokeWidth={2.2} />
  </button>
</TitleBar>

<div class="body">
  {#if !connected}
    <Empty icon="server" title={s.offline} hint={s.offlineHint} />
  {:else if !report}
    <LoadState
      serverID={win.serverID}
      {error}
      icon="shield"
      loading={s.loading}
      onretry={() => void refresh()}
    />
  {:else}
    {@const r = report}
    {#if error}<p class="error">{error}</p>{/if}

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
          <pre class="command">{r.disablePassword}</pre>
          <p class="hint strong">{s.password.keepOpen}</p>
          <button class="btn btn-primary" onclick={() => sudo(r.disablePassword)}
            >{t.common.runInTerminal}</button
          >
        {/if}
      {/if}
    </section>

    <section>
      <h3>{s.failed.title(r.failedTotal)}</h3>
      {#if !r.source}
        <p class="hint">{s.noLogs}</p>
      {:else if r.failed.length === 0}
        <p class="hint">{s.failed.none}</p>
      {:else}
        <table>
          <thead>
            <tr
              ><th>{s.ip}</th><th class="num">{s.count}</th><th>{s.users}</th><th>{s.last}</th><th
              ></th></tr
            >
          </thead>
          <tbody>
            {#each r.failed as a (a.ip)}
              <tr>
                <td class="mono"
                  >{a.ip}{#if banned.has(a.ip)}<span class="tag">{s.banned}</span>{/if}</td
                >
                <td class="num">{a.count}</td>
                <td class="mono muted">{a.users.join(', ')}</td>
                <td class="muted">{a.last}</td>
                <td class="act">
                  {#if r.ufw.installed}
                    <button class="link" onclick={() => sudo(`sudo ufw deny from ${a.ip}`)}
                      >{s.block}</button
                    >
                  {:else if r.fail2ban.installed && !banned.has(a.ip)}
                    <button
                      class="link"
                      onclick={() => sudo(`sudo fail2ban-client set sshd banip ${a.ip}`)}
                      >{s.ban}</button
                    >
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
        <p class="hint">{s.source(r.source)}</p>
      {/if}
    </section>

    <section>
      <h3>{s.accepted}</h3>
      {#if r.accepted.length === 0}
        <p class="hint">{r.source ? s.noAccepted : s.noLogs}</p>
      {:else}
        <table>
          <thead
            ><tr><th>{s.time}</th><th>{s.user}</th><th>{s.ip}</th><th>{s.method}</th></tr></thead
          >
          <tbody>
            {#each r.accepted as l, i (i)}
              <tr>
                <td class="muted">{l.time}</td>
                <td>{l.user}</td>
                <td class="mono">{l.ip}</td>
                <td class:warn-text={l.method === 'password'}>{l.method}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
      {#if r.last.length}
        <details>
          <summary>{s.lastTitle}</summary>
          <pre class="raw">{r.last.join('\n')}</pre>
        </details>
      {/if}
      {#if r.lastbDenied}
        <p class="hint">{s.lastbRoot}</p>
        <button class="btn" onclick={() => sudo('sudo lastb -n 50 -i -w')}>{s.lastbRun}</button>
      {:else if r.lastb.length}
        <details>
          <summary>{s.lastbTitle}</summary>
          <pre class="raw">{r.lastb.join('\n')}</pre>
        </details>
      {/if}
    </section>

    <section>
      <h3>{s.firewall}</h3>
      <div class="fw">
        <div>
          <strong>ufw</strong>
          {#if !r.ufw.installed}
            <p class="hint">{s.notInstalled}</p>
          {:else if r.ufw.needsRoot}
            <p class="hint">{s.needsRoot}</p>
            <button class="btn" onclick={() => sudo('sudo ufw status verbose')}
              >{t.common.runInTerminal}</button
            >
          {:else}
            <p class:ok={r.ufw.active} class:warn-text={!r.ufw.active}>
              {r.ufw.active ? s.active : s.inactive}
            </p>
            {#if r.ufw.lines.length}<pre class="raw">{r.ufw.lines.join('\n')}</pre>{/if}
          {/if}
        </div>
        <div>
          <strong>fail2ban</strong>
          {#if !r.fail2ban.installed}
            <p class="hint">{s.notInstalled}</p>
          {:else if r.fail2ban.needsRoot}
            <p class="hint">{s.needsRoot}</p>
            <button class="btn" onclick={() => sudo('sudo fail2ban-client status sshd')}
              >{t.common.runInTerminal}</button
            >
          {:else}
            <p class:ok={r.fail2ban.active}>
              {r.fail2ban.active ? s.jailActive(r.fail2ban.banned?.length ?? 0) : s.inactive}
            </p>
          {/if}
        </div>
      </div>
      <p class="hint">{s.firewallHint}</p>
    </section>
  {/if}
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .tool {
    border: 0;
    background: none;
    padding: 3px;
    display: flex;
    color: var(--fg-muted);
  }

  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 14px 18px 18px;
    background: var(--content-bg);
    display: flex;
    flex-direction: column;
    gap: 18px;
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

  .card {
    padding: 12px 14px;
    border-radius: 10px;
    background: var(--fill-soft);
  }

  .card.warn {
    box-shadow: inset 3px 0 0 var(--warn);
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .strong {
    color: var(--fg);
    font-weight: 600;
  }

  .command,
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
    margin-bottom: 6px;
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

  .muted {
    color: var(--fg-muted);
  }

  .tag {
    margin-left: 6px;
    font-size: var(--text-xxs);
    padding: 0 5px;
    border-radius: 4px;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .act {
    text-align: right;
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
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .ok {
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

  .tool.spin :global(svg) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      rotate: 360deg;
    }
  }
</style>
