// Typed wrappers for running a command on several servers
// (internal/app/fleet.go, .claude/docs/adr/0021).
import * as App from '../../../wailsjs/go/app/App'
import { fleet } from '../../../wailsjs/go/models'

export type JobState = 'queued' | 'running' | 'done' | 'failed' | 'stopped'

/** One server's part of a run (fleet:job). */
export interface FleetJob {
  runID: string
  serverID: string
  state: JobState
  /** The terminal session, once it runs. */
  termID?: string
  /** The exit code once finished; -1 when it never finished. */
  exit: number
  /** An error code when the session could not start. */
  error?: string
}

export interface FleetItem {
  serverID: string
  command: string
}

/** How a run must be confirmed. */
export interface FleetCheck {
  destructive: boolean
  /** The user types the number of servers to confirm. */
  typed: boolean
}

/** The package update run: each server's own command, and the servers left out. */
export interface FleetUpdate {
  items: FleetItem[]
  skipped: { serverID: string; reason: string }[]
}

export const fleetCheck = (command: string, n: number): Promise<FleetCheck> =>
  App.FleetCheck(command, n)

export const fleetPlanUpdate = (serverIDs: string[]): Promise<FleetUpdate> =>
  App.FleetPlanUpdate(serverIDs)

export async function fleetStart(
  items: FleetItem[],
  cols: number,
  rows: number,
): Promise<{ id: string; jobs: FleetJob[] }> {
  const r = await App.FleetStart(
    items.map((i) => fleet.Item.createFrom(i)),
    cols,
    rows,
  )
  return { id: r.id, jobs: r.jobs as FleetJob[] }
}

export const fleetStop = (runID: string): Promise<void> => App.FleetStop(runID)
