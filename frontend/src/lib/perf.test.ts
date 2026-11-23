import { describe, expect, it } from 'vitest'

import {
  frameP95,
  framesPerSecond,
  knobsEqual,
  knobsFromStored,
  machineIsSlow,
  migratePerf,
  PRESET_KNOBS,
  presetMatching,
  recommendPreset,
  SLOW_FRAME_MS,
  SMOOTH_FRAME_MS,
} from './perf'

describe('presetMatching', () => {
  it('recognizes each named preset', () => {
    expect(presetMatching(PRESET_KNOBS.beautiful)).toBe('beautiful')
    expect(presetMatching(PRESET_KNOBS.balanced)).toBe('balanced')
    expect(presetMatching(PRESET_KNOBS.light)).toBe('light')
  })

  it('is custom when one knob leaves the preset', () => {
    expect(presetMatching({ ...PRESET_KNOBS.light, wallpaper: true })).toBe('custom')
    expect(presetMatching({ ...PRESET_KNOBS.beautiful, noDmabuf: true })).toBe('custom')
  })
})

describe('the wallpaperImage knob', () => {
  it('is on in Beautiful and Balanced and off in Light', () => {
    expect(PRESET_KNOBS.beautiful.wallpaperImage).toBe(true)
    expect(PRESET_KNOBS.balanced.wallpaperImage).toBe(true)
    expect(PRESET_KNOBS.light.wallpaperImage).toBe(false)
  })

  it('is part of what makes a preset', () => {
    expect(presetMatching({ ...PRESET_KNOBS.light, wallpaperImage: true })).toBe('custom')
    expect(presetMatching({ ...PRESET_KNOBS.balanced, wallpaperImage: false })).toBe('custom')
  })

  it('is on for a file written before pictures existed, and read back when stored', () => {
    expect(knobsFromStored({}).wallpaperImage).toBe(true)
    expect(knobsFromStored({ perfWallpaperImage: false }).wallpaperImage).toBe(false)
    expect(knobsFromStored({ perfWallpaperImage: 'no' }).wallpaperImage).toBe(true)
  })
})

describe('the widgets knob', () => {
  it('is on in Beautiful and Balanced and off in Light', () => {
    expect(PRESET_KNOBS.beautiful.widgets).toBe(true)
    expect(PRESET_KNOBS.balanced.widgets).toBe(true)
    expect(PRESET_KNOBS.light.widgets).toBe(false)
  })

  it('is part of what makes a preset: Light with widgets on is custom', () => {
    expect(presetMatching({ ...PRESET_KNOBS.light, widgets: true })).toBe('custom')
    expect(presetMatching({ ...PRESET_KNOBS.beautiful, widgets: false })).toBe('custom')
    expect(knobsEqual(PRESET_KNOBS.light, { ...PRESET_KNOBS.light, widgets: true })).toBe(false)
  })

  it('is on for a file written before widgets existed, and read back when stored', () => {
    expect(knobsFromStored({}).widgets).toBe(true)
    expect(knobsFromStored({ perfWidgets: false }).widgets).toBe(false)
    expect(knobsFromStored({ perfWidgets: 'no' }).widgets).toBe(true)
  })

  it('keeps a saved named preset as it is', () => {
    expect(migratePerf({ perfPreset: 'light' }).knobs.widgets).toBe(false)
    expect(migratePerf({ perfPreset: 'balanced' }).knobs.widgets).toBe(true)
    expect(migratePerf({ reduceGlass: true }).knobs.widgets).toBe(true)
  })
})

describe('migratePerf', () => {
  it('starts a new install on Auto, painted as Beautiful', () => {
    const got = migratePerf(null)
    expect(got.preset).toBe('auto')
    expect(knobsEqual(got.knobs, PRESET_KNOBS.beautiful)).toBe(true)
    expect(got.later).toBe(false)
  })

  it('keeps a chosen preset', () => {
    expect(migratePerf({ perfPreset: 'light' }).preset).toBe('light')
    expect(knobsEqual(migratePerf({ perfPreset: 'light' }).knobs, PRESET_KNOBS.light)).toBe(true)
  })

  it('turns the old Reduce glass toggle into custom knobs', () => {
    const got = migratePerf({ reduceGlass: true })
    expect(got.preset).toBe('custom')
    expect(got.knobs.blur).toBe(false)
    expect(got.knobs.opaque).toBe(true)
    expect(got.knobs.shadow).toBe('short')
    expect(got.knobs.wallpaper).toBe(true)
    expect(got.knobs.motion).toBe(true)
  })

  it('keeps Reduce motion with the frosted look', () => {
    const got = migratePerf({ reduceMotion: true })
    expect(got.preset).toBe('custom')
    expect(got.knobs.motion).toBe(false)
    expect(got.knobs.blur).toBe(true)
  })
})

describe('machineIsSlow', () => {
  const fast = Array.from({ length: 40 }, () => 16)
  const slow = Array.from({ length: 40 }, () => SLOW_FRAME_MS + 10)

  it('treats a software renderer as slow', () => {
    expect(machineIsSlow('llvmpipe (LLVM 15)', fast)).toBe(true)
    expect(machineIsSlow('ANGLE (SwiftShader)', [])).toBe(true)
  })

  it('uses the frame tail when a GPU is named', () => {
    expect(machineIsSlow('ANGLE (NVIDIA, GeForce)', fast)).toBe(false)
    expect(machineIsSlow('ANGLE (NVIDIA, GeForce)', slow)).toBe(true)
  })

  it('does not nag on a short sample', () => {
    expect(machineIsSlow('NVIDIA', [40, 40, 40])).toBe(false)
    expect(frameP95([10, 20, 30, 40])).toBe(30)
  })
})

describe('recommendPreset', () => {
  const frames = (ms: number) => Array.from({ length: 60 }, () => ms)

  it('is Beautiful when frames stay smooth', () => {
    expect(recommendPreset('Apple M2', frames(16.7))).toBe('beautiful')
  })

  it('is Balanced between smooth and slow', () => {
    expect(recommendPreset('Apple M2', frames(SMOOTH_FRAME_MS + 4))).toBe('balanced')
  })

  it('is Light when slow or drawn in software', () => {
    expect(recommendPreset('Apple M2', frames(SLOW_FRAME_MS + 5))).toBe('light')
    expect(recommendPreset('llvmpipe (LLVM 15.0.7, 256 bits)', frames(16.7))).toBe('light')
  })

  it('trusts a short sample to be smooth', () => {
    expect(recommendPreset('Apple M2', [40, 40])).toBe('beautiful')
  })
})

describe('framesPerSecond', () => {
  it('reads the median frame gap', () => {
    expect(framesPerSecond([16.7, 16.6, 16.7, 50])).toBe(60)
    expect(framesPerSecond([])).toBe(0)
  })
})
