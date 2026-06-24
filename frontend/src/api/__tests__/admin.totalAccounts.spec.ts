import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
  },
}))

import { totalAccountsAPI } from '@/api/admin/totalAccounts'

describe('admin total accounts api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('lists total account pool records through the total account pool endpoint', async () => {
    const page = {
      items: [{ id: 7, name: '@pool', platform: 'x_twitter' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await totalAccountsAPI.list({ page: 1, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/admin/total-accounts', { params: { page: 1, page_size: 20 } })
    expect(result).toEqual(page)
  })

  it('passes total account pool filters through to the backend endpoint', async () => {
    const page = {
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0,
    }
    get.mockResolvedValue({ data: page })

    const params = {
      page: 1,
      page_size: 20,
      platform: 'x_twitter',
      account_status: 'available',
      task_status: 'stored',
      search: 'owner@example.com',
      unassigned: true,
    }
    const result = await totalAccountsAPI.list(params)

    expect(get).toHaveBeenCalledWith('/admin/total-accounts', { params })
    expect(result).toEqual(page)
  })

  it('batch assigns ownership through the total account pool endpoint', async () => {
    const batchResult = { total: 2, succeeded: 2, skipped: 0, failed: 0, errors: [], items: [] }
    post.mockResolvedValue({ data: batchResult })

    const result = await totalAccountsAPI.batchAssign([7, 8], 42)

    expect(post).toHaveBeenCalledWith('/admin/total-accounts/batch-assign', { ids: [7, 8], user_id: 42 })
    expect(result).toEqual(batchResult)
  })

  it('batch reclaims ownership through the total account pool endpoint', async () => {
    const batchResult = { total: 1, succeeded: 1, skipped: 0, failed: 0, errors: [], items: [] }
    post.mockResolvedValue({ data: batchResult })

    const result = await totalAccountsAPI.batchReclaim([7])

    expect(post).toHaveBeenCalledWith('/admin/total-accounts/batch-reclaim', { ids: [7] })
    expect(result).toEqual(batchResult)
  })

  it('batch deletes total account pool records through the total account pool endpoint', async () => {
    const batchResult = { total: 1, succeeded: 1, skipped: 0, failed: 0, errors: [], items: [] }
    post.mockResolvedValue({ data: batchResult })

    const result = await totalAccountsAPI.batchDelete([7])

    expect(post).toHaveBeenCalledWith('/admin/total-accounts/batch-delete', { ids: [7] })
    expect(result).toEqual(batchResult)
  })

  it('updates total account pool records through the total account pool endpoint', async () => {
    const account = { id: 7, name: '@pool', platform: 'x_twitter', account_status: 'limited', task_status: 'stored' }
    const payload = { password: 'new-secret', account_status: 'limited', remark: 'ops note' }
    put.mockResolvedValue({ data: account })

    const result = await totalAccountsAPI.update(7, payload)

    expect(put).toHaveBeenCalledWith('/admin/total-accounts/7', payload)
    expect(result).toEqual(account)
  })

  it('imports total account pool records through the total account pool endpoint', async () => {
    const file = new File(['name,password,two_factor\n@pool,secret,totp\n'], 'accounts.csv', { type: 'text/csv' })
    const importResult = { total: 1, succeeded: 1, created: 1, skipped: 0, failed: 0, duplicates: 0, errors: [], items: [] }
    post.mockResolvedValue({ data: importResult })

    const result = await totalAccountsAPI.importAccounts(file, 'x_twitter')

    expect(post).toHaveBeenCalledTimes(1)
    expect(post.mock.calls[0][0]).toBe('/admin/total-accounts/import')
    expect(post.mock.calls[0][1]).toBeInstanceOf(FormData)
    expect(post.mock.calls[0][2]).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } })
    expect(result).toEqual(importResult)
  })

  it('exports total account pool records through the total account pool endpoint', async () => {
    const blob = new Blob(['platform,name\nx_twitter,@pool\n'], { type: 'text/csv' })
    get.mockResolvedValue({ data: blob })

    const result = await totalAccountsAPI.exportAccounts()

    expect(get).toHaveBeenCalledWith('/admin/total-accounts/export', { responseType: 'blob' })
    expect(result).toBe(blob)
  })

  it('passes total account pool export filters through to the backend endpoint', async () => {
    const blob = new Blob(['platform,name\nx_twitter,@pool\n'], { type: 'text/csv' })
    const params = { search: '@pool', account_status: 'available', assigned: true, account_ids: [301, 302] }
    get.mockResolvedValue({ data: blob })

    const result = await totalAccountsAPI.exportAccounts(params)

    expect(get).toHaveBeenCalledWith('/admin/total-accounts/export', {
      params: {
        search: '@pool',
        account_status: 'available',
        assigned: true,
        account_ids: '301,302',
      },
      responseType: 'blob',
    })
    expect(result).toBe(blob)
  })
})
