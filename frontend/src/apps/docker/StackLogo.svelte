<script lang="ts">
  // A compose project's logo: its own icon file, else the framework its code
  // uses, else what its images suggest. Shared by the cards and the table so
  // both show the same thing.
  import type { StackIcon } from '../../lib/api/docker'
  import Icon from '../../lib/ui/Icon.svelte'
  import { techIcons } from '../../lib/ui/techIcons'
  import type { Group } from './containers'
  import { knownTech, projectIcon } from './projectIcon'
  import { groupTech } from './tech'

  interface Props {
    serverID: string
    group: Group
    /** Box size in pixels. */
    size?: number
    /** Shown as a letter when nothing better is known; nothing when omitted. */
    letter?: string
  }

  let { serverID, group, size = 34, letter }: Props = $props()

  const dir = $derived(group.project ? group.containers[0]?.projectDir : undefined)
  let found = $state.raw<StackIcon>({})
  $effect(() => {
    const d = dir
    found = {}
    if (!d) return
    let live = true
    void projectIcon(serverID, d).then((r) => {
      if (live) found = r
    })
    return () => {
      live = false
    }
  })
  // The project's own code says more than the images it runs.
  const tech = $derived(knownTech(found.tech) ?? groupTech(group))
  const glyph = $derived(Math.round(size * 0.6))
</script>

{#if found.icon}
  <span class="logo" style:--box="{size}px"
    ><img src={found.icon} alt="" style:width="{glyph + 2}px" style:height="{glyph + 2}px" /></span
  >
{:else if tech}
  {@const l = techIcons[tech]}
  <span class="logo" style:--box="{size}px" style:color={l.color ?? 'var(--fg)'} title={tech}>
    <Icon def={l} size={glyph} />
  </span>
{:else if letter}
  <span class="logo letter" style:--box="{size}px" aria-hidden="true"
    >{letter.slice(0, 1).toUpperCase()}</span
  >
{/if}

<style>
  .logo {
    flex: none;
    display: grid;
    place-items: center;
    width: var(--box);
    height: var(--box);
    border-radius: calc(var(--box) / 4);
    background: var(--chart-surface);
    box-shadow: inset 0 0 0 1px var(--divider-soft);
  }

  .logo img {
    object-fit: contain;
  }

  .letter {
    color: var(--fg-muted);
    font-weight: 600;
    font-size: var(--text-xs);
  }
</style>
