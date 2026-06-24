import type { UsageLog, UsageQueryParams, UsageTaskMediaPreviewLocator, UsageTaskMediaRef } from '@/api/usage'
import { EXECUTABLE_SOCIAL_TASK_ACTIONS } from '@/types/socialTask'
import type { ExecutableSocialTaskAction } from '@/types/socialTask'

export interface UsageCsvLabels {
  platform: string
  operation: string
  account: string
  result: string
  cost: string
  summary: string
  time: string
  target: string
  content: string
}

export interface UsageCsvFormatters {
  actionLabel: (value?: string | null) => string
  platformLabel: (value?: string | null) => string
  statusLabel: (value?: string | null) => string
  resultSummary: (row: UsageLog) => string
  resultMessage: (row: UsageLog) => string
  formatCurrency?: (value?: number) => string
  formatDate?: (value?: string) => string
}

export type UsageStatsQueryParams = Pick<UsageQueryParams, 'start_date' | 'end_date' | 'operation' | 'platform' | 'account' | 'status'>

export interface UsageFilterState {
  startDate?: string | null
  endDate?: string | null
  operation?: ExecutableSocialTaskAction | 'all' | '' | null
  platform?: string | null
  account?: string | null
  status?: string | null
}

export interface UsagePageQueryState {
  page: number
  pageSize: number
  sortBy?: string
  sortOrder?: string
}

export interface ParsedProxySnapshotEndpoint {
  kind: 'endpoint'
  endpoint: string
}

export interface ParsedProxySnapshotStructured {
  kind: 'structured'
  name: string
  endpoint: string
  status: string
}

export type ParsedProxySnapshot = ParsedProxySnapshotEndpoint | ParsedProxySnapshotStructured

