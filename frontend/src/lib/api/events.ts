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

/** Percent for cpu, disk and GPU load; bytes for memory; bytes per second for rates. */
export interface SysmonTick {
  serverID: string
  cpu: number
  memUsed: number
  memTotal: number
  /** Filesystem of /. */
  diskPct: number
  rxBps: number
  txBps: number
  load1: number
  /** Seconds since boot. */
  uptime: number
  load5: number
  load15: number
  procs: number
  cpuSplit: { user: number; system: number; iowait: number; steal: number }
  cores: number[]
  memCache: number
  memFree: number
  swapTotal: number
  swapUsed: number
  readBps: number
  writeBps: number
  disks: { name: string; readBps: number; writeBps: number }[]
  mounts: Mount[]
  ifaces: NetIface[]
  gpus: Gpu[]
}

export interface Mount {
  device: string
  path: string
  size: number
  used: number
  avail: number
  pct: number
}

export interface NetIface {
  name: string
  rxBps: number
  txBps: number
  rxTotal: number
  txTotal: number
}

/** Readings the driver does not report are -1. */
export interface Gpu {
  index: number
  name: string
  util: number
  memUsed: number
  memTotal: number
  /** °C */
  temp: number
  /** Watts */
  power: number
}

function on<T>(name: EventName, handler: (payload: T) => void): Unsubscribe {
  return EventsOn(name, (payload: T) => handler(payload))
}

export const onConnStatus = (h: (p: ConnStatus) => void) => on(Events.ConnStatus, h)
export const onHostKeyPrompt = (h: (p: HostKeyPrompt) => void) => on(Events.ConnHostKeyPrompt, h)
export const onTermData = (h: (p: TermData) => void) => on(Events.TermData, h)
export const onTermExit = (h: (p: TermExit) => void) => on(Events.TermExit, h)
export const onSysmonTick = (h: (p: SysmonTick) => void) => on(Events.SysmonTick, h)
