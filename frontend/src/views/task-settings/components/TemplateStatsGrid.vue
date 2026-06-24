<template>
  <CommonStatsGrid
    card-class="min-w-0 rounded-lg border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    grid-class="grid min-w-0 gap-2 sm:grid-cols-3"
    section-test-id="template-stats"
    :stats="normalizedStats"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CommonStatsGrid, { type CommonStatCard } from '@/components/common/CommonStatsGrid.vue'

export interface TemplateStatCard {
  testId: string
  label: string
  meta: string
  value: number | string
}

const props = defineProps<{
  stats: TemplateStatCard[]
}>()

const normalizedStats = computed<CommonStatCard[]>(() =>
  props.stats.map(stat => ({
    key: stat.testId,
    label: stat.label,
    meta: stat.meta,
    testId: stat.testId,
    value: stat.value,
  })),
)
</script>
