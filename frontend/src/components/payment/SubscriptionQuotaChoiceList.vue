<template>
  <div class="flex flex-wrap gap-2">
    <button
      v-for="choice in quotaChoices"
      :key="choice.planID"
      type="button"
      :class="[
        minWidthClass,
        'rounded-lg border px-3 py-2 text-left transition-colors',
        isSelected(choice.planID)
          ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
          : 'border-gray-200 text-gray-700 hover:bg-gray-50 dark:border-dark-700 dark:text-gray-200 dark:hover:bg-dark-700',
      ]"
      :aria-pressed="isSelected(choice.planID)"
      @click="emit('select', choice.plan)"
    >
      <span class="flex items-center gap-1.5 text-sm font-semibold">
        {{ formatSubscriptionQuotaAmount(choice.quotaUSD, unlimitedLabel) }}
        <span
          v-if="choice.isBestValue"
          class="rounded bg-emerald-50 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200"
        >
          {{ t('payment.subscriptionPicker.bestValue') }}
        </span>
      </span>
      <span class="block text-xs text-gray-500 dark:text-gray-400">{{ formatAmount(choice.price) }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import {
  getSubscriptionQuotaChoices,
  type SubscriptionQuotaPackage,
} from '@/utils/subscriptionPlanCatalog'
import { formatSubscriptionQuotaAmount } from '@/utils/subscriptionPlanDisplay'

const props = withDefaults(defineProps<{
  quotaPackage: SubscriptionQuotaPackage
  selectedPlanId: number | null | undefined
  formatAmount: (value: number) => string
  unlimitedLabel?: string
  minWidthClass?: string
}>(), {
  unlimitedLabel: '',
  minWidthClass: 'min-w-[112px]',
})

const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const quotaChoices = computed(() => getSubscriptionQuotaChoices(props.quotaPackage))
const unlimitedLabel = computed(() => props.unlimitedLabel || t('payment.planCard.unlimited'))

function isSelected(planID: number): boolean {
  return props.selectedPlanId === planID
}
</script>
