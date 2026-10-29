<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../api/errors'
  import { locale, t } from '../i18n'
  import { fadeOut } from '../motion'
  import { appLock } from '../stores/lock.svelte'
  import { notifications } from '../stores/notifications.svelte'
  import DavishMark from '../ui/DavishMark.svelte'
  import Icon from '../ui/Icon.svelte'

  let now = $state(new Date())
  let password = $state('')
  let error = $state('')
  let busy = $state(false)
  let shake = $state(false)
  let resetting = $state(false)
  let input = $state<HTMLInputElement>()
  let root = $state<HTMLElement>()
  // Notifications that came in while locked show only as a count.
  const seen = untrack(() => notifications.unread)

  const waitSec = $derived(
    appLock.state.retryAt > now.getTime()
      ? Math.ceil((appLock.state.retryAt - now.getTime()) / 1000)
      : 0,
  )

  $effect(() => {
    const timer = setInterval(() => (now = new Date()), 1000)
    return () => clearInterval(timer)
  })

  // Touch ID asks once when the lock shows; the button asks again. touchID()
  // reads busy before its first await, so it must stay untracked or a
  // cancel (busy -> false) re-runs the effect and reopens the sheet forever.
  $effect(() => {
    requestAnimationFrame(() => input?.focus())
    untrack(() => {
      if (appLock.state.biometric) void touchID()
    })
  })

  async function touchID() {
    if (busy) return
    busy = true
    error = ''
    try {
      await appLock.touchID(t.lock.touchIDReason)
    } catch (e) {
      // Cancelling the sheet is not worth a message; a real failure is.
      if (toUserError(e).code !== 'LOCK_BIOMETRIC_FAILED') error = toUserError(e).message
      input?.focus()
    } finally {
      busy = false
    }
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    if (!password || busy || waitSec > 0) return
    busy = true
    error = ''
    try {
      await appLock.unlock(password)
      password = ''
    } catch (err) {
      error = toUserError(err).message
      password = ''
      shake = true
      requestAnimationFrame(() => input?.focus())
    } finally {
      busy = false
    }
  }

  async function reset() {
    busy = true
    try {
      await appLock.reset()
    } catch (e) {
      error = toUserError(e).message
      resetting = false
    } finally {
      busy = false
    }
  }

  // Keys stay on the lock screen: one aimed elsewhere (a terminal that
  // still had focus) is dropped before it lands, and one typed here stops
  // at the lock instead of reaching the desktop's shortcuts.
  function trap(e: KeyboardEvent) {
    if (e.target instanceof Node && root?.contains(e.target)) return
    e.preventDefault()
    e.stopPropagation()
    input?.focus()
  }
</script>

<svelte:window onkeydowncapture={trap} />

<div
  class="lock"
  role="dialog"
  aria-modal="true"
  aria-label={t.lock.title}
  tabindex="-1"
  bind:this={root}
  onkeydown={(e) => e.stopPropagation()}
  out:fadeOut
>
  <div class="clock">
    {now.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
  </div>
  <div class="date">
    {now.toLocaleDateString(locale, { weekday: 'long', day: 'numeric', month: 'long' })}
  </div>

  <div class="card">
    <DavishMark size={56} />
    <h1>{t.lock.title}</h1>

    {#if resetting}
      <p class="body">{t.lock.resetBody}</p>
      <div class="actions">
        <button type="button" class="btn" disabled={busy} onclick={() => (resetting = false)}
          >{t.lock.cancel}</button
        >
        <button type="button" class="btn btn-danger" disabled={busy} onclick={reset}
          >{t.lock.resetConfirm}</button
        >
      </div>
    {:else}
      <form class="entry" class:shake onsubmit={submit} onanimationend={() => (shake = false)}>
        <input
          bind:this={input}
          bind:value={password}
          type="password"
          autocomplete="current-password"
          placeholder={t.lock.password}
          aria-label={t.lock.password}
          disabled={waitSec > 0}
        />
        <button
          type="submit"
          class="go"
          aria-label={t.lock.unlock}
          disabled={!password || busy || waitSec > 0}
        >
          <Icon name="arrowRight" size={14} strokeWidth={2.4} />
        </button>
      </form>
      <p class="error" role="alert">
        {waitSec > 0 ? t.lock.retryIn(waitSec) : error}
      </p>
      {#if appLock.state.biometric}
        <button type="button" class="touch" disabled={busy} onclick={touchID}>
          <Icon name="fingerprint" size={22} />
          <span>{t.lock.touchID}</span>
        </button>
      {/if}
      <button type="button" class="link" onclick={() => (resetting = true)}>{t.lock.forgot}</button>
    {/if}
  </div>

  {#if notifications.unread > seen}
    <div class="notice">
      <Icon name="bell" size={13} />
      {t.lock.newNotice}
    </div>
  {/if}
</div>

<style>
  .lock {
    position: fixed;
    inset: 0;
    z-index: 12000;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-top: 12vh;
    gap: 4px;
    background: var(--sleep-bg);
    color: var(--fg-on-dark);
    -webkit-backdrop-filter: var(--glass-dark-blur);
    backdrop-filter: var(--glass-dark-blur);
    --wails-draggable: no-drag;
  }

  .clock {
    font-size: 72px;
    font-weight: 300;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
  }

  .date {
    font-size: var(--text-lg);
    color: var(--fg-on-dark-muted);
  }

  .card {
    width: min(320px, calc(100vw - 40px));
    margin-top: 48px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
  }

  h1 {
    margin: 4px 0 6px;
    font-size: var(--text-lg);
    font-weight: 600;
  }

  .entry {
    width: 240px;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 4px 4px 14px;
    border-radius: 999px;
    background: var(--tab-hover-dark);
    box-shadow: inset 0 0 0 1px var(--divider-dark);
  }

  .entry input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--fg-on-dark);
    font-size: var(--text-md);
  }

  .entry input::placeholder {
    color: var(--fg-on-dark-muted);
  }

  .go {
    display: flex;
    padding: 6px;
    border: 0;
    border-radius: 50%;
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .go:disabled {
    opacity: 0.4;
  }

  /* Wrong password: a short side-to-side shake, like macOS. */
  .shake {
    animation: shake var(--dur-slow) var(--ease-out);
  }

  @keyframes shake {
    20%,
    60% {
      transform: translateX(-8px);
    }
    40%,
    80% {
      transform: translateX(8px);
    }
  }

  .error {
    min-height: 18px;
    margin: 0;
    font-size: var(--text-sm);
    color: var(--danger-on-dark);
    text-align: center;
  }

  .touch {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 8px 14px;
    border: 0;
    border-radius: var(--item-radius);
    background: none;
    color: var(--fg-on-dark);
    font-size: var(--text-sm);
  }

  .touch:hover:not(:disabled) {
    background: var(--tab-hover-dark);
  }

  .link {
    border: 0;
    background: none;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-sm);
  }

  .link:hover {
    color: var(--fg-on-dark);
    text-decoration: underline;
  }

  .body {
    margin: 0;
    font-size: var(--text-sm);
    line-height: 1.45;
    color: var(--fg-on-dark-muted);
    text-align: center;
  }

  .actions {
    display: flex;
    gap: 8px;
    margin-top: 6px;
  }

  .notice {
    position: absolute;
    bottom: 40px;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-sm);
    color: var(--fg-on-dark-muted);
  }

  @media (prefers-reduced-motion: reduce) {
    .shake {
      animation: none;
    }
  }
</style>
