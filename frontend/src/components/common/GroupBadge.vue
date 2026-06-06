<template>
  <span
    :class="[
      'inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium transition-colors',
      badgeClass
    ]"
  >
    <span :class="['h-1.5 w-1.5 shrink-0 rounded-full', dotClass]" aria-hidden="true"></span>
    <span class="truncate">{{ name }}</span>
    <span v-if="showLabel" :class="labelClass">
      <template v-if="hasCustomRate">
        <span class="mr-0.5 line-through opacity-50">{{ formatRate(rateMultiplier) }}</span>
        <span class="font-bold">{{ formatRate(userRateMultiplier) }}</span>
      </template>
      <template v-else>
        {{ labelText }}
      </template>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionType } from '@/types'
import { getPlatformColor } from '@/utils/platformColors'

interface Props {
  name: string
  platform?: string | null
  subscriptionType?: SubscriptionType
  rateMultiplier?: number | null
  userRateMultiplier?: number | null
  showRate?: boolean
  daysRemaining?: number | null
  alwaysShowRate?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  platform: 'social',
  subscriptionType: 'standard',
  showRate: true,
  daysRemaining: null,
  userRateMultiplier: null,
  alwaysShowRate: false
})

const { t } = useI18n()

const normalizedPlatform = computed(() => {
  const value = String(props.platform || '').trim().toLowerCase()
  if (value === 'twitter' || value === 'x') return 'x_twitter'
  return value || 'social'
})

const isSubscription = computed(() => props.subscriptionType === 'subscription')

const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== null &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

const showLabel = computed(() => {
  if (!props.showRate) return false
  if (isSubscription.value && !props.alwaysShowRate) return true
  return props.rateMultiplier !== null && props.rateMultiplier !== undefined
})

const labelText = computed(() => {
  if (isSubscription.value && !props.alwaysShowRate) {
    if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
      if (props.daysRemaining <= 0) return t('admin.users.expired')
      return t('admin.users.daysRemaining', { days: props.daysRemaining })
    }
    return t('groups.subscription')
  }
  return formatRate(props.rateMultiplier)
})

const badgeClass = computed(() => {
  const colors = getPlatformColor(normalizedPlatform.value)
  return `${colors.bg} ${colors.text} ${colors.border}`
})

const labelClass = computed(() => {
  const base = 'rounded px-1.5 py-0.5 text-[10px] font-semibold'
  if (isSubscription.value && !props.alwaysShowRate) {
    if (props.daysRemaining !== null && props.daysRemaining !== undefined && props.daysRemaining <= 7) {
      return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300`
    }
    return `${base} bg-black/10 dark:bg-white/10`
  }
  return `${base} bg-black/10 dark:bg-white/10`
})

const dotClass = computed(() => {
  switch (normalizedPlatform.value) {
    case 'instagram':
      return 'bg-pink-500'
    case 'facebook':
      return 'bg-blue-500'
    case 'tiktok':
      return 'bg-gray-900 dark:bg-gray-100'
    case 'x_twitter':
      return 'bg-gray-500'
    default:
      return 'bg-primary-500'
  }
})

function formatRate(value: number | null | undefined): string {
  if (value === null || value === undefined) return ''
  return `${value}x`
}
</script>
