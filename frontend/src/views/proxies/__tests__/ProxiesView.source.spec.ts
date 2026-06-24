import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewPath = resolve(testDir, '../ProxiesView.vue')
const dialogsPath = resolve(testDir, '../ProxyDialogs.vue')
const tablePath = resolve(testDir, '../components/ProxyTable.vue')
const toolbarPath = resolve(testDir, '../components/ProxyToolbar.vue')
const managementPath = resolve(testDir, '../useProxyManagement.ts')
const proxyErrorMessagesPath = resolve(testDir, '../proxyErrorMessages.ts')
const proxyActionTitlesPath = resolve(testDir, '../proxyActionTitles.ts')
const proxyStatusUtilPath = resolve(testDir, '../../../utils/proxyStatus.ts')
const proxyTestSummaryPath = resolve(testDir, '../../../utils/proxyTestSummary.ts')
const proxyApiPath = resolve(testDir, '../../../api/proxies.ts')
const enLocalePath = resolve(testDir, '../../../i18n/locales/en.ts')
const zhLocalePath = resolve(testDir, '../../../i18n/locales/zh.ts')
const viewSource = readFileSync(viewPath, 'utf8')
const dialogsSource = readFileSync(dialogsPath, 'utf8')
const tableSource = readFileSync(tablePath, 'utf8')
const toolbarSource = readFileSync(toolbarPath, 'utf8')
const managementSource = readFileSync(managementPath, 'utf8')
const proxyErrorMessagesSource = readFileSync(proxyErrorMessagesPath, 'utf8')
const proxyActionTitlesSource = readFileSync(proxyActionTitlesPath, 'utf8')
const proxyStatusUtilSource = readFileSync(proxyStatusUtilPath, 'utf8')
const proxyTestSummarySource = readFileSync(proxyTestSummaryPath, 'utf8')
const proxyApiSource = readFileSync(proxyApiPath, 'utf8')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')

