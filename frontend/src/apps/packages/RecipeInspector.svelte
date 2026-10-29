<script lang="ts">
  import type { Recipe } from '../../lib/api/packages'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'

  interface Props {
    serverID: string
    recipe: Recipe
    /** The installed version; empty when not installed. */
    version: string
  }

  let { serverID, recipe, version }: Props = $props()
  const r = t.packages.recipes
</script>

<section class="inspector">
  <h3>{recipe.label}</h3>
  {#if version}
    <p class="ok"><Icon name="check" size={14} strokeWidth={2.6} /> {r.installed(version)}</p>
  {:else}
    <p class="muted">{r.notInstalled}</p>
  {/if}
  <p>{r.body}</p>
  <!-- The whole script shows before it runs (docs/SECURITY.md). -->
  <pre class="code">{recipe.script}</pre>
  <div class="actions">
    <button
      type="button"
      class="btn {version ? '' : 'btn-primary'}"
      onclick={() => openTerminalAt(serverID, undefined, recipe.command)}
    >
      <Icon name="terminal" size={13} />
      {version ? r.reinstall : r.install}
    </button>
  </div>
</section>

<style>
  .inspector {
    display: flex;
    flex-direction: column;
    gap: 8px;
    height: 100%;
    min-height: 0;
    padding: 16px;
    box-sizing: border-box;
  }

  h3,
  p {
    margin: 0;
  }

  .ok {
    display: flex;
    gap: 6px;
    align-items: center;
    color: var(--ok);
  }

  .muted {
    color: var(--fg-muted);
  }

  .code {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 8px 10px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre;
    user-select: text;
  }

  .actions {
    display: flex;
    gap: 8px;
  }
</style>
