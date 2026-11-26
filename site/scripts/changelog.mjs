// Prepares the repo's changelogs for the site (run before dev and build):
// copies CHANGELOG.md and CHANGELOG.en.md to src/generated/, leaving out
// the Unreleased section, turning #123 into a link to the issue and @name
// into a link to the person, and pointing links to the other changelog at
// the site's own page. The release is written once, in the repo.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'

const REPO = 'https://github.com/2hjaito/davish'
const ISSUE = /(?<![\w&/#])#(\d{1,5})(?!\w)/g
const PERSON = /(?<![\w@/.])@([A-Za-z0-9][A-Za-z0-9-]{0,38})(?![\w-])/g
// Parts that must stay as written: links, code, bare addresses and tags.
const KEEP = /(\[[^\]]*\]\([^)]*\)|`[^`]*`|https?:\/\/\S+|<[^>]*>)/

function link(text) {
  return text
    .replace(ISSUE, (m, n) => `[${m}](${REPO}/issues/${n})`)
    .replace(PERSON, (m, who) => `[${m}](https://github.com/${who})`)
}

function line(text) {
  return text
    .split(KEEP)
    .map((part, i) => (i % 2 ? part : link(part)))
    .join('')
}

function ownLinks(text) {
  return text
    .replace(/\]\(CHANGELOG\.md\)/g, '](/vi/changelog/)')
    .replace(/\]\(CHANGELOG\.en\.md\)/g, '](/changelog/)')
}

export function prepare(source) {
  const out = []
  let fence = false
  let skipping = false
  for (const raw of source.split('\n')) {
    if (/^## /.test(raw)) skipping = /unreleased/i.test(raw)
    if (skipping) continue
    if (/^\s*```/.test(raw)) fence = !fence
    // The page has its own title.
    if (/^# /.test(raw)) continue
    // "## [1.15.1] - 2026-11-24" reads better without the brackets.
    if (/^## \[/.test(raw)) out.push(raw.replace(/^## \[([^\]]+)\]/, '## $1'))
    else out.push(fence || /^\s*```/.test(raw) || /^#{1,6} /.test(raw) ? raw : line(ownLinks(raw)))
  }
  return out.join('\n')
}

const root = new URL('../../', import.meta.url)
const dir = new URL('../src/generated/', import.meta.url)
mkdirSync(dir, { recursive: true })
for (const [from, to] of [
  ['CHANGELOG.en.md', 'changelog.en.md'],
  ['CHANGELOG.md', 'changelog.vi.md'],
]) {
  writeFileSync(new URL(to, dir), prepare(readFileSync(new URL(from, root), 'utf8')))
}
