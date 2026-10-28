// The nginx calls, or demo data in the plain browser preview.
import {
  nginxList,
  nginxPlanToggle,
  nginxReload,
  nginxReloadCommand,
  nginxTest,
  nginxToggle,
  type TogglePlan,
} from '../../lib/api/nginx'
import { hasRuntime } from '../../lib/api/runtime'
import { mockList, mockPlanToggle, mockReload, mockTest, mockToggle } from './mock'

export const backend = hasRuntime
  ? {
      list: nginxList,
      planToggle: nginxPlanToggle,
      toggle: nginxToggle,
      test: nginxTest,
      reloadCommand: nginxReloadCommand,
      reload: nginxReload,
    }
  : {
      list: (_serverID: string) => mockList(),
      planToggle: (_serverID: string, name: string, enable: boolean) =>
        mockPlanToggle(name, enable),
      toggle: (_serverID: string, plan: TogglePlan) => mockToggle(plan),
      test: (_serverID: string) => mockTest(),
      reloadCommand: () => Promise.resolve('nginx -t && systemctl reload nginx'),
      reload: (_serverID: string) => mockReload(),
    }
