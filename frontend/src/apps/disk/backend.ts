// The disk calls, or demo data in the plain browser preview.
import {
  diskCancel,
  diskClean,
  diskCleanups,
  diskMounts,
  diskOverview,
  diskPlanRemove,
  diskPlanTruncate,
  diskRemove,
  diskTruncate,
  diskUsage,
} from '../../lib/api/disk'
import { hasRuntime } from '../../lib/api/runtime'
import {
  mockCancel,
  mockClean,
  mockCleanups,
  mockMounts,
  mockOverview,
  mockPlanRemove,
  mockPlanTruncate,
  mockRemove,
  mockUsage,
} from './mock'

export const backend = hasRuntime
  ? {
      mounts: diskMounts,
      usage: diskUsage,
      overview: diskOverview,
      cancel: diskCancel,
      cleanups: diskCleanups,
      clean: diskClean,
      planRemove: diskPlanRemove,
      remove: diskRemove,
      planTruncate: diskPlanTruncate,
      truncate: diskTruncate,
    }
  : {
      mounts: (_serverID: string) => mockMounts(),
      usage: (_serverID: string, dir: string, token: string) => mockUsage(dir, token),
      overview: (_serverID: string, mount: string, token: string) => mockOverview(mount, token),
      cancel: mockCancel,
      cleanups: (_serverID: string) => mockCleanups(),
      clean: (_serverID: string, id: string, _command: string) => mockClean(id),
      planRemove: (_serverID: string, path: string) => mockPlanRemove(path),
      remove: (_serverID: string, path: string, _command: string) => mockRemove(path),
      planTruncate: mockPlanTruncate,
      truncate: (_serverID: string, _path: string, _command: string) =>
        new Promise<void>((r) => setTimeout(r, 400)),
    }
