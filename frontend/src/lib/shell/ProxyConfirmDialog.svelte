<script lang="ts">
  import { toUserError } from '../api/errors'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import ConfirmCommand from '../ui/ConfirmCommand.svelte'

  const prompt = $derived(servers.proxyPrompt)
  let error = $state('')

  async function confirm() {
    error = ''
    try {
      await servers.confirmProxy()
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<!-- A proxy command runs on this computer, so the user sees the exact line
     before the first run, and again if it changes (ADR 0031). -->
<ConfirmCommand
  command={prompt?.command ?? null}
  title={prompt ? t.proxyConfirm.title(prompt.name) : ''}
  body={t.proxyConfirm.body}
  action={t.proxyConfirm.action}
  tone="primary"
  note={error || t.proxyConfirm.note}
  onclose={() => {
    error = ''
    servers.rejectProxy()
  }}
  onconfirm={confirm}
/>
