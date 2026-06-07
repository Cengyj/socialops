import { beforeEach, describe, expect, it, vi } from 'vitest'

const { put } = vi.hoisted(() => ({
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    put,
  },
}))

import promoAPI from '@/api/admin/promo'

describe('admin promo api', () => {
  beforeEach(() => {
    put.mockReset()
  })

  it('keeps zero expires_at in update payload so the backend can clear expiry', async () => {
    const response = {
      id: 7,
      code: 'WELCOME',
      bonus_amount: 5,
      max_uses: 10,
      used_count: 1,
      status: 'active',
      expires_at: null,
      notes: 'initial',
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-04T00:00:00Z',
    }
    const payload = { expires_at: 0 }
    put.mockResolvedValue({ data: response })

    const result = await promoAPI.update(7, payload)

    expect(put).toHaveBeenCalledWith('/admin/promo-codes/7', payload)
    expect(result).toEqual(response)
  })
})
