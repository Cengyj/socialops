import { beforeEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

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

import { accountWorkbenchAdminAPI } from '@/api/accountWorkbench'

const mockApiSource = readFileSync(
  resolve(__dirname, '../../../../tools/mock-api.mjs'),
  'utf8',
)
const accountWorkbenchApiSource = readFileSync(
  resolve(__dirname, '../accountWorkbench.ts'),
  'utf8',
)

describe('admin account workbench api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('unwraps paginated account list responses for account pool pages', async () => {
    const page = {
      items: [{ id: 1, name: '@qa', platform: 'x_twitter', account_status: 'available', task_status: 'stored', created_at: '', updated_at: '' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await accountWorkbenchAdminAPI.list({ page: 1, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/admin/accounts', { params: { page: 1, page_size: 20 } })
    expect(result).toEqual(page)
  })

  it('does not expose the removed admin task-log reader', () => {
    expect(('list' + 'TaskLogs') in accountWorkbenchAdminAPI).toBe(false)
  })

  it('uses void single-delete and structured batch-delete account contracts', async () => {
    const batchResult = {
      total: 2,
      succeeded: 1,
      skipped: 1,
      failed: 0,
      duplicates: 0,
      errors: [],
      items: [
        { id: 7, status: 'succeeded' },
        { id: 0, status: 'skipped', reason: 'invalid_id', error: 'account could not be processed' },
      ],
    }
    del.mockResolvedValueOnce({ data: undefined })
    post.mockResolvedValueOnce({ data: batchResult })

    await expect(accountWorkbenchAdminAPI.delete(7)).resolves.toBeUndefined()
    await expect(accountWorkbenchAdminAPI.batchDelete([7, 0])).resolves.toEqual(batchResult)

    expect(del).toHaveBeenCalledWith('/admin/accounts/7')
    expect(post).toHaveBeenCalledWith('/admin/accounts/batch-delete', { ids: [7, 0] })
  })

  it('exposes selected workbench upload without restoring the removed register placeholder', async () => {
    const batchResult = { total: 2, succeeded: 1, skipped: 1, failed: 0, items: [] }
    post.mockResolvedValueOnce({ data: batchResult })

    await expect(accountWorkbenchAdminAPI.storeWorkbenchAccounts([1, 2])).resolves.toEqual(batchResult)

    expect(post).toHaveBeenCalledWith('/admin/accounts/store-workbench', { account_ids: [1, 2] })
    expect('register' in accountWorkbenchAdminAPI).toBe(false)
    expect(accountWorkbenchApiSource).not.toContain('`${ADMIN_BASE}/register`')
    expect(accountWorkbenchApiSource).toContain('`${ADMIN_BASE}/store-workbench`')
    expect(mockApiSource).not.toContain('/api/v1/admin/accounts/register')
    expect(mockApiSource).toContain('/api/v1/admin/accounts/store-workbench')
  })

  it('does not expose removed task estimation contracts in admin wrapper or mock API', () => {
    expect(('estimate' + 'Task') in accountWorkbenchAdminAPI).toBe(false)
    expect(mockApiSource).not.toContain('${userTaskPath}/estimate')
    expect(mockApiSource).not.toContain('${adminTaskPath}/estimate')
    expect(mockApiSource).not.toContain('billing_estimate')
    expect(mockApiSource).not.toContain('billing_estimates')
  })

  it('keeps default proxy snapshots in the create contract without a register placeholder payload', () => {
    const createRequestSource = accountWorkbenchApiSource.slice(
      accountWorkbenchApiSource.indexOf('export interface CreateSocialAccountRequest'),
      accountWorkbenchApiSource.indexOf('export interface UpdateSocialAccountRequest'),
    )

    expect(createRequestSource).toContain('default_proxy_snapshot?: string')
    expect(accountWorkbenchApiSource).not.toContain('RegisterSocialAccountRequest')
    expect(accountWorkbenchApiSource).not.toContain('SocialAccountStoreResult')
    expect(accountWorkbenchApiSource).not.toContain('register(data: CreateSocialAccountRequest)')
  })

  it('matches backend wording for rejected old .xls account-pool imports', () => {
    expect(mockApiSource).toContain('old .xls social account imports are not supported')
    expect(mockApiSource).not.toContain(['leg' + 'acy', '.xls social account imports are not supported'].join(' '))
  })
})
