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

import accountWorkbenchAPI from '@/api/accountWorkbench'

describe('user account workbench api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('unwraps the current user account list for dashboard widgets', async () => {
    const page = {
      items: [{
        id: 1,
        name: '@mine',
        platform: 'x_twitter',
        username: 'mine',
        platform_user_id: 'x-1',
        password: 'pool-secret',
        phone: '+15550000001',
        email: 'mine@example.com',
        email_password: 'mail-secret',
        execution_auth: '{"access_token":"token"}',
        default_proxy_snapshot: '{"id":2,"endpoint":"http://proxy.local:8080"}',
        account_status: 'available',
        task_status: 'idle',
        created_at: '',
        updated_at: ''
      }],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await accountWorkbenchAPI.listMyAccounts({ page: 1, page_size: 100 })

    expect(get).toHaveBeenCalledWith('/accounts', { params: { page: 1, page_size: 100 } })
    expect(result).toEqual(page)
  })

  it('does not expose the removed account task-log reader', () => {
    expect(('listMy' + 'TaskLogs') in accountWorkbenchAPI).toBe(false)
  })

  it('does not expose the removed task estimation endpoint', () => {
    expect(('estimate' + 'Task') in accountWorkbenchAPI).toBe(false)
  })

  it('submits user tasks with a saved template contract only', async () => {
    post.mockResolvedValue({ data: { submitted: 1, enqueued: 1, failed_closed: 0, logs: [] } })

    await accountWorkbenchAPI.submitTask({
      account_ids: [7],
      template_id: 'tmpl_follow_1',
      client_request_id: 'user-template-submit',
    })

    expect(post).toHaveBeenCalledWith('/accounts/tasks', {
      account_ids: [7],
      template_id: 'tmpl_follow_1',
      client_request_id: 'user-template-submit',
    })
    expect(JSON.stringify(post.mock.calls[0][1])).not.toContain('"action"')
    expect(JSON.stringify(post.mock.calls[0][1])).not.toContain('"target"')
    expect(JSON.stringify(post.mock.calls[0][1])).not.toContain('"content"')
  })

  it('does not expose the removed single-account import endpoint', () => {
    expect(('importMy' + 'Account') in accountWorkbenchAPI).toBe(false)
  })

  it('uses unified account workbench endpoints for user import and delete operations', async () => {
    post.mockResolvedValueOnce({
      data: {
        total: 2,
        succeeded: 2,
        imported: 2,
        skipped: 0,
        failed: 0,
        duplicates: 0,
        errors: [],
        items: [
          { id: 1, name: '@one', status: 'succeeded' },
          { id: 2, name: '@two', status: 'succeeded' },
        ],
        accounts: [],
      },
    })
    post.mockResolvedValueOnce({ data: { total: 2, removed: 2, skipped: 0, errors: [] } })
    del.mockResolvedValue({ data: undefined })

    const importResult = await accountWorkbenchAPI.batchImportMyAccounts([
      { platform: 'x_twitter', name: '@one' },
      { platform: 'x_twitter', name: '@two' },
    ])
    await accountWorkbenchAPI.batchDeleteMyAccounts([1, 2])
    await accountWorkbenchAPI.deleteMyAccount(1)

    expect(post).toHaveBeenNthCalledWith(1, '/accounts/batch-import', { accounts: [
      { platform: 'x_twitter', name: '@one' },
      { platform: 'x_twitter', name: '@two' },
    ] })
    expect(post).toHaveBeenNthCalledWith(2, '/accounts/batch-delete', { ids: [1, 2] })
    expect(del).toHaveBeenCalledWith('/accounts/1')
    expect(importResult).toMatchObject({
      total: 2,
      succeeded: 2,
      imported: 2,
      skipped: 0,
      failed: 0,
      duplicates: 0,
      items: [
        { id: 1, name: '@one', status: 'succeeded' },
        { id: 2, name: '@two', status: 'succeeded' },
      ],
    })
  })

  it('updates only mutable current-user account fields through the user endpoint', async () => {
    const payload = {
      password: 'new-password',
      email: 'new@example.com',
      email_password: 'mail-secret',
      two_factor: 'totp',
      backup_code: 'backup',
      email_client_id: 'client-id',
      email_token: 'email-token',
      auth_cookie: 'ct0=token',
      execution_auth: '{"access_token":"token"}',
      remark: 'operator note',
    }
    put.mockResolvedValue({ data: { id: 7, name: '@kept', platform: 'x_twitter', account_status: 'available', task_status: 'stored', created_at: '', updated_at: '' } })

    await accountWorkbenchAPI.updateMyAccount(7, payload)

    expect(put).toHaveBeenCalledWith('/accounts/7', payload)
    expect(JSON.stringify(put.mock.calls[0][1])).not.toContain('platform_user_id')
    expect(JSON.stringify(put.mock.calls[0][1])).not.toContain('registration_ip')
    expect(JSON.stringify(put.mock.calls[0][1])).not.toContain('"name"')
    expect(JSON.stringify(put.mock.calls[0][1])).not.toContain('"platform"')
  })
})
