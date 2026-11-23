// The render bench's way in (scripts/bench.sh --render). Outside a bench, and
// outside Wails, this asks the app once and does nothing more: the scenarios
// are loaded only when the script that started the process asked for them.
import { BenchConfig, BenchReport } from '../../../wailsjs/go/app/App'
import { Quit } from '../../../wailsjs/runtime/runtime'

export interface BenchRequest {
  render: boolean
  preset: string
}

/** What the script that started this process asked for; nothing in a normal run. */
export async function benchConfig(): Promise<BenchRequest> {
  if (typeof window === 'undefined' || !('go' in window)) return { render: false, preset: '' }
  try {
    return await BenchConfig()
  } catch {
    return { render: false, preset: '' }
  }
}

/** One result line to the script, through the app's stderr. */
export function benchReport(line: string): Promise<void> {
  return 'go' in window ? BenchReport(line) : Promise.resolve()
}

/** Ends the process once the scenarios are done. Not `quit()`: demo mode counts as no runtime there. */
export function benchQuit() {
  if (typeof window !== 'undefined' && 'runtime' in window) Quit()
}

/** Runs the render scenarios if this process was started to; see lib/bench/render.ts. */
export async function startBench() {
  const request = await benchConfig()
  if (request.render) await (await import('../bench/render')).runRenderBench(request)
}
