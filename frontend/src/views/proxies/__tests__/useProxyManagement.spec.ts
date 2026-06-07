import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import type { ProxyCheckResult, UserProxy } from '@/api/proxies'
import { useProxyManagement } from '../useProxyManagement'

function createProxy(overrides: Partial<UserProxy> = {}): UserProxy {
  return {
    id: overrides.id ?? 1,
    user_id: overrides.user_id ?? 10,
    name: overrides.name ?? 'Main proxy',
    ip_type: overrides.ip_type ?? 'residential',
    endpoint: overrides.endpoint ?? 'http://proxy.example.com:8080',
    status: overrides.status ?? 'unknown',
    latency_ms: overrides.latency_ms ?? null,
    last_check_at: overrides.last_check_at ?? null,
    remark: overrides.remark ?? null,
    created_at: overrides.created_at ?? '2026-06-06T00:00:00Z',
    updated_at: overrides.updated_at ?? '2026-06-06T01:00:00Z',
  }
}

function setupManagement() {
  const api = {
    list: vi.fn(),
    test: vi.fn(),
    testAll: vi.fn(),
  }
  const notifier = {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }
  const recordDiagnostic = vi.fn()
  const t = vi.fn((key: string, params?: Record<string, unknown>) => {
    if (key === 'proxies.batchTestPartial') return `partial ${params?.failed}/${params?.total}`
    if (key === 'proxies.batchTestSubmitted') return `submitted ${params?.count}`
    if (key === 'proxies.testResult') return `tested ${params?.status}`
    return key
  })

  const management = useProxyManagement({
    api,
    notifier,
    recordDiagnostic,
    t,
  })

  return { api, notifier, recordDiagnostic, t, management }
}

describe('useProxyManagement', () => {
  it('loads proxies through backend filters and prunes stale selections', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    management.selectedIds.value = [7, 99]
    management.searchQuery.value = 'tokyo'
    management.statusFilter.value = 'online'
    management.typeFilter.value = 'residential'
    await management.loadProxies()

    expect(api.list).toHaveBeenCalledWith({
      page: 1,
      page_size: 200,
      search: 'tokyo',
      status: 'online',
      ip_type: 'residential',
    })
    expect(management.proxies.value).toHaveLength(1)
    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      name: 'Tokyo proxy',
      type: 'residential',
      status: 'online',
      endpoint: 'http://proxy.example.com:8080',
    })
    expect(management.selectedIds.value).toEqual([7])
  })

  it('keeps row-level failed results when selected proxy tests partially reject', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [
        createProxy({ id: 1, name: 'Proxy one' }),
        createProxy({ id: 2, name: 'Proxy two' }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    const okResult: ProxyCheckResult = { id: 1, status: 'online', latency_ms: 42 }
    api.test.mockImplementation((id: number) => {
      if (id === 1) return Promise.resolve(okResult)
      return Promise.reject(new Error('dial failed'))
    })

    await management.loadProxies()
    management.selectedIds.value = [1, 2]
    await management.testSelected()
    await flushPromises()

    expect(management.lastTestResults.value).toEqual([
      okResult,
      { id: 2, status: 'unknown', latency_ms: 0, error: 'dial failed' },
    ])
    expect(notifier.showWarning).toHaveBeenCalledWith('partial 1/2')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_selected_item', expect.any(Error))
  })

  it('keeps dialog state parent-owned and resets targets on close', () => {
    const { management } = setupManagement()
    const row = {
      id: 3,
      name: 'Editable proxy',
      type: 'mobile' as const,
      endpoint: 'http://editable.example.com:8080',
      status: 'unknown',
      latency: null,
      lastCheck: '-',
      remark: 'remark',
      updatedAt: '2026-06-06T01:00:00Z',
    }

    management.openEditDialog(row)
    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value).toEqual(row)

    management.closeProxyFormDialog()
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()

    management.openDeleteDialog(row)
    expect(management.proxyDeleteDialogOpen.value).toBe(true)
    expect(management.proxyToDelete.value).toEqual(row)

    management.closeProxyDeleteDialog()
    expect(management.proxyDeleteDialogOpen.value).toBe(false)
    expect(management.proxyToDelete.value).toBeNull()
  })
})
