import { describe, expect, it } from 'vitest'

import { formatAccountWorkbenchDate } from '../accountWorkbenchDate'

describe('account workbench date formatting', () => {
  it('keeps existing locale date formatting for valid values', () => {
    const value = '2026-06-01T00:00:00Z'
    expect(formatAccountWorkbenchDate(value)).toBe(new Date(value).toLocaleString())
  })

  it('uses a compact fallback for missing or invalid dates', () => {
    expect(formatAccountWorkbenchDate()).toBe('-')
    expect(formatAccountWorkbenchDate(null)).toBe('-')
    expect(formatAccountWorkbenchDate('not-a-date')).toBe('-')
  })
})
