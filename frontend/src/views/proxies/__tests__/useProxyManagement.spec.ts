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
    if (key === 'proxies.batchTestPartial') return `partial ${params?.failed}/${params?.total}/${params?.online}/${params?.offline}/${params?.unknown}`
    if (key === 'proxies.testResultSummary') return `summary ${params?.total}/${params?.online}/${params?.offline}/${params?.unknown}`
    if (key === 'proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE') return 'Proxy service is temporarily unavailable.'
    if (key === 'proxies.errors.SOCIAL_IP_NOT_FOUND') return 'Proxy not found.'
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

  it('omits invalid proxy status and type filters from backend requests', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    management.statusFilter.value = 'running'
    management.typeFilter.value = 'serverless'
    await management.loadProxies()

    expect(api.list).toHaveBeenCalledWith({
      page: 1,
      page_size: 200,
    })
    expect(management.hasActiveProxyFilters.value).toBe(false)
    expect(management.proxies.value.map(proxy => proxy.id)).toEqual([7])
  })

  it('normalizes proxy type filters, rows, and labels to existing proxy types', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', ip_type: ' RESIDENTIAL ' as UserProxy['ip_type'] })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    management.typeFilter.value = ' RESIDENTIAL '
    await management.loadProxies()

    expect(api.list).toHaveBeenCalledWith({
      page: 1,
      page_size: 200,
      ip_type: 'residential',
    })
    expect(management.hasActiveProxyFilters.value).toBe(true)
    expect(management.proxies.value[0]).toMatchObject({ id: 7, type: 'residential' })
    expect(management.proxyTypeLabel(' RESIDENTIAL ')).toBe('proxies.types.residential')
    expect(management.proxyTypeLabel(' vendor_custom ')).toBe('vendor_custom')
  })

  it('trims proxy endpoints before table display, searching, and editing use them', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', endpoint: '  http://proxy.example.com:8080  ' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    management.searchQuery.value = 'proxy.example.com:8080'
    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      endpoint: 'http://proxy.example.com:8080',
    })
    management.openEditDialog(management.proxies.value[0])
    expect(management.editingProxy.value?.endpoint).toBe('http://proxy.example.com:8080')
  })

  it('trims proxy names before table display, searching, and editing use them', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: '  Tokyo proxy  ', endpoint: 'http://proxy.example.com:8080' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    management.searchQuery.value = 'Tokyo proxy'
    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      name: 'Tokyo proxy',
    })
    management.openEditDialog(management.proxies.value[0])
    expect(management.editingProxy.value?.name).toBe('Tokyo proxy')
    expect(JSON.stringify(management.proxies.value)).not.toContain('  Tokyo proxy  ')
  })

  it('trims proxy remarks before editing and local filter matching use them', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', remark: '  primary note  ' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    management.searchQuery.value = 'primary note'
    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      remark: 'primary note',
    })
    management.openEditDialog(management.proxies.value[0])
    expect(management.editingProxy.value?.remark).toBe('primary note')
    expect(JSON.stringify(management.proxies.value)).not.toContain('  primary note  ')

    const savePromise = management.handleProxySaved(createProxy({ id: 7, name: 'Tokyo proxy', remark: '  primary note  ' }))
    await flushPromises()
    expect(management.proxies.value[0]?.remark).toBe('primary note')

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', remark: 'primary note' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await savePromise
  })

  it('hides invalid proxy last-check timestamps instead of showing Invalid Date', async () => {
    const { api, management } = setupManagement()
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', last_check_at: 'not-a-date' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })

    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      lastCheck: '-',
    })
    expect(JSON.stringify(management.proxies.value)).not.toContain('Invalid Date')
  })

  it('keeps test-all available when filters hide an existing proxy pool', async () => {
    const { api, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    await management.loadProxies()
    expect(management.hasAnyProxy.value).toBe(true)

    management.searchQuery.value = 'no-match'
    await management.loadProxies()

    expect(management.proxies.value).toEqual([])
    expect(management.hasActiveProxyFilters.value).toBe(true)
    expect(management.hasAnyProxy.value).toBe(true)

    management.searchQuery.value = ''
    await management.loadProxies()

    expect(management.hasActiveProxyFilters.value).toBe(false)
    expect(management.hasAnyProxy.value).toBe(false)
  })

  it('does not submit test-all when no proxy pool is known yet', async () => {
    const { api, notifier, management } = setupManagement()

    await management.testAll()

    expect(api.testAll).not.toHaveBeenCalled()
    expect(management.testing.value).toBe(false)
    expect(notifier.showError).not.toHaveBeenCalledWith('proxies.testFailed')
    expect(notifier.showSuccess).not.toHaveBeenCalled()
  })

  it('reports test-all summaries even when active filters hide the tested rows', async () => {
    const { api, notifier, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    api.testAll.mockResolvedValue([{ id: 7, status: 'online', latency_ms: 37 }])

    await management.loadProxies()
    management.searchQuery.value = 'no-match'
    await management.loadProxies()
    expect(management.proxies.value).toEqual([])

    await management.testAll()

    expect(management.proxies.value).toEqual([])
    expect(management.hasActiveProxyFilters.value).toBe(true)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/1/0/0')
  })

  it('removes test-all rows from the current filtered view when their status no longer matches', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 1, name: 'Proxy one', status: 'online' }),
          createProxy({ id: 2, name: 'Proxy two', status: 'online' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.testAll.mockResolvedValue([
      { id: 1, status: 'offline', latency_ms: 0 },
      { id: 2, status: 'online', latency_ms: 42 },
    ])

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [1, 2]

    const testPromise = management.testAll()
    await flushPromises()

    expect(management.proxies.value).toEqual([
      expect.objectContaining({ id: 2, status: 'online', latency: 42 }),
    ])
    expect(management.selectedIds.value).toEqual([2])
    expect(management.hasAnyProxy.value).toBe(true)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 2/1/1/0')

    resolveRefresh?.({
      items: [createProxy({ id: 2, name: 'Proxy two', status: 'online', latency_ms: 42 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('keeps test-all filtered rows stable when the follow-up refresh fails', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 1, name: 'Proxy one', status: 'online' }),
          createProxy({ id: 2, name: 'Proxy two', status: 'online' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))
    api.testAll.mockResolvedValue([
      { id: 1, status: 'offline', latency_ms: 0 },
      { id: 2, status: 'online', latency_ms: 42 },
    ])

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [1, 2]

    await management.testAll()

    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.proxies.value).toEqual([
      expect.objectContaining({ id: 2, status: 'online', latency: 42 }),
    ])
    expect(management.selectedIds.value).toEqual([2])
    expect(management.hasAnyProxy.value).toBe(true)
    expect(management.loadError.value).toBe('proxies.failedToLoad')
    expect(management.testing.value).toBe(false)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 2/1/1/0')
    expect(notifier.showError).toHaveBeenCalledWith('proxies.failedToLoad')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.load', expect.any(Error))
  })

  it('reports test-all failures without opening result details', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.testAll.mockRejectedValue(new Error('test all failed'))
    management.hasAnyProxy.value = true

    await management.testAll()

    expect(api.testAll).toHaveBeenCalledTimes(1)
    expect(management.testing.value).toBe(false)
    expect(notifier.showError).toHaveBeenCalledWith('proxies.testFailed')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_all', expect.any(Error))
  })

  it('does not report an empty test-all response as a successful zero-count submission', async () => {
    const { api, notifier, management } = setupManagement()
    api.testAll.mockResolvedValue([])
    management.hasAnyProxy.value = true
    api.list.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 0,
    })

    await management.testAll()

    expect(notifier.showError).toHaveBeenCalledWith('proxies.testFailed')
    expect(notifier.showSuccess).not.toHaveBeenCalled()
  })

  it('maps proxy service availability load and test errors to friendly messages', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    const unavailable = { code: 'SOCIAL_IP_SERVICE_UNAVAILABLE', message: 'social IP service is unavailable' }
    api.list.mockRejectedValueOnce(unavailable)
    api.testAll.mockRejectedValueOnce(unavailable)

    await management.loadProxies()
    management.hasAnyProxy.value = true
    await management.testAll()

    expect(management.loadError.value).toBe('Proxy service is temporarily unavailable.')
    expect(notifier.showError).toHaveBeenCalledWith('Proxy service is temporarily unavailable.')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.load', unavailable)
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_all', unavailable)
  })

  it('ignores stale list responses when a newer proxy load finishes first', async () => {
    const { api, management } = setupManagement()
    let resolveFirst: ((value: unknown) => void) | undefined
    let resolveSecond: ((value: unknown) => void) | undefined
    api.list
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveFirst = resolve
      }))
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveSecond = resolve
      }))

    management.selectedIds.value = [1, 2]
    const firstLoad = management.loadProxies()
    const secondLoad = management.loadProxies()

    resolveSecond?.({
      items: [createProxy({ id: 2, name: 'Fresh proxy', status: 'online' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await secondLoad

    expect(management.loading.value).toBe(false)
    expect(management.proxies.value).toHaveLength(1)
    expect(management.proxies.value[0]).toMatchObject({ id: 2, name: 'Fresh proxy' })
    expect(management.selectedIds.value).toEqual([2])

    resolveFirst?.({
      items: [createProxy({ id: 1, name: 'Stale proxy', status: 'offline' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await firstLoad

    expect(management.loading.value).toBe(false)
    expect(management.proxies.value).toHaveLength(1)
    expect(management.proxies.value[0]).toMatchObject({ id: 2, name: 'Fresh proxy' })
    expect(management.selectedIds.value).toEqual([2])
  })

  it('reports selected proxy tests with a partial-failure summary when one request rejects', async () => {
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

    expect(notifier.showWarning).toHaveBeenCalledWith('partial 1/2/1/0/1')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_selected_item', expect.any(Error))
  })

  it('reports selected proxy test service availability failures without result cards', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    const unavailable = { code: 'SOCIAL_IP_SERVICE_UNAVAILABLE', message: 'social IP service is unavailable' }
    api.list.mockResolvedValue({
      items: [createProxy({ id: 2, name: 'Proxy two' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    api.test.mockRejectedValue(unavailable)

    await management.loadProxies()
    management.selectedIds.value = [2]
    await management.testSelected()

    expect(notifier.showError).toHaveBeenCalledWith('proxies.testFailed')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_selected_item', unavailable)
  })

  it('reports selected proxy tests as failed when every request rejects', async () => {
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
    api.test.mockRejectedValue(new Error('dial failed token=secret'))

    await management.loadProxies()
    management.selectedIds.value = [1, 2]
    await management.testSelected()
    await flushPromises()

    expect(notifier.showError).toHaveBeenCalledWith('proxies.testFailed')
    expect(notifier.showWarning).not.toHaveBeenCalled()
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_selected_item', expect.any(Error))
  })

  it('keeps selected connectivity checks locked while a request is in flight', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveTest!: (value: ProxyCheckResult) => void
    api.test.mockReturnValue(new Promise((resolve) => {
      resolveTest = resolve
    }))
    api.list.mockResolvedValue({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: 37 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    management.selectedIds.value = [7]

    const testPromise = management.testSelected()
    await flushPromises()

    expect(api.test).toHaveBeenCalledWith(7)
    expect(management.testing.value).toBe(true)
    expect(notifier.showSuccess).not.toHaveBeenCalled()

    resolveTest({ id: 7, status: 'online', latency_ms: 37 })
    await testPromise

    expect(management.testing.value).toBe(false)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/1/0/0')
  })

  it('syncs a single proxy test result to its row before the next list refresh finishes', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown', latency_ms: null })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockResolvedValue({ id: 7, status: 'online', latency_ms: 37 })

    await management.loadProxies()
    const testPromise = management.testProxy(7)
    await flushPromises()

    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      status: 'online',
      latency: 37,
    })
    expect(management.proxies.value[0].lastCheck).not.toBe('-')
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/1/0/0')

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: 37 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('reports single proxy test failures without result cards', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.test.mockRejectedValue(new Error('single proxy test failed'))

    await management.testProxy(7)

    expect(api.test).toHaveBeenCalledWith(7)
    expect(management.testing.value).toBe(false)
    expect(notifier.showError).toHaveBeenCalledWith('proxies.testFailed')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test', expect.any(Error))
  })

  it('removes a tested proxy from the current filtered view when its status no longer matches', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockResolvedValue({ id: 7, status: 'offline', latency_ms: 0 })

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [7]

    const testPromise = management.testProxy(7)
    await flushPromises()

    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
    expect(management.hasAnyProxy.value).toBe(true)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/0/1/0')

    resolveRefresh?.({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('normalizes list and connectivity result statuses to the existing proxy states', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: ' ONLINE ' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockResolvedValue({ id: 7, status: ' OFFLINE ', latency_ms: 0 })

    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({ id: 7, status: 'online' })
    expect(management.stats.value.find(stat => stat.label === 'proxies.stats.online')?.value).toBe(1)

    const testPromise = management.testProxy(7)
    await flushPromises()

    expect(management.proxies.value[0]).toMatchObject({ id: 7, status: 'offline', latency: null })
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/0/1/0')

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'offline', latency_ms: 0 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('normalizes non-positive proxy latencies from list and connectivity results', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown', latency_ms: -7 })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockResolvedValue({ id: 7, status: 'online', latency_ms: -4 })

    await management.loadProxies()

    expect(management.proxies.value[0]).toMatchObject({ id: 7, latency: null })

    const testPromise = management.testProxy(7)
    await flushPromises()

    expect(management.proxies.value[0]).toMatchObject({ id: 7, status: 'online', latency: null })
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/1/0/0')

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: -4 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise

    expect(management.proxies.value[0]).toMatchObject({ id: 7, latency: null })
  })

  it('refreshes the list when a single proxy test reports a stale proxy id', async () => {
    const { api, management, notifier, recordDiagnostic } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    api.test.mockRejectedValue({ code: 'SOCIAL_IP_NOT_FOUND', message: 'raw not found' })

    await management.loadProxies()
    await management.testProxy(7)

    expect(api.test).toHaveBeenCalledWith(7)
    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.proxies.value).toEqual([])
    expect(notifier.showError).toHaveBeenCalledWith('Proxy not found.')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test', expect.objectContaining({ code: 'SOCIAL_IP_NOT_FOUND' }))
  })

  it('removes filtered proxy state when a single test reports a missing proxy', async () => {
    const { api, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
    api.test.mockRejectedValue({ code: 'SOCIAL_IP_NOT_FOUND', message: 'raw not found' })

    management.searchQuery.value = 'Tokyo'
    await management.loadProxies()
    management.selectedIds.value = [7]

    await management.testProxy(7)

    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.hasActiveProxyFilters.value).toBe(true)
    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
  })

  it('ignores duplicate single proxy tests while a connectivity check is already running', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown' }),
          createProxy({ id: 8, name: 'Backup proxy', status: 'unknown' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockResolvedValue({ id: 7, status: 'online', latency_ms: 37 })

    await management.loadProxies()
    const firstTest = management.testProxy(7)
    await flushPromises()
    await management.testProxy(8)

    expect(api.test).toHaveBeenCalledTimes(1)
    expect(api.test).toHaveBeenCalledWith(7)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 1/1/0/0')

    resolveRefresh?.({
      items: [
        createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: 37 }),
        createProxy({ id: 8, name: 'Backup proxy', status: 'unknown' }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await firstTest
  })

  it('does not start connectivity checks while the proxy list is refreshing', async () => {
    const { api, management } = setupManagement()
    let resolveLoad: ((value: unknown) => void) | undefined
    api.list.mockImplementationOnce(() => new Promise(resolve => {
      resolveLoad = resolve
    }))

    const loadPromise = management.loadProxies()
    await flushPromises()
    expect(management.loading.value).toBe(true)

    management.selectedIds.value = [7]
    await management.testProxy(7)
    await management.testSelected()
    await management.testAll()

    expect(api.test).not.toHaveBeenCalled()
    expect(api.testAll).not.toHaveBeenCalled()

    resolveLoad?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await loadPromise
  })

  it('ignores stale row and selection actions while the proxy list is refreshing', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' }),
          createProxy({ id: 8, name: 'Backup proxy', status: 'unknown' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    const staleRow = management.proxies.value[0]
    management.selectedIds.value = [8]

    const refreshPromise = management.loadProxies()
    await flushPromises()
    expect(management.loading.value).toBe(true)

    management.toggleSelection(7)
    management.toggleAllVisible()
    management.openEditDialog(staleRow)
    management.openDeleteDialog(staleRow)

    expect(management.selectedIds.value).toEqual([8])
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()
    expect(management.proxyDeleteDialogOpen.value).toBe(false)
    expect(management.proxyToDelete.value).toBeNull()

    resolveRefresh?.({
      items: [
        createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' }),
        createProxy({ id: 8, name: 'Backup proxy', status: 'unknown' }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await refreshPromise

    management.toggleSelection(7)
    management.openEditDialog(management.proxies.value[0])

    expect(management.selectedIds.value).toEqual([8, 7])
    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value?.id).toBe(7)
  })

  it('ignores stale create actions while connectivity checks are running but keeps refresh-time creation available', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    let resolveCreateTimeRefresh: ((value: unknown) => void) | undefined
    let resolveTest: ((value: ProxyCheckResult) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveCreateTimeRefresh = resolve
      }))
    api.test.mockImplementation(() => new Promise<ProxyCheckResult>(resolve => {
      resolveTest = resolve
    }))

    await management.loadProxies()
    const testPromise = management.testProxy(7)
    await flushPromises()

    expect(management.testing.value).toBe(true)
    management.openCreateDialog()
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()

    resolveTest?.({ id: 7, status: 'online', latency_ms: 37 })
    await flushPromises()
    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: 37 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise

    expect(management.testing.value).toBe(false)

    const refreshPromise = management.loadProxies()
    await flushPromises()
    expect(management.loading.value).toBe(true)
    management.openCreateDialog()
    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value).toBeNull()

    resolveCreateTimeRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online', latency_ms: 37 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await refreshPromise
  })

  it('syncs selected proxy test results to visible rows before the next list refresh finishes', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 1, name: 'Proxy one', status: 'unknown' }),
          createProxy({ id: 2, name: 'Proxy two', status: 'unknown' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockImplementation((id: number) => {
      if (id === 1) return Promise.resolve({ id: 1, status: 'online', latency_ms: 31 })
      return Promise.reject(new Error('dial failed'))
    })

    await management.loadProxies()
    management.selectedIds.value = [1, 2]
    const testPromise = management.testSelected()
    await flushPromises()

    expect(management.proxies.value).toEqual([
      expect.objectContaining({ id: 1, status: 'online', latency: 31 }),
      expect.objectContaining({ id: 2, status: 'unknown', latency: null }),
    ])
    expect(notifier.showWarning).toHaveBeenCalledWith('partial 1/2/1/0/1')

    resolveRefresh?.({
      items: [
        createProxy({ id: 1, name: 'Proxy one', status: 'online', latency_ms: 31 }),
        createProxy({ id: 2, name: 'Proxy two', status: 'unknown', latency_ms: null }),
      ],
      total: 2,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('removes selected tested proxies from the current filtered view when their status no longer matches', async () => {
    const { api, notifier, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 1, name: 'Proxy one', status: 'online' }),
          createProxy({ id: 2, name: 'Proxy two', status: 'online' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))
    api.test.mockImplementation((id: number) => {
      if (id === 1) return Promise.resolve({ id: 1, status: 'offline', latency_ms: 0 })
      return Promise.resolve({ id: 2, status: 'online', latency_ms: 42 })
    })

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [1, 2]

    const testPromise = management.testSelected()
    await flushPromises()

    expect(management.proxies.value).toEqual([
      expect.objectContaining({ id: 2, status: 'online', latency: 42 }),
    ])
    expect(management.selectedIds.value).toEqual([2])
    expect(management.hasAnyProxy.value).toBe(true)
    expect(notifier.showSuccess).toHaveBeenCalledWith('summary 2/1/1/0')

    resolveRefresh?.({
      items: [createProxy({ id: 2, name: 'Proxy two', status: 'online', latency_ms: 42 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await testPromise
  })

  it('keeps selected-test filtered rows stable when the follow-up refresh fails', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 1, name: 'Proxy one', status: 'online' }),
          createProxy({ id: 2, name: 'Proxy two', status: 'online' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))
    api.test.mockImplementation((id: number) => {
      if (id === 1) return Promise.resolve({ id: 1, status: 'offline', latency_ms: 0 })
      return Promise.reject(new Error('dial failed'))
    })

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [1, 2]

    await management.testSelected()

    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
    expect(management.hasAnyProxy.value).toBe(true)
    expect(management.loadError.value).toBe('proxies.failedToLoad')
    expect(management.testing.value).toBe(false)
    expect(notifier.showWarning).toHaveBeenCalledWith('partial 1/2/0/1/1')
    expect(notifier.showError).toHaveBeenCalledWith('proxies.failedToLoad')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.test_selected_item', expect.any(Error))
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.load', expect.any(Error))
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

  it('closes stale edit and delete dialogs when refreshed proxies disappear', async () => {
    const { api, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    await management.loadProxies()
    const row = management.proxies.value[0]
    management.openEditDialog(row)
    management.openDeleteDialog(row)
    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value?.id).toBe(7)
    expect(management.proxyDeleteDialogOpen.value).toBe(true)
    expect(management.proxyToDelete.value?.id).toBe(7)

    await management.loadProxies()

    expect(management.proxies.value).toEqual([])
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()
    expect(management.proxyDeleteDialogOpen.value).toBe(false)
    expect(management.proxyToDelete.value).toBeNull()
  })

  it('refreshes the delete confirmation target when the proxy remains in the list', async () => {
    const { api, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', endpoint: 'http://old-proxy.example.com:8080' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy updated', endpoint: 'http://new-proxy.example.com:8080' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    await management.loadProxies()
    management.openDeleteDialog(management.proxies.value[0])
    expect(management.proxyDeleteDialogOpen.value).toBe(true)
    expect(management.proxyToDelete.value).toMatchObject({
      id: 7,
      name: 'Tokyo proxy',
      endpoint: 'http://old-proxy.example.com:8080',
    })

    await management.loadProxies()

    expect(management.proxyDeleteDialogOpen.value).toBe(true)
    expect(management.proxyToDelete.value).toMatchObject({
      id: 7,
      name: 'Tokyo proxy updated',
      endpoint: 'http://new-proxy.example.com:8080',
    })
  })

  it('refreshes the pristine edit target when the proxy remains in the list', async () => {
    const { api, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', endpoint: 'http://old-proxy.example.com:8080', remark: 'old note' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy updated', endpoint: 'http://new-proxy.example.com:8080', remark: 'new note' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    await management.loadProxies()
    management.openEditDialog(management.proxies.value[0])
    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value).toMatchObject({
      id: 7,
      name: 'Tokyo proxy',
      endpoint: 'http://old-proxy.example.com:8080',
      remark: 'old note',
    })

    await management.loadProxies()

    expect(management.proxyFormDialogOpen.value).toBe(true)
    expect(management.editingProxy.value).toMatchObject({
      id: 7,
      name: 'Tokyo proxy updated',
      endpoint: 'http://new-proxy.example.com:8080',
      remark: 'new note',
    })
  })

  it('syncs an edited proxy locally before the next list refresh finishes', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'unknown' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    management.openEditDialog(management.proxies.value[0])

    const savePromise = management.handleProxySaved(createProxy({
      id: 7,
      name: 'Tokyo proxy updated',
      status: 'online',
      latency_ms: 42,
      last_check_at: '2026-06-06T02:00:00Z',
    }))
    await flushPromises()

    expect(management.proxies.value).toHaveLength(1)
    expect(management.proxies.value[0]).toMatchObject({
      id: 7,
      name: 'Tokyo proxy updated',
      status: 'online',
      latency: 42,
    })
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()
    expect(management.stats.value.find(stat => stat.label === 'proxies.stats.online')?.value).toBe(1)

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy updated', status: 'online', latency_ms: 42 })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await savePromise
  })

  it('adds a newly saved proxy locally before the next list refresh finishes', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    expect(management.proxies.value).toEqual([])
    expect(management.hasAnyProxy.value).toBe(false)

    const createdProxy = createProxy({ id: 9, name: 'Singapore proxy', status: 'unknown' })
    const savePromise = management.handleProxySaved(createdProxy)
    await flushPromises()

    expect(management.proxies.value.map(proxy => proxy.id)).toEqual([9])
    expect(management.proxies.value[0]).toMatchObject({ name: 'Singapore proxy', status: 'unknown' })
    expect(management.hasAnyProxy.value).toBe(true)

    resolveRefresh?.({
      items: [createdProxy],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await savePromise
  })

  it('removes an edited proxy from the current filtered view when it no longer matches', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [7]

    const savePromise = management.handleProxySaved(createProxy({ id: 7, name: 'Tokyo proxy', status: 'offline' }))
    await flushPromises()

    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
    expect(management.hasAnyProxy.value).toBe(true)

    resolveRefresh?.({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await savePromise
  })

  it('keeps an edited proxy out of the current filtered view when the follow-up refresh fails', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))

    management.statusFilter.value = 'online'
    await management.loadProxies()
    management.selectedIds.value = [7]
    management.openEditDialog(management.proxies.value[0])

    await management.handleProxySaved(createProxy({ id: 7, name: 'Tokyo proxy updated', status: 'offline' }))

    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
    expect(management.proxyFormDialogOpen.value).toBe(false)
    expect(management.editingProxy.value).toBeNull()
    expect(management.hasAnyProxy.value).toBe(true)
    expect(management.loadError.value).toBe('proxies.failedToLoad')
    expect(notifier.showError).toHaveBeenCalledWith('proxies.failedToLoad')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.load', expect.any(Error))
  })

  it('keeps a saved proxy visible when stale filter values are invalid', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    management.statusFilter.value = 'running'
    management.typeFilter.value = 'serverless'

    const savePromise = management.handleProxySaved(createProxy({ id: 7, name: 'Tokyo proxy updated', status: 'online' }))
    await flushPromises()

    expect(management.proxies.value).toEqual([
      expect.objectContaining({ id: 7, name: 'Tokyo proxy updated', status: 'online' }),
    ])
    expect(management.hasActiveProxyFilters.value).toBe(false)

    resolveRefresh?.({
      items: [createProxy({ id: 7, name: 'Tokyo proxy updated', status: 'online' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await savePromise
  })

  it('removes a deleted proxy from local state before the next list refresh finishes', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [
          createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' }),
          createProxy({ id: 8, name: 'Backup proxy', status: 'offline' }),
        ],
        total: 2,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    management.selectedIds.value = [7, 8]
    management.openDeleteDialog(management.proxies.value[0])

    const deletePromise = management.handleProxyDeleted(7)
    await flushPromises()

    expect(management.proxies.value.map(proxy => proxy.id)).toEqual([8])
    expect(management.selectedIds.value).toEqual([8])
    expect(management.proxyDeleteDialogOpen.value).toBe(false)
    expect(management.proxyToDelete.value).toBeNull()
    expect(management.stats.value.find(stat => stat.label === 'proxies.stats.total')?.value).toBe(1)
    expect(management.hasAnyProxy.value).toBe(true)

    resolveRefresh?.({
      items: [createProxy({ id: 8, name: 'Backup proxy', status: 'offline' })],
      total: 1,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await deletePromise
  })

  it('keeps deleted proxy state removed when the follow-up refresh fails', async () => {
    const { api, notifier, recordDiagnostic, management } = setupManagement()
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy', status: 'online' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockRejectedValueOnce(new Error('follow-up refresh failed'))

    await management.loadProxies()
    management.selectedIds.value = [7]
    management.openDeleteDialog(management.proxies.value[0])

    await management.handleProxyDeleted(7)

    expect(api.list).toHaveBeenCalledTimes(2)
    expect(management.proxies.value).toEqual([])
    expect(management.selectedIds.value).toEqual([])
    expect(management.proxyDeleteDialogOpen.value).toBe(false)
    expect(management.proxyToDelete.value).toBeNull()
    expect(management.hasAnyProxy.value).toBe(false)
    expect(management.loadError.value).toBe('proxies.failedToLoad')
    expect(notifier.showError).toHaveBeenCalledWith('proxies.failedToLoad')
    expect(recordDiagnostic).toHaveBeenCalledWith('proxies.load', expect.any(Error))
  })

  it('marks the proxy pool empty immediately when the last unfiltered proxy is deleted', async () => {
    const { api, management } = setupManagement()
    let resolveRefresh: ((value: unknown) => void) | undefined
    api.list
      .mockResolvedValueOnce({
        items: [createProxy({ id: 7, name: 'Tokyo proxy' })],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockImplementationOnce(() => new Promise(resolve => {
        resolveRefresh = resolve
      }))

    await management.loadProxies()
    expect(management.hasAnyProxy.value).toBe(true)

    const deletePromise = management.handleProxyDeleted(7)
    await flushPromises()

    expect(management.proxies.value).toEqual([])
    expect(management.hasAnyProxy.value).toBe(false)

    resolveRefresh?.({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await deletePromise
  })
})
