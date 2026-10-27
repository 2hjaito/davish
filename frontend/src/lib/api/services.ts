// Typed wrappers for the service bindings (internal/app/services.go).
import * as App from '../../../wailsjs/go/app/App'

export type ServiceAction = 'start' | 'stop' | 'restart'

/** One row of systemctl list-units. */
export interface Unit {
  name: string
  /** loaded, not-found, masked, error */
  load: string
  /** active, inactive, failed, activating, deactivating, reloading */
  active: string
  /** running, exited, dead, failed, auto-restart… */
  sub: string
  description: string
}

export interface UnitStatus {
  unit: string
  description: string
  load: string
  active: string
  sub: string
  /** Unit file state: enabled, disabled, static, masked… */
  enabled: string
  /** Seconds the unit has been active; 0 when it is not. */
  activeFor: number
  mainPID: number
  /** Bytes, -1 when not tracked. */
  memory: number
  restarts: number
  /** success, exit-code, signal, timeout… */
  result: string
  /** The unit file, to open in the Editor. */
  file: string
}

export interface Journal {
  lines: string[]
  /** The user may not read system logs; lines is empty for that reason. */
  restricted: boolean
  /** The same read as root, to prefill in a terminal. */
  sudo: string
}

export interface ActionPlan {
  unit: string
  action: ServiceAction
  /** Exactly what servicesAct runs; show it before stop and restart. */
  command: string
  /** Stop and restart interrupt the service: confirm them first. */
  confirm: boolean
  /** The command with sudo, for a terminal when the server needs root. */
  sudo: string
}

export const servicesList = (serverID: string): Promise<Unit[]> => App.ServicesList(serverID)

export const servicesStatus = (serverID: string, unit: string): Promise<UnitStatus> =>
  App.ServicesStatus(serverID, unit)

export const servicesJournal = (serverID: string, unit: string): Promise<Journal> =>
  App.ServicesJournal(serverID, unit)

export const servicesPlan = (unit: string, action: ServiceAction): Promise<ActionPlan> =>
  App.ServicesPlan(unit, action) as Promise<ActionPlan>

export const servicesAct = (serverID: string, unit: string, action: ServiceAction): Promise<void> =>
  App.ServicesAct(serverID, unit, action)
