<script lang="ts">
  import type { NginxReload, NginxTest } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { openFileInEditor } from '../../lib/wm/intents.svelte'

  interface Props {
    serverID: string
    /** A test or a reload to report; closed while null. */
    result: { kind: 'test'; test: NginxTest } | { kind: 'reload'; reload: NginxReload } | null
    onclose: () => void
  }

  let { serverID, result, onclose }: Props = $props()

  const test = $derived(result?.kind === 'test' ? result.test : result?.reload.test)
  const reload = $derived(result?.kind === 'reload' ? result.reload : null)
  const errors = $derived(test?.issues.filter((i) => i.level !== 'warn') ?? [])
  const warnings = $derived(test?.issues.filter((i) => i.level === 'warn') ?? [])

  function open(file: string, line?: number) {
    openFileInEditor(serverID, file, line)
    onclose()
  }
</script>

<Dialog
  open={result !== null}
  title={result ? t.nginx.resultTitle[result.kind] : ''}
  width={540}
  {onclose}
>
  {#if test}
    {#if reload?.reloaded}
      <p class="state ok"><Icon name="check" size={15} strokeWidth={2.6} /> {t.nginx.reloaded}</p>
    {:else if reload?.needsRoot}
      <NeedsRoot
        {serverID}
        command={reload.sudo}
        message={t.nginx.reloadNeedsRoot}
        onopen={onclose}
      />
    {:else if test.ok}
      <p class="state ok"><Icon name="check" size={15} strokeWidth={2.6} /> {t.nginx.testOk}</p>
    {:else if test.needsRoot}
      <p class="state warn"><Icon name="lock" size={15} /> {t.nginx.testNeedsRoot}</p>
    {:else}
      <p class="state bad">
        <Icon name="warning" size={15} />
        {reload ? t.nginx.testFailed : t.nginx.testFailedOnly}
      </p>
    {/if}

    {#if errors.length > 0 || warnings.length > 0}
      <ul class="issues">
        {#each [...errors, ...warnings] as is, i (i)}
          <li class={is.level === 'warn' ? 'warn' : 'bad'}>
            <span class="msg">{is.message}</span>
            {#if is.file}
              <button type="button" class="link" onclick={() => open(is.file!, is.line)}>
                {t.nginx.openAt(is.file, is.line)}
              </button>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}

    <details>
      <summary>{t.nginx.output}</summary>
      <!-- Server output is shown as text only (docs/SECURITY.md). -->
      <pre class="output">{test.output}</pre>
    </details>
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.nginx.close}</button>
  {/snippet}
</Dialog>

<style>
  .state {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .state :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
  }

  .state.ok {
    color: var(--ok);
  }

  .state.warn {
    color: var(--warn);
  }

  .state.bad {
    color: var(--danger);
  }

  .issues {
    margin: 0 0 10px;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .issues li {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 6px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    border-left: 0;
  }

  .msg {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    word-break: break-word;
    user-select: text;
  }

  .issues .bad .msg {
    color: var(--danger);
  }

  .issues .warn .msg {
    color: var(--warn);
  }

  .link {
    align-self: flex-start;
    padding: 0;
    border: 0;
    background: none;
    font-size: var(--text-xs);
    color: var(--accent);
  }

  .link:hover {
    text-decoration: underline;
  }

  summary {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    cursor: pointer;
  }

  .output {
    margin: 6px 0 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
  }

  .spacer {
    flex: 1;
  }
</style>
