import { describe, expect, it } from 'vitest'

import {
  frameP95,
  knobsEqual,
  machineIsSlow,
  migratePerf,
  PRESET_KNOBS,
  presetMatching,
  SLOW_FRAME_MS,
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
