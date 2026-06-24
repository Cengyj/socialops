import { computed, getCurrentScope, onScopeDispose, ref, watch } from 'vue'
import proxiesAPI from '@/api/proxies'
import type { ProxyCheckResult, ProxyType, UserProxy } from '@/api/proxies'
import { useAppStore } from '@/stores/app'
import { formatAccountWorkbenchDate } from '@/utils/accountWorkbenchDate'
import { extractApiErrorCode, extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic as defaultRecordClientDiagnostic } from '@/utils/clientDiagnostics'
import { normalizeProxyStatus } from '@/utils/proxyStatus'
import { proxyTestResultSummary } from '@/utils/proxyTestSummary'
import {
  removeSelectedIds,
  retainExistingSelectedIds,
  toggleSelectedId,
  toggleVisibleSelectedIds,
  visibleSelectionState,
} from '@/utils/selection'
import type { PaginatedResponse, SelectOption } from '@/types'
import { createProxyErrorMessages } from './proxyErrorMessages'

export interface ProxyRow {
  id: number
  name: string
  type: ProxyType
  endpoint: string
  status: string
  latency: number | null
  lastCheck: string
  remark: string
  updatedAt: string
}

interface ProxyManagementApi {
  list(params?: {
    page?: number
    page_size?: number
    status?: string
    ip_type?: ProxyType
    search?: string
  }): Promise<PaginatedResponse<UserProxy>>
  test(id: number): Promise<ProxyCheckResult>
  testAll(): Promise<ProxyCheckResult[]>
}

interface ProxyNotifier {
  showError(message: string): void
  showSuccess(message: string): void
  showWarning(message: string): void
}

type TranslateFn = (key: string, paramsOrFallback?: Record<string, unknown> | string) => string
type RecordDiagnosticFn = (scope: string, error: unknown) => void

interface UseProxyManagementOptions {
  api?: ProxyManagementApi
  notifier?: ProxyNotifier
  recordDiagnostic?: RecordDiagnosticFn
  t: TranslateFn
}

