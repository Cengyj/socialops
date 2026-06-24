<template>
  <AppLayout>
    <div class="space-y-6">
      <UsageStatsCards :items="statCards" />

      <UsageFiltersToolbar
        :platform="platformFilter"
        :operation="operationFilter"
        :status="statusFilter"
        :start-date="startDate"
        :end-date="endDate"
        :platform-options="platformFilterOptions"
        :operation-options="operationFilterOptions"
        :status-options="statusFilterOptions"
        :has-active-filters="hasActiveFilters"
        :loading="loading"
        :exporting="exporting"
        @update:platform="updatePlatformFilter"
        @update:operation="updateOperationFilter"
        @update:status="updateStatusFilter"
        @update:start-date="startDate = $event"
        @update:end-date="endDate = $event"
        @date-change="handleDateRangeChange"
        @refresh="loadData"
        @clear="clearFilters"
        @export-csv="exportCsv"
      />

      <UsageRecordsTable
        :rows="rows"
        :loading="loading"
        :load-error="loadError"
        :has-active-filters="hasActiveFilters"
        :total-rows="totalRows"
        :page="page"
        :page-size="pageSize"
        :sort-by="sortBy"
        :sort-order="sortOrder"
        :detail-loading="detailLoading"
        :active-detail-id="activeDetailId"
        @retry="loadData"
        @clear="clearFilters"
        @open-detail="openDetail"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        @sort-change="handleSortChange"
      />

      <UsageDetailDialog
        :show="detailDialogOpen"
        :loading="detailLoading"
        :detail="activeDetail"
        :overview-rows="detailOverviewRows"
        :result-rows="detailResultRows"
        :proxy-rows="detailProxyRows"
        :payload-rows="detailPayloadRows"
        :payload-profile-rows="detailPayloadProfileRows"
        :payload-media-cards="detailPayloadMediaCards"
        :template-summary-rows="detailTemplateSummaryRows"
        :template-pool-cards="detailTemplatePoolCards"
        :template-profile-rows="detailTemplateProfileRows"
        :template-media-cards="detailTemplateMediaCards"
        :technical-rows="detailTechnicalRows"
        @close="closeDetailDialog"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import UsageDetailDialog from '@/components/usage/UsageDetailDialog.vue'
import UsageFiltersToolbar from '@/components/usage/UsageFiltersToolbar.vue'
import UsageRecordsTable from '@/components/usage/UsageRecordsTable.vue'
import UsageStatsCards from '@/components/usage/UsageStatsCards.vue'
import { usageAPI } from '@/api/usage'
import type {
  UsageLog,
  UsageStats,
} from '@/api/usage'
import { EXECUTABLE_SOCIAL_TASK_ACTIONS } from '@/types/socialTask'
import type { ExecutableSocialTaskAction } from '@/types/socialTask'
import { useAppStore } from '@/stores/app'
import type { SelectOption } from '@/types'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'
import { buildUsageDetailViewModel } from '@/utils/usageDetailViewModel'
import {
  buildUsageExportQueryParams,
  buildUsageCsv,
  buildUsageListQueryParams,
  buildUsageStatsQueryParams,
  collectDetailMediaPreviewLocators,
  defaultUsageEndDate,
  defaultUsageStartDate,
  formatCurrency,
  formatNumber,
  formatPercentage,
  isExecutableUsageOperation,
  isFinalUsageStatus,
  normalizeUsageFilterValue,
  normalizeUsageOperationFilterValue,
  normalizeUsageOptionValue,
  normalizeUsageSelectValue,
  normalizeUsageStatusFilterValue,
} from '@/utils/usageRecords'
import type { UsageFilterState } from '@/utils/usageRecords'
import { formatWorkbenchTaskSummaryMeta } from '@/utils/workbenchTaskSummary'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const loadError = ref(false)
const exporting = ref(false)
const rows = ref<UsageLog[]>([])
const usageStats = ref<UsageStats | null>(null)
type UsageOperationFilterValue = ExecutableSocialTaskAction | 'all' | ''
const operationFilter = ref<UsageOperationFilterValue>('all')
const statusFilter = ref('all')
const platformFilter = ref('all')
const startDate = ref(defaultUsageStartDate())
const endDate = ref(defaultUsageEndDate())
const page = ref(1)
const pageSize = ref(20)
const sortBy = ref<'platform' | 'operation' | 'account' | 'status' | 'cost' | 'time'>('time')
const sortOrder = ref<'asc' | 'desc'>('desc')
const totalRows = ref(0)
const detailDialogOpen = ref(false)
const detailLoading = ref(false)
const activeDetailId = ref<number | null>(null)
const activeDetail = ref<UsageLog | null>(null)
const detailMediaPreviewURLs = ref<Record<string, string>>({})
let detailMediaPreviewToken = 0
let usageLoadToken = 0

