<template>
  <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
    <div
      v-for="stat in stats"
      :key="stat.label"
      class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
      <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  stats: Array<{
    label: string
    value: number
  }>
}>()
</script>
