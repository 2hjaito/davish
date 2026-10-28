<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import { browserOpen, browserSites, type Site } from '../../lib/api/ports'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n/vi'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { selectRequests } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { displayAddress, frameURL, parseAddress } from './address'

  let { win }: AppProps = $props()

  const b = t.browser

  let address = $state('')
  let src = $state('')
  let title = $state('')
  let error = $state('')
  let frame = $state<HTMLIFrameElement>()
  // Local proxy port → site, to show addresses as the server sees them.
  let sites = $state.raw<Site[]>([])

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  async function go(text: string) {
    error = ''
    const a = parseAddress(text)
    if (!a) {
      error = b.badAddress
      return
    }
    try {
      const site = await browserOpen(win.serverID, a.port)
      sites = [...sites.filter((s) => s.localPort !== site.localPort), site]
      address = displayAddress(frameURL(site.localPort, a.path, site.token), a.port)
      src = frameURL(site.localPort, a.path, site.token)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  // Only nav.js in the frame reports where the page went (ADR 0008).
  async function onMessage(e: MessageEvent) {
    if (!frame || e.source !== frame.contentWindow) return
    const d = e.data as { davish?: string; href?: string; title?: string } | null
    if (d?.davish !== 'nav' || typeof d.href !== 'string') return
    let local: number
    try {
      local = Number(new URL(d.href).port)
    } catch {
      return
    }
    let site = sites.find((s) => s.localPort === local)
    if (!site) {
      // A redirect to another port of the server opened another proxy.
      sites = await browserSites()
      site = sites.find((s) => s.localPort === local)
    }
    if (site?.serverID !== win.serverID) return
    address = displayAddress(d.href, site.remotePort)
    title = typeof d.title === 'string' ? d.title : ''
  }

  const send = (cmd: 'back' | 'forward' | 'reload') =>
    frame?.contentWindow?.postMessage({ davish: cmd }, '*')

  // Ports asked for a port of this server.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'browser' && r.serverID === win.serverID))
      return
    untrack(() => {
      const req = selectRequests.take('browser', win.serverID)
      if (req) void go(req.key)
    })
  })

  useAppMenu(
    () => [
      { label: b.back, disabled: !src, run: () => send('back') },
      { label: b.forward, disabled: !src, run: () => send('forward') },
    ],
    () => send('reload'),
  )
</script>

<svelte:window onmessage={(e) => void onMessage(e)} />

<TitleBar>
  <div class="bar">
    <button class="tool" aria-label={b.back} disabled={!src} onclick={() => send('back')}>
      <Icon name="arrowRight" size={14} strokeWidth={2.2} />
    </button>
    <button class="tool fwd" aria-label={b.forward} disabled={!src} onclick={() => send('forward')}>
      <Icon name="arrowRight" size={14} strokeWidth={2.2} />
    </button>
    <button class="tool" aria-label={b.reload} disabled={!src} onclick={() => send('reload')}>
      <Icon name="refresh" size={13} strokeWidth={2.2} />
    </button>
    <form
      class="address"
      onsubmit={(e) => {
        e.preventDefault()
        void go(address)
      }}
    >
      <Icon name="server" size={12} />
      <input
        bind:value={address}
        aria-label={b.address}
        placeholder={b.placeholder}
        spellcheck="false"
        autocapitalize="off"
        disabled={!hasRuntime || !connected}
      />
    </form>
  </div>
</TitleBar>

<div class="body">
  {#if !hasRuntime}
    <Empty icon="globe2" title={b.appOnly} />
  {:else if !connected}
    <Empty icon="server" title={b.offline} hint={b.offlineHint} />
  {:else if !src}
    <Empty icon="globe2" title={error || b.start} hint={b.hint} />
  {:else}
    {#if error}<p class="error">{error}</p>{/if}
    <!-- Its own origin (127.0.0.1:<port>), so the page cannot reach
         Davish; no allow-top-navigation, so it cannot replace it. -->
    <iframe
      bind:this={frame}
      {src}
      title={title || address}
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-modals allow-downloads"
    ></iframe>
  {/if}
</div>

<style>
  .bar {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
  }

  .tool {
    border: 0;
    background: none;
    padding: 4px;
    display: flex;
    border-radius: 6px;
    color: var(--fg-muted);
  }

  .tool:first-child {
    rotate: 180deg;
  }

  .tool:hover:not(:disabled) {
    background: var(--glass-hover);
  }

  .address {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: 6px;
    padding: 3px 10px;
    border-radius: 7px;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .address input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: none;
    font-size: var(--text-sm);
    color: var(--fg);
    outline: none;
  }

  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
  }

  iframe {
    flex: 1;
    width: 100%;
    border: 0;
    background: #fff;
  }

  .error {
    margin: 0;
    padding: 6px 12px;
    color: var(--danger);
    font-size: var(--text-sm);
  }
</style>
