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

/** Where a finished upload landed: Files reloads that folder. */
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

/**
 * The notification for a finished job, or null when the result is already
 * in sight: an upload shows up in its Files window, a cancel was the user's.
 */
export function describeTransfer(j: Transfer, at = Date.now()): Notification | null {
  const base = { id: `transfer-${j.id}`, serverID: j.serverID, at }
  if (j.state === 'failed') {
    const title = j.kind === 'upload' ? t.transfers.uploadFailed : t.transfers.downloadFailed
    return { ...base, level: 'error', title: title(j.name), body: j.error ?? '' }
  }
  if (j.state === 'done' && j.kind === 'download') {
    return {
      ...base,
      level: 'info',
      title: t.transfers.downloaded(j.name),
      body: t.transfers.savedIn,
    }
  }
  return null
}

class Transfers {
  /** Oldest first. */
  list = $state<Transfer[]>([])
  /** Bumped per landingKey when an upload finishes there. */
  landed = $state<Record<string, number>>({})

  // Go starts a job before the binding call returns, so a small one can
  // report progress or even finish before add() knows it.
  #early = new Map<string, { progress?: TransferProgress; done?: TransferDone }>()
  #onFinish: (t: Transfer) => void = () => {}

  /** onFinish runs once per job that ends, e.g. to notify. */
  init(onFinish: (t: Transfer) => void): () => void {
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
    this.#onFinish = onFinish
    return () => {
      this.#onFinish = () => {}
      offProgress()
      offDone()
    }
  }

  add(jobs: Job[]) {
    for (const job of jobs) {
      this.list.push({ ...job, done: 0, total: 0, bps: 0, state: 'running' })
      const j = this.list[this.list.length - 1]
      const early = this.#early.get(job.id)
      if (!j || !early) continue
      this.#early.delete(job.id)
      if (early.progress) this.#progress(j, early.progress)
      if (early.done) this.#finish(j, early.done)
    }
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
      if (j.kind === 'upload') {
        const key = landingKey(j.serverID, parentOf(j.dst))
        this.landed[key] = (this.landed[key] ?? 0) + 1
      }
    }
    this.#onFinish(j)
  }
}

export const transfers = new Transfers()
