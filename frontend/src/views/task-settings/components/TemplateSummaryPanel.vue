<template>
  <aside class="min-w-0 space-y-4">
    <section data-testid="template-summary-panel" class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('taskSettings.summary.title') }}</div>
      <div class="mt-3 space-y-3 text-sm">
        <div v-for="row in rows" :key="row.key" class="flex min-w-0 justify-between gap-3">
          <span class="min-w-0 break-words text-gray-500 dark:text-dark-400">{{ row.label }}</span>
          <span class="min-w-0 break-words text-right font-medium text-gray-900 dark:text-white" :title="String(row.value)">{{ row.value }}</span>
        </div>
      </div>
    </section>

    <section data-testid="template-validation-panel" class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div
        v-if="validationResult"
        role="status"
        aria-live="polite"
        aria-atomic="true"
        :class="[
          'min-w-0 break-words rounded-lg border p-3 text-sm',
          validationResult.valid
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300'
            : 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300'
        ]"
      >
        <div class="font-medium">{{ validationResult.valid ? t('taskSettings.validation.valid') : t('taskSettings.validation.invalid') }}</div>
        <ul v-if="normalizedValidationErrors.length > 0" class="mt-2 list-disc space-y-1 pl-5">
          <li v-for="error in normalizedValidationErrors" :key="error" class="min-w-0 break-words" :title="error">{{ error }}</li>
        </ul>
      </div>
      <div v-else-if="saveDisabledReason" class="min-w-0 break-words rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-300" role="status" aria-live="polite" aria-atomic="true" :title="saveDisabledReason">
        {{ saveDisabledReason }}
      </div>
      <div v-else class="min-w-0 break-words rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300" role="status" aria-live="polite" aria-atomic="true" :title="t('taskSettings.status.ready')">
        {{ t('taskSettings.status.ready') }}
      </div>

      <div class="mt-3 min-w-0 break-words rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm leading-6 text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200" role="status" aria-live="polite" aria-atomic="true" :title="executionHint">
        {{ executionHint }}
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TaskTemplateValidationResult } from '@/api/taskSettings'
import { normalizeTemplateValidationErrors } from '../templateValidationMessages'

export interface TemplateSummaryRow {
  key: string
  label: string
  value: string | number
}

const props = defineProps<{
  rows: TemplateSummaryRow[]
  saveDisabledReason: string
  validationResult: TaskTemplateValidationResult | null
}>()

const { t } = useI18n()
const executionHint = computed(() => t('taskSettings.summary.executionHint'))
const normalizedValidationErrors = computed(() => normalizeTemplateValidationErrors(props.validationResult?.errors, t))
</script>
