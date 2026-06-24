import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import dashboardAPI from '@/api/admin/dashboard'

describe('admin dashboard api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads SocialOps dashboard stats through the current admin endpoint', async () => {
    const stats = {
      total_users: 42,
      total_accounts: 12,
      normal_accounts: 7,
      total_operations: 120,
      today_operations: 17,
      total_charged: 18.75,
      today_charged: 2.4,
      recent_operations_per_minute: 2,
    }
    get.mockResolvedValue({ data: stats })

    const result = await dashboardAPI.getStats()

    expect(get).toHaveBeenCalledWith('/admin/dashboard/stats')
    expect(result).toEqual(stats)
    expect(result).not.toHaveProperty('total_api_keys')
    expect(result).not.toHaveProperty('total_tokens')
    expect(result).not.toHaveProperty('total_requests')
    expect(result).not.toHaveProperty('total_actual_cost')
    expect(result).not.toHaveProperty('rpm')
    expect(result).not.toHaveProperty('total_cost')
    expect(result).not.toHaveProperty('tpm')
  })

  it('returns the dashboard task trend array without removed wrapper normalization', async () => {
    const trend = [
      { date: '2026-06-01', operations: 9, charged: 1.1 },
      { date: '2026-06-02', operations: 17, charged: 2.4 },
    ]
    get.mockResolvedValue({ data: trend })

    const result = await dashboardAPI.getUsageTrend({ granularity: 'day' })

    expect(get).toHaveBeenCalledWith('/admin/dashboard/trend', { params: { granularity: 'day' } })
    expect(result).toEqual(trend)
    expect(result[0]).not.toHaveProperty('requests')
    expect(result[0]).not.toHaveProperty('actual_cost')
    expect(result[0]).not.toHaveProperty('total_tokens')
    expect(result[0]).not.toHaveProperty('cost')
  })

  it('does not unwrap the removed dashboard trend wrapper shape', async () => {
    const removedWrapperShape = {
      trend: [{ date: '2026-06-01', operations: 9, charged: 1.1 }],
    }
    get.mockResolvedValue({ data: removedWrapperShape })

    const result = await dashboardAPI.getUsageTrend({ granularity: 'day' })

    expect(result).toEqual(removedWrapperShape)
  })

  it('returns the user task trend array without removed wrapper normalization', async () => {
    const trend = [
      {
        date: '2026-06-01',
        user_id: 7,
        email: 'operator@example.com',
        username: 'operator',
        operations: 9,
        charged: 1.1,
      },
    ]
    get.mockResolvedValue({ data: trend })

    const result = await dashboardAPI.getUserUsageTrend({ limit: 5 })

    expect(get).toHaveBeenCalledWith('/admin/dashboard/users-trend', { params: { limit: 5 } })
    expect(result).toEqual(trend)
    expect(result[0]).not.toHaveProperty('requests')
    expect(result[0]).not.toHaveProperty('actual_cost')
    expect(result[0]).not.toHaveProperty('tokens')
    expect(result[0]).not.toHaveProperty('cost')
  })

  it('does not unwrap the removed user trend wrapper shape', async () => {
    const removedWrapperShape = {
      trend: [{
        date: '2026-06-01',
        user_id: 7,
        email: 'operator@example.com',
        operations: 9,
        charged: 1.1,
      }],
    }
    get.mockResolvedValue({ data: removedWrapperShape })

    const result = await dashboardAPI.getUserUsageTrend({ limit: 5 })

    expect(result).toEqual(removedWrapperShape)
  })

  it('returns the user spending ranking object without array fallback normalization', async () => {
    const ranking = {
      ranking: [
        {
          user_id: 7,
          email: 'operator@example.com',
          operations: 11,
          charged: 3.5,
        },
      ],
      total_charged: 3.5,
      total_operations: 11,
    }
    get.mockResolvedValue({ data: ranking })

    const result = await dashboardAPI.getUserSpendingRanking({ limit: 5 })

    expect(get).toHaveBeenCalledWith('/admin/dashboard/users-ranking', { params: { limit: 5 } })
    expect(result).toEqual(ranking)
    expect(result).not.toHaveProperty('total_actual_cost')
    expect(result).not.toHaveProperty('total_requests')
    expect(result).not.toHaveProperty('total_tokens')
    expect(result.ranking[0]).not.toHaveProperty('requests')
    expect(result.ranking[0]).not.toHaveProperty('actual_cost')
    expect(result.ranking[0]).not.toHaveProperty('tokens')
    expect(result.ranking[0]).not.toHaveProperty('username')
  })

  it('does not convert the removed ranking array fallback shape', async () => {
    const removedArrayFallbackShape = [
      {
        user_id: 7,
        email: 'operator@example.com',
        operations: 11,
        charged: 3.5,
      },
    ]
    get.mockResolvedValue({ data: removedArrayFallbackShape })

    const result = await dashboardAPI.getUserSpendingRanking({ limit: 5 })

    expect(result).toEqual(removedArrayFallbackShape)
  })
})
