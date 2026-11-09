// The security call, or demo data in the plain browser preview.
import { hasRuntime } from '../../lib/api/runtime'
import { securityReport, type HourCount, type SecurityReport } from '../../lib/api/security'

// A week of probing that comes in waves, ending 2026-10-28 09:00.
function demoHours(): HourCount[] {
  const end = Date.UTC(2026, 9, 28, 9)
  return Array.from({ length: 168 }, (_, i) => {
    const at = new Date(end - (167 - i) * 3600_000)
    const wave = 16 + Math.sin(i / 5.3) * 12 + Math.sin(i / 1.7) * 5
    return {
      hour: at.toISOString().slice(0, 13),
      count: i % 23 === 7 ? 55 : Math.round(wave),
    }
  })
}

const demo: SecurityReport = {
  root: false,
  source: 'journal',
  accepted: [
    { time: '2026-10-28T09:12:40+0700', user: 'dangth', ip: '113.161.42.7', method: 'publickey' },
    { time: '2026-10-27T21:03:11+0700', user: 'deploy', ip: '113.161.42.7', method: 'publickey' },
  ],
  failed: [
    {
      ip: '45.148.10.92',
      count: 1284,
      users: ['root', 'admin', 'ubuntu'],
      last: '2026-10-28T09:40:02+0700',
    },
    {
      ip: '193.32.162.77',
      count: 640,
      users: ['root', 'admin'],
      last: '2026-10-28T08:59:10+0700',
    },
    { ip: '218.92.0.34', count: 512, users: ['root'], last: '2026-10-28T08:15:44+0700' },
    { ip: '61.177.172.140', count: 208, users: ['root'], last: '2026-10-28T07:02:31+0700' },
    {
      ip: '80.94.95.115',
      count: 96,
      users: ['ubuntu', 'test'],
      last: '2026-10-27T22:40:18+0700',
    },
    {
      ip: '103.89.91.6',
      count: 37,
      users: ['oracle', 'test', 'git'],
      last: '2026-10-27T23:51:09+0700',
    },
  ],
  failedTotal: 1833,
  invalidTotal: 412,
  failedIPs: 61,
  acceptedTotal: 43,
  targets: [
    { user: 'root', count: 1306 },
    { user: 'admin', count: 188 },
    { user: 'ubuntu', count: 142 },
    { user: 'test', count: 78 },
    { user: 'oracle', count: 64 },
    { user: 'postgres', count: 50 },
    { user: 'git', count: 36 },
    { user: 'deploy', count: 22 },
  ],
  hourly: demoHours(),
  sessions: [
    { user: 'dangth', tty: 'pts/0', since: '2026-10-28 09:12', from: '113.161.42.7' },
    { user: 'deploy', tty: 'pts/1', since: '2026-10-28 08:47', from: '113.161.42.7' },
  ],
  last: ['dangth   pts/0        113.161.42.7     Wed Oct 28 09:12   still logged in'],
  lastb: [],
  lastbDenied: true,
  ufw: { installed: true, needsRoot: true, active: false, lines: [] },
  fail2ban: {
    installed: true,
    needsRoot: false,
    active: true,
    lines: [],
    banned: ['45.148.10.92', '193.32.162.77'],
  },
  passwordAuth: 'yes',
  disablePassword: `sudo sh -c 'f=/etc/ssh/sshd_config.d/60-davish-no-password.conf; printf "PasswordAuthentication no\\nKbdInteractiveAuthentication no\\n" > "$f" && sshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd; } || { rm -f "$f"; echo "sshd -t failed: change reverted"; }'`,
}

export const backend = hasRuntime
  ? { report: securityReport }
  : {
      report: (_serverID: string) =>
        new Promise<SecurityReport>((r) => setTimeout(() => r(demo), 300)),
    }
