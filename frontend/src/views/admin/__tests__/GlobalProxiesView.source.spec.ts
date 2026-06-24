import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewPath = resolve(testDir, '../GlobalProxiesView.vue')
const apiPath = resolve(testDir, '../../../api/admin/globalProxies.ts')
const routerPath = resolve(testDir, '../../../router/index.ts')
const sidebarPath = resolve(testDir, '../../../components/layout/AppSidebar.vue')
const zhLocalePath = resolve(testDir, '../../../i18n/locales/zh.ts')
const enLocalePath = resolve(testDir, '../../../i18n/locales/en.ts')

const viewSource = readFileSync(viewPath, 'utf8')
const apiSource = readFileSync(apiPath, 'utf8')
const routerSource = readFileSync(routerPath, 'utf8')
const sidebarSource = readFileSync(sidebarPath, 'utf8')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')

describe('GlobalProxiesView source contract', () => {
  it('uses the dedicated global proxy API instead of old admin proxy routes', () => {
    expect(apiSource).toContain("const BASE = '/admin/global-proxies'")
    expect(viewSource).toContain("import adminGlobalProxiesAPI")
    expect(viewSource).toContain('adminGlobalProxiesAPI.list')
    expect(viewSource).toContain('adminGlobalProxiesAPI.create')
    expect(viewSource).toContain('adminGlobalProxiesAPI.update')
    expect(viewSource).toContain('adminGlobalProxiesAPI.delete')
    expect(viewSource).toContain('adminGlobalProxiesAPI.test')
    expect(viewSource).toContain('adminGlobalProxiesAPI.testAll')
    expect(apiSource).not.toContain('/admin/login-proxies')
    expect(viewSource).not.toContain('/admin/login-proxies')
    expect(apiSource).not.toContain('/admin/proxies')
    expect(viewSource).not.toContain('/admin/proxies')
  })

  it('registers the admin route and sidebar entry under the account center', () => {
    expect(routerSource).toContain("path: '/admin/global-proxies'")
    expect(routerSource).toContain("name: 'AdminGlobalProxies'")
    expect(routerSource).toContain("component: () => import('@/views/admin/GlobalProxiesView.vue')")
    expect(sidebarSource).toContain("{ path: '/admin/global-proxies', label: t('nav.globalProxies'), icon: ServerIcon }")
    expect(zhLocaleSource).toContain('"globalProxies": "全局代理池"')
    expect(enLocaleSource).toContain('"globalProxies": "Global Proxy Pool"')
  })

  it('keeps list management behavior aligned with existing proxy workflows', () => {
    expect(viewSource).toContain('<SearchInput v-model="searchQuery"')
    expect(viewSource).toContain('v-model="statusFilter"')
    expect(viewSource).toContain('v-model="typeFilter"')
    expect(viewSource).toContain('@click="testSelected"')
    expect(viewSource).toContain('@click="testAll"')
    expect(viewSource).toContain('@click="openCreateDialog"')
    expect(viewSource).toContain('<BaseDialog :show="deleteOpen"')
    expect(viewSource).toContain("t('admin.globalProxies.deleteDialog.impact')")
    expect(viewSource).toContain('Promise.allSettled')
    expect(viewSource).toContain('showProxyTestSummaryToast')
    expect(viewSource).toContain("t('proxies.testResultSummary'")
    expect(viewSource).not.toContain('lastTestResults')
    expect(viewSource).not.toContain("t('proxies.testResults")
  })

  it('closes successful save and delete dialogs even while operation locks are active', () => {
    expect(viewSource).toContain('function requestCloseForm()')
    expect(viewSource).toContain('if (saving.value) return')
    expect(viewSource).toContain('function closeFormDialog()')
    expect(viewSource).toContain('closeFormDialog()')
    expect(viewSource).toContain('function requestCloseDelete()')
    expect(viewSource).toContain('if (deleting.value) return')
    expect(viewSource).toContain('function closeDeleteDialog()')
    expect(viewSource).toContain('closeDeleteDialog()')
  })

  it('documents static proxy URLs and dynamic extraction URLs without adding user ownership', () => {
    expect(viewSource).toContain("t('admin.globalProxies.form.endpointHint')")
    expect(zhLocaleSource).toContain('可填写静态代理地址，或填写返回 ip/port/username/password 的提取链接')
    expect(enLocaleSource).toContain('Enter a static proxy URL, or an extraction URL')
    expect(viewSource).toContain('<option value="dynamic">{{ t(\'proxies.types.dynamic\') }}</option>')
    expect(apiSource).not.toContain('user_id')
    expect(viewSource).not.toContain('user_id')
  })
})