describe('ProxiesView source contract', () => {
  it('keeps ProxiesView as a thin orchestration layer', () => {
    expect(viewSource).toContain("import { useProxyManagement } from './useProxyManagement'")
    expect(viewSource).toContain("import ProxyStatsGrid from './components/ProxyStatsGrid.vue'")
    expect(viewSource).toContain("import ProxyToolbar from './components/ProxyToolbar.vue'")
    expect(viewSource).toContain("import ProxyTable from './components/ProxyTable.vue'")
    expect(viewSource).toContain("import LoadErrorBanner from '@/components/common/LoadErrorBanner.vue'")
    expect(viewSource).not.toContain("ProxyTestResultsPanel")
    expect(viewSource).not.toContain("import proxiesAPI from '@/api/proxies'")
    expect(viewSource).not.toContain('Promise.allSettled')
    expect(viewSource).not.toContain('function buildFailedProxyCheckResult')
    expect(viewSource).not.toContain('const proxyListParams = computed')
  })

  it('keeps the page load error affordance aligned with the other core pages', () => {
    const loadErrorSource = viewSource.slice(
      viewSource.indexOf('<LoadErrorBanner'),
      viewSource.indexOf('<ProxyStatsGrid'),
    )

    expect(loadErrorSource).toContain('v-if="loadError"')
    expect(loadErrorSource).toContain(":title=\"t('proxies.failedToLoad')\"")
    expect(loadErrorSource).toContain(':message="loadError"')
    expect(loadErrorSource).toContain(":retry-label=\"t('common.retry')\"")
    expect(loadErrorSource).toContain('@retry="loadProxies"')
    expect(viewSource).not.toContain('<div v-if="loadError" class="rounded-lg border border-red-200')
  })

  it('uses backend proxy filters instead of searching only the first loaded page', () => {
    expect(managementSource).toContain('const proxyListParams = computed')
    expect(managementSource).toContain('api.list(proxyListParams.value)')
    expect(managementSource).toContain("params.search = search")
    expect(managementSource).toContain("params.status = status")
    expect(managementSource).toContain("params.ip_type = type")
    expect(managementSource).toContain('normalizeProxyStatusFilter(statusFilter.value)')
    expect(managementSource).toContain('normalizeProxyTypeFilter(typeFilter.value)')
    expect(managementSource).toContain('watch([searchQuery, statusFilter, typeFilter]')
    expect(viewSource).not.toContain('return proxies.value.filter((proxy) =>')
  })

  it('keeps dynamic proxy type available across API, filters, and dialogs', () => {
    expect(proxyApiSource).toContain("'dynamic'")
    expect(managementSource).toContain("{ value: 'dynamic', label: t('proxies.types.dynamic') }")
    expect(managementSource).toContain("normalized === 'dynamic'")
    expect(dialogsSource).toContain("{ value: 'dynamic', label: t('proxies.types.dynamic') }")
    expect(zhLocaleSource).toContain('"dynamic": "动态"')
    expect(enLocaleSource).toContain('"dynamic": "Dynamic"')
  })

  it('uses the shared proxy status normalizer for row and result status handling', () => {
    expect(managementSource).toContain("import { normalizeProxyStatus } from '@/utils/proxyStatus'")
    expect(managementSource).toContain('status: normalizeProxyStatus(proxy.status)')
    expect(managementSource).toContain('status: normalizeProxyStatus(result.status)')
    expect(managementSource).not.toContain('function normalizeProxyStatus(value: unknown)')
    expect(proxyStatusUtilSource).toContain('export function normalizeProxyStatus')
  })

  it('uses the shared account workbench date formatter for proxy last-check values', () => {
    expect(managementSource).toContain("import { formatAccountWorkbenchDate } from '@/utils/accountWorkbenchDate'")
    expect(managementSource).toContain('lastCheck: formatAccountWorkbenchDate(proxy.last_check_at)')
    expect(managementSource).not.toContain('lastCheck: proxy.last_check_at ? new Date(proxy.last_check_at).toLocaleString()')
  })

  it('uses product-grade confirmation and toast-based connectivity result feedback', () => {
    expect(viewSource).toContain('<ProxyDialogs')
    expect(dialogsSource).toContain('showDelete')
    expect(dialogsSource).toContain('proxyToDelete')
    expect(dialogsSource).toContain('confirmDeleteProxy')
    expect(dialogsSource).toContain("t('proxies.deleteDialog.title')")
    expect(viewSource).not.toContain('window.confirm')
    expect(dialogsSource).not.toContain('window.confirm')

    expect(viewSource).not.toContain('<ProxyTestResultsPanel')
    expect(viewSource).not.toContain('lastTestResults')
    expect(viewSource).not.toContain('clearTestResults')
    expect(managementSource).not.toContain('lastTestResults')
    expect(managementSource).not.toContain('clearTestResults')
    expect(managementSource).not.toContain('proxyNameById')
    expect(managementSource).not.toContain('useOperationResultPreview')
    expect(managementSource).not.toContain('PROXY_TEST_RESULT_PREVIEW_LIMIT')
    expect(managementSource).toContain("import { proxyTestResultSummary } from '@/utils/proxyTestSummary'")
    expect(managementSource).toContain('function showProxyTestSummaryToast')
    expect(managementSource).toContain("t('proxies.testResultSummary'")
    expect(proxyTestSummarySource).toContain('export function proxyTestResultSummary')
  })

  it('uses action-specific safe proxy save errors instead of a generic fallback', () => {
    expect(dialogsSource).toContain("import { createProxyErrorMessages } from './proxyErrorMessages'")
    expect(managementSource).toContain("import { createProxyErrorMessages } from './proxyErrorMessages'")
    expect(proxyErrorMessagesSource).toContain('SOCIAL_IP_SERVICE_UNAVAILABLE')
    expect(proxyErrorMessagesSource).toContain("t('proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE')")
    expect(dialogsSource).toContain("extractSafeApiErrorMessage(error, t('proxies.saveFailed'), proxyErrorMessages.value)")
    expect(managementSource).toContain("extractSafeApiErrorMessage(error, t('proxies.failedToLoad'), proxyErrorMessages.value)")
    expect(managementSource).toContain("extractSafeApiErrorMessage(error, t('proxies.testFailed'), proxyErrorMessages.value)")
    expect(dialogsSource).not.toContain("extractSafeApiErrorMessage(error, t('common.error'), proxyErrorMessages.value)")
    expect(enLocaleSource).toContain('"saveFailed": "Failed to save proxy"')
    expect(enLocaleSource).toContain('"SOCIAL_IP_SERVICE_UNAVAILABLE": "Proxy service is temporarily unavailable. Try again later."')
    expect(zhLocaleSource).toContain('"saveFailed": "保存代理失败"')
    expect(zhLocaleSource).toContain('"SOCIAL_IP_SERVICE_UNAVAILABLE": "代理服务暂不可用，请稍后重试。"')
  })

  it('keeps proxy endpoint labels localized in the Chinese UI', () => {
    expect(zhLocaleSource).toContain('"description": "管理你自己的执行代理。只有在线且代理地址有效的代理可以分配给账号。"')
    expect(zhLocaleSource).toContain('"endpoint": "代理地址"')
    expect(zhLocaleSource).toContain('"endpointHint": "可填写静态代理地址，或填写返回 ip/port/username/password 的提取链接；执行时会从链接获取新的代理。"')
    expect(zhLocaleSource).toContain('"INVALID_PROXY_ENDPOINT": "代理地址格式无效，请填写完整的 http、https、socks5 地址或有效的代理提取链接。"')
    expect(zhLocaleSource).toContain('"hint": "只能分配属于你且代理地址有效的在线代理；清空后会移除账号默认代理。"')
  })

  it('preserves per-proxy feedback when selected connectivity checks partially fail', () => {
    expect(managementSource).toContain('Promise.allSettled')
    expect(managementSource).toContain('buildFailedProxyCheckResult')
    expect(managementSource).toContain("recordDiagnostic('proxies.test_selected_item'")
    expect(managementSource).toContain("extractSafeApiErrorMessage(error, t('proxies.testFailed'), proxyErrorMessages.value)")
    expect(managementSource).not.toContain('extractApiErrorMessage(error, t(\'proxies.testFailed\'))')
    expect(managementSource).toContain('showProxyTestSummaryToast(results, rejectedCount)')
    expect(managementSource).toContain('applyProxyTestResultsToRows(results)')
  })

  it('guards every connectivity check entry point against duplicate submissions', () => {
    const singleTestSource = managementSource.slice(
      managementSource.indexOf('async function testProxy'),
      managementSource.indexOf('async function testSelected'),
    )
    const selectedTestSource = managementSource.slice(
      managementSource.indexOf('async function testSelected'),
      managementSource.indexOf('async function testAll'),
    )
    const allTestSource = managementSource.slice(
      managementSource.indexOf('async function testAll'),
      managementSource.indexOf('async function handleProxySaved'),
    )

    expect(singleTestSource).toContain('if (loading.value || testing.value) return')
    expect(selectedTestSource).toContain('if (selectedIds.value.length === 0 || loading.value || testing.value) return')
    expect(allTestSource).toContain('if (loading.value || testing.value || !hasAnyProxy.value) return')
  })

  it('locks row-level proxy actions while connectivity checks are running', () => {
    const headerSelectSource = tableSource.slice(
      tableSource.indexOf('<template #header-select'),
      tableSource.indexOf('</template>', tableSource.indexOf('<template #header-select')),
    )
    const cellSelectSource = tableSource.slice(
      tableSource.indexOf('<template #cell-select'),
      tableSource.indexOf('</template>', tableSource.indexOf('<template #cell-select')),
    )
    const nameCellSource = tableSource.slice(
      tableSource.indexOf('<template #cell-name'),
      tableSource.indexOf('</template>', tableSource.indexOf('<template #cell-name')),
    )
    const rowActionsSource = tableSource.slice(
      tableSource.indexOf('<template #cell-actions'),
      tableSource.indexOf('</template>', tableSource.indexOf('<template #cell-actions')),
    )

    expect(headerSelectSource).toContain(':disabled="loading || testing"')
    expect(cellSelectSource).toContain(':disabled="loading || testing"')
    expect(nameCellSource).toContain(':disabled="loading || testing"')
    expect(rowActionsSource.match(/:disabled="loading \|\| testing"/g)).toHaveLength(3)
    expect(rowActionsSource).toContain(':aria-label="rowActionTestButtonTitle"')
    expect(rowActionsSource).toContain(':aria-label="rowActionEditButtonTitle"')
    expect(rowActionsSource).toContain(':aria-label="rowActionDeleteButtonTitle"')
    expect(rowActionsSource).toContain(':title="rowActionTestButtonTitle"')
    expect(rowActionsSource).toContain(':title="rowActionEditButtonTitle"')
    expect(rowActionsSource).toContain(':title="rowActionDeleteButtonTitle"')
    expect(tableSource).toContain("from '../proxyActionTitles'")
    expect(tableSource).toContain('const rowActionTestButtonTitle = computed(() => buildRowTestButtonTitle')
    expect(tableSource).toContain('const rowActionEditButtonTitle = computed(() => buildRowEditButtonTitle')
    expect(tableSource).toContain('const rowActionDeleteButtonTitle = computed(() => buildRowDeleteButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyRowTestButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyRowEditButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyRowDeleteButtonTitle')
    expect(nameCellSource).toContain("@click=\"emit('edit', row)\"")
    expect(rowActionsSource).toContain("@click=\"emit('test', row.id)\"")
    expect(rowActionsSource).toContain("@click=\"emit('edit', row)\"")
    expect(rowActionsSource).toContain("@click=\"emit('delete', row)\"")
    expect(managementSource).toContain('function openCreateDialog() {\n    if (testing.value) return')
    expect(managementSource).toContain('function openEditDialog(row: ProxyRow) {\n    if (loading.value || testing.value) return')
    expect(managementSource).toContain('function openDeleteDialog(row: ProxyRow) {\n    if (loading.value || testing.value) return')
    expect(managementSource).toContain('function toggleSelection(id: number) {\n    if (loading.value || testing.value) return')
    expect(managementSource).toContain('function toggleAllVisible() {\n    if (loading.value || testing.value) return')
  })

  it('locks create and test toolbar actions while connectivity checks are running', () => {
    const toolbarActionsSource = toolbarSource.slice(
      toolbarSource.indexOf('<div class="grid grid-cols-1 gap-2'),
      toolbarSource.indexOf('</template>'),
    )

    expect(toolbarActionsSource).toContain(':disabled="selectedCount === 0 || loading || testing"')
    expect(toolbarActionsSource).toContain(':disabled="loading || testing || !hasProxies"')
    expect(toolbarActionsSource).toContain(':disabled="testing"')
    expect(toolbarActionsSource).toContain(':aria-label="refreshButtonTitle"')
    expect(toolbarActionsSource).toContain(':aria-label="testSelectedButtonTitle"')
    expect(toolbarActionsSource).toContain(':aria-label="testAllButtonTitle"')
    expect(toolbarActionsSource).toContain(':aria-label="createButtonTitle"')
    expect(toolbarActionsSource).toContain(':title="testSelectedButtonTitle"')
    expect(toolbarActionsSource).toContain(':title="testAllButtonTitle"')
    expect(toolbarActionsSource).toContain(':title="createButtonTitle"')
    expect(toolbarActionsSource).toContain("@click=\"emit('create')\"")
    expect(toolbarSource).toContain("from '../proxyActionTitles'")
    expect(toolbarSource).toContain('const testSelectedButtonTitle = computed(() => buildTestSelectedButtonTitle')
    expect(toolbarSource).toContain('const testAllButtonTitle = computed(() => buildTestAllButtonTitle')
    expect(toolbarSource).toContain('const createButtonTitle = computed(() => buildCreateButtonTitle')
    expect(proxyActionTitlesSource).toContain("if (state.selectedCount === 0) return t('proxies.selection.noneSelected')")
    expect(proxyActionTitlesSource).toContain("if (!state.hasProxies) return t('proxies.noProxiesToTest')")
    expect(enLocaleSource).toContain('"noneSelected": "Select at least one proxy to test."')
    expect(enLocaleSource).toContain('"noProxiesToTest": "No proxies to test."')
    expect(zhLocaleSource).toContain('"noneSelected": "请至少选择一个代理后再测试。"')
    expect(zhLocaleSource).toContain('"noProxiesToTest": "当前没有可测试的代理。"')
  })

  it('distinguishes an empty proxy pool from zero matches after filters are applied', () => {
    expect(viewSource).toContain('hasActiveProxyFilters')
    expect(tableSource).toContain("hasActiveProxyFilters ? t('proxies.noResults.title') : t('proxies.empty.title')")
    expect(tableSource).toContain("hasActiveProxyFilters ? t('proxies.noResults.description') : t('proxies.empty.description')")
    expect(tableSource).toContain('v-if="!hasActiveProxyFilters"')
    expect(tableSource).toContain('const emptyCreateButtonTitle = computed(() => buildCreateButtonTitle')
    const emptyCreateActionSource = tableSource.slice(
      tableSource.indexOf('v-if="!hasActiveProxyFilters"'),
      tableSource.indexOf('</button>', tableSource.indexOf('v-if="!hasActiveProxyFilters"')),
    )
    expect(emptyCreateActionSource).toContain(':disabled="testing"')
    expect(emptyCreateActionSource).toContain(':aria-label="emptyCreateButtonTitle"')
    expect(emptyCreateActionSource).toContain(':title="emptyCreateButtonTitle"')
    expect(emptyCreateActionSource).toContain("@click=\"emit('create')\"")
  })

  it('renders proxy modals through the shared dialog component with parent-owned open state', () => {
    expect(viewSource).toContain('<div class="contents">')
    expect(viewSource).toContain('<ProxyDialogs')
    expect(viewSource).toContain(':show-form="proxyFormDialogOpen"')
    expect(viewSource).toContain(':show-delete="proxyDeleteDialogOpen"')
    expect(viewSource).toContain(':form-locked="loading || testing"')
    expect(viewSource).toContain(':delete-locked="loading || testing"')
    expect(viewSource).toContain(':editing-proxy="editingProxy"')
    expect(viewSource).toContain(':proxy-to-delete="proxyToDelete"')
    expect(managementSource).toContain('proxyFormDialogOpen.value = true')
    expect(managementSource).toContain('proxyDeleteDialogOpen.value = true')
    expect(viewSource).not.toContain('proxyDialogsRef')
    expect(dialogsSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(dialogsSource).toContain('<BaseDialog :show="showForm"')
    expect(dialogsSource).toContain('<BaseDialog :show="showDelete"')
    expect(dialogsSource).toContain('id="proxy-create-dialog"')
    expect(dialogsSource).toContain('id="proxy-delete-dialog"')
    expect(dialogsSource).toContain('id="proxy-type"')
    expect(dialogsSource).toContain('formLocked?: boolean')
    expect(dialogsSource).toContain('deleteLocked?: boolean')
    expect(dialogsSource).toContain('const editFormLocked = computed(() => Boolean(props.editingProxy && props.formLocked))')
    expect(dialogsSource).toContain('const formInputsDisabled = computed(() => saving.value || editFormLocked.value)')
    expect(dialogsSource).toContain('const formSubmitDisabledReason = computed(() => {')
    expect(dialogsSource).toContain("if (editFormLocked.value) return t('common.processing')")
    expect(dialogsSource).toContain("if (form.name.trim() === '') return proxyErrorMessages.value.SOCIAL_IP_NAME_REQUIRED")
    expect(dialogsSource).toContain("if (form.ipType === null) return proxyErrorMessages.value.SOCIAL_IP_TYPE_INVALID")
    expect(dialogsSource).toContain("if (props.editingProxy && formMatchesProxy(formBaselineProxy)) return t('proxies.noChanges')")
    expect(dialogsSource).toContain('const canSubmitForm = computed(() => !formSubmitDisabledReason.value)')
    expect(dialogsSource).toContain("from './proxyActionTitles'")
    expect(dialogsSource).toContain('const formCancelButtonTitle = computed(() => buildFormCancelButtonTitle')
    expect(dialogsSource).toContain('const formSubmitButtonLabel = computed(() => buildFormSubmitButtonLabel')
    expect(dialogsSource).toContain('const formSubmitButtonTitle = computed')
    expect(dialogsSource).toContain('const deleteCancelButtonTitle = computed(() => buildDeleteCancelButtonTitle')
    expect(dialogsSource).toContain('const deleteConfirmButtonTitle = computed(() => buildDeleteConfirmButtonTitle')
    expect(dialogsSource).toContain(':aria-label="formCancelButtonTitle"')
    expect(dialogsSource).toContain(':title="formCancelButtonTitle"')
    expect(dialogsSource).toContain(':aria-label="formSubmitButtonLabel"')
    expect(dialogsSource).toContain(':title="formSubmitButtonTitle"')
    expect(dialogsSource).toContain(':aria-label="deleteCancelButtonTitle"')
    expect(dialogsSource).toContain(':title="deleteCancelButtonTitle"')
    expect(dialogsSource).toContain(':title="deleteConfirmButtonTitle"')
    expect(dialogsSource).toContain(':aria-label="deleteConfirmButtonTitle"')
    expect(proxyActionTitlesSource).toContain('export function proxyFormCancelButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyFormSubmitButtonLabel')
    expect(proxyActionTitlesSource).toContain('export function proxyFormSubmitButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyDeleteCancelButtonTitle')
    expect(proxyActionTitlesSource).toContain('export function proxyDeleteConfirmButtonTitle')
    expect(dialogsSource).toContain(':disabled="deleting || deleteLocked || !proxyToDelete"')
    expect(dialogsSource.match(/:disabled="formInputsDisabled"/g)).toHaveLength(4)
    expect(dialogsSource.match(/:disabled="saving"/g)).toHaveLength(1)
    expect(dialogsSource).toContain(':disabled="!canSubmitForm || saving"')
    expect(dialogsSource).not.toContain('<Select v-model="form.ipType"')
    expect(dialogsSource).not.toContain('defineExpose')
    expect(dialogsSource).not.toContain('modal-overlay')
    expect(viewSource).not.toContain('<Teleport to="body">')
    expect(dialogsSource).not.toContain("t('common.close', 'Close')")
  })

  it('escapes endpoint examples so vue-i18n does not parse @host as a linked message', () => {
    expect(enLocaleSource).not.toContain('user:pass@host')
    expect(zhLocaleSource).not.toContain('user:pass@host')
    expect(enLocaleSource).toContain("user:pass{'@'}host")
    expect(zhLocaleSource).toContain("user:pass{'@'}host")
  })
})
