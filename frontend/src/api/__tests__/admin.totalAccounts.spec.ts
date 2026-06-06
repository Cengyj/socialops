import { beforeEach, describe, expect, it, vi } from 'vitest'

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

import { totalAccountsAPI } from '@/api/admin/totalAccounts'

describe('admin total accounts api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('lists total account pool records through the total account pool endpoint', async () => {
    const page = {
      items: [{ id: 7, name: '@pool', platform: 'x_twitter' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await totalAccountsAPI.list({ page: 1, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/admin/total-accounts', { params: { page: 1, page_size: 20 } })
    expect(result).toEqual(page)
  })

  it('assigns ownership through the total account pool endpoint', async () => {
    const account = { id: 7, name: '@pool', platform: 'x_twitter' }
    post.mockResolvedValue({ data: account })

    const result = await totalAccountsAPI.assign(7, 42)

    expect(post).toHaveBeenCalledWith('/admin/total-accounts/7/assign', { user_id: 42 })
    expect(result).toEqual(account)
  })

  it('reclaims ownership through the total account pool endpoint', async () => {
    const account = { id: 7, name: '@pool', platform: 'x_twitter' }
    post.mockResolvedValue({ data: account })

    const result = await totalAccountsAPI.reclaim(7)

    expect(post).toHaveBeenCalledWith('/admin/total-accounts/7/reclaim')
    expect(result).toEqual(account)
  })
})
