// The security call, or demo data in the plain browser preview.
import { hasRuntime } from '../../lib/api/runtime'
import { securityReport, type SecurityReport } from '../../lib/api/security'

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
    { ip: '218.92.0.34', count: 512, users: ['root'], last: '2026-10-28T08:15:44+0700' },
    {
      ip: '103.89.91.6',
      count: 37,
      users: ['oracle', 'test', 'git'],
      last: '2026-10-27T23:51:09+0700',
    },
  ],
  failedTotal: 1833,
  last: ['dangth   pts/0        113.161.42.7     Wed Oct 28 09:12   still logged in'],
  lastb: [],
  lastbDenied: true,
  ufw: { installed: true, needsRoot: true, active: false, lines: [] },
  fail2ban: { installed: false, needsRoot: false, active: false, lines: [] },
  passwordAuth: 'yes',
  disablePassword: `sudo sh -c 'f=/etc/ssh/sshd_config.d/60-davish-no-password.conf; printf "PasswordAuthentication no\\nKbdInteractiveAuthentication no\\n" > "$f" && sshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd; } || { rm -f "$f"; echo "sshd -t failed: change reverted"; }'`,
}

export const backend = hasRuntime
  ? { report: securityReport }
  : {
      report: (_serverID: string) =>
        new Promise<SecurityReport>((r) => setTimeout(() => r(demo), 300)),
    }
