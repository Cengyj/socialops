<template>
  <div class="card overflow-hidden">
    <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('usage.records') }}</h2>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full min-w-max divide-y divide-gray-200 dark:divide-dark-700" data-testid="usage-records-table">
        <thead class="bg-gray-50/80 dark:bg-dark-800/80">
          <tr>
            <th class="whitespace-nowrap px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-platform"
                @click="emitSortChange('platform')"
              >
                <span>{{ t('usage.platform') }}</span>
                <Icon :name="sortIcon('platform')" size="xs" :class="sortIconClass('platform')" />
              </button>
            </th>
            <th class="whitespace-nowrap px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-operation"
                @click="emitSortChange('operation')"
              >
                <span>{{ t('usage.operation') }}</span>
                <Icon :name="sortIcon('operation')" size="xs" :class="sortIconClass('operation')" />
              </button>
            </th>
            <th class="whitespace-nowrap px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-account"
                @click="emitSortChange('account')"
              >
                <span>{{ t('usage.account') }}</span>
                <Icon :name="sortIcon('account')" size="xs" :class="sortIconClass('account')" />
              </button>
            </th>
            <th class="whitespace-nowrap px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-status"
                @click="emitSortChange('status')"
              >
                <span>{{ t('usage.result') }}</span>
                <Icon :name="sortIcon('status')" size="xs" :class="sortIconClass('status')" />
              </button>
            </th>
            <th class="whitespace-nowrap px-4 py-3 text-right align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="ml-auto inline-flex items-center justify-end gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-cost"
                @click="emitSortChange('cost')"
              >
                <span>{{ t('usage.cost') }}</span>
                <Icon :name="sortIcon('cost')" size="xs" :class="sortIconClass('cost')" />
              </button>
            </th>
            <th class="px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">{{ t('usage.summary') }}</th>
            <th class="whitespace-nowrap px-4 py-3 text-left align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">
              <button
                type="button"
                class="inline-flex items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-dark-200"
                data-testid="usage-sort-time"
                @click="emitSortChange('time')"
              >
                <span>{{ t('usage.time') }}</span>
                <Icon :name="sortIcon('time')" size="xs" :class="sortIconClass('time')" />
              </button>
            </th>
            <th class="whitespace-nowrap px-4 py-3 text-right align-middle text-xs font-semibold uppercase tracking-wider text-gray-600 dark:text-dark-300 sm:px-6">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
          <tr v-for="row in rows" :key="row.id" class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-800">
            <td class="whitespace-nowrap px-4 py-4 align-middle text-sm text-gray-600 dark:text-gray-300 sm:px-6">{{ platformLabel(row.platform) }}</td>
            <td class="whitespace-nowrap px-4 py-4 align-middle text-sm font-medium text-gray-900 dark:text-white sm:px-6">{{ actionLabel(row.operation) }}</td>
            <td class="whitespace-nowrap px-4 py-4 align-middle text-sm text-gray-900 dark:text-gray-100 sm:px-6">{{ row.account_name || '-' }}</td>
            <td class="whitespace-nowrap px-4 py-4 align-middle text-sm text-gray-900 dark:text-gray-100 sm:px-6">
              <span :class="['badge', statusClass(row.status)]">{{ statusLabel(row.status) }}</span>
            </td>
            <td class="whitespace-nowrap px-4 py-4 text-right align-middle text-sm font-medium tabular-nums text-green-600 dark:text-green-400 sm:px-6">{{ formatCurrency(row.cost) }}</td>
            <td class="max-w-sm px-4 py-4 align-middle text-sm text-gray-900 dark:text-gray-100 sm:px-6">
              <div v-if="resultSummary(row)" class="line-clamp-2 font-medium text-gray-900 dark:text-white">{{ resultSummary(row) }}</div>
              <div class="line-clamp-2 text-gray-500 dark:text-gray-400" :class="resultSummary(row) ? 'mt-1' : ''">{{ resultMessage(row) }}</div>
            </td>
            <td class="whitespace-nowrap px-4 py-4 align-middle text-sm text-gray-600 dark:text-gray-300 sm:px-6">{{ formatDate(row.completed_at || row.created_at) }}</td>
            <td class="whitespace-nowrap px-4 py-4 text-right align-middle text-sm text-gray-900 dark:text-gray-100 sm:px-6">
              <button
                type="button"
                class="btn btn-secondary inline-flex items-center justify-center px-2 py-1 text-xs"
                :data-testid="`usage-detail-button-${row.id}`"
                :disabled="detailLoading && activeDetailId === row.id"
                @click="emit('open-detail', row.id)"
              >
                {{ t('usage.actions.viewDetails') }}
              </button>
            </td>
          </tr>
          <tr v-if="loading && rows.length === 0">
            <td class="px-4 py-12 text-center align-middle text-sm text-gray-500 dark:text-dark-400 sm:px-6" colspan="8">
              <UsageTableState state="loading" />
            </td>
          </tr>
          <tr v-else-if="!loading && loadError && rows.length === 0">
            <td class="px-4 py-12 text-center align-middle text-sm text-gray-500 dark:text-dark-400 sm:px-6" colspan="8">
              <UsageTableState state="error" @retry="emit('retry')" />
            </td>
          </tr>
          <tr v-else-if="!loading && rows.length === 0">
            <td class="px-4 py-12 text-center align-middle text-sm text-gray-500 dark:text-dark-400 sm:px-6" colspan="8">
              <UsageTableState state="empty" :filtered="hasActiveFilters" @clear="emit('clear')" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <Pagination
      v-if="totalRows > 0"
      :total="totalRows"
      :page="page"
      :page-size="pageSize"
      @update:page="emit('update:page', $event)"
      @update:page-size="emit('update:pageSize', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import UsageTableState from '@/components/usage/UsageTableState.vue'
