// The SSH key calls, or demo data in the plain browser preview.
import {
  type GitHost,
  type KeyPlan,
  keysAdd,
  keysAuthorized,
  keysCreate,
  keysGenerateLocal,
  keysGitTest,
  keysLocal,
  keysPlanAdd,
  keysPlanCreate,
  keysPlanGitTest,
  keysPlanRemove,
  keysRemove,
  keysServer,
} from '../../lib/api/keys'
import { hasRuntime } from '../../lib/api/runtime'
import * as mock from './mock'

export const backend = hasRuntime
  ? {
      authorized: keysAuthorized,
      planAdd: keysPlanAdd,
      add: keysAdd,
      planRemove: keysPlanRemove,
      remove: keysRemove,
      server: keysServer,
      planCreate: keysPlanCreate,
      create: keysCreate,
      planGitTest: keysPlanGitTest,
      gitTest: keysGitTest,
      local: keysLocal,
      generateLocal: keysGenerateLocal,
    }
  : {
      authorized: (_serverID: string) => mock.mockAuthorized(),
      planAdd: (_serverID: string, text: string) => mock.mockPlanAdd(text),
      add: (_serverID: string, text: string, _command: string) => mock.mockAdd(text),
      planRemove: (_serverID: string, fingerprint: string) => mock.mockPlanRemove(fingerprint),
      remove: (_serverID: string, plan: KeyPlan) => mock.mockRemove(plan),
      server: (_serverID: string) => mock.mockServerKeys(),
      planCreate: (_serverID: string, name: string, comment: string) =>
        mock.mockPlanCreate(name, comment),
      create: (_serverID: string, name: string, comment: string, _command: string) =>
        mock.mockCreate(name, comment),
      planGitTest: (_serverID: string, host: GitHost, name: string) =>
        mock.mockPlanGitTest(host, name),
      gitTest: (_serverID: string, host: GitHost, name: string, _command: string) =>
        mock.mockGitTest(host, name),
      local: () => mock.mockLocal(),
      generateLocal: mock.mockGenerateLocal,
    }
