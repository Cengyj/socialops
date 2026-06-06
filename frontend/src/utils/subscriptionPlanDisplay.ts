import type { SubscriptionPlan } from '@/types/payment'
import {
  computePlanValidityDays,
  getPlanGuardrails,
  getPlanQuotaAmount,
  getPlanQuotaPeriod,
  normalizeValidityUnit,
  type QuotaGuardrailPeriod,
} from '@/utils/subscriptionQuotaPlans'
import { getPlanPlatform } from '@/utils/subscriptionPackages'

export const SUBSCRIPTION_QUOTA_PRESETS = [50, 100, 200, 500, 1000] as const

export const SUBSCRIPTION_PLAN_PLATFORM_VALUES = [
  'x_twitter',
  'instagram',
  'tiktok',
  'facebook',
] as const

export type SubscriptionPlanPlatform = typeof SUBSCRIPTION_PLAN_PLATFORM_VALUES[number]
export type TranslateFn = (key: string, params?: Record<string, unknown>) => string

export interface SubscriptionPlanOption {
  value: number
  label: string
  description: string
  platform: string
  plan_id: number
  validityLabel: string
  defaultValidityDays: number
  summary: string
  quota_usd: number | null
  guardrailSummary: string
  forSale: boolean
  [key: string]: unknown
}

export function normalizePlanPlatformLabelKey(platform: string | null | undefined): string {
  const value = String(platform || '').trim().toLowerCase()
  if (value === 'twitter' || value === 'x') return 'x_twitter'
  return value || 'social'
}

export function getSubscriptionPlatformLabel(platform: string | null | undefined, fallbackLabel = 'SocialOps'): string {
  const normalized = normalizePlanPlatformLabelKey(platform)
  switch (normalized) {
    case 'x_twitter':
      return 'X / Twitter'
    case 'instagram':
      return 'Instagram'
    case 'tiktok':
      return 'TikTok'
    case 'facebook':
      return 'Facebook'
    default:
      return normalized !== 'social' ? String(platform || '').trim() : fallbackLabel
  }
}

export function getSubscriptionPlanPlatformOptions() {
  return SUBSCRIPTION_PLAN_PLATFORM_VALUES.map((value) => ({
    value,
    label: getSubscriptionPlatformLabel(value),
  }))
}

export function formatSubscriptionQuotaAmount(value: unknown, unlimitedLabel: string): string {
  if (value === null || value === undefined || value === '') return unlimitedLabel
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return unlimitedLabel
  return `$${num.toFixed(2)}`
}

export function formatSubscriptionPlanValidity(
  plan: Pick<SubscriptionPlan, 'validity_days' | 'validity_unit'>,
  t: TranslateFn,
  locale = '',
): string {
  const value = Number(plan.validity_days || 0) || 0
  const unit = normalizeValidityUnit(plan.validity_unit)
  const separator = locale.toLowerCase().startsWith('zh') ? '' : ' '
  return `${value}${separator}${t(`payment.admin.${unit}`)}`
}

export function formatSubscriptionPlanValiditySuffix(
  plan: Pick<SubscriptionPlan, 'validity_days' | 'validity_unit'>,
  t: TranslateFn,
  locale = '',
): string {
  const unit = normalizeValidityUnit(plan.validity_unit)
  if (unit === 'months' && Number(plan.validity_days) === 1) return t('payment.perMonth')
  if (unit === 'years' && Number(plan.validity_days) === 1) return t('payment.perYear')
  return formatSubscriptionPlanValidity(plan, t, locale)
}

export function formatSubscriptionPlanGuardrails(
  plan: Pick<SubscriptionPlan, 'quota_usd' | 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'>,
  t: TranslateFn,
): string {
  return getPlanGuardrails(plan)
    .map((guardrail) => `${guardrailLabel(guardrail.period, t)}: $${guardrail.amount.toFixed(2)}`)
    .join(' / ')
}

export function describeSubscriptionPlan(
  plan: SubscriptionPlan,
  t: TranslateFn,
  locale = '',
): string {
  const quota = formatSubscriptionQuotaAmount(getPlanQuotaAmount(plan), t('payment.admin.unlimited'))
  return [
    getSubscriptionPlatformLabel(getPlanPlatform(plan), t('payment.platformFallback')),
    `${formatPlanQuotaPeriodLabel(plan, t)} ${quota}`,
    formatSubscriptionPlanValidity(plan, t, locale),
  ].join(' - ')
}

export function formatPlanQuotaPeriodLabel(
  plan: Pick<SubscriptionPlan, 'quota_usd' | 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'>,
  t: TranslateFn,
): string {
  const period = getPlanQuotaPeriod(plan)
  if (period === 'daily') return t('payment.planCard.todayQuota')
  if (period === 'weekly') return t('payment.planCard.thisWeekQuota')
  if (period === 'monthly') return t('payment.planCard.thisMonthQuota')
  return t('payment.planCard.periodQuota')
}

export function toSubscriptionPlanOption(
  plan: SubscriptionPlan,
  t: TranslateFn,
  locale = '',
): SubscriptionPlanOption {
  return {
    value: plan.id,
    label: plan.name,
    description: plan.description || '',
    platform: getPlanPlatform(plan),
    plan_id: plan.id,
    validityLabel: formatSubscriptionPlanValidity(plan, t, locale),
    defaultValidityDays: computePlanValidityDays(plan),
    summary: describeSubscriptionPlan(plan, t, locale),
    quota_usd: getPlanQuotaAmount(plan),
    guardrailSummary: formatSubscriptionPlanGuardrails(plan, t),
    forSale: plan.for_sale,
  }
}

function guardrailLabel(period: QuotaGuardrailPeriod, t: TranslateFn): string {
  return period === 'daily' ? t('payment.admin.dailyGuardrail') : t('payment.admin.weeklyGuardrail')
}
