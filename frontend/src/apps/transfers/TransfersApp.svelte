<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { transferCancel } from '../../lib/api/transfer'
  import { t } from '../../lib/i18n'
  import { notifications } from '../../lib/stores/notifications.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import { type Transfer, transfers } from '../../lib/stores/transfers.svelte'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { percent, statusText } from './status'

  let { win }: AppProps = $props()

  /** Newest first, every server: transfers outlive the window that started them. */
  const list = $derived([...transfers.list].reverse())
  const anyFinished = $derived(transfers.list.some((j) => j.state !== 'running'))

  const kindIcon = { upload: 'arrowUp', download: 'arrowDown', copy: 'copy' } as const
  const kindLabel = {
    upload: t.transfers.upload,
    download: t.transfers.download,
    copy: t.transfers.copy,
  }

  function serverName(id: string) {
    return servers.list.find((s) => s.id === id)?.name ?? id
  }

  async function cancel(j: Transfer) {
    try {
      await transferCancel(j.id)
    } catch (e) {
      // Usually it just finished; anything else is worth saying.
      notifications.add({
        id: `transfer-cancel-${j.id}`,
        level: 'error',
        title: t.transfers.cancel(j.name),
        body: toUserError(e).message,
        serverID: j.serverID,
        at: Date.now(),
      })
    }
  }
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="spacer"></span>
</TitleBar>

<div class="body">
  {#if list.length === 0}
    <Empty icon="download" title={t.transfers.empty} hint={t.transfers.emptyHint} />
  {:else}
    <ul class="jobs">
      {#each list as j (j.id)}
        <li class="job" class:failed={j.state === 'failed'}>
          <span class="kind" title={kindLabel[j.kind]}>
            <Icon name={kindIcon[j.kind]} size={14} strokeWidth={2.4} />
          </span>
          <div class="info">
            <div class="line">
              <span class="name" title={j.kind === 'download' ? j.src : j.dst}>{j.name}</span>
              <span class="server">
                {j.kind === 'copy' && j.fromServerID
                  ? t.transfers.route(serverName(j.fromServerID), serverName(j.serverID))
                  : serverName(j.serverID)}
              </span>
            </div>
            {#if j.state === 'running'}
              <Meter pct={percent(j)} label={j.name} warn={101} danger={101} />
            {/if}
            <span class="status">{statusText(j)}</span>
          </div>
          {#if j.state === 'running'}
            <button
              class="cancel"
              aria-label={t.transfers.cancel(j.name)}
              onclick={() => cancel(j)}
            >
              <Icon name="close" size={13} strokeWidth={2.4} />
            </button>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}

  <div class="footer">
    <button class="btn" disabled={!anyFinished} onclick={() => transfers.clearFinished()}>
      {t.transfers.clearFinished}
    </button>
  </div>
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
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

  .jobs {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 6px 8px;
    list-style: none;
    overflow-y: auto;
  }

  .job {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
  }

  .job:nth-child(even) {
    background: var(--row-stripe);
  }

  .kind {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    flex-shrink: 0;
    border-radius: 50%;
    background: var(--accent-soft);
    color: var(--accent);
  }

  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .line {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-width: 0;
  }

  .name {
    min-width: 0;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .server {
    flex-shrink: 0;
    margin-left: auto;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .status {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .failed .status {
    color: var(--danger);
  }

  .cancel {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    flex-shrink: 0;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: var(--fill-soft);
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .cancel:hover {
    background: var(--fill-track);
    color: var(--fg);
  }

  .footer {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    padding: 8px 12px;
    border-top: 1px solid var(--divider-soft);
  }
</style>
