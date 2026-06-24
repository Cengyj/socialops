import { describe, expect, it } from 'vitest'

import { proxyTestResultSummary } from '../proxyTestSummary'

describe('proxyTestResultSummary', () => {
  it('counts normalized proxy connectivity statuses', () => {
    expect(proxyTestResultSummary([
      { status: ' online ' },
      { status: 'OFFLINE' },
      { status: 'timeout' },
      { status: null },
    ])).toEqual({
      total: 4,
      online: 1,
      offline: 1,
      unknown: 2,
    })
  })

  it('returns an empty summary for empty result lists', () => {
    expect(proxyTestResultSummary([])).toEqual({
      total: 0,
      online: 0,
      offline: 0,
      unknown: 0,
    })
  })
})
