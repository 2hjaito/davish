// Numbers for the stats bar, read once at build time. Anything that cannot
// be read is left out rather than guessed.
import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

export interface Stats {
  downloads?: number
  stars?: number
  apps: number
}

const headers = {
  Accept: 'application/vnd.github+json',
  'User-Agent': 'davish-site',
  ...(process.env.GITHUB_TOKEN ? { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` } : {}),
}

async function json<T>(url: string, h: Record<string, string> = {}): Promise<T | undefined> {
  try {
    const res = await fetch(url, { headers: h })
    return res.ok ? ((await res.json()) as T) : undefined
  } catch {
    return undefined
  }
}

// Each folder in frontend/src/apps is one app of the desktop. Builds run
// from site/, so the app sits one level up.
function apps(): number {
  const dir = join(process.cwd(), '..', 'frontend', 'src', 'apps')
  return readdirSync(dir).filter((n) => statSync(join(dir, n)).isDirectory()).length
}

let cached: Promise<Stats> | undefined

export function stats(): Promise<Stats> {
  cached ??= (async () => {
    // The README badge's own number, kept up to date by repo-stats.yml.
    const dl = await json<{ message: string }>('https://raw.githubusercontent.com/2hjaito/davish/stats/downloads.json')
    const repo = await json<{ stargazers_count: number }>('https://api.github.com/repos/2hjaito/davish', headers)
    const n = Number(dl?.message?.replace(/[^\d.]/g, ''))
    return {
      downloads: Number.isFinite(n) && n > 0 ? n : undefined,
      stars: repo?.stargazers_count,
      apps: apps(),
    }
  })()
  return cached
}
