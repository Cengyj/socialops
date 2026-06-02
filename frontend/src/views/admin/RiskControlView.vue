<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.riskControl.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="loadData">
          {{ t('common.refresh') }}
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.runtimeStatus') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ statusLabel }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.accountRules') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">0</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.recentEvents') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ logs.length }}</p>
        </div>
      </div>

      <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200">
        {{ status.message || t('admin.riskControl.skeletonMessage') }}
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.riskControl.records') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.scope') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.target') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.status') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="row in logs" :key="row.id">
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ row.scope || '-' }}</td>
                <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ row.target || '-' }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ row.status || '-' }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.created_at) }}</td>
              </tr>
              <tr v-if="!loading && logs.length === 0">
                <td class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="4">
                  {{ t('admin.riskControl.empty') }}
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
import { adminRiskControlAPI, type RiskControlLog, type RiskControlStatus } from '@/api/admin/riskControl'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const status = ref<RiskControlStatus>({ enabled: false, status: 'disabled' })
const logs = ref<RiskControlLog[]>([])

const statusLabel = computed(() => {
  if (status.value.enabled) return t('admin.riskControl.enabled')
  return t('admin.riskControl.disabled')
})

async function loadData() {
  loading.value = true
  try {
    const [statusResult, logsResult] = await Promise.all([
      adminRiskControlAPI.getStatus(),
      adminRiskControlAPI.listLogs({ page: 1, page_size: 20 }),
    ])
    status.value = statusResult ?? { enabled: false, status: 'disabled' }
    logs.value = logsResult.items ?? []
  } catch (error) {
    appStore.showError(t('admin.riskControl.loadFailed'))
  } finally {
    loading.value = false
  }
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

onMounted(loadData)
</script>
