<template>
  <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
    <div
      v-for="item in items"
      :key="item.label"
      :class="['card card-hover p-4', item.cardClass]"
      data-testid="usage-stat-card"
    >
      <div class="flex items-center gap-3">
        <div :class="['flex h-11 w-11 shrink-0 items-center justify-center rounded-xl', item.iconWrapClass]">
          <Icon :name="item.icon" size="md" :class="item.iconClass" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p :class="['mt-1 truncate text-xl font-bold tabular-nums', item.valueClass]">{{ item.value }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ item.meta }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  items: Array<{
    label: string
    value: string
    meta: string
    icon: 'chart' | 'checkCircle' | 'xCircle' | 'trendingUp' | 'dollar'
    iconWrapClass: string
    iconClass: string
    valueClass: string
    cardClass?: string
  }>
}>()
</script>
