// The cron calls, or demo data in the plain browser preview.
import { cronApply, cronList, cronPlan, cronPreview, cronRuns } from '../../lib/api/cron'
import type { CronChange, CronPlan } from '../../lib/api/cron'
import { filesHome, filesStat } from '../../lib/api/files'
import { hasRuntime } from '../../lib/api/runtime'
import { mockApply, mockList, mockPlan, mockPreview, mockRuns } from './mock'

export const backend = hasRuntime
  ? {
      list: cronList,
      plan: cronPlan,
      apply: cronApply,
      preview: cronPreview,
      runs: cronRuns,
      home: filesHome,
      isDir: async (serverID: string, path: string) => (await filesStat(serverID, path)).dir,
    }
  : {
      list: (_serverID: string) => mockList(),
      plan: (_serverID: string, c: CronChange) => mockPlan(c),
      apply: (_serverID: string, c: CronChange, _plan: CronPlan) => mockApply(c),
      preview: (schedule: string, _zone: string) => mockPreview(schedule),
      runs: (_serverID: string, command: string) => mockRuns(command),
      home: (_serverID: string) => Promise.resolve('/home/dangth'),
      isDir: (_serverID: string, path: string) => Promise.resolve(!path.includes('.')),
    }