type UsageStatCard = {
  label: string
  value: string
  meta: string
  icon: 'chart' | 'checkCircle' | 'xCircle' | 'trendingUp' | 'dollar'
  iconWrapClass: string
  iconClass: string
  valueClass: string
  cardClass?: string
}

const hasActiveFilters = computed(() => (
  normalizeUsageFilterValue(operationFilter.value) !== '' ||
  normalizeUsageStatusFilterValue(statusFilter.value) !== '' ||
  normalizeUsageFilterValue(platformFilter.value) !== '' ||
  startDate.value !== defaultUsageStartDate() ||
  endDate.value !== defaultUsageEndDate()
))

const operationFilterOptions = computed<SelectOption[]>(() => {
  const baseOperations = [...EXECUTABLE_SOCIAL_TASK_ACTIONS]
  const rowOperations = rows.value
    .map(row => normalizeUsageOptionValue(row.operation))
    .filter(isExecutableUsageOperation)
    .filter(Boolean)
  const values = Array.from(new Set([...baseOperations, ...rowOperations]))
  return [
    { value: 'all', label: t('usage.filters.allOperations') },
    ...values.map(value => ({ value, label: actionLabel(value) })),
  ]
})

const statusFilterOptions = computed<SelectOption[]>(() => {
  const baseStatuses = ['success', 'failed']
  const rowStatuses = rows.value
    .map(row => normalizeUsageOptionValue(row.status))
    .filter(value => value === 'success' || value === 'failed')
    .filter(Boolean)
  const values = Array.from(new Set([...baseStatuses, ...rowStatuses]))
  return [
    { value: 'all', label: t('usage.filters.allStatuses') },
    ...values.map(value => ({ value, label: statusLabel(value) })),
  ]
})

const platformFilterOptions = computed<SelectOption[]>(() => {
  const basePlatforms = ['x_twitter']
  const rowPlatforms = rows.value
    .map(row => normalizeUsageOptionValue(row.platform))
    .filter(Boolean)
  const values = Array.from(new Set([...basePlatforms, ...rowPlatforms]))
  return [
    { value: 'all', label: t('usage.filters.allPlatforms') },
    ...values.map(value => ({ value, label: platformLabel(value) })),
  ]
})

const statCards = computed<UsageStatCard[]>(() => [
  {
    label: t('usage.totalOperations'),
    value: formatNumber(usageStats.value?.total_operations ?? rows.value.length),
    meta: t('usage.inSelectedRange'),
    icon: 'chart',
    iconWrapClass: 'bg-blue-100 dark:bg-blue-900/30',
    iconClass: 'text-blue-600 dark:text-blue-400',
    valueClass: 'text-gray-900 dark:text-white',
  },
  {
    label: t('usage.successCount'),
    value: formatNumber(usageStats.value?.success_count ?? rows.value.filter(row => row.status === 'success').length),
    meta: t('usage.inSelectedRange'),
    icon: 'checkCircle',
    iconWrapClass: 'bg-emerald-100 dark:bg-emerald-900/30',
    iconClass: 'text-emerald-600 dark:text-emerald-400',
    valueClass: 'text-emerald-600 dark:text-emerald-400',
  },
  {
    label: t('usage.failedCount'),
    value: formatNumber(usageStats.value?.failed_count ?? rows.value.filter(row => row.status === 'failed').length),
    meta: t('usage.inSelectedRange'),
    icon: 'xCircle',
    iconWrapClass: 'bg-rose-100 dark:bg-rose-900/30',
    iconClass: 'text-rose-600 dark:text-rose-400',
    valueClass: 'text-rose-600 dark:text-rose-400',
  },
  {
    label: t('usage.successRate'),
    value: formatPercentage(usageStats.value?.success_count ?? rows.value.filter(row => row.status === 'success').length, usageStats.value?.total_operations ?? rows.value.length),
    meta: t('usage.inSelectedRange'),
    icon: 'trendingUp',
    iconWrapClass: 'bg-violet-100 dark:bg-violet-900/30',
    iconClass: 'text-violet-600 dark:text-violet-400',
    valueClass: 'text-gray-900 dark:text-white',
  },
  {
    label: t('usage.totalCharged'),
    value: formatCurrency(usageStats.value?.total_charged ?? rows.value.reduce((sum, row) => sum + (row.cost || 0), 0)),
    meta: t('usage.successOnlyBilling'),
    icon: 'dollar',
    iconWrapClass: 'bg-green-100 dark:bg-green-900/30',
    iconClass: 'text-green-600 dark:text-green-400',
    valueClass: 'text-green-600 dark:text-green-400',
    cardClass: 'border-green-100 bg-green-50/40 dark:border-green-900/30 dark:bg-green-900/10',
  },
])

