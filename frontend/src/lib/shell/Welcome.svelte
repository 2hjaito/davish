<script lang="ts">
  import { t } from '../i18n/vi'
  import DavishMark from '../ui/DavishMark.svelte'
  import Icon from '../ui/Icon.svelte'
  import type { IconName } from '../ui/icons'
  import { isMac } from './platform'

  interface Props {
    onadd: () => void
    onimport: () => void
  }

  let { onadd, onimport }: Props = $props()

  const points: readonly { icon: IconName; title: string; body: string }[] = t.welcome.points
</script>

<!-- First run: nothing to show until there is a server. -->
<section class="welcome glass glass-strong" aria-labelledby="welcome-title">
  <DavishMark size={84} />
  <h1 id="welcome-title">{t.welcome.title}</h1>
  <p class="lead">{t.welcome.lead}</p>
  <ul>
    {#each points as p (p.title)}
      <li>
        <span class="icon"><Icon name={p.icon} size={16} /></span>
        <span>
          <strong>{p.title}</strong>
          <small>{p.body}</small>
        </span>
      </li>
    {/each}
  </ul>
  <!-- svelte-ignore a11y_autofocus -->
  <button type="button" class="btn btn-primary add" autofocus onclick={onadd}>
    <Icon name="plus" size={14} strokeWidth={2.4} />
    {t.welcome.add}
  </button>
  <button type="button" class="link" onclick={onimport}>{t.welcome.import}</button>
  <p class="foot">{t.welcome.foot(isMac ? '⌘' : 'Ctrl+')}</p>
</section>

<style>
  .welcome {
    position: absolute;
    top: 50%;
    left: 50%;
    translate: -50% -50%;
    width: min(500px, calc(100vw - 40px));
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 32px 36px 22px;
    border-radius: 22px;
    text-align: center;
    animation: rise var(--dur-slow) var(--ease-out);
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.98);
    }
  }

  h1 {
    margin: 10px 0 0;
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.01em;
  }

  .lead {
    margin: 0 0 10px;
    color: var(--fg-muted);
    font-size: var(--text-md);
  }

  ul {
    margin: 4px 0 14px;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 10px;
    text-align: left;
    width: 100%;
  }

  li {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .icon {
    flex-shrink: 0;
    display: flex;
    padding: 7px;
    border-radius: 9px;
    background: var(--accent-soft);
    color: var(--accent);
  }

  li span:last-child {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-sm);
    line-height: 1.35;
  }

  .add {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 18px;
    font-size: var(--text-md);
  }

  .link {
    border: 0;
    background: none;
    padding: 4px 8px;
    font-size: var(--text-sm);
    color: var(--accent);
  }

  .foot {
    margin: 10px 0 0;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }
</style>
