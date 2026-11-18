// The latest release, read once at build time; the deploy runs again on
// every release (.github/workflows/site.yml). Without it, links go to the
// releases page.
const repo = 'https://github.com/2hjaito/davish'

export interface Release {
  version: string
  notes: string
  files: Record<'macos' | 'macosZip' | 'windows' | 'windowsZip' | 'linux' | 'sums', string>
}

const latest = `${repo}/releases/latest`

function fallback(): Release {
  const f = { macos: latest, macosZip: latest, windows: latest, windowsZip: latest, linux: latest, sums: latest }
  return { version: '', notes: latest, files: f }
}

let cached: Promise<Release> | undefined

export function release(): Promise<Release> {
  cached ??= load()
  return cached
}

async function load(): Promise<Release> {
  try {
    const res = await fetch('https://api.github.com/repos/2hjaito/davish/releases/latest', {
      headers: {
        Accept: 'application/vnd.github+json',
        'User-Agent': 'davish-site',
        ...(process.env.GITHUB_TOKEN ? { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` } : {}),
      },
    })
    if (!res.ok) throw new Error(`GitHub answered ${res.status}`)
    const r = (await res.json()) as { tag_name: string; html_url: string; assets: { name: string; browser_download_url: string }[] }
    const pick = (end: string) => r.assets.find((a) => a.name.endsWith(end))?.browser_download_url ?? r.html_url
    return {
      version: r.tag_name.replace(/^v/, ''),
      notes: r.html_url,
      files: {
        macos: pick('-macos-universal.pkg'),
        macosZip: pick('-macos-universal.zip'),
        windows: pick('-windows-amd64-setup.exe'),
        windowsZip: pick('-windows-amd64.zip'),
        linux: pick('-linux-amd64.tar.gz'),
        sums: pick('SHA256SUMS.txt'),
      },
    }
  } catch (e) {
    console.warn(`site: no release info, linking to the releases page (${String(e)})`)
    return fallback()
  }
}

export const links = {
  repo,
  features: (lang: string) => `${repo}/blob/main/${lang === 'vi' ? 'FEATURES.vi.md' : 'FEATURES.md'}`,
  readme: (lang: string) =>
    `${repo}/blob/main/${lang === 'vi' ? 'README.vi.md' : lang === 'zh-CN' ? 'README.zh-CN.md' : 'README.md'}`,
  security: `${repo}/blob/main/SECURITY.md`,
  changelog: (lang: string) => `${repo}/blob/main/${lang === 'vi' ? 'CHANGELOG.md' : 'CHANGELOG.en.md'}`,
  issues: `${repo}/issues`,
}
