import { beforeEach, describe, expect, expectTypeOf, it, vi } from 'vitest'
import type { RedeemCode } from '@/types'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { getHistory, redeem } from '@/api/redeem'

describe('redeem api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('matches the backend redeem-code DTO returned after redeeming', async () => {
    const payload: RedeemCode = {
      id: 7,
      code: 'BALANCE-1',
      type: 'balance',
      value: 10,
      status: 'used',
      used_by: 42,
      used_at: '2026-06-01T00:00:00Z',
      created_at: '2026-05-01T00:00:00Z',
      expires_at: null,
      group_id: null,
      plan_id: null,
      validity_days: 0,
    }
    post.mockResolvedValue({ data: payload })

    const result = await redeem('BALANCE-1')

    expect(post).toHaveBeenCalledWith('/redeem', { code: 'BALANCE-1' })
    expect(result).toEqual(payload)
    expectTypeOf(result).toEqualTypeOf<RedeemCode>()
  })

  it('loads redeem history from the user endpoint', async () => {
    const payload: RedeemCode[] = []
    get.mockResolvedValue({ data: payload })

    const result = await getHistory()

    expect(get).toHaveBeenCalledWith('/redeem/history')
    expect(result).toEqual(payload)
  })
})
