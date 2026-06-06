<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ title }}</h1>
              <p v-if="description" class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">{{ description }}</p>
            </div>
            <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
              <slot name="actions" />
            </div>
          </div>

          <slot name="alerts" />

          <div v-if="stats.length > 0" :class="statGridClass">
            <div
              v-for="stat in stats"
              :key="stat.label"
              class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
              <div v-if="stat.meta" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ stat.meta }}</div>
            </div>
          </div>

          <slot name="controls" />
          <slot name="import" />
        </div>
      </template>

      <template #table>
        <slot name="table" />
      </template>
    </TablePageLayout>

    <slot name="after-table" />
    <slot name="dialogs" />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'

export interface AccountWorkbenchStat {
  label: string
  value: string | number
  meta?: string
}

withDefaults(defineProps<{
  title: string
  description?: string
  stats?: AccountWorkbenchStat[]
  statGridClass?: string
}>(), {
  description: '',
  stats: () => [],
  statGridClass: 'grid gap-3 md:grid-cols-4',
})
</script>
