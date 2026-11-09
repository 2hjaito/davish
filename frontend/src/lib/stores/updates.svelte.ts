// Davish's own updates (docs/adr/0009-self-update.md): checked at start and
// every few hours while allowed, or by hand from the menu.
import { toUserError } from '../api/errors'
import { hasRuntime } from '../api/runtime'
import {
  onUpdateProgress,
  updateCheck,
  updateInstall,
  updateRestart,
  type UpdateStatus,
} from '../api/update'
import { t } from '../i18n'
import { notifications } from './notifications.svelte'
import { load, save } from './persist'
import { settings } from './settings.svelte'

const EVERY_MS = 6 * 60 * 60 * 1000

type Phase = 'idle' | 'checking' | 'downloading' | 'ready'

class Updates {
  status = $state.raw<UpdateStatus | null>(null)
  phase = $state<Phase>('idle')
  progress = $state(0)
  error = $state('')
  /** What failed, for the dialog's details: the error code and the backend's text. */
  errorDetail = $state('')
  dialog = $state(false)
  /** The version "Bỏ qua bản này" was pressed for. */
  skipped = $state(load('update:skip', ''))

  /** Checks now and then every few hours; returns the stop function. */
  init(): () => void {
    if (!hasRuntime) return () => {}
    const off = onUpdateProgress((p) => (this.progress = p.total > 0 ? p.done / p.total : 0))
    const auto = () => settings.autoUpdate && void this.check(false)
    const first = setTimeout(auto, 5_000)
    const timer = setInterval(auto, EVERY_MS)
    return () => {
      off()
      clearTimeout(first)
      clearInterval(timer)
    }
  }

  /**
   * Asks GitHub. By hand, the dialog opens whatever the answer; on its own,
   * only a version not skipped makes a notification.
   */
  async check(manual: boolean) {
    if (this.phase !== 'idle') {
      if (manual) this.dialog = true
      return
    }
    this.phase = 'checking'
    this.error = ''
    this.errorDetail = ''
    if (manual) this.dialog = true
    try {
      this.status = await updateCheck()
    } catch (e) {
      this.fail(e)
    } finally {
      this.phase = 'idle'
    }
    const latest = this.status?.latest
    if (!manual && this.status?.available && latest && latest.version !== this.skipped) {
      notifications.add({
        id: `update-${latest.version}`,
        level: 'info',
        title: t.update.available(latest.version),
        serverID: '',
        body: t.update.availableBody,
        at: Date.now(),
        action: { label: t.update.show, run: () => (this.dialog = true) },
      })
    }
  }

  async install() {
    const v = this.status?.latest?.version
    if (!v || this.phase !== 'idle') return
    this.phase = 'downloading'
    this.progress = 0
    this.error = ''
    this.errorDetail = ''
    try {
      await updateInstall(v)
      this.phase = 'ready'
    } catch (e) {
      this.fail(e)
      this.phase = 'idle'
    }
  }

  async restart() {
    try {
      await updateRestart()
    } catch (e) {
      this.fail(e)
    }
  }

  private fail(e: unknown) {
    const err = toUserError(e)
    this.error = err.message
    this.errorDetail = `${err.code}: ${err.detail}`
  }

  skip() {
    const v = this.status?.latest?.version
    if (v) {
      this.skipped = v
      save('update:skip', v)
      notifications.dismiss(`update-${v}`)
    }
    this.dialog = false
  }
}

export const updates = new Updates()
