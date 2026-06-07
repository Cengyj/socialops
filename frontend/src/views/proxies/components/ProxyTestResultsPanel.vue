<template>
  <div v-if="results.length > 0" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('proxies.testResults.title') }}</div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('proxies.testResults.summary', summary) }}
        </div>
      </div>
      <button type="button" class="btn btn-secondary btn-sm h-8 shrink-0 justify-center" @click="emit('clear')">
        {{ t('common.clear') }}
      </button>
    </div>
    <div class="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="row in previewRows"
        :key="row.id"
        :class="['rounded-lg border px-3 py-2 text-xs', rowToneClass(row.status)]"
      >
        <div class="flex min-w-0 items-center justify-between gap-2">
          <span class="min-w-0 truncate font-medium">{{ proxyNameById(row.id) }}</span>
          <span class="shrink-0">{{ proxyStatusLabel(row.status) }}</span>
        </div>
        <div class="mt-1 text-xs opacity-80">
          {{ row.latency_ms ? `${row.latency_ms}ms` : '-' }}
          <span v-if="row.error"> · {{ row.error }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ProxyCheckResult } from '@/api/proxies'

defineProps<{
  previewRows: ProxyCheckResult[]
  proxyNameById: (id: number) => string
  proxyStatusLabel: (status: string) => string
  results: ProxyCheckResult[]
  rowToneClass: (status: string) => string
  summary: {
    total: number
    online: number
    offline: number
    unknown: number
  }
}>()

const emit = defineEmits<{
  clear: []
}>()

const { t } = useI18n()
</script>
