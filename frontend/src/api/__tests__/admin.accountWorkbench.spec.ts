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

  it('does not expose removed workbench-to-pool admin endpoints', () => {
    expect(('register') in accountWorkbenchAdminAPI).toBe(false)
    expect(('store' + 'WorkbenchAccounts') in accountWorkbenchAdminAPI).toBe(false)
    expect(accountWorkbenchApiSource).not.toContain('/admin/accounts/register')
    expect(accountWorkbenchApiSource).not.toContain('/admin/accounts/store-workbench')
    expect(mockApiSource).not.toContain('/api/v1/admin/accounts/register')
    expect(mockApiSource).not.toContain('/api/v1/admin/accounts/store-workbench')
  })

  it('does not expose removed task estimation contracts in admin wrapper or mock API', () => {
    expect(('estimate' + 'Task') in accountWorkbenchAdminAPI).toBe(false)
    expect(mockApiSource).not.toContain('${userTaskPath}/estimate')
    expect(mockApiSource).not.toContain('${adminTaskPath}/estimate')
    expect(mockApiSource).not.toContain('billing_estimate')
    expect(mockApiSource).not.toContain('billing_estimates')
  })

  it('keeps default proxy snapshots in the create contract without workbench registration shortcuts', () => {
    const createRequestSource = accountWorkbenchApiSource.slice(
      accountWorkbenchApiSource.indexOf('export interface CreateSocialAccountRequest'),
      accountWorkbenchApiSource.indexOf('export interface UpdateSocialAccountRequest'),
    )

    expect(createRequestSource).toContain('default_proxy_snapshot?: string')
    expect(accountWorkbenchApiSource).not.toContain('RegisterSocialAccountRequest')
    expect(accountWorkbenchApiSource).not.toContain('SocialAccountStoreResult')
    expect(accountWorkbenchApiSource).not.toContain('register(data: RegisterSocialAccountRequest)')
  })
})
