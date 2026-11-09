import { describe, expect, it } from 'vitest'

import { updateLogLines, updateReport } from './updateReport'

const tail = [
  '{"time":"t1","level":"INFO","msg":"connected","server":"a"}',
  '{"time":"t2","level":"INFO","msg":"update download retry","api":false,"try":1}',
  '{"time":"t3","level":"WARN","msg":"update failed","version":"1.8.1","err":"x"}',
  '',
].join('\n')

describe('updateLogLines', () => {
  it('keeps only the updater lines', () => {
    expect(updateLogLines(tail)).toEqual([
      '{"time":"t2","level":"INFO","msg":"update download retry","api":false,"try":1}',
      '{"time":"t3","level":"WARN","msg":"update failed","version":"1.8.1","err":"x"}',
    ])
  })

  it('keeps the newest when there are too many', () => {
    expect(updateLogLines(tail, 1)).toHaveLength(1)
    expect(updateLogLines(tail, 1)[0]).toContain('t3')
  })
})

describe('updateReport', () => {
  it('names the version, the error and the log lines', () => {
    const text = updateReport(
      { version: '1.8.0', os: 'darwin/arm64', logPath: '/l/davish.log', logTail: tail },
      'UPDATE_DOWNLOAD_FAILED: download answered 504',
    )
    expect(text).toContain('Davish 1.8.0 · darwin/arm64')
    expect(text).toContain('Error: UPDATE_DOWNLOAD_FAILED: download answered 504')
    expect(text).toContain('/l/davish.log')
    expect(text).not.toContain('connected')
  })

  it('works without support info, as in the browser preview', () => {
    expect(updateReport(null, 'E: x')).toBe('Error: E: x')
  })
})
