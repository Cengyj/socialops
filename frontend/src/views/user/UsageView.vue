<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('usage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('usage.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="loadData">
          {{ t('common.refresh') }}
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div v-for="item in statCards" :key="item.label" class="card p-5">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap">
            <Select
              :model-value="operationFilter"
              :options="operationFilterOptions"
              class="w-full sm:w-48"
              @update:model-value="updateOperationFilter"
            />
            <Select
              :model-value="statusFilter"
              :options="statusFilterOptions"
              class="w-full sm:w-44"
              @update:model-value="updateStatusFilter"
            />
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm h-10 justify-center"
            data-testid="usage-clear-filters"
            :disabled="!hasActiveFilters || loading"
            @click="clearFilters"
          >
            {{ t('usage.filters.clear') }}
          </button>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('usage.records') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.operation') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.platform') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.account') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.status') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.quantity') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.chargeStatus') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.cost') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.result') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="row in rows" :key="row.id">
                <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ actionLabel(row.operation) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ platformLabel(row.platform) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ row.account_name || '-' }}</td>
                <td class="px-5 py-3 text-sm">
                  <span :class="['badge', statusClass(row.status)]">{{ statusLabel(row.status) }}</span>
                </td>
                <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatNumber(row.quantity) }}</td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ chargeStatusLabel(row.charge_status) }}</td>
                <td class="px-5 py-3 text-right text-sm text-gray-600 dark:text-gray-300">{{ formatCurrency(row.cost) }}</td>
                <td class="max-w-xs px-5 py-3 text-sm text-gray-600 dark:text-gray-300">
                  <div v-if="resultSummary(row)" class="line-clamp-2 font-medium text-gray-900 dark:text-white">{{ resultSummary(row) }}</div>
                  <div class="line-clamp-2" :class="resultSummary(row) ? 'mt-1' : ''">{{ resultMessage(row) }}</div>
                </td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.completed_at || row.created_at) }}</td>
                <td class="px-5 py-3 text-right text-sm">
                  <button
                    type="button"
                    class="btn btn-secondary px-2 py-1 text-xs"
                    :data-testid="`usage-detail-button-${row.id}`"
                    :disabled="detailLoading && activeDetailId === row.id"
                    @click="openDetail(row.id)"
                  >
                    {{ t('usage.actions.viewDetails') }}
                  </button>
                </td>
              </tr>
              <tr v-if="!loading && rows.length === 0">
                <td class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="10">
                  {{ t('usage.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <BaseDialog :show="detailDialogOpen" :title="t('usage.detailTitle')" width="wide" @close="closeDetailDialog">
        <div class="space-y-5">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('usage.detailDescription') }}</p>
          </div>

          <div v-if="detailLoading" class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
            {{ t('usage.detailLoading') }}
          </div>

          <div v-else-if="activeDetail" class="space-y-5">
            <section class="space-y-3">
              <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.summary') }}</h3>
              <div class="grid gap-3 md:grid-cols-2">
                <div v-for="item in detailSummaryRows" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-white">{{ item.value }}</div>
                </div>
              </div>
            </section>

            <section v-if="detailPayloadRows.length > 0" class="space-y-3">
              <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.payload') }}</h3>
              <div class="grid gap-3 md:grid-cols-2">
                <div v-for="item in detailPayloadRows" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-white">{{ item.value }}</div>
                </div>
              </div>
            </section>

            <section v-if="detailPayloadProfileRows.length > 0" class="space-y-3">
              <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.profile') }}</h3>
              <div class="grid gap-3 md:grid-cols-2">
                <div v-for="item in detailPayloadProfileRows" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-white">{{ item.value }}</div>
                </div>
              </div>
            </section>

            <section v-if="detailPayloadMediaCards.length > 0" class="space-y-3">
              <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.media') }}</h3>
              <div class="grid gap-3 md:grid-cols-2">
                <div v-for="card in detailPayloadMediaCards" :key="card.title" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ card.title }}</div>
                  <div v-if="card.previewSrc" class="mt-3 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-700">
                    <img
                      :src="card.previewSrc"
                      :alt="card.title"
                      :data-testid="card.previewTestId"
                      class="h-40 w-full object-cover"
                    >
                  </div>
                  <div class="mt-2 space-y-2">
                    <div v-for="item in card.rows" :key="item.label" class="text-sm text-gray-900 dark:text-white">
                      <span class="font-medium text-gray-600 dark:text-gray-300">{{ item.label }}:</span>
                      <span class="ml-2 break-words">{{ item.value }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section v-if="detailTemplateSummaryRows.length > 0 || detailTemplatePoolCards.length > 0 || detailTemplateProfileRows.length > 0 || detailTemplateMediaCards.length > 0" class="space-y-3">
              <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.template') }}</h3>
              <div v-if="detailTemplateSummaryRows.length > 0" class="grid gap-3 md:grid-cols-2">
                <div v-for="item in detailTemplateSummaryRows" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-white">{{ item.value }}</div>
                </div>
              </div>
              <div v-if="detailTemplatePoolCards.length > 0" class="grid gap-3 md:grid-cols-2">
                <div v-for="card in detailTemplatePoolCards" :key="card.title" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ card.title }}</div>
                  <div class="mt-2 space-y-2">
                    <div v-for="value in card.values" :key="value" class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-900 dark:bg-dark-700 dark:text-white">
                      {{ value }}
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="detailTemplateProfileRows.length > 0" class="grid gap-3 md:grid-cols-2">
                <div v-for="item in detailTemplateProfileRows" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-white">{{ item.value }}</div>
                </div>
              </div>
              <div v-if="detailTemplateMediaCards.length > 0" class="grid gap-3 md:grid-cols-2">
                <div v-for="card in detailTemplateMediaCards" :key="card.title" class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
                  <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ card.title }}</div>
                  <div v-if="card.previewSrc" class="mt-3 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-700">
                    <img
                      :src="card.previewSrc"
                      :alt="card.title"
                      :data-testid="card.previewTestId"
                      class="h-40 w-full object-cover"
                    >
                  </div>
                  <div class="mt-2 space-y-2">
                    <div v-for="item in card.rows" :key="item.label" class="text-sm text-gray-900 dark:text-white">
                      <span class="font-medium text-gray-600 dark:text-gray-300">{{ item.label }}:</span>
                      <span class="ml-2 break-words">{{ item.value }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </section>
          </div>

          <div v-else class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
            {{ t('usage.detailEmpty') }}
          </div>
        </div>

        <template #footer>
          <button type="button" class="btn btn-primary" @click="closeDetailDialog">{{ t('common.close') }}</button>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { usageAPI } from '@/api/usage'
import type {
  SocialProfileUpdateParams,
  SocialTaskMediaRef,
  SocialTaskTemplateSnapshot,
  UsageTaskMediaPreviewLocator,
  UsageLog,
  UsageQueryParams,
  UsageStats,
} from '@/api/usage'
import { useAppStore } from '@/stores/app'
import type { SelectOption } from '@/types'
import { formatSocialTaskResultMessage } from '@/utils/socialTaskResultMessage'
import { formatWorkbenchTaskSummaryMeta } from '@/utils/workbenchTaskSummary'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const rows = ref<UsageLog[]>([])
const usageStats = ref<UsageStats | null>(null)
const successStats = ref<UsageStats | null>(null)
const operationFilter = ref('all')
const statusFilter = ref('all')
const detailDialogOpen = ref(false)
const detailLoading = ref(false)
const activeDetailId = ref<number | null>(null)
const activeDetail = ref<UsageLog | null>(null)
const detailMediaPreviewURLs = ref<Record<string, string>>({})
let detailMediaPreviewToken = 0
let usageLoadToken = 0

const hasActiveFilters = computed(() => normalizeUsageFilterValue(operationFilter.value) !== '' || normalizeUsageFilterValue(statusFilter.value) !== '')

const operationFilterOptions = computed<SelectOption[]>(() => {
  const baseOperations = [
    'login_check',
    'follow',
    'like',
    'retweet',
    'reply',
    'quote',
    'post',
    'update_profile',
    'update_avatar',
    'update_banner',
  ]
  const rowOperations = rows.value
    .map(row => normalizeUsageOptionValue(row.operation))
    .filter(Boolean)
  const values = Array.from(new Set([...baseOperations, ...rowOperations]))
  return [
    { value: 'all', label: t('usage.filters.allOperations') },
    ...values.map(value => ({ value, label: actionLabel(value) })),
  ]
})

const statusFilterOptions = computed<SelectOption[]>(() => {
  const baseStatuses = ['success', 'failed', 'pending', 'running', 'queued']
  const rowStatuses = rows.value
    .map(row => normalizeUsageOptionValue(row.status))
    .filter(Boolean)
  const values = Array.from(new Set([...baseStatuses, ...rowStatuses]))
  return [
    { value: 'all', label: t('usage.filters.allStatuses') },
    ...values.map(value => ({ value, label: statusLabel(value) })),
  ]
})

const statCards = computed(() => [
  { label: t('usage.totalOperations'), value: formatNumber(usageStats.value?.total_requests ?? rows.value.length) },
  { label: t('usage.totalQuantity'), value: formatNumber(usageStats.value?.total_tokens ?? usageStats.value?.total_quantity ?? rows.value.reduce((sum, row) => sum + (row.quantity || 0), 0)) },
  { label: t('usage.successCount'), value: formatNumber(successStats.value?.total_requests ?? rows.value.filter(row => row.status === 'success').length) },
  { label: t('usage.totalCost'), value: formatCurrency(usageStats.value?.total_actual_cost ?? usageStats.value?.total_cost ?? rows.value.reduce((sum, row) => sum + (row.cost || 0), 0)) },
])

const detailSummaryRows = computed(() => {
  const detail = activeDetail.value
  if (!detail) return []
  return [
    ...buildDetailRows([
    [t('usage.detailLabels.operation'), actionLabel(detail.operation)],
    [t('usage.detailLabels.platform'), platformLabel(detail.platform)],
    [t('usage.detailLabels.account'), detail.account_name],
    [t('usage.detailLabels.status'), statusLabel(detail.status)],
    [t('usage.detailLabels.chargeStatus'), chargeStatusLabel(detail.charge_status)],
    [t('usage.detailLabels.chargeSource'), chargeSourceLabel(detail.charge_source)],
    [t('usage.detailLabels.cost'), formatCurrency(detail.cost)],
    [t('usage.detailLabels.quantity'), formatNumber(detail.quantity)],
    [t('usage.detailLabels.result'), resultMessage(detail)],
    [t('usage.detailLabels.createdAt'), formatDate(detail.created_at)],
    [t('usage.detailLabels.completedAt'), formatDate(detail.completed_at || detail.created_at)],
    [t('usage.detailLabels.billingRequestId'), detail.billing_request_id],
    [t('usage.detailLabels.idempotencyKey'), detail.idempotency_key],
    [t('usage.detailLabels.target'), firstNonEmpty(detail.target, detail.payload?.target)],
    [t('usage.detailLabels.content'), firstNonEmpty(detail.content, detail.payload?.post?.text)],
    [t('usage.detailLabels.quotePostUrl'), firstNonEmpty(detail.payload?.post?.quote_post_url, detail.template_snapshot?.params?.quote_post_url)],
    ]),
    ...buildProxySnapshotRows(detail.proxy_snapshot),
  ]
})

const detailPayloadRows = computed(() => {
  const payload = activeDetail.value?.payload
  if (!payload) return []
  return buildDetailRows([
    [t('usage.detailLabels.target'), payload.target],
    [t('usage.detailLabels.content'), payload.post?.text],
    [t('usage.detailLabels.quotePostUrl'), payload.post?.quote_post_url],
  ])
})

const detailPayloadProfileRows = computed(() => buildProfileRows(activeDetail.value?.payload?.profile))
const detailPayloadMediaCards = computed(() => buildPayloadMediaCards(activeDetail.value, detailMediaPreviewURLs.value))

const detailTemplateSummaryRows = computed(() => buildTemplateSummaryRows(activeDetail.value?.template_snapshot))
const detailTemplatePoolCards = computed(() => buildTemplatePoolCards(activeDetail.value?.template_snapshot))
const detailTemplateProfileRows = computed(() => buildProfileRows(activeDetail.value?.template_snapshot?.params?.profile))
const detailTemplateMediaCards = computed(() => buildTemplateMediaCards(activeDetail.value?.template_snapshot, detailMediaPreviewURLs.value))

async function loadData() {
  const loadToken = ++usageLoadToken
  loading.value = true
  try {
    const listParams = buildUsageListParams()
    const statsParams = buildFilteredUsageStatsParams()
    const successStatsParams = buildSuccessUsageStatsParams()
    const [listResult, statsResult, successStatsResult] = await Promise.allSettled([
      usageAPI.list(listParams),
      statsParams ? usageAPI.getStats(statsParams) : usageAPI.getStats(),
      usageAPI.getStats(successStatsParams),
    ])
    if (loadToken !== usageLoadToken) return
    if (listResult.status === 'rejected') {
      throw listResult.reason
    }
    rows.value = listResult.value.items ?? []
    usageStats.value = statsResult.status === 'fulfilled' ? statsResult.value : null
    successStats.value = successStatsResult.status === 'fulfilled' ? successStatsResult.value : null
  } catch (error) {
    if (loadToken !== usageLoadToken) return
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    if (loadToken === usageLoadToken) {
      loading.value = false
    }
  }
}

function updateOperationFilter(value: string | number | boolean | null) {
  operationFilter.value = normalizeUsageSelectValue(value)
  void loadData()
}

function updateStatusFilter(value: string | number | boolean | null) {
  statusFilter.value = normalizeUsageSelectValue(value)
  void loadData()
}

function clearFilters() {
  if (!hasActiveFilters.value) return
  operationFilter.value = 'all'
  statusFilter.value = 'all'
  void loadData()
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

function statusClass(status: string) {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-error'
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

function formatCurrency(value?: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function formatNumber(value?: number) {
  return Number(value || 0).toLocaleString()
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function firstNonEmpty(...values: Array<string | null | undefined>) {
  for (const value of values) {
    const normalized = String(value || '').trim()
    if (normalized) return normalized
  }
  return ''
}

function buildDetailRows(entries: Array<[string, string | null | undefined]>) {
  return entries
    .map(([label, value]) => ({ label, value: String(value || '').trim() }))
    .filter(item => item.value)
}

function buildProxySnapshotRows(value?: string | null) {
  const raw = String(value || '').trim()
  if (!raw) return []

  const parsed = parseProxySnapshotValue(raw)
  if (!parsed) {
    return buildDetailRows([[t('usage.detailLabels.proxySnapshot'), raw]])
  }

  if (parsed.kind === 'endpoint') {
    return buildDetailRows([[t('usage.detailLabels.proxyEndpoint'), parsed.endpoint]])
  }

  return buildDetailRows([
    [t('usage.detailLabels.proxyName'), parsed.name],
    [t('usage.detailLabels.proxyEndpoint'), parsed.endpoint],
    [t('usage.detailLabels.proxyStatus'), proxyStatusLabel(parsed.status)],
  ])
}

function parseProxySnapshotValue(raw: string): { kind: 'endpoint'; endpoint: string } | { kind: 'structured'; name: string; endpoint: string; status: string } | null {
  try {
    const parsed = JSON.parse(raw) as unknown
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const payload = parsed as Record<string, unknown>
      const name = normalizeSnapshotText(payload.name)
      const endpoint = normalizeSnapshotText(payload.endpoint)
      const status = normalizeSnapshotText(payload.status)
      if (name || endpoint || status) {
        return {
          kind: 'structured',
          name,
          endpoint,
          status,
        }
      }
    }
  } catch {
    // Fallback to legacy plain endpoint handling below.
  }

  if (looksLikeURL(raw)) {
    return { kind: 'endpoint', endpoint: raw }
  }

  return null
}

function normalizeSnapshotText(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function looksLikeURL(value: string) {
  try {
    const parsed = new URL(value)
    return parsed.protocol !== '' && parsed.host !== ''
  } catch {
    return false
  }
}

function normalizeUsageSelectValue(value: string | number | boolean | null) {
  const normalized = String(value ?? '').trim().toLowerCase()
  return normalized || 'all'
}

function normalizeUsageOptionValue(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

function normalizeUsageFilterValue(value?: string | null) {
  const normalized = normalizeUsageOptionValue(value)
  return normalized === '' || normalized === 'all' ? '' : normalized
}

function buildUsageListParams(): UsageQueryParams {
  const params: UsageQueryParams = {
    page: 1,
    page_size: 20,
  }
  const operation = normalizeUsageFilterValue(operationFilter.value)
  const status = normalizeUsageFilterValue(statusFilter.value)
  if (operation) params.operation = operation
  if (status) params.status = status
  return params
}

function buildFilteredUsageStatsParams(): Pick<UsageQueryParams, 'operation' | 'status'> | undefined {
  const params: Pick<UsageQueryParams, 'operation' | 'status'> = {}
  const operation = normalizeUsageFilterValue(operationFilter.value)
  const status = normalizeUsageFilterValue(statusFilter.value)
  if (operation) params.operation = operation
  if (status) params.status = status
  return Object.keys(params).length > 0 ? params : undefined
}

function buildSuccessUsageStatsParams(): Pick<UsageQueryParams, 'operation' | 'status'> {
  const params: Pick<UsageQueryParams, 'operation' | 'status'> = {
    status: 'success',
  }
  const operation = normalizeUsageFilterValue(operationFilter.value)
  if (operation) params.operation = operation
  return params
}

function buildProfileRows(profile?: SocialProfileUpdateParams | null) {
  if (!profile) return []
  return buildDetailRows([
    [t('usage.detailLabels.displayName'), profile.display_name],
    [t('usage.detailLabels.screenName'), profile.screen_name],
    [t('usage.detailLabels.description'), profile.description],
    [t('usage.detailLabels.location'), profile.location],
    [t('usage.detailLabels.url'), profile.url],
  ])
}

function buildPayloadMediaCards(detail?: UsageLog | null, previewURLs: Record<string, string> = {}) {
  if (!detail) return []
  const cards = buildMediaCards(detail.payload?.post?.media, 'payload', 'post', previewURLs)
  const avatarCard = buildNamedMediaCard(t('usage.detailLabels.avatar'), detail.payload?.avatar, mediaPreviewKey('payload', 'avatar'), previewURLs)
  const bannerCard = buildNamedMediaCard(t('usage.detailLabels.banner'), detail.payload?.banner, mediaPreviewKey('payload', 'banner'), previewURLs)
  if (avatarCard) cards.push(avatarCard)
  if (bannerCard) cards.push(bannerCard)
  return cards
}

function buildTemplateSummaryRows(snapshot?: SocialTaskTemplateSnapshot | null) {
  if (!snapshot) return []
  return buildDetailRows([
    [t('usage.detailLabels.templateName'), snapshot.template_name],
    [t('usage.detailLabels.templateType'), actionLabel(snapshot.template_type)],
    [t('usage.detailLabels.quotePostUrl'), snapshot.params?.quote_post_url],
  ])
}

function buildTemplatePoolCards(snapshot?: SocialTaskTemplateSnapshot | null) {
  const params = snapshot?.params
  if (!params) return []
  const cards: Array<{ title: string; values: string[] }> = []
  const targets = normalizeStringList(params.targets)
  const contents = normalizeStringList(params.contents)
  if (targets.length > 0) cards.push({ title: t('usage.detailSections.targets'), values: targets })
  if (contents.length > 0) cards.push({ title: t('usage.detailSections.contents'), values: contents })
  return cards
}

function buildTemplateMediaCards(snapshot?: SocialTaskTemplateSnapshot | null, previewURLs: Record<string, string> = {}) {
  if (!snapshot?.params) return []
  const cards = buildMediaCards(snapshot.params.media, 'template', 'post', previewURLs)
  const avatarCard = buildNamedMediaCard(t('usage.detailLabels.avatar'), snapshot.params.avatar, mediaPreviewKey('template', 'avatar'), previewURLs)
  const bannerCard = buildNamedMediaCard(t('usage.detailLabels.banner'), snapshot.params.banner, mediaPreviewKey('template', 'banner'), previewURLs)
  if (avatarCard) cards.push(avatarCard)
  if (bannerCard) cards.push(bannerCard)
  return cards
}

function buildMediaCards(
  items?: SocialTaskMediaRef[] | null,
  scope: 'payload' | 'template' = 'payload',
  section: 'post' | 'avatar' | 'banner' = 'post',
  previewURLs: Record<string, string> = {},
) {
  return (items ?? [])
    .map((item, index) => buildNamedMediaCard(
      t('usage.detailLabels.mediaItem', { index: index + 1 }),
      item,
      mediaPreviewKey(scope, section, index),
      previewURLs,
    ))
    .filter((item): item is { title: string; rows: Array<{ label: string; value: string }>; previewSrc: string; previewTestId: string } => !!item)
}

function buildNamedMediaCard(title: string, item?: SocialTaskMediaRef | null, previewKey = '', previewURLs: Record<string, string> = {}) {
  if (!item || !hasMediaMetadata(item)) return null
  const rows = buildDetailRows([
    [t('usage.detailLabels.fileName'), item.file_name],
    [t('usage.detailLabels.contentType'), item.content_type],
    [t('usage.detailLabels.dimensions'), formatMediaDimensions(item)],
    [t('usage.detailLabels.byteSize'), formatByteSize(item.byte_size)],
    [t('usage.detailLabels.source'), item.source],
  ])
  if (rows.length === 0) return null
  return {
    title,
    rows,
    previewSrc: previewURLs[previewKey] || '',
    previewTestId: mediaPreviewTestID(previewKey),
  }
}

function hasMediaMetadata(item?: SocialTaskMediaRef | null) {
  if (!item) return false
  return Boolean(
    String(item.source || '').trim() ||
    String(item.file_name || '').trim() ||
    String(item.content_type || '').trim() ||
    Number(item.byte_size || 0) > 0 ||
    Number(item.width || 0) > 0 ||
    Number(item.height || 0) > 0
  )
}

function formatMediaDimensions(item?: SocialTaskMediaRef | null) {
  const width = Number(item?.width || 0)
  const height = Number(item?.height || 0)
  if (width <= 0 || height <= 0) return ''
  return `${formatNumber(width)} × ${formatNumber(height)}`
}

function formatByteSize(value?: number | null) {
  const normalized = Number(value || 0)
  if (normalized <= 0) return ''
  return `${formatNumber(normalized)} B`
}

function normalizeStringList(values?: string[] | null) {
  return (values ?? []).map(value => value.trim()).filter(Boolean)
}

function mediaPreviewKey(scope: 'payload' | 'template', section: 'post' | 'avatar' | 'banner', index?: number) {
  if (typeof index === 'number' && index >= 0) return `${scope}:${section}:${index}`
  return `${scope}:${section}`
}

function mediaPreviewTestID(previewKey: string) {
  return `usage-media-preview-${previewKey.replace(/[^a-z0-9]+/gi, '-')}`
}

function shouldAttemptMediaPreview(item?: SocialTaskMediaRef | null) {
  if (!item || !hasMediaMetadata(item)) return false
  const contentType = String(item.content_type || '').trim().toLowerCase()
  return contentType === '' || contentType.startsWith('image/')
}

function collectDetailMediaPreviewLocators(detail: UsageLog): Array<{ key: string; locator: UsageTaskMediaPreviewLocator }> {
  const items: Array<{ key: string; locator: UsageTaskMediaPreviewLocator }> = []
  detail.payload?.post?.media?.forEach((item, index) => {
    if (!shouldAttemptMediaPreview(item)) return
    items.push({ key: mediaPreviewKey('payload', 'post', index), locator: { scope: 'payload', section: 'post', index } })
  })
  if (shouldAttemptMediaPreview(detail.payload?.avatar)) {
    items.push({ key: mediaPreviewKey('payload', 'avatar'), locator: { scope: 'payload', section: 'avatar' } })
  }
  if (shouldAttemptMediaPreview(detail.payload?.banner)) {
    items.push({ key: mediaPreviewKey('payload', 'banner'), locator: { scope: 'payload', section: 'banner' } })
  }
  detail.template_snapshot?.params?.media?.forEach((item, index) => {
    if (!shouldAttemptMediaPreview(item)) return
    items.push({ key: mediaPreviewKey('template', 'post', index), locator: { scope: 'template', section: 'post', index } })
  })
  if (shouldAttemptMediaPreview(detail.template_snapshot?.params?.avatar)) {
    items.push({ key: mediaPreviewKey('template', 'avatar'), locator: { scope: 'template', section: 'avatar' } })
  }
  if (shouldAttemptMediaPreview(detail.template_snapshot?.params?.banner)) {
    items.push({ key: mediaPreviewKey('template', 'banner'), locator: { scope: 'template', section: 'banner' } })
  }
  return items
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
