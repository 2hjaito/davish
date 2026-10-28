import {
  onTransferDone,
  onTransferProgress,
  type TransferDone,
  type TransferProgress,
} from '../api/events'
import { toUserError } from '../api/errors'
import { hasRuntime } from '../api/runtime'
import type { Job } from '../api/transfer'
import { t } from '../i18n/vi'
import type { Notification } from './notifications.svelte'

export type TransferState = 'running' | 'done' | 'failed' | 'canceled'

export interface Transfer extends Job {
  done: number
  /** 0 until a folder has been walked. */
  total: number
  bps: number
  state: TransferState
  /** User-facing reason, when failed. */
  error?: string
}

/** Where a finished upload or copy landed: Files reloads that folder. */
export function landingKey(serverID: string, dir: string): string {
  return `${serverID}\n${dir}`
}

function parentOf(path: string): string {
  const i = path.lastIndexOf('/')
  return i <= 0 ? '/' : path.slice(0, i)
}

export interface Summary {
  count: number
  /** 0 to 1, over the jobs whose size is known. */
  fraction: number
}

/** Running jobs of a server, with their combined progress. */
export function summarize(list: readonly Transfer[], serverID: string): Summary {
  let count = 0
  let done = 0
  let total = 0
  for (const j of list) {
    if (j.serverID !== serverID || j.state !== 'running') continue
    count++
    done += j.done
    total += j.total
  }
  return { count, fraction: total > 0 ? Math.min(1, done / total) : 0 }
}

/** What a transfer notification's button does; the shell supplies it. */
export interface TransferActions {
  /** Opens the Transfers window. */
  showProgress: (j: Transfer) => void
  /** Shows a finished download in the system file manager. */
  reveal: (j: Transfer) => void
  /** Opens a finished copy's folder in the destination's Files. */
  showCopy?: (j: Transfer) => void
  /** A server's name, for copies between servers. */
  nameOf?: (serverID: string) => string
}

/**
 * The notification for a job, or null when there is nothing to say. All
 * share one id per job, so "downloading" turns into "downloaded" in place.
 * An upload says nothing while it runs or when it lands: it shows up in
 * its Files window. A cancel was the user's own doing.
 */
export function describeTransfer(
  j: Transfer,
  actions: TransferActions,
  at = Date.now(),
): Notification | null {
  const base = { id: transferNoticeId(j), serverID: j.serverID, at }
  if (j.kind === 'copy') return describeCopy(j, actions, base)
  switch (j.state) {
    case 'failed': {
      const title = j.kind === 'upload' ? t.transfers.uploadFailed : t.transfers.downloadFailed
      return { ...base, level: 'error', title: title(j.name), body: j.error ?? '' }
    }
    case 'done':
      if (j.kind !== 'download') return null
      return {
        ...base,
        level: 'info',
        title: t.transfers.downloaded(j.name),
        body: t.transfers.savedIn,
        action: { label: t.transfers.reveal, run: () => actions.reveal(j) },
      }
    case 'running':
      if (j.kind !== 'download') return null
      return {
        ...base,
        level: 'info',
        title: t.transfers.downloading(j.name),
        body: t.transfers.savingTo,
        action: { label: t.transfers.showProgress, run: () => actions.showProgress(j) },
      }
  }
  return null
}

// A copy between servers can run for minutes and lands out of sight, so
// it says when it starts and how it ends, like a download.
function describeCopy(
  j: Transfer,
  actions: TransferActions,
  base: { id: string; serverID: string; at: number },
): Notification | null {
  const to = actions.nameOf?.(j.serverID) ?? j.serverID
  switch (j.state) {
    case 'failed':
      return { ...base, level: 'error', title: t.transfers.copyFailed(j.name), body: j.error ?? '' }
    case 'done':
      return {
        ...base,
        level: 'info',
        title: t.transfers.copied(j.name),
        body: t.transfers.copiedTo(to, parentOf(j.dst)),
        action: actions.showCopy && {
          label: t.transfers.openFolder,
          run: () => actions.showCopy?.(j),
        },
      }
    case 'running':
      return {
        ...base,
        level: 'info',
        title: t.transfers.copying(j.name),
        body: t.transfers.copyingTo(to),
        action: { label: t.transfers.showProgress, run: () => actions.showProgress(j) },
      }
  }
  return null
}

