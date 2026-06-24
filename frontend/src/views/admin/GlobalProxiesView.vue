<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div v-if="loadError" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <span class="min-w-0 break-words">{{ loadError }}</span>
              <button type="button" class="btn btn-secondary btn-sm" @click="loadProxies">{{ t('common.retry') }}</button>
            </div>
          </div>

          <CommonStatsGrid
            card-class="rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800"
            grid-class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
            layout="stacked"
            :stats="stats"
          />

          <div class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
            <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
              <div class="flex flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap">
                <SearchInput v-model="searchQuery" :placeholder="t('admin.globalProxies.searchPlaceholder')" class="w-full sm:w-72" />
                <select v-model="statusFilter" class="input h-10 w-full sm:w-40">
                  <option value="all">{{ t('proxies.filters.allStatus') }}</option>
                  <option value="online">{{ t('proxies.status.online') }}</option>
                  <option value="offline">{{ t('proxies.status.offline') }}</option>
                  <option value="unknown">{{ t('proxies.status.unknown') }}</option>
                </select>
                <select v-model="typeFilter" class="input h-10 w-full sm:w-44">
                  <option value="all">{{ t('proxies.filters.allTypes') }}</option>
                  <option value="residential">{{ t('proxies.types.residential') }}</option>
                  <option value="static">{{ t('proxies.types.static') }}</option>
                  <option value="dynamic">{{ t('proxies.types.dynamic') }}</option>
                  <option value="mobile">{{ t('proxies.types.mobile') }}</option>
                  <option value="datacenter">{{ t('proxies.types.datacenter') }}</option>
                </select>
              </div>
              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:flex xl:items-center">
                <div class="flex h-10 min-w-0 items-center justify-center rounded-lg border border-gray-200 bg-gray-50 px-3 text-sm font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
                  <span class="min-w-0 truncate">{{ t('proxies.selection.selectedCount', { count: selectedIds.length }) }}</span>
                </div>
                <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="loading" @click="loadProxies">
                  <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                  <span>{{ t('common.refresh') }}</span>
                </button>
                <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="selectedIds.length === 0 || loading || testing" @click="testSelected">
                  <Icon name="play" size="sm" />
                  <span>{{ t('proxies.testSelected') }}</span>
                </button>
                <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="loading || testing || !hasAnyProxy" @click="testAll">
                  <Icon name="checkCircle" size="sm" />
                  <span>{{ t('proxies.testAll') }}</span>
                </button>
                <button type="button" class="btn btn-primary btn-sm h-10 justify-center" :disabled="testing" @click="openCreateDialog">
                  <Icon name="plus" size="sm" />
                  <span>{{ t('admin.globalProxies.addProxy') }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="proxies" :loading="loading" row-key="id" default-sort-key="id" default-sort-order="asc">
          <template #header-select>
            <input type="checkbox" class="h-4 w-4" :checked="allVisibleSelected" :disabled="loading || testing" @change="toggleAllVisible" />
          </template>
          <template #cell-select="{ row }">
            <input type="checkbox" class="h-4 w-4" :checked="selectedIds.includes(row.id)" :disabled="loading || testing" @change="toggleSelection(row.id)" />
          </template>
          <template #cell-type="{ value }">
            <span class="rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">{{ proxyTypeLabel(String(value)) }}</span>
          </template>
          <template #cell-endpoint="{ value }">
            <span class="block min-w-0 break-all text-right sm:max-w-[420px] sm:truncate" :title="String(value || '')">{{ value || '-' }}</span>
          </template>
          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(String(value))]">{{ proxyStatusLabel(String(value)) }}</span>
          </template>
          <template #cell-latency="{ value }">
            <span>{{ value ? `${value}ms` : '-' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center justify-end gap-2">
              <button type="button" class="btn btn-secondary min-w-0 px-2 py-1 text-xs" :disabled="loading || testing" @click="testProxy(row.id)">{{ t('proxies.test') }}</button>
              <button type="button" class="btn btn-secondary min-w-0 px-2 py-1 text-xs" :disabled="loading || testing" @click="openEditDialog(row)">{{ t('common.edit') }}</button>
              <button type="button" class="btn btn-danger min-w-0 px-2 py-1 text-xs" :disabled="loading || testing" @click="openDeleteDialog(row)">{{ t('common.delete') }}</button>
            </div>
          </template>
          <template #empty>
            <div class="flex flex-col items-center py-8 text-center">
              <Icon name="server" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">{{ hasActiveFilters ? t('admin.globalProxies.noResults.title') : t('admin.globalProxies.empty.title') }}</p>
              <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">{{ hasActiveFilters ? t('admin.globalProxies.noResults.description') : t('admin.globalProxies.empty.description') }}</p>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="formOpen" :title="editingProxy ? t('admin.globalProxies.editTitle') : t('admin.globalProxies.addProxy')" width="wide" @close="requestCloseForm">
      <div class="grid gap-3 sm:grid-cols-2">
        <div>
          <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="admin-global-proxy-name">{{ t('proxies.form.name') }}</label>
          <input id="admin-global-proxy-name" v-model="form.name" class="input" :disabled="saving" :placeholder="t('admin.globalProxies.form.namePlaceholder')" />
        </div>
        <div>
          <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="admin-global-proxy-type">{{ t('proxies.form.type') }}</label>
          <select id="admin-global-proxy-type" v-model="form.ipType" class="input" :disabled="saving">
            <option value="residential">{{ t('proxies.types.residential') }}</option>
            <option value="static">{{ t('proxies.types.static') }}</option>
            <option value="dynamic">{{ t('proxies.types.dynamic') }}</option>
            <option value="mobile">{{ t('proxies.types.mobile') }}</option>
            <option value="datacenter">{{ t('proxies.types.datacenter') }}</option>
          </select>
        </div>
        <div class="sm:col-span-2">
          <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="admin-global-proxy-endpoint">{{ t('proxies.form.endpoint') }}</label>
          <input id="admin-global-proxy-endpoint" v-model="form.endpoint" class="input" :disabled="saving" :placeholder="t('proxies.form.endpointPlaceholder')" />
          <p class="mt-2 break-words text-xs text-gray-500 dark:text-gray-400">{{ t('admin.globalProxies.form.endpointHint') }}</p>
        </div>
        <div class="sm:col-span-2">
          <label class="mb-2 block text-xs font-medium text-gray-500 dark:text-gray-400" for="admin-global-proxy-remark">{{ t('proxies.form.remark') }}</label>
          <textarea id="admin-global-proxy-remark" v-model="form.remark" class="input min-h-[90px]" :disabled="saving" :placeholder="t('proxies.form.remarkPlaceholder')" />
        </div>
      </div>
      <div v-if="formError" class="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">{{ formError }}</div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="requestCloseForm">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="saving || form.name.trim() === ''" @click="saveProxy">{{ saving ? t('common.saving') : t('common.confirm') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="deleteOpen" :title="t('admin.globalProxies.deleteDialog.title')" width="normal" @close="requestCloseDelete">
      <div v-if="proxyToDelete" class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.globalProxies.deleteDialog.description', { name: proxyToDelete.name }) }}</p>
        <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('admin.globalProxies.deleteDialog.impact') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="deleting" @click="requestCloseDelete">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-danger" :disabled="deleting || !proxyToDelete" @click="deleteProxy">{{ deleting ? t('common.processing') : t('common.delete') }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import CommonStatsGrid from '@/components/common/CommonStatsGrid.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import adminGlobalProxiesAPI, { type AdminGlobalProxy, type AdminGlobalProxyCheckResult } from '@/api/admin/globalProxies'
import type { ProxyType } from '@/api/proxies'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { extractSafeApiErrorMessage } from '@/utils/apiError'
import { recordClientDiagnostic } from '@/utils/clientDiagnostics'
import { formatAccountWorkbenchDate } from '@/utils/accountWorkbenchDate'
import { normalizeProxyStatus } from '@/utils/proxyStatus'
import { proxyTestResultSummary } from '@/utils/proxyTestSummary'

interface ProxyRow {
  id: number
  name: string
  type: ProxyType
  endpoint: string
  status: string
  latency: number | null
  lastCheck: string
  lastUsed: string
  remark: string
  updatedAt: string
}

const { t } = useI18n()
const appStore = useAppStore()
const proxies = ref<ProxyRow[]>([])
const loading = ref(false)
const testing = ref(false)
const saving = ref(false)
const deleting = ref(false)
const loadError = ref('')
const selectedIds = ref<number[]>([])
const searchQuery = ref('')
const statusFilter = ref('all')
const typeFilter = ref('all')
const hasAnyProxy = ref(false)
const formOpen = ref(false)
const deleteOpen = ref(false)
const editingProxy = ref<ProxyRow | null>(null)
const proxyToDelete = ref<ProxyRow | null>(null)
const formError = ref('')
const form = reactive({
  name: '',
  ipType: 'residential' as ProxyType,
  endpoint: '',
  remark: '',
})

let filterTimer: ReturnType<typeof setTimeout> | undefined
watch([searchQuery, statusFilter, typeFilter], () => {
  if (filterTimer) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    void loadProxies()
  }, 250)
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-[56px] min-w-[56px]' },
  { key: 'id', label: 'ID', sortable: true, class: 'w-[84px] min-w-[84px]' },
  { key: 'name', label: t('proxies.columns.name'), sortable: true, class: 'min-w-[180px]' },
  { key: 'type', label: t('proxies.columns.type'), sortable: true, class: 'min-w-[128px]' },
  { key: 'endpoint', label: t('proxies.columns.endpoint'), class: 'min-w-[300px]' },
  { key: 'status', label: t('proxies.columns.status'), sortable: true, class: 'min-w-[120px]' },
  { key: 'latency', label: t('proxies.columns.latency'), sortable: true, class: 'min-w-[100px]' },
  { key: 'lastCheck', label: t('proxies.columns.lastCheck'), sortable: true, class: 'min-w-[180px]' },
  { key: 'lastUsed', label: t('admin.globalProxies.columns.lastUsed'), sortable: true, class: 'min-w-[180px]' },
  { key: 'actions', label: t('common.actions'), class: 'w-[180px] min-w-[180px]' },
])

const stats = computed(() => [
  { key: 'total', label: t('proxies.stats.total'), value: proxies.value.length },
  { key: 'online', label: t('proxies.stats.online'), value: proxies.value.filter(proxy => proxy.status === 'online').length },
  { key: 'offline', label: t('proxies.stats.offline'), value: proxies.value.filter(proxy => proxy.status === 'offline').length },
  { key: 'unknown', label: t('proxies.stats.unknown'), value: proxies.value.filter(proxy => proxy.status === 'unknown').length },
])
const hasActiveFilters = computed(() => searchQuery.value.trim() !== '' || statusFilter.value !== 'all' || typeFilter.value !== 'all')
const visibleIds = computed(() => proxies.value.map(proxy => proxy.id))
const allVisibleSelected = computed(() => visibleIds.value.length > 0 && visibleIds.value.every(id => selectedIds.value.includes(id)))

async function loadProxies() {
  loading.value = true
  loadError.value = ''
  try {
    const result = await adminGlobalProxiesAPI.list({
      page: 1,
      page_size: 200,
      ...(searchQuery.value.trim() ? { search: searchQuery.value.trim() } : {}),
      ...(statusFilter.value !== 'all' ? { status: statusFilter.value } : {}),
      ...(typeFilter.value !== 'all' ? { ip_type: typeFilter.value as ProxyType } : {}),
    })
    proxies.value = (result.items ?? []).map(mapProxy)
    hasAnyProxy.value = result.total > 0 || proxies.value.length > 0
    selectedIds.value = selectedIds.value.filter(id => proxies.value.some(proxy => proxy.id === id))
  } catch (error) {
    recordClientDiagnostic('admin.global_proxies.load', error)
    loadError.value = extractSafeApiErrorMessage(error, t('admin.globalProxies.failedToLoad'))
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

function mapProxy(proxy: AdminGlobalProxy): ProxyRow {
  return {
    id: proxy.id,
    name: String(proxy.name ?? '').trim(),
    type: proxy.ip_type,
    endpoint: String(proxy.endpoint ?? '').trim(),
    status: normalizeProxyStatus(proxy.status),
    latency: Number.isFinite(Number(proxy.latency_ms)) && Number(proxy.latency_ms) > 0 ? Number(proxy.latency_ms) : null,
    lastCheck: formatAccountWorkbenchDate(proxy.last_check_at),
    lastUsed: formatAccountWorkbenchDate(proxy.last_used_at),
    remark: String(proxy.remark ?? '').trim(),
    updatedAt: proxy.updated_at,
  }
}

function openCreateDialog() {
  editingProxy.value = null
  form.name = ''
  form.ipType = 'residential'
  form.endpoint = ''
  form.remark = ''
  formError.value = ''
  formOpen.value = true
}

function openEditDialog(row: ProxyRow) {
  editingProxy.value = row
  form.name = row.name
  form.ipType = row.type
  form.endpoint = row.endpoint
  form.remark = row.remark
  formError.value = ''
  formOpen.value = true
}

function requestCloseForm() {
  if (saving.value) return
  closeFormDialog()
}

function closeFormDialog() {
  formOpen.value = false
  editingProxy.value = null
}

function openDeleteDialog(row: ProxyRow) {
  proxyToDelete.value = row
  deleteOpen.value = true
}

function requestCloseDelete() {
  if (deleting.value) return
  closeDeleteDialog()
}

function closeDeleteDialog() {
  deleteOpen.value = false
  proxyToDelete.value = null
}

async function saveProxy() {
  if (saving.value || form.name.trim() === '') return
  saving.value = true
  formError.value = ''
  const payload = {
    name: form.name.trim(),
    ip_type: form.ipType,
    endpoint: form.endpoint.trim(),
    remark: form.remark.trim(),
  }
  try {
    if (editingProxy.value) {
      await adminGlobalProxiesAPI.update(editingProxy.value.id, payload)
      appStore.showSuccess(t('admin.globalProxies.saved'))
    } else {
      await adminGlobalProxiesAPI.create(payload)
      appStore.showSuccess(t('admin.globalProxies.created'))
    }
    closeFormDialog()
    await loadProxies()
  } catch (error) {
    recordClientDiagnostic('admin.global_proxies.save', error)
    formError.value = extractSafeApiErrorMessage(error, t('admin.globalProxies.saveFailed'))
    appStore.showError(formError.value)
  } finally {
    saving.value = false
  }
}

async function deleteProxy() {
  if (deleting.value || !proxyToDelete.value) return
  deleting.value = true
  try {
    await adminGlobalProxiesAPI.delete(proxyToDelete.value.id)
    appStore.showSuccess(t('admin.globalProxies.deleted'))
    closeDeleteDialog()
    await loadProxies()
  } catch (error) {
    recordClientDiagnostic('admin.global_proxies.delete', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('admin.globalProxies.deleteFailed')))
  } finally {
    deleting.value = false
  }
}

async function testProxy(id: number) {
  if (testing.value) return
  testing.value = true
  try {
    const result = normalizeTestResult(await adminGlobalProxiesAPI.test(id))
    showProxyTestSummaryToast([result])
    await loadProxies()
  } catch (error) {
    recordClientDiagnostic('admin.global_proxies.test', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed')))
  } finally {
    testing.value = false
  }
}

async function testSelected() {
  if (testing.value || selectedIds.value.length === 0) return
  testing.value = true
  try {
    const ids = [...selectedIds.value]
    const settled = await Promise.allSettled(ids.map(id => adminGlobalProxiesAPI.test(id)))
    const results = settled.map((item, index) => item.status === 'fulfilled'
      ? normalizeTestResult(item.value)
      : buildFailedGlobalProxyCheckResult(ids[index], item.reason))
    const rejectedCount = settled.filter(item => item.status === 'rejected').length
    showProxyTestSummaryToast(results, rejectedCount)
    await loadProxies()
  } finally {
    testing.value = false
  }
}

async function testAll() {
  if (testing.value || !hasAnyProxy.value) return
  testing.value = true
  try {
    const results = (await adminGlobalProxiesAPI.testAll()).map(normalizeTestResult)
    showProxyTestSummaryToast(results)
    await loadProxies()
  } catch (error) {
    recordClientDiagnostic('admin.global_proxies.test_all', error)
    appStore.showError(extractSafeApiErrorMessage(error, t('proxies.testFailed')))
  } finally {
    testing.value = false
  }
}

function normalizeTestResult(result: AdminGlobalProxyCheckResult): AdminGlobalProxyCheckResult {
  return {
    ...result,
    status: normalizeProxyStatus(result.status),
    latency_ms: Number(result.latency_ms) > 0 ? Number(result.latency_ms) : 0,
  }
}

function buildFailedGlobalProxyCheckResult(id: number, error: unknown): AdminGlobalProxyCheckResult {
  recordClientDiagnostic('admin.global_proxies.test_selected_item', error)
  return { id, status: 'unknown', latency_ms: 0 }
}

function showProxyTestSummaryToast(results: AdminGlobalProxyCheckResult[], rejectedCount = 0) {
  if (results.length === 0) {
    appStore.showError(t('proxies.testFailed'))
    return
  }
  if (rejectedCount > 0 && rejectedCount === results.length) {
    appStore.showError(t('proxies.testFailed'))
    return
  }
  const summary = proxyTestResultSummary(results)
  if (rejectedCount > 0) {
    appStore.showWarning(t('proxies.batchTestPartial', { ...summary, failed: rejectedCount }))
    return
  }
  appStore.showSuccess(t('proxies.testResultSummary', { ...summary }))
}

function toggleSelection(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter(item => item !== id)
    : [...selectedIds.value, id]
}

function toggleAllVisible() {
  selectedIds.value = allVisibleSelected.value ? [] : [...visibleIds.value]
}

function proxyStatusLabel(status: string) {
  const normalized = normalizeProxyStatus(status)
  return t(`proxies.status.${normalized}`, normalized)
}

function proxyTypeLabel(type: string) {
  return t(`proxies.types.${type}`, type)
}

function statusBadgeClass(status: string) {
  const normalized = normalizeProxyStatus(status)
  if (normalized === 'online') return 'badge-success'
  if (normalized === 'offline') return 'badge-danger'
  return 'badge-warning'
}

void loadProxies()
</script>
