<script lang="ts">
  import { type Image, type ImagePlan, imageRef } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import ActionMenu from '../../lib/ui/ActionMenu.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { ago } from './containers'

  interface Props {
    serverID: string
    images: readonly Image[]
    /** Every image, for the space summary whatever the filter. */
    all: readonly Image[]
    onopen: (ref: string) => void
    onchanged: () => void
  }

  let { serverID, images, all, onopen, onchanged }: Props = $props()

  const d = t.docker.img
  const tb = t.docker.table

  let removing = $state.raw<{ plan: ImagePlan; image: Image } | null>(null)
  let error = $state('')

  const total = $derived(all.reduce((s, i) => s + i.size, 0))
  const used = $derived(all.filter((i) => i.containers > 0).reduce((s, i) => s + i.size, 0))
  const name = (i: Image) => (i.dangling ? d.untagged : i.repository)

  async function run(i: Image) {
    try {
      const p = await backend.planImage(imageRef(i))
      openTerminalAt(serverID, undefined, p.run)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function askRemove(i: Image) {
    error = ''
    try {
      removing = { plan: await backend.planImage(imageRef(i)), image: i }
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function remove() {
    const r = removing
    removing = null
    if (!r) return
    try {
      await backend.removeImage(serverID, r.plan.ref, r.plan.command)
      onchanged()
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<div class="dk-summary">
  <div class="dk-card">
    <small>{d.space}</small>
    <strong>{formatBytes(used)} / {formatBytes(total)}</strong>
    <div class="dk-meter"><span style:width="{total ? (used / total) * 100 : 0}%"></span></div>
  </div>
  <div class="dk-card">
    <small>{d.countLabel}</small>
    <strong>{all.length}</strong>
    <small>{d.unusedCount(all.filter((i) => i.containers === 0).length)}</small>
  </div>
</div>

{#if error}
  <div class="dk-banner error">
    <Icon name="warning" size={14} /><span class="grow">{error}</span>
  </div>
{/if}

<div class="dk-scroll">
  {#if images.length === 0}
    <p class="dk-empty">{all.length ? d.noMatch : d.none}</p>
  {:else}
    <table class="dk-table">
      <thead>
        <tr>
          <th>{tb.name}</th>
          <th>{d.tag}</th>
          <th>{d.id}</th>
          <th>{d.created}</th>
          <th class="num">{d.size}</th>
          <th class="num">{tb.actions}</th>
        </tr>
      </thead>
      <tbody>
        {#each images as i (i.id + i.tag)}
          <tr onclick={() => onopen(imageRef(i))}>
            <td class="dk-name">
              <span
                class="dk-dot"
                class:ok={i.containers > 0}
                title={i.containers > 0 ? d.usedBy(i.containers) : d.unused}
              ></span>
              <span class:muted={i.dangling}>{name(i)}</span>
            </td>
            <td>{i.dangling ? '–' : i.tag}</td>
            <td class="mono">{i.id.replace(/^sha256:/, '').slice(0, 12)}</td>
            <td class="muted">{ago(i.createdSince)}</td>
            <td class="num">{formatBytes(i.size)}</td>
            <td>
              <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
              <div class="dk-actions" onclick={(e) => e.stopPropagation()}>
                <button class="icon-btn go" aria-label={d.run} onclick={() => void run(i)}
                  ><Icon name="play" size={15} /></button
                >
                <ActionMenu
                  label={tb.more(name(i))}
                  actions={[
                    { label: d.layersOpen, run: () => onopen(imageRef(i)) },
                    { label: d.run, run: () => void run(i) },
                  ]}
                />
                <button
                  class="icon-btn danger"
                  aria-label={d.remove}
                  disabled={i.containers > 0}
                  title={i.containers > 0 ? d.inUse : d.remove}
                  onclick={() => void askRemove(i)}><Icon name="trash" size={14} /></button
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
  command={removing?.plan.command ?? null}
  title={removing ? d.removeTitle(name(removing.image)) : ''}
  body={d.removeBody}
  action={d.remove}
  onclose={() => (removing = null)}
  onconfirm={() => void remove()}
/>