export function transferNoticeId(j: Pick<Transfer, 'id'>): string {
  return `transfer-${j.id}`
}

// Sample jobs for the browser preview only; the real app starts empty.
const demo: Transfer[] = hasRuntime
  ? []
  : [
      {
        ...{ id: 'd1', kind: 'download', serverID: 'dangth-vps', name: 'syslog', dir: false },
        ...{ src: '/var/log/syslog', dst: '~/Downloads/syslog' },
        ...{ done: 1_887_436, total: 1_887_436, bps: 0, state: 'done' },
      },
      {
        ...{ id: 'd2', kind: 'upload', serverID: 'dangth-vps', name: 'nginx.conf', dir: false },
        ...{ src: '~/nginx.conf', dst: '/etc/nginx/nginx.conf' },
        ...{ done: 0, total: 2458, bps: 0, state: 'failed', error: 'Không có quyền truy cập.' },
      },
      {
        ...{
          id: 'd3',
          kind: 'upload',
          serverID: 'dangth-vps',
          name: 'backup-2026-10.tar.gz',
          dir: false,
        },
        ...{ src: '~/backup-2026-10.tar.gz', dst: '/srv/backups/backup-2026-10.tar.gz' },
        ...{ done: 412_000_000, total: 1_073_741_824, bps: 11_800_000, state: 'running' },
      },
    ]

class Transfers {
  /** Oldest first. */
  list = $state<Transfer[]>(demo)
  /** Bumped per landingKey when an upload finishes there. */
  landed = $state<Record<string, number>>({})

  // Go starts a job before the binding call returns, so a small one can
  // report progress or even finish before add() knows it.
  #early = new Map<string, { progress?: TransferProgress; done?: TransferDone }>()
  #onFinish: (t: Transfer) => void = () => {}
  #onStart: (t: Transfer) => void = () => {}

  /** onStart and onFinish run once per job as it begins and ends, e.g. to notify. */
  init(onStart: (t: Transfer) => void, onFinish: (t: Transfer) => void): () => void {
    if (!hasRuntime) return () => {}
    const offProgress = onTransferProgress((p) => {
      const j = this.#find(p.jobID)
      if (j) this.#progress(j, p)
      else this.#early.set(p.jobID, { ...this.#early.get(p.jobID), progress: p })
    })
    const offDone = onTransferDone((d) => {
      const j = this.#find(d.jobID)
      if (j) this.#finish(j, d)
      else this.#early.set(d.jobID, { ...this.#early.get(d.jobID), done: d })
    })
    this.#onStart = onStart
    this.#onFinish = onFinish
    return () => {
      this.#onStart = () => {}
      this.#onFinish = () => {}
      offProgress()
      offDone()
    }
  }

  add(jobs: Job[]) {
    for (const job of jobs) {
      this.list.push({ ...job, done: 0, total: 0, bps: 0, state: 'running' })
      const j = this.list[this.list.length - 1]
      if (!j) continue
      this.#onStart(j)
      const early = this.#early.get(job.id)
      if (!early) continue
      this.#early.delete(job.id)
      if (early.progress) this.#progress(j, early.progress)
      if (early.done) this.#finish(j, early.done)
    }
  }

  /** Drops jobs that are over, keeping the running ones. */
  clearFinished() {
    this.list = this.list.filter((j) => j.state === 'running')
  }

  #find(id: string): Transfer | undefined {
    return this.list.find((j) => j.id === id)
  }

  #progress(j: Transfer, p: TransferProgress) {
    j.done = p.done
    j.total = p.total
    j.bps = p.bps
  }

  #finish(j: Transfer, d: TransferDone) {
    if (j.state !== 'running') return
    j.bps = 0
    if (d.canceled) j.state = 'canceled'
    else if (d.error) {
      j.state = 'failed'
      j.error = toUserError(d.error).message
    } else {
      j.state = 'done'
      if (j.kind === 'upload' || j.kind === 'copy') {
        const key = landingKey(j.serverID, parentOf(j.dst))
        this.landed[key] = (this.landed[key] ?? 0) + 1
      }
    }
    this.#onFinish(j)
  }
}

export const transfers = new Transfers()
