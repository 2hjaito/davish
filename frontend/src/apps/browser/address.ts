// The Browser app's address bar: what the user types names a port of the
// server, which the proxy serves at a local port of this machine.

export interface Address {
  port: number
  /** Path, query and hash, starting with '/'. */
  path: string
}

/**
 * Reads "localhost:3000/admin", ":3000", "3000", "http://127.0.0.1:8080/x"
 * or "localhost/x" (port 80); undefined for anything that is not the server.
 */
export function parseAddress(text: string): Address | undefined {
  let s = text.trim()
  if (!s) return undefined
  if (/^\d+(\/|$)/.test(s)) s = `localhost:${s}`
  if (s.startsWith(':')) s = `localhost${s}`
  if (!/^[a-z]+:\/\//i.test(s)) s = `http://${s}`
  let u: URL
  try {
    u = new URL(s)
  } catch {
    return undefined
  }
  if (u.protocol !== 'http:') return undefined
  if (!['localhost', '127.0.0.1', '0.0.0.0', '[::1]'].includes(u.hostname)) return undefined
  const port = u.port ? Number(u.port) : 80
  if (!(port >= 1 && port <= 65535)) return undefined
  return { port, path: `${u.pathname}${u.search}${u.hash}` }
}

/** The URL the frame loads: the proxy's port, the path, and the token. */
export function frameURL(localPort: number, path: string, token: string): string {
  const u = new URL(path, `http://127.0.0.1:${localPort}`)
  u.searchParams.set('__davish', token)
  return u.toString()
}

/** How a proxied page's address reads in the bar, as on the server. */
export function displayAddress(href: string, remotePort: number): string {
  const u = new URL(href)
  u.searchParams.delete('__davish')
  const port = remotePort === 80 ? '' : `:${remotePort}`
  return `localhost${port}${u.pathname}${u.search}${u.hash}`
}
