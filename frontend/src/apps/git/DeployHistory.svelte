<script lang="ts">
  import { formatElapsed } from '../../lib/format'
  import { locale, t } from '../../lib/i18n'
  import { type DeployRecord, deploys } from '../../lib/stores/deploys.svelte'
  import Icon from '../../lib/ui/Icon.svelte'

  interface Props {
    serverID: string
    dir: string
  }

  let { serverID, dir }: Props = $props()
  const d = t.deploys
  const SHOWN = 5

  const records = $derived(deploys.historyOf(serverID, dir).slice(0, SHOWN))

  const icon = (r: DeployRecord) =>
    r.state === 'ok' ? 'check' : r.state === 'failed' ? 'warning' : 'stop'
  const total = (r: DeployRecord) => r.durations.reduce((a, b) => a + b, 0)
  const when = (at: number) =>
    new Date(at).toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' })
</script>

<section class="history" aria-label={d.history}>
  <h4>{d.history}</h4>
  {#if records.length === 0}
    <p class="none">{d.never}</p>
  {:else}
    <ul>
      {#each records as r, i (i)}
        <li class={r.state}>
          <span class="icon"><Icon name={icon(r)} size={12} /></span>
          <span class="state">
            {d.state[r.state]}{#if r.step && r.state !== 'ok'}&nbsp;{d.atStep(r.step)}{/if}
          </span>
          {#if r.commit}<code>{r.commit}</code>{/if}
          <span class="meta">{when(r.at)} · {formatElapsed(total(r))}</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .history {
    font-size: var(--text-sm);
  }

  h4 {
    margin: 0 0 6px;
    color: var(--fg-muted);
    font-size: var(--text-xs);
    font-weight: 600;
  }

  .none {
    margin: 0;
    color: var(--fg-muted);
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .icon {
    display: flex;
    color: var(--warn);
  }

  .ok .icon {
    color: var(--ok);
  }

  .failed .icon {
    color: var(--danger);
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .meta {
    margin-left: auto;
    color: var(--fg-muted);
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
</style>
