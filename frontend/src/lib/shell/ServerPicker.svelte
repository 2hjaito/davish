<script lang="ts">
  import { t } from '../i18n'
  import { type ConnState, servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import { openServers } from '../wm/intents.svelte'
  import Icon from '../ui/Icon.svelte'
  import TagChip from '../ui/TagChip.svelte'
  import { shortcut } from './platform'
  import Popover from './Popover.svelte'
  import { activeFilter, matchesTag, tagGroups } from './tags'

  let open = $state(false)
  let dragging = $state<string | null>(null)

  // Filter by tag only makes sense once there is more than one to choose from.
  let tagFilter = $state('')
  const groups = $derived(tagGroups(servers.list))
  const filterable = $derived(groups.length > 1)
  const filter = $derived(activeFilter(groups, tagFilter))
  const shown = $derived(servers.list.filter((s) => matchesTag(s, filter)))

  const dotColor: Record<ConnState, string> = {
    connected: 'var(--ok)',
    connecting: 'var(--warn)',
    reconnecting: 'var(--warn)',
    disconnected: 'var(--offline)',
    failed: 'var(--danger)',
  }

  function choose(id: string) {
    servers.select(id)
    open = false
  }

  function edit(e: MouseEvent, id: string) {
    e.stopPropagation()
    open = false
    ui.editServer(id)
  }

  function retry(e: MouseEvent, id: string) {
    e.stopPropagation()
    void servers.reconnect(id)
  }

  // Native drag and drop reorders the list; the new order is saved on drop.
  function dragOver(e: DragEvent, overId: string) {
    if (!dragging || dragging === overId) return
    e.preventDefault()
    const ids = servers.list.map((s) => s.id)
    const from = ids.indexOf(dragging)
    const to = ids.indexOf(overId)
    ids.splice(to, 0, ...ids.splice(from, 1))
    servers.list = ids.flatMap((id) => servers.list.find((s) => s.id === id) ?? [])
  }

  function drop() {
    if (!dragging) return
    dragging = null
    void servers.reorder(servers.list.map((s) => s.id))
  }
</script>

<div class="anchor">
  <button
    class="mbtn picker"
    aria-haspopup="menu"
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    <span class="dot" style:background={dotColor[servers.active?.state ?? 'disconnected']}></span>
    <span class="name">{servers.active?.name ?? t.servers.none}</span>
    {#if servers.active?.tag}
      <TagChip tag={servers.active.tag} color={servers.active.color} solid />
    {/if}
    <Icon name="chevronDown" size={10} strokeWidth={3} />
  </button>

  <Popover {open} onclose={() => (open = false)} label={t.servers.picker} width={300}>
    <div class="header">{t.servers.header}</div>
    {#if filterable}
      <div class="filters" role="group" aria-label={t.servers.tagFilter}>
        <button
          type="button"
          class="fchip"
          aria-pressed={filter === ''}
          onclick={() => (tagFilter = '')}>{t.servers.tagAll}</button
        >
        {#each groups as g (g.tag)}
          <button
            type="button"
            class="fchip"
            aria-pressed={filter.toLowerCase() === g.tag.toLowerCase()}
            onclick={() => (tagFilter = g.tag)}
          >
            {#if g.color}<span class="swatch" style:background={g.color}></span>{/if}{g.tag}
          </button>
        {/each}
      </div>
    {/if}
    <!-- Many servers scroll here, so the actions below stay in reach (#178). -->
    <div class="servers">
      {#each shown as s (s.id)}
        {@const i = servers.list.indexOf(s)}
        {@const active = s.id === servers.activeId}
        <div
          class="row"
          class:dragging={dragging === s.id}
          draggable="true"
          title={t.servers.dragToReorder}
          role="presentation"
          ondragstart={(e) => {
            dragging = s.id
            e.dataTransfer?.setData('text/plain', s.id)
          }}
          ondragover={(e) => dragOver(e, s.id)}
          ondrop={drop}
          ondragend={drop}
        >
          <button role="menuitem" class="item" class:active onclick={() => choose(s.id)}>
            <span
              class="dot"
              style:background={active && s.state === 'connected'
                ? 'var(--ok-on-accent)'
                : (s.color ?? dotColor[s.state])}
              class:ring={!!s.color}
              style:--ring={dotColor[s.state]}
            ></span>
            <span class="text">
              <span class="name">{s.name}</span>
              <span class="sub">{s.user}@{s.host} · {t.servers.status[s.state]}</span>
            </span>
            {#if s.tag}<TagChip tag={s.tag} color={s.color} />{/if}
            {#if i < 9}<span class="sub key">{shortcut(String(i + 1))}</span>{/if}
          </button>
          {#if s.state === 'failed'}
            <button class="tool" aria-label={t.servers.reconnect} onclick={(e) => retry(e, s.id)}>
              <Icon name="refresh" size={13} />
            </button>
          {/if}
          {#if s.state === 'connected' || s.state === 'connecting' || s.state === 'reconnecting'}
            <button
              class="tool"
              aria-label={t.servers.overview.disconnect}
              title={t.servers.overview.disconnect}
              onclick={(e) => {
                e.stopPropagation()
                void servers.disconnect(s.id)
              }}
            >
              <Icon name="power" size={13} />
            </button>
          {/if}
          <button class="tool" aria-label={t.servers.edit} onclick={(e) => edit(e, s.id)}>
            <Icon name="editor" size={13} />
          </button>
        </div>
      {:else}
        <p class="empty">{t.servers.none}</p>
      {/each}
    </div>
    <div class="divider"></div>
    {#if servers.activeId}
      <button
        role="menuitem"
        class="item"
        onclick={() => {
          open = false
          openServers(servers.activeId)
        }}
      >
        <Icon name="grid" size={14} />
        {t.servers.overview.open}
      </button>
    {/if}
    <button
      role="menuitem"
      class="item"
      onclick={() => {
        open = false
        ui.addServer()
      }}
    >
      <Icon name="plus" size={14} strokeWidth={2.2} />
      {t.servers.add}
    </button>
    <button
      role="menuitem"
      class="item"
      onclick={() => {
        open = false
        ui.importDialog = true
      }}
    >
      <Icon name="download" size={14} />
      {t.servers.import}
    </button>
  </Popover>
</div>

<style>
  .servers {
    /* The menu bar above, the actions below and some air. */
    max-height: calc(100vh - 200px);
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .anchor {
    position: relative;
  }

  .picker {
    font-weight: 700;
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--glass-highlight);
  }

  /* A long name must not push the menus off the bar. */
  .picker .name {
    max-width: 160px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  /* A custom server color keeps the state visible as a ring. */
  .dot.ring {
    box-shadow: 0 0 0 2px var(--ring);
  }

  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
    padding: 2px 10px 6px;
  }

  .fchip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 2px 9px;
    border: 0;
    border-radius: 999px;
    background: var(--fill-soft);
    color: var(--fg);
    font-size: var(--text-xs);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .fchip:hover {
    background: var(--row-hover);
  }

  .fchip[aria-pressed='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .swatch {
    width: 8px;
    height: 8px;
    border-radius: 2px;
  }

  /* On the accent row a tinted chip would vanish; give it a light ground. */
  .item.active :global(.tag) {
    background: rgba(255, 255, 255, 0.92);
    color: color-mix(in srgb, var(--c, #555) 85%, #000);
  }

  .header {
    padding: 6px 10px 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    letter-spacing: 0.04em;
  }

  .row {
    position: relative;
    display: flex;
    align-items: center;
    border-radius: var(--item-radius);
  }

  .row.dragging {
    opacity: 0.5;
  }

  .item {
    flex: 1;
    min-width: 0;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border: 0;
    border-radius: var(--item-radius);
    background: transparent;
    text-align: left;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .item:hover:not(.active) {
    background: var(--row-hover);
  }

  .item.active {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .text {
    flex: 1;
    min-width: 0;
  }

  .name {
    display: block;
    font-weight: 600;
  }

  .sub {
    display: block;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .key {
    flex-shrink: 0;
  }

  .active .sub {
    color: inherit;
    opacity: 0.85;
  }

  /* Row tools appear on hover, over the shortcut hint. */
  .tool {
    display: flex;
    padding: 4px;
    margin-left: 2px;
    border: 0;
    border-radius: 6px;
    background: var(--content-bg);
    color: var(--fg-muted);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .row:hover .tool,
  .tool:focus-visible {
    opacity: 1;
  }

  .tool:hover {
    color: var(--accent);
  }

  .empty {
    margin: 0;
    padding: 8px 10px;
    color: var(--fg-subtle);
    font-size: var(--text-sm);
  }

  .divider {
    height: 1px;
    background: var(--divider);
    margin: 6px 8px;
  }
</style>
