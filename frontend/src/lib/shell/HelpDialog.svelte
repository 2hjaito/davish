<script lang="ts">
  import { getApp } from '../../apps/registry'
  import { t } from '../i18n/vi'
  import { ui } from '../stores/ui.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import { buildMenus } from './menuModel.svelte'
  import { comboLabel } from './menus.svelte'
  import { isMac } from './platform'

  const topic = $derived(ui.help)
  const app = $derived(topic?.kind === 'app' ? getApp(topic.appId) : undefined)
  const help = $derived(topic?.kind === 'app' ? t.help.apps[topic.appId] : undefined)

  const title = $derived(
    topic?.kind === 'app'
      ? t.menu.appHelp(app?.label ?? '')
      : topic?.kind === 'shortcuts'
        ? t.help.shortcutsTitle
        : t.help.aboutTitle,
  )

  // Every shortcut is a menu item's, grouped by menu, plus a few that
  // live outside the menus.
  const groups = $derived.by(() => {
    if (topic?.kind !== 'shortcuts') return []
    const out = buildMenus()
      .filter((m) => m.id !== 'app')
      .map((m) => ({
        label: m.label,
        rows: m.items.flatMap((it) =>
          it?.combo ? [{ key: comboLabel(it.combo), label: it.label }] : [],
        ),
      }))
      .filter((g) => g.rows.length > 0)
    const mod = isMac ? '⌘' : 'Ctrl+'
    out.push({
      label: t.help.scopes.other,
      rows: [
        { key: `${mod}1…${mod}9`, label: t.help.workspace },
        { key: 'Esc', label: t.help.escape },
        { key: '↑ ↓ ← → Enter', label: t.help.menuNav },
      ],
    })
    return out
  })

  const close = () => (ui.help = null)
</script>

<Dialog
  open={topic !== null}
  {title}
  onclose={close}
  width={topic?.kind === 'shortcuts' ? 520 : 440}
>
  {#if topic?.kind === 'app' && help}
    <section>
      <h3>{t.help.does}</h3>
      <p>{help.does}</p>
    </section>
    {#if help.actions.length}
      <section>
        <h3>{t.help.actions}</h3>
        <ul>
          {#each help.actions as a (a)}<li>{a}</li>{/each}
        </ul>
      </section>
    {/if}
    {#if help.keys.length}
      <section>
        <h3>{t.help.keys}</h3>
        <ul>
          {#each help.keys as k (k)}<li>{k}</li>{/each}
        </ul>
      </section>
    {/if}
    <section>
      <h3>{t.help.runs}</h3>
      {#if help.runs.length}
        <ul class="mono">
          {#each help.runs as r (r)}<li>{r}</li>{/each}
        </ul>
      {:else}
        <p>{t.help.runsNone}</p>
      {/if}
    </section>
  {:else if topic?.kind === 'shortcuts'}
    <div class="groups">
      {#each groups as g (g.label)}
        <section>
          <h3>{g.label}</h3>
          <dl>
            {#each g.rows as r (r.key + r.label)}
              <dt>{r.label}</dt>
              <dd><kbd>{r.key}</kbd></dd>
            {/each}
          </dl>
        </section>
      {/each}
    </div>
  {:else if topic?.kind === 'about'}
    <p class="version">{t.help.version(__APP_VERSION__)}</p>
    <p>{t.help.aboutBody}</p>
    <p class="muted">{t.help.changelogHint}</p>
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn btn-primary" onclick={close}>{t.help.close}</button>
  {/snippet}
</Dialog>

<style>
  section + section {
    margin-top: 12px;
  }

  h3 {
    margin: 0 0 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  p,
  ul {
    margin: 0;
    font-size: var(--text-sm);
  }

  ul {
    padding-left: 18px;
  }

  .groups {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px 20px;
  }

  .groups section + section {
    margin-top: 0;
  }

  dl {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 3px 10px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dd {
    margin: 0;
  }

  kbd {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    padding: 1px 5px;
    border-radius: 4px;
    background: var(--fill-soft);
  }

  .version {
    font-weight: 600;
    margin-bottom: 6px;
  }

  .muted {
    margin-top: 8px;
    color: var(--fg-muted);
  }

  .spacer {
    flex: 1;
  }
</style>
