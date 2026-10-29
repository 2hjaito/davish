// The package calls, or demo data in the plain browser preview.
import {
  packagesApps,
  packagesInventory,
  packagesNpmGlobals,
  packagesPlan,
  packagesRecipes,
} from '../../lib/api/packages'
import { hasRuntime } from '../../lib/api/runtime'
import { mockApps, mockInventory, mockNpmGlobals, mockPlan, mockRecipes } from './mock'

export const backend = hasRuntime
  ? {
      inventory: packagesInventory,
      apps: packagesApps,
      npmGlobals: packagesNpmGlobals,
      plan: packagesPlan,
      recipes: packagesRecipes,
    }
  : {
      inventory: (_serverID: string) => mockInventory(),
      apps: (_serverID: string) => mockApps(),
      npmGlobals: (_serverID: string) => mockNpmGlobals(),
      plan: mockPlan,
      recipes: mockRecipes,
    }
