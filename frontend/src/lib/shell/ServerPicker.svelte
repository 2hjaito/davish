<script lang="ts">
  import { t } from '../i18n'
  import { type ConnState, servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import Icon from '../ui/Icon.svelte'
  import { shortcut } from './platform'
  import Popover from './Popover.svelte'

  let open = $state(false)
  let dragging = $state<string | null>(null)

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
    {servers.active?.name ?? t.servers.none}
    <Icon name="chevronDown" size={10} strokeWidth={3} />
  </button>

  <Popover {open} onclose={() => (open = false)} label={t.servers.picker} width={300}>
    <div class="header">{t.servers.header}</div>
    {#each servers.list as s, i (s.id)}
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
          {#if i < 9}<span class="sub key">{shortcut(String(i + 1))}</span>{/if}
        </button>
        {#if s.state === 'failed'}
          <button class="tool" aria-label={t.servers.reconnect} onclick={(e) => retry(e, s.id)}>
            <Icon name="refresh" size={13} />
          </button>
        {/if}
        <button class="tool" aria-label={t.servers.edit} onclick={(e) => edit(e, s.id)}>
          <Icon name="editor" size={13} />
        </button>
      </div>
    {:else}
      <p class="empty">{t.servers.none}</p>
    {/each}
    <div class="divider"></div>
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
