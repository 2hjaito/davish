// Which technology a container's image is, for the logo on its card. Only
// images Davish knows by name: a stack with one of the user's own images gets
// no logo from here, since its databases and proxies say little about it.
import type { Group } from './containers'
import type { Tech } from '../../lib/ui/techIcons'

/** Matched against the image's repository path without registry or tag, in order. */
const RULES: readonly [RegExp, Tech][] = [
  [/(^|\/)n8n$/, 'n8n'],
  [/(^|\/)wordpress$/, 'wordpress'],
  [/(^|\/)ghost$/, 'ghost'],
  [/(^|\/)strapi$/, 'strapi'],
  [/(^|\/)nextcloud$/, 'nextcloud'],
  [/(^|\/)jellyfin$/, 'jellyfin'],
  [/(^|\/)gitea$/, 'gitea'],
  [/(^|\/)(vaultwarden\/)?server$|(^|\/)vaultwarden$/, 'vaultwarden'],
  [/(^|\/)portainer(-ce|-ee)?$/, 'portainer'],
  [/(^|\/)uptime-kuma$/, 'uptimekuma'],
  [/(^|\/)home-?assistant$/, 'homeassistant'],
  [/(^|\/)grafana(-oss|-enterprise)?$/, 'grafana'],
  [/(^|\/)prometheus$/, 'prometheus'],
  [/^supabase\//, 'supabase'],
  [/(^|\/)node$/, 'node'],
  [/(^|\/)bun$/, 'bun'],
  [/(^|\/)deno$/, 'deno'],
  [/(^|\/)python$/, 'python'],
  [/(^|\/)golang$/, 'go'],
  [/(^|\/)php$/, 'php'],
  [/(^|\/)ruby$/, 'ruby'],
  [/(^|\/)rust$/, 'rust'],
  [/(^|\/)dotnet\/(aspnet|runtime|sdk)$/, 'dotnet'],
  [/(^|\/)nginx(-proxy-manager|-unprivileged)?$/, 'nginx'],
  [/(^|\/)traefik$/, 'traefik'],
  [/(^|\/)caddy$/, 'caddy'],
  [/(^|\/)(postgres|postgresql|postgis|timescaledb(-ha)?)$/, 'postgres'],
  [/(^|\/)mysql(-server)?$/, 'mysql'],
  [/(^|\/)mariadb$/, 'mariadb'],
  [/(^|\/)redis(-stack)?$/, 'redis'],
  [/(^|\/)mongo(db)?$/, 'mongodb'],
  [/(^|\/)elasticsearch$/, 'elasticsearch'],
  [/(^|\/)meilisearch$/, 'meilisearch'],
  [/(^|\/)rabbitmq$/, 'rabbitmq'],
  [/(^|\/)minio$/, 'minio'],
]

/** "ghcr.io/foo/bar:1.2@sha256:…" → "foo/bar"; Docker Hub's "library/" dropped. */
export function repository(image: string): string {
  let r = image.split('@')[0] ?? ''
  const slash = r.lastIndexOf('/')
  const colon = r.lastIndexOf(':')
  if (colon > slash) r = r.slice(0, colon)
  const parts = r.split('/')
  // A first part with a dot or port, or "localhost", is a registry.
  if (parts.length > 1 && /[.:]|^localhost$/.test(parts[0] ?? '')) parts.shift()
  if (parts[0] === 'library') parts.shift()
  return parts.join('/').toLowerCase()
}

export function imageTech(image: string): Tech | undefined {
  const repo = repository(image)
  return RULES.find(([re]) => re.test(repo))?.[1]
}

/**
 * A card's logo from its images: none when any image is unknown (the user's
 * own app), else the first by RULES order, which puts the product ahead of
 * the runtime, proxy and database that serve it.
 */
export function groupTech(g: Group): Tech | undefined {
  const techs = g.containers.map((c) => imageTech(c.image))
  if (techs.length === 0 || techs.some((t) => t === undefined)) return undefined
  const rank = (t: Tech) => RULES.findIndex(([, x]) => x === t)
  return (techs as Tech[]).sort((a, b) => rank(a) - rank(b))[0]
}
