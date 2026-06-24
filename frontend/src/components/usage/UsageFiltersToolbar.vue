<template>
  <div class="card">
    <div class="px-6 py-4">
      <div class="flex flex-wrap items-end gap-4" data-testid="usage-filter-toolbar">
        <div class="w-full min-w-[160px] sm:w-44">
          <label class="input-label">{{ t('usage.platform') }}</label>
          <Select
            :model-value="platform"
            :options="platformOptions"
            class="w-full"
            @update:model-value="emit('update:platform', $event)"
          />
        </div>
        <div class="w-full min-w-[180px] sm:w-48">
          <label class="input-label">{{ t('usage.operation') }}</label>
          <Select
            :model-value="operation"
            :options="operationOptions"
            class="w-full"
            @update:model-value="emit('update:operation', $event)"
          />
        </div>
        <div class="w-full min-w-[160px] sm:w-44">
          <label class="input-label">{{ t('usage.result') }}</label>
          <Select
            :model-value="status"
            :options="statusOptions"
            class="w-full"
            @update:model-value="emit('update:status', $event)"
          />
        </div>
        <div class="w-full min-w-[260px] sm:w-auto">
          <label class="input-label">{{ t('usage.timeRange') }}</label>
          <DateRangePicker
            :start-date="startDate"
            :end-date="endDate"
            @update:start-date="emit('update:startDate', $event)"
            @update:end-date="emit('update:endDate', $event)"
            @change="emit('date-change')"
          />
        </div>
        <div class="ml-auto flex w-full flex-wrap items-center gap-3 sm:w-auto" data-testid="usage-filter-actions">
          <button
            type="button"
            class="btn btn-secondary inline-flex h-10 flex-1 items-center justify-center gap-2 sm:flex-none"
            data-testid="usage-refresh"
            :disabled="loading"
            @click="emit('refresh')"
          >
            <Icon name="refresh" size="sm" />
            {{ t('common.refresh') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary h-10 flex-1 justify-center sm:flex-none"
            data-testid="usage-clear-filters"
            :disabled="!hasActiveFilters || loading"
            @click="emit('clear')"
          >
            {{ t('usage.filters.clear') }}
          </button>
          <button
            type="button"
            class="btn btn-primary inline-flex h-10 flex-1 items-center justify-center gap-2 sm:flex-none"
            data-testid="usage-export-csv"
            :disabled="loading || exporting"
            @click="emit('export-csv')"
          >
            <Icon name="download" size="sm" />
            {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import type { SelectOption } from '@/types'

defineProps<{
  platform: string
  operation: string
  status: string
  startDate: string
  endDate: string
  platformOptions: SelectOption[]
  operationOptions: SelectOption[]
  statusOptions: SelectOption[]
  hasActiveFilters: boolean
  loading: boolean
  exporting: boolean
}>()

const emit = defineEmits<{
  'update:platform': [value: string | number | boolean | null]
  'update:operation': [value: string | number | boolean | null]
  'update:status': [value: string | number | boolean | null]
  'update:startDate': [value: string]
  'update:endDate': [value: string]
  'date-change': []
  refresh: []
  clear: []
  'export-csv': []
}>()

const { t } = useI18n()
</script>
