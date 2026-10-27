// The cron calls, or demo data in the plain browser preview.
import { cronApply, cronList, cronPlan, cronPreview, cronRuns } from '../../lib/api/cron'
import type { CronChange } from '../../lib/api/cron'
import { hasRuntime } from '../../lib/api/runtime'
import { mockApply, mockList, mockPlan, mockPreview, mockRuns } from './mock'

export const backend = hasRuntime
  ? { list: cronList, plan: cronPlan, apply: cronApply, preview: cronPreview, runs: cronRuns }
  : {
      list: (_serverID: string) => mockList(),
      plan: (_serverID: string, c: CronChange) => mockPlan(c),
      apply: (_serverID: string, c: CronChange, _version: string) => mockApply(c),
      preview: (schedule: string, _zone: string) => mockPreview(schedule),
      runs: (_serverID: string, command: string) => mockRuns(command),
    }
