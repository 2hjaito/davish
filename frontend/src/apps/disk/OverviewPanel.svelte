<script lang="ts">
  import type { Overview } from '../../lib/api/disk'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { sudoPrompt } from '../../lib/stores/sudoPrompt.svelte'
  import { nameOf } from './tree'

  interface Props {
    serverID: string
    overview: Overview
    /** What the cleanups would free together. */
    cleanable: number
    onclean: () => void
    onfile: (path: string) => void
    /** Measures again, after the sudo password was given. */
    onrescan: () => void
  }

  let { serverID, overview: o, cleanable, onclean, onfile, onrescan }: Props = $props()

  async function unlock() {
    if (o.sudoScan && (await sudoPrompt.ask(serverID, o.sudoScan))) onrescan()
  }

  const d = t.disk.ov
  const SHOWN = 6

  const held = $derived(o.deleted.reduce((s, x) => s + x.size, 0))
  const total = $derived(Math.max(1, o.mount.total))
  const free = $derived(Math.max(0, o.mount.total - o.mount.used))
  const pct = (n: number) => `${(n / total) * 100}%`
  // Worth explaining from a gigabyte, or a twentieth of what is used.
  const notable = (n: number) => n >= 1 << 30 || n >= o.mount.used / 20
</script>

<div class="ov">
  <h3 class="mono">{o.mount.path}</h3>
  <p class="muted">
    {t.disk.used(
      formatBytes(o.mount.used),
      formatBytes(o.mount.total),
      Math.round((o.mount.used / total) * 100),
    )}
    {#if o.root}· {d.asRoot}{/if}
  </p>

  <div class="stack" role="img" aria-label={d.barLabel}>
    <span class="seg measured" style:width={pct(Math.min(o.measured, o.mount.used))}></span>
    <span class="seg held" style:width={pct(held)}></span>
    <span class="seg unmeasured" style:width={pct(o.unmeasured)}></span>
  </div>
  <ul class="legend">
    <li><span class="dot measured"></span>{d.measured}<b>{formatBytes(o.measured)}</b></li>
    {#if held > 0}<li><span class="dot held"></span>{d.held}<b>{formatBytes(held)}</b></li>{/if}
    {#if o.unmeasured > 0}
      <li><span class="dot unmeasured"></span>{d.unmeasured}<b>{formatBytes(o.unmeasured)}</b></li>
    {/if}
    <li><span class="dot free"></span>{d.free}<b>{formatBytes(free)}</b></li>
  </ul>
  {#if o.partial}<p class="warn">{d.partial}</p>{/if}

  {#if notable(o.unmeasured)}
    <section>
      <h4><Icon name="lock" size={13} />{d.unmeasuredTitle(formatBytes(o.unmeasured))}</h4>
      {#if !o.root && o.unreadable.length > 0}
        <p>{d.unreadableBody}</p>
        <ul class="paths mono">
          {#each o.unreadable.slice(0, SHOWN) as p (p)}<li>{p}</li>{/each}
        </ul>
        {#if o.unreadable.length > SHOWN}
          <p class="muted">{d.more(o.unreadable.length - SHOWN)}</p>
        {/if}
        <div class="actions">
          {#if o.askSudo && o.sudoScan}
            <button class="btn btn-primary" onclick={() => void unlock()}>
              <Icon name="key" size={13} />
              {d.measureAsRoot}
            </button>
          {/if}
          <button class="btn" onclick={() => openTerminalAt(serverID, undefined, o.sudoMeasure)}>
            <Icon name="terminal" size={13} />
            {d.measureSudo}
          </button>
        </div>
      {:else}
        <p>{d.unmeasuredOther}</p>
      {/if}
    </section>
  {/if}

  {#if held > 0}
    <section>
      <h4><Icon name="warning" size={13} />{d.heldTitle(formatBytes(held))}</h4>
      <p>{d.heldBody}</p>
      <ul class="files">
        {#each o.deleted.slice(0, SHOWN) as f (`${f.pid}${f.path}`)}
          <li>
            <span class="mono name" title={f.path}>{f.path}</span>
            <span class="muted">{d.heldBy(f.command, f.pid)}</span>
            <b>{formatBytes(f.size)}</b>
          </li>
        {/each}
      </ul>
    </section>
  {/if}

  {#if cleanable > 0}
    <section>
      <h4><Icon name="trash" size={13} />{d.cleanTitle(formatBytes(cleanable))}</h4>
      <button class="btn btn-primary" onclick={onclean}>{d.cleanOpen}</button>
    </section>
  {/if}

  {#if o.files.length > 0}
    <section>
      <h4><Icon name="file" size={13} />{d.largest}</h4>
      <ul class="files">
        {#each o.files.slice(0, 5) as f (f.path)}
          <li>
            <button class="link name" title={f.path} onclick={() => onfile(f.path)}>
              {nameOf(f.path)}
            </button>
            <b>{formatBytes(f.size)}</b>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
</div>

<style>
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .ov {
    align-self: stretch;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
  }

  h4 {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0 0 6px;
    font-size: var(--text-sm);
  }

  p {
    margin: 0 0 8px;
    font-size: var(--text-sm);
    line-height: 1.45;
  }

  .muted {
    color: var(--fg-muted);
  }

  .warn {
    color: var(--warn);
    font-size: var(--text-xs);
  }

  .stack {
    display: flex;
    height: 12px;
    border-radius: 6px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .seg {
    height: 100%;
  }

  .measured {
    background: var(--accent);
  }

  .held {
    background: var(--warn);
  }

  .unmeasured {
    background: repeating-linear-gradient(-45deg, var(--fg-subtle) 0 3px, transparent 3px 6px);
  }

  .free {
    background: var(--fill-track);
  }

  .legend {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px 16px;
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .legend li {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .legend b {
    margin-left: auto;
    color: var(--fg);
    font-weight: 500;
    font-variant-numeric: tabular-nums;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  section {
    margin-top: 8px;
    padding-top: 12px;
    border-top: 1px solid var(--divider-soft);
  }

  .paths {
    margin: 0 0 8px;
    padding: 0;
    list-style: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .files {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: var(--text-sm);
  }

  .files li {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .files b {
    font-weight: 500;
    font-variant-numeric: tabular-nums;
  }

  .link {
    padding: 0;
    border: 0;
    background: none;
    font: inherit;
    text-align: left;
    color: var(--accent);
    cursor: pointer;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
</style>
