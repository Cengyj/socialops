<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.usage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.usage.description') }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadData">{{ t('common.refresh') }}</button>
          <button class="btn btn-primary" :disabled="cleanupLoading" @click="createCleanupTask">{{ t('admin.usage.cleanup.button') }}</button>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div v-for="item in statCards" :key="item.label" class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.usage.records') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.usage.user') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.operation') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.status') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.quantity') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="row in rows" :key="row.id">
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">#{{ row.user_id }}</td>
                <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ row.operation || '-' }}</td>
                <td class="px-5 py-3 text-sm">
                  <span :class="['badge', statusClass(row.status)]">{{ row.status || '-' }}</span>
                </td>
                <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ row.quantity ?? 0 }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.created_at) }}</td>
              </tr>
              <tr v-if="!loading && rows.length === 0">
                <td class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="5">
                  {{ t('admin.usage.empty') }}
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
import { adminUsageAPI } from '@/api/admin/usage'
import type { UsageLog, UsageStats } from '@/api/usage'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const cleanupLoading = ref(false)
const rows = ref<UsageLog[]>([])
const stats = ref<UsageStats>({})

const statCards = computed(() => [
  { label: t('usage.totalOperations'), value: String(stats.value.total_operations ?? stats.value.total_requests ?? rows.value.length) },
  { label: t('usage.totalQuantity'), value: String(stats.value.total_quantity ?? rows.value.reduce((sum, row) => sum + (row.quantity || 0), 0)) },
  { label: t('usage.successCount'), value: String(stats.value.success_count ?? rows.value.filter(row => row.status === 'success').length) },
  { label: t('usage.failedCount'), value: String(stats.value.failed_count ?? rows.value.filter(row => row.status === 'failed').length) },
])

async function loadData() {
  loading.value = true
  try {
    const [listResult, statsResult] = await Promise.all([
      adminUsageAPI.list({ page: 1, page_size: 20 }),
      adminUsageAPI.getStats(),
    ])
    rows.value = listResult.items ?? []
    stats.value = statsResult ?? {}
  } catch (error) {
    appStore.showError(t('admin.usage.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function createCleanupTask() {
  cleanupLoading.value = true
  try {
    const result = await adminUsageAPI.createCleanupTask({})
    appStore.showSuccess(result.message || t('admin.usage.cleanupQueued'))
  } catch (error) {
    appStore.showError(t('admin.usage.cleanupFailed'))
  } finally {
    cleanupLoading.value = false
  }
}

function statusClass(status: string) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-error'
  return 'badge-warning'
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

onMounted(loadData)
</script>
