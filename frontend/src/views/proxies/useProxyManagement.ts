import { computed, getCurrentScope, onScopeDispose, ref, watch } from 'vue'
import proxiesAPI from '@/api/proxies'
import type { ProxyCheckResult, ProxyType, UserProxy } from '@/api/proxies'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage, extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic as defaultRecordClientDiagnostic } from '@/utils/clientDiagnostics'
import type { PaginatedResponse, SelectOption } from '@/types'

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

  const proxies = ref<ProxyRow[]>([])
  const loading = ref(false)
  const testing = ref(false)
  const loadError = ref('')
  const searchQuery = ref('')
  const statusFilter = ref<string | number | boolean | null>('all')
  const typeFilter = ref<string | number | boolean | null>('all')
  const selectedIds = ref<number[]>([])
  const lastTestResults = ref<ProxyCheckResult[]>([])
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
    { value: 'mobile', label: t('proxies.types.mobile') },
    { value: 'datacenter', label: t('proxies.types.datacenter') },
  ])

  const typeOptions = computed<SelectOption[]>(() => [
    { value: 'all', label: t('proxies.filters.allTypes') },
    ...typeOptionsWithoutAll.value,
  ])

  const proxyListParams = computed(() => {
    const search = searchQuery.value.trim()
    const status = String(statusFilter.value || 'all')
    const type = String(typeFilter.value || 'all')
    const params: { page: number; page_size: number; search?: string; status?: string; ip_type?: ProxyType } = { page: 1, page_size: 200 }
    if (search) params.search = search
    if (status !== 'all') params.status = status
    if (type !== 'all') params.ip_type = type as ProxyType
    return params
  })

  const hasActiveProxyFilters = computed(() => {
    const search = searchQuery.value.trim()
    const status = String(statusFilter.value || 'all')
    const type = String(typeFilter.value || 'all')
    return search !== '' || status !== 'all' || type !== 'all'
  })

  const stats = computed(() => [
    { label: t('proxies.stats.total'), value: proxies.value.length },
    { label: t('proxies.stats.online'), value: proxies.value.filter(proxy => proxy.status === 'online').length },
    { label: t('proxies.stats.offline'), value: proxies.value.filter(proxy => proxy.status === 'offline').length },
    { label: t('proxies.stats.unknown'), value: proxies.value.filter(proxy => proxy.status === 'unknown').length },
  ])

  const visibleIds = computed(() => proxies.value.map(proxy => proxy.id))
  const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))
  const someVisibleSelected = computed(() => visibleIds.value.some(id => selectedIds.value.includes(id)) && !allVisibleSelected.value)
  const testResultSummary = computed(() => {
    const total = lastTestResults.value.length
    const online = lastTestResults.value.filter(row => row.status === 'online').length
    const offline = lastTestResults.value.filter(row => row.status === 'offline').length
    const unknown = total - online - offline
    return { total, online, offline, unknown }
  })
  const testResultPreviewRows = computed(() => lastTestResults.value.slice(0, 6))

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
    loading.value = true
    loadError.value = ''
    try {
      const result = await api.list(proxyListParams.value)
      proxies.value = (result.items ?? []).map(mapProxy)
      selectedIds.value = selectedIds.value.filter(id => proxies.value.some(proxy => proxy.id === id))
    } catch (error) {
      recordDiagnostic('proxies.load', error)
      loadError.value = extractSafeApiErrorMessage(error, t('proxies.failedToLoad'))
      notifier.showError(loadError.value)
    } finally {
      loading.value = false
    }
  }

  function mapProxy(proxy: UserProxy): ProxyRow {
    return {
      id: proxy.id,
      name: proxy.name,
      type: proxy.ip_type,
      endpoint: proxy.endpoint ?? '',
      status: proxy.status || 'unknown',
      latency: proxy.latency_ms ?? null,
      lastCheck: proxy.last_check_at ? new Date(proxy.last_check_at).toLocaleString() : '-',
      remark: proxy.remark ?? '',
      updatedAt: proxy.updated_at,
    }
  }

  function openCreateDialog() {
    editingProxy.value = null
    proxyFormDialogOpen.value = true
  }

  function openEditDialog(row: ProxyRow) {
    editingProxy.value = row
    proxyFormDialogOpen.value = true
  }

  function openDeleteDialog(row: ProxyRow) {
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
    testing.value = true
    try {
      const result = await api.test(id)
      recordProxyTestResults([result])
      notifier.showSuccess(t('proxies.testResult', { status: proxyStatusLabel(result.status) }))
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed')))
    } finally {
      testing.value = false
    }
  }

  async function testSelected() {
    if (selectedIds.value.length === 0 || testing.value) return
    testing.value = true
    const ids = [...selectedIds.value]
    try {
      const settledResults = await Promise.allSettled(ids.map(id => api.test(id)))
      const results: ProxyCheckResult[] = settledResults.map((result, index) => {
        if (result.status === 'fulfilled') return result.value
        return buildFailedProxyCheckResult(ids[index], result.reason)
      })
      const rejectedCount = settledResults.filter(result => result.status === 'rejected').length
      recordProxyTestResults(results)
      if (rejectedCount === ids.length) {
        notifier.showError(t('proxies.testFailed'))
      } else if (rejectedCount > 0) {
        notifier.showWarning(t('proxies.batchTestPartial', { failed: rejectedCount, total: ids.length }))
      } else {
        notifier.showSuccess(t('proxies.batchTestSubmitted', { count: ids.length }))
      }
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test_selected', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed')))
    } finally {
      testing.value = false
    }
  }

  async function testAll() {
    if (testing.value) return
    testing.value = true
    try {
      const result = await api.testAll()
      recordProxyTestResults(result)
      notifier.showSuccess(t('proxies.batchTestSubmitted', { count: result.length }))
      await loadProxies()
    } catch (error) {
      recordDiagnostic('proxies.test_all', error)
      notifier.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed')))
    } finally {
      testing.value = false
    }
  }

  async function handleProxySaved() {
    closeProxyFormDialog()
    await loadProxies()
  }

  async function handleProxyDeleted(id: number) {
    closeProxyDeleteDialog()
    selectedIds.value = selectedIds.value.filter(selectedId => selectedId !== id)
    await loadProxies()
  }

  function recordProxyTestResults(results: ProxyCheckResult[]) {
    lastTestResults.value = results
  }

  function buildFailedProxyCheckResult(id: number, error: unknown): ProxyCheckResult {
    recordDiagnostic('proxies.test_selected_item', error)
    return {
      id,
      status: 'unknown',
      latency_ms: 0,
      error: extractApiErrorMessage(error, t('proxies.testFailed')),
    }
  }

  function clearTestResults() {
    lastTestResults.value = []
  }

  function proxyNameById(id: number) {
    return proxies.value.find(proxy => proxy.id === id)?.name || `#${id}`
  }

  function proxyTestRowToneClass(status: string) {
    if (status === 'online') return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-200'
    if (status === 'offline') return 'border-red-200 bg-red-50 text-red-800 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200'
    return 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200'
  }

  function isSelected(id: number) {
    return selectedIds.value.includes(id)
  }

  function toggleSelection(id: number) {
    selectedIds.value = isSelected(id) ? selectedIds.value.filter(selectedId => selectedId !== id) : [...selectedIds.value, id]
  }

  function toggleAllVisible() {
    if (allVisibleSelected.value) {
      selectedIds.value = selectedIds.value.filter(id => !visibleIds.value.includes(id))
      return
    }
    selectedIds.value = Array.from(new Set([...selectedIds.value, ...visibleIds.value]))
  }

  function proxyStatusLabel(status: string) {
    return t(`proxies.status.${status}`, status || '-')
  }

  function proxyTypeLabel(type: string) {
    return t(`proxies.types.${type}`, type || '-')
  }

  function statusBadgeClass(status: string) {
    if (status === 'online') return 'badge-success'
    if (status === 'offline') return 'badge-danger'
    return 'badge-warning'
  }

  return {
    allVisibleSelected,
    closeProxyDeleteDialog,
    closeProxyFormDialog,
    clearTestResults,
    dispose,
    editingProxy,
    hasActiveProxyFilters,
    handleProxyDeleted,
    handleProxySaved,
    isSelected,
    lastTestResults,
    loadError,
    loading,
    loadProxies,
    openCreateDialog,
    openDeleteDialog,
    openEditDialog,
    proxies,
    proxyDeleteDialogOpen,
    proxyFormDialogOpen,
    proxyNameById,
    proxyStatusLabel,
    proxyTestRowToneClass,
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
    testResultPreviewRows,
    testResultSummary,
    testSelected,
    testing,
    toggleAllVisible,
    toggleSelection,
    typeFilter,
    typeOptions,
  }
}
