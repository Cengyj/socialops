<template>
  <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
    <div class="flex min-w-0 flex-1 flex-col items-start" :title="description || undefined">
      <GroupBadge
        :name="name"
        :platform="platform"
        :subscription-type="subscriptionType"
        :rate-multiplier="rateMultiplier"
        :show-rate="false"
        class="group-option-badge"
      />
      <span
        v-if="description"
        class="mt-1.5 line-clamp-2 w-full text-left text-xs leading-relaxed text-gray-500 dark:text-gray-400"
      >
        {{ description }}
      </span>
    </div>

    <div class="flex shrink-0 items-center gap-2 pt-0.5">
      <span
        v-if="rateMultiplier !== undefined && rateMultiplier !== null"
        :class="[
          'inline-flex items-center whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold',
          ratePillClass
        ]"
      >
        <template v-if="hasCustomRate">
          <span class="mr-1 line-through opacity-50">{{ rateMultiplier }}x</span>
          <span class="font-bold">{{ userRateMultiplier }}x</span>
        </template>
        <template v-else>
          {{ rateMultiplier }}x
        </template>
      </span>
      <Icon
        v-if="showCheckmark && selected"
        name="check"
        size="sm"
        class="shrink-0 text-primary-600 dark:text-primary-400"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SubscriptionType } from '@/types'
import { getPlatformColor } from '@/utils/platformColors'
import GroupBadge from './GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  name: string
  platform?: string | null
  subscriptionType?: SubscriptionType
  rateMultiplier?: number | null
  userRateMultiplier?: number | null
  description?: string | null
  selected?: boolean
  showCheckmark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  platform: 'social',
  subscriptionType: 'standard',
  selected: false,
  showCheckmark: true,
  userRateMultiplier: null
})

const normalizedPlatform = computed(() => {
  const value = String(props.platform || '').trim().toLowerCase()
  if (value === 'twitter' || value === 'x') return 'x_twitter'
  return value || 'social'
})

const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== null &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

const ratePillClass = computed(() => {
  const colors = getPlatformColor(normalizedPlatform.value)
  return `${colors.bg} ${colors.text}`
})
</script>

<style scoped>
.group-option-badge :deep(span.truncate) {
  font-weight: 600;
}
</style>
