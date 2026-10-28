<script lang="ts">
  import { untrack } from 'svelte'

  import { type Image, type ImageDetail, type ImagePlan, imageRef } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    image: Image
    onchanged: () => void
  }

  let { serverID, image, onchanged }: Props = $props()

  const d = t.docker.img

  let detail = $state.raw<ImageDetail | null>(null)
  let error = $state('')
  let removing = $state<ImagePlan | null>(null)
  let busy = $state(false)

  const ref = $derived(imageRef(image))
  const shortID = $derived(image.id.replace(/^sha256:/, '').slice(0, 12))
  const name = $derived(image.dangling ? d.untagged : `${image.repository}:${image.tag}`)
  const largest = $derived(Math.max(1, ...(detail?.layers ?? []).map((l) => l.size)))

  $effect(() => {
    const r = ref
    untrack(() => {
      detail = null
      error = ''
      backend.image(serverID, r).then(
        (x) => r === ref && (detail = x),
        (e: unknown) => r === ref && (error = toUserError(e).message),
      )
    })
  })

  async function run() {
    try {
      const p = await backend.planImage(ref)
      openTerminalAt(serverID, undefined, p.run)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function askRemove() {
    try {
      removing = await backend.planImage(ref)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function remove(p: ImagePlan) {
    removing = null
    busy = true
    try {
      await backend.removeImage(serverID, p.ref, p.command)
      onchanged()
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }

  const quote = (args?: string[]) => (args?.length ? JSON.stringify(args) : '—')
</script>

<div class="inspector">
  <header>
    <div class="title">
      <h3 title={name}>{name}</h3>
      <p class="muted mono">{shortID}</p>
    </div>
    <dl class="facts">
      <div>
        <dt>{d.created}</dt>
        <dd>{image.createdSince}</dd>
      </div>
      <div>
        <dt>{d.size}</dt>
        <dd>{formatBytes(image.size)}</dd>
      </div>
      <div>
        <dt>{d.used}</dt>
        <dd>{image.containers > 0 ? d.usedBy(image.containers) : d.unused}</dd>
      </div>
    </dl>
  </header>

  <div class="actions">
    <button class="btn btn-primary" onclick={() => void run()}>
      <Icon name="play" size={12} />
      {d.run}
    </button>
    <button class="btn remove" disabled={busy} onclick={() => void askRemove()}>
      <Icon name="trash" size={12} />
      {d.remove}
    </button>
  </div>
  {#if error}<p class="error">{error}</p>{/if}

  <section>
    <h4>{d.layers(detail?.layers.length ?? 0)}</h4>
    {#if !detail}
      {#if !error}<p class="muted">{d.loading}</p>{/if}
    {:else}
      <ol class="layers">
        {#each detail.layers as l, i (i)}
          <li class:empty={l.size === 0}>
            <span class="idx">{i}</span>
            <code title={l.createdBy}>{l.createdBy || l.comment || '—'}</code>
            <span class="bar"><span style:width="{(l.size / largest) * 100}%"></span></span>
            <span class="size">{l.size ? formatBytes(l.size) : '0 B'}</span>
          </li>
        {/each}
      </ol>
    {/if}
  </section>

  {#if detail}
    <section>
      <h4>{d.config}</h4>
      <dl class="config">
        <dt>{d.platform}</dt>
        <dd>{detail.os}/{detail.architecture}</dd>
        <dt>Entrypoint</dt>
        <dd class="mono">{quote(detail.entrypoint)}</dd>
        <dt>Cmd</dt>
        <dd class="mono">{quote(detail.cmd)}</dd>
        <dt>{d.workdir}</dt>
        <dd class="mono">{detail.workingDir || '/'}</dd>
        <dt>User</dt>
        <dd class="mono">{detail.user || 'root'}</dd>
        <dt>{d.ports}</dt>
        <dd class="mono">{detail.exposedPorts?.join(', ') || '—'}</dd>
      </dl>
      {#if detail.env?.length}
        <details>
          <summary>{d.env(detail.env.length)}</summary>
          <pre>{detail.env.join('\n')}</pre>
        </details>
      {/if}
    </section>
  {/if}
</div>

<ConfirmCommand
  command={removing?.command ?? null}
  title={d.removeTitle(name)}
  body={image.containers > 0 ? d.removeInUse(image.containers) : d.removeBody}
  action={d.remove}
  onclose={() => (removing = null)}
  onconfirm={() => removing && void remove(removing)}
/>

<style>
  .inspector {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 16px 18px;
    height: 100%;
    box-sizing: border-box;
    overflow: auto;
  }

  header {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .title p {
    margin: 2px 0 0;
    font-size: var(--text-xs);
  }

  .facts {
    display: flex;
    gap: 24px;
    margin: 0;
  }

  .facts dt,
  h4 {
    font-size: var(--text-xxs);
    font-weight: 600;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  .facts dd {
    margin: 2px 0 0;
    font-size: var(--text-sm);
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .remove {
    color: var(--danger);
  }

  h4 {
    margin: 0 0 6px;
  }

  .layers {
    list-style: none;
    margin: 0;
    padding: 0;
    border-radius: 10px;
    background: var(--fill-soft);
    overflow: hidden;
  }

  .layers li {
    display: grid;
    grid-template-columns: 24px 1fr 60px 70px;
    align-items: center;
    gap: 10px;
    padding: 6px 10px;
    font-size: var(--text-sm);
  }

  .layers li + li {
    border-top: 1px solid var(--divider-soft);
  }

  .layers li.empty {
    color: var(--fg-muted);
  }

  .idx {
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bar {
    height: 5px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
  }

  .size {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .config {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  .config dt {
    color: var(--fg-muted);
  }

  .config dd {
    margin: 0;
    word-break: break-all;
  }

  summary {
    margin-top: 8px;
    font-size: var(--text-sm);
    cursor: default;
  }

  pre {
    margin: 6px 0 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
  }

  .muted {
    color: var(--fg-muted);
  }

  .error {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }
</style>
