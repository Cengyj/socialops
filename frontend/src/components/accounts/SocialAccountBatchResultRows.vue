<template>
  <div v-if="visibleRows.length > 0" class="grid min-w-0 gap-1.5" role="list">
    <div
      v-for="row in visibleRows"
      :key="row.key"
      :class="['grid min-w-0 gap-1 rounded-md px-2 py-1.5 text-xs sm:grid-cols-[minmax(0,1fr)_88px_minmax(0,1.2fr)]', row.toneClass]"
      role="listitem"
    >
      <span class="min-w-0 break-all font-medium sm:truncate" :title="row.label">{{ row.label }}</span>
      <span v-if="!combineStatusAndMessage" class="min-w-0 break-words" :title="row.statusLabel">{{ row.statusLabel }}</span>
      <span v-if="!combineStatusAndMessage" class="min-w-0 break-words opacity-85" :title="row.message">{{ row.message }}</span>
      <span v-else class="min-w-0 break-words text-xs opacity-85 sm:col-span-2 sm:text-right" :title="row.combinedMessage">
        {{ row.combinedMessage }}
      </span>
    </div>
    <div
      v-if="remainingCount > 0"
      class="min-w-0 break-words text-xs font-medium text-gray-500 dark:text-gray-400"
      role="listitem"
      aria-live="polite"
      aria-atomic="true"
      :title="rowsMoreText"
    >
      {{ rowsMoreText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { SocialAccountBatchItemResult } from '@/api/accountWorkbench'

const props = withDefaults(defineProps<{
  combineStatusAndMessage?: boolean
  itemLabel: (item: SocialAccountBatchItemResult, index: number) => string
  itemMessage: (item: SocialAccountBatchItemResult) => string
  items: SocialAccountBatchItemResult[]
  remainingCount: number
  rowToneClass: (status?: string | null) => string
  rowsMoreText: string
  statusLabel: (status?: string | null) => string
}>(), {
  combineStatusAndMessage: false,
})

const visibleRows = computed(() => props.items.map((item, index) => {
  const label = resultRowLabel(item, index)
  const statusLabel = resultRowText(props.statusLabel(item.status))
  const message = resultRowText(props.itemMessage(item))
  return {
    combinedMessage: combineResultRowMessage(statusLabel, message),
    key: `${item.id ?? index}-${item.status}-${item.reason || item.error || ''}`,
    label,
    message,
    statusLabel,
    toneClass: props.rowToneClass(item.status),
  }
}))

function resultRowLabel(item: SocialAccountBatchItemResult, index: number) {
  const fallback = typeof item.id === 'number' ? `#${item.id}` : `#${index + 1}`
  return resultRowText(props.itemLabel(item, index), fallback)
}

function resultRowText(value: string | null | undefined, fallback = '-') {
  const trimmed = String(value || '').trim()
  return trimmed || fallback
}

function combineResultRowMessage(statusLabel: string, message: string) {
  if (statusLabel === '-' && message === '-') return '-'
  if (statusLabel === '-') return message
  if (message === '-') return statusLabel
  return `${statusLabel} · ${message}`
}
</script>
