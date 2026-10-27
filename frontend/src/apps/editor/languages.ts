// Which syntax a server file gets, from its path and first line. Config
// files often have no extension, so the folder and shebang count too.

export type LanguageId = 'nginx' | 'yaml' | 'json' | 'shell' | 'env' | 'conf' | 'markdown' | 'plain'

export const LANGUAGE_LABELS: Record<LanguageId, string> = {
  nginx: 'Nginx',
  yaml: 'YAML',
  json: 'JSON',
  shell: 'Shell',
  env: 'Env',
  conf: 'Config',
  markdown: 'Markdown',
  plain: 'Text',
}

const SHELL_NAMES = new Set([
  '.bashrc',
  '.bash_profile',
  '.bash_aliases',
  '.bash_logout',
  '.profile',
  '.zshrc',
  '.zprofile',
  '.zshenv',
])

function ext(name: string): string {
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1) : ''
}

export function detectLanguage(path: string, firstLine = ''): LanguageId {
  const name = path.slice(path.lastIndexOf('/') + 1).toLowerCase()
  const e = ext(name)
  if (e === 'json') return 'json'
  if (e === 'yml' || e === 'yaml') return 'yaml'
  if (e === 'md' || e === 'markdown') return 'markdown'
  if (e === 'sh' || e === 'bash' || e === 'zsh' || SHELL_NAMES.has(name)) return 'shell'
  if (name === '.env' || name.startsWith('.env.') || e === 'env') return 'env'
  // nginx keeps sites without an extension under its own folder.
  if (name === 'nginx.conf' || e === 'nginx' || /\/nginx\//.test(path)) return 'nginx'
  if (/^#!.*\b(ba|z|da|k)?sh\b/.test(firstLine)) return 'shell'
  if (['conf', 'cfg', 'ini', 'cnf', 'properties'].includes(e) || name.endsWith('_config')) {
    return 'conf'
  }
  // Files directly under /etc without an extension are almost always
  // "key value" config with # comments (hosts, fstab, sysctl…).
  if (e === '' && path.startsWith('/etc/')) return 'conf'
  return 'plain'
}
