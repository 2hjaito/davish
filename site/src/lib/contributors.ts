// Everyone who helped, read once at build time: people with commits first
// (by count), then those who opened issues or pull requests. The owner's
// accounts and bots are left out. Anything unreadable leaves the list empty.
export interface Person {
  login: string
  avatar: string
  url: string
  code: boolean
}

const OWNER = new Set(['2hjaito', 'danqth'])
const headers = {
  Accept: 'application/vnd.github+json',
  'User-Agent': 'davish-site',
  ...(process.env.GITHUB_TOKEN ? { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` } : {}),
}

interface User {
  login: string
  type: string
  avatar_url: string
  html_url: string
}

async function get<T>(url: string): Promise<T[]> {
  try {
    const res = await fetch(url, { headers })
    return res.ok ? ((await res.json()) as T[]) : []
  } catch {
    return []
  }
}

let cached: Promise<Person[]> | undefined

export function contributors(): Promise<Person[]> {
  cached ??= (async () => {
    const api = 'https://api.github.com/repos/2hjaito/davish'
    const code = await get<User & { contributions: number }>(`${api}/contributors?per_page=100`)
    const people = new Map<string, Person>()
    const add = (u: User, isCode: boolean) => {
      if (u.type !== 'User' || OWNER.has(u.login) || people.has(u.login)) return
      people.set(u.login, { login: u.login, avatar: u.avatar_url, url: u.html_url, code: isCode })
    }
    for (const u of code.sort((a, b) => b.contributions - a.contributions)) add(u, true)
    for (let page = 1; page <= 5; page++) {
      const issues = await get<{ user: User }>(`${api}/issues?state=all&per_page=100&page=${page}`)
      for (const i of issues) add(i.user, false)
      if (issues.length < 100) break
    }
    return [...people.values()]
  })()
  return cached
}
