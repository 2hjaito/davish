// Deploy recipes live on this machine (settings.json), per server and
// repository: never written to the server. They share one setting, keyed
// by server and folder, as setting names cannot hold a path.
import { load, save } from '../../lib/stores/persist'

const SETTING = 'deploy-recipes'
const key = (serverID: string, dir: string) => `${serverID}:${dir}`

/** The saved steps, or a first recipe from what the repository holds. */
export function loadRecipe(serverID: string, dir: string, install: string): string[] {
  const saved = load<Record<string, string[]>>(SETTING, {})[key(serverID, dir)]
  if (saved && saved.length > 0) return saved
  return [
    'git pull --ff-only',
    ...(install ? [install] : []),
    '# npm run build',
    '# sudo systemctl restart my-app',
    '# docker compose up -d --build',
  ]
}

export function saveRecipe(serverID: string, dir: string, steps: string[]): void {
  save(SETTING, { ...load<Record<string, string[]>>(SETTING, {}), [key(serverID, dir)]: steps })
}