export function formatCurrency(value?: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

export function formatNumber(value?: number) {
  return Number(value || 0).toLocaleString()
}

export function formatPercentage(numerator?: number, denominator?: number) {
  const total = Number(denominator || 0)
  if (total <= 0) return '0%'
  const percentage = (Number(numerator || 0) / total) * 100
  return `${percentage.toFixed(1).replace(/\.0$/, '')}%`
}

export function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

export function defaultUsageEndDate(now = new Date()) {
  return formatDateInputValue(now)
}

export function defaultUsageStartDate(now = new Date()) {
  const date = new Date(now)
  date.setDate(date.getDate() - 29)
  return formatDateInputValue(date)
}

export function formatDateInputValue(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function firstNonEmpty(...values: Array<string | null | undefined>) {
  for (const value of values) {
    const normalized = String(value || '').trim()
    if (normalized) return normalized
  }
  return ''
}

export function isFinalUsageStatus(value?: string | null) {
  const normalized = String(value || '').trim().toLowerCase()
  return normalized === 'success' || normalized === 'failed'
}

export function buildDetailRows<TMeta extends Record<string, unknown> = Record<string, never>>(entries: Array<[string, string | null | undefined, TMeta?]>) {
  return entries
    .map(([label, value, meta]) => ({ label, value: String(value || '').trim(), ...(meta ?? {}) }))
    .filter(item => item.value)
}

export function normalizeUsageSelectValue(value: string | number | boolean | null) {
  const normalized = String(value ?? '').trim().toLowerCase()
  return normalized || 'all'
}

export function normalizeUsageOptionValue(value?: string | null) {
  return String(value || '').trim().toLowerCase()
}

export function normalizeUsageFilterValue(value?: string | null) {
  const normalized = normalizeUsageOptionValue(value)
  return normalized === '' || normalized === 'all' ? '' : normalized
}

export function normalizeUsageOperationFilterValue(value?: string | null): ExecutableSocialTaskAction | '' {
  const normalized = normalizeUsageFilterValue(value)
  return isExecutableUsageOperation(normalized) ? normalized : ''
}

export function normalizeUsageStatusFilterValue(value?: string | null): 'success' | 'failed' | '' {
  const normalized = normalizeUsageFilterValue(value)
  if (normalized === 'success' || normalized === 'failed') return normalized
  return ''
}

export function isExecutableUsageOperation(value?: string | null): value is ExecutableSocialTaskAction {
  return (EXECUTABLE_SOCIAL_TASK_ACTIONS as readonly string[]).includes(normalizeUsageOptionValue(value))
}

export function buildUsageFilterParams(state: UsageFilterState): UsageStatsQueryParams {
  const params: UsageStatsQueryParams = {}
  const operation = normalizeUsageOperationFilterValue(state.operation)
  const status = normalizeUsageStatusFilterValue(state.status)
  const platform = normalizeUsageFilterValue(state.platform)
  const account = String(state.account || '').trim()
  if (state.startDate) params.start_date = state.startDate
  if (state.endDate) params.end_date = state.endDate
  if (operation) params.operation = operation
  if (platform) params.platform = platform
  if (account) params.account = account
  if (status) params.status = status
  return params
}

export function buildUsageListQueryParams(
  filters: UsageFilterState,
  pageState: UsagePageQueryState,
): UsageQueryParams {
  return {
    page: pageState.page,
    page_size: pageState.pageSize,
    sort_by: pageState.sortBy || 'time',
    sort_order: pageState.sortOrder || 'desc',
    ...buildUsageFilterParams(filters),
  }
}

export function buildUsageExportQueryParams(
  filters: UsageFilterState,
  pageState: Pick<UsagePageQueryState, 'page' | 'pageSize'> & Partial<Pick<UsagePageQueryState, 'sortBy' | 'sortOrder'>>,
): UsageQueryParams {
  return buildUsageListQueryParams(filters, {
    ...pageState,
    sortBy: pageState.sortBy || 'time',
    sortOrder: pageState.sortOrder || 'desc',
  })
}

export function buildUsageStatsQueryParams(filters: UsageFilterState): UsageStatsQueryParams {
  return buildUsageFilterParams(filters)
}

export function parseProxySnapshotValue(raw: string): ParsedProxySnapshot | null {
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
    // Fallback to plain endpoint handling below.
  }

  if (looksLikeURL(raw)) {
    return { kind: 'endpoint', endpoint: raw }
  }

  return null
}

export function buildUsageCsv(
  items: UsageLog[],
  labels: UsageCsvLabels,
  formatters: UsageCsvFormatters,
) {
  const formatMoney = formatters.formatCurrency ?? formatCurrency
  const formatTime = formatters.formatDate ?? formatDate
  const headers = [
    labels.platform,
    labels.operation,
    labels.account,
    labels.result,
    labels.cost,
    labels.summary,
    labels.time,
    labels.target,
    labels.content,
  ]
  const rows = items.map(row => [
    formatters.platformLabel(row.platform),
    formatters.actionLabel(row.operation),
    row.account_name || '',
    formatters.statusLabel(row.status),
    formatMoney(row.cost),
    [formatters.resultSummary(row), formatters.resultMessage(row)].filter(Boolean).join(' - '),
    formatTime(row.completed_at || row.created_at),
    firstNonEmpty(row.target, row.payload?.target),
    firstNonEmpty(row.content, row.payload?.post?.text),
  ])
  return [headers, ...rows]
    .map(row => row.map(csvCell).join(','))
    .join('\r\n')
}

export function csvCell(value: unknown) {
  let text = String(value ?? '').replace(/\r?\n/g, ' ').trim()
  if (/^[=+\-@\t\r]/.test(text)) {
    text = `'${text}`
  }
  return `"${text.replace(/"/g, '""')}"`
}

export function hasMediaMetadata(item?: UsageTaskMediaRef | null) {
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

export function formatMediaDimensions(item?: UsageTaskMediaRef | null) {
  const width = Number(item?.width || 0)
  const height = Number(item?.height || 0)
  if (width <= 0 || height <= 0) return ''
  return `${formatNumber(width)} × ${formatNumber(height)}`
}

export function formatByteSize(value?: number | null) {
  const normalized = Number(value || 0)
  if (normalized <= 0) return ''
  return `${formatNumber(normalized)} B`
}

export function normalizeStringList(values?: string[] | null) {
  return (values ?? []).map(value => value.trim()).filter(Boolean)
}

export function mediaPreviewKey(scope: 'payload' | 'template', section: 'post' | 'avatar' | 'banner', index?: number) {
  if (typeof index === 'number' && index >= 0) return `${scope}:${section}:${index}`
  return `${scope}:${section}`
}

export function mediaPreviewTestID(previewKey: string) {
  return `usage-media-preview-${previewKey.replace(/[^a-z0-9]+/gi, '-')}`
}

export function shouldAttemptMediaPreview(item?: UsageTaskMediaRef | null) {
  if (!item || !hasMediaMetadata(item)) return false
  const contentType = String(item.content_type || '').trim().toLowerCase()
  return contentType === '' || contentType.startsWith('image/')
}

export function collectDetailMediaPreviewLocators(detail: UsageLog): Array<{ key: string; locator: UsageTaskMediaPreviewLocator }> {
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
