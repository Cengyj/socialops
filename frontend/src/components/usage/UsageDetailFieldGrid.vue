<template>
  <div class="grid gap-3 md:grid-cols-2">
    <div
      v-for="item in rows"
      :key="item.label"
      :class="cardClasses(item)"
    >
      <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
      <span v-if="item.badgeClass" :class="['badge mt-2', item.badgeClass]">{{ item.value }}</span>
      <div v-else :class="valueClasses(item)">{{ item.value }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">

export interface UsageDetailFieldRow {
  label: string
  value: string
  span?: 'auto' | 'full'
  valueTone?: 'normal' | 'money' | 'success' | 'danger' | 'muted' | 'technical'
  badgeClass?: string
  valueClass?: string
}

const props = withDefaults(defineProps<{
  rows: UsageDetailFieldRow[]
  tone?: 'default' | 'muted'
  valueStyle?: 'normal' | 'technical'
}>(), {
  tone: 'default',
  valueStyle: 'normal',
})

function cardClasses(item: UsageDetailFieldRow) {
  return [
    'rounded-xl border px-4 py-3',
    item.span === 'full' ? 'md:col-span-2' : '',
    props.tone === 'muted'
      ? 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800/60'
      : 'border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800',
  ]
}

function valueClasses(item: UsageDetailFieldRow) {
  const tone = item.valueTone || (props.valueStyle === 'technical' ? 'technical' : 'normal')
  const toneClass: Record<NonNullable<UsageDetailFieldRow['valueTone']>, string> = {
    normal: 'text-sm text-gray-900 dark:text-white',
    money: 'text-sm font-semibold tabular-nums text-green-600 dark:text-green-400',
    success: 'text-sm font-medium text-emerald-600 dark:text-emerald-400',
    danger: 'text-sm font-medium text-red-600 dark:text-red-400',
    muted: 'text-sm text-gray-600 dark:text-gray-300',
    technical: 'font-mono text-xs text-gray-600 dark:text-gray-300',
  }

  return [
    'mt-1 whitespace-pre-wrap break-words',
    item.valueClass || toneClass[tone],
  ]
}
</script>
