<template>
  <div class="space-y-5">
    <div v-if="quotaPackages.length === 0" class="card py-16 text-center">
      <Icon name="gift" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
      <p class="text-gray-500 dark:text-gray-400">{{ t('payment.noPlans') }}</p>
    </div>

    <div v-else class="space-y-4">
      <article
        v-for="quotaPackage in quotaPackages"
        :key="quotaPackage.key"
        class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(quotaPackage.platform)]">
                <SubscriptionPlatformLogo :platform="quotaPackage.platform" compact />
                {{ platformLabel(quotaPackage.platform) }}
              </span>
              <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                {{ packageTitle(quotaPackage) }}
              </h3>
            </div>
            <p v-if="quotaPackage.description" class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
              {{ quotaPackage.description }}
            </p>
          </div>
          <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
            {{ validityLabel(quotaPackage.defaultPlan) }}
          </span>
        </div>

        <div class="mt-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('payment.subscriptionPicker.chooseQuota') }}
            </span>
            <span class="text-xs text-gray-400 dark:text-gray-500">
              {{ t('payment.subscriptionPicker.executionOnly') }}
            </span>
          </div>
          <SubscriptionQuotaChoiceList
            :quota-package="quotaPackage"
            :selected-plan-id="selectedPlanForPackage(quotaPackage).id"
            :format-amount="formatAmount"
            :unlimited-label="t('payment.planCard.unlimited')"
            @select="selectPackagePlan(quotaPackage, $event)"
          />
        </div>

        <div class="mt-4 rounded-lg border border-gray-100 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-700/40">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ planQuotaLabel(selectedPlanForPackage(quotaPackage)) }}</div>
              <div class="mt-0.5 text-lg font-semibold text-gray-900 dark:text-white">
                {{ quotaLabel(selectedPlanForPackage(quotaPackage)) }}
              </div>
            </div>
            <div class="text-right">
              <div v-if="selectedPlanForPackage(quotaPackage).original_price" class="text-xs text-gray-400 line-through">
                {{ formatAmount(selectedPlanForPackage(quotaPackage).original_price || 0) }}
              </div>
              <div :class="['text-xl font-bold', platformTextClass(quotaPackage.platform)]">
                {{ formatAmount(selectedPlanForPackage(quotaPackage).price) }}
              </div>
            </div>
          </div>
          <div v-if="selectedPlanForPackage(quotaPackage).features.length > 0" class="mt-3 flex flex-wrap gap-2">
            <span
              v-for="feature in selectedPlanForPackage(quotaPackage).features"
              :key="feature"
              class="rounded-full bg-white px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
            >
              {{ feature }}
            </span>
          </div>
        </div>

        <button
          type="button"
          :class="[
            'mt-4 w-full rounded-lg px-4 py-2.5 text-sm font-semibold transition-colors',
            primaryButtonClass(quotaPackage.platform),
          ]"
          @click="emit('select', selectedPlanForPackage(quotaPackage))"
        >
          {{ isRenewal(selectedPlanForPackage(quotaPackage)) ? t('payment.renewNow') : t('payment.subscribeNow') }}
        </button>
      </article>
    </div>

    <div v-if="showActiveSubscriptions && activeSubscriptions.length > 0" class="space-y-2">
      <p class="text-xs font-medium text-gray-400 dark:text-gray-500">{{ t('payment.activeSubscription') }}</p>
      <div class="space-y-2">
        <div
          v-for="subscription in activeSubscriptions"
          :key="subscription.id"
          class="flex items-center gap-3 rounded-lg border border-gray-100 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800"
        >
          <div :class="['h-6 w-1 shrink-0 rounded-full', platformAccentBarClass(subscriptionPlatform(subscription))]" />
          <div class="min-w-0 flex-1">
            <div class="flex min-w-0 items-center gap-1.5">
              <span class="truncate text-xs font-semibold text-gray-900 dark:text-white">
                {{ subscriptionTitle(subscription) || t('payment.packageFallback', { id: subscription.plan_id || subscription.group_id }) }}
              </span>
              <span :class="['inline-flex shrink-0 items-center gap-1 rounded-full px-1.5 py-0.5 text-[9px] font-medium', platformBadgeLightClass(subscriptionPlatform(subscription))]">
                <SubscriptionPlatformLogo :platform="subscriptionPlatform(subscription)" compact />
                {{ platformLabel(subscriptionPlatform(subscription)) }}
              </span>
            </div>
            <div class="flex flex-wrap gap-x-3 text-[11px] text-gray-400 dark:text-gray-500">
              <span>{{ subscriptionQuotaSummary(subscription) }}</span>
              <span v-if="subscription.expires_at">
                {{ t('userSubscriptions.daysRemaining', { days: daysRemaining(subscription.expires_at) }) }}
              </span>
              <span v-else>{{ t('userSubscriptions.noExpiration') }}</span>
            </div>
          </div>
          <span class="badge badge-success shrink-0 text-[10px]">{{ t('userSubscriptions.status.active') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import SubscriptionPlatformLogo from '@/components/payment/SubscriptionPlatformLogo.vue'
import SubscriptionQuotaChoiceList from '@/components/payment/SubscriptionQuotaChoiceList.vue'
import { getPlatformColor } from '@/utils/platformColors'
import {
  buildSubscriptionQuotaPackages,
  type SubscriptionQuotaPackage,
} from '@/utils/subscriptionPlanCatalog'
import {
  getPlanQuotaAmount,
  getPlanQuotaPeriod,
  getSubscriptionQuotaUsage,
} from '@/utils/subscriptionQuotaPlans'
import {
  getSubscriptionPlatform,
  getSubscriptionTitle,
  subscriptionMatchesPlan,
} from '@/utils/subscriptionPackages'
import {
  formatSubscriptionPlanValidity,
  formatSubscriptionQuotaAmount,
  getSubscriptionPlatformLabel,
} from '@/utils/subscriptionPlanDisplay'

const props = withDefaults(defineProps<{
  plans: SubscriptionPlan[]
  activeSubscriptions?: UserSubscription[]
  formatAmount: (value: number) => string
  showActiveSubscriptions?: boolean
  initialSelectedPlanIds?: number[]
}>(), {
  activeSubscriptions: () => [],
  showActiveSubscriptions: true,
  initialSelectedPlanIds: () => [],
})

const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t, locale } = useI18n()
const selectedPlanIDs = reactive<Record<string, number>>({})

const quotaPackages = computed(() => buildSubscriptionQuotaPackages(props.plans))
const activeSubscriptions = computed(() => props.activeSubscriptions || [])

watch(
  [quotaPackages, () => props.initialSelectedPlanIds],
  ([items]) => {
    const activeKeys = new Set(items.map((quotaPackage) => quotaPackage.key))
    for (const key of Object.keys(selectedPlanIDs)) {
      if (!activeKeys.has(key)) delete selectedPlanIDs[key]
    }
    for (const quotaPackage of items) {
      const currentPlanID = selectedPlanIDs[quotaPackage.key]
      const currentStillAvailable = quotaPackage.plans.some((plan) => plan.id === currentPlanID)
      if (currentStillAvailable) continue
      selectedPlanIDs[quotaPackage.key] = initialSelectedPlanForPackage(quotaPackage)?.id || quotaPackage.defaultPlan.id
    }
  },
  { immediate: true },
)

function selectedPlanForPackage(quotaPackage: SubscriptionQuotaPackage): SubscriptionPlan {
  return quotaPackage.plans.find((plan) => plan.id === selectedPlanIDs[quotaPackage.key]) || quotaPackage.defaultPlan
}

function selectPackagePlan(quotaPackage: SubscriptionQuotaPackage, plan: SubscriptionPlan) {
  selectedPlanIDs[quotaPackage.key] = plan.id
}

function initialSelectedPlanForPackage(quotaPackage: SubscriptionQuotaPackage): SubscriptionPlan | null {
  if (!props.initialSelectedPlanIds.length) return null
  const selectedIDs = new Set(props.initialSelectedPlanIds)
  return quotaPackage.plans.find((plan) => selectedIDs.has(plan.id)) || null
}

function packageTitle(quotaPackage: SubscriptionQuotaPackage): string {
  return quotaPackage.title || t('payment.subscriptionPicker.packageFallback', {
    platform: platformLabel(quotaPackage.platform),
    validity: validityLabel(quotaPackage.defaultPlan),
  })
}

function validityLabel(plan: SubscriptionPlan): string {
  return formatSubscriptionPlanValidity(plan, t, locale.value)
}

function quotaLabel(plan: SubscriptionPlan): string {
  return formatSubscriptionQuotaAmount(getPlanQuotaAmount(plan), t('payment.planCard.unlimited'))
}

function quotaPeriodLabel(period: 'daily' | 'weekly' | 'monthly' | null | undefined): string {
  if (period === 'daily') return t('payment.planCard.todayQuota')
  if (period === 'weekly') return t('payment.planCard.thisWeekQuota')
  if (period === 'monthly') return t('payment.planCard.thisMonthQuota')
  return t('payment.planCard.periodQuota')
}

function planQuotaLabel(plan: SubscriptionPlan): string {
  return quotaPeriodLabel(getPlanQuotaPeriod(plan))
}

function isRenewal(plan: SubscriptionPlan): boolean {
  return activeSubscriptions.value.some((sub) => sub.status === 'active' && subscriptionMatchesPlan(sub, plan))
}

function platformLabel(platform: string): string {
  return getSubscriptionPlatformLabel(platform, t('payment.platformFallback'))
}

function platformBadgeClass(platform: string): string {
  const colors = getPlatformColor(platform)
  return `${colors.bg} ${colors.text} ${colors.border}`
}

function platformBadgeLightClass(platform: string): string {
  const colors = getPlatformColor(platform)
  return `${colors.bg} ${colors.text}`
}

function platformTextClass(platform: string): string {
  return getPlatformColor(platform).text
}

function primaryButtonClass(platform: string): string {
  const normalized = String(platform || '').toLowerCase()
  if (normalized === 'instagram') return 'bg-pink-600 text-white hover:bg-pink-700'
  if (normalized === 'facebook') return 'bg-blue-600 text-white hover:bg-blue-700'
  return 'bg-primary-600 text-white hover:bg-primary-700'
}

function platformAccentBarClass(platform: string): string {
  switch (platform) {
    case 'instagram':
      return 'bg-pink-500'
    case 'tiktok':
      return 'bg-gray-900 dark:bg-gray-100'
    case 'facebook':
      return 'bg-blue-500'
    case 'x_twitter':
    default:
      return 'bg-gray-500'
  }
}

function subscriptionPlatform(subscription: UserSubscription): string {
  return getSubscriptionPlatform(subscription)
}

function subscriptionTitle(subscription: UserSubscription): string | null {
  return getSubscriptionTitle(subscription)
}

function subscriptionQuotaSummary(subscription: UserSubscription): string {
  const usage = getSubscriptionQuotaUsage(subscription)
  return usage
    ? `${quotaPeriodLabel(usage.period)}: $${usage.amount.toFixed(2)}`
    : `${t('payment.planCard.quota')}: ${t('payment.planCard.unlimited')}`
}

function daysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}
</script>
