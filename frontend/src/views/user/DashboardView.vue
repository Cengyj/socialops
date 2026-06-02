<template>
  <AppLayout>
    <div class="space-y-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('dashboard.title') }}</h1>
      <div class="grid gap-4 md:grid-cols-2">
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.myAccounts') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ accountCount }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.tasksCompleted') }}</p>
          <p class="mt-2 text-2xl font-semibold text-green-600">{{ taskCount }}</p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import socialAccountsAPI from '@/api/socialAccounts'

const { t } = useI18n()
const accountCount = ref(0)
const taskCount = ref(0)

onMounted(async () => {
  try {
    const accounts = await socialAccountsAPI.listMyAccounts({ page: 1, page_size: 1 })
    accountCount.value = accounts.total || 0
  } catch { /* ignore */ }
  try {
    const tasks = await socialAccountsAPI.listMyTaskLogs({ page: 1, page_size: 1 })
    taskCount.value = tasks.total || 0
  } catch { /* ignore */ }
})
</script>
