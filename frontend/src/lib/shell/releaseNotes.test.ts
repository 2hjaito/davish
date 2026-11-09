import { describe, expect, it } from 'vitest'

import { releaseNotes } from './releaseNotes'

const body = [
  '### Fixed',
  '- The window moves.',
  '',
  '<!-- davish:vi -->',
  '<details><summary>Tiếng Việt</summary>',
  '',
  '### Sửa',
  '- Cửa sổ di chuyển được.',
  '',
  '</details>',
  '',
  '---',
  '[Code signing policy](https://example.com)',
].join('\n')

describe('releaseNotes', () => {
  it('gives English to an English interface', () => {
    expect(releaseNotes(body, 'en').trim()).toBe('### Fixed\n- The window moves.')
  })

  it('gives Vietnamese, without the fold, to a Vietnamese interface', () => {
    expect(releaseNotes(body, 'vi')).toBe('### Sửa\n- Cửa sổ di chuyển được.')
  })

  it('falls back to English for languages without notes of their own', () => {
    expect(releaseNotes(body, 'zh-CN')).toContain('The window moves.')
  })

  it('keeps an older one-language release whole', () => {
    expect(releaseNotes('### Sửa\n- x', 'en')).toBe('### Sửa\n- x')
  })
})
