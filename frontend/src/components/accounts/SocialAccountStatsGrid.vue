<template>
  <CommonStatsGrid :grid-class="gridClass" :stats="normalizedStats" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CommonStatsGrid, { type CommonStatCard } from '@/components/common/CommonStatsGrid.vue'

export interface SocialAccountStatCard {
  key: string
  label: string
  value: string | number
  meta?: string
}

const props = withDefaults(defineProps<{
  gridClass?: string
  stats: SocialAccountStatCard[]
  testIdPrefix: string
}>(), {
  gridClass: 'grid gap-2 sm:grid-cols-2 xl:grid-cols-4',
})

const normalizedStats = computed<CommonStatCard[]>(() =>
  props.stats.map(stat => ({
    ...stat,
    testId: `${props.testIdPrefix}-${stat.key}`,
  })),
)
</script>
