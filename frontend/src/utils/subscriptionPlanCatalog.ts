import type { SubscriptionPlan } from '@/types/payment'
import { getPlanPlatform } from '@/utils/subscriptionPackages'
import {
  computePlanValidityDays,
  getPlanQuotaAmount,
  normalizeValidityUnit,
} from '@/utils/subscriptionQuotaPlans'

export interface SubscriptionQuotaPackage {
  key: string
  platform: string
  /**
   * @deprecated Use title. Kept for older tests/components during the package model migration.
   */
  familyName: string
  title: string
  description: string
  validityDays: number
  validityUnit: 'days' | 'weeks' | 'months' | 'years'
  plans: SubscriptionPlan[]
  defaultPlan: SubscriptionPlan
  bestValuePlanID: number | null
}

/** @deprecated Use SubscriptionQuotaPackage. */
export type SubscriptionPlanFamily = SubscriptionQuotaPackage

export interface SubscriptionQuotaChoice {
  plan: SubscriptionPlan
  planID: number
  quotaUSD: number | null
  price: number
  originalPrice: number | null
  isBestValue: boolean
}

export function buildSubscriptionQuotaPackages(plans: SubscriptionPlan[]): SubscriptionQuotaPackage[] {
  const buckets = new Map<string, SubscriptionPlan[]>()

  for (const plan of plans) {
    const key = subscriptionQuotaPackageKey(plan)
    const bucket = buckets.get(key)
    if (bucket) bucket.push(plan)
    else buckets.set(key, [plan])
  }

  return [...buckets.entries()]
    .map(([key, familyPlans]) => {
      const sortedPlans = sortSubscriptionPlansByQuota(familyPlans)
      const defaultPlan = sortedPlans[0]
      const title = packageDisplayNameCandidate(sortedPlans)
      return {
        key,
        platform: getPlanPlatform(defaultPlan),
        familyName: title,
        title,
        description: firstNonEmpty(sortedPlans.map((plan) => plan.description)),
        validityDays: computePlanValidityDays(defaultPlan),
        validityUnit: normalizeValidityUnit(defaultPlan.validity_unit),
        plans: sortedPlans,
        defaultPlan,
        bestValuePlanID: bestValuePlanID(sortedPlans),
      }
    })
    .sort(compareSubscriptionQuotaPackages)
}

/** @deprecated Use buildSubscriptionQuotaPackages. */
export function buildSubscriptionPlanFamilies(plans: SubscriptionPlan[]): SubscriptionPlanFamily[] {
  return buildSubscriptionQuotaPackages(plans)
}

export function getSubscriptionQuotaChoices(quotaPackage: SubscriptionQuotaPackage): SubscriptionQuotaChoice[] {
  return quotaPackage.plans.map((plan) => ({
    plan,
    planID: plan.id,
    quotaUSD: getPlanQuotaAmount(plan),
    price: plan.price,
    originalPrice: normalizePositivePrice(plan.original_price),
    isBestValue: quotaPackage.bestValuePlanID === plan.id,
  }))
}

export function sortSubscriptionPlansByQuota(plans: SubscriptionPlan[]): SubscriptionPlan[] {
  return [...plans].sort((left, right) => {
    const leftQuota = getPlanQuotaAmount(left) ?? Number.POSITIVE_INFINITY
    const rightQuota = getPlanQuotaAmount(right) ?? Number.POSITIVE_INFINITY
    if (leftQuota !== rightQuota) return leftQuota - rightQuota
    if (left.price !== right.price) return left.price - right.price
    if ((left.sort_order || 0) !== (right.sort_order || 0)) return (left.sort_order || 0) - (right.sort_order || 0)
    return left.id - right.id
  })
}

export function subscriptionQuotaPackageKey(plan: SubscriptionPlan): string {
  return [
    getPlanPlatform(plan),
    normalizeValidityUnit(plan.validity_unit),
    Number(plan.validity_days || 0),
    normalizePackageToken(plan.product_name || plan.name),
  ].join(':')
}

/** @deprecated Use subscriptionQuotaPackageKey. */
export function subscriptionPlanFamilyKey(plan: SubscriptionPlan): string {
  return subscriptionQuotaPackageKey(plan)
}

export function getSubscriptionPlanDiscountPercent(plan: SubscriptionPlan): number {
  const originalPrice = Number(plan.original_price || 0)
  if (!Number.isFinite(originalPrice) || originalPrice <= 0 || plan.price >= originalPrice) return 0
  return Math.round((1 - plan.price / originalPrice) * 100)
}

export function getSubscriptionPlanValueScore(plan: SubscriptionPlan): number {
  const quota = getPlanQuotaAmount(plan)
  if (!quota || plan.price <= 0) return 0
  return quota / plan.price
}

export function bestValuePlanID(plans: SubscriptionPlan[]): number | null {
  let selected: SubscriptionPlan | null = null
  let selectedScore = 0

  for (const plan of plans) {
    const score = getSubscriptionPlanValueScore(plan)
    if (score > selectedScore) {
      selected = plan
      selectedScore = score
    }
  }

  return selected?.id ?? null
}

function compareSubscriptionQuotaPackages(left: SubscriptionQuotaPackage, right: SubscriptionQuotaPackage): number {
  const leftSort = Math.min(...left.plans.map((plan) => plan.sort_order || 0))
  const rightSort = Math.min(...right.plans.map((plan) => plan.sort_order || 0))
  if (leftSort !== rightSort) return leftSort - rightSort
  if (left.platform !== right.platform) return left.platform.localeCompare(right.platform)
  if (left.validityDays !== right.validityDays) return left.validityDays - right.validityDays
  return left.title.localeCompare(right.title)
}

function packageDisplayNameCandidate(plans: SubscriptionPlan[]): string {
  const productNames = plans
    .map((plan) => stripQuotaText(plan.product_name))
    .filter(Boolean)
  const uniqueProductNames = [...new Set(productNames)]
  if (uniqueProductNames.length === 1) return uniqueProductNames[0]

  const strippedNames = plans
    .map((plan) => stripQuotaText(plan.name))
    .filter(Boolean)

  const uniqueNames = [...new Set(strippedNames)]
  if (uniqueNames.length === 1) return uniqueNames[0]

  return ''
}

function normalizePackageToken(value: string | null | undefined): string {
  return stripQuotaText(value)
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'default'
}

function stripQuotaText(value: string | null | undefined): string {
  return String(value || '')
    .normalize('NFKC')
    .replace(/\$\s*\d+(?:\.\d+)?/g, ' ')
    .replace(
      /\d+(?:\.\d+)?\s*(usd|usdt|dollars?|\u7f8e\u5143|\u7f8e\u91d1|\u5200|\u5143|\u989d\u5ea6|\u914d\u989d|quota)/gi,
      ' ',
    )
    .replace(/\u989d\u5ea6|\u914d\u989d|\u5957\u9910/g, ' ')
    .replace(/\b(quota|plan|package|tier)\b/gi, ' ')
    .replace(/\d+(?:\.\d+)?\s*$/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function firstNonEmpty(values: Array<string | null | undefined>): string {
  for (const value of values) {
    const trimmed = String(value || '').trim()
    if (trimmed) return trimmed
  }
  return ''
}

function normalizePositivePrice(value: number | null | undefined): number | null {
  const num = Number(value)
  return Number.isFinite(num) && num > 0 ? num : null
}
