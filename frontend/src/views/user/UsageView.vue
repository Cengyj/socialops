<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('usage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('usage.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="loadData">
          {{ t('common.refresh') }}
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div v-for="item in statCards" :key="item.label" class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('usage.records') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.operation') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.platform') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.account') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.status') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.quantity') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.chargeStatus') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.cost') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.result') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="row in rows" :key="row.id">
                <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ actionLabel(row.operation) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ platformLabel(row.platform) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ row.account_name || '-' }}</td>
                <td class="px-5 py-3 text-sm">
                  <span :class="['badge', statusClass(row.status)]">{{ statusLabel(row.status) }}</span>
                </td>
                <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatNumber(row.quantity) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ chargeStatusLabel(row.charge_status) }}</td>
                <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatCurrency(row.cost) }}</td>
                <td class="max-w-xs px-5 py-3 text-sm text-gray-600 dark:text-gray-300">
                  <span class="line-clamp-2">{{ resultMessage(row) }}</span>
                </td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.completed_at || row.created_at) }}</td>
              </tr>
              <tr v-if="!loading && rows.length === 0">
                <td class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="9">
                  {{ t('usage.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { usageAPI } from '@/api/usage'
import type { UsageLog, UsageStats } from '@/api/usage'
import { useAppStore } from '@/stores/app'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const rows = ref<UsageLog[]>([])
const usageStats = ref<UsageStats | null>(null)
const successStats = ref<UsageStats | null>(null)

const statCards = computed(() => [
  { label: t('usage.totalOperations'), value: formatNumber(usageStats.value?.total_requests ?? rows.value.length) },
  { label: t('usage.totalQuantity'), value: formatNumber(usageStats.value?.total_tokens ?? usageStats.value?.total_quantity ?? rows.value.reduce((sum, row) => sum + (row.quantity || 0), 0)) },
  { label: t('usage.successCount'), value: formatNumber(successStats.value?.total_requests ?? rows.value.filter(row => row.status === 'success').length) },
  { label: t('usage.totalCost'), value: formatCurrency(usageStats.value?.total_actual_cost ?? usageStats.value?.total_cost ?? rows.value.reduce((sum, row) => sum + (row.cost || 0), 0)) },
])

async function loadData() {
  loading.value = true
  try {
    const [listResult, statsResult, successStatsResult] = await Promise.allSettled([
      usageAPI.list({ page: 1, page_size: 20 }),
      usageAPI.getStats(),
      usageAPI.getStats({ status: 'success' }),
    ])
    if (listResult.status === 'rejected') {
      throw listResult.reason
    }
    rows.value = listResult.value.items ?? []
    usageStats.value = statsResult.status === 'fulfilled' ? statsResult.value : null
    successStats.value = successStatsResult.status === 'fulfilled' ? successStatsResult.value : null
  } catch (error) {
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function statusClass(status: string) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-error'
  return 'badge-warning'
}

function actionLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return t('common.unknown')
  const key = `usage.actions.${normalized}`
  const translated = t(key)
  return translated === key ? value || t('common.unknown') : translated
}

function statusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.statuses.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function platformLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.platforms.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function chargeStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.chargeStatuses.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function resultMessage(row: UsageLog) {
  return formatSocialTaskResultMessage(row, t)
}

function formatCurrency(value?: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function formatNumber(value?: number) {
  return Number(value || 0).toLocaleString()
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

onMounted(loadData)
</script>
