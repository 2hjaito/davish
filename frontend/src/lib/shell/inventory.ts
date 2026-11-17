// Helpers for the Ansible inventory tab of the import dialog.
import type { InventoryHost, InventoryIssue, InventoryListing } from '../api/servers'
import { t } from '../i18n'

/** The hosts shown for a group filter; '' shows every host. */
export function visibleHosts(hosts: readonly InventoryHost[], group: string): InventoryHost[] {
  return group === '' ? [...hosts] : hosts.filter((h) => h.groups.includes(group))
}

/** How many importable (not yet saved) hosts each group holds. */
export function groupCounts(hosts: readonly InventoryHost[]): Map<string, number> {
  const counts = new Map<string, number>()
  for (const h of hosts) {
    if (h.exists) continue
    for (const g of h.groups) counts.set(g, (counts.get(g) ?? 0) + 1)
  }
  return counts
}

/** A skip reason or a note, in the user's language. */
export function issueText(i: InventoryIssue): string {
  const m = t.importDialog.issues
  const d = i.detail ?? ''
  switch (i.code) {
    case 'connection':
      return m.connection(d)
    case 'jinja':
      return m.jinja(d)
    case 'vault':
      return m.vault(d)
    case 'port':
      return m.port(d)
    case 'range':
      return m.range(d)
    case 'password':
      return m.password
    case 'sshArgs':
      return m.sshArgs(d)
  }
}

/** The address as ssh would show it. */
export const address = (h: Pick<InventoryHost, 'user' | 'address' | 'port'>): string =>
  `${h.user}@${h.address}${h.port === 22 ? '' : `:${h.port}`}`

/** A sample inventory for the browser preview, which cannot read files. */
export function mockInventory(): InventoryListing {
  const host = (
    name: string,
    groups: string[],
    extra: Partial<InventoryHost> = {},
  ): InventoryHost => ({
    name,
    groups,
    address: name,
    user: 'deploy',
    port: 22,
    exists: false,
    ...extra,
  })
  return {
    path: '/etc/ansible/hosts',
    groups: ['ungrouped', 'web', 'db', 'prod'],
    hosts: [
      host('bastion', ['ungrouped'], { address: '203.0.113.10', user: 'jump' }),
      host('web1.example.com', ['web', 'prod'], { proxyJump: 'bastion' }),
      host('web2.example.com', ['web', 'prod'], { proxyJump: 'bastion' }),
      host('web3.example.com', ['web', 'prod'], { proxyJump: 'bastion', exists: true }),
      host('db1', ['db', 'prod'], {
        address: '10.0.1.5',
        user: 'ops',
        port: 2222,
        proxyJump: 'bastion',
        notes: [{ code: 'password' }],
      }),
    ],
    skipped: [
      { name: 'db2', groups: ['db', 'prod'], issue: { code: 'vault', detail: 'ansible_host' } },
      { name: 'win1', groups: ['win'], issue: { code: 'connection', detail: 'winrm' } },
    ],
  }
}
