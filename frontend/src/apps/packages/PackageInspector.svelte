<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type {
    NpmGlobal,
    Package,
    PackageOp,
    PackageUpdate,
    ServerApp,
  } from '../../lib/api/packages'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openFileInEditor, openSelected, SERVICES_SIZE } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  /** What is selected: a program, or a package with its update if any. */
  export type Selected =
    | { kind: 'app'; app: ServerApp }
    | { kind: 'package'; name: string; pkg?: Package; update?: PackageUpdate }

  interface Props {
    serverID: string
    sel: Selected
    /** Types the sudo command for op into a terminal. */
    onterminal: (op: PackageOp, names: string[]) => void
  }

  let { serverID, sel, onterminal }: Props = $props()

  let npm = $state.raw<NpmGlobal[] | null>(null)
  let npmError = $state('')
  const key = $derived(sel.kind === 'app' ? sel.app.id : sel.name)
  $effect(() => {
    void key
    untrack(() => {
      npm = null
      npmError = ''
    })
  })

  async function loadNpm() {
    try {
      npm = await backend.npmGlobals(serverID)
    } catch (e) {
      npmError = toUserError(e).message
    }
  }

  function openUnit(unit: string) {
    openSelected('services', serverID, unit, { title: t.apps.services, size: SERVICES_SIZE })
  }
</script>

<section class="inspector" aria-label={sel.kind === 'app' ? sel.app.label : sel.name}>
  <header>
    <span class="icon"><Icon name="package" size={18} /></span>
    <div class="heading">
      <h3>{sel.kind === 'app' ? sel.app.label : sel.name}</h3>
      <p class="sub">
        {#if sel.kind === 'app'}
          {t.packages.kinds[sel.app.kind]} · {sel.app.version || t.packages.noVersion}
        {:else}
          {sel.pkg?.summary ?? ''}
        {/if}
      </p>
    </div>
  </header>

  {#if sel.kind === 'app'}
    {@const a = sel.app}
    <dl class="facts">
      <dt>{t.packages.version}</dt>
      <dd>{a.version || t.packages.noVersion}</dd>
      <dt>{t.packages.path}</dt>
      <dd><code>{a.path}</code></dd>
      {#if a.versions.length > 0}
        <dt>{t.packages.versions[a.id] ?? t.packages.version}</dt>
        <dd>{a.versions.join(', ')}</dd>
      {/if}
    </dl>

    {#if a.configs.length > 0}
      <h4>{t.packages.configs}</h4>
      <div class="links">
        {#each a.configs as c (c)}
          <button type="button" class="link" onclick={() => openFileInEditor(serverID, c)}
            >{c}</button
          >
        {/each}
      </div>
    {/if}

    {#if a.units.length > 0}
      <h4>{t.packages.units}</h4>
      <div class="actions">
        {#each a.units as u (u)}
          <button type="button" class="btn" onclick={() => openUnit(u)}>
            <Icon name="services" size={13} />
            {u}
          </button>
        {/each}
      </div>
    {/if}

    {#if a.id === 'node' || a.id === 'npm'}
      <h4>{t.packages.npmGlobals}</h4>
      {#if npm}
        {#if npm.length === 0}
          <p class="muted">{t.packages.npmEmpty}</p>
        {:else}
          <ul class="npm">
            {#each npm as p (p.name)}<li><span>{p.name}</span><code>{p.version}</code></li>{/each}
          </ul>
        {/if}
      {:else}
        <div class="actions">
          <button type="button" class="btn" onclick={() => void loadNpm()}
            >{t.packages.loadNpm}</button
          >
        </div>
        {#if npmError}<p class="bad">{npmError}</p>{/if}
      {/if}
    {/if}
  {:else}
    <dl class="facts">
      {#if sel.update}
        <dt>{t.packages.current}</dt>
        <dd><code>{sel.update.current || sel.pkg?.version}</code></dd>
        <dt>{t.packages.available}</dt>
        <dd>
          <code>{sel.update.available}</code>
          {#if sel.update.security}<span class="badge">{t.packages.security}</span>{/if}
        </dd>
      {:else if sel.pkg}
        <dt>{t.packages.version}</dt>
        <dd><code>{sel.pkg.version}</code></dd>
      {/if}
      {#if sel.pkg && sel.pkg.size > 0}
        <dt>{t.packages.size}</dt>
        <dd>{formatBytes(sel.pkg.size)}</dd>
      {/if}
    </dl>
    <div class="actions">
      {#if sel.update}
        <button
          type="button"
          class="btn btn-primary"
          onclick={() => sel.kind === 'package' && onterminal('upgrade', [sel.name])}
          >{t.packages.upgrade}</button
        >
      {/if}
      {#if sel.pkg}
        <button
          type="button"
          class="btn remove"
          onclick={() => sel.kind === 'package' && onterminal('remove', [sel.name])}
          >{t.packages.remove}</button
        >
      {/if}
    </div>
    <p class="muted">{t.packages.terminalHint}</p>
  {/if}
</section>

<style>
  .inspector {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px 16px;
  }

  header {
    display: flex;
    gap: 10px;
    align-items: flex-start;
  }

  .icon {
    display: flex;
    padding: 6px;
    border-radius: var(--item-radius);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .heading {
    min-width: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    overflow-wrap: anywhere;
  }

  .sub {
    margin: 2px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .badge {
    margin-left: 6px;
    padding: 1px 7px;
    border-radius: 999px;
    background: var(--danger);
    color: var(--fg-on-accent);
    font-size: var(--text-xs);
    font-weight: 600;
  }

  h4 {
    margin: 6px 0 0;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .links {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .link {
    padding: 0;
    border: 0;
    background: none;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .link:hover {
    text-decoration: underline;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .remove {
    color: var(--danger);
  }

  .npm {
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: var(--text-sm);
  }

  .npm li {
    display: flex;
    justify-content: space-between;
    padding: 3px 0;
    border-bottom: 1px solid var(--divider-soft);
  }

  .muted {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .bad {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }
</style>
