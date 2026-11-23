// The render bench's scenarios (scripts/bench.sh --render), run inside the
// built app on the demo's sample data, so no server is touched. The script
// starts a fresh process per preset; this starts demo mode, sets the preset
// and times three things a person notices on a slow machine: nothing
// happening (idle), a long list scrolling and a window being dragged. Results
// leave through the app's stderr, one line each, and the app quits.
import { benchQuit, benchReport, type BenchRequest } from '../api/bench'
import { isDemo, startDemo } from '../api/runtime'
import { rendererName } from '../shell/perfCheck.svelte'
import { servers } from '../stores/servers.svelte'
import { settings } from '../stores/settings.svelte'
import { ui } from '../stores/ui.svelte'
import { openFilesAt } from '../wm/intents.svelte'
import { wm } from '../wm/windowManager.svelte'
import { measure, resultLine, summarize, watchLongTasks } from './frames'

const SCENARIO_MS = 3000
/** Time for the look to settle after a preset is applied. */
const SETTLE_MS = 2000
/** Time the script has to read the process tree after the last result line. */
const QUIT_DELAY_MS = 1000
const PRESETS = ['beautiful', 'balanced', 'light'] as const
type Preset = (typeof PRESETS)[number]

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

/** Polls until get returns something; throws after 5 s. */
async function waitFor<T>(what: string, get: () => T | null | undefined | false): Promise<T> {
  for (let i = 0; i < 100; i++) {
    const v = get()
    if (v) return v
    await sleep(50)
  }
  throw new Error(`timed out waiting for ${what}`)
}

async function scenario(name: string, preset: Preset, run: () => Promise<number[]>) {
  const stop = watchLongTasks()
  const gaps = await run()
  await benchReport(resultLine(name, preset, summarize(gaps), stop()))
}

/** Scrolls /usr/bin of the demo (3000 entries) at 1500 px/s, looping. */
async function scroll(preset: Preset, serverID: string) {
  openFilesAt(serverID, '/usr/bin')
  const list = await waitFor('the file list', () => {
    const el = document.querySelector<HTMLElement>('.scroller[role="listbox"]')
    return el && el.scrollHeight > el.clientHeight * 2 ? el : null
  })
  const max = list.scrollHeight - list.clientHeight
  await scenario('scroll', preset, () =>
    measure(SCENARIO_MS, (t) => {
      list.scrollTop = (t * 1.5) % max
    }),
  )
}

/** Moves the Files window in a slow loop, as dragging it would. */
async function drag(preset: Preset, serverID: string) {
  const win = await waitFor('the Files window', () =>
    wm.visible(serverID).find((w) => w.appId === 'files'),
  )
  wm.focus(win.id)
  const base = { ...wm.frame(win) }
  await scenario('drag', preset, () =>
    measure(SCENARIO_MS, (t) =>
      wm.setRect(win.id, {
        ...base,
        x: base.x + 160 * Math.sin(t / 350),
        y: base.y + 60 * Math.sin(t / 500),
      }),
    ),
  )
  wm.setRect(win.id, base)
}

export async function runRenderBench(request: BenchRequest) {
  // The demo is where the sample data is; this restarts the page in it, and
  // the app comes back here with isDemo set.
  if (!isDemo) {
    startDemo('en')
    return
  }
  const preset = PRESETS.find((p) => p === request.preset) ?? 'beautiful'
  try {
    await benchReport(
      `env preset=${preset} renderer=${(rendererName() || 'none').replaceAll(' ', '_')} ` +
        `dpr=${window.devicePixelRatio} size=${innerWidth}x${innerHeight}`,
    )
    settings.usePreset(preset)
    ui.tour = false
    await sleep(SETTLE_MS)
    const serverID = servers.activeId
    await scenario('idle', preset, () => measure(SCENARIO_MS))
    await scroll(preset, serverID)
    await drag(preset, serverID)
    await benchReport('done')
  } catch (e) {
    await benchReport(`error=${(e instanceof Error ? e.message : String(e)).replaceAll(' ', '_')}`)
  } finally {
    // The script reads the process tree's CPU once it sees the last line.
    await sleep(QUIT_DELAY_MS)
    benchQuit()
  }
}
