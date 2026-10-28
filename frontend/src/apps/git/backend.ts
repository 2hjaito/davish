// The git calls, or demo data in the plain browser preview.
import {
  gitBranches,
  gitDiff,
  gitFind,
  gitLog,
  gitPlan,
  gitRun,
  gitStatus,
} from '../../lib/api/git'
import { hasRuntime } from '../../lib/api/runtime'
import { mockBranches, mockDiff, mockFind, mockLog, mockPlan, mockRun, mockStatus } from './mock'

export const backend = hasRuntime
  ? {
      find: gitFind,
      status: gitStatus,
      log: gitLog,
      diff: gitDiff,
      branches: gitBranches,
      plan: gitPlan,
      run: gitRun,
    }
  : {
      find: (_serverID: string, extra: string[]) => mockFind(extra),
      status: (_serverID: string, dir: string) => mockStatus(dir),
      log: (_serverID: string, dir: string, skip: number) => mockLog(dir, skip),
      diff: mockDiff,
      branches: (_serverID: string, _dir: string) => mockBranches(),
      plan: mockPlan,
      run: mockRun,
    }
