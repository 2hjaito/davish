// The service calls, or demo data in the plain browser preview.
import { hasRuntime } from '../../lib/api/runtime'
import {
  servicesAct,
  servicesJournal,
  servicesList,
  servicesPlan,
  servicesStatus,
} from '../../lib/api/services'
import { mockAct, mockJournal, mockList, mockPlan, mockStatus } from './mock'

export const backend = hasRuntime
  ? {
      list: servicesList,
      status: servicesStatus,
      journal: servicesJournal,
      plan: servicesPlan,
      act: servicesAct,
    }
  : {
      list: (_serverID: string) => mockList(),
      status: (_serverID: string, unit: string) => mockStatus(unit),
      journal: (_serverID: string, unit: string) => mockJournal(unit),
      plan: mockPlan,
      act: (_serverID: string, unit: string, action: Parameters<typeof mockAct>[1]) =>
        mockAct(unit, action),
    }
