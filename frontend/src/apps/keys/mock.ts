// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  Authorized,
  CreatePlan,
  GitHost,
  GitTestPlan,
  GitTestResult,
  KeyPlan,
  LocalKey,
  ServerKey,
} from '../../lib/api/keys'

const delay = <T>(v: T, ms = 150) => new Promise<T>((r) => setTimeout(() => r(v), ms))
const fp = (seed: string) => `SHA256:${btoa(seed.padEnd(32, seed)).slice(0, 43)}`
const pub = (seed: string, comment: string) =>
  `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI${btoa(seed.padEnd(24, '.')).slice(0, 32)} ${comment}`

let authorized: Authorized = {
  path: '/home/dangth/.ssh/authorized_keys',
  keys: [
    ['laptop', 'dang@macbook', true],
    ['ci', 'github-actions deploy', false],
    ['old', 'old-office-pc', false],
  ].map(([seed, comment, login]) => ({
    type: 'ssh-ed25519',
    fingerprint: fp(seed as string),
    comment: comment as string,
    options: [],
    login: login as boolean,
    text: pub(seed as string, comment as string),
  })),
}

let serverKeys: ServerKey[] = [
  {
    name: 'deploy_davitek',
    path: '/home/dangth/.ssh/deploy_davitek',
    type: 'ssh-ed25519',
    fingerprint: fp('deploy'),
    comment: 'deploy@dangth.dev',
    public: pub('deploy', 'deploy@dangth.dev'),
  },
]

let localKeys: LocalKey[] = [
  {
    name: 'id_ed25519',
    path: '/Users/dangth/.ssh/id_ed25519',
    type: 'ssh-ed25519',
    fingerprint: fp('laptop'),
    comment: 'dang@macbook',
    public: pub('laptop', 'dang@macbook'),
    encrypted: false,
  },
]

export const mockAuthorized = () => delay({ ...authorized, keys: [...authorized.keys] })

export function mockPlanAdd(text: string): Promise<KeyPlan> {
  const comment = text.trim().split(/\s+/).slice(2).join(' ')
  const command = `mkdir -p -m 700 -- /home/dangth/.ssh && touch -- ${authorized.path} && chmod 600 -- ${authorized.path} && { grep -qxF -- '${text}' ${authorized.path} || printf '%s\\n' '${text}' >> ${authorized.path}; }`
  return delay({ fingerprint: fp(text), comment, login: false, command })
}

export async function mockAdd(text: string): Promise<void> {
  await delay(null)
  const comment = text.trim().split(/\s+/).slice(2).join(' ')
  authorized = {
    ...authorized,
    keys: [
      ...authorized.keys,
      { type: 'ssh-ed25519', fingerprint: fp(text), comment, options: [], login: false, text },
    ],
  }
}

export function mockPlanRemove(fingerprint: string): Promise<KeyPlan> {
  const k = authorized.keys.find((x) => x.fingerprint === fingerprint)!
  const backup = `${authorized.path}.bak-20261028T120000Z`
  const command = `cp -p -- ${authorized.path} ${backup} && { grep -vxF -e '${k.text}' -- ${authorized.path} > ${authorized.path}.davish-tmp || [ $? -eq 1 ]; } && chmod 600 -- ${authorized.path}.davish-tmp && mv -- ${authorized.path}.davish-tmp ${authorized.path}`
  return delay({ fingerprint, comment: k.comment, login: k.login, backup, command })
}

export async function mockRemove(plan: KeyPlan): Promise<void> {
  await delay(null)
  authorized = {
    ...authorized,
    keys: authorized.keys.filter((k) => k.fingerprint !== plan.fingerprint),
  }
}

export const mockServerKeys = () => delay([...serverKeys])

export function mockPlanCreate(name: string, comment: string): Promise<CreatePlan> {
  const path = `/home/dangth/.ssh/${name}`
  return delay({
    name,
    path,
    comment,
    command: `mkdir -p -m 700 -- /home/dangth/.ssh && ssh-keygen -q -t ed25519 -N '' -C '${comment}' -f ${path}`,
  })
}

export async function mockCreate(name: string, comment: string): Promise<void> {
  await delay(null, 400)
  serverKeys = [
    ...serverKeys,
    {
      name,
      path: `/home/dangth/.ssh/${name}`,
      type: 'ssh-ed25519',
      fingerprint: fp(name),
      comment,
      public: pub(name, comment),
    },
  ]
}

export function mockPlanGitTest(host: GitHost, name: string): Promise<GitTestPlan> {
  return delay({
    host,
    key: name,
    command: `f="$(mktemp)" && ssh-keyscan -t ed25519 ${host} > "$f" … ssh -T -i /home/dangth/.ssh/${name} git@${host}`,
  })
}

/** The deploy key works; any other key is unknown to the host. */
export const mockGitTest = (host: GitHost, name: string): Promise<GitTestResult> =>
  delay(
    name === 'deploy_davitek'
      ? {
          outcome: 'ok' as const,
          account: 'davitek',
          output: `Hi davitek! You've successfully authenticated, but ${host} does not provide shell access.`,
        }
      : { outcome: 'denied' as const, output: `git@${host}: Permission denied (publickey).` },
    900,
  )

export const mockLocal = () => delay([...localKeys])

export async function mockGenerateLocal(
  name: string,
  comment: string,
  passphrase: string,
): Promise<LocalKey> {
  await delay(null, 300)
  if (localKeys.some((k) => k.name === name)) throw { code: 'FS_EXISTS' }
  const k: LocalKey = {
    name,
    path: `/Users/dangth/.ssh/${name}`,
    type: 'ssh-ed25519',
    fingerprint: fp(name),
    comment,
    public: pub(name, comment),
    encrypted: passphrase !== '',
  }
  localKeys = [...localKeys, k]
  return k
}
