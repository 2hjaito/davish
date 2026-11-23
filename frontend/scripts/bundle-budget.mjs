// Fails when the built JS grows past its budget (.claude/docs/PLAN.md M21). The
// initial bundle is what the first screen loads: the entry, the shell it
// imports right away (App.svelte) and their static imports. App windows
// load later, on first use, and count only toward the total. A run loads one
// language (ADR 0010), so the total counts the largest of them, not all.
import { readFileSync } from 'node:fs'
import process from 'node:process'
import { gzipSync } from 'node:zlib'

const BUDGET = {
  initialKB: 118, // gzip; 96 when set; the performance presets (#176) are read at start, before the first paint (~1.8); the lock screen shows the user's picture and name (~1); Check again recommends a preset and the wallpaper takes any color (~0.9); Terminal's solid background and scrollback live in the settings store, read at start (~0.1); M53 widget frame: the layout and store the desktop icons read to keep clear of widgets, and the chunk split from loading the layer and each widget on first use (~3.2)
  totalKB: 714, // gzip; 506 when set, 580 by v1.2; v1.3 added windows loaded on first use; the Docker cards' tech and framework logos (~28) load with Docker; M47 server search in Files and Logs (~4); M49 run on several servers (~6), with room for M48 Ansible import (~2); performance presets (~4); M51 SQLite rows with sort and filter, loaded with Database (~4); Settings as System Settings panes with search and drawn icons, loaded with Settings (~3); Git account page and one-click clone, Settings account and About panes (~3); wallpaper picker, preset recommendation and the account header's inline edits (~1); M54 Terminal color schemes with their tiles in Settings (~1.5); Terminal font and cursor, Editor font size, wrapping, line numbers, whitespace marks and indent (~2.2); M53 widget frame, its layer and the Monitor widget (~4.9)
}

const dist = new URL('../dist/', import.meta.url)
const manifest = JSON.parse(readFileSync(new URL('.vite/manifest.json', dist), 'utf8'))

const gz = (file) => gzipSync(readFileSync(new URL(file, dist))).length / 1024

const initial = new Set()
function walk(key) {
  const chunk = manifest[key]
  if (!chunk || initial.has(chunk.file)) return
  initial.add(chunk.file)
  for (const k of chunk.imports ?? []) walk(k)
}
walk('index.html')
walk('src/App.svelte')

const js = new Set(
  Object.values(manifest)
    .map((c) => c.file)
    .filter((f) => f.endsWith('.js')),
)
const sum = (files) => [...files].filter((f) => f.endsWith('.js')).reduce((s, f) => s + gz(f), 0)
const locales = Object.entries(manifest)
  .filter(([key]) => /^src\/lib\/i18n\/[a-z]+\.ts$/.test(key) && !key.endsWith('index.ts'))
  .map(([, c]) => c.file)
for (const f of locales) js.delete(f)
const initialKB = sum(initial)
const totalKB = sum(js) + Math.max(0, ...locales.map(gz))

console.log(`initial JS: ${initialKB.toFixed(1)} kB gzip (budget ${BUDGET.initialKB})`)
console.log(`total JS:   ${totalKB.toFixed(1)} kB gzip (budget ${BUDGET.totalKB})`)
if (initialKB > BUDGET.initialKB || totalKB > BUDGET.totalKB) {
  console.error('JS bundle over budget')
  process.exit(1)
}
