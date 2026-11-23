<script lang="ts">
  import type { Snippet } from 'svelte'

  /** neutral for a plain reading; the others tint the card by how things are. */
  export type Tone = 'neutral' | 'ok' | 'warn' | 'danger'

  interface Props {
    tone?: Tone
    children: Snippet
  }

  let { tone = 'neutral', children }: Props = $props()
</script>

<!-- The face of a widget: dark in both themes, tinted from its top left by the state. -->
<div class="card {tone}">{@render children()}</div>

<style>
  .card {
    position: relative;
    box-sizing: border-box;
    height: 100%;
    padding: 12px 14px;
    overflow: hidden;
    border: 1px solid var(--widget-edge);
    border-radius: var(--widget-radius);
    background: linear-gradient(165deg, var(--widget-top), var(--widget-base) 70%);
    color: var(--fg-on-dark);
  }

  .card.ok {
    background:
      radial-gradient(120% 90% at 0% 0%, var(--widget-tint-ok), transparent 65%),
      linear-gradient(165deg, var(--widget-top), var(--widget-base) 70%);
  }

  .card.warn {
    background:
      radial-gradient(120% 90% at 0% 0%, var(--widget-tint-warn), transparent 65%),
      linear-gradient(165deg, var(--widget-top), var(--widget-base) 70%);
  }

  .card.danger {
    background:
      radial-gradient(120% 90% at 0% 0%, var(--widget-tint-danger), transparent 65%),
      linear-gradient(165deg, var(--widget-top), var(--widget-base) 70%);
  }
</style>
