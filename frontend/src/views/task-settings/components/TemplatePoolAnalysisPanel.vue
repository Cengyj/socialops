<template>
  <div data-testid="template-pool-analysis-panel" class="min-w-0 space-y-3">
    <div class="grid min-w-0 grid-cols-2 gap-2">
      <div class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-valid">
        <p class="min-w-0 break-words text-xs text-gray-500 dark:text-dark-400" :title="t('taskSettings.pool.valid')">{{ t('taskSettings.pool.valid') }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ analysis.validCount }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-empty-lines">
        <p class="min-w-0 break-words text-xs text-gray-500 dark:text-dark-400" :title="t('taskSettings.pool.emptyLines')">{{ t('taskSettings.pool.emptyLines') }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ analysis.emptyLineCount }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-duplicates">
        <p class="min-w-0 break-words text-xs text-gray-500 dark:text-dark-400" :title="t('taskSettings.pool.duplicates')">{{ t('taskSettings.pool.duplicates') }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ analysis.duplicateCount }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="pool-too-long">
        <p class="min-w-0 break-words text-xs text-gray-500 dark:text-dark-400" :title="t('taskSettings.pool.tooLong')">{{ t('taskSettings.pool.tooLong') }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ analysis.tooLongCount }}</p>
      </div>
      <div class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <p class="min-w-0 break-words text-xs text-gray-500 dark:text-dark-400" :title="t('taskSettings.pool.remaining')">{{ t('taskSettings.pool.remaining') }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ analysis.remaining }}</p>
      </div>
    </div>

    <div
      data-testid="pool-capacity"
      :role="analysis.overCapacity ? 'status' : undefined"
      :aria-live="analysis.overCapacity ? 'polite' : undefined"
      :aria-atomic="analysis.overCapacity ? 'true' : undefined"
      :class="[
        'min-w-0 break-words rounded-lg border p-3 text-sm',
        analysis.overCapacity
          ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300'
          : 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-900/50 dark:text-dark-300'
      ]"
      :title="capacityMessage"
    >
      {{ capacityMessage }}
    </div>

    <div
      v-if="analysis.emptyLineCount > 0"
      data-testid="pool-empty-lines-hint"
      class="min-w-0 break-words rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200"
      role="status"
      aria-live="polite"
      aria-atomic="true"
      :title="t('taskSettings.pool.emptyLinesHint', { count: analysis.emptyLineCount })"
    >
      {{ t('taskSettings.pool.emptyLinesHint', { count: analysis.emptyLineCount }) }}
    </div>
    <div
      v-if="analysis.tooLongCount > 0"
      data-testid="pool-too-long-hint"
      class="min-w-0 break-words rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
      role="status"
      aria-live="polite"
      aria-atomic="true"
      :title="t('taskSettings.pool.tooLongHint', { max: maxValueLength })"
    >
      {{ t('taskSettings.pool.tooLongHint', { max: maxValueLength }) }}
    </div>
    <div
      v-else-if="analysis.duplicateCount > 0"
      data-testid="pool-duplicate-hint"
      class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300"
      role="status"
      aria-live="polite"
      aria-atomic="true"
      :title="t('taskSettings.pool.duplicateHint', { count: analysis.duplicateCount })"
    >
      {{ t('taskSettings.pool.duplicateHint', { count: analysis.duplicateCount }) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TemplatePoolAnalysis } from '../templatePool'

defineProps<{
  analysis: TemplatePoolAnalysis
  capacityMessage: string
  maxValueLength: number
}>()

const { t } = useI18n()
</script>
