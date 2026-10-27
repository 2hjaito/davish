import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { ConnState } from './servers'

// Event names emitted by Go. Keep in sync with internal/events/events.go.
export const Events = {
  ConnStatus: 'conn:status',
  ConnHostKeyPrompt: 'conn:hostkey-prompt',
  TermData: 'term:data',
  TermExit: 'term:exit',
  SysmonTick: 'sysmon:tick',
  LogsLine: 'logs:line',
  LogsEnd: 'logs:end',
  TransferProgress: 'transfer:progress',
  TransferDone: 'transfer:done',
  NotifyNew: 'notify:new',
  AuditEntry: 'audit:entry',
} as const

export type EventName = (typeof Events)[keyof typeof Events]

export type Unsubscribe = () => void

export interface ConnStatus {
  serverID: string
  state: ConnState
  error?: { code: string; message: string }
}

export interface HostKeyPrompt {
  serverID: string
  host: string
  keyType: string
  fingerprint: string
}

export interface TermData {
  id: string
  b64: string
}

export interface TermExit {
  id: string
  code: number
}

/** Bytes for memory, bytes per second for network, percent for cpu and disk. */
export interface SysmonTick {
  serverID: string
  cpu: number
  memUsed: number
  memTotal: number
  diskPct: number
  rxBps: number
  txBps: number
  load1: number
  uptime: number
}

function on<T>(name: EventName, handler: (payload: T) => void): Unsubscribe {
  return EventsOn(name, (payload: T) => handler(payload))
}

export const onConnStatus = (h: (p: ConnStatus) => void) => on(Events.ConnStatus, h)
export const onHostKeyPrompt = (h: (p: HostKeyPrompt) => void) => on(Events.ConnHostKeyPrompt, h)
export const onTermData = (h: (p: TermData) => void) => on(Events.TermData, h)
export const onTermExit = (h: (p: TermExit) => void) => on(Events.TermExit, h)
export const onSysmonTick = (h: (p: SysmonTick) => void) => on(Events.SysmonTick, h)
