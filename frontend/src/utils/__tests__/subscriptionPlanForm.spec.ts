import { describe, expect, it } from 'vitest'
import {
  buildSubscriptionPlanPayload,
  createSubscriptionPlanFormDefaults,
  normalizePlanFeaturesText,
  subscriptionPlanToFormState,
  validateSubscriptionPlanForm,
} from '../subscriptionPlanForm'
import type { SubscriptionPlan } from '@/types/payment'

describe('subscriptionPlanForm', () => {
  it('defaults new plans to hidden until explicitly published', () => {
    expect(createSubscriptionPlanFormDefaults().for_sale).toBe(false)
  })

  it('builds the quota package payload from a form state', () => {
    const form = {
      ...createSubscriptionPlanFormDefaults(),
      name: 'Twitter Monthly 100',
      platform: 'x_twitter',
      product_name: 'Twitter Monthly',
      price: 8.8,
      original_price: 10,
      quota_usd: 100,
      daily_limit_usd: 10,
      weekly_limit_usd: 60,
      for_sale: true,
    }

    expect(buildSubscriptionPlanPayload(form, 'Follow\n\nPost ')).toEqual({
      name: 'Twitter Monthly 100',
      platform: 'x_twitter',
      description: '',
      price: 8.8,
      original_price: 10,
      validity_days: 1,
      validity_unit: 'months',
      quota_usd: 100,
      daily_limit_usd: 10,
      weekly_limit_usd: 60,
      sort_order: 0,
      for_sale: true,
      product_name: 'Twitter Monthly',
      features: 'Follow\nPost',
    })
  })

  it('reads quota_usd from the stored monthly limit fallback', () => {
    const form = subscriptionPlanToFormState({
      id: 1,
      group_id: 10,
      platform: 'x_twitter',
      group_platform: 'x_twitter',
      name: 'Twitter Monthly 100',
      description: '',
      price: 8.8,
      validity_days: 1,
      validity_unit: 'months',
      features: [],
      product_name: 'Twitter Monthly',
      for_sale: true,
      sort_order: 0,
      quota_usd: null,
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: 100,
    } as SubscriptionPlan)

    expect(form.quota_usd).toBe(100)
    expect(form.product_name).toBe('Twitter Monthly')
  })

  it('normalizes feature text', () => {
    expect(normalizePlanFeaturesText(' A \n\n B ')).toBe('A\nB')
  })

  it('validates guardrails against the package quota', () => {
    const base = {
      ...createSubscriptionPlanFormDefaults(),
      quota_usd: 100,
    }

    expect(validateSubscriptionPlanForm({ ...base, daily_limit_usd: 120 })).toEqual({
      valid: false,
      messageKey: 'payment.admin.dailyGuardrailExceedsQuota',
    })
    expect(validateSubscriptionPlanForm({ ...base, weekly_limit_usd: 120 })).toEqual({
      valid: false,
      messageKey: 'payment.admin.weeklyGuardrailExceedsQuota',
    })
    expect(validateSubscriptionPlanForm({ ...base, daily_limit_usd: 60, weekly_limit_usd: 50 })).toEqual({
      valid: false,
      messageKey: 'payment.admin.dailyGuardrailExceedsWeekly',
    })
    expect(validateSubscriptionPlanForm({ ...base, daily_limit_usd: 10, weekly_limit_usd: 50 })).toEqual({
      valid: true,
    })
  })
})
