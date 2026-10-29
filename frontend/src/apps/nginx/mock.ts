// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  CreatePlan,
  CreateResult,
  EditPlan,
  NewSite,
  NginxReload,
  NginxSite,
  NginxTest,
  RemovePlan,
  SiteEdit,
  SiteOptions,
  SSLPlan,
  TogglePlan,
} from '../../lib/api/nginx'

const day = 86_400
const now = () => Math.floor(Date.now() / 1000)

let sites: NginxSite[] = [
  {
    name: 'default',
    path: '/etc/nginx/sites-available/default',
    kind: 'site',
    enabled: true,
    toggleable: true,
    servers: [
      {
        names: ['_'],
        listen: ['80 default_server', '[::]:80 default_server'],
        root: '/var/www/html',
        ssl: false,
        certs: [],
        proxyPass: [],
        line: 22,
      },
    ],
    upstreams: [],
    certs: [],
  },
  {
    name: 'shop.davitek.com',
    path: '/etc/nginx/sites-available/shop.davitek.com',
    kind: 'site',
    enabled: true,
    toggleable: true,
    servers: [
      {
        names: ['shop.davitek.com', 'davitek.com'],
        listen: ['443 ssl http2', '[::]:443 ssl'],
        ssl: true,
        certs: ['/etc/letsencrypt/live/shop.davitek.com/fullchain.pem'],
        proxyPass: ['http://shop_backend'],
        line: 6,
      },
      {
        names: ['shop.davitek.com'],
        listen: ['80'],
        ssl: false,
        certs: [],
        proxyPass: [],
        line: 24,
      },
    ],
    upstreams: ['shop_backend'],
    certs: [
      {
        path: '/etc/letsencrypt/live/shop.davitek.com/fullchain.pem',
        notAfter: now() + 9 * day,
        names: ['shop.davitek.com', 'davitek.com'],
      },
    ],
  },
  {
    name: 'old-blog',
    path: '/etc/nginx/sites-available/old-blog',
    kind: 'site',
    enabled: false,
    toggleable: true,
    servers: [
      {
        names: ['blog.dangth.dev'],
        listen: ['80'],
        root: '/srv/blog',
        ssl: false,
        certs: [],
        proxyPass: [],
        line: 1,
      },
    ],
    upstreams: [],
    certs: [],
  },
  {
    name: 'gzip.conf',
    path: '/etc/nginx/conf.d/gzip.conf',
    kind: 'conf',
    enabled: true,
    toggleable: false,
    servers: [],
    upstreams: [],
    certs: [],
  },
]

const delay = <T>(v: T, ms = 150) => new Promise<T>((r) => setTimeout(() => r(v), ms))

export const mockList = () => delay(sites.map((s) => ({ ...s })))

export function mockPlanToggle(name: string, enable: boolean): Promise<TogglePlan> {
  const command = enable
    ? `ln -s -- /etc/nginx/sites-available/${name} /etc/nginx/sites-enabled/${name}`
    : `rm -- /etc/nginx/sites-enabled/${name}`
  return delay({ name, enable, command, sudo: `sudo ${command}` })
}

