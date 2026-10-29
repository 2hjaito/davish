// The nginx calls, or demo data in the plain browser preview.
import {
  nginxCreate,
  nginxEdit,
  nginxList,
  nginxPlanCreate,
  nginxPlanEdit,
  nginxPlanSSL,
  nginxPlanRemove,
  nginxPlanToggle,
  nginxRemove,
  nginxReload,
  nginxReloadCommand,
  nginxRenderSite,
  nginxSiteOptions,
  nginxTest,
  nginxToggle,
  type CreatePlan,
  type EditPlan,
  type NewSite,
  type RemovePlan,
  type SiteEdit,
  type TogglePlan,
} from '../../lib/api/nginx'
import { hasRuntime } from '../../lib/api/runtime'
import {
  mockCreate,
  mockEdit,
  mockList,
  mockPlanCreate,
  mockPlanEdit,
  mockPlanSSL,
  mockPlanRemove,
  mockPlanToggle,
  mockReload,
  mockRemove,
  mockRenderSite,
  mockSiteOptions,
  mockTest,
  mockToggle,
} from './mock'

export const backend = hasRuntime
  ? {
      list: nginxList,
      planToggle: nginxPlanToggle,
      toggle: nginxToggle,
      test: nginxTest,
      reloadCommand: nginxReloadCommand,
      reload: nginxReload,
      planRemove: nginxPlanRemove,
      remove: nginxRemove,
      siteOptions: nginxSiteOptions,
      renderSite: nginxRenderSite,
      planCreate: nginxPlanCreate,
      create: nginxCreate,
      planSSL: nginxPlanSSL,
      planEdit: nginxPlanEdit,
      edit: nginxEdit,
    }
  : {
      list: (_serverID: string) => mockList(),
      planToggle: (_serverID: string, name: string, enable: boolean) =>
        mockPlanToggle(name, enable),
      toggle: (_serverID: string, plan: TogglePlan) => mockToggle(plan),
      test: (_serverID: string) => mockTest(),
      reloadCommand: () => Promise.resolve('nginx -t && systemctl reload nginx'),
      reload: (_serverID: string) => mockReload(),
      planRemove: (_serverID: string, path: string, withCerts: boolean) =>
        mockPlanRemove(path, withCerts),
      remove: (_serverID: string, plan: RemovePlan) => mockRemove(plan),
      siteOptions: (_serverID: string) => mockSiteOptions(),
      renderSite: mockRenderSite,
      planCreate: (_serverID: string, site: NewSite, config: string) =>
        mockPlanCreate(site, config),
      create: (_serverID: string, _site: NewSite, plan: CreatePlan) => mockCreate(plan),
      planSSL: (_serverID: string, names: string[], redirect: boolean) =>
        mockPlanSSL(names, redirect),
      planEdit: (_serverID: string, path: string, edit: SiteEdit) => mockPlanEdit(path, edit),
      edit: (_serverID: string, _edit: SiteEdit, plan: EditPlan) => mockEdit(plan),
    }
