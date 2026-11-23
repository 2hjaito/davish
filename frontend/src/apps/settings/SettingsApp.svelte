<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { lang, type Lang, LANGS, locale, switchLang, t } from '../../lib/i18n'
  import { appLock, LOCK_AFTER, MIN_LOCK_PASSWORD } from '../../lib/stores/lock.svelte'
  import { load, save } from '../../lib/stores/persist'
  import {
    ALERTS,
    type BackupMode,
    cleanFontName,
    DOCK_POSITIONS,
    DOCK_SIZES,
    type DockSize,
    EDITOR_FONT,
    EDITOR_INDENTS,
    MONO_FAMILIES,
    type MonoFont,
    settings,
    SUDO_FORGET_AFTER,
    TERMINAL_FONT,
    TERMINAL_CURSORS,
    TERMINAL_SCHEMES,
    TERMINAL_SCROLLBACK,
    THEMES,
    WALLPAPER_ACCENTS,
    WALLPAPER_DIM,
  } from '../../lib/stores/settings.svelte'
  import DavishMark from '../../lib/ui/DavishMark.svelte'
  import ProfileAvatar from '../../lib/ui/ProfileAvatar.svelte'
  import { MAX_NAME, profile } from '../../lib/stores/profile.svelte'
  import { avatarImage, IMAGE_ACCEPT } from '../../lib/shell/shortcutLook'
  import { wallpaperImage } from '../../lib/stores/wallpaperImage.svelte'
  import { WALLPAPER_ACCEPT, wallpaperFromFile } from '../../lib/wallpaperFile'
  import Icon from '../../lib/ui/Icon.svelte'
  import PopupMenu from '../../lib/ui/PopupMenu.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { STAT_IDS, type StatId } from '../../lib/shell/statIds'
  import { relaunch } from '../../lib/api/update'
  import { hasRuntime } from '../../lib/api/runtime'
  import { NAMED_PRESETS, type NamedPreset } from '../../lib/perf'
  import { isLinux } from '../../lib/shell/platform'
  import { perfCheck } from '../../lib/shell/perfCheck.svelte'
  import { updates } from '../../lib/stores/updates.svelte'
  import { swatch } from '../terminal/schemes'
  import PaneIcon from './PaneIcon.svelte'
  import { type Pane, PANES, paneLabel, searchPanes } from './panes'
  import ThemePreview from './ThemePreview.svelte'
  import type { AppProps } from '../types'

  // The pane's name stands in for the window title, so win is not read.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  let { win }: AppProps = $props()
  const ctx = getWindowContext()

  // A color picked outside the swatches. Dragging in the picker fires input
  // on every move; only the color it rests on is applied and saved.
  const customAccent = $derived(
    !(WALLPAPER_ACCENTS as readonly string[]).includes(settings.wallpaperAccent),
  )
  let accentTimer: ReturnType<typeof setTimeout> | undefined
  function pickAccent(color: string) {
    clearTimeout(accentTimer)
    accentTimer = setTimeout(() => settings.set('wallpaperAccent', color.toLowerCase()), 120)
  }
  $effect(() => () => clearTimeout(accentTimer))
  let colorInput = $state<HTMLInputElement>()
  function openPicker() {
    const input = colorInput
    if (!input) return
    // click, not showPicker: WebKit's showPicker loses the anchor and opens
    // the panel in a screen corner; a click opens it under the wheel.
    input.click()
  }

  // Like System Settings: a sidebar of panes, back and forward between the
  // ones visited, and the last one shown again next time.
  const saved = load<Pane>('settingsPane', 'general')
  let history = $state<Pane[]>([PANES.includes(saved) ? saved : 'general'])
  let at = $state(0)
  const pane = $derived(history[at] ?? 'general')

  function go(p: Pane) {
    if (p === pane) return
    history = [...history.slice(0, at + 1), p]
    at = history.length - 1
    save('settingsPane', p)
  }

  function step(dir: 1 | -1) {
    const next = at + dir
    if (next < 0 || next >= history.length) return
    at = next
    save('settingsPane', history[at])
  }

  let query = $state('')
  const found = $derived(searchPanes(query))

  // Typing shows the first pane that matches unless the current one does.
  function onSearch() {
    const hits = searchPanes(query)
    const first = hits[0]
    if (first && !hits.includes(pane)) go(first)
  }

  function stepEditorFont(dir: 1 | -1) {
    settings.set('editorFontSize', settings.editorFontSize + dir * EDITOR_FONT.step)
  }
  function stepFont(dir: 1 | -1) {
    settings.set('terminalFontSize', settings.terminalFontSize + dir * TERMINAL_FONT.step)
  }

  // A language change needs a restart, so it waits for a confirming click.
  let pickedLang = $state<Lang>(lang)

  const fonts = Object.keys(MONO_FAMILIES) as MonoFont[]
  const dockSizes = Object.keys(DOCK_SIZES) as DockSize[]

  function toggleStat(id: StatId, on: boolean) {
    const now = settings.menubarStats
    settings.set(
      'menubarStats',
      on ? [...now.filter((s) => s !== id), id] : now.filter((s) => s !== id),
    )
  }
  const backups: BackupMode[] = ['etc', 'always', 'never']

  // The account picture: a file from this computer, cropped square and
  // shrunk (as a shortcut icon), kept only in settings.json.
  let avatarMsg = $state('')
  let editingName = $state(false)
  // The picture's menu (Choose…, Remove), under the picture like macOS.
  let avatarMenu = $state<{ x: number; y: number } | null>(null)
  let avatarButton = $state<HTMLButtonElement>()
  let avatarInput = $state<HTMLInputElement>()
  function openAvatarMenu() {
    if (!profile.avatar) {
      avatarInput?.click()
      return
    }
    const r = avatarButton?.getBoundingClientRect()
    if (r) avatarMenu = { x: r.left + r.width / 2 - 100, y: r.bottom + 6 }
  }
  async function pickAvatar(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    input.value = '' // so the same file can be picked again
    if (!file) return
    try {
      profile.set({ avatar: await avatarImage(file) })
      avatarMsg = ''
    } catch {
      avatarMsg = t.settings.avatarBad
    }
  }
  // The wallpaper picture: scaled in the browser, kept by Go in a file of its own.
  let pictureMsg = $state('')
  let pictureBusy = $state(false)
  let pictureInput = $state<HTMLInputElement>()
  $effect(() => {
    void wallpaperImage.ensure()
  })
  async function pickPicture(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return
    pictureBusy = true
    try {
      await wallpaperImage.set(await wallpaperFromFile(file))
      pictureMsg = ''
    } catch {
      pictureMsg = t.settings.wallpaperBad
    } finally {
      pictureBusy = false
    }
  }
  async function removePicture() {
    pictureBusy = true
    try {
      await wallpaperImage.clear()
      pictureMsg = ''
    } catch {
      pictureMsg = t.settings.wallpaperBad
    } finally {
      pictureBusy = false
    }
  }
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

  type BoolKnob =
    | 'blur'
    | 'opaque'
    | 'wallpaper'
    | 'wallpaperImage'
    | 'motion'
    | 'widgets'
    | 'noDmabuf'
    | 'noCompositing'
  const perfKnobs: { key: BoolKnob; label: string; hint: string }[] = [
    { key: 'blur', label: t.settings.perfBlur, hint: t.settings.perfBlurHint },
    { key: 'opaque', label: t.settings.perfOpaque, hint: t.settings.perfOpaqueHint },
    { key: 'wallpaper', label: t.settings.perfWallpaper, hint: t.settings.perfWallpaperHint },
    {
      key: 'wallpaperImage',
      label: t.settings.perfWallpaperImage,
      hint: t.settings.perfWallpaperImageHint,
    },
    { key: 'motion', label: t.settings.perfMotion, hint: t.settings.perfMotionHint },
    { key: 'widgets', label: t.settings.perfWidgets, hint: t.settings.perfWidgetsHint },
  ]
  // WebKitGTK workarounds, only on Linux.
  if (isLinux)
    perfKnobs.push(
      { key: 'noDmabuf', label: t.settings.perfNoDmabuf, hint: t.settings.perfNoDmabufHint },
      {
        key: 'noCompositing',
        label: t.settings.perfNoCompositing,
        hint: t.settings.perfNoCompositingHint,
      },
    )
