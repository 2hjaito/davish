<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    open: boolean
    title: string
    onclose: () => void
    width?: number
    children: Snippet
    footer?: Snippet
  }

  let { open, title, onclose, width = 420, children, footer }: Props = $props()

  let el = $state<HTMLDialogElement>()

  // The native modal dialog traps focus, sits above every window and closes
  // on Esc, so none of that is reimplemented here.
  $effect(() => {
    if (!el) return
    if (open && !el.open) el.showModal()
    if (!open && el.open) el.close()
  })
</script>

<dialog
  bind:this={el}
  class="dialog glass glass-strong"
  style:width="{width}px"
  aria-label={title}
  oncancel={(e) => {
    e.preventDefault()
    onclose()
  }}
>
  {#if open}
    <h2>{title}</h2>
    <div class="body">{@render children()}</div>
    {#if footer}<div class="footer">{@render footer()}</div>{/if}
  {/if}
</dialog>

<style>
  .dialog {
    padding: 18px 20px 16px;
    border-radius: var(--menu-radius);
    color: var(--fg);
    max-height: calc(100vh - 80px);
    overflow-y: auto;
  }

  .dialog[open] {
    animation: pop var(--dur) var(--ease-out);
  }

  .dialog::backdrop {
    background: var(--overlay);
  }

  @keyframes pop {
    from {
      opacity: 0;
      transform: scale(0.96);
    }
  }

  h2 {
    margin: 0 0 12px;
    font-size: var(--text-lg);
    font-weight: 600;
  }

  .footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 18px;
  }
</style>
