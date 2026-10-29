<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity'

  import type {
    ActionPlan,
    ComposeAction,
    Container,
    ContainerAction,
    ContainerStats,
  } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import ActionMenu from '../../lib/ui/ActionMenu.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import {
    openFileInEditor,
    openInBrowser,
    openLogs,
    openTerminalAt,
  } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { ago, canDo, type Group, groups, health, parsePorts } from './containers'

  interface Props {
    serverID: string
    containers: readonly Container[]
    stats: Record<string, ContainerStats>
    /** Opens a container's detail page. */
    onopen: (name: string) => void
    /** Shows an image's layers. */
    onimage: (ref: string) => void
    onchanged: () => void
  }

  let { serverID, containers, stats, onopen, onimage, onchanged }: Props = $props()

  const d = t.docker
  const tb = t.docker.table
  // Compose projects start folded, like Docker Desktop.
  const unfolded = new SvelteSet<string>()

  let confirming = $state.raw<ActionPlan | null>(null)
  let busy = $state<string | null>(null)
  let error = $state('')
  /** The command to run with sudo, after the server refused for lack of root. */
  let sudo = $state('')

  const shown = $derived(groups(containers))

  const running = (c: Container) =>
    c.state === 'running' || c.state === 'restarting' || c.state === 'paused'

  function groupHealth(g: Group) {
    const hs = g.containers.map((c) => health(c.state))
    if (hs.includes('failed')) return 'failed'
    if (hs.every((h) => h === 'ok')) return 'ok'
    if (hs.includes('ok') || hs.includes('busy')) return 'busy'
    return 'off'
  }

  function sum(g: Group, f: (s: ContainerStats) => number): number | undefined {
    const ss = g.containers.map((c) => stats[c.id]).filter((s) => s !== undefined)
    return ss.length ? ss.reduce((a, s) => a + f(s), 0) : undefined
  }

  async function act(c: Container, action: ContainerAction) {
    error = ''
    sudo = ''
    try {
      const plan = await backend.plan(c.name, action)
      if (plan.confirm) confirming = plan
      else await run(plan)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function run(plan: ActionPlan) {
    confirming = null
    busy = plan.name
    try {
      await backend.act(serverID, plan.name, plan.action)
      onchanged()
    } catch (e) {
      const ue = toUserError(e)
      if (ue.code === 'CMD_NEEDS_ROOT') sudo = plan.sudo
      else error = ue.message
    } finally {
      busy = null
    }
  }

  function menu(c: Container) {
    return [
      { label: tb.details, run: () => onopen(c.name) },
      { label: tb.logs, run: () => openLogs(serverID, { kind: 'docker', target: c.name }) },
      {
        label: tb.shell,
        disabled: c.state !== 'running',
        run: () => openTerminalAt(serverID, undefined, `docker exec -it ${c.name} sh`),
      },
      null,
      {
        label: `${d.actions.restart}…`,
        disabled: !canDo(c.state, 'restart'),
        run: () => void act(c, 'restart'),
      },
      {
        label: `${d.actions.rm}…`,
        danger: true,
        disabled: running(c),
        run: () => void act(c, 'rm'),
      },
    ]
  }

  async function compose(g: Group, action: ComposeAction) {
    const c = g.containers[0]
    try {
      const cmd = await backend.composeCommand(
        g.project ?? '',
        c?.projectDir ?? '',
        c?.projectFile ?? '',
        action,
      )
      openTerminalAt(serverID, c?.projectDir || undefined, cmd)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  // Compose commands are typed into a terminal: the user reads them and
  // presses Enter (docs/SECURITY.md).
  function projectMenu(g: Group) {
    const c = g.containers[0]
    const known = !!c?.projectDir
    return [
      { label: d.compose.up, disabled: !known, run: () => void compose(g, 'up') },
      { label: d.compose.pull, disabled: !known, run: () => void compose(g, 'pull') },
      { label: d.compose.restart, run: () => void compose(g, 'restart') },
      {
        label: d.compose.file,
        disabled: !c?.projectFile,
        run: () => c?.projectFile && openFileInEditor(serverID, c.projectFile),
      },
      null,
      { label: d.compose.down, danger: true, run: () => void compose(g, 'down') },
    ]
  }

  const kind = $derived(confirming?.action ?? 'stop')
</script>

{#snippet row(c: Container, child: boolean)}
  {@const s = stats[c.id]}
  {@const ports = parsePorts(c.ports)}
  <tr class:dk-child={child} onclick={() => onopen(c.name)}>
    <td class="dk-name">
      <span class="dk-dot {health(c.state)}" title={d.states[c.state] ?? c.state}></span>
      <span>{child && c.service ? c.service : c.name}</span>
      {#if child && c.service}<span class="dk-sub">{c.name}</span>{/if}
    </td>
    <td class="mono">{c.id.slice(0, 12)}</td>
    <td>
      <button
        class="dk-link"
        title={c.image}
        onclick={(e) => {
          e.stopPropagation()
          onimage(
            c.image.includes(':') || c.image.startsWith('sha256') ? c.image : `${c.image}:latest`,
          )
        }}>{c.image}</button
      >
    </td>
    <td>
      {#each ports as p (`${p.host}:${p.container}`)}
        {#if p.host}
          <button
            class="dk-chip"
            title={tb.openPort(p.host)}
            onclick={(e) => {
              e.stopPropagation()
              openInBrowser(serverID, p.host ?? 0)
            }}>{p.host}:{p.container}</button
          >
        {:else}
          <span class="dk-chip">{p.container}</span>
        {/if}
      {:else}
        <span class="muted">–</span>
      {/each}
    </td>
    <td class="num">{s ? formatPct(s.cpu) : '–'}</td>
    <td class="num">{s ? formatBytes(s.memUsed) : '–'}</td>
    <td class="muted" title={c.status}>
      {d.states[c.state] ?? c.state}{#if c.runningFor}<span class="dk-sub"
          >{` · ${ago(c.runningFor)}`}</span
        >{/if}
    </td>
    <td>
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="dk-actions" onclick={(e) => e.stopPropagation()}>
        {#if running(c)}
          <button
            class="icon-btn"
            aria-label={d.actions.stop}
            disabled={busy === c.name}
            onclick={() => void act(c, 'stop')}><Icon name="stop" size={14} /></button
          >
        {:else}
          <button
            class="icon-btn go"
            aria-label={d.actions.start}
            disabled={busy === c.name || !canDo(c.state, 'start')}
            onclick={() => void act(c, 'start')}><Icon name="play" size={15} /></button
          >
        {/if}
        <ActionMenu label={tb.more(c.name)} actions={menu(c)} />
        <button
          class="icon-btn danger"
          aria-label={d.actions.rm}
          disabled={running(c) || busy === c.name}
          title={running(c) ? tb.stopFirst : d.actions.rm}
          onclick={() => void act(c, 'rm')}><Icon name="trash" size={14} /></button
        >
      </div>
    </td>
  </tr>
{/snippet}

{#if error}
  <div class="dk-banner error">
    <Icon name="warning" size={14} /><span class="grow">{error}</span>
  </div>
{/if}
{#if sudo}
  <div class="dk-banner">
    <span class="grow">{t.common.needsRoot}</span>
    <button class="btn" onclick={() => (openTerminalAt(serverID, undefined, sudo), (sudo = ''))}
      >{t.common.runInTerminal}</button
    >
  </div>
{/if}

<div class="dk-scroll">
  {#if shown.length === 0}
    <p class="dk-empty">{d.noMatch}</p>
  {:else}
    <table class="dk-table">
      <thead>
        <tr>
          <th>{tb.name}</th>
          <th>{tb.id}</th>
          <th>{tb.image}</th>
          <th>{tb.ports}</th>
          <th class="num">CPU</th>
          <th class="num">RAM</th>
          <th>{tb.status}</th>
          <th class="num">{tb.actions}</th>
        </tr>
      </thead>
      <tbody>
        {#each shown as g (g.key)}
          {#if g.project}
            {@const open = unfolded.has(g.project)}
            {@const cpu = sum(g, (s) => s.cpu)}
            {@const mem = sum(g, (s) => s.memUsed)}
            {@const up = g.containers.filter((c) => c.state === 'running').length}
            <tr
              class="dk-group"
              onclick={() =>
                open ? unfolded.delete(g.project ?? '') : unfolded.add(g.project ?? '')}
            >
              <td class="dk-name">
                <button
                  class="dk-chevron"
                  class:open
                  aria-label={open ? tb.fold : tb.unfold}
                  aria-expanded={open}><Icon name="chevronRight" size={13} /></button
                >
                <span class="dk-dot {groupHealth(g)}"></span>
                <span>{g.project}</span>
                <span class="dk-sub">{tb.compose(up, g.containers.length)}</span>
              </td>
              <td></td>
              <td></td>
              <td></td>
              <td class="num">{cpu !== undefined ? formatPct(cpu) : ''}</td>
              <td class="num">{mem !== undefined ? formatBytes(mem) : ''}</td>
              <td></td>
              <td class="num" onclick={(e) => e.stopPropagation()}>
                <ActionMenu label={d.compose.more(g.project)} actions={projectMenu(g)} />
              </td>
            </tr>
            {#if open}
              {#each g.containers as c (c.id)}{@render row(c, true)}{/each}
            {/if}
          {:else}
            {@render row(g.containers[0] as Container, false)}
          {/if}
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<ConfirmCommand
  command={confirming?.command ?? null}
  title={confirming ? d.confirmTitle[kind === 'start' ? 'stop' : kind](confirming.name) : ''}
  body={d.confirmBody[kind === 'start' ? 'stop' : kind]}
  action={d.actions[kind]}
  onclose={() => (confirming = null)}
  onconfirm={() => confirming && void run(confirming)}
/>