export function mockPlanRemove(path: string, withCerts: boolean): Promise<RemovePlan> {
  const site = sites.find((s) => s.path === path)!
  const certs = [
    ...new Set(
      site.servers
        .flatMap((s) => s.certs)
        .map((c) => /^\/etc\/letsencrypt\/live\/([^/]+)\//.exec(c)?.[1])
        .filter((n): n is string => !!n),
    ),
  ]
  const steps = [
    ...(site.toggleable && site.enabled ? [`rm -- /etc/nginx/sites-enabled/${site.name}`] : []),
    `rm -- ${site.path}`,
    ...(withCerts ? certs.map((n) => `certbot delete --non-interactive --cert-name ${n}`) : []),
  ]
  return delay({
    path,
    name: site.name,
    enabled: site.enabled,
    certs,
    withCerts: withCerts && certs.length > 0,
    sharedCerts: [],
    command: steps.join(' && '),
    sudo: steps.map((s) => `sudo ${s}`).join(' && '),
  })
}

export async function mockRemove(plan: RemovePlan): Promise<void> {
  await delay(null)
  sites = sites.filter((s) => s.path !== plan.path)
}

export async function mockToggle(plan: TogglePlan): Promise<void> {
  await delay(null)
  sites = sites.map((s) => (s.name === plan.name ? { ...s, enabled: plan.enable } : s))
}

const okTest: NginxTest = {
  ok: true,
  needsRoot: false,
  issues: [],
  output:
    'nginx: the configuration file /etc/nginx/nginx.conf syntax is ok\nnginx: configuration file /etc/nginx/nginx.conf test is successful',
}

// old-blog enabled breaks the demo config, to show an error with a line.
const broken = () => sites.some((s) => s.name === 'old-blog' && s.enabled)
const badTest = (): NginxTest => ({
  ok: false,
  needsRoot: false,
  issues: [
    {
      level: 'emerg',
      message: 'unknown directive "bogus_directive"',
      file: '/etc/nginx/sites-available/old-blog',
      line: 7,
    },
  ],
  output:
    '2026/10/27 [emerg] 1#1: unknown directive "bogus_directive" in /etc/nginx/sites-available/old-blog:7\nnginx: configuration file /etc/nginx/nginx.conf test failed',
})

export const mockTest = () => delay(broken() ? badTest() : okTest, 400)

export const mockReload = (): Promise<NginxReload> =>
  delay(
    broken()
      ? { test: badTest(), reloaded: false, needsRoot: false, sudo: '' }
      : {
          test: okTest,
          reloaded: false,
          needsRoot: true,
          sudo: 'sudo nginx -t && sudo systemctl reload nginx',
        },
    500,
  )

export const mockSiteOptions = (): Promise<SiteOptions> =>
  delay({ confD: false, phpSockets: ['/run/php/php8.3-fpm.sock'] })

export function mockRenderSite(site: NewSite): Promise<string> {
  const body =
    site.type === 'proxy'
      ? `    location / {\n        proxy_pass http://${site.upstream};\n        proxy_set_header Host $host;\n    }`
      : `    root ${site.root};\n    index index.html;`
  return delay(`server {\n    listen 80;\n    server_name ${site.names.join(' ')};\n\n${body}\n}\n`)
}

export function mockPlanCreate(site: NewSite, config: string): Promise<CreatePlan> {
  const name = site.names[0] ?? 'site'
  const path = `/etc/nginx/sites-available/${name}`
  const link = `/etc/nginx/sites-enabled/${name}`
  const command = `printf %s '${config}' | tee -- ${path} > /dev/null && ln -s -- ${path} ${link} && { nginx -t && systemctl reload nginx || { rm -- ${link}; false; }; }`
  return delay({ name, path, config, command, sudo: command })
}

export async function mockCreate(plan: CreatePlan): Promise<CreateResult> {
  await delay(null, 500)
  const name = plan.name
  sites = [
    ...sites,
    {
      name,
      path: plan.path,
      kind: 'site',
      enabled: true,
      toggleable: true,
      servers: [{ names: [name], listen: ['80'], ssl: false, certs: [], proxyPass: [], line: 1 }],
      upstreams: [],
      certs: [],
    },
  ]
  return { created: true, test: okTest, needsRoot: false, sudo: plan.sudo }
}

export function mockPlanSSL(names: string[], redirect: boolean): Promise<SSLPlan> {
  const ok = names.filter((n) => n.includes('.') && !n.startsWith('*'))
  return delay({
    names: ok,
    skipped: names.filter((n) => !ok.includes(n)),
    serverIPs: ['203.0.113.5'],
    dns: ok.map((n, i) =>
      i === 0
        ? { name: n, ips: ['203.0.113.5'], ok: true }
        : { name: n, ips: [], ok: false, error: 'no DNS record' },
    ),
    redirect,
    sudo: ok.length
      ? `sudo certbot --nginx ${ok.map((n) => `-d ${n}`).join(' ')} ${redirect ? '--redirect' : '--no-redirect'}`
      : '',
  })
}

export function mockPlanEdit(path: string, edit: SiteEdit): Promise<EditPlan> {
  const backup = `${path}.bak-20261029T080000Z`
  const lines = [
    edit.addNames.length ? `    server_name … ${edit.addNames.join(' ')};` : '',
    edit.upstream ? `        proxy_pass http://${edit.upstream};` : '',
    edit.gzip ? `    gzip ${edit.gzip};` : '',
    edit.maxBody ? `    client_max_body_size ${edit.maxBody};` : '',
  ].filter(Boolean)
  const after = `server {\n    listen 80;\n${lines.join('\n')}\n}\n`
  const command = `cp -p -- ${path} ${backup} && printf %s '${after}' | tee -- ${path} > /dev/null && { nginx -t && systemctl reload nginx || { cp -p -- ${backup} ${path}; false; }; }`
  return delay({ path, backup, before: '', after, command, sudo: command })
}

export const mockEdit = (plan: EditPlan): Promise<CreateResult> =>
  delay({ created: true, test: okTest, needsRoot: false, sudo: plan.sudo }, 500)
