// The security call, or demo data in the plain browser preview.
import { hasRuntime } from '../../lib/api/runtime'
import {
  securityApply,
  securityDisablePassword,
  securityEnablePassword,
  securityPasswordPlan,
  securityPasswordStatus,
  securityPlan,
  securityReport,
  type HourCount,
  type PasswordPlan,
  type PasswordStatus,
  type SecurityAction,
  type SecurityReport,
} from '../../lib/api/security'

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
  sudo: true,
  canSudo: true,
  asRoot: false,
  askSudo: false,
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
  ufw: {
    installed: true,
    needsRoot: false,
    active: true,
    lines: [
      'To                         Action      From',
      '--                         ------      ----',
      'Anywhere                   DENY        218.92.0.34',
      'Anywhere                   DENY        61.177.172.140',
      '22/tcp                     ALLOW       Anywhere',
      '80,443/tcp                 ALLOW       Anywhere',
    ],
  },
  fail2ban: {
    installed: true,
    needsRoot: false,
    active: true,
    lines: [],
    banned: ['45.148.10.92', '193.32.162.77'],
  },
  passwordAuth: 'yes',
  passwordOurs: false,
  disablePassword: `sudo sh -c 'f=/etc/ssh/sshd_config.d/60-davish-no-password.conf; printf "PasswordAuthentication no\\nKbdInteractiveAuthentication no\\n" > "$f" && sshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd; } || { rm -f "$f"; echo "sshd -t failed: change reverted"; }'`,
}

// The preview's plan mirrors internal/security/actions.go.
function demoPlan(action: SecurityAction, ip: string) {
  const line = {
    block: `ufw delete deny from ${ip}; ufw insert 1 deny from ${ip} || ufw deny from ${ip}`,
    unblock: `ufw delete deny from ${ip}`,
    ban: `fail2ban-client set sshd banip ${ip}`,
    unban: `fail2ban-client set sshd unbanip ${ip}`,
  }[action]
  const sudo = (p: string) => (line.includes(';') ? `${p}sh -c '${line}'` : p + line)
  return Promise.resolve({ command: sudo('sudo -n '), terminal: sudo('sudo ') })
}

// The preview applies an action to its own report, so the lists change.
function demoApply(action: SecurityAction, ip: string) {
  const rule = `Anywhere                   DENY        ${ip}`
  const f = demo.fail2ban
  if (action === 'block') demo.ufw.lines.splice(2, 0, rule)
  if (action === 'unblock') demo.ufw.lines = demo.ufw.lines.filter((l) => l !== rule)
  if (action === 'ban') f.banned = [...(f.banned ?? []), ip]
  if (action === 'unban') f.banned = (f.banned ?? []).filter((b) => b !== ip)
  return new Promise<void>((r) => setTimeout(r, 300))
}

