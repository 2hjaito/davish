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
