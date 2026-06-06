import { describe, expect, it } from 'vitest'

import { formatSafeUserTaskResult } from '../socialTaskResult'

describe('formatSafeUserTaskResult', () => {
  it('keeps concise business results readable', () => {
    expect(formatSafeUserTaskResult('follow succeeded', 'hidden')).toBe('follow succeeded')
  })

  it('hides internal diagnostic and sensitive response details', () => {
    const raw = 'upstream response body authorization Bearer abc token=secret proxy http://user:pass@127.0.0.1:8080 trace_id=trace-123'

    expect(formatSafeUserTaskResult(raw, 'Task failed; details hidden')).toBe('Task failed; details hidden')
  })

  it('normalizes whitespace and caps long results', () => {
    const raw = `ok ${'x'.repeat(200)}`

    expect(formatSafeUserTaskResult(raw)).toHaveLength(163)
    expect(formatSafeUserTaskResult('  ok \n done  ')).toBe('ok done')
  })
})