</script>

{#snippet head(label: string, hint?: string)}
  <span class="head">
    <span class="label">{label}</span>
    {#if hint}<small>{hint}</small>{/if}
  </span>
{/snippet}

{#snippet toggle(label: string, hint: string, checked: boolean, set: (on: boolean) => void)}
  <label class="row">
    {@render head(label, hint)}
    <input
      type="checkbox"
      class="switch"
      role="switch"
      {checked}
      onchange={(e) => set(e.currentTarget.checked)}
    />
  </label>
{/snippet}

{#snippet stepper(
  value: number,
  atMin: boolean,
  atMax: boolean,
  smaller: string,
  larger: string,
  step: (dir: 1 | -1) => void,
)}
  <div class="stepper">
    <button aria-label={smaller} disabled={atMin} onclick={() => step(-1)}>
      <Icon name="minus" size={12} strokeWidth={2.2} />
    </button>
    <output aria-live="polite">{value.toLocaleString(locale)}</output>
    <button aria-label={larger} disabled={atMax} onclick={() => step(1)}>
      <Icon name="plus" size={12} strokeWidth={2.2} />
    </button>
  </div>
{/snippet}

{#snippet segmented<T extends string | number>(
  label: string,
  options: readonly T[],
  value: T,
  name: (v: T) => string,
  pick: (v: T) => void,
)}
  <div class="segmented" role="radiogroup" aria-label={label}>
    {#each options as o (o)}
      <button role="radio" aria-checked={value === o} onclick={() => pick(o)}>{name(o)}</button>
    {/each}
  </div>
{/snippet}

<div class="settings">
  <nav class="sidebar" aria-label={t.settings.panes}>
    <TitleBar bare />
    <div class="search">
      <Icon name="search" size={13} />
      <input
        type="search"
        placeholder={t.settings.search}
        aria-label={t.settings.search}
        spellcheck="false"
        bind:value={query}
        oninput={onSearch}
        onkeydown={(e) => {
          if (e.key === 'Escape' && query) {
            e.stopPropagation()
            query = ''
          }
        }}
      />
    </div>
    <div class="items">
      {#each found as p (p)}
        {#if p === 'account' && !query}
          <button
            class="about"
            class:active={pane === 'account'}
            aria-current={pane === 'account' ? 'page' : undefined}
            title={paneLabel('account')}
            onclick={() => go('account')}
          >
            <ProfileAvatar size={36} />
            <span class="about-text">
              <strong>{profile.name || t.settings.account}</strong>
              <small>{t.settings.accountHint}</small>
            </span>
          </button>
        {:else}
          <button
            class="item"
            class:last={p === 'about'}
            class:active={pane === p}
            aria-current={pane === p ? 'page' : undefined}
            aria-label={paneLabel(p)}
            title={paneLabel(p)}
            onclick={() => go(p)}
          >
            <PaneIcon pane={p} />
            <span class="name">{paneLabel(p)}</span>
          </button>
        {/if}
      {/each}
      {#if found.length === 0}
        <p class="none">{t.settings.noResults}</p>
      {/if}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <div class="history">
        <button
          aria-label={t.settings.back}
          title={t.settings.back}
          disabled={at === 0}
          onclick={() => step(-1)}
        >
          <Icon name="chevronLeft" size={14} strokeWidth={2.2} />
        </button>
        <span class="sep"></span>
        <button
          aria-label={t.settings.forward}
          title={t.settings.forward}
          disabled={at >= history.length - 1}
          onclick={() => step(1)}
        >
          <Icon name="chevronRight" size={14} strokeWidth={2.2} />
        </button>
      </div>
      <h2 class="title">{paneLabel(pane)}</h2>
    </div>

    <div class="content">
      {#if pane === 'general'}
        <div class="card">
          <div class="row">
            {@render head(t.settings.language, t.settings.languageHint)}
            <!-- A list rather than a segmented control: languages keep being added. -->
            <select class="input pick" aria-label={t.settings.language} bind:value={pickedLang}>
              {#each LANGS as l (l)}
                <option value={l}>{t.settings.langs[l]}</option>
              {/each}
            </select>
          </div>
          {#if pickedLang !== lang}
            <div class="row">
              <small>{t.settings.langRestart}</small>
              <button type="button" class="btn" onclick={() => switchLang(pickedLang)}
                >{t.settings.langApply}</button
              >
            </div>
          {/if}
        </div>
      {:else if pane === 'account'}
        <div class="profile">
          <button
            class="avatar-edit"
            bind:this={avatarButton}
            aria-label={t.settings.avatarPick}
            title={t.settings.avatarPick}
            aria-haspopup={profile.avatar ? 'menu' : undefined}
            onclick={openAvatarMenu}
          >
            <ProfileAvatar size={88} />
            <span class="avatar-cover" aria-hidden="true"><Icon name="camera" size={24} /></span>
          </button>
          <input
            class="file"
            type="file"
            accept={IMAGE_ACCEPT}
            tabindex="-1"
            aria-hidden="true"
            bind:this={avatarInput}
            onchange={pickAvatar}
          />
          {#if editingName}
            <!-- svelte-ignore a11y_autofocus -->
            <input
              class="name-field"
              type="text"
              maxlength={MAX_NAME}
              placeholder={t.settings.profileNamePlaceholder}
              aria-label={t.settings.profileName}
              spellcheck="false"
              autofocus
              value={profile.name}
              onfocus={(e) => e.currentTarget.select()}
              onkeydown={(e) => {
                if (e.key === 'Enter') e.currentTarget.blur()
                if (e.key === 'Escape') {
                  e.currentTarget.value = profile.name
                  editingName = false
                }
              }}
              onblur={(e) => {
                if (editingName) profile.set({ name: e.currentTarget.value.trim() })
                editingName = false
              }}
            />
          {:else}
            <button
              class="name-field"
              class:empty={!profile.name}
              title={t.settings.profileNameEdit}
              onclick={() => (editingName = true)}
            >
              {profile.name || t.settings.profileNamePlaceholder}
            </button>
          {/if}
          {#if avatarMsg}<small class="msg">{avatarMsg}</small>{/if}
        </div>
        {#if avatarMenu}
          <PopupMenu
            x={avatarMenu.x}
            y={avatarMenu.y}
            label={t.settings.avatarPick}
            anchor={avatarButton}
            actions={[
              { label: t.settings.avatarPick, run: () => avatarInput?.click() },
              {
                label: t.settings.avatarRemove,
                danger: true,
                run: () => profile.set({ avatar: '' }),
              },
            ]}
            onclose={() => (avatarMenu = null)}
          />
        {/if}
        <p class="intro">{t.settings.profileHint}</p>

        <h3>{t.settings.lock}</h3>
        <div class="card">
          {#if !appLock.state.enabled}
            <form class="row lockform" onsubmit={saveLock}>
              <small>{t.settings.lockIntro}</small>
              <label class="field">
                <span>{t.settings.lockNew}</span>
                <input
                  class="input"
                  type="password"
                  autocomplete="new-password"
                  bind:value={next}
                />
              </label>
              <label class="field">
                <span>{t.settings.lockConfirm}</span>
                <input
                  class="input"
                  type="password"
                  autocomplete="new-password"
                  bind:value={again}
                />
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
              {@render head(
                t.settings.lockOn,
                appLock.state.biometric ? t.settings.lockOnTouchID : t.settings.lockOnPassword,
              )}
              <button class="btn" onclick={() => appLock.lock()}>{t.settings.lockNow}</button>
            </div>
            <div class="row">
              {@render head(t.settings.lockAfter, t.settings.lockAfterHint)}
              {@render segmented(
                t.settings.lockAfter,
                LOCK_AFTER,
                settings.lockAfter,
                (m) => t.settings.lockAfters[m as keyof typeof t.settings.lockAfters],
                (m) => settings.set('lockAfter', m),
              )}
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
                    <input
                      class="input"
                      type="password"
                      autocomplete="new-password"
                      bind:value={next}
                    />
                  </label>
                  <label class="field">
                    <span>{t.settings.lockConfirm}</span>
                    <input
                      class="input"
                      type="password"
                      autocomplete="new-password"
                      bind:value={again}
                    />
                  </label>
                {/if}
                <div class="formfoot">
                  <small class="msg"
                    >{lockForm === 'change' && next ? (lockProblem ?? '') : lockMsg}</small
                  >
                  <button type="button" class="btn" onclick={closeLockForm}>{t.lock.cancel}</button>
                  <button
                    class="btn {lockForm === 'disable' ? 'btn-danger' : 'btn-primary'}"
                    disabled={lockBusy ||
                      !current ||
                      (lockForm === 'change' && lockProblem !== null)}
                    >{lockForm === 'disable' ? t.settings.lockDisable : t.settings.lockSave}</button
                  >
                </div>
              </form>
            {:else}
              <div class="row lockactions">
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
        </div>
      {:else if pane === 'about'}
        <div class="aboutpane">
          <DavishMark size={72} />
          <strong>Davish</strong>
          <small>{t.settings.version(__APP_VERSION__)}</small>
          <p>{t.help.aboutBody}</p>
          <small>{t.settings.license}</small>
        </div>

        <h3>{t.settings.updates}</h3>
        <div class="card">
          {@render toggle(
            t.settings.autoUpdate,
            t.settings.autoUpdateHint,
            settings.autoUpdate,
            (on) => settings.set('autoUpdate', on),
          )}
          <div class="row">
            {@render head(t.settings.version(__APP_VERSION__))}
            <button class="btn" disabled={!hasRuntime} onclick={() => void updates.check(true)}
              >{t.settings.checkNow}</button
            >
          </div>
        </div>
      {:else if pane === 'appearance'}
        <div class="card">
          <div class="row stack">
            {@render head(t.settings.theme, t.settings.themeHint)}
            <div class="themes" role="radiogroup" aria-label={t.settings.theme}>
              {#each THEMES as th (th)}
                <button
                  class="theme"
                  role="radio"
                  aria-checked={settings.theme === th}
                  onclick={() => settings.set('theme', th)}
                >
                  <span class="preview"><ThemePreview theme={th} /></span>
                  <span>{t.settings.themes[th]}</span>
                </button>
              {/each}
            </div>
          </div>
        </div>

        <h3>{t.settings.wallpaper}</h3>
        <div class="card">
          <div class="row">
            <div class="swatches" role="radiogroup" aria-label={t.settings.wallpaper}>
              <span class="custom-slot">
                <button
                  class="swatch custom"
                  role="radio"
                  aria-checked={customAccent}
                  aria-label={t.settings.wallpaperCustom}
                  title={t.settings.wallpaperCustom}
                  style:--picked={customAccent ? settings.wallpaperAccent : undefined}
                  onclick={openPicker}
                ></button>
                <!-- The native well hides behind the wheel: WebKit draws it square, whatever its opacity. -->
                <input
                  bind:this={colorInput}
                  type="color"
                  tabindex="-1"
                  aria-hidden="true"
                  value={settings.wallpaperAccent}
                  oninput={(e) => pickAccent(e.currentTarget.value)}
                />
              </span>
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
            {@render head(t.settings.wallpaperPicture, t.settings.wallpaperPictureHint)}
            <div class="actions">
              <input
                bind:this={pictureInput}
                type="file"
                accept={WALLPAPER_ACCEPT}
                hidden
                onchange={pickPicture}
              />
              <button
                type="button"
                class="btn"
                disabled={pictureBusy}
                onclick={() => pictureInput?.click()}
                >{wallpaperImage.has
                  ? t.settings.wallpaperChange
                  : t.settings.wallpaperChoose}</button
              >
              {#if wallpaperImage.has}
                <button type="button" class="btn" disabled={pictureBusy} onclick={removePicture}
                  >{t.settings.wallpaperRemove}</button
                >
              {/if}
            </div>
          </div>
          {#if wallpaperImage.has}
            <div class="row">
              {@render head(t.settings.wallpaperDim, t.settings.wallpaperDimHint)}
              <div class="stepper">
                <button
                  aria-label="{t.settings.lower} {t.settings.wallpaperDim}"
                  disabled={settings.wallpaperDim <= WALLPAPER_DIM.min}
                  onclick={() =>
                    settings.set(
                      'wallpaperDim',
                      Math.max(WALLPAPER_DIM.min, settings.wallpaperDim - WALLPAPER_DIM.step),
                    )}
                >
                  <Icon name="minus" size={12} strokeWidth={2.2} />
                </button>
                <output aria-live="polite">{settings.wallpaperDim}%</output>
                <button
                  aria-label="{t.settings.higher} {t.settings.wallpaperDim}"
                  disabled={settings.wallpaperDim >= WALLPAPER_DIM.max}
                  onclick={() =>
                    settings.set(
                      'wallpaperDim',
                      Math.min(WALLPAPER_DIM.max, settings.wallpaperDim + WALLPAPER_DIM.step),
                    )}
                >
                  <Icon name="plus" size={12} strokeWidth={2.2} />
                </button>
              </div>
            </div>
            {#if !settings.knobs.wallpaperImage}
              <div class="row"><small class="msg">{t.settings.wallpaperHidden}</small></div>
            {/if}
          {/if}
          {#if pictureMsg}<div class="row"><small class="msg">{pictureMsg}</small></div>{/if}
        </div>

        <h3>{t.settings.monoFont}</h3>
        <div class="card">
          <div class="row">
            {@render head(t.settings.monoFont, t.settings.monoFontHint)}
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
        </div>
      {:else if pane === 'performance'}
        <div class="card">
          <div class="row stack">
            {@render head(t.settings.perfPreset, t.settings.perfPresetHint)}
            <div class="preset-line">
              {@render segmented(
                t.settings.perfPreset,
                NAMED_PRESETS,
                settings.perfPreset as NamedPreset,
                (p) => t.settings.perfPresets[p],
                (p) => settings.usePreset(p),
              )}
              <button
                type="button"
                class="btn"
                disabled={perfCheck.running}
                onclick={() => void perfCheck.start(true)}
              >
                {perfCheck.running ? t.settings.perfChecking : t.settings.perfCheck}
              </button>
            </div>
            {#if settings.perfPreset === 'custom'}
              <small>{t.settings.perfCustom}</small>
            {/if}
            {#if perfCheck.running && perfCheck.probing}
              <small>{t.settings.perfProbing}</small>
            {:else if perfCheck.pick}
              {@const pick = perfCheck.pick}
              <div class="perf-result">
                <small>
                  {t.settings.perfResult(perfCheck.fps, t.settings.perfPresets[pick])}
                  {#if settings.perfPreset === pick}{t.settings.perfOnIt}{/if}
                </small>
                {#if settings.perfPreset !== pick}
                  <button
                    type="button"
                    class="btn btn-primary"
                    onclick={() => settings.usePreset(pick)}
                  >
                    {t.settings.perfUse(t.settings.perfPresets[pick])}
                  </button>
                {/if}
              </div>
            {:else if perfCheck.last === 'slow'}
              <small>{t.settings.perfSlow}</small>
            {:else if perfCheck.last === 'fine'}
              <small>{t.settings.perfFine}</small>
            {/if}
          </div>
        </div>

        <h3>{t.settings.perfAdvanced}</h3>
        <div class="card">
          {@render toggle(
            t.settings.perfShadow,
            t.settings.perfShadowHint,
            settings.knobs.shadow === 'short',
            (on) => settings.setKnob('shadow', on ? 'short' : 'deep'),
          )}
          {#each perfKnobs as k (k.key)}
            {@render toggle(k.label, k.hint, settings.knobs[k.key], (on) =>
              settings.setKnob(k.key, on),
            )}
          {/each}
          {#if settings.perfNeedsRestart}
            <div class="row">
              <small>{t.settings.perfRestartHint}</small>
              <button
                type="button"
                class="btn"
                disabled={!hasRuntime}
                onclick={() => void relaunch()}
              >
                {t.settings.perfRestart}
              </button>
            </div>
          {/if}
        </div>
      {:else if pane === 'menubar'}
        <p class="intro">{t.settings.menubarStatsHint}</p>
        <h3>{t.settings.menubarStats}</h3>
        <div class="card">
          {#each STAT_IDS as id (id)}
            {@render toggle(
              t.menubar.stats.names[id],
              '',
              settings.menubarStats.includes(id),
              (on) => toggleStat(id, on),
            )}
          {/each}
        </div>
      {:else if pane === 'dock'}
        <div class="card">
          <div class="row">
            {@render head(t.settings.dockSize, t.settings.dockSizeHint)}
            {@render segmented(
              t.settings.dockSize,
              dockSizes,
              settings.dockSize,
              (s) => t.settings.dockSizes[s],
              (s) => settings.set('dockSize', s),
            )}
          </div>
          <div class="row">
            {@render head(t.settings.dockPosition, t.settings.dockPositionHint)}
            {@render segmented(
              t.settings.dockPosition,
              DOCK_POSITIONS,
              settings.dockPosition,
              (p) => t.settings.dockPositions[p],
              (p) => settings.set('dockPosition', p),
            )}
          </div>
        </div>
        <div class="card">
          {@render toggle(
            t.settings.dockTransparent,
            t.settings.dockTransparentHint,
            settings.dockTransparent,
            (on) => settings.set('dockTransparent', on),
          )}
          {@render toggle(
            t.settings.dockAutohide,
            t.settings.dockAutohideHint,
            settings.dockAutohide,
            (on) => settings.set('dockAutohide', on),
          )}
        </div>
      {:else if pane === 'terminal'}
        <div class="card">
          <div class="row">
            {@render head(t.settings.fontSize, t.settings.fontSizeHint)}
            {@render stepper(
              settings.terminalFontSize,
              settings.terminalFontSize <= TERMINAL_FONT.min,
              settings.terminalFontSize >= TERMINAL_FONT.max,
              t.settings.fontSmaller,
              t.settings.fontLarger,
              stepFont,
            )}
          </div>
          <div class="row stack">
            {@render head(t.settings.termScheme, t.settings.termSchemeHint)}
            <div class="themes" role="radiogroup" aria-label={t.settings.termScheme}>
              {#each TERMINAL_SCHEMES as id (id)}
                {@const sw = swatch(id)}
                <button
                  class="theme"
                  role="radio"
                  aria-checked={settings.terminalScheme === id}
                  onclick={() => settings.set('terminalScheme', id)}
                >
                  <span class="preview">
                    <span class="scheme" style:background={sw.bg} style:color={sw.fg}>
                      <span>
                        <span style:color={sw.accents[0]}>~</span>
                        <span style:color={sw.accents[1]}>$</span> ls
                      </span>
                      <span class="dots">
                        {#each sw.accents as c (c)}<i style:background={c}></i>{/each}
                      </span>
                    </span>
                  </span>
                  <span>{t.settings.termSchemes[id]}</span>
                </button>
              {/each}
            </div>
          </div>
          <label class="row">
            {@render head(t.settings.termFont, t.settings.termFontHint)}
            <input
              class="input fontname"
              type="text"
              spellcheck="false"
              autocomplete="off"
              maxlength="60"
              placeholder={t.settings.termFontPlaceholder}
              value={settings.terminalFont}
              onchange={(e) => {
                settings.set('terminalFont', cleanFontName(e.currentTarget.value))
                e.currentTarget.value = settings.terminalFont
              }}
            />
          </label>
          <div class="row">
            {@render head(t.settings.termCursor)}
            {@render segmented(
              t.settings.termCursor,
              TERMINAL_CURSORS,
              settings.terminalCursor,
              (c) => t.settings.termCursors[c],
              (c) => settings.set('terminalCursor', c),
            )}
          </div>
          {@render toggle(
            t.settings.termBlink,
            t.settings.termBlinkHint,
            settings.terminalBlink,
            (on) => settings.set('terminalBlink', on),
          )}
          {@render toggle(t.settings.prompt, t.settings.promptHint, settings.terminalPrompt, (on) =>
            settings.set('terminalPrompt', on),
          )}
          {@render toggle(t.settings.gpu, t.settings.gpuHint, settings.terminalGpu, (on) =>
            settings.set('terminalGpu', on),
          )}
          {@render toggle(
            t.settings.termSolid,
            t.settings.termSolidHint,
            settings.terminalOpaque,
            (on) => settings.set('terminalSolid', on),
          )}
          <div class="row">
            {@render head(t.settings.scrollback, t.settings.scrollbackHint)}
            {@render segmented(
              t.settings.scrollback,
              TERMINAL_SCROLLBACK,
              settings.terminalScrollback,
              (n) => n.toLocaleString(locale),
              (n) => settings.set('terminalScrollback', n),
            )}
          </div>
        </div>
      {:else if pane === 'editor'}
        <div class="card">
          <div class="row">
            {@render head(t.settings.editorFont, t.settings.editorFontHint)}
            {@render stepper(
              settings.editorFontSize,
              settings.editorFontSize <= EDITOR_FONT.min,
              settings.editorFontSize >= EDITOR_FONT.max,
              t.settings.fontSmaller,
              t.settings.fontLarger,
              stepEditorFont,
            )}
          </div>
          {@render toggle(
            t.settings.editorWrap,
            t.settings.editorWrapHint,
            settings.editorWrap,
            (on) => settings.set('editorWrap', on),
          )}
          {@render toggle(
            t.settings.editorLineNumbers,
            t.settings.editorLineNumbersHint,
            settings.editorLineNumbers,
            (on) => settings.set('editorLineNumbers', on),
          )}
          {@render toggle(
            t.settings.editorWhitespace,
            t.settings.editorWhitespaceHint,
            settings.editorWhitespace,
            (on) => settings.set('editorWhitespace', on),
          )}
          <div class="row">
            {@render head(t.settings.editorIndent, t.settings.editorIndentHint)}
            {@render segmented(
              t.settings.editorIndent,
              EDITOR_INDENTS,
              settings.editorIndent,
              (i) => t.settings.editorIndents[i],
              (i) => settings.set('editorIndent', i),
            )}
          </div>
          <div class="row">
            {@render head(t.settings.backup, t.settings.backupHint)}
            {@render segmented(
              t.settings.backup,
              backups,
              settings.editorBackup,
              (b) => t.settings.backups[b],
              (b) => settings.set('editorBackup', b),
            )}
          </div>
        </div>
      {:else if pane === 'files'}
        <div class="card">
          {@render toggle(
            t.settings.filesHidden,
            t.settings.filesHiddenHint,
            settings.filesShowHidden,
            (on) => settings.set('filesShowHidden', on),
          )}
        </div>
      {:else if pane === 'alerts'}
        <div class="card">
          {#each pctRows as r (r.key)}
            <div class="row">
              {@render head(r.label, r.hint)}
              <div class="stepper">
                <button
                  aria-label="{t.settings.lower} {r.label}"
                  disabled={settings[r.key] <= ALERTS.pct.min}
                  onclick={() =>
                    settings.set(
                      r.key,
                      Math.max(ALERTS.pct.min, settings[r.key] - ALERTS.pct.step),
                    )}
                >
                  <Icon name="minus" size={12} strokeWidth={2.2} />
                </button>
                <output aria-live="polite">{settings[r.key]}%</output>
                <button
                  aria-label="{t.settings.higher} {r.label}"
                  disabled={settings[r.key] >= ALERTS.pct.max}
                  onclick={() =>
                    settings.set(
                      r.key,
                      Math.min(ALERTS.pct.max, settings[r.key] + ALERTS.pct.step),
                    )}
                >
                  <Icon name="plus" size={12} strokeWidth={2.2} />
                </button>
              </div>
            </div>
          {/each}
          <div class="row">
            {@render head(t.settings.alertHold, t.settings.alertHoldHint)}
            {@render segmented(
              t.settings.alertHold,
              ALERTS.holds,
              settings.alertHold,
              (h) => t.settings.holds[h] ?? '',
              (h) => settings.set('alertHold', h),
            )}
          </div>
        </div>
      {:else if pane === 'security'}
        <div class="card">
          <div class="row">
            {@render head(t.settings.lock, t.settings.lockMoved)}
            <button class="btn" onclick={() => go('account')}>{t.settings.openAccount}</button>
          </div>
        </div>

        <div class="card">
          <div class="row">
            {@render head(t.settings.sudoForget, t.settings.sudoForgetHint)}
            {@render segmented(
              t.settings.sudoForget,
              SUDO_FORGET_AFTER,
              settings.sudoForgetAfter,
              (m) => t.settings.sudoForgets[m] ?? '',
              (m) => settings.set('sudoForgetAfter', m),
            )}
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  /* System Settings: a translucent sidebar of colored tiles beside an
     almost opaque pane of grouped cards. */
  .settings {
    flex: 1;
    min-height: 0;
    display: flex;
  }

  .sidebar {
    width: 210px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0 10px 8px;
    padding: 5px 8px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg-subtle);
    transition: box-shadow var(--dur-fast) var(--ease-out);
  }

  .search:focus-within {
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .search input {
    flex: 1;
    min-width: 0;
    border: 0;
    padding: 0;
    background: none;
    outline: none;
    color: var(--fg);
  }

  .search input::-webkit-search-cancel-button {
    cursor: pointer;
  }

  .items {
    flex: 1;
    min-height: 0;
    padding: 0 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 1px;
    overflow-y: auto;
  }

  .item,
  .about {
    display: flex;
    align-items: center;
    gap: 9px;
    text-align: left;
    border: 0;
    background: none;
    color: inherit;
    padding: 5px 8px;
    border-radius: var(--control-radius);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .about {
    margin-bottom: 8px;
    padding: 6px 8px;
  }

  .about-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .about-text strong,
  .about-text small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .about-text strong {
    font-weight: 600;
  }

  .item:hover:not(.active),
  .about:hover:not(.active) {
    background: var(--row-hover);
  }

  .item.active {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .about.active {
    background: var(--sidebar-active);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .none {
    margin: 12px 8px;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
    border-left: 1px solid var(--divider-soft);
  }

  .toolbar {
    height: var(--window-titlebar-h);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 16px;
  }

  .history {
    display: flex;
    align-items: center;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .history button {
    display: flex;
    padding: 4px 7px;
    border: 0;
    border-radius: 8px;
    background: none;
    color: inherit;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .history button:hover:not(:disabled) {
    background: var(--fill-track);
  }

  .history button:disabled {
    opacity: 0.3;
  }

  .sep {
    width: 1px;
    height: 14px;
    background: var(--divider);
  }

  .title {
    min-width: 0;
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 6px 20px 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  h3 {
    margin: 6px 0 -4px 12px;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--fg-muted);
  }

  /* The account pane opens with the picture, as in System Settings. */
  .profile {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 8px 0 4px;
  }

  .profile .msg {
    text-align: center;
  }

  /* The picture: hover dims it under a camera; a click changes or removes it. */
  .avatar-edit {
    position: relative;
    display: block;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: none;
    cursor: pointer;
  }

  .avatar-cover {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.4);
    color: #fff;
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
    /* A layer of its own from the start: WebKit otherwise moves the camera by a
       half pixel when the fade begins and ends. */
    will-change: opacity;
  }

  .avatar-edit:hover .avatar-cover,
  .avatar-edit:focus-visible .avatar-cover {
    opacity: 1;
  }

  .avatar-edit:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  /* The name and its field share one box, so editing does not move anything. */
  .name-field {
    box-sizing: border-box;
    width: min(300px, 100%);
    height: 34px;
    padding: 0 10px;
    border: 1px solid transparent;
    border-radius: 8px;
    background: transparent;
    color: var(--fg);
    font: inherit;
    font-size: 18px;
    font-weight: 600;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  button.name-field {
    cursor: text;
  }

  button.name-field:hover {
    background: var(--fill-soft);
  }

  button.name-field.empty {
    color: var(--fg-muted);
    font-weight: 500;
  }

  input.name-field {
    border-color: var(--accent);
    background: var(--content-bg);
    box-shadow: 0 0 0 3px var(--accent-soft);
    outline: none;
  }

  .file {
    display: none;
  }

  .aboutpane {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 12px 24px 4px;
    text-align: center;
  }

  .aboutpane strong {
    margin-top: 6px;
    font-size: var(--text-lg);
    font-weight: 600;
  }

  .aboutpane p {
    max-width: 420px;
    margin: 8px 0 4px;
    font-size: var(--text-sm);
  }

  .item.last {
    margin-top: 10px;
  }

  .intro {
    margin: 0 12px;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .card {
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    padding: 0 14px;
    border-radius: 10px;
    background: var(--fill-soft);
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 40px;
    padding: 9px 0;
  }

  .row + .row {
    border-top: 1px solid var(--divider-soft);
  }

  .row.stack {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .head {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .pick {
    flex: none;
    width: auto;
    min-width: 160px;
  }

  .themes {
    display: flex;
    justify-content: space-around;
    gap: 12px;
    flex-wrap: wrap;
  }

  .theme {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    border: 0;
    padding: 0;
    background: none;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .preview {
    padding: 2px;
    border: 2px solid transparent;
    border-radius: 10px;
    transition: border-color var(--dur-fast) var(--ease-out);
  }

  .theme[aria-checked='true'] {
    color: var(--fg);
    font-weight: 600;
  }

  .theme[aria-checked='true'] .preview {
    border-color: var(--accent);
  }

  .theme:hover:not([aria-checked='true']) .preview {
    border-color: var(--divider);
  }

  .fontname {
    flex-shrink: 0;
    width: 200px;
  }

  /* A tile of a terminal scheme: its background, a prompt and three accent dots. */
  .scheme {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    box-sizing: border-box;
    width: 88px;
    height: 56px;
    padding: 8px 9px;
    border-radius: 7px;
    font: 600 11px/1.2 var(--font-mono, monospace);
    text-align: left;
  }

  .scheme .dots {
    display: flex;
    gap: 4px;
  }

  .scheme .dots i {
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }

  /* A choice of a few, like macOS segmented controls. */
  .segmented {
    flex-shrink: 0;
    display: flex;
    flex-wrap: wrap;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .perf-result {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .preset-line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
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

  /* Segmented − value + control, like the macOS font size stepper. */
  .actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

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
    flex-wrap: wrap;
    gap: 10px;
  }

  .swatch {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 2px solid var(--glass-border);
    padding: 0;
    transition:
      transform var(--dur-fast) var(--ease-spring),
      box-shadow var(--dur-fast) var(--ease-out);
  }

  .swatch:hover {
    transform: scale(1.12);
  }

  .swatch[aria-checked='true'] {
    border-color: var(--content-bg);
    box-shadow: 0 0 0 2px var(--accent);
  }

  /* Any color: a color wheel, showing the picked color in its middle once chosen. */
  .swatch.custom {
    position: relative;
    cursor: pointer;
    background:
      radial-gradient(circle, var(--picked, transparent) 0 38%, transparent 40%),
      conic-gradient(#ff6b6b, #ffd93d, #6bcb77, #4dd4e6, #4d96ff, #b06bff, #ff6bb5, #ff6b6b);
    /* Inside the border only: under the see-through border the wheel's corners showed as a square. */
    background-clip: padding-box;
  }

  .custom-slot {
    position: relative;
    display: inline-flex;
    isolation: isolate;
  }

  /* Small and centered behind the wheel, so it never shows and the picker opens under the wheel. */
  .custom-slot input {
    position: absolute;
    left: 50%;
    top: 50%;
    z-index: -1;
    width: 12px;
    height: 12px;
    margin: -6px 0 0 -6px;
    padding: 0;
    border: 0;
    opacity: 0;
    pointer-events: none;
  }

  /* A narrow window keeps only the tiles, as in Files and Monitor. */
  @container window (max-width: 560px) {
    .sidebar {
      width: var(--sidebar-rail-w);
    }

    .search,
    .about,
    .name,
    .none {
      display: none;
    }

    .item {
      justify-content: center;
    }

    .row:not(.stack) {
      flex-wrap: wrap;
    }
  }
</style>
