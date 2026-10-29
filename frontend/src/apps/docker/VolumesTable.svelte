<script lang="ts">
  import type { Volume, VolumePlan } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import ActionMenu from '../../lib/ui/ActionMenu.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openFilesAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    volumes: readonly Volume[]
    all: readonly Volume[]
    onchanged: () => void
  }

  let { serverID, volumes, all, onchanged }: Props = $props()

  const v = t.docker.vol
  const tb = t.docker.table

  let removing = $state.raw<VolumePlan | null>(null)
  let error = $state('')

  const total = $derived(all.reduce((s, x) => s + Math.max(0, x.size), 0))
  const unused = $derived(all.filter((x) => x.containers === 0))
  const unusedSize = $derived(unused.reduce((s, x) => s + Math.max(0, x.size), 0))

  function created(iso: string): string {
    const ms = Date.parse(iso)
    return Number.isNaN(ms) ? '–' : new Date(ms).toLocaleDateString('vi-VN')
  }

  async function askRemove(x: Volume) {
    error = ''
    try {
      removing = await backend.planVolume(x.name)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function remove() {
    const p = removing
    removing = null
    if (!p) return
    try {
      await backend.removeVolume(serverID, p.name, p.command)
      onchanged()
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<div class="dk-summary">
  <div class="dk-card">
    <small>{v.space}</small>
    <strong>{formatBytes(total)}</strong>
    <small>{v.count(all.length)}</small>
  </div>
  <div class="dk-card">
    <small>{v.unusedLabel}</small>
    <strong>{formatBytes(unusedSize)}</strong>
    <small>{v.count(unused.length)}</small>
  </div>
</div>

{#if error}
  <div class="dk-banner error">
    <Icon name="warning" size={14} /><span class="grow">{error}</span>
  </div>
{/if}

<div class="dk-scroll">
  {#if volumes.length === 0}
    <p class="dk-empty">{all.length ? v.noMatch : v.none}</p>
  {:else}
    <table class="dk-table">
      <thead>
        <tr>
          <th>{tb.name}</th>
          <th>{v.project}</th>
          <th>{v.created}</th>
          <th class="num">{v.size}</th>
          <th class="num">{tb.actions}</th>
        </tr>
      </thead>
      <tbody>
        {#each volumes as x (x.name)}
          <tr>
            <td class="dk-name" title={x.name}>
              <span
                class="dk-dot"
                class:ok={x.containers > 0}
                title={x.containers > 0 ? v.usedBy(x.containers) : v.unused}
              ></span>
              {#if x.anonymous}
                <span class="mono">{x.name.slice(0, 12)}…</span><span class="dk-sub"
                  >{v.anonymous}</span
                >
              {:else}
                <span>{x.name}</span>
              {/if}
            </td>
            <td class="muted">{x.project ?? '–'}</td>
            <td class="muted">{created(x.created)}</td>
            <td class="num">{x.size < 0 ? '–' : formatBytes(x.size)}</td>
            <td>
              <div class="dk-actions">
                <ActionMenu
                  label={tb.more(x.name)}
                  actions={[{ label: v.openFiles, run: () => openFilesAt(serverID, x.mountpoint) }]}
                />
                <button
                  class="icon-btn danger"
                  aria-label={v.remove}
                  disabled={x.containers > 0}
                  title={x.containers > 0 ? v.inUse : v.remove}
                  onclick={() => void askRemove(x)}><Icon name="trash" size={14} /></button
                >
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<ConfirmCommand
  command={removing?.command ?? null}
  title={removing ? v.removeTitle(removing.name) : ''}
  body={v.removeBody}
  action={v.remove}
  onclose={() => (removing = null)}
  onconfirm={() => void remove()}
/>
