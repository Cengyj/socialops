import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { usageAPI } from '@/api/usage'

describe('user usage api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('unwraps paginated SocialOps usage records', async () => {
    const page = {
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          quantity: 1,
          cost: 0.1,
          charge_status: 'charged',
          created_at: '2026-06-01T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.list({ page: 1, page_size: 20, operation: 'follow' })

    expect(get).toHaveBeenCalledWith('/usage', { params: { page: 1, page_size: 20, operation: 'follow' } })
    expect(result).toEqual(page)
  })

  it('keeps query as the same usage list contract', async () => {
    const page = { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.query({ status: 'success' })

    expect(get).toHaveBeenCalledWith('/usage', { params: { status: 'success' } })
    expect(result).toEqual(page)
  })

  it('unwraps one usage record by id', async () => {
    const record = {
      id: 7,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'like',
      status: 'success',
      quantity: 1,
      cost: 0.1,
      charge_status: 'charged',
      created_at: '2026-06-01T00:00:00Z',
    }
    get.mockResolvedValue({ data: record })

    const result = await usageAPI.getById(7)

    expect(get).toHaveBeenCalledWith('/usage/7')
    expect(result).toEqual(record)
  })

  it('passes SocialOps usage stat filters to the stats endpoint', async () => {
    const stats = { total_requests: 3, total_actual_cost: 0.3 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getStats({ status: 'success', operation: 'follow' })

    expect(get).toHaveBeenCalledWith('/usage/stats', { params: { status: 'success', operation: 'follow' } })
    expect(result).toEqual(stats)
  })

  it('keeps the date-range stats helper on the same stats endpoint', async () => {
    const stats = { total_requests: 3, total_actual_cost: 0.3 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getStatsByDateRange('2026-06-01', '2026-06-02')

    expect(get).toHaveBeenCalledWith('/usage/stats', {
      params: {
        start_date: '2026-06-01',
        end_date: '2026-06-02',
      },
    })
    expect(result).toEqual(stats)
  })

  it('unwraps user dashboard usage stats', async () => {
    const stats = { total_requests: 12, today_requests: 3, total_actual_cost: 1.2 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getDashboardStats()

    expect(get).toHaveBeenCalledWith('/usage/dashboard/stats')
    expect(result).toEqual(stats)
  })

  it('normalizes dashboard trend arrays from the current backend response', async () => {
    const trend = [{ date: '2026-06-01', requests: 2, actual_cost: 0.2 }]
    get.mockResolvedValue({ data: trend })

    const result = await usageAPI.getDashboardTrend({ granularity: 'day' })

    expect(get).toHaveBeenCalledWith('/usage/dashboard/trend', { params: { granularity: 'day' } })
    expect(result).toEqual(trend)
  })

  it('normalizes dashboard trend wrapper responses from compatible callers', async () => {
    const trend = [{ date: '2026-06-01', requests: 2, actual_cost: 0.2 }]
    get.mockResolvedValue({ data: { trend } })

    const result = await usageAPI.getDashboardTrend()

    expect(get).toHaveBeenCalledWith('/usage/dashboard/trend', { params: undefined })
    expect(result).toEqual(trend)
  })
})
