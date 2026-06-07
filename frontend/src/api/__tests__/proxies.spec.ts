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

import proxiesAPI from '@/api/proxies'

describe('user scoped proxies api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('uses /proxies for current-user proxy lists', async () => {
    const page = {
      items: [{ id: 1, user_id: 1, name: 'proxy', ip_type: 'residential', status: 'online', created_at: '', updated_at: '' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await proxiesAPI.list({ page: 1, page_size: 20, status: 'online' })

    expect(get).toHaveBeenCalledWith('/proxies', { params: { page: 1, page_size: 20, status: 'online' } })
    expect(result).toEqual(page)
  })

  it('creates proxies without user_id because ownership comes from JWT', async () => {
    const proxy = { id: 2, user_id: 7, name: 'proxy', ip_type: 'static', status: 'unknown', created_at: '', updated_at: '' }
    post.mockResolvedValue({ data: proxy })

    const payload = { name: 'proxy', ip_type: 'static' as const, endpoint: 'http://proxy.example:8080', remark: 'note' }
    const result = await proxiesAPI.create(payload)

    expect(post).toHaveBeenCalledWith('/proxies', payload)
    expect(post.mock.calls[0][1]).not.toHaveProperty('user_id')
    expect(result).toEqual(proxy)
  })

  it('supports single and batch tests through user-scoped endpoints', async () => {
    const check = { id: 1, status: 'online', latency_ms: 42 }
    post.mockResolvedValueOnce({ data: check })
    post.mockResolvedValueOnce({ data: [check] })

    await expect(proxiesAPI.test(1)).resolves.toEqual(check)
    await expect(proxiesAPI.testAll()).resolves.toEqual([check])

    expect(post).toHaveBeenNthCalledWith(1, '/proxies/1/test')
    expect(post).toHaveBeenNthCalledWith(2, '/proxies/test')
  })

  it('updates and deletes through /proxies', async () => {
    const proxy = { id: 2, user_id: 7, name: 'proxy', ip_type: 'static', endpoint: '', remark: '', status: 'unknown', created_at: '', updated_at: '' }
    put.mockResolvedValue({ data: proxy })
    del.mockResolvedValue({ data: undefined })

    await expect(proxiesAPI.update(2, { endpoint: '', remark: '' })).resolves.toEqual(proxy)
    await expect(proxiesAPI.delete(2)).resolves.toBeUndefined()

    expect(put).toHaveBeenCalledWith('/proxies/2', { endpoint: '', remark: '' })
    expect(del).toHaveBeenCalledWith('/proxies/2')
  })
})
