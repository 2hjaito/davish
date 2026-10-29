<script lang="ts">
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'

  // The folders searched for repositories besides the usual ones.
  interface Props {
    open: boolean
    folders: string[]
    onchange: (folders: string[]) => void
    onclose: () => void
  }

  let { open, folders, onchange, onclose }: Props = $props()

  let input = $state('')
  let bad = $state(false)

  function add() {
    // Same rule as git.checkExtra: absolute, one line.
    const p = input.trim().replace(/(.)\/+$/, '$1')
    if (!p.startsWith('/') || /[\r\n]/.test(p)) {
      bad = true
      return
    }
    if (!folders.includes(p)) onchange([...folders, p])
    input = ''
    bad = false
  }
</script>

<Dialog {open} title={t.git.addFolderTitle} width={460} {onclose}>
  <form id="git-folder" onsubmit={(e) => (e.preventDefault(), add())}>
    <label class="field">
      <span>{t.git.addFolderLabel}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input mono"
        spellcheck="false"
        autocomplete="off"
        autofocus
        placeholder="/home/deploy/apps"
        bind:value={input}
        oninput={() => (bad = false)}
      />
      <small class:bad>{bad ? t.git.addFolderBad : t.git.addFolderHint}</small>
    </label>
  </form>
  {#if folders.length > 0}
    <h4>{t.git.folders}</h4>
    <ul>
      {#each folders as f (f)}
        <li>
          <code>{f}</code>
          <button
            type="button"
            class="remove"
            aria-label={t.git.removeFolder(f)}
            title={t.git.removeFolder(f)}
            onclick={() => onchange(folders.filter((o) => o !== f))}
          >
            <Icon name="close" size={12} strokeWidth={2.2} />
          </button>
        </li>
      {/each}
    </ul>
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button type="submit" form="git-folder" class="btn btn-primary" disabled={!input.trim()}
      >{t.git.addFolderAction}</button
    >
  {/snippet}
</Dialog>

<style>
  small.bad {
    color: var(--danger);
  }

  h4 {
    margin: 14px 0 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 3px 0;
  }

  code {
    flex: 1;
    min-width: 0;
    overflow-wrap: anywhere;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .remove {
    display: flex;
    padding: 4px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
  }

  .remove:hover {
    background: var(--row-hover);
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
