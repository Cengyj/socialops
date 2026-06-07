import { beforeEach, describe, expect, expectTypeOf, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import subscriptionsAPI, { getSubscriptionSummary, getSubscriptionsProgress } from '@/api/subscriptions'
import type { SubscriptionSummary } from '@/api/subscriptions'
import type { SubscriptionProgressEntry } from '@/types'

describe('user subscriptions api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('matches the backend progress list envelope', async () => {
    const payload: SubscriptionProgressEntry[] = [{
      subscription: {
        id: 12,
        user_id: 3,
        group_id: 4,
        plan_id: 5,
        plan_name: 'X Starter',
        plan_platform: 'x_twitter',
        quota_usd: 100,
        daily_limit_usd: 10,
        weekly_limit_usd: 50,
        monthly_limit_usd: 100,
        status: 'active',
        starts_at: '2026-01-01T00:00:00Z',
        expires_at: '2026-02-01T00:00:00Z',
        daily_usage_usd: 2,
        weekly_usage_usd: 8,
        monthly_usage_usd: 20,
        daily_window_start: '2026-01-01T00:00:00Z',
        weekly_window_start: '2026-01-01T00:00:00Z',
        monthly_window_start: '2026-01-01T00:00:00Z',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
      progress: {
        id: 12,
        group_name: 'X Starter',
        expires_at: '2026-02-01T00:00:00Z',
        expires_in_days: 28,
        monthly: {
          limit_usd: 100,
          used_usd: 20,
          remaining_usd: 80,
          percentage: 20,
          window_start: '2026-01-01T00:00:00Z',
          resets_at: '2026-01-31T00:00:00Z',
          resets_in_seconds: 86_400,
        },
      },
    }]
    get.mockResolvedValue({ data: payload })

    const result = await getSubscriptionsProgress()

    expect(get).toHaveBeenCalledWith('/subscriptions/progress')
    expect(result).toEqual(payload)
    expectTypeOf(result).toEqualTypeOf<SubscriptionProgressEntry[]>()
  })

  it('does not expose the missing user per-subscription progress endpoint', () => {
    expect(('getSubscription' + 'Progress') in subscriptionsAPI).toBe(false)
  })

  it('matches the backend summary contract', async () => {
    const payload: SubscriptionSummary = {
      active_count: 1,
      subscriptions: [{
        id: 12,
        group_name: 'X Starter',
        status: 'active',
        daily_progress: 25,
        weekly_progress: 25,
        monthly_progress: 25,
        expires_at: '2026-02-01T00:00:00Z',
        days_remaining: 28,
      }],
    }
    get.mockResolvedValue({ data: payload })

    const result = await getSubscriptionSummary()

    expect(get).toHaveBeenCalledWith('/subscriptions/summary')
    expect(result).toEqual(payload)
    expectTypeOf(result).toEqualTypeOf<SubscriptionSummary>()
  })
})
