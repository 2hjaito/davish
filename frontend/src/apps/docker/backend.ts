// The Docker calls, or demo data in the plain browser preview.
import {
  type ContainerAction,
  dockerAct,
  dockerInspect,
  dockerList,
  dockerLogs,
  dockerPlan,
  dockerStats,
} from '../../lib/api/docker'
import { hasRuntime } from '../../lib/api/runtime'
import { mockAct, mockInspect, mockList, mockLogs, mockPlan, mockStats } from './mock'

export const backend = hasRuntime
  ? {
      list: dockerList,
      stats: dockerStats,
      inspect: dockerInspect,
      logs: dockerLogs,
      plan: dockerPlan,
      act: dockerAct,
    }
  : {
      list: (_serverID: string) => mockList(),
      stats: (_serverID: string) => mockStats(),
      inspect: (_serverID: string, name: string) => mockInspect(name),
      logs: (_serverID: string, name: string) => mockLogs(name),
      plan: mockPlan,
      act: (_serverID: string, name: string, action: ContainerAction) => mockAct(name, action),
    }