export function useProxyManagement(options: UseProxyManagementOptions) {
  const api = options.api ?? proxiesAPI
  const notifier = options.notifier ?? useAppStore()
  const recordDiagnostic = options.recordDiagnostic ?? defaultRecordClientDiagnostic
  const t = options.t
  const proxyErrorMessages = computed(() => createProxyErrorMessages(t))

  const proxies = ref<ProxyRow[]>([])
  const loading = ref(false)
  const testing = ref(false)
  const loadError = ref('')
  const hasAnyProxy = ref(false)
  const searchQuery = ref('')
  const statusFilter = ref<string | number | boolean | null>('all')
  const typeFilter = ref<string | number | boolean | null>('all')
  const selectedIds = ref<number[]>([])
  const proxyFormDialogOpen = ref(false)
  const proxyDeleteDialogOpen = ref(false)
  const editingProxy = ref<ProxyRow | null>(null)
  const proxyToDelete = ref<ProxyRow | null>(null)

  const statusOptions = computed<SelectOption[]>(() => [
    { value: 'all', label: t('proxies.filters.allStatus') },
    { value: 'online', label: t('proxies.status.online') },
    { value: 'offline', label: t('proxies.status.offline') },
    { value: 'unknown', label: t('proxies.status.unknown') },
  ])

  const typeOptionsWithoutAll = computed<SelectOption[]>(() => [
    { value: 'residential', label: t('proxies.types.residential') },
    { value: 'static', label: t('proxies.types.static') },
    { value: 'dynamic', label: t('proxies.types.dynamic') },
    { value: 'mobile', label: t('proxies.types.mobile') },
    { value: 'datacenter', label: t('proxies.types.datacenter') },
  ])

  const typeOptions = computed<SelectOption[]>(() => [
    { value: 'all', label: t('proxies.filters.allTypes') },
    ...typeOptionsWithoutAll.value,
  ])

  const proxyListParams = computed(() => {
    const search = searchQuery.value.trim()
    const status = normalizeProxyStatusFilter(statusFilter.value)
    const type = normalizeProxyTypeFilter(typeFilter.value)
    const params: { page: number; page_size: number; search?: string; status?: string; ip_type?: ProxyType } = { page: 1, page_size: 200 }
    if (search) params.search = search
    if (status !== 'all') params.status = status
    if (type !== 'all') params.ip_type = type
    return params
  })

  const hasActiveProxyFilters = computed(() => {
    const search = searchQuery.value.trim()
    const status = normalizeProxyStatusFilter(statusFilter.value)
    const type = normalizeProxyTypeFilter(typeFilter.value)
    return search !== '' || status !== 'all' || type !== 'all'
  })

  const stats = computed(() => [
    { label: t('proxies.stats.total'), value: proxies.value.length },
    { label: t('proxies.stats.online'), value: proxies.value.filter(proxy => proxy.status === 'online').length },
    { label: t('proxies.stats.offline'), value: proxies.value.filter(proxy => proxy.status === 'offline').length },
    { label: t('proxies.stats.unknown'), value: proxies.value.filter(proxy => proxy.status === 'unknown').length },
  ])

  const visibleIds = computed(() => proxies.value.map(proxy => proxy.id))
  const currentVisibleSelectionState = computed(() => visibleSelectionState(selectedIds.value, visibleIds.value))
  const allVisibleSelected = computed(() => currentVisibleSelectionState.value.allSelected)
  const someVisibleSelected = computed(() => currentVisibleSelectionState.value.someSelected)

  let latestLoadRequestID = 0
  let filterReloadTimer: ReturnType<typeof setTimeout> | undefined
  const stopFilterWatcher = watch([searchQuery, statusFilter, typeFilter], () => {
    if (filterReloadTimer) clearTimeout(filterReloadTimer)
    filterReloadTimer = setTimeout(() => {
      void loadProxies()
    }, 250)
  })

  if (getCurrentScope()) {
    onScopeDispose(dispose)
  }

  function dispose() {
    stopFilterWatcher()
    if (filterReloadTimer) clearTimeout(filterReloadTimer)
  }

  async function loadProxies() {
    const requestID = ++latestLoadRequestID
    loading.value = true
    loadError.value = ''
    try {
      const result = await api.list(proxyListParams.value)
      if (!isLatestLoadRequest(requestID)) return
      const rows = (result.items ?? []).map(mapProxy)
      proxies.value = rows
      syncKnownProxyPoolState(result)
      selectedIds.value = retainExistingSelectedIds(selectedIds.value, proxies.value)
      syncDialogsFromProxyList()
    } catch (error) {
      if (!isLatestLoadRequest(requestID)) return
      recordDiagnostic('proxies.load', error)
      loadError.value = extractSafeApiErrorMessage(error, t('proxies.failedToLoad'), proxyErrorMessages.value)
      notifier.showError(loadError.value)
    } finally {
      if (isLatestLoadRequest(requestID)) {
        loading.value = false
      }
    }
  }

  function isLatestLoadRequest(requestID: number) {
    return requestID === latestLoadRequestID
  }

  function mapProxy(proxy: UserProxy): ProxyRow {
    return {
      id: proxy.id,
      name: String(proxy.name ?? '').trim(),
      type: normalizeProxyType(proxy.ip_type),
      endpoint: String(proxy.endpoint ?? '').trim(),
      status: normalizeProxyStatus(proxy.status),
      latency: normalizeProxyLatency(proxy.latency_ms),
      lastCheck: formatAccountWorkbenchDate(proxy.last_check_at),
      remark: String(proxy.remark ?? '').trim(),
      updatedAt: proxy.updated_at,
    }
  }

  function openCreateDialog() {
    if (testing.value) return
    editingProxy.value = null
    proxyFormDialogOpen.value = true
  }

  function openEditDialog(row: ProxyRow) {
    if (loading.value || testing.value) return
    editingProxy.value = row
    proxyFormDialogOpen.value = true
  }

  function openDeleteDialog(row: ProxyRow) {
    if (loading.value || testing.value) return
    proxyToDelete.value = row
    proxyDeleteDialogOpen.value = true
  }

  function closeProxyFormDialog() {
    proxyFormDialogOpen.value = false
    editingProxy.value = null
  }

  function closeProxyDeleteDialog() {
    proxyDeleteDialogOpen.value = false
    proxyToDelete.value = null
  }

  async function testProxy(id: number) {
    if (loading.value || testing.value) return
    testing.value = true
    try {
      const result = normalizeProxyCheckResult(await api.test(id))
      applyProxyTestResultsToRows([result])
      showProxyTestSummaryToast([result])
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed'), proxyErrorMessages.value))
      if (isProxyNotFoundError(error)) {
        removeDeletedProxyFromLocalState(id)
        await loadProxies()
      }
    } finally {
      testing.value = false
    }
  }

  async function testSelected() {
    if (selectedIds.value.length === 0 || loading.value || testing.value) return
    testing.value = true
    const ids = [...selectedIds.value]
    try {
      const settledResults = await Promise.allSettled(ids.map(id => api.test(id)))
      const results = settledResults.map((result, index) => {
        if (result.status === 'fulfilled') return result.value
        return buildFailedProxyCheckResult(ids[index], result.reason)
      }).map(normalizeProxyCheckResult)
      const rejectedCount = settledResults.filter(result => result.status === 'rejected').length
      applyProxyTestResultsToRows(results)
      showProxyTestSummaryToast(results, rejectedCount)
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test_selected', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed'), proxyErrorMessages.value))
    } finally {
      testing.value = false
    }
  }

  async function testAll() {
    if (loading.value || testing.value || !hasAnyProxy.value) return
    testing.value = true
    try {
      const result = (await api.testAll()).map(normalizeProxyCheckResult)
      applyProxyTestResultsToRows(result)
      showProxyTestSummaryToast(result)
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test_all', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed'), proxyErrorMessages.value))
    } finally {
      testing.value = false
    }
  }

  async function handleProxySaved(proxy?: UserProxy) {
    closeProxyFormDialog()
    if (proxy) {
      upsertSavedProxyIntoLocalState(proxy)
    } else {
      hasAnyProxy.value = true
    }
    await loadProxies()
  }

  function upsertSavedProxyIntoLocalState(proxy: UserProxy) {
    hasAnyProxy.value = true
    const row = mapProxy(proxy)
    const existingIndex = proxies.value.findIndex(item => item.id === row.id)
    if (!proxyMatchesActiveFilters(row)) {
      if (existingIndex >= 0) {
        proxies.value.splice(existingIndex, 1)
      }
      selectedIds.value = removeSelectedIds(selectedIds.value, [row.id])
      syncDialogsFromProxyList()
      return
    }
    if (existingIndex >= 0) {
      proxies.value.splice(existingIndex, 1, row)
    } else {
      proxies.value = [row, ...proxies.value]
    }
    syncDialogsFromProxyList()
  }

  async function handleProxyDeleted(id: number) {
    closeProxyDeleteDialog()
    removeDeletedProxyFromLocalState(id)
    await loadProxies()
  }

  function removeDeletedProxyFromLocalState(id: number) {
    proxies.value = proxies.value.filter(proxy => proxy.id !== id)
    selectedIds.value = removeSelectedIds(selectedIds.value, [id])
    if (!hasActiveProxyFilters.value) {
      hasAnyProxy.value = proxies.value.length > 0
    }
    syncDialogsFromProxyList()
  }

  function applyProxyTestResultsToRows(results: ProxyCheckResult[]) {
    if (results.length === 0 || proxies.value.length === 0) return
    const checkedAt = new Date().toLocaleString()
    const resultByID = new Map(results.map(result => [result.id, result]))
    proxies.value = proxies.value.map(proxy => {
      const result = resultByID.get(proxy.id)
      if (!result) return proxy
      return {
        ...proxy,
        status: normalizeProxyStatus(result.status),
        latency: normalizeProxyLatency(result.latency_ms),
        lastCheck: checkedAt,
      }
    }).filter(proxyMatchesActiveFilters)
    selectedIds.value = retainExistingSelectedIds(selectedIds.value, proxies.value)
    syncDialogsFromProxyList()
  }

  function buildFailedProxyCheckResult(id: number, error: unknown): ProxyCheckResult {
    recordDiagnostic('proxies.test_selected_item', error)
    return {
      id,
      status: 'unknown',
      latency_ms: 0,
    }
  }

  function showProxyTestSummaryToast(results: ProxyCheckResult[], rejectedCount = 0) {
    const total = results.length
    if (total <= 0) {
      notifier.showError(t('proxies.testFailed'))
      return
    }
    if (rejectedCount > 0 && rejectedCount === total) {
      notifier.showError(t('proxies.testFailed'))
      return
    }
    if (rejectedCount > 0) {
      notifier.showWarning(t('proxies.batchTestPartial', {
        ...proxyTestResultSummary(results),
        failed: rejectedCount,
      }))
      return
    }
    notifier.showSuccess(t('proxies.testResultSummary', { ...proxyTestResultSummary(results) }))
  }

  function syncDialogsFromProxyList() {
    if (editingProxy.value) {
      const updatedEditTarget = proxies.value.find(proxy => proxy.id === editingProxy.value?.id)
      if (updatedEditTarget) {
        editingProxy.value = updatedEditTarget
      } else {
        closeProxyFormDialog()
      }
    }
    if (!proxyToDelete.value) return
    const updatedDeleteTarget = proxies.value.find(proxy => proxy.id === proxyToDelete.value?.id)
    if (updatedDeleteTarget) {
      proxyToDelete.value = updatedDeleteTarget
    } else {
      closeProxyDeleteDialog()
    }
  }

  function syncKnownProxyPoolState(result: PaginatedResponse<UserProxy>) {
    if (!hasActiveProxyFilters.value) {
      hasAnyProxy.value = result.total > 0 || proxies.value.length > 0
      return
    }
    if (result.total > 0 || proxies.value.length > 0) {
      hasAnyProxy.value = true
    }
  }

  function proxyMatchesActiveFilters(proxy: ProxyRow) {
    const status = normalizeProxyStatusFilter(statusFilter.value)
    if (status !== 'all' && proxy.status !== status) return false
    const type = normalizeProxyTypeFilter(typeFilter.value)
    if (type !== 'all' && normalizeProxyType(proxy.type) !== type) return false
    const search = searchQuery.value.trim().toLowerCase()
    if (!search) return true
    return [proxy.name, proxy.endpoint, proxy.remark]
      .some(value => value.toLowerCase().includes(search))
  }

  function isSelected(id: number) {
    return selectedIds.value.includes(id)
  }

  function toggleSelection(id: number) {
    if (loading.value || testing.value) return
    selectedIds.value = toggleSelectedId(selectedIds.value, id)
  }

  function toggleAllVisible() {
    if (loading.value || testing.value) return
    selectedIds.value = toggleVisibleSelectedIds(selectedIds.value, visibleIds.value, allVisibleSelected.value)
  }

  function proxyStatusLabel(status: string) {
    const normalized = normalizeProxyStatus(status)
    return t(`proxies.status.${normalized}`, normalized)
  }

  function proxyTypeLabel(type: string) {
    const normalized = normalizeKnownProxyType(type)
    if (normalized) return t(`proxies.types.${normalized}`, normalized)
    return proxyTypeFallbackText(type)
  }

  function statusBadgeClass(status: string) {
    const normalized = normalizeProxyStatus(status)
    if (normalized === 'online') return 'badge-success'
    if (normalized === 'offline') return 'badge-danger'
    return 'badge-warning'
  }

  function normalizeProxyCheckResult(result: ProxyCheckResult): ProxyCheckResult {
    return {
      ...result,
      status: normalizeProxyStatus(result.status),
      latency_ms: normalizeProxyLatency(result.latency_ms) ?? 0,
    }
  }

  function normalizeProxyLatency(value: unknown) {
    const latency = Number(value)
    return Number.isFinite(latency) && latency > 0 ? latency : null
  }

  function normalizeProxyType(value: unknown): ProxyType {
    return normalizeKnownProxyType(value) ?? (proxyTypeFallbackText(value) as ProxyType)
  }

  function normalizeKnownProxyType(value: unknown): ProxyType | null {
    const normalized = String(value ?? '').trim().toLowerCase()
    if (
      normalized === 'residential' ||
      normalized === 'static' ||
      normalized === 'dynamic' ||
      normalized === 'mobile' ||
      normalized === 'datacenter'
    ) return normalized
    return null
  }

  function proxyTypeFallbackText(value: unknown) {
    const trimmed = String(value ?? '').trim()
    return trimmed || '-'
  }

  function normalizeProxyStatusFilter(value: string | number | boolean | null) {
    const normalized = String(value ?? '').trim().toLowerCase()
    if (normalized === 'online' || normalized === 'offline' || normalized === 'unknown') return normalized
    return 'all'
  }

  function normalizeProxyTypeFilter(value: string | number | boolean | null): ProxyType | 'all' {
    return normalizeKnownProxyType(value) ?? 'all'
  }

  function isProxyNotFoundError(error: unknown) {
    return extractApiErrorCode(error) === 'SOCIAL_IP_NOT_FOUND'
  }

  return {
    allVisibleSelected,
    closeProxyDeleteDialog,
    closeProxyFormDialog,
    dispose,
    editingProxy,
    hasActiveProxyFilters,
    hasAnyProxy,
    handleProxyDeleted,
    handleProxySaved,
    isSelected,
    loadError,
    loading,
    loadProxies,
    openCreateDialog,
    openDeleteDialog,
    openEditDialog,
    proxies,
    proxyDeleteDialogOpen,
    proxyFormDialogOpen,
    proxyStatusLabel,
    proxyToDelete,
    proxyTypeLabel,
    searchQuery,
    selectedIds,
    someVisibleSelected,
    stats,
    statusBadgeClass,
    statusFilter,
    statusOptions,
    testAll,
    testProxy,
    testSelected,
    testing,
    toggleAllVisible,
    toggleSelection,
    typeFilter,
    typeOptions,
  }
}
