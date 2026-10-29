<script lang="ts">
  import type { Entry } from '../../lib/api/files'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import type { DropTarget } from './remotes.svelte'

  interface Props {
    /** What to copy; closed while null. */
    entry: Entry | null
    /** The other servers and their folders, as in the sidebar. */
    others: {
      server: { id: string; name: string }
      online: boolean
      places: { label: string; path: string; icon: IconName; title: string }[]
    }[]
    onclose: () => void
    oncopy: (entry: Entry, target: DropTarget) => void
  }

  let { entry, others, onclose, oncopy }: Props = $props()
</script>

<!-- The keyboard way to what dragging onto another server's folder does. -->
<Dialog
  open={entry !== null}
  title={entry ? t.files.copyToTitle(entry.name) : ''}
  width={420}
  {onclose}
>
  {#if others.length === 0}
    <p class="muted">{t.files.copyToNone}</p>
  {:else}
    <p class="muted">{t.files.copyToHint}</p>
    {#each others as o (o.server.id)}
      <span class="group">{o.server.name}{o.online ? '' : ` · ${t.files.otherOffline}`}</span>
      {#each o.places as p (p.path)}
        <button
          type="button"
          class="target"
          disabled={!o.online}
          title={p.title}
          onclick={() => entry && oncopy(entry, { serverID: o.server.id, path: p.path })}
        >
          <Icon name={p.icon} size={15} />
          <span>{p.label}</span>
          <code>{p.title}</code>
        </button>
      {/each}
    {/each}
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
  {/snippet}
</Dialog>

<style>
  .muted {
    margin: 0 0 8px;
    color: var(--fg-muted);
    font-size: var(--text-sm);
    line-height: 1.4;
  }

  .group {
    display: block;
    margin: 10px 0 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-subtle);
  }

  .target {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 8px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    text-align: left;
  }

  .target:hover:not(:disabled),
  .target:focus-visible {
    background: var(--row-hover);
  }

  .target :global(svg) {
    color: var(--accent);
    flex-shrink: 0;
  }

  .target code {
    margin-left: auto;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .target:disabled {
    color: var(--fg-subtle);
  }

  .spacer {
    flex: 1;
  }
</style>