// The preview's password change shows the scripts of internal/security/password.go.
const demoPasswordPlan = (): PasswordPlan => ({
  status: {
    command:
      'sudo -n sh -c \'echo --eff--; sshd -T 2>/dev/null | grep -iE "^(passwordauthentication|kbdinteractiveauthentication|authenticationmethods) "; echo --include--; [ -d /etc/ssh/sshd_config.d ] && grep -qiE "^[[:space:]]*include[[:space:]].*sshd_config\\.d" /etc/ssh/sshd_config && echo yes; echo --ours--; [ -e /etc/ssh/sshd_config.d/00-davish-no-password.conf ] && echo yes; echo --cfg--; cat /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null; true\'',
    terminal:
      'sudo sh -c \'echo --eff--; sshd -T 2>/dev/null | grep -iE "^(passwordauthentication|kbdinteractiveauthentication|authenticationmethods) "; echo --include--; [ -d /etc/ssh/sshd_config.d ] && grep -qiE "^[[:space:]]*include[[:space:]].*sshd_config\\.d" /etc/ssh/sshd_config && echo yes; echo --ours--; [ -e /etc/ssh/sshd_config.d/00-davish-no-password.conf ] && echo yes; echo --cfg--; cat /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null; true\'',
  },
  disable: {
    command:
      'sudo -n sh -c \'f=/etc/ssh/sshd_config.d/00-davish-no-password.conf\n[ -d /etc/ssh/sshd_config.d ] && grep -qiE "^[[:space:]]*include[[:space:]].*sshd_config\\.d" /etc/ssh/sshd_config || { echo DAVISH_NO_INCLUDE; exit 3; }\nb=/var/backups/davish-sshd-$(date +%Y%m%d-%H%M%S)\nmkdir -p "$b" && cp -a /etc/ssh/sshd_config /etc/ssh/sshd_config.d "$b"/ || { echo DAVISH_BACKUP; exit 3; }\nprintf "PasswordAuthentication no\\nKbdInteractiveAuthentication no\\n" > "$f"\nsshd -t || { rm -f "$f"; echo DAVISH_SSHD_T; exit 4; }\nt=$(sshd -T) && echo "$t" | grep -qix "passwordauthentication no" && ! echo "$t" | grep -qixE "kbdinteractiveauthentication yes" || { rm -f "$f"; echo DAVISH_OVERRIDDEN; exit 5; }\n{ systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null; } || { rm -f "$f"; echo DAVISH_RELOAD; exit 6; }\necho "backup in $b"\'',
    terminal:
      'sudo sh -c \'f=/etc/ssh/sshd_config.d/00-davish-no-password.conf\n[ -d /etc/ssh/sshd_config.d ] && grep -qiE "^[[:space:]]*include[[:space:]].*sshd_config\\.d" /etc/ssh/sshd_config || { echo DAVISH_NO_INCLUDE; exit 3; }\nb=/var/backups/davish-sshd-$(date +%Y%m%d-%H%M%S)\nmkdir -p "$b" && cp -a /etc/ssh/sshd_config /etc/ssh/sshd_config.d "$b"/ || { echo DAVISH_BACKUP; exit 3; }\nprintf "PasswordAuthentication no\\nKbdInteractiveAuthentication no\\n" > "$f"\nsshd -t || { rm -f "$f"; echo DAVISH_SSHD_T; exit 4; }\nt=$(sshd -T) && echo "$t" | grep -qix "passwordauthentication no" && ! echo "$t" | grep -qixE "kbdinteractiveauthentication yes" || { rm -f "$f"; echo DAVISH_OVERRIDDEN; exit 5; }\n{ systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null; } || { rm -f "$f"; echo DAVISH_RELOAD; exit 6; }\necho "backup in $b"\'',
  },
  enable: {
    command:
      "sudo -n sh -c 'rm -f /etc/ssh/sshd_config.d/00-davish-no-password.conf\nsshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null; }'",
    terminal:
      "sudo sh -c 'rm -f /etc/ssh/sshd_config.d/00-davish-no-password.conf\nsshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null; }'",
  },
})
const demoPasswordStatus = (): PasswordStatus => ({
  password: demo.passwordAuth,
  kbdInteractive: 'no',
  ours: demo.passwordOurs,
  blocked: [],
})
function demoSetPassword(off: boolean) {
  demo.passwordAuth = off ? 'no' : 'yes'
  demo.passwordOurs = off
  return new Promise<void>((r) => setTimeout(r, 600))
}

export const backend = hasRuntime
  ? {
      report: securityReport,
      plan: securityPlan,
      apply: securityApply,
      passwordStatus: securityPasswordStatus,
      passwordPlan: securityPasswordPlan,
      disablePassword: securityDisablePassword,
      enablePassword: securityEnablePassword,
    }
  : {
      report: (_serverID: string) =>
        new Promise<SecurityReport>((r) => setTimeout(() => r({ ...demo }), 300)),
      plan: (action: SecurityAction, ip: string, _root: boolean) => demoPlan(action, ip),
      apply: (_serverID: string, action: SecurityAction, ip: string, _command: string) =>
        demoApply(action, ip),
      passwordStatus: (_serverID: string) => Promise.resolve(demoPasswordStatus()),
      passwordPlan: (_root: boolean) => Promise.resolve(demoPasswordPlan()),
      disablePassword: (_serverID: string, _command: string) => demoSetPassword(true),
      enablePassword: (_serverID: string, _command: string) => demoSetPassword(false),
    }