const detailViewModel = computed(() => buildUsageDetailViewModel(activeDetail.value, detailMediaPreviewURLs.value, {
  t: translateUsageDetail,
  actionLabel,
  platformLabel,
  statusLabel,
  chargeStatusLabel,
  chargeSourceLabel,
  proxyStatusLabel,
  resultMessage,
}))

const detailOverviewRows = computed(() => detailViewModel.value.overviewRows)
const detailResultRows = computed(() => detailViewModel.value.resultRows)
const detailProxyRows = computed(() => detailViewModel.value.proxyRows)
const detailPayloadRows = computed(() => detailViewModel.value.payloadRows)
const detailPayloadProfileRows = computed(() => detailViewModel.value.payloadProfileRows)
const detailPayloadMediaCards = computed(() => detailViewModel.value.payloadMediaCards)
const detailTemplateSummaryRows = computed(() => detailViewModel.value.templateSummaryRows)
const detailTemplatePoolCards = computed(() => detailViewModel.value.templatePoolCards)
const detailTemplateProfileRows = computed(() => detailViewModel.value.templateProfileRows)
const detailTemplateMediaCards = computed(() => detailViewModel.value.templateMediaCards)
const detailTechnicalRows = computed(() => detailViewModel.value.technicalRows)

async function loadData() {
  const loadToken = ++usageLoadToken
  loading.value = true
  try {
    const listParams = buildUsageListParams()
    const statsParams = buildUsageStatsParams()
    const [listResult, statsResult] = await Promise.allSettled([
      usageAPI.list(listParams),
      usageAPI.getStats(statsParams),
    ])
    if (loadToken !== usageLoadToken) return
    if (listResult.status === 'rejected') {
      throw listResult.reason
    }
    const listItems = listResult.value.items ?? []
    const finalItems = listItems.filter(row => isFinalUsageStatus(row.status))
    rows.value = finalItems
    totalRows.value = listResult.value.total ?? finalItems.length
    page.value = listResult.value.page ?? page.value
    pageSize.value = listResult.value.page_size ?? pageSize.value
    usageStats.value = statsResult.status === 'fulfilled' ? statsResult.value : null
    loadError.value = false
  } catch (error) {
    if (loadToken !== usageLoadToken) return
    loadError.value = true
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    if (loadToken === usageLoadToken) {
      loading.value = false
    }
  }
}

function updateOperationFilter(value: string | number | boolean | null) {
  const normalized = normalizeUsageSelectValue(value)
  operationFilter.value = normalized === 'all' ? 'all' : normalizeUsageOperationFilterValue(normalized)
  page.value = 1
  void loadData()
}

function updateStatusFilter(value: string | number | boolean | null) {
  statusFilter.value = normalizeUsageStatusFilterValue(normalizeUsageSelectValue(value)) || 'all'
  page.value = 1
  void loadData()
}

function updatePlatformFilter(value: string | number | boolean | null) {
  platformFilter.value = normalizeUsageSelectValue(value)
  page.value = 1
  void loadData()
}

function clearFilters() {
  if (!hasActiveFilters.value) return
  operationFilter.value = 'all'
  statusFilter.value = 'all'
  platformFilter.value = 'all'
  startDate.value = defaultUsageStartDate()
  endDate.value = defaultUsageEndDate()
  page.value = 1
  void loadData()
}

function handleDateRangeChange() {
  page.value = 1
  void loadData()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  void loadData()
}

function handlePageSizeChange(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  void loadData()
}

function handleSortChange(nextSortBy: 'platform' | 'operation' | 'account' | 'status' | 'cost' | 'time', nextSortOrder: 'asc' | 'desc') {
  sortBy.value = nextSortBy
  sortOrder.value = nextSortOrder
  page.value = 1
  void loadData()
}

async function exportCsv() {
  if (exporting.value) return
  exporting.value = true
  try {
    const items = await loadUsageExportRows()
    if (items.length === 0) {
      appStore.showError(t('usage.exportEmpty'))
      return
    }
    downloadCsv(items)
  } catch {
    appStore.showError(t('usage.exportFailed'))
  } finally {
    exporting.value = false
  }
}

