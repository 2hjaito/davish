// Save policy. Until Settings has its own switch (M10), backups are on for
// system configuration, where a bad edit can take a service down.
export function backupByDefault(path: string): boolean {
  return path.startsWith('/etc/')
}
