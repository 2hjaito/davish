// Everyone who helped, read once at build time: the maintainer first, then
// people with commits (by count), then those who opened issues or pull
// requests, then those who started or answered a discussion. Bots and the
// team account are left out. Anything unreadable leaves the list empty.
/** What someone did: commits, an issue or pull request, or a discussion. */
export type Role = 'code' | 'issue' | 'discussion'

export interface Person {
  login: string
  avatar: string
  url: string
  /** Every way they helped, in the order the list ranks them. */
  roles: Role[]
}

// The maintainer is listed, and always first; the team account is not.
const MAINTAINER = '2hjaito'
const LEFT_OUT = new Set(['danqth'])
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

// Discussions are only in GraphQL, which needs a token (the site workflow has
// one); without it they are skipped and the rest of the list still builds.
async function discussionPeople(): Promise<User[]> {
  if (!process.env.GITHUB_TOKEN) return []
  // 25 x 50 x 20 stays well under GraphQL's 500,000-node limit per request.
  const query = `query($after: String) { repository(owner: "2hjaito", name: "davish") {
    discussions(first: 25, after: $after) {
      pageInfo { hasNextPage endCursor }
      nodes {
        author { login avatarUrl url __typename }
        comments(first: 50) { nodes {
          author { login avatarUrl url __typename }
          replies(first: 20) { nodes { author { login avatarUrl url __typename } } }
        } }
      }
    } } }`
  interface Author {
    login: string
    avatarUrl: string
    url: string
    __typename: string
  }
  interface Node {
    author: Author | null
    comments?: { nodes: (Node & { replies?: { nodes: Node[] } })[] }
  }
  const out: User[] = []
  const take = (a: Author | null) => {
    if (a) out.push({ login: a.login, type: a.__typename, avatar_url: a.avatarUrl, html_url: a.url })
  }
  try {
    let after: string | null = null
    for (let page = 0; page < 5; page++) {
      const res = await fetch('https://api.github.com/graphql', {
        method: 'POST',
        headers,
        body: JSON.stringify({ query, variables: { after } }),
      })
      if (!res.ok) break
      const body = (await res.json()) as {
        data?: {
          repository?: {
            discussions: { pageInfo: { hasNextPage: boolean; endCursor: string }; nodes: Node[] }
          }
        }
      }
      const d = body.data?.repository?.discussions
      if (!d) {
        console.warn('contributors: could not read discussions')
        break
      }
      for (const n of d.nodes) {
        take(n.author)
        for (const c of n.comments?.nodes ?? []) {
          take(c.author)
          for (const r of c.replies?.nodes ?? []) take(r.author)
        }
      }
      if (!d.pageInfo.hasNextPage) break
      after = d.pageInfo.endCursor
    }
  } catch {
    // Leave out whoever was read before the failure.
  }
  return out
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
    // A person keeps the place of their first role and gains the others.
    const add = (u: User, role: Role) => {
      if (u.type !== 'User' || LEFT_OUT.has(u.login)) return
      const known = people.get(u.login)
      if (!known) {
        people.set(u.login, { login: u.login, avatar: u.avatar_url, url: u.html_url, roles: [role] })
      } else if (!known.roles.includes(role)) known.roles.push(role)
    }
    // Commits first, by count, with the maintainer moved to the front.
    const byCount = code.sort((a, b) => b.contributions - a.contributions)
    const me = byCount.find((u) => u.login === MAINTAINER)
    if (me) add(me, 'code')
    for (const u of byCount) add(u, 'code')
    for (let page = 1; page <= 5; page++) {
      const issues = await get<{ user: User }>(`${api}/issues?state=all&per_page=100&page=${page}`)
      for (const i of issues) add(i.user, 'issue')
      if (issues.length < 100) break
    }
    for (const u of await discussionPeople()) add(u, 'discussion')
    return [...people.values()]
  })()
  return cached
}
