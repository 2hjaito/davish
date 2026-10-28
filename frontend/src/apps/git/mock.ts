// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
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

function commits(dir: string): Commit[] {
  const seed = dir.length
  return Array.from({ length: 140 }, (_, i) => {
    const hash = ((seed * 7919 + i * 104729) >>> 0).toString(16).padStart(8, '0').repeat(5)
    return {
      hash,
      short: hash.slice(0, 7),
      author: authors[(i + seed) % authors.length] ?? '',
      time: now - i * 9_000 - seed * 300,
      subject: subjects[(i + seed) % subjects.length] ?? '',
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
              ? 'From github.com:acme/docs\n   1a2b3c4..5d6e7f8  main -> origin/main'
              : '',
          error: '',
        },
    600,
  )
