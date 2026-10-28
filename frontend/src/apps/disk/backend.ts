// The disk calls, or demo data in the plain browser preview.
import { diskCancel, diskClean, diskCleanups, diskMounts, diskUsage } from '../../lib/api/disk'
import { hasRuntime } from '../../lib/api/runtime'
import { mockCancel, mockClean, mockCleanups, mockMounts, mockUsage } from './mock'

export const backend = hasRuntime
  ? {
      mounts: diskMounts,
      usage: diskUsage,
      cancel: diskCancel,
      cleanups: diskCleanups,
      clean: diskClean,
    }
  : {
      mounts: (_serverID: string) => mockMounts(),
      usage: (_serverID: string, dir: string, token: string) => mockUsage(dir, token),
      cancel: mockCancel,
      cleanups: (_serverID: string) => mockCleanups(),
      clean: (_serverID: string, id: string, _command: string) => mockClean(id),
    }
