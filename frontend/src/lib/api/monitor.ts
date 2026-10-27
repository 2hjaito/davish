// Typed wrappers for the monitor bindings (internal/app/monitor.go).
import * as App from '../../../wailsjs/go/app/App'

/** Starts sysmon:tick for the server; pair every call with monitorUnwatch. */
export const monitorWatch = (serverID: string): Promise<void> => App.MonitorWatch(serverID)
export const monitorUnwatch = (serverID: string): Promise<void> => App.MonitorUnwatch(serverID)
