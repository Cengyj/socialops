<template>
  <div :data-testid="testId" class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 text-sm font-medium text-gray-900 dark:text-white">{{ title }}</div>
    <div class="mb-3 rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-900/60 dark:bg-primary-900/20 dark:text-primary-300" role="status" aria-live="polite" aria-atomic="true" :title="summary">
      {{ summary }}
    </div>
    <SocialAccountBatchResultRows
      :combine-status-and-message="combineStatusAndMessage"
      :items="items"
      :remaining-count="remainingCount"
      :rows-more-text="rowsMoreText"
      :item-label="itemLabel"
      :status-label="statusLabel"
      :item-message="itemMessage"
      :row-tone-class="rowToneClass"
    />
  </div>
</template>

<script setup lang="ts">
import type { SocialAccountBatchItemResult } from '@/api/accountWorkbench'
import SocialAccountBatchResultRows from '@/components/accounts/SocialAccountBatchResultRows.vue'

withDefaults(defineProps<{
  combineStatusAndMessage?: boolean
  itemLabel: (item: SocialAccountBatchItemResult, index: number) => string
  itemMessage: (item: SocialAccountBatchItemResult) => string
  items: SocialAccountBatchItemResult[]
  remainingCount: number
  rowToneClass: (status?: string | null) => string
  rowsMoreText: string
  statusLabel: (status?: string | null) => string
  summary: string
  testId?: string
  title: string
}>(), {
  combineStatusAndMessage: true,
  testId: undefined,
})
</script>
