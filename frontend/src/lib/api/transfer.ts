// Typed wrappers for the transfer bindings (internal/app/transfer.go).
import * as App from '../../../wailsjs/go/app/App'
import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime'
import { hasRuntime } from './runtime'

export interface Job {
  id: string
  kind: 'upload' | 'download'
  serverID: string
  /** The final name; differs from the source when both were kept. */
  name: string
  src: string
  dst: string
  dir: boolean
}

export interface UploadItem {
  local: string
  name: string
  dir: boolean
  /** 0 for folders. */
  size: number
  /** The target folder already has this name. */
  exists: boolean
  existsDir: boolean
  /** What replacing runs, for the confirmation dialog; absent when replacing is not offered. */
  replaceCommand?: string
}

export interface UploadRequest {
  local: string
  /** Without it, a taken name uploads as "name (1)". */
  replace: boolean
}

/** Empty when the user canceled. */
export const transferPickFiles = async (title: string): Promise<string[]> =>
  (await App.TransferPickFiles(title)) ?? []
export const transferPickFolder = (title: string): Promise<string> => App.TransferPickFolder(title)

export const transferPlanUpload = (
  serverID: string,
  dir: string,
  locals: string[],
): Promise<UploadItem[]> => App.TransferPlanUpload(serverID, dir, locals) as Promise<UploadItem[]>

export const transferUpload = (
  serverID: string,
  dir: string,
  reqs: UploadRequest[],
): Promise<Job[]> => App.TransferUpload(serverID, dir, reqs) as Promise<Job[]>

/** Saves into the user's Downloads folder, never replacing a file there. */
export const transferDownload = (serverID: string, paths: string[]): Promise<Job[]> =>
  App.TransferDownload(serverID, paths) as Promise<Job[]>

export const transferCancel = (jobID: string): Promise<void> => App.TransferCancel(jobID)

type DropHandler = (paths: string[]) => void

// Wails reports OS file drops once for the whole webview; each Files
// window registers its element and the drop goes to the one under the
// pointer. Elements need the CSS property --wails-drop-target: drop.
const targets = new Map<Element, DropHandler>()

/** Calls handler with the local paths dropped onto el. */
export function registerDropTarget(el: Element, handler: DropHandler): () => void {
  if (!hasRuntime) return () => {}
  if (targets.size === 0) {
    OnFileDrop((x, y, paths) => {
      const hit = document.elementFromPoint(x, y)
      for (const [target, h] of targets) {
        if (hit && target.contains(hit)) return h(paths)
      }
    }, true)
  }
  targets.set(el, handler)
  return () => {
    targets.delete(el)
    if (targets.size === 0) OnFileDropOff()
  }
}
