import { describe, expect, it } from 'vitest'

import { normalizeProxyStatus } from '../proxyStatus'

describe('proxyStatus utilities', () => {
  it('normalizes known proxy statuses and falls back to unknown', () => {
    expect(normalizeProxyStatus(' ONLINE ')).toBe('online')
    expect(normalizeProxyStatus('offline')).toBe('offline')
    expect(normalizeProxyStatus('Unknown')).toBe('unknown')
    expect(normalizeProxyStatus('degraded')).toBe('unknown')
    expect(normalizeProxyStatus(null)).toBe('unknown')
  })
})