async function openDetail(id: number) {
  detailDialogOpen.value = true
  detailLoading.value = true
  activeDetailId.value = id
  activeDetail.value = null
  detailMediaPreviewToken += 1
  clearDetailMediaPreviewURLs()
  const previewToken = detailMediaPreviewToken
  try {
    const detail = await usageAPI.getById(id)
    activeDetail.value = detail
    void loadDetailMediaPreviews(id, detail, previewToken)
  } catch (error) {
    appStore.showError(t('usage.detailLoadFailed'))
    detailDialogOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function closeDetailDialog() {
  detailMediaPreviewToken += 1
  clearDetailMediaPreviewURLs()
  detailDialogOpen.value = false
  detailLoading.value = false
  activeDetailId.value = null
  activeDetail.value = null
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

function chargeStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  const key = `usage.chargeStatuses.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function chargeSourceLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return t('common.none')
  const key = `usage.chargeSources.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized
}

function proxyStatusLabel(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return ''
  const key = `usage.proxyStatuses.${normalized}`
  const translated = t(key)
  if (translated !== key) return translated
  return normalized
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

function translateUsageDetail(key: string, params?: Record<string, string | number>) {
  return params ? t(key, params) : t(key)
}

function currentUsageFilterState(): UsageFilterState {
  return {
    startDate: startDate.value,
    endDate: endDate.value,
    operation: operationFilter.value,
    platform: platformFilter.value,
    status: statusFilter.value,
  }
}

function buildUsageListParams() {
  return buildUsageListQueryParams(currentUsageFilterState(), {
    page: page.value,
    pageSize: pageSize.value,
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
  })
}

function buildUsageExportParams(exportPage: number) {
  return buildUsageExportQueryParams(currentUsageFilterState(), {
    page: exportPage,
    pageSize: usageExportPageSize,
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
  })
}

function buildUsageStatsParams() {
  return buildUsageStatsQueryParams(currentUsageFilterState())
}

const usageExportPageSize = 100
const usageExportMaxRows = 10000

async function loadUsageExportRows() {
  const collected: UsageLog[] = []
  let exportPage = 1
  let totalPages = 1

  do {
    const result = await usageAPI.list(buildUsageExportParams(exportPage))
    const items = result.items ?? []
    collected.push(...items)
    totalPages = Math.max(1, result.pages || Math.ceil((result.total || 0) / (result.page_size || usageExportPageSize)) || 1)
    if (items.length === 0 || collected.length >= usageExportMaxRows) break
    exportPage += 1
  } while (exportPage <= totalPages)

  return collected.slice(0, usageExportMaxRows)
}

function downloadCsv(items: UsageLog[]) {
  const csv = buildUsageCsv(items, {
    platform: t('usage.platform'),
    operation: t('usage.operation'),
    account: t('usage.account'),
    result: t('usage.result'),
    cost: t('usage.cost'),
    summary: t('usage.summary'),
    time: t('usage.time'),
    target: t('usage.detailLabels.target'),
    content: t('usage.detailLabels.content'),
  }, {
    actionLabel,
    platformLabel,
    statusLabel,
    resultSummary,
    resultMessage,
  })
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' })
  const url = createObjectURLSafe(blob)
  if (!url) {
    throw new Error('Object URL unavailable')
  }
  const link = document.createElement('a')
  try {
    link.href = url
    link.download = `socialops-usage-${startDate.value || 'all'}-${endDate.value || 'all'}.csv`
    document.body.appendChild(link)
    link.click()
  } finally {
    if (link.parentNode) {
      link.parentNode.removeChild(link)
    }
    revokeObjectURLSafe(url)
  }
}

async function loadDetailMediaPreviews(id: number, detail: UsageLog, previewToken: number) {
  const requests = collectDetailMediaPreviewLocators(detail)
  await Promise.all(requests.map(async ({ key, locator }) => {
    try {
      const blob = await usageAPI.previewTaskMedia(id, locator)
      const objectURL = createObjectURLSafe(blob)
      if (!objectURL) return
      if (previewToken !== detailMediaPreviewToken) {
        revokeObjectURLSafe(objectURL)
        return
      }
      revokeObjectURLSafe(detailMediaPreviewURLs.value[key] || '')
      detailMediaPreviewURLs.value = {
        ...detailMediaPreviewURLs.value,
        [key]: objectURL,
      }
    } catch {
      // Missing or unsupported task-history media stays non-previewable.
    }
  }))
}

function createObjectURLSafe(blob: Blob) {
  const fn = globalThis.URL && typeof globalThis.URL.createObjectURL === 'function'
    ? globalThis.URL.createObjectURL.bind(globalThis.URL)
    : null
  return fn ? fn(blob) : ''
}

function revokeObjectURLSafe(url: string) {
  if (!url) return
  const fn = globalThis.URL && typeof globalThis.URL.revokeObjectURL === 'function'
    ? globalThis.URL.revokeObjectURL.bind(globalThis.URL)
    : null
  if (fn) fn(url)
}

function clearDetailMediaPreviewURLs() {
  Object.values(detailMediaPreviewURLs.value).forEach(url => revokeObjectURLSafe(url))
  detailMediaPreviewURLs.value = {}
}

onMounted(loadData)
onBeforeUnmount(() => {
  detailMediaPreviewToken += 1
  clearDetailMediaPreviewURLs()
})
</script>
