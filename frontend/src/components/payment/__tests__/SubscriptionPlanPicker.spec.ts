import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SubscriptionPlanPicker from '../SubscriptionPlanPicker.vue'
import type { UserSubscription } from '@/types'
import type { SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'payment.subscriptionPicker.packageFallback') {
        return `${params?.platform} ${params?.validity}`
      }
      if (key === 'payment.planCard.unlimited') return 'Unlimited'
      if (key === 'payment.activeSubscription') return 'Active Subscription'
      if (key === 'payment.subscribeNow') return 'Subscribe Now'
      if (key === 'payment.renewNow') return 'Renew'
      if (key === 'payment.subscriptionPicker.chooseQuota') return 'Choose quota'
      if (key === 'payment.subscriptionPicker.executionOnly') return 'Execution only'
      if (key === 'payment.subscriptionPicker.bestValue') return 'Best value'
      if (key === 'payment.planCard.periodQuota') return 'Period Quota'
      if (key === 'payment.planCard.todayQuota') return "Today's Quota"
      if (key === 'payment.planCard.thisWeekQuota') return 'This Week Quota'
      if (key === 'payment.planCard.thisMonthQuota') return 'Monthly Quota'
      if (key === 'payment.planCard.dailyLimit') return 'Daily Limit'
      if (key === 'payment.planCard.weeklyLimit') return 'Weekly Limit'
      if (key === 'payment.admin.months') return 'month'
      if (key === 'userSubscriptions.daysRemaining') return `${params?.days} days remaining`
      if (key === 'userSubscriptions.status.active') return 'Active'
      if (key === 'userSubscriptions.noExpiration') return 'No expiration'
      return key
    },
    locale: { value: 'en' },
  }),
}))

const makePlan = (overrides: Partial<SubscriptionPlan>): SubscriptionPlan => ({
  id: 1,
  group_id: 10,
  platform: 'x_twitter',
  group_platform: 'x_twitter',
  name: 'X Execute 100 USD',
  description: 'Execution package',
  price: 29,
  original_price: 39,
  validity_days: 1,
  validity_unit: 'months',
  features: ['Follow tasks'],
  product_name: 'X Execute 100 USD',
  for_sale: true,
  sort_order: 0,
  quota_usd: 100,
  daily_limit_usd: 6,
  weekly_limit_usd: 28,
  monthly_limit_usd: 100,
  ...overrides,
})

const makeSubscription = (overrides: Partial<UserSubscription> = {}): UserSubscription => ({
  id: 10,
  user_id: 20,
  group_id: 10,
  plan_id: 1,
  plan_name: 'X Execute 100 USD',
  plan_platform: 'x_twitter',
  quota_usd: 100,
  daily_limit_usd: 6,
  weekly_limit_usd: 28,
  monthly_limit_usd: 100,
  status: 'active',
  starts_at: '2026-06-01T00:00:00Z',
  daily_usage_usd: 1,
  weekly_usage_usd: 2,
  monthly_usage_usd: 3,
  daily_window_start: '2026-06-03T00:00:00Z',
  weekly_window_start: '2026-06-01T00:00:00Z',
  monthly_window_start: '2026-06-01T00:00:00Z',
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z',
  expires_at: '2026-07-01T00:00:00Z',
  ...overrides,
})

function mountPicker(overrides: Partial<InstanceType<typeof SubscriptionPlanPicker>['$props']> = {}) {
  return mount(SubscriptionPlanPicker, {
    props: {
      plans: [
        makePlan({ id: 1, quota_usd: 100, monthly_limit_usd: 100, price: 29, product_name: 'X Execute 100 USD' }),
        makePlan({ id: 2, quota_usd: 200, monthly_limit_usd: 200, price: 49, product_name: 'X Execute 200 USD' }),
      ],
      activeSubscriptions: [],
      formatAmount: (value: number) => `$${value.toFixed(2)}`,
      ...overrides,
    },
    global: {
      stubs: {
        Icon: true,
      },
    },
  })
}

describe('SubscriptionPlanPicker', () => {
  it('renders one package with quota choices instead of duplicate cards', () => {
    const wrapper = mountPicker()

    expect(wrapper.text()).toContain('X Execute')
    expect(wrapper.text()).toContain('$100.00')
    expect(wrapper.text()).toContain('$200.00')
    expect(wrapper.text()).toContain('Execution only')
    expect(wrapper.text()).toContain('Best value')
  })

  it('does not mix daily or weekly guardrails into the purchase card quota display', () => {
    const wrapper = mountPicker()

    expect(wrapper.text()).toContain('Monthly Quota')
    expect(wrapper.text()).not.toContain('Period Quota')
    expect(wrapper.text()).not.toContain('Daily Limit')
    expect(wrapper.text()).not.toContain('Weekly Limit')
  })

  it('emits the selected quota plan', async () => {
    const wrapper = mountPicker()
    const quotaButtons = wrapper.findAll('button').filter((button) => button.text().includes('$200.00'))

    expect(quotaButtons).toHaveLength(1)
    await quotaButtons[0].trigger('click')
    await wrapper.find('button.mt-4').trigger('click')

    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ id: 2, quota_usd: 200 })
  })

  it('uses an initial selected quota when provided', () => {
    const wrapper = mount(SubscriptionPlanPicker, {
      props: {
        plans: [
          makePlan({ id: 1, quota_usd: 100, monthly_limit_usd: 100, price: 29, product_name: 'X Execute 100 USD' }),
          makePlan({ id: 2, quota_usd: 200, monthly_limit_usd: 200, price: 49, product_name: 'X Execute 200 USD' }),
        ],
        activeSubscriptions: [],
        formatAmount: (value: number) => `$${value.toFixed(2)}`,
        initialSelectedPlanIds: [2],
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const selectedQuota = wrapper.findAll('button').find((button) => button.attributes('aria-pressed') === 'true')
    expect(selectedQuota?.text()).toContain('$200.00')
  })

  it('summarizes active subscriptions with one canonical period quota', () => {
    const wrapper = mountPicker({
      activeSubscriptions: [makeSubscription()],
    })

    expect(wrapper.text()).toContain('Monthly Quota: $100.00')
    expect(wrapper.text()).not.toContain('Daily Limit')
    expect(wrapper.text()).not.toContain('Weekly Limit')
  })

  it('summarizes daily-only subscriptions as today quota', () => {
    const wrapper = mountPicker({
      activeSubscriptions: [makeSubscription({
        quota_usd: null,
        monthly_limit_usd: null,
        weekly_limit_usd: null,
        daily_limit_usd: 6,
      })],
    })

    expect(wrapper.text()).toContain("Today's Quota: $6.00")
    expect(wrapper.text()).not.toContain('Monthly Quota: $6.00')
  })

  it('uses branded platform logos for purchase packages and active subscription summaries', () => {
    const wrapper = mountPicker({
      activeSubscriptions: [makeSubscription()],
    })

    expect(wrapper.findAll('[data-platform-logo]').length).toBeGreaterThanOrEqual(2)
    expect(wrapper.findAll('[data-platform-logo]').every((logo) => logo.text() === '')).toBe(true)
  })
})
