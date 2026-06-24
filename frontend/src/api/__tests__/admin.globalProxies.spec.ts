import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
    delete: del,
  },
}))

import adminGlobalProxiesAPI from '@/api/admin/globalProxies'

describe('admin global proxies api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('lists global proxies through the dedicated admin endpoint', async () => {
    const page = {
      items: [{ id: 1, name: 'global proxy', ip_type: 'residential', status: 'online', created_at: '', updated_at: '' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await adminGlobalProxiesAPI.list({ page: 1, page_size: 20, status: 'online', search: 'global' })

    expect(get).toHaveBeenCalledWith('/admin/global-proxies', { params: { page: 1, page_size: 20, status: 'online', search: 'global' } })
    expect(result).toEqual(page)
  })

  it('creates global proxies without a user owner', async () => {
    const proxy = { id: 2, name: 'global proxy', ip_type: 'dynamic', status: 'unknown', created_at: '', updated_at: '' }
    post.mockResolvedValue({ data: proxy })

    const payload = { name: 'global proxy', ip_type: 'dynamic' as const, endpoint: 'https://provider.example/proxy', remark: 'shared pool' }
    const result = await adminGlobalProxiesAPI.create(payload)

    expect(post).toHaveBeenCalledWith('/admin/global-proxies', payload)
    expect(post.mock.calls[0][1]).not.toHaveProperty('user_id')
    expect(result).toEqual(proxy)
  })

  it('supports single and batch connectivity tests through the global proxy endpoint', async () => {
    const check = { id: 1, status: 'online', latency_ms: 42 }
    post.mockResolvedValueOnce({ data: check })
    post.mockResolvedValueOnce({ data: [check] })

    await expect(adminGlobalProxiesAPI.test(1)).resolves.toEqual(check)
    await expect(adminGlobalProxiesAPI.testAll()).resolves.toEqual([check])

    expect(post).toHaveBeenNthCalledWith(1, '/admin/global-proxies/1/test')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/global-proxies/test')
  })

  it('updates and deletes through the global proxy endpoint', async () => {
    const proxy = { id: 2, name: 'global proxy', ip_type: 'static', endpoint: '', remark: '', status: 'unknown', created_at: '', updated_at: '' }
    put.mockResolvedValue({ data: proxy })
    del.mockResolvedValue({ data: undefined })

    await expect(adminGlobalProxiesAPI.update(2, { endpoint: '', remark: '' })).resolves.toEqual(proxy)
    await expect(adminGlobalProxiesAPI.delete(2)).resolves.toBeUndefined()

    expect(put).toHaveBeenCalledWith('/admin/global-proxies/2', { endpoint: '', remark: '' })
    expect(del).toHaveBeenCalledWith('/admin/global-proxies/2')
  })
})
