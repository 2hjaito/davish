<script lang="ts">
  import type { NginxSite } from '../../lib/api/nginx'
  import { locale, t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openFileInEditor } from '../../lib/wm/intents.svelte'
  import { certState, daysLeft } from './sites'

  interface Props {
    serverID: string
    site: NginxSite
    busy: boolean
    /** A note after a toggle, with a way to reload. */
    flash: string
    ontoggle: (site: NginxSite) => void
    onremove: (site: NginxSite) => void
    onreload: () => void
  }

  let { serverID, site, busy, flash, ontoggle, onremove, onreload }: Props = $props()

  const fmtDate = new Intl.DateTimeFormat(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
</script>

<section class="inspector" aria-label={site.name}>
  <header>
    <span class="dot" class:on={site.enabled} aria-hidden="true"></span>
    <div class="heading">
      <h3>{site.name}</h3>
      <p class="path"><code>{site.path}</code></p>
    </div>
  </header>

  <p class="state" class:on={site.enabled}>
    {site.kind === 'conf' ? t.nginx.alwaysOn : site.enabled ? t.nginx.on : t.nginx.off}
  </p>

  <div class="actions">
    {#if site.toggleable}
      <button type="button" class="btn" disabled={busy} onclick={() => ontoggle(site)}>
        {site.enabled ? t.nginx.disable : t.nginx.enable}
      </button>
    {/if}
    <button type="button" class="btn" onclick={() => openFileInEditor(serverID, site.path)}>
      {t.nginx.openFile}
    </button>
    <button type="button" class="btn remove" disabled={busy} onclick={() => onremove(site)}>
      {t.nginx.remove}
    </button>
  </div>
  {#if !site.toggleable && site.kind === 'site'}<p class="hint">{t.nginx.fileInEnabled}</p>{/if}

  {#if flash}
    <div class="flash" role="status">
      <span>{flash}</span>
      <button type="button" class="btn btn-primary" disabled={busy} onclick={onreload}>
        {t.nginx.reloadNow}
      </button>
    </div>
  {/if}

  {#if site.error}<p class="bad">{t.nginx.unreadable}</p>{/if}

  {#each site.servers as srv, i (i)}
    <div class="block">
      <div class="block-head">
        <h4>{t.nginx.servers}</h4>
        <button
          type="button"
          class="link"
          onclick={() => openFileInEditor(serverID, site.path, srv.line)}
        >
          {t.nginx.serverAt(srv.line)}
        </button>
      </div>
      <dl class="facts">
        <dt>{t.nginx.names}</dt>
        <dd>{srv.names.length > 0 ? srv.names.join(', ') : t.nginx.none}</dd>
        <dt>{t.nginx.listen}</dt>
        <dd class="chips">
          {#each srv.listen as l (l)}<code class="chip" class:tls={/\bssl\b/.test(l)}>{l}</code
            >{/each}
        </dd>
        {#if srv.root}
          <dt>{t.nginx.root}</dt>
          <dd><code>{srv.root}</code></dd>
        {/if}
        {#if srv.proxyPass.length > 0}
          <dt>{t.nginx.proxy}</dt>
          <dd><code>{srv.proxyPass.join('\n')}</code></dd>
        {/if}
      </dl>
    </div>
  {/each}

  {#if site.upstreams.length > 0}
    <dl class="facts">
      <dt>{t.nginx.upstreams}</dt>
      <dd><code>{site.upstreams.join(', ')}</code></dd>
    </dl>
  {/if}

  {#if site.certs.length > 0}
    <h4>{t.nginx.certs}</h4>
    <ul class="certs">
      {#each site.certs as c (c.path)}
        {@const state = certState(c)}
        <li>
          <Icon name="shield" size={14} />
          <div>
            <code>{c.path}</code>
            {#if c.names.length > 0}<span class="muted">{c.names.join(', ')}</span>{/if}
            <span class="when {state}">
              {#if c.error === 'unreadable'}{t.nginx.certUnreadable}
              {:else if c.error}{t.nginx.certInvalid}
              {:else if state === 'expired'}{t.nginx.certExpired(-daysLeft(c))}
              {:else}{t.nginx.certLeft(daysLeft(c))} · {t.nginx.expires(
                  fmtDate.format(c.notAfter * 1000),
                )}{/if}
            </span>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .remove {
    margin-left: auto;
    color: var(--danger);
  }

  .inspector {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px 16px;
    overflow-y: auto;
  }

  header {
    display: flex;
    gap: 10px;
    align-items: flex-start;
  }

  .heading {
    min-width: 0;
  }

  .dot {
    width: 10px;
    height: 10px;
    margin-top: 6px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--offline);
  }

  .dot.on {
    background: var(--ok);
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    overflow-wrap: anywhere;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }

  .path {
    margin: 2px 0 0;
    color: var(--fg-muted);
  }

  .state {
    margin: 0;
    font-weight: 600;
    color: var(--fg-muted);
  }

  .state.on {
    color: var(--ok);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .hint,
  .muted {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .flash {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--accent-soft);
    font-size: var(--text-sm);
  }

  .flash span {
    flex: 1;
  }

  .bad {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .block {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 10px 12px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
  }

  .block-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  h4 {
    margin: 4px 0 0;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
  }

  .block h4 {
    margin: 0;
  }

  .link {
    padding: 0;
    border: 0;
    background: none;
    font-size: var(--text-xs);
    color: var(--accent);
  }

  .link:hover {
    text-decoration: underline;
  }

  .facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 5px 14px;
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

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .chip {
    padding: 0 6px;
    border-radius: 5px;
    background: var(--fill-soft);
    font-size: var(--text-xs);
  }

  .chip.tls {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .certs {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .certs li {
    display: flex;
    gap: 8px;
  }

  .certs li > :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--accent);
  }

  .certs li div {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }

  .when {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .when.soon,
  .when.unknown {
    color: var(--warn);
  }

  .when.expired {
    color: var(--danger);
    font-weight: 600;
  }
</style>
