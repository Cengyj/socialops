<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="grid gap-4 md:grid-cols-4">
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.totalAccounts') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ stats.total_accounts }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.availableAccounts') }}</p>
          <p class="mt-2 text-2xl font-semibold text-green-600">{{ stats.available_accounts }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.totalUsers') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ stats.total_users }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.tasksToday') }}</p>
          <p class="mt-2 text-2xl font-semibold text-blue-600">{{ stats.tasks_today }}</p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'

const { t } = useI18n()

const stats = ref({
  total_accounts: 0,
  available_accounts: 0,
  total_users: 0,
  active_users: 0,
  tasks_today: 0,
})

onMounted(async () => {
  const [dashboardResult, socialAccountResult] = await Promise.allSettled([
    adminAPI.dashboard.getStats(),
    adminAPI.socialAccounts.getStats(),
  ])

  if (dashboardResult.status === 'fulfilled') {
    stats.value.total_users = dashboardResult.value.total_users || 0
    stats.value.active_users = dashboardResult.value.active_users || 0
    stats.value.tasks_today = dashboardResult.value.today_requests || 0
  }

  if (socialAccountResult.status === 'fulfilled') {
    stats.value.total_accounts = socialAccountResult.value.total || 0
    stats.value.available_accounts = socialAccountResult.value.available || 0
  }
})
</script>
