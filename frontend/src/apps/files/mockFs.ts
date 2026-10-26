import type { IconName } from '../../lib/ui/icons'

// Mock directory listings until SFTP lands in M5.
export interface Entry {
  name: string
  dir: boolean
  size?: number
  modified: string
}

const d = (name: string, modified: string): Entry => ({ name, dir: true, modified })
const f = (name: string, size: number, modified: string): Entry => ({
  name,
  dir: false,
  size,
  modified,
})

export const mockFs: Record<string, Entry[]> = {
  '/': [
    d('etc', '20/10/2026'),
    d('home', '01/08/2026'),
    d('opt', '12/10/2026'),
    d('var', '01/08/2026'),
  ],
  '/home': [d('dangth', '25/10/2026')],
  '/home/dangth': [
    d('projects', '25/10/2026'),
    f('.bashrc', 3800, '01/08/2026'),
    f('.profile', 807, '01/08/2026'),
  ],
  '/etc': [
    d('nginx', '20/10/2026'),
    d('systemd', '12/10/2026'),
    f('hosts', 221, '01/08/2026'),
    f('fstab', 657, '01/08/2026'),
  ],
  '/etc/nginx': [
    d('conf.d', '12/10/2026'),
    d('sites-available', '20/10/2026'),
    d('sites-enabled', '20/10/2026'),
    d('snippets', '03/08/2026'),
    f('nginx.conf', 2.4 * 1024, '20/10/2026'),
    f('mime.types', 5.1 * 1024, '03/08/2026'),
    f('proxy_params', 210, '03/08/2026'),
  ],
  '/var': [d('log', '26/10/2026'), d('lib', '12/10/2026')],
  '/var/log': [
    d('journal', '26/10/2026'),
    d('nginx', '26/10/2026'),
    f('auth.log', 182 * 1024, '26/10/2026'),
    f('syslog', 1.8 * 1024 ** 2, '26/10/2026'),
  ],
  '/opt': [d('grafana', '12/10/2026'), d('lightrag', '24/10/2026')],
}

export interface Place {
  label: string
  path: string
  icon: IconName
}

export const places: Place[] = [
  { label: 'Home', path: '/home/dangth', icon: 'home' },
  { label: '/etc', path: '/etc', icon: 'folder' },
  { label: '/var/log', path: '/var/log', icon: 'folder' },
  { label: '/opt', path: '/opt', icon: 'folder' },
]

export function join(dir: string, name: string): string {
  return dir === '/' ? `/${name}` : `${dir}/${name}`
}

/** Shortcuts shown for other servers in the sidebar. */
export const remotePlaces: Place[] = [{ label: 'Backups', path: '/srv/backups', icon: 'folder' }]
