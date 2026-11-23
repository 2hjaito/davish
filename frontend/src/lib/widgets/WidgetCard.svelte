<script lang="ts">
  import type { Snippet } from 'svelte'

  /** neutral for a plain reading; the others tint the card by how things are. */
  export type Tone = 'neutral' | 'ok' | 'warn' | 'danger'

  interface Props {
    tone?: Tone
    /** A CSS color: what kind of widget it is, which tints the card unless the state does. */
    accent?: string
    children: Snippet
  }

  let { tone = 'neutral', accent, children }: Props = $props()
</script>

<!-- The face of a widget: dark in both themes, tinted from its top left by its kind, or by the state when something is wrong. -->
<div class="card {tone}" style:--accent={accent}>{@render children()}</div>

<style>
  .card {
    --tint: color-mix(in srgb, var(--accent, transparent) 24%, transparent);
    position: relative;
    box-sizing: border-box;
    height: 100%;
    padding: 12px 14px;
    overflow: hidden;
    border: 1px solid var(--widget-edge);
    border-radius: var(--widget-radius);
    background:
      radial-gradient(120% 90% at 0% 0%, var(--tint), transparent 65%),
      linear-gradient(165deg, var(--widget-top), var(--widget-base) 70%);
    color: var(--fg-on-dark);
  }

  /* Wrong or close to it wins over the kind. */
  .card.warn {
    --tint: var(--widget-tint-warn);
  }

  .card.danger {
    --tint: var(--widget-tint-danger);
  }
</style>
