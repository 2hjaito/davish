<script lang="ts">
  import Icon from '../ui/Icon.svelte'
  import type { WidgetKind } from './kinds'
  import { LOOK } from './look'

  interface Props {
    kind: WidgetKind
    size?: number
  }

  let { kind, size = 18 }: Props = $props()
</script>

<span
  class="badge"
  style:--accent={LOOK[kind].accent}
  style:width="{size}px"
  style:height="{size}px"
  aria-hidden="true"
>
  <Icon name={LOOK[kind].icon} size={Math.round(size * 0.64)} strokeWidth={2.2} />
</span>

<style>
  .badge {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    background: color-mix(in srgb, var(--accent) 26%, transparent);
    color: var(--accent);
  }
</style>
