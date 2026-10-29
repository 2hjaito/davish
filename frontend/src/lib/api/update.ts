// Typed wrappers for the self-update bindings (internal/app/update.go).
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'

export interface Release {
  /** Without the v. */
  version: string
  /** The release notes, Markdown from CHANGELOG.md. */
  notes: string
  /** The release page, for a manual download. */
  url: string
  /** Installable from inside Davish: signed, with an archive for this OS. */
  signed: boolean
  size: number
}

export interface UpdateStatus {
  current: string
  latest?: Release
  available: boolean
}

export interface UpdateProgress {
  done: number
  total: number
}

export const updateCheck = (): Promise<UpdateStatus> => App.UpdateCheck()
export const updateInstall = (version: string): Promise<void> => App.UpdateInstall(version)
export const updateRestart = (): Promise<void> => App.UpdateRestart()

export const onUpdateProgress = (h: (p: UpdateProgress) => void): (() => void) =>
  EventsOn('update:progress', h)
