import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewPath = resolve(testDir, '../ProxiesView.vue')
const dialogsPath = resolve(testDir, '../ProxyDialogs.vue')
const tablePath = resolve(testDir, '../components/ProxyTable.vue')
const managementPath = resolve(testDir, '../useProxyManagement.ts')
const enLocalePath = resolve(testDir, '../../../i18n/locales/en.ts')
const zhLocalePath = resolve(testDir, '../../../i18n/locales/zh.ts')
const viewSource = readFileSync(viewPath, 'utf8')
const dialogsSource = readFileSync(dialogsPath, 'utf8')
const tableSource = readFileSync(tablePath, 'utf8')
const managementSource = readFileSync(managementPath, 'utf8')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')

describe('ProxiesView source contract', () => {
  it('keeps ProxiesView as a thin orchestration layer', () => {
    expect(viewSource).toContain("import { useProxyManagement } from './useProxyManagement'")
    expect(viewSource).toContain("import ProxyStatsGrid from './components/ProxyStatsGrid.vue'")
    expect(viewSource).toContain("import ProxyToolbar from './components/ProxyToolbar.vue'")
    expect(viewSource).toContain("import ProxyTestResultsPanel from './components/ProxyTestResultsPanel.vue'")
    expect(viewSource).toContain("import ProxyTable from './components/ProxyTable.vue'")
    expect(viewSource).not.toContain("import proxiesAPI from '@/api/proxies'")
    expect(viewSource).not.toContain('Promise.allSettled')
    expect(viewSource).not.toContain('function buildFailedProxyCheckResult')
    expect(viewSource).not.toContain('const proxyListParams = computed')
  })

  it('uses backend proxy filters instead of searching only the first loaded page', () => {
    expect(managementSource).toContain('const proxyListParams = computed')
    expect(managementSource).toContain('api.list(proxyListParams.value)')
    expect(managementSource).toContain("params.search = search")
    expect(managementSource).toContain("params.status = status")
    expect(managementSource).toContain("params.ip_type = type as ProxyType")
    expect(managementSource).toContain('watch([searchQuery, statusFilter, typeFilter]')
    expect(viewSource).not.toContain('return proxies.value.filter((proxy) =>')
  })

  it('uses product-grade confirmation and visible connectivity result feedback', () => {
    expect(viewSource).toContain('<ProxyDialogs')
    expect(dialogsSource).toContain('showDelete')
    expect(dialogsSource).toContain('proxyToDelete')
    expect(dialogsSource).toContain('confirmDeleteProxy')
    expect(dialogsSource).toContain("t('proxies.deleteDialog.title')")
    expect(viewSource).not.toContain('window.confirm')
    expect(dialogsSource).not.toContain('window.confirm')

    expect(viewSource).toContain('<ProxyTestResultsPanel')
    expect(viewSource).toContain(':results="lastTestResults"')
    expect(viewSource).toContain(':summary="testResultSummary"')
    expect(viewSource).toContain(':preview-rows="testResultPreviewRows"')
  })

  it('preserves per-proxy feedback when selected connectivity checks partially fail', () => {
    expect(managementSource).toContain('Promise.allSettled')
    expect(managementSource).toContain('buildFailedProxyCheckResult')
    expect(managementSource).toContain("recordDiagnostic('proxies.test_selected_item'")
    expect(managementSource).toContain('recordProxyTestResults(results)')
  })

  it('distinguishes an empty proxy pool from zero matches after filters are applied', () => {
    expect(viewSource).toContain('hasActiveProxyFilters')
    expect(tableSource).toContain("hasActiveProxyFilters ? t('proxies.noResults.title') : t('proxies.empty.title')")
    expect(tableSource).toContain("hasActiveProxyFilters ? t('proxies.noResults.description') : t('proxies.empty.description')")
    expect(tableSource).toContain('v-if="!hasActiveProxyFilters"')
  })

  it('renders proxy modals through the shared dialog component with parent-owned open state', () => {
    expect(viewSource).toContain('<div class="contents">')
    expect(viewSource).toContain('<ProxyDialogs')
    expect(viewSource).toContain(':show-form="proxyFormDialogOpen"')
    expect(viewSource).toContain(':show-delete="proxyDeleteDialogOpen"')
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
