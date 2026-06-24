<template>
  <section :data-testid="sectionTestId" :class="gridClass">
    <div
      v-for="stat in stats"
      :key="stat.key"
      :data-testid="stat.testId"
      :class="cardClass"
    >
      <div v-if="layout === 'stacked'">
        <div class="truncate text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
        <div :class="stackedValueClass">{{ stat.value }}</div>
      </div>
      <div v-else :class="['flex justify-between gap-3', stat.meta ? 'items-start' : 'items-center']">
        <div class="min-w-0">
          <div class="truncate text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
          <div v-if="stat.meta" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ stat.meta }}</div>
        </div>
        <div class="shrink-0 text-lg font-semibold leading-6 text-gray-900 dark:text-white">{{ stat.value }}</div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
export interface CommonStatCard {
  key: string
  label: string
  value: string | number
  meta?: string
  testId?: string
}

withDefaults(defineProps<{
  cardClass?: string
  gridClass?: string
  layout?: 'inline' | 'stacked'
  sectionTestId?: string
  stackedValueClass?: string
  stats: CommonStatCard[]
}>(), {
  cardClass: 'min-w-0 rounded-lg border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-dark-700 dark:bg-dark-800',
  gridClass: 'grid gap-2 sm:grid-cols-2 xl:grid-cols-4',
  layout: 'inline',
  sectionTestId: undefined,
  stackedValueClass: 'mt-1 text-xl font-semibold text-gray-900 dark:text-white',
})
</script>
