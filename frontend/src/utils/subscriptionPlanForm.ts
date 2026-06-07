import type { SubscriptionPlan } from '@/types/payment'
import type { SubscriptionPlanPayload } from '@/api/admin/payment'
import { getPlanPlatform } from '@/utils/subscriptionPackages'
import { getPlanQuotaAmount } from '@/utils/subscriptionQuotaPlans'

export interface SubscriptionPlanFormState {
  name: string
  platform: string
  product_name: string
  description: string
  price: number
  original_price: number
  validity_days: number
  validity_unit: string
  quota_usd: number | null
  daily_limit_usd: number | null
  weekly_limit_usd: number | null
  sort_order: number
  for_sale: boolean
}

export interface SubscriptionPlanFormValidationResult {
  valid: boolean
  messageKey?: string
}

export function createSubscriptionPlanFormDefaults(): SubscriptionPlanFormState {
  return {
    name: '',
    platform: 'x_twitter',
    product_name: '',
    description: '',
    price: 0,
    original_price: 0,
    validity_days: 1,
    validity_unit: 'months',
    quota_usd: null,
    daily_limit_usd: null,
    weekly_limit_usd: null,
    sort_order: 0,
    for_sale: false,
  }
}

export function subscriptionPlanToFormState(plan: SubscriptionPlan | null): SubscriptionPlanFormState {
  if (!plan) return createSubscriptionPlanFormDefaults()

  return {
    name: plan.name,
    platform: getPlanPlatform(plan),
    product_name: plan.product_name || '',
    description: plan.description || '',
    price: plan.price,
    original_price: plan.original_price || 0,
    validity_days: plan.validity_days,
    validity_unit: plan.validity_unit || 'months',
    quota_usd: getPlanQuotaAmount(plan),
    daily_limit_usd: plan.daily_limit_usd ?? null,
    weekly_limit_usd: plan.weekly_limit_usd ?? null,
    sort_order: plan.sort_order || 0,
    for_sale: plan.for_sale,
  }
}

export function buildSubscriptionPlanPayload(
  form: SubscriptionPlanFormState,
  featuresText: string,
): SubscriptionPlanPayload {
  const trimmedName = form.name.trim()
  const productName = form.product_name.trim() || trimmedName

  return {
    name: trimmedName,
    platform: form.platform,
    description: form.description.trim(),
    price: form.price,
    original_price: normalizeOptionalPositiveNumber(form.original_price),
    validity_days: form.validity_days,
    validity_unit: form.validity_unit,
    quota_usd: normalizeRequiredPositiveNumber(form.quota_usd),
    daily_limit_usd: normalizeOptionalPositiveNumber(form.daily_limit_usd) || 0,
    weekly_limit_usd: normalizeOptionalPositiveNumber(form.weekly_limit_usd) || 0,
    sort_order: form.sort_order,
    for_sale: form.for_sale,
    product_name: productName,
    features: normalizePlanFeaturesText(featuresText),
  }
}

export function validateSubscriptionPlanForm(form: SubscriptionPlanFormState): SubscriptionPlanFormValidationResult {
  const quota = positiveNumberOrNull(form.quota_usd)
  if (!quota) return { valid: false, messageKey: 'payment.admin.quotaRequired' }

  const daily = positiveNumberOrNull(form.daily_limit_usd)
  const weekly = positiveNumberOrNull(form.weekly_limit_usd)

  if (daily !== null && daily > quota) {
    return { valid: false, messageKey: 'payment.admin.dailyGuardrailExceedsQuota' }
  }
  if (weekly !== null && weekly > quota) {
    return { valid: false, messageKey: 'payment.admin.weeklyGuardrailExceedsQuota' }
  }
  if (daily !== null && weekly !== null && daily > weekly) {
    return { valid: false, messageKey: 'payment.admin.dailyGuardrailExceedsWeekly' }
  }

  return { valid: true }
}

export function normalizePlanFeaturesText(value: string): string {
  return value
    .split('\n')
    .map((feature) => feature.trim())
    .filter(Boolean)
    .join('\n')
}

export function shouldShowSubscriptionPlanAdvancedSettings(form: SubscriptionPlanFormState, featuresText: string): boolean {
  return Boolean(
    form.product_name.trim() ||
    form.description.trim() ||
    form.daily_limit_usd ||
    form.weekly_limit_usd ||
    normalizePlanFeaturesText(featuresText),
  )
}

function normalizeRequiredPositiveNumber(value: number | null): number {
  const num = Number(value)
  return Number.isFinite(num) ? num : 0
}

function normalizeOptionalPositiveNumber(value: number | null): number | undefined {
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return undefined
  return num
}

function positiveNumberOrNull(value: number | null): number | null {
  const num = Number(value)
  return Number.isFinite(num) && num > 0 ? num : null
}
