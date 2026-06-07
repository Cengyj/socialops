import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'

type PlanQuotaLike = Pick<SubscriptionPlan, 'quota_usd' | 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'>
type SubscriptionQuotaLike = Pick<UserSubscription, 'quota_usd' | 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd' | 'group'>
type ValidityLike = Pick<SubscriptionPlan, 'validity_days' | 'validity_unit'>

export type QuotaGuardrailPeriod = 'daily' | 'weekly'
export type SubscriptionQuotaPeriod = 'daily' | 'weekly' | 'monthly'

export interface QuotaGuardrail {
  period: QuotaGuardrailPeriod
  amount: number
}

export interface SubscriptionQuotaUsage {
  period: SubscriptionQuotaPeriod
  amount: number
  used: number
  windowStart: string | null
}

export function normalizeQuotaAmount(value: number | null | undefined): number | null {
  if (value === null || value === undefined) return null
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return null
  return num
}

export function getPlanQuotaAmount(plan: PlanQuotaLike): number | null {
  return (
    normalizeQuotaAmount(plan.quota_usd) ??
    normalizeQuotaAmount(plan.monthly_limit_usd) ??
    normalizeQuotaAmount(plan.weekly_limit_usd) ??
    normalizeQuotaAmount(plan.daily_limit_usd)
  )
}

export function getPlanQuotaPeriod(plan: PlanQuotaLike): SubscriptionQuotaPeriod | null {
  if (normalizeQuotaAmount(plan.quota_usd) !== null || normalizeQuotaAmount(plan.monthly_limit_usd) !== null) {
    return 'monthly'
  }
  if (normalizeQuotaAmount(plan.weekly_limit_usd) !== null) {
    return 'weekly'
  }
  if (normalizeQuotaAmount(plan.daily_limit_usd) !== null) {
    return 'daily'
  }
  return null
}

export function getSubscriptionQuotaAmount(subscription: SubscriptionQuotaLike): number | null {
  return (
    normalizeQuotaAmount(subscription.quota_usd) ??
    normalizeQuotaAmount(subscription.monthly_limit_usd ?? subscription.group?.monthly_limit_usd) ??
    normalizeQuotaAmount(subscription.weekly_limit_usd ?? subscription.group?.weekly_limit_usd) ??
    normalizeQuotaAmount(subscription.daily_limit_usd ?? subscription.group?.daily_limit_usd)
  )
}

export function getSubscriptionQuotaUsage(
  subscription: SubscriptionQuotaLike & Pick<
    UserSubscription,
    | 'daily_usage_usd'
    | 'weekly_usage_usd'
    | 'monthly_usage_usd'
    | 'daily_window_start'
    | 'weekly_window_start'
    | 'monthly_window_start'
  >
): SubscriptionQuotaUsage | null {
  const quota = normalizeQuotaAmount(subscription.quota_usd)
  const monthly = normalizeQuotaAmount(subscription.monthly_limit_usd ?? subscription.group?.monthly_limit_usd)
  if (quota !== null || monthly !== null) {
    const amount = quota ?? monthly
    if (amount !== null) {
      return {
        period: 'monthly',
        amount,
        used: Number(subscription.monthly_usage_usd || 0),
        windowStart: subscription.monthly_window_start,
      }
    }
  }

  const weekly = normalizeQuotaAmount(subscription.weekly_limit_usd ?? subscription.group?.weekly_limit_usd)
  if (weekly !== null) {
    return {
      period: 'weekly',
      amount: weekly,
      used: Number(subscription.weekly_usage_usd || 0),
      windowStart: subscription.weekly_window_start,
    }
  }

  const daily = normalizeQuotaAmount(subscription.daily_limit_usd ?? subscription.group?.daily_limit_usd)
  if (daily !== null) {
    return {
      period: 'daily',
      amount: daily,
      used: Number(subscription.daily_usage_usd || 0),
      windowStart: subscription.daily_window_start,
    }
  }

  return null
}

export function hasPlanGuardrails(plan: PlanQuotaLike): boolean {
  return normalizeQuotaAmount(plan.daily_limit_usd) !== null || normalizeQuotaAmount(plan.weekly_limit_usd) !== null
}

export function getPlanGuardrails(plan: PlanQuotaLike): QuotaGuardrail[] {
  return [
    { period: 'daily' as const, amount: normalizeQuotaAmount(plan.daily_limit_usd) },
    { period: 'weekly' as const, amount: normalizeQuotaAmount(plan.weekly_limit_usd) },
  ].filter((item): item is QuotaGuardrail => item.amount !== null)
}

export function computePlanValidityDays(plan: ValidityLike): number {
  const base = Number(plan.validity_days || 0)
  const unit = String(plan.validity_unit || 'days').toLowerCase()
  if (base <= 0) return 30
  if (unit === 'week' || unit === 'weeks') return base * 7
  if (unit === 'month' || unit === 'months') return base * 30
  if (unit === 'year' || unit === 'years') return base * 365
  return base
}

export function normalizeValidityUnit(unit: string | null | undefined): 'days' | 'weeks' | 'months' | 'years' {
  const raw = String(unit || 'days').toLowerCase()
  if (raw === 'week' || raw === 'weeks') return 'weeks'
  if (raw === 'month' || raw === 'months') return 'months'
  if (raw === 'year' || raw === 'years') return 'years'
  return 'days'
}