import type { UsageLog } from '@/api/usage'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'
import { formatCurrency, formatDate } from '@/utils/usageRecords'
import { formatWorkbenchTaskSummaryMeta } from '@/utils/workbenchTaskSummary'

type UsageTableSortBy = 'platform' | 'operation' | 'account' | 'status' | 'cost' | 'time'
type UsageTableSortOrder = 'asc' | 'desc'

const props = defineProps<{
  rows: UsageLog[]
  loading: boolean
  loadError: boolean
  hasActiveFilters: boolean
  totalRows: number
  page: number
  pageSize: number
  sortBy: UsageTableSortBy
  sortOrder: UsageTableSortOrder
  detailLoading: boolean
  activeDetailId: number | null
}>()

const emit = defineEmits<{
  retry: []
  clear: []
  'open-detail': [id: number]
  'update:page': [page: number]
  'update:pageSize': [pageSize: number]
  'sort-change': [sortBy: UsageTableSortBy, sortOrder: UsageTableSortOrder]
}>()

const { t } = useI18n()

function statusClass(status: string) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-danger'
  return 'badge-warning'
}

function actionLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return t('common.unknown')
  const key = `usage.actions.${normalized}`
  const translated = t(key)
  return translated === key ? value || t('common.unknown') : translated
}

function statusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.statuses.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function platformLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.platforms.${normalized}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

function resultMessage(row: UsageLog) {
  return formatSocialTaskResultMessage(row, t)
}

function resultSummary(row: UsageLog) {
  const summary = formatWorkbenchTaskSummaryMeta({
    action: row.operation,
    target: row.target,
    content: row.content,
    payload: row.payload,
    template_snapshot: row.template_snapshot,
  }, t, {
    actionKeyPrefix: 'usage.actions',
    summaryKeyPrefix: 'usage',
  })

  return summary.endsWith(`· ${t('usage.taskSummaryNoDetails')}`) ? '' : summary
}

function emitSortChange(nextSortBy: UsageTableSortBy) {
  const nextSortOrder = props.sortBy === nextSortBy && props.sortOrder === 'asc' ? 'desc' : 'asc'
  emit('sort-change', nextSortBy, nextSortOrder)
}

function sortIcon(column: UsageTableSortBy): 'arrowsUpDown' | 'arrowUp' | 'arrowDown' {
  if (props.sortBy !== column) return 'arrowsUpDown'
  return props.sortOrder === 'asc' ? 'arrowUp' : 'arrowDown'
}

function sortIconClass(column: UsageTableSortBy) {
  return props.sortBy === column
    ? 'text-primary-600 dark:text-primary-400'
    : 'text-gray-400 dark:text-dark-500'
}
</script>
