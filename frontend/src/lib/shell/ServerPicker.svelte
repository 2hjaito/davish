<script lang="ts">
  import { t } from '../i18n/vi'
  import { type ConnState, servers } from '../stores/servers.svelte'
  import Icon from '../ui/Icon.svelte'
  import { shortcut } from './platform'
  import Popover from './Popover.svelte'

  let open = $state(false)

  const dotColor: Record<ConnState, string> = {
    connected: 'var(--ok)',
    connecting: 'var(--warn)',
    disconnected: 'var(--offline)',
  }

  function choose(id: string) {
    servers.select(id)
    open = false
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
    {servers.active?.name}
    <Icon name="chevronDown" size={10} strokeWidth={3} />
  </button>

  <Popover {open} onclose={() => (open = false)} label={t.servers.picker}>
    <div class="header">{t.servers.header}</div>
    {#each servers.list as s, i (s.id)}
      {@const active = s.id === servers.activeId}
      <button role="menuitem" class="item" class:active onclick={() => choose(s.id)}>
        <span
          class="dot"
          style:background={active && s.state === 'connected'
            ? 'var(--ok-on-accent)'
            : dotColor[s.state]}
        ></span>
        <span class="text">
          <span class="name">{s.name}</span>
          <span class="sub">
            {[`${s.user}@`, s.os, t.servers.status[s.state]].filter(Boolean).join(' · ')}
          </span>
        </span>
        {#if i < 9}<span class="sub">{shortcut(String(i + 1))}</span>{/if}
      </button>
    {/each}
    <div class="divider"></div>
    <button role="menuitem" class="item" onclick={() => (open = false)}>
      <Icon name="plus" size={14} strokeWidth={2.2} />
      {t.servers.add}
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

  .header {
    padding: 6px 10px 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    letter-spacing: 0.04em;
  }

  .item {
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
  }

  .active .sub {
    color: inherit;
    opacity: 0.85;
  }

  .divider {
    height: 1px;
    background: var(--divider);
    margin: 6px 8px;
  }
</style>
