import type { BackupMode } from '../../lib/stores/settings.svelte'

// Save policy (Settings → Editor). By default backups are on for system
// configuration, where a bad edit can take a service down.
export function backupFor(path: string, mode: BackupMode = 'etc'): boolean {
  if (mode === 'always') return true
  if (mode === 'never') return false
  return path.startsWith('/etc/')
}
