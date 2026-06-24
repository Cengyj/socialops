<template>
  <CommonStatsGrid
    card-class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    grid-class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
    layout="stacked"
    :stats="normalizedStats"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CommonStatsGrid, { type CommonStatCard } from '@/components/common/CommonStatsGrid.vue'

const props = defineProps<{
  stats: Array<{
    label: string
    value: number
  }>
}>()

const normalizedStats = computed<CommonStatCard[]>(() =>
  props.stats.map(stat => ({
    key: stat.label,
    label: stat.label,
    value: stat.value,
  })),
)
</script>
