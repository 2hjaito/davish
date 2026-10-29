// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  ClonePlan,
  CloneSpec,
  Branch,
  Commit,
  Diff,
  DiffRequest,
  GitOp,
  OpResult,
  Plan,
  Repo,
  Status,
} from '../../lib/api/git'

const delay = <T>(v: T, ms = 200) => new Promise<T>((r) => setTimeout(() => r(v), ms))

const repos: Repo[] = [
  { path: '/home/deploy/api', name: 'api' },
  { path: '/srv/docs', name: 'docs' },
  { path: '/var/www/shop', name: 'shop' },
]

const subjects = [
  'fix(cart): keep the coupon after login',
  'feat(checkout): pay by bank transfer',
  'chore(deps): bump express to 4.21.1',
  'docs: explain the deploy steps',
  'refactor(orders): split the invoice builder',
  'fix(api): return 404 for a missing product',
  'feat(admin): export orders as CSV',
  'test(cart): cover empty carts',
]
const authors = ['Trần Minh', 'Lê Hoa', 'deploy-bot']
const now = 1793170000

// A history with a long-lived develop branch above main, and on main a
// feature branch merged every few commits.
function commits(dir: string): Commit[] {
  const seed = dir.length
  const ids: [id: string, parents: string[], refs: Commit['refs']][] = [
    [
      'd2',
      ['d1'],
      [
        { name: 'develop', kind: 'local' },
        { name: 'origin/develop', kind: 'remote' },
      ],
    ],
    ['d1', ['m0'], []],
  ]
  for (let b = 0; b < 28; b++) {
    const next = b < 27 ? [`m${b + 1}`] : []
    ids.push(
      [
        `m${b}`,
        [`a${b}`, `f2${b}`],
        b === 0
          ? [
              { name: 'main', kind: 'head' },
              { name: 'origin/main', kind: 'remote' },
            ]
          : [],
      ],
      [`f2${b}`, [`f1${b}`], []],
      [`a${b}`, [`b${b}`], []],
      [`f1${b}`, [`b${b}`], []],
      [`b${b}`, next, b % 3 === 1 ? [{ name: `v1.${9 - Math.floor(b / 3)}.0`, kind: 'tag' }] : []],
    )
  }
  const hashOf = new Map(
    ids.map(([id], i) => [
      id,
      ((seed * 7919 + i * 104729) >>> 0).toString(16).padStart(8, '0').repeat(5),
    ]),
  )
  return ids.map(([id, parents, refs], i) => {
    const hash = hashOf.get(id) ?? ''
    const merge = parents.length > 1
    return {
      hash,
      short: hash.slice(0, 7),
      author: authors[(i + seed) % authors.length] ?? '',
      time: now - i * 9_000 - seed * 300,
      subject: merge
        ? `Merge branch 'feature/${subjects[i % subjects.length]?.split(':')[0] ?? 'x'}'`
        : (subjects[(i + seed) % subjects.length] ?? ''),
      parents: parents.map((p) => hashOf.get(p) ?? ''),
      refs,
    }
  })
}

export const mockFind = (extra: string[]): Promise<Repo[]> =>
  delay(
    [...repos, ...extra.map((p) => ({ path: `${p}/app`, name: 'app' }))].sort((a, b) =>
      a.path.localeCompare(b.path),
    ),
  )

export const mockStatus = (dir: string): Promise<Status> => {
  if (dir === '/srv/docs')
    return delay({
      branch: 'main',
      upstream: 'origin/main',
      ahead: 0,
      behind: 0,
      last: commits(dir)[0] ?? null,
      changes: [],
      truncated: false,
      install: '',
    })
  return delay({
    branch: dir === '/home/deploy/api' ? 'develop' : 'main',
    upstream: dir === '/home/deploy/api' ? '' : 'origin/main',
    ahead: 1,
    behind: 3,
    last: commits(dir)[0] ?? null,
    changes: [
      { path: 'src/cart/coupon.ts', origPath: '', index: '.', worktree: 'M', conflict: false },
      { path: 'src/orders/invoice.ts', origPath: '', index: 'M', worktree: '.', conflict: false },
      {
        path: 'src/orders/export.ts',
        origPath: 'src/orders/csv.ts',
        index: 'R',
        worktree: '.',
        conflict: false,
      },
      { path: 'public/old-banner.png', origPath: '', index: '.', worktree: 'D', conflict: false },
      { path: '.env.example', origPath: '', index: '?', worktree: '?', conflict: false },
    ],
    truncated: false,
    install: dir === '/var/www/shop' ? 'npm ci' : '',
  })
}

