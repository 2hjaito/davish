<script lang="ts">
  import { t } from '../../lib/i18n'
  import { openApp } from '../../lib/shell/menuModel.svelte'
  import { settings } from '../../lib/stores/settings.svelte'
  import { ui } from '../../lib/stores/ui.svelte'
  import { widgets } from '../../lib/stores/widgets.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import { addWidget } from '../../lib/widgets/actions'
  import { WIDGET_KINDS } from '../../lib/widgets/kinds'
  import { countKinds, type WidgetSize } from '../../lib/widgets/layout'
  import { servers } from '../../lib/stores/servers.svelte'
  import { widgetAvailable, widgetDefs } from '../../lib/widgets/registry'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { wm } from '../../lib/wm/windowManager.svelte'
  import type { AppProps } from '../types'

  let { win }: AppProps = $props()

  const w = t.widgets

  const on = $derived(settings.knobs.widgets)
  const counts = $derived(countKinds(widgets.of(win.serverID)))
  const total = $derived(widgets.of(win.serverID).length)

  // The widgets are on the desktop behind the windows: to move them, this
  // window steps aside.
  function editOnDesktop() {
    ui.widgetEdit = true
    wm.minimize(win.id)
  }

  const add = (kind: (typeof WIDGET_KINDS)[number], size: WidgetSize) =>
    addWidget(win.serverID, kind, size, false)
</script>

<TitleBar>
  <span class="title">{win.title}</span>
</TitleBar>

<div class="body">
  {#if !on}
    <Empty icon="widgets" title={w.offTitle} hint={w.offHint} />
    <div class="center">
      <button class="btn" onclick={() => openApp('settings')}>{w.openSettings}</button>
    </div>
  {:else}
    <div class="head">
      <p>{w.intro}</p>
      <button class="btn" disabled={total === 0} onclick={editOnDesktop}>{w.edit}</button>
    </div>
    <div class="gallery">
      {#each WIDGET_KINDS.filter( (k) => widgetAvailable(k, servers.platformOf(win.serverID)) ) as kind (kind)}
        {@const def = widgetDefs[kind]}
        <section class="card" aria-label={def.label()}>
          <div class="preview">
            {#await def.load() then W}<W.default serverID={win.serverID} size="S" />{/await}
          </div>
          <h3>{def.label()}</h3>
          <p class="about">{w.about[kind]}</p>
          <p class="count">{w.onDesktop(counts[kind])}</p>
          <div class="add">
            {#if def.sizes.includes('S')}
              <button class="btn" onclick={() => add(kind, 'S')}>{w.addSmall}</button>
            {/if}
            {#if def.sizes.includes('M')}
              <button class="btn" onclick={() => add(kind, 'M')}>{w.addMedium}</button>
            {/if}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: auto;
    background: var(--content-bg);
  }

  .head {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px 6px;
  }

  .head p {
    flex: 1;
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .gallery {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 14px;
    padding: 12px 16px 16px;
  }

  .card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px;
    border: 1px solid var(--divider);
    border-radius: var(--card-radius);
    background: var(--card-bg);
  }

  /* The size and look of a widget on the desktop, so what is added is what shows. */
  .preview {
    box-sizing: border-box;
    height: 92px;
    margin-bottom: 8px;
    padding: 10px 12px;
    border: 1px solid var(--divider);
    border-radius: var(--card-radius);
    background: var(--content-bg);
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
  }

  .about {
    flex: 1;
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .count {
    margin: 4px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .add {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }

  .center {
    display: flex;
    justify-content: center;
  }
</style>
