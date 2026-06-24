<template>
  <div :data-testid="testId" class="border-t border-gray-100 pt-3 text-sm dark:border-dark-700">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0 break-words font-medium text-gray-900 dark:text-white" role="status" aria-live="polite" aria-atomic="true" :title="summary">{{ summary }}</div>
      <button
        type="button"
        class="btn btn-secondary btn-sm h-8 w-8 min-w-[2rem] max-w-[2rem] shrink-0 justify-center px-0"
        :aria-label="dismissLabel"
        :title="dismissLabel"
        @click="emit('dismiss')"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>
    <SocialAccountBatchResultRows
      class="mt-2"
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
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  dismissLabel: string
  itemLabel: (item: SocialAccountBatchItemResult, index: number) => string
  itemMessage: (item: SocialAccountBatchItemResult) => string
  items: SocialAccountBatchItemResult[]
  remainingCount: number
  rowToneClass: (status?: string | null) => string
  rowsMoreText: string
  statusLabel: (status?: string | null) => string
  summary: string
  testId: string
}>()

const emit = defineEmits<{
  dismiss: []
}>()
</script>
