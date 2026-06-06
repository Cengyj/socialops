import { beforeEach, describe, expect, it, vi } from 'vitest'

const { del } = vi.hoisted(() => ({
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    delete: del,
  },
}))

import { clearUserSettings } from '@/api/admin/affiliates'

describe('admin affiliates api', () => {
  beforeEach(() => {
    del.mockReset()
  })

  it('clears custom affiliate settings through the existing admin delete endpoint', async () => {
    del.mockResolvedValue({ data: { user_id: 42 } })

    const result = await clearUserSettings(42)

    expect(del).toHaveBeenCalledWith('/admin/affiliates/users/42')
    expect(result).toEqual({ user_id: 42 })
  })
})
