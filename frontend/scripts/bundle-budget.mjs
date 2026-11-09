// Fails when the built JS grows past its budget (docs/PLAN.md M21). The
// initial bundle is what the first screen loads: the entry, the shell it
// imports right away (App.svelte) and their static imports. App windows
// load later, on first use, and count only toward the total. A run loads one
// language (ADR 0010), so the total counts the largest of them, not all.
import { readFileSync } from 'node:fs'
import process from 'node:process'
import { gzipSync } from 'node:zlib'

const BUDGET = {
  initialKB: 110, // gzip; 96 when set
  totalKB: 620, // gzip; 506 when set, 580 by v1.2; v1.3 added windows loaded on first use
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
