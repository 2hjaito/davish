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
import { gridOf } from '../shell/iconGrid'
import { openFilesAt } from '../wm/intents.svelte'
import { wm } from '../wm/windowManager.svelte'
import { addWidget } from '../widgets/actions'
import { measure, resultLine, summarize, watchLongTasks } from './frames'
import { widgetMix } from './mix'
import { widgets } from '../stores/widgets.svelte'
import { wallpaperImage } from '../stores/wallpaperImage.svelte'

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

/** What one process measures: a preset, with this many widgets on the desktop, with or without a picture as the wallpaper. */
interface Run {
  preset: Preset
  widgets: number
  picture: boolean
}

/**
 * Times one scenario. The two marks around it let the script read the CPU of
 * the process tree over just this window: the cost of widgets is small beside
 * a scroll or a drag, so a CPU figure for the whole run would bury it.
 */
async function scenario(run: Run, name: string, go: () => Promise<number[]>) {
  const stop = watchLongTasks()
  await benchReport(`mark phase=start scenario=${name}`)
  const t0 = performance.now()
  const gaps = await go()
  const ms = Math.round(performance.now() - t0)
  await benchReport(`mark phase=end scenario=${name} ms=${ms}`)
  await benchReport(resultLine(name, run.preset, run.widgets, summarize(gaps), stop(), run.picture))
}

/**
 * Takes every widget off every desktop. The browser preview keeps them in
 * localStorage, which outlives the process (WebKit does not follow the empty
 * HOME the script gives), so a run would start with the last run's widgets.
 */
function clearWidgets() {
  for (const id of Object.keys(widgets.byServer))
    for (const w of widgets.of(id)) widgets.remove(id, w.id)
}

/** Puts the first n widgets of the mix on the desktop; how many found room. */
function addWidgets(n: number, serverID: string): number {
  return widgetMix(n).filter(([kind, size]) => addWidget(serverID, kind, size, false)).length
}

/** Scrolls /usr/bin of the demo (3000 entries) at 1500 px/s, looping. */
async function scroll(run: Run, serverID: string) {
  openFilesAt(serverID, '/usr/bin')
  const list = await waitFor('the file list', () => {
    const el = document.querySelector<HTMLElement>('.scroller[role="listbox"]')
    return el && el.scrollHeight > el.clientHeight * 2 ? el : null
  })
  const max = list.scrollHeight - list.clientHeight
  await scenario(run, 'scroll', () =>
    measure(SCENARIO_MS, (t) => {
      list.scrollTop = (t * 1.5) % max
    }),
  )
}

/** Moves the Files window in a slow loop, as dragging it would. */
async function drag(run: Run, serverID: string) {
  const win = await waitFor('the Files window', () =>
    wm.visible(serverID).find((w) => w.appId === 'files'),
  )
  wm.focus(win.id)
  const base = { ...wm.frame(win) }
  await scenario(run, 'drag', () =>
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

/**
 * A busy 2560 x 1440 picture, drawn here so the bench needs no file: soft
 * gradients with many small shapes, which a JPEG keeps at the size of a photo.
 */
async function makePicture(): Promise<string> {
  const c = document.createElement('canvas')
  c.width = 2560
  c.height = 1440
  const g = c.getContext('2d')
  if (!g) throw new Error('no canvas')
  const sky = g.createLinearGradient(0, 0, 2560, 1440)
  sky.addColorStop(0, '#1f3b73')
  sky.addColorStop(0.5, '#c46a8f')
  sky.addColorStop(1, '#f3b562')
  g.fillStyle = sky
  g.fillRect(0, 0, 2560, 1440)
  // A fixed sequence, so every run decodes the same picture.
  let seed = 7
  const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647
  for (let i = 0; i < 1500; i++) {
    g.fillStyle = `hsla(${Math.round(rnd() * 360)}, 60%, ${30 + Math.round(rnd() * 40)}%, 0.25)`
    g.beginPath()
    g.arc(rnd() * 2560, rnd() * 1440, 6 + rnd() * 90, 0, Math.PI * 2)
    g.fill()
  }
  return c.toDataURL('image/jpeg', 0.85)
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
      `env preset=${preset} widgets=${request.widgets} picture=${request.picture ? 1 : 0} ` +
        `renderer=${(rendererName() || 'none').replaceAll(' ', '_')} ` +
        `dpr=${window.devicePixelRatio} size=${innerWidth}x${innerHeight}`,
    )
    settings.usePreset(preset)
    ui.tour = false
    const serverID = servers.activeId
    // Widgets are placed on the grid of the desktop, which has no size until the
    // shell has laid itself out.
    await waitFor('the desktop', () => wm.area.w > 400 && wm.area.h > 300)
    clearWidgets()
    // Placed even when the preset hides them (Light): the knob is what is measured then.
    const placed = addWidgets(request.widgets, serverID)
    if (request.picture) {
      await wallpaperImage.set(await makePicture())
      // Light hides the picture by its knob; the others show it.
      await waitFor('the wallpaper', () =>
        preset === 'light' ? true : document.documentElement.dataset.perfWallpaper === 'image',
      )
    }
    const run: Run = { preset, widgets: request.widgets, picture: request.picture }
    const grid = gridOf(wm.area)
    await benchReport(
      `env placed=${placed} grid=${grid.cols}x${grid.rows} server=${serverID || 'none'}`,
    )
    await sleep(SETTLE_MS)
    await scenario(run, 'idle', () => measure(SCENARIO_MS))
    await scroll(run, serverID)
    await drag(run, serverID)
    await benchReport('done')
  } catch (e) {
    await benchReport(`error=${(e instanceof Error ? e.message : String(e)).replaceAll(' ', '_')}`)
  } finally {
    // The preview keeps its state in localStorage, which the app's webview
    // shares: a bench must not leave its preset or widgets behind.
    try {
      if (request.picture) await wallpaperImage.clear()
      localStorage.clear()
    } catch {
      // Nothing to clear.
    }
    // The script reads the process tree's CPU once it sees the last line.
    await sleep(QUIT_DELAY_MS)
    benchQuit()
  }
}
