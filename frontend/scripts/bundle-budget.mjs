// Fails when the built JS grows past its budget (.claude/docs/PLAN.md M21). The
// initial bundle is what the first screen loads: the entry, the shell it
// imports right away (App.svelte) and their static imports. App windows
// load later, on first use, and count only toward the total. A run loads one
// language (ADR 0010), so the total counts the largest of them, not all.
import { readFileSync } from 'node:fs'
import process from 'node:process'
import { gzipSync } from 'node:zlib'

const BUDGET = {
  initialKB: 128, // gzip; 96 when set; the performance presets (#176) are read at start, before the first paint (~1.8); the lock screen shows the user's picture and name (~1); Check again recommends a preset and the wallpaper takes any color (~0.9); Terminal's solid background and scrollback live in the settings store, read at start (~0.1); M53 widget frame: the layout and store the desktop icons read to keep clear of widgets, and the chunk split from loading the layer and each widget on first use (~3.2); the Widgets app: its drawn icon, its entry and its words (~1.4); the Docker health widget: its entry and its words (~0.8); the Services widget: its entry and its words (~0.5); server tags: the chip, the picker's filter and the title bar's tag, which the shell shows at start (~1.2); the server menu in folders and its ⌘1…⌘9 order, which the shell reads at start (~0.3); the system logo beside a server's name, whose small component and loader the shell draws at start (~0.7); the launch setting: its first-launch choice and words, read at start (~0.2); the window drag setting joins the launch setting in the store the shell reads at start (~0.3); the wallpaper picture: its knob, its store and the choice of the picture, read at start (~0.8); reordering the Dock by dragging: its press, drag and easing code and the slot maths, which the shell draws at start (~1.1)
  totalKB: 760, // gzip; 506 when set, 580 by v1.2; v1.3 added windows loaded on first use; the Docker cards' tech and framework logos (~28) load with Docker; M47 server search in Files and Logs (~4); M49 run on several servers (~6), with room for M48 Ansible import (~2); performance presets (~4); M51 SQLite rows with sort and filter, loaded with Database (~4); Settings as System Settings panes with search and drawn icons, loaded with Settings (~3); Git account page and one-click clone, Settings account and About panes (~3); wallpaper picker, preset recommendation and the account header's inline edits (~1); M54 Terminal color schemes with their tiles in Settings (~1.5); Terminal font and cursor, Editor font size, wrapping, line numbers, whitespace marks and indent (~2.2); M53 widget frame, its layer and the Monitor widget (~4.9); the Widgets app and its gallery (~3); the Docker health widget and its shared reading (~3); the widgets as dark cards with rings, grids of dots and area charts (~2); the Services widget and its bar (~3); the Disk widget (~2.5); server tags with their words in four languages (~1.5); server groups: All servers on the Finder layout with folders, loaded with that app (~1.6); the password switch in Security with its words and the key-checked change (~2.6); the 16 system logos of the server lists, a chunk loaded when the first server with a known system is drawn (~11.4); the window drag setting with the launch setting (~0.2); the wallpaper picture with its row in Settings and its words in four languages (~1.7); the list of files a cleanup would remove, with its words in four languages (~1.8); German, now the largest language and the one the total counts (~2.4)
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
