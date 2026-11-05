<script lang="ts">
  import { untrack } from 'svelte'

  import { t } from '../i18n'
  import { type Shortcut, type ShortcutLook } from '../stores/shortcuts.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import Icon from '../ui/Icon.svelte'
  import {
    COLOR_CHOICES,
    type ColorChoice,
    ICON_CHOICES,
    type IconChoice,
    IMAGE_ACCEPT,
    iconImage,
    iconOf,
    labelFor,
    titleOf,
  } from './shortcutLook'
  import ShortcutTile from './ShortcutTile.svelte'

  interface Props {
    item: Shortcut
    /** The picture item shows now, if any. */
    currentImage: string
    onclose: () => void
    onsave: (look: ShortcutLook) => void
  }

  let { item, currentImage, onclose, onsave }: Props = $props()

  const s = t.shortcuts
  const colors = Object.entries(COLOR_CHOICES) as [ColorChoice, string][]

  // A draft: nothing changes on the desktop until Save.
  let title = $state(untrack(() => titleOf(item)))
  let icon = $state(untrack(() => item.icon))
  let color = $state(untrack(() => item.color))
  let image = $state(untrack(() => currentImage))
  let failed = $state(false)
  let picker = $state<HTMLInputElement>()

  async function pick(ev: Event) {
    const input = ev.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    input.value = '' // so the same file can be picked again
    if (!file) return
    failed = false
    try {
      image = await iconImage(file)
    } catch {
      failed = true
    }
  }

  // A picture hides the glyph, so choosing a glyph means leaving the picture.
  function chooseIcon(choice: IconChoice | undefined) {
    icon = choice
    image = ''
  }

  function chooseColor(choice: string | undefined) {
    color = choice
    image = ''
  }

  function reset() {
    title = item.name
    icon = undefined
    color = undefined
    image = ''
    failed = false
  }

  function save() {
    onsave({ label: labelFor(title, item.name), icon, color, image })
  }

  function onKeyDown(ev: KeyboardEvent) {
    if (ev.key !== 'Enter') return
    ev.preventDefault()
    save()
  }
</script>

<Dialog open title={s.customizeTitle(item.name)} {onclose} width={380}>
  <div class="head">
    <ShortcutTile icon={iconOf({ icon, kind: item.kind })} {color} {image} />
    <label class="field">
      <span>{s.nameField}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input"
        bind:value={title}
        maxlength="64"
        spellcheck="false"
        autofocus
        placeholder={item.name}
        onfocus={(e) => e.currentTarget.select()}
        onkeydown={onKeyDown}
      />
    </label>
  </div>
  <p class="note">{s.nameNote}</p>

  <h3>{s.iconField}</h3>
  <div class="choices">
    <button
      type="button"
      class="choice"
      aria-pressed={!icon && !image}
      aria-label={s.iconDefault}
      data-own-tooltip
      title={s.iconDefault}
      onclick={() => chooseIcon(undefined)}
    >
      <Icon name={iconOf({ kind: item.kind })} size={16} />
      <span class="auto">A</span>
    </button>
    {#each ICON_CHOICES as choice (choice)}
      <button
        type="button"
        class="choice"
        aria-pressed={icon === choice && !image}
        aria-label={s.iconNames[choice]}
        data-own-tooltip
        title={s.iconNames[choice]}
        onclick={() => chooseIcon(choice)}
      >
        <Icon name={choice} size={16} />
      </button>
    {/each}
  </div>

  <h3>{s.colorField}</h3>
  <div class="choices">
    <button
      type="button"
      class="swatch glass"
      aria-pressed={!color && !image}
      aria-label={s.colorDefault}
      data-own-tooltip
      title={s.colorDefault}
      onclick={() => chooseColor(undefined)}
    ></button>
    {#each colors as [name, value] (name)}
      <button
        type="button"
        class="swatch"
        style:background={value}
        aria-pressed={color === value && !image}
        aria-label={s.colorNames[name]}
        data-own-tooltip
        title={s.colorNames[name]}
        onclick={() => chooseColor(value)}
      ></button>
    {/each}
  </div>

  <h3>{s.imageField}</h3>
  <div class="image">
    <input
      bind:this={picker}
      class="picker"
      type="file"
      accept={IMAGE_ACCEPT}
      tabindex="-1"
      aria-hidden="true"
      onchange={pick}
    />
    <button type="button" class="btn" onclick={() => picker?.click()}>{s.chooseImage}</button>
    {#if image}
      <button type="button" class="btn" onclick={() => (image = '')}>{s.removeImage}</button>
    {/if}
  </div>
  <p class="note" class:error={failed} role={failed ? 'alert' : undefined}>
    {failed ? s.imageFailed : s.imageNote}
  </p>

  {#snippet footer()}
    <button type="button" class="btn" onclick={reset}>{s.resetLook}</button>
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-primary" onclick={save}>{s.save}</button>
  {/snippet}
</Dialog>

<style>
  .head {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .field {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    font-size: var(--text-xs);
    font-weight: 500;
  }

  .field input {
    font-size: var(--text-sm);
    font-weight: 400;
  }

  h3 {
    margin: 14px 0 6px;
    font-size: var(--text-xs);
    font-weight: 600;
  }

  .note {
    margin: 6px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
    line-height: 1.4;
  }

  .note.error {
    color: var(--danger);
  }

  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .choice {
    position: relative;
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    padding: 0;
    border: none;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg);
    cursor: default;
  }

  .auto {
    position: absolute;
    right: 2px;
    bottom: 1px;
    font-size: 8px;
    font-weight: 700;
  }

  .swatch {
    width: 24px;
    height: 24px;
    padding: 0;
    border-radius: 50%;
    cursor: default;
  }

  .swatch:not(.glass) {
    border: none;
  }

  .choice:focus-visible,
  .swatch:focus-visible,
  [aria-pressed='true'] {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .image {
    display: flex;
    gap: 8px;
  }

  .picker {
    display: none;
  }

  .spacer {
    flex: 1;
  }
</style>