export const mockLog = (dir: string, skip: number): Promise<Commit[]> =>
  delay(commits(dir).slice(skip, skip + 100))

const sample = `diff --git a/src/cart/coupon.ts b/src/cart/coupon.ts
index 3b18e51..a9c4f02 100644
--- a/src/cart/coupon.ts
+++ b/src/cart/coupon.ts
@@ -12,9 +12,12 @@ export function applyCoupon(cart: Cart, code: string): Cart {
   const coupon = findCoupon(code)
   if (!coupon) return cart
-  if (coupon.expires < Date.now()) return cart
+  if (coupon.expires < Date.now()) {
+    log.info('coupon expired', { code })
+    return cart
+  }
   return {
     ...cart,
-    discount: coupon.amount,
+    discount: Math.min(coupon.amount, cart.total),
   }
 }
`

export const mockDiff = (_serverID: string, req: DiffRequest): Promise<Diff> =>
  delay({
    text: req.untracked
      ? `diff --git a/${req.file} b/${req.file}\nnew file mode 100644\n--- /dev/null\n+++ b/${req.file}\n@@ -0,0 +1,2 @@\n+DATABASE_URL=\n+SECRET_KEY=\n`
      : sample,
    truncated: false,
  })

export const mockBranches = (): Promise<Branch[]> =>
  delay([
    { name: 'develop', remote: false, current: true },
    { name: 'main', remote: false, current: false },
    { name: 'feature/bank-transfer', remote: true, current: false },
  ])

const commands: Record<GitOp, (branch: string) => string> = {
  fetch: () => 'env GIT_TERMINAL_PROMPT=0 timeout 120 git fetch --prune',
  pull: () => 'env GIT_TERMINAL_PROMPT=0 timeout 120 git pull --ff-only',
  switch: (b) => `git switch ${b}`,
}

export const mockPlan = (
  _serverID: string,
  dir: string,
  op: GitOp,
  branch: string,
): Promise<Plan> =>
  delay({ command: `cd -- ${dir} && ${commands[op](branch)}`, dirty: dir === '/srv/docs' ? 0 : 4 })

export const mockRun = (
  _serverID: string,
  dir: string,
  op: GitOp,
  _branch: string,
  _command: string,
): Promise<OpResult> =>
  delay(
    dir === '/var/www/shop' && op !== 'switch'
      ? {
          output:
            'git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.',
          error: 'GIT_AUTH_FAILED',
        }
      : {
          output:
            op === 'fetch'
              ? 'From github.com:davitek/docs\n   1a2b3c4..5d6e7f8  main -> origin/main'
              : '',
          error: '',
        },
    600,
  )

export function mockPlanClone(spec: CloneSpec): Promise<ClonePlan> {
  const ssh = !spec.url.startsWith('https://')
  const host = /(?:@|\/\/)([^:/]+)/.exec(spec.url)?.[1] ?? ''
  const name = (spec.url.split(/[/:]/).pop() ?? '').replace(/\.git$/, '')
  const branch = spec.branch ? `--branch ${spec.branch} ` : ''
  return Promise.resolve({
    command: spec.dir ? `git clone ${branch}-- ${spec.url} ${spec.dir}` : '',
    ssh,
    host,
    pinned: ssh && (host === 'github.com' || host === 'gitlab.com'),
    name,
  })
}

export const mockInitCommand = (dir: string, remote: string, url: string): Promise<string> =>
  Promise.resolve(
    `mkdir -p -- ${dir} && cd -- ${dir} && git init${url ? ` && git remote add ${remote || 'origin'} ${url}` : ''}`,
  )

export const mockRemoteCommand = (dir: string, remote: string, url: string): Promise<string> =>
  Promise.resolve(`cd -- ${dir} && git remote set-url ${remote || 'origin'} ${url}`)

export function mockDeployLine(dir: string, steps: string[]): Promise<string> {
  const kept = steps.map((s) => s.trim()).filter((s) => s && !s.startsWith('#'))
  const body = kept.flatMap((s, i) => [`echo '▶ ${i + 1}/${kept.length}  ${s}'`, `{ ${s}; }`])
  return Promise.resolve(
    [`cd -- ${dir}`, ...body, "echo '✓ Deploy finished'"].join(' && ') +
      " || echo '✗ Deploy stopped: the step above failed'",
  )
}
