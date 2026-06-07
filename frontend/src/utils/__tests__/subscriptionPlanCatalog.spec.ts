import { describe, expect, it } from 'vitest'
import type { SubscriptionPlan } from '@/types/payment'
import {
  bestValuePlanID,
  buildSubscriptionQuotaPackages,
  getSubscriptionQuotaChoices,
  subscriptionQuotaPackageKey,
} from '../subscriptionPlanCatalog'

function plan(overrides: Partial<SubscriptionPlan>): SubscriptionPlan {
  return {
    id: 1,
    group_id: 10,
    platform: 'x_twitter',
    group_platform: 'x_twitter',
    name: 'Twitter Monthly 50 USD',
    description: '',
    price: 5,
    validity_days: 1,
    validity_unit: 'months',
    features: [],
    product_name: 'Twitter Monthly',
    for_sale: true,
    sort_order: 0,
    quota_usd: 50,
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: 50,
    ...overrides,
  }
}

describe('subscriptionPlanCatalog', () => {
  it('groups quota variants by package, platform, and validity', () => {
    const quotaPackages = buildSubscriptionQuotaPackages([
      plan({ id: 2, quota_usd: 100, monthly_limit_usd: 100, price: 9, name: 'Twitter Monthly 100 USD' }),
      plan({ id: 1, quota_usd: 50, monthly_limit_usd: 50, price: 5, name: 'Twitter Monthly 50 USD' }),
      plan({ id: 3, platform: 'instagram', group_platform: 'instagram', product_name: 'Instagram Monthly' }),
    ])

    expect(quotaPackages).toHaveLength(2)
    const twitterPackage = quotaPackages.find((quotaPackage) => quotaPackage.platform === 'x_twitter')
    expect(twitterPackage?.plans.map((item) => item.id)).toEqual([1, 2])
    expect(twitterPackage?.defaultPlan.id).toBe(1)
  })

  it('strips quota text from names when no explicit product name exists', () => {
    const first = plan({ product_name: '', name: 'X Execute 50 USD', quota_usd: 50 })
    const second = plan({ id: 2, product_name: '', name: 'X Execute 100 USD', quota_usd: 100 })

    expect(subscriptionQuotaPackageKey(first)).toBe(subscriptionQuotaPackageKey(second))
  })

  it('uses a cleaned product name as the package display name', () => {
    const quotaPackages = buildSubscriptionQuotaPackages([
      plan({ product_name: 'X Execute 50 USD', name: 'X Execute 50 USD', quota_usd: 50 }),
      plan({ id: 2, product_name: 'X Execute 100 USD', name: 'X Execute 100 USD', quota_usd: 100 }),
    ])

    expect(quotaPackages[0].title).toBe('X Execute')
  })

  it('normalizes readable Chinese quota names without removing period numbers', () => {
    const first = plan({
      product_name: 'X 30\u5929\u6267\u884c\u5957\u9910 50\u7f8e\u5143',
      name: 'X 30\u5929\u6267\u884c\u5957\u9910 50\u7f8e\u5143',
      quota_usd: 50,
    })
    const second = plan({
      id: 2,
      product_name: 'X 30\u5929\u6267\u884c\u5957\u9910 100\u7f8e\u5143',
      name: 'X 30\u5929\u6267\u884c\u5957\u9910 100\u7f8e\u5143',
      quota_usd: 100,
    })

    const quotaPackages = buildSubscriptionQuotaPackages([first, second])

    expect(subscriptionQuotaPackageKey(first)).toBe(subscriptionQuotaPackageKey(second))
    expect(quotaPackages[0].title).toBe('X 30\u5929\u6267\u884c')
  })

  it('normalizes Chinese quota names without removing non-quota numbers', () => {
    const first = plan({ product_name: 'X 30天执行套餐 50美元', name: 'X 30天执行套餐 50美元', quota_usd: 50 })
    const second = plan({ id: 2, product_name: 'X 30天执行套餐 100美元', name: 'X 30天执行套餐 100美元', quota_usd: 100 })

    const quotaPackages = buildSubscriptionQuotaPackages([first, second])

    expect(subscriptionQuotaPackageKey(first)).toBe(subscriptionQuotaPackageKey(second))
    expect(quotaPackages[0].title).toBe('X 30天执行')
  })

  it('identifies the best value plan by quota per price', () => {
    expect(bestValuePlanID([
      plan({ id: 1, quota_usd: 50, price: 5 }),
      plan({ id: 2, quota_usd: 200, price: 12 }),
    ])).toBe(2)
  })

  it('builds quota choices with the best value flag', () => {
    const [quotaPackage] = buildSubscriptionQuotaPackages([
      plan({ id: 1, quota_usd: 100, monthly_limit_usd: 100, price: 20 }),
      plan({ id: 2, quota_usd: 200, monthly_limit_usd: 200, price: 30 }),
    ])

    expect(quotaPackage.title).toBe('Twitter Monthly')
    expect(quotaPackage.bestValuePlanID).toBe(2)
    expect(getSubscriptionQuotaChoices(quotaPackage)).toEqual([
      expect.objectContaining({ planID: 1, quotaUSD: 100, price: 20, isBestValue: false }),
      expect.objectContaining({ planID: 2, quotaUSD: 200, price: 30, isBestValue: true }),
    ])
  })
})
