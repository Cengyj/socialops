import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SubscriptionQuotaChoiceList from '../SubscriptionQuotaChoiceList.vue'
import type { SubscriptionPlan } from '@/types/payment'
import { buildSubscriptionQuotaPackages } from '@/utils/subscriptionPlanCatalog'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      if (key === 'payment.planCard.unlimited') return 'Unlimited'
      if (key === 'payment.subscriptionPicker.bestValue') return 'Best value'
      return key
    },
  }),
}))

function plan(overrides: Partial<SubscriptionPlan>): SubscriptionPlan {
  return {
    id: 1,
    group_id: 10,
    platform: 'x_twitter',
    group_platform: 'x_twitter',
    name: 'X Execute 100 USD',
    description: '',
    price: 29,
    validity_days: 1,
    validity_unit: 'months',
    features: [],
    product_name: 'X Execute',
    for_sale: true,
    sort_order: 0,
    quota_usd: 100,
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: 100,
    ...overrides,
  }
}

describe('SubscriptionQuotaChoiceList', () => {
  it('renders quota choices and emits only the selected plan', async () => {
    const [quotaPackage] = buildSubscriptionQuotaPackages([
      plan({ id: 1, quota_usd: 100, monthly_limit_usd: 100, price: 29 }),
      plan({ id: 2, quota_usd: 200, monthly_limit_usd: 200, price: 49 }),
    ])

    const wrapper = mount(SubscriptionQuotaChoiceList, {
      props: {
        quotaPackage,
        selectedPlanId: 1,
        formatAmount: (value: number) => `$${value.toFixed(2)}`,
      },
    })

    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(2)
    expect(buttons[0].attributes('aria-pressed')).toBe('true')
    expect(wrapper.text()).toContain('$100.00')
    expect(wrapper.text()).toContain('$200.00')
    expect(wrapper.text()).toContain('Best value')

    await buttons[1].trigger('click')

    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ id: 2, quota_usd: 200 })
  })
})
