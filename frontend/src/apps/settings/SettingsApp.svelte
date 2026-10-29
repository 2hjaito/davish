<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n/vi'
  import { appLock, LOCK_AFTER, MIN_LOCK_PASSWORD } from '../../lib/stores/lock.svelte'
  import {
    ALERTS,
    type BackupMode,
    MONO_FAMILIES,
    type MonoFont,
    settings,
    TERMINAL_FONT,
    WALLPAPER_ACCENTS,
  } from '../../lib/stores/settings.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { hasRuntime } from '../../lib/api/runtime'
  import { updates } from '../../lib/stores/updates.svelte'
  import type { AppProps } from '../types'

  let { win }: AppProps = $props()

  function stepFont(dir: 1 | -1) {
    settings.set('terminalFontSize', settings.terminalFontSize + dir * TERMINAL_FONT.step)
  }

  const fonts = Object.keys(MONO_FAMILIES) as MonoFont[]
  const backups: BackupMode[] = ['etc', 'always', 'never']
  // Lock password forms: one at a time (set, change or turn off).
  let lockForm = $state<'change' | 'disable' | null>(null)
  let current = $state('')
  let next = $state('')
  let again = $state('')
  let lockMsg = $state('')
  let lockBusy = $state(false)

  const lockProblem = $derived(
    lockForm === 'disable'
      ? null
      : next.length < MIN_LOCK_PASSWORD
        ? t.settings.lockShort(MIN_LOCK_PASSWORD)
        : next !== again
          ? t.settings.lockMismatch
          : null,
  )

  function closeLockForm() {
    lockForm = null
    current = next = again = ''
  }

  async function saveLock(e: SubmitEvent) {
    e.preventDefault()
    if (lockBusy || (lockForm !== 'disable' && lockProblem)) return
    lockBusy = true
    lockMsg = ''
    try {
      if (lockForm === 'disable') await appLock.disable(current)
      else await appLock.setPassword(current, next)
      closeLockForm()
      lockMsg = t.settings.lockSaved
    } catch (err) {
      lockMsg = toUserError(err).message
    } finally {
      lockBusy = false
    }
  }

  const pctRows = [
    { key: 'alertMem', label: t.settings.alertMem, hint: t.settings.alertMemHint },
    { key: 'alertDisk', label: t.settings.alertDisk, hint: t.settings.alertDiskHint },
  ] as const
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="spacer"></span>
</TitleBar>

