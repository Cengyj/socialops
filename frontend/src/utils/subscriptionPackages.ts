import type { UserSubscription } from '@/types'
import type { SubscriptionPlan } from '@/types/payment'

function normalizePositiveLimit(value: number | null | undefined): number | null {
  if (value === null || value === undefined || Number.isNaN(value) || value <= 0) {
    return null
  }
  return value
}

export function normalizeSubscriptionPlatform(platform: string | null | undefined): string {
  const value = String(platform || '').trim().toLowerCase()
  if (value === 'twitter' || value === 'x') return 'x_twitter'
  return value || 'social'
}

export function getPlanPlatform(
  plan: Pick<SubscriptionPlan, 'platform' | 'group_platform'>
): string {
  return normalizeSubscriptionPlatform(plan.platform || plan.group_platform)
}

export function getSubscriptionPlatform(
  subscription: Pick<UserSubscription, 'plan_platform' | 'group'>
): string {
  return normalizeSubscriptionPlatform(subscription.plan_platform || subscription.group?.platform)
}

export function getSubscriptionTitle(
  subscription: Pick<UserSubscription, 'plan_name' | 'group'>
): string | null {
  const value = subscription.plan_name?.trim() || subscription.group?.name?.trim()
  return value || null
}

export function getSubscriptionDailyLimit(
  subscription: Pick<UserSubscription, 'daily_limit_usd' | 'group'>
): number | null {
  return normalizePositiveLimit(subscription.daily_limit_usd ?? subscription.group?.daily_limit_usd)
}

export function getSubscriptionWeeklyLimit(
  subscription: Pick<UserSubscription, 'weekly_limit_usd' | 'group'>
): number | null {
  return normalizePositiveLimit(subscription.weekly_limit_usd ?? subscription.group?.weekly_limit_usd)
}

export function getSubscriptionMonthlyLimit(
  subscription: Pick<UserSubscription, 'quota_usd' | 'monthly_limit_usd' | 'group'>
): number | null {
  return normalizePositiveLimit(subscription.quota_usd ?? subscription.monthly_limit_usd ?? subscription.group?.monthly_limit_usd)
}

export function hasSubscriptionLimits(
  subscription: Pick<UserSubscription, 'quota_usd' | 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd' | 'group'>
): boolean {
  return (
    getSubscriptionDailyLimit(subscription) !== null ||
    getSubscriptionWeeklyLimit(subscription) !== null ||
    getSubscriptionMonthlyLimit(subscription) !== null
  )
}

export function subscriptionMatchesPlan(
  subscription: Pick<UserSubscription, 'plan_id' | 'group_id'>,
  plan: Pick<SubscriptionPlan, 'id' | 'group_id'>
): boolean {
  if (subscription.plan_id !== null && subscription.plan_id !== undefined) {
    return subscription.plan_id === plan.id
  }
  return subscription.group_id === plan.group_id
}
