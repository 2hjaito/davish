<script lang="ts">
  import { t } from '../lib/i18n/vi'
  import TitleBar from '../lib/wm/TitleBar.svelte'
  import type { AppProps } from './types'

  let { win }: AppProps = $props()
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="spacer"></span>
</TitleBar>
<div class="body">
  <strong>{t.placeholder.title}</strong>
  <span>{t.placeholder.body}</span>
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-size: var(--text-md);
    font-weight: 600;
  }

  .spacer {
    width: 52px;
  }

  .body {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    font-size: var(--text-md);
    opacity: 0.75;
  }
</style>