<div class="body">
  <h2>{t.settings.appearance}</h2>

  <label class="row">
    <span>
      <strong>{t.settings.reduceGlass}</strong>
      <small>{t.settings.reduceGlassHint}</small>
    </span>
    <input
      type="checkbox"
      class="switch"
      role="switch"
      checked={settings.reduceGlass}
      onchange={(e) => settings.set('reduceGlass', e.currentTarget.checked)}
    />
  </label>

  <label class="row">
    <span>
      <strong>{t.settings.reduceMotion}</strong>
      <small>{t.settings.reduceMotionHint}</small>
    </span>
    <input
      type="checkbox"
      class="switch"
      role="switch"
      checked={settings.reduceMotion}
      onchange={(e) => settings.set('reduceMotion', e.currentTarget.checked)}
    />
  </label>

  <div class="row">
    <strong>{t.settings.wallpaper}</strong>
    <div class="swatches" role="radiogroup" aria-label={t.settings.wallpaper}>
      {#each WALLPAPER_ACCENTS as c (c)}
        <button
          class="swatch"
          role="radio"
          aria-checked={settings.wallpaperAccent === c}
          aria-label={t.settings.wallpaperColor(c)}
          style:background={c}
          onclick={() => settings.set('wallpaperAccent', c)}
        ></button>
      {/each}
    </div>
  </div>

  <div class="row">
    <span>
      <strong>{t.settings.monoFont}</strong>
      <small>{t.settings.monoFontHint}</small>
    </span>
    <div class="segmented" role="radiogroup" aria-label={t.settings.monoFont}>
      {#each fonts as f (f)}
        <button
          role="radio"
          aria-checked={settings.monoFont === f}
          style:font-family={MONO_FAMILIES[f]}
          onclick={() => settings.set('monoFont', f)}>{t.settings.monoFonts[f]}</button
        >
      {/each}
    </div>
  </div>

  <h2 class="section">{t.settings.terminal}</h2>

  <div class="row">
    <span>
      <strong>{t.settings.fontSize}</strong>
      <small>{t.settings.fontSizeHint}</small>
    </span>
    <div class="stepper">
      <button
        aria-label={t.settings.fontSmaller}
        disabled={settings.terminalFontSize <= TERMINAL_FONT.min}
        onclick={() => stepFont(-1)}
      >
        <Icon name="minus" size={12} strokeWidth={2.2} />
      </button>
      <output aria-live="polite">{settings.terminalFontSize.toLocaleString('vi-VN')}</output>
      <button
        aria-label={t.settings.fontLarger}
        disabled={settings.terminalFontSize >= TERMINAL_FONT.max}
        onclick={() => stepFont(1)}
      >
        <Icon name="plus" size={12} strokeWidth={2.2} />
      </button>
    </div>
  </div>

  <label class="row">
    <span>
      <strong>{t.settings.prompt}</strong>
      <small>{t.settings.promptHint}</small>
    </span>
    <input
      type="checkbox"
      class="switch"
      role="switch"
      checked={settings.terminalPrompt}
      onchange={(e) => settings.set('terminalPrompt', e.currentTarget.checked)}
    />
  </label>

  <label class="row">
    <span>
      <strong>{t.settings.gpu}</strong>
      <small>{t.settings.gpuHint}</small>
    </span>
    <input
      type="checkbox"
      class="switch"
      role="switch"
      checked={settings.terminalGpu}
      onchange={(e) => settings.set('terminalGpu', e.currentTarget.checked)}
    />
  </label>

  <h2 class="section">{t.settings.alerts}</h2>

  {#each pctRows as r (r.key)}
    <div class="row">
      <span>
        <strong>{r.label}</strong>
        <small>{r.hint}</small>
      </span>
      <div class="stepper">
        <button
          aria-label="{t.settings.lower} {r.label}"
          disabled={settings[r.key] <= ALERTS.pct.min}
          onclick={() =>
            settings.set(r.key, Math.max(ALERTS.pct.min, settings[r.key] - ALERTS.pct.step))}
        >
          <Icon name="minus" size={12} strokeWidth={2.2} />
        </button>
        <output aria-live="polite">{settings[r.key]}%</output>
        <button
          aria-label="{t.settings.higher} {r.label}"
          disabled={settings[r.key] >= ALERTS.pct.max}
          onclick={() =>
            settings.set(r.key, Math.min(ALERTS.pct.max, settings[r.key] + ALERTS.pct.step))}
        >
          <Icon name="plus" size={12} strokeWidth={2.2} />
        </button>
      </div>
    </div>
  {/each}

  <div class="row">
    <span>
      <strong>{t.settings.alertHold}</strong>
      <small>{t.settings.alertHoldHint}</small>
    </span>
    <div class="segmented" role="radiogroup" aria-label={t.settings.alertHold}>
      {#each ALERTS.holds as h (h)}
        <button
          role="radio"
          aria-checked={settings.alertHold === h}
          onclick={() => settings.set('alertHold', h)}>{t.settings.holds[h]}</button
        >
      {/each}
    </div>
  </div>

  <h2 class="section">{t.settings.lock}</h2>

  {#if !appLock.state.enabled}
    <form class="row lockform" onsubmit={saveLock}>
      <small>{t.settings.lockIntro}</small>
      <label class="field">
        <span>{t.settings.lockNew}</span>
        <input class="input" type="password" autocomplete="new-password" bind:value={next} />
      </label>
      <label class="field">
        <span>{t.settings.lockConfirm}</span>
        <input class="input" type="password" autocomplete="new-password" bind:value={again} />
      </label>
      <div class="formfoot">
        <small class="msg">{next ? (lockProblem ?? '') : lockMsg}</small>
        <button class="btn btn-primary" disabled={lockBusy || lockProblem !== null}
          >{t.settings.lockEnable}</button
        >
      </div>
    </form>
  {:else}
    <div class="row">
      <span>
        <strong>{t.settings.lockOn}</strong>
        <small
          >{appLock.state.biometric ? t.settings.lockOnTouchID : t.settings.lockOnPassword}</small
        >
      </span>
      <button class="btn" onclick={() => appLock.lock()}>{t.settings.lockNow}</button>
    </div>
    <div class="row">
      <span>
        <strong>{t.settings.lockAfter}</strong>
        <small>{t.settings.lockAfterHint}</small>
      </span>
      <div class="segmented" role="radiogroup" aria-label={t.settings.lockAfter}>
        {#each LOCK_AFTER as m (m)}
          <button
            role="radio"
            aria-checked={settings.lockAfter === m}
            onclick={() => settings.set('lockAfter', m)}>{t.settings.lockAfters[m]}</button
          >
        {/each}
      </div>
    </div>
    {#if lockForm}
      <form class="row lockform" onsubmit={saveLock}>
        <label class="field">
          <span>{t.settings.lockCurrent}</span>
          <input
            class="input"
            type="password"
            autocomplete="current-password"
            bind:value={current}
          />
          {#if lockForm === 'disable'}<small>{t.settings.lockDisableHint}</small>{/if}
        </label>
        {#if lockForm === 'change'}
          <label class="field">
            <span>{t.settings.lockNew}</span>
            <input class="input" type="password" autocomplete="new-password" bind:value={next} />
          </label>
          <label class="field">
            <span>{t.settings.lockConfirm}</span>
            <input class="input" type="password" autocomplete="new-password" bind:value={again} />
          </label>
        {/if}
        <div class="formfoot">
          <small class="msg">{lockForm === 'change' && next ? (lockProblem ?? '') : lockMsg}</small>
          <button type="button" class="btn" onclick={closeLockForm}>{t.lock.cancel}</button>
          <button
            class="btn {lockForm === 'disable' ? 'btn-danger' : 'btn-primary'}"
            disabled={lockBusy || !current || (lockForm === 'change' && lockProblem !== null)}
            >{lockForm === 'disable' ? t.settings.lockDisable : t.settings.lockSave}</button
          >
        </div>
      </form>
    {:else}
      <div class="lockactions">
        {#if lockMsg}<small class="msg">{lockMsg}</small>{/if}
        <button class="btn" onclick={() => ((lockForm = 'change'), (lockMsg = ''))}
          >{t.settings.lockChange}</button
        >
        <button class="btn" onclick={() => ((lockForm = 'disable'), (lockMsg = ''))}
          >{t.settings.lockDisable}</button
        >
      </div>
    {/if}
  {/if}

  <h2 class="section">{t.settings.editor}</h2>

  <div class="row">
    <span>
      <strong>{t.settings.backup}</strong>
      <small>{t.settings.backupHint}</small>
    </span>
    <div class="segmented" role="radiogroup" aria-label={t.settings.backup}>
      {#each backups as b (b)}
        <button
          role="radio"
          aria-checked={settings.editorBackup === b}
          onclick={() => settings.set('editorBackup', b)}>{t.settings.backups[b]}</button
        >
      {/each}
    </div>
  </div>

  <h2 class="section">{t.settings.updates}</h2>

  <label class="row">
    <span>
      <strong>{t.settings.autoUpdate}</strong>
      <small>{t.settings.autoUpdateHint}</small>
    </span>
    <input
      type="checkbox"
      class="switch"
      role="switch"
      checked={settings.autoUpdate}
      onchange={(e) => settings.set('autoUpdate', e.currentTarget.checked)}
    />
  </label>
  <div class="row">
    <span>
      <strong>{t.settings.version(__APP_VERSION__)}</strong>
    </span>
    <button class="btn" disabled={!hasRuntime} onclick={() => void updates.check(true)}
      >{t.settings.checkNow}</button
    >
  </div>
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .spacer {
    width: 52px;
  }

  .body {
    flex: 1;
    overflow-y: auto;
    padding: 14px 20px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  h2 {
    margin: 0 0 6px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .section {
    margin-top: 14px;
  }

  .lockform {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .formfoot,
  .lockactions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  .msg {
    flex: 1;
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 10px 12px;
    border-radius: var(--item-radius);
    background: var(--glass-highlight);
  }

  .row span {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  /* A choice of a few, like macOS segmented controls. */
  .segmented {
    flex-shrink: 0;
    display: flex;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .segmented button {
    padding: 3px 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .segmented button[aria-checked='true'] {
    background: var(--chart-surface);
    color: var(--fg);
    font-weight: 600;
    box-shadow: var(--switch-knob-shadow);
  }

  /* Segmented − value + control, like the macOS font size stepper. */
  .stepper {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .stepper button {
    display: flex;
    border: 0;
    padding: 6px 8px;
    border-radius: var(--control-radius);
    background: none;
    color: inherit;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .stepper button:hover:not(:disabled) {
    background: var(--fill-track);
  }

  .stepper button:disabled {
    opacity: 0.35;
  }

  .stepper output {
    min-width: 36px;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .swatches {
    display: flex;
    gap: 8px;
  }

  .swatch {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 2px solid var(--glass-border);
    padding: 0;
    transition:
      transform var(--dur-fast) var(--ease-spring),
      border-color var(--dur-fast) var(--ease-out);
  }

  .swatch:hover {
    transform: scale(1.12);
  }

  .swatch[aria-checked='true'] {
    border-color: var(--accent);
  }
</style>
