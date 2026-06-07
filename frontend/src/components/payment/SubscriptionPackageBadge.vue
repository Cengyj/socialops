<template>
  <div :class="['flex min-w-0 items-start gap-3', compact ? 'gap-2' : 'gap-3']">
    <div
      :class="[
        'flex shrink-0 items-center justify-center rounded-lg border',
        compact ? 'h-8 w-8' : 'h-9 w-9',
        platformColors.bg,
        platformColors.text,
        platformColors.border,
      ]"
    >
      <SubscriptionPlatformLogo :platform="normalizedPlatform" :compact="compact" />
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {{ name || fallbackName }}
        </span>
        <span
          :class="[
            'shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium',
            platformColors.bg,
            platformColors.text,
          ]"
        >
          {{ platformLabel }}
        </span>
        <span
          v-if="hidden"
          class="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
        >
          {{ hiddenText }}
        </span>
      </div>

      <div
        v-if="description || quotaDisplay || validityLabel"
        class="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-gray-400"
      >
        <span v-if="quotaDisplay" class="whitespace-nowrap">{{ quotaDisplay }}</span>
        <span v-if="quotaDisplay && validityLabel" class="text-gray-300 dark:text-dark-500">/</span>
        <span v-if="validityLabel" class="whitespace-nowrap">{{ validityLabel }}</span>
        <span v-if="description" class="min-w-0 truncate">{{ description }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SubscriptionPlatformLogo from './SubscriptionPlatformLogo.vue'
import { getPlatformColor } from '@/utils/platformColors'
import { getSubscriptionPlatformLabel, normalizePlanPlatformLabelKey } from '@/utils/subscriptionPlanDisplay'

const props = withDefaults(defineProps<{
  name?: string | null
  platform?: string | null
  quotaDisplay?: string | null
  validityLabel?: string | null
  description?: string | null
  hidden?: boolean
  compact?: boolean
  hiddenLabel?: string
}>(), {
  name: '',
  platform: 'social',
  quotaDisplay: '',
  validityLabel: '',
  description: '',
  hidden: false,
  compact: false,
  hiddenLabel: '',
})

const { t } = useI18n()

const fallbackName = computed(() => t('payment.admin.packageLabel'))
const hiddenText = computed(() => props.hiddenLabel || t('payment.admin.hidden'))
const normalizedPlatform = computed(() => normalizePlanPlatformLabelKey(props.platform))
const platformColors = computed(() => getPlatformColor(normalizedPlatform.value))
const platformLabel = computed(() => {
  return getSubscriptionPlatformLabel(props.platform, t('payment.platformFallback'))
})
</script>
