import { machineIsSlow, PROBE_MS } from '../perf'
import { settings } from '../stores/settings.svelte'

/** The WebGL renderer, or "" when WebGL is missing. */
export function rendererName(): string {
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl')
    if (!gl) return ''
    const ext = gl.getExtension('WEBGL_debug_renderer_info')
    if (!ext) return ''
    const name = gl.getParameter(ext.UNMASKED_RENDERER_WEBGL)
    return typeof name === 'string' ? name : ''
  } catch {
    return ''
  }
}

/** Frame gaps over ms milliseconds. The first gap is dropped. */
export function sampleFrames(ms: number): Promise<number[]> {
  return new Promise((resolve) => {
    const samples: number[] = []
    let last = performance.now()
    const start = last
    const tick = (now: number) => {
      samples.push(now - last)
      last = now
      if (now - start < ms) requestAnimationFrame(tick)
      else resolve(samples.slice(1))
    }
    requestAnimationFrame(tick)
  })
}

class PerfCheck {
  /** Show the banner that offers the Light preset. */
  suggestion = $state(false)
  running = $state(false)
  /** Result of the latest check, for the Settings line. */
  last = $state<'idle' | 'slow' | 'fine'>('idle')

  /**
   * Times about 1.5s of frames without blocking the UI. manual is Check
   * again: a slow result is offered even after Later.
   */
  async start(manual: boolean) {
    if (this.running) return
    this.running = true
    if (manual) this.suggestion = false
    try {
      const renderer = rendererName()
      const frames = await sampleFrames(PROBE_MS)
      const slow = machineIsSlow(renderer, frames)
      this.last = slow ? 'slow' : 'fine'
      const offer = slow && settings.perfPreset !== 'light' && (manual || !settings.perfLater)
      this.suggestion = offer
    } finally {
      this.running = false
    }
  }

  hide() {
    this.suggestion = false
  }
}

export const perfCheck = new PerfCheck()
