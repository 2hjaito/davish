// Demo directory tree for running the frontend outside Wails (vite dev,
// previews), shaped like the real file bindings.
import type { Entry } from '../../lib/api/files'
import { joinPath } from './listing'

type Node = { dir: true; days: number } | { dir: false; size: number; days: number }

const d = (days: number): Node => ({ dir: true, days })
const f = (size: number, days: number): Node => ({ dir: false, size, days })

const tree: Record<string, Record<string, Node>> = {
  '/': { etc: d(7), home: d(87), opt: d(15), tmp: d(0), usr: d(87), var: d(87) },
  '/usr': { bin: d(3) },
  // Big enough to exercise the virtual list.
  '/usr/bin': Object.fromEntries(
    Array.from({ length: 3000 }, (_, i) => [
      `tool-${i}`,
      f(10_000 + ((i * 7919) % 900_000), i % 90),
    ]),
  ),
  '/home': { dangth: d(2) },
  '/home/dangth': { projects: d(2), '.bashrc': f(3800, 87), '.profile': f(807, 87) },
  '/etc': { nginx: d(7), systemd: d(15), hosts: f(221, 87), fstab: f(657, 87) },
  '/etc/nginx': {
    'conf.d': d(15),
    'sites-available': d(7),
    'sites-enabled': d(7),
    snippets: d(85),
    'nginx.conf': f(2458, 7),
    'mime.types': f(5222, 85),
    proxy_params: f(210, 85),
  },
  '/var': { log: d(1), lib: d(15) },
  '/var/log': {
    journal: d(0),
    nginx: d(0),
    'auth.log': f(186_368, 0),
    syslog: f(1_887_436, 0),
  },
  '/opt': { grafana: d(15), lightrag: d(3) },
}

export const mockHome = '/home/dangth'

export function mockList(dir: string): Entry[] {
  const now = Date.now() / 1000
  return Object.entries(tree[dir] ?? {}).map(([name, n]) => ({
    name,
    path: joinPath(dir, name),
    kind: n.dir ? 'dir' : 'file',
    dir: n.dir,
    size: n.dir ? 4096 : n.size,
    mode: n.dir ? 'drwxr-xr-x' : '-rw-r--r--',
    perm: n.dir ? 0o755 : 0o644,
    modTime: Math.round(now - n.days * 86_400 - 3600),
    uid: 0,
    gid: 0,
  }))
}

export function mockIsDir(path: string): boolean {
  return path in tree
}

const NGINX_CONF = `user www-data;
worker_processes auto;
pid /run/nginx.pid;

events {
    worker_connections 768;
}

http {
    sendfile on;
    tcp_nopush on;
    types_hash_max_size 2048;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

    gzip on;

    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
`

const BASHRC = `# ~/.bashrc: executed by bash for non-login shells.
case $- in
    *i*) ;;
      *) return;;
esac

HISTSIZE=1000
export EDITOR=vim
alias ll='ls -alF'
if [ -f ~/.bash_aliases ]; then
    . ~/.bash_aliases
fi
`

/** File content for the browser preview; real files come over SFTP. */
export function mockRead(path: string): string {
  if (path.endsWith('nginx.conf')) return NGINX_CONF
  if (path.endsWith('.bashrc')) return BASHRC
  return `# ${path}\n# Nội dung mô phỏng cho bản xem thử trên trình duyệt.\n`
}
