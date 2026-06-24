<template>
  <div class="grid gap-3 md:grid-cols-2">
    <div
      v-for="card in cards"
      :key="card.title"
      class="rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ card.title }}</div>
      <div
        v-if="card.previewSrc"
        class="mt-3 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-700"
      >
        <img
          :src="card.previewSrc"
          :alt="card.title"
          :data-testid="card.previewTestId"
          class="h-40 w-full object-cover"
        >
      </div>
      <div class="mt-3 space-y-2">
        <div v-for="item in card.rows" :key="item.label" class="rounded-lg bg-gray-50 px-3 py-2 text-sm text-gray-900 dark:bg-dark-700 dark:text-white">
          <span class="font-medium text-gray-600 dark:text-gray-300">{{ item.label }}:</span>
          <span class="ml-2 break-words">{{ item.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface UsageDetailMediaRow {
  label: string
  value: string
}

export interface UsageDetailMediaCard {
  title: string
  rows: UsageDetailMediaRow[]
  previewSrc: string
  previewTestId: string
}

defineProps<{
  cards: UsageDetailMediaCard[]
}>()
</script>
