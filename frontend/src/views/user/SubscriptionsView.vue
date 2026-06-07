<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <!-- Error State -->
      <div v-else-if="loadError" class="card p-12 text-center">
        <div
          class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-50 dark:bg-red-950/30"
        >
          <Icon name="exclamationTriangle" size="xl" class="text-red-500" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('userSubscriptions.failedToLoad') }}
        </h3>
        <p class="mx-auto max-w-md text-sm text-gray-500 dark:text-dark-400">
          {{ loadError }}
        </p>
        <button type="button" class="btn btn-primary mt-5" @click="loadSubscriptions">
          {{ t('common.retry') }}
        </button>
      </div>

      <!-- Empty State -->
      <div v-else-if="subscriptions.length === 0" class="card p-12 text-center">
        <div
          class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
        >
          <Icon name="creditCard" size="xl" class="text-gray-400" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('userSubscriptions.noActiveSubscriptions') }}
        </h3>
        <p class="text-gray-500 dark:text-dark-400">
          {{ t('userSubscriptions.noActiveSubscriptionsDesc') }}
        </p>
      </div>

      <!-- Subscriptions Grid -->
      <div v-else class="grid gap-6 lg:grid-cols-2">
        <div
          v-for="subscription in subscriptions"
          :key="subscription.id"
          class="overflow-hidden rounded-2xl border bg-white dark:bg-dark-800"
          :class="platformBorderClass(subscriptionPlatform(subscription))"
        >
          <!-- Header -->
          <div
            class="flex items-center justify-between border-b border-gray-100 p-4 dark:border-dark-700"
          >
            <div class="flex items-center gap-3">
              <div
                :class="[
                  'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border',
                  platformBadgeClass(subscriptionPlatform(subscription))
                ]"
              >
                <SubscriptionPlatformLogo :platform="subscriptionPlatform(subscription)" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="font-semibold text-gray-900 dark:text-white">
                    {{ subscriptionTitle(subscription) || t('payment.packageFallback', { id: subscription.plan_id || subscription.group_id }) }}
                  </h3>
                  <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium', platformBadgeClass(subscriptionPlatform(subscription))]">
                    <SubscriptionPlatformLogo :platform="subscriptionPlatform(subscription)" compact />
                    <span>{{ subscriptionPlatformLabel(subscription) }}</span>
                  </span>
                </div>
                <p v-if="subscription.group?.description" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ subscription.group.description }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'rounded-full px-2 py-0.5 text-xs font-medium',
                  subscription.status === 'active'
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
                    : subscription.status === 'expired'
                      ? 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
                      : 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
                ]"
              >
                {{ t(`userSubscriptions.status.${subscription.status}`) }}
              </span>
              <button
                v-if="subscription.status === 'active'"
                :class="['rounded-lg px-3 py-1.5 text-xs font-semibold text-white transition-colors', platformButtonClass(subscriptionPlatform(subscription))]"
                @click="router.push({ path: '/purchase', query: renewQuery(subscription) })"
              >
                {{ t('payment.renewNow') }}
              </button>
            </div>
          </div>

          <!-- Usage Progress -->
          <div class="space-y-4 p-4">
            <!-- Expiration Info -->
            <div v-if="subscription.expires_at" class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-dark-400">{{
                t('userSubscriptions.expires')
              }}</span>
              <span :class="getExpirationClass(subscription.expires_at)">
                {{ formatExpirationDate(subscription.expires_at) }}
              </span>
            </div>
            <div v-else class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-dark-400">{{
                t('userSubscriptions.expires')
              }}</span>
              <span class="text-gray-700 dark:text-gray-300">{{
                t('userSubscriptions.noExpiration')
              }}</span>
            </div>

            <!-- Period quota usage -->
            <div v-if="quotaLimit(subscription) !== null" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ quotaUsageLabel(subscription) }}
                </span>
                <span class="text-sm text-gray-500 dark:text-dark-400">
                  ${{ quotaUsed(subscription).toFixed(2) }} / ${{ quotaLimit(subscription)?.toFixed(2) }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      quotaUsed(subscription),
                      quotaLimit(subscription)
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      quotaUsed(subscription),
                      quotaLimit(subscription)
                    )
                  }"
                ></div>
              </div>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ formatQuotaUsageWindow(subscription) }}
              </p>
            </div>

            <!-- No limits configured - Unlimited badge -->
            <div
              v-if="!hasUsageLimits(subscription)"
              class="flex items-center justify-center rounded-xl bg-gradient-to-r from-emerald-50 to-teal-50 py-6 dark:from-emerald-900/20 dark:to-teal-900/20"
            >
              <div class="flex items-center gap-3">
                <Icon name="sparkles" size="xl" class="text-emerald-600 dark:text-emerald-400" />
                <div>
                  <p class="text-sm font-medium text-emerald-700 dark:text-emerald-300">
                    {{ t('userSubscriptions.unlimited') }}
                  </p>
                  <p class="text-xs text-emerald-600/70 dark:text-emerald-400/70">
                    {{ t('userSubscriptions.unlimitedDesc') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import { formatDateOnly } from '@/utils/format'
import { getPlatformColor } from '@/utils/platformColors'
import { getRemainingDurationParts, isOneTimeDailyQuota, type RemainingDurationParts } from '@/utils/subscriptionQuota'
import {
  getSubscriptionPlatform,
  getSubscriptionTitle,
  hasSubscriptionLimits,
  normalizeSubscriptionPlatform,
} from '@/utils/subscriptionPackages'
import { getSubscriptionQuotaUsage } from '@/utils/subscriptionQuotaPlans'
import { getSubscriptionPlatformLabel } from '@/utils/subscriptionPlanDisplay'

function platformLabel(platform: string): string {
  return getSubscriptionPlatformLabel(platform, t('payment.platformFallback'))
}

function platformBadgeClass(platform: string): string {
  const colors = getPlatformColor(normalizePlatform(platform))
  return `${colors.border} ${colors.bg} ${colors.text}`
}

function platformBorderClass(platform: string): string {
  return `border-l-4 ${getPlatformColor(normalizePlatform(platform)).border}`
}

function platformButtonClass(platform: string): string {
  const normalized = normalizePlatform(platform)
  if (normalized === 'instagram') return 'bg-pink-600 hover:bg-pink-700'
  if (normalized === 'facebook') return 'bg-blue-600 hover:bg-blue-700'
  return 'bg-primary-600 hover:bg-primary-700'
}

function normalizePlatform(platform: string): string {
  return normalizeSubscriptionPlatform(platform)
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)
const loadError = ref('')

async function loadSubscriptions() {
  try {
    loading.value = true
    loadError.value = ''
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    recordClientDiagnostic('subscriptions.load_my_subscriptions', error)
    loadError.value = extractSafeApiErrorMessage(error, t('userSubscriptions.failedToLoad'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return 'bg-gray-400'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-red-500'
  if (percentage >= 70) return 'bg-orange-500'
  return 'bg-green-500'
}

function subscriptionPlatform(subscription: UserSubscription): string {
  return getSubscriptionPlatform(subscription)
}

function subscriptionTitle(subscription: UserSubscription): string | null {
  return getSubscriptionTitle(subscription)
}

function subscriptionPlatformLabel(subscription: UserSubscription): string {
  return platformLabel(subscriptionPlatform(subscription))
}

function hasUsageLimits(subscription: UserSubscription): boolean {
  return hasSubscriptionLimits(subscription)
}

function quotaUsage(subscription: UserSubscription) {
  return getSubscriptionQuotaUsage(subscription)
}

function quotaLimit(subscription: UserSubscription): number | null {
  return quotaUsage(subscription)?.amount ?? null
}

function quotaUsed(subscription: UserSubscription): number {
  return quotaUsage(subscription)?.used ?? 0
}

function quotaUsageLabel(subscription: UserSubscription): string {
  const period = quotaUsage(subscription)?.period
  if (period === 'daily') return t('payment.planCard.todayQuota')
  if (period === 'weekly') return t('payment.planCard.thisWeekQuota')
  if (period === 'monthly') return t('payment.planCard.thisMonthQuota')
  return t('payment.planCard.periodQuota')
}

function renewQuery(subscription: UserSubscription): Record<string, string> {
  if (subscription.plan_id) {
    return { tab: 'subscription', plan_id: String(subscription.plan_id) }
  }
  return { tab: 'subscription', group: String(subscription.group_id) }
}

function formatExpirationDate(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (days < 0) {
    return t('userSubscriptions.status.expired')
  }

  const dateStr = formatDateOnly(expires)

  if (days === 0) {
    return `${dateStr} (${t('common.today')})`
  }
  if (days === 1) {
    return `${dateStr} (${t('common.tomorrow')})`
  }

  return t('userSubscriptions.daysRemaining', { days }) + ` (${dateStr})`
}

function getExpirationClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (days <= 0) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 3) return 'text-red-600 dark:text-red-400'
  if (days <= 7) return 'text-orange-600 dark:text-orange-400'
  return 'text-gray-700 dark:text-gray-300'
}

function formatDurationParts(parts: RemainingDurationParts): string {
  if (parts.days > 0) {
    return t('userSubscriptions.durationDaysHours', { days: parts.days, hours: parts.hours })
  }

  if (parts.hours > 0) {
    return t('userSubscriptions.durationHoursMinutes', { hours: parts.hours, minutes: parts.minutes })
  }

  return t('userSubscriptions.durationMinutes', { minutes: parts.minutes })
}

function formatDailyUsageWindow(subscription: UserSubscription): string {
  if (isOneTimeDailyQuota(subscription) && subscription.expires_at) {
    const parts = getRemainingDurationParts(subscription.expires_at)
    if (!parts) return t('userSubscriptions.windowNotActive')
    return t('userSubscriptions.quotaEndsIn', { time: formatDurationParts(parts) })
  }

  return t('userSubscriptions.resetIn', {
    time: formatResetTime(subscription.daily_window_start, 24)
  })
}

function formatQuotaUsageWindow(subscription: UserSubscription): string {
  const usage = quotaUsage(subscription)
  if (!usage) return t('userSubscriptions.windowNotActive')
  if (usage.period === 'daily') return formatDailyUsageWindow(subscription)
  const windowHours = usage.period === 'weekly' ? 168 : 720
  return t('userSubscriptions.resetIn', {
    time: formatResetTime(usage.windowStart, windowHours)
  })
}

function formatResetTime(windowStart: string | null, windowHours: number): string {
  if (!windowStart) return t('userSubscriptions.windowNotActive')

  const start = new Date(windowStart)
  const end = new Date(start.getTime() + windowHours * 60 * 60 * 1000)
  const parts = getRemainingDurationParts(end)

  return parts ? formatDurationParts(parts) : t('userSubscriptions.windowNotActive')
}

onMounted(() => {
  loadSubscriptions()
})
</script>
