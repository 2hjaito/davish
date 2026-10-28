// The git calls, or demo data in the plain browser preview.
import { gitDiff, gitFind, gitLog, gitStatus } from '../../lib/api/git'
import { hasRuntime } from '../../lib/api/runtime'
import { mockDiff, mockFind, mockLog, mockStatus } from './mock'

export const backend = hasRuntime
  ? { find: gitFind, status: gitStatus, log: gitLog, diff: gitDiff }
  : {
      find: (_serverID: string, extra: string[]) => mockFind(extra),
      status: (_serverID: string, dir: string) => mockStatus(dir),
      log: (_serverID: string, dir: string, skip: number) => mockLog(dir, skip),
      diff: mockDiff,
    }
