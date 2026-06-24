import { afterEach, describe, expect, it } from 'vitest'
import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process'
import { once } from 'node:events'
import { createServer } from 'node:net'
import { resolve } from 'node:path'
import { utils, write } from 'xlsx'

interface MockEnvelope<T> {
  code: number | string
  message: string
  data: T
  metadata?: Record<string, unknown>
}

interface MockPage<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

interface MockAccount {
  id: number
  name?: string
  platform?: string
  platform_user_id?: string | null
  auth_cookie?: string
  default_proxy_snapshot?: string
  phone?: string
  password?: string
  registration_ip?: string
  remark?: string
  account_status: string
  task_status?: string
  assigned_user_id?: number | null
  assigned_user_email?: string
  default_proxy_configured?: boolean
}

interface MockImportResult {
  total: number
  succeeded: number
  created: number
  skipped: number
  failed: number
  duplicates: number
  errors: string[]
  items: Array<{
    id?: number
    name?: string
    status: string
    reason?: string
    error?: string
  }>
  accounts: MockAccount[]
}

interface MockBatchResult {
  total: number
  succeeded: number
  skipped: number
  failed: number
  errors?: string[]
  items: Array<{
    id?: number
    name?: string
    status: string
    reason?: string
    error?: string
  }>
}

interface TestUploadFile {
  name: string
  type: string
  bytes: Uint8Array
}

interface MockTaskLog {
  id: number
  action?: string
  social_account_id?: number
  status: string
  target?: string
  content?: string
  charged: boolean
  charged_amount: number
  charge_status: string
  payload?: Record<string, unknown>
  template_snapshot?: {
    template_id?: string
    template_name?: string
    template_type?: string
    params?: Record<string, unknown>
  }
}

interface MockUsageLog {
  id: number
  user_id: number
  social_account_id: number
  platform: string
  account_name: string
  operation: string
  status: string
  quantity: number
  cost: number
  charge_status: string
  result_message?: string | null
  created_at: string
  completed_at?: string | null
}

interface MockUsageStats {
  total_operations: number
  success_count: number
  failed_count: number
  total_charged: number
}

interface MockDashboardStats {
  total_operations: number
  total_charged: number
  today_operations: number
  today_charged: number
  recent_operations_per_minute: number
  by_platform?: Array<{
    platform: string
    total_operations: number
    total_charged: number
    today_operations: number
    today_charged: number
  }>
}

interface MockDashboardTrendPoint {
  date: string
  operations: number
  charged: number
}

interface MockAdminDashboardStats {
  total_users: number
  today_new_users: number
  active_users: number
  hourly_active_users: number
  total_accounts: number
  normal_accounts: number
  error_accounts: number
  ratelimit_accounts: number
  overload_accounts: number
  total_operations: number
  total_charged: number
  today_operations: number
  today_charged: number
  average_duration_ms: number
  recent_operations_per_minute: number
  stats_updated_at: string
  stats_stale: boolean
}

interface MockAdminUserTrendPoint {
  date: string
  user_id: number
  email: string
  username: string
  operations: number
  charged: number
}

interface MockAdminRanking {
  ranking: Array<{
    user_id: number
    email: string
    operations: number
    charged: number
  }>
  total_operations: number
  total_charged: number
}

interface MockAdminUserUsageStats {
  total_operations: number
  total_charged: number
}

interface MockSubmitTaskResponse {
  submitted: number
  enqueued: number
  failed_closed: number
  logs: MockTaskLog[]
}

let mockServer: ChildProcessWithoutNullStreams | null = null

afterEach(async () => {
  if (!mockServer || mockServer.killed) {
    mockServer = null
    return
  }
  mockServer.kill()
  await Promise.race([once(mockServer, 'exit'), delay(1000)])
  mockServer = null
})

describe('mock API social task execution contract', () => {
  it('rejects unsupported admin mock task actions before creating task logs', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const statsBefore = await requestJson<MockAdminDashboardStats>(`${baseUrl}/api/v1/admin/dashboard/stats`, {
      headers: adminHeaders,
    })
    expect(statsBefore.code).toBe(0)

    for (const action of ['', 'tweet', 'dm', 'message', 'unsupported_action']) {
      const rejected = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/admin/accounts/tasks`, {
        method: 'POST',
        headers: adminHeaders,
        body: {
          account_ids: [1],
          action,
          client_request_id: `mock-admin-unsupported-${action || 'blank'}`,
        },
      })

      expect(rejected.code).toBe('SOCIAL_TASK_UNSUPPORTED_ACTION')
      expect(rejected.message).toContain('unsupported social task action')
    }

    const statsAfter = await requestJson<MockAdminDashboardStats>(`${baseUrl}/api/v1/admin/dashboard/stats`, {
      headers: adminHeaders,
    })
    expect(statsAfter.code).toBe(0)
    expect(statsAfter.data.total_operations).toBe(statsBefore.data.total_operations)
    expect(statsAfter.data.total_charged).toBe(statsBefore.data.total_charged)
  })

  it('rejects oversized task template pools like the real backend', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const targets = Array.from({ length: 501 }, (_, index) => `@target_${index + 1}`)

    const result = await requestJson<{ valid: boolean; errors: string[] }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Oversized follow',
        type: 'follow',
        params: { targets },
      },
    })

    expect(result.code).toBe('TASK_TEMPLATE_INVALID')
    expect(result.message).toContain('target list cannot exceed 500 items')
  })

  it('keeps structured mock task template payloads aligned with current SocialOps templates', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const template = await requestJson<{ id: string; type: string; params: Record<string, unknown> }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Profile refresh',
        type: 'update_profile',
        params: {
          profile: {
            display_name: 'Northwind Ops',
            description: 'SocialOps managed profile',
          },
        },
      },
    })
    expect(template.code).toBe(0)
    expect(template.data).toMatchObject({
      type: 'update_profile',
      params: {
        profile: {
          display_name: 'Northwind Ops',
          description: 'SocialOps managed profile',
        },
      },
    })

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        template_id: template.data.id,
        client_request_id: 'mock-structured-template',
      },
    })
    expect(result.code).toBe(0)
    expect(result.data.logs[0]).toMatchObject({
      action: 'update_profile',
      payload: {
        profile: {
          display_name: 'Northwind Ops',
          description: 'SocialOps managed profile',
        },
      },
      template_snapshot: {
        template_type: 'update_profile',
        params: {
          profile: {
            display_name: 'Northwind Ops',
            description: 'SocialOps managed profile',
          },
        },
      },
    })
  })

  it('submits parameterized user tasks with the default template when no template id is sent', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const template = await requestJson<{ id: string; type: string; params: Record<string, unknown> }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Default follow template',
        type: 'follow',
        params: { targets: ['@default_target'] },
        is_default: true,
      },
    })
    expect(template.code).toBe(0)

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'follow',
        client_request_id: 'mock-default-follow-template',
      },
    })

    expect(result.code).toBe(0)
    expect(result.data).toMatchObject({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
    })
    expect(result.data.logs[0]).toMatchObject({
      social_account_id: account!.id,
      action: 'follow',
      status: 'success',
      target: '@default_target',
      template_snapshot: {
        template_id: template.data.id,
        template_name: 'Default follow template',
        template_type: 'follow',
        params: {
          targets: ['@default_target'],
        },
      },
    })

    const logs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&statuses=success&limit=1`, {
      headers: authHeaders,
    })
    expect(logs.code).toBe(0)
    expect(logs.data.logs).toHaveLength(1)
    expect(logs.data.logs[0]).toMatchObject({
      id: result.data.logs[0].id,
      social_account_id: account!.id,
      action: 'follow',
      status: 'success',
      target: '@default_target',
      template_snapshot: {
        template_id: template.data.id,
      },
    })
    expect(JSON.stringify(logs.data.logs[0])).not.toContain('idempotency_key')
  })

  it('expands default profile templates into profile payloads and snapshots', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const profile = {
      display_name: 'Phase 27 Ops',
      screen_name: 'phase27_ops',
      description: 'SocialOps managed profile update',
      location: 'Shanghai',
      url: 'https://example.com/socialops',
    }

    const template = await requestJson<{
      id: string
      type: string
      params: {
        profile?: Record<string, unknown>
      }
    }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Default profile template',
        type: 'update_profile',
        params: { profile },
        is_default: true,
      },
    })
    expect(template.code).toBe(0)
    expect(template.data).toMatchObject({
      type: 'update_profile',
      params: { profile },
    })

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'update_profile',
        client_request_id: 'mock-default-profile-template',
      },
    })

    expect(result.code).toBe(0)
    expect(result.data.logs[0]).toMatchObject({
      social_account_id: account!.id,
      action: 'update_profile',
      status: 'success',
      payload: { profile },
      template_snapshot: {
        template_id: template.data.id,
        template_name: 'Default profile template',
        template_type: 'update_profile',
        params: { profile },
      },
    })

    const logs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&statuses=success&limit=1`, {
      headers: authHeaders,
    })
    expect(logs.code).toBe(0)
    expect(logs.data.logs[0]).toMatchObject({
      id: result.data.logs[0].id,
      action: 'update_profile',
      payload: { profile },
      template_snapshot: {
        template_id: template.data.id,
      },
    })
  })

  it('expands default avatar and banner templates into media payloads and snapshots', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const avatarTemplate = await requestJson<{
      id: string
      params: {
        avatar?: Record<string, unknown>
      }
    }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Default avatar template',
        type: 'update_avatar',
        params: {
          avatar: {
            source: 'inline',
            url: 'data:image/png;base64,QUJD',
            content_type: 'image/png',
            file_name: 'default-avatar.png',
            width: 400,
            height: 400,
          },
        },
        is_default: true,
      },
    })
    expect(avatarTemplate.code).toBe(0)
    expect(avatarTemplate.data.params.avatar).toMatchObject({
      source: 'library',
      content_type: 'image/png',
      file_name: 'default-avatar.png',
      width: 400,
      height: 400,
    })
    expect(String(avatarTemplate.data.params.avatar?.storage_key || '')).toMatch(/^social-task\/2\//)
    expect(avatarTemplate.data.params.avatar).not.toHaveProperty('url')

    const bannerTemplate = await requestJson<{
      id: string
      params: {
        banner?: Record<string, unknown>
      }
    }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Default banner template',
        type: 'update_banner',
        params: {
          banner: {
            source: 'inline',
            url: 'data:image/png;base64,REVG',
            content_type: 'image/png',
            file_name: 'default-banner.png',
            width: 1500,
            height: 500,
          },
        },
        is_default: true,
      },
    })
    expect(bannerTemplate.code).toBe(0)
    expect(bannerTemplate.data.params.banner).toMatchObject({
      source: 'library',
      content_type: 'image/png',
      file_name: 'default-banner.png',
      width: 1500,
      height: 500,
    })
    expect(String(bannerTemplate.data.params.banner?.storage_key || '')).toMatch(/^social-task\/2\//)
    expect(bannerTemplate.data.params.banner).not.toHaveProperty('url')

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const avatarResult = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'update_avatar',
        client_request_id: 'mock-default-avatar-template',
      },
    })
    expect(avatarResult.code).toBe(0)
    expect(avatarResult.data.logs[0]).toMatchObject({
      social_account_id: account!.id,
      action: 'update_avatar',
      status: 'success',
      payload: {
        avatar: {
          source: 'library',
          file_name: 'default-avatar.png',
          width: 400,
          height: 400,
        },
      },
      template_snapshot: {
        template_id: avatarTemplate.data.id,
        template_name: 'Default avatar template',
        template_type: 'update_avatar',
        params: {
          avatar: {
            source: 'library',
            file_name: 'default-avatar.png',
            width: 400,
            height: 400,
          },
        },
      },
    })
    expect(avatarResult.data.logs[0].payload?.avatar).not.toHaveProperty('url')
    expect(avatarResult.data.logs[0].template_snapshot?.params?.avatar).not.toHaveProperty('url')

    const bannerResult = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'update_banner',
        client_request_id: 'mock-default-banner-template',
      },
    })
    expect(bannerResult.code).toBe(0)
    expect(bannerResult.data.logs[0]).toMatchObject({
      social_account_id: account!.id,
      action: 'update_banner',
      status: 'success',
      payload: {
        banner: {
          source: 'library',
          file_name: 'default-banner.png',
          width: 1500,
          height: 500,
        },
      },
      template_snapshot: {
        template_id: bannerTemplate.data.id,
        template_name: 'Default banner template',
        template_type: 'update_banner',
        params: {
          banner: {
            source: 'library',
            file_name: 'default-banner.png',
            width: 1500,
            height: 500,
          },
        },
      },
    })
    expect(bannerResult.data.logs[0].payload?.banner).not.toHaveProperty('url')
    expect(bannerResult.data.logs[0].template_snapshot?.params?.banner).not.toHaveProperty('url')

    const logs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&statuses=success&limit=2`, {
      headers: authHeaders,
    })
    expect(logs.code).toBe(0)
    expect(logs.data.logs.map((log) => log.action)).toEqual(['update_banner', 'update_avatar'])
    expect(logs.data.logs[0].template_snapshot?.template_id).toBe(bannerTemplate.data.id)
    expect(logs.data.logs[1].template_snapshot?.template_id).toBe(avatarTemplate.data.id)
  })

  it('expands default post templates into content, payload text, media, and snapshots', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const content = 'Phase 26 post content'

    const template = await requestJson<{
      id: string
      type: string
      params: {
        contents?: string[]
        media?: Array<Record<string, unknown>>
      }
    }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Default post media template',
        type: 'post',
        params: {
          contents: [content],
          media: [{
            source: 'inline',
            url: 'data:image/png;base64,QUJD',
            content_type: 'image/png',
            file_name: 'phase26-post.png',
            width: 1200,
            height: 675,
          }],
        },
        is_default: true,
      },
    })
    expect(template.code).toBe(0)
    expect(template.data.params.media?.[0]).toMatchObject({
      source: 'library',
      content_type: 'image/png',
      file_name: 'phase26-post.png',
    })
    expect(template.data.params.media?.[0]).not.toHaveProperty('url')

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'post',
        client_request_id: 'mock-default-post-media-template',
      },
    })

    expect(result.code).toBe(0)
    expect(result.data.logs[0]).toMatchObject({
      social_account_id: account!.id,
      action: 'post',
      status: 'success',
      content,
      payload: {
        post: {
          text: content,
          media: [
            expect.objectContaining({
              source: 'library',
              content_type: 'image/png',
              file_name: 'phase26-post.png',
            }),
          ],
        },
      },
      template_snapshot: {
        template_id: template.data.id,
        template_name: 'Default post media template',
        template_type: 'post',
        params: {
          contents: [content],
          media: [
            expect.objectContaining({
              source: 'library',
              content_type: 'image/png',
              file_name: 'phase26-post.png',
            }),
          ],
        },
      },
    })
    const submittedMedia = (result.data.logs[0].payload?.post as { media?: Array<Record<string, unknown>> } | undefined)?.media?.[0]
    expect(submittedMedia).not.toHaveProperty('url')
    expect(String(submittedMedia?.storage_key || '')).toMatch(/^social-task\/2\//)

    const logs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&statuses=success&limit=1`, {
      headers: authHeaders,
    })
    expect(logs.code).toBe(0)
    expect(logs.data.logs[0]).toMatchObject({
      id: result.data.logs[0].id,
      action: 'post',
      content,
      payload: {
        post: {
          text: content,
        },
      },
      template_snapshot: {
        template_id: template.data.id,
      },
    })
  })

  it('materializes inline task-setting media and serves owned media previews like the real backend', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const template = await requestJson<{ params: { avatar?: Record<string, unknown> } }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Avatar media preview',
        type: 'update_avatar',
        params: {
          avatar: {
            source: 'inline',
            url: 'data:image/png;base64,QUJD',
            content_type: 'image/png',
            file_name: 'avatar-preview.png',
            width: 400,
            height: 400,
          },
        },
      },
    })

    expect(template.code).toBe(0)
    expect(template.data.params.avatar).toMatchObject({
      source: 'library',
      content_type: 'image/png',
      file_name: 'avatar-preview.png',
      width: 400,
      height: 400,
    })
    expect(String(template.data.params.avatar?.storage_key || '')).toMatch(/^social-task\/2\//)
    expect(template.data.params.avatar).not.toHaveProperty('url')

    const preview = await fetch(`${baseUrl}/api/v1/task-settings/media?storage_key=${encodeURIComponent(String(template.data.params.avatar?.storage_key))}`, {
      headers: authHeaders,
    })
    expect(preview.status).toBe(200)
    expect(preview.headers.get('content-type')).toContain('image/png')
    expect(preview.headers.get('content-disposition')).toContain('avatar-preview.png')
    expect(Buffer.from(await preview.arrayBuffer()).toString('utf8')).toBe('ABC')

    const rejected = await requestJson<unknown>(`${baseUrl}/api/v1/task-settings/media?storage_key=${encodeURIComponent('media/avatar.png')}`, {
      headers: authHeaders,
    })
    expect(rejected.code).toBe('TASK_TEMPLATE_MEDIA_SOURCE_UNSUPPORTED')
  })

  it('charges successful mock login tasks with the current SocialOps unit price', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const removedTemplate = await requestJson<{ id: string }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Login check',
        type: 'login_check',
        params: {},
      },
    })
    expect(removedTemplate.code).toBe('TASK_TEMPLATE_INVALID')
    expect(removedTemplate.message).toContain('unsupported task template type')

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'login',
        client_request_id: 'mock-task-unit-price',
      },
    })

    expect(result.code).toBe(0)
    expect(result.data).toMatchObject({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
    })
    expect(result.data.logs[0]).toMatchObject({
      status: 'success',
      charged: true,
      charged_amount: 0.1,
      charge_status: 'charged',
    })
  })

  it('rejects mock task submission before creating logs when balance cannot cover pending tasks', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-access-token' }

    const balance = await requestJson<{ balance: number }>(`${baseUrl}/api/v1/admin/users/2/balance`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        balance: 0,
        operation: 'set',
        notes: 'mock insufficient funds test',
      },
    })
    expect(balance.code).toBe(0)
    expect(balance.data.balance).toBe(0)

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const beforeLogs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&limit=100`, {
      headers: authHeaders,
    })
    const beforeUsage = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: authHeaders,
    })

    const rejected = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'login',
        client_request_id: 'mock-insufficient-funds',
      },
    })

    expect(rejected.code).toBe('SOCIAL_TASK_INSUFFICIENT_FUNDS')
    expect(rejected.message).toContain('insufficient subscription allowance')
    expect(rejected.metadata).toMatchObject({
      required_total: '0.10',
      wallet_balance: '0.00',
      wallet_required: '0.10',
    })

    const afterLogs = await requestJson<{ logs: MockTaskLog[] }>(`${baseUrl}/api/v1/accounts/tasks?account_ids=${account!.id}&limit=100`, {
      headers: authHeaders,
    })
    const afterUsage = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: authHeaders,
    })
    expect(afterLogs.data.logs).toHaveLength(beforeLogs.data.logs.length)
    expect(afterUsage.data.items).toHaveLength(beforeUsage.data.items.length)
  })

  it('rejects mock task submission when the account already has an active task', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const busyAccount = accounts.data.items.find((item) => item.name === 'x_busy_ops_03')
    expect(busyAccount).toBeDefined()

    const rejected = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [busyAccount!.id],
        action: 'login_check',
        client_request_id: 'mock-active-task-rejected',
      },
    })

    expect(rejected.code).toBe('SOCIAL_TASK_ACCOUNT_BUSY')
    expect(rejected.message).toContain('active task')

    const usage = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: authHeaders,
    })
    expect(usage.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ social_account_id: busyAccount!.id }),
    ]))
  })

  it('replays mock active task submissions with the same idempotency key', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const busyAccount = accounts.data.items.find((item) => item.name === 'x_busy_ops_03')
    expect(busyAccount).toBeDefined()

    const replayed = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [busyAccount!.id],
        action: 'login_check',
        client_request_id: 'mock-active-task-running',
      },
    })

    expect(replayed.code).toBe(0)
    expect(replayed.data).toMatchObject({
      submitted: 1,
      enqueued: 0,
      failed_closed: 0,
    })
    expect(replayed.data.logs[0]).toMatchObject({
      social_account_id: busyAccount!.id,
      action: 'login_check',
      status: 'running',
      charged: false,
      charged_amount: 0,
      charge_status: 'not_charged',
    })
  })

  it('replays only matching accounts during mixed mock idempotent batch submissions', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const suffix = Date.now()

    const imported = await requestJson<{ succeeded: number; items: Array<{ id?: number }>; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        accounts: [
          {
            name: `mock_idem_first_${suffix}`,
            platform: 'x_twitter',
            password: 'account-secret',
            two_factor: 'H6X33U477GHC22AR',
          },
          {
            name: `mock_idem_second_${suffix}`,
            platform: 'x_twitter',
            password: 'account-secret',
            two_factor: 'H6X33U477GHC22AR',
          },
        ],
      },
    })
    expect(imported.code).toBe(0)
    expect(imported.data.succeeded).toBe(2)
    const firstAccountID = imported.data.items[0].id
    const secondAccountID = imported.data.items[1].id
    expect(firstAccountID).toBeGreaterThan(0)
    expect(secondAccountID).toBeGreaterThan(0)

    const usableProxies = await requestJson<Array<{ id: number }>>(`${baseUrl}/api/v1/proxies/usable`, { headers: authHeaders })
    expect(usableProxies.data.length).toBeGreaterThan(0)
    for (const accountID of [firstAccountID, secondAccountID]) {
      const withProxy = await requestJson<MockAccount>(`${baseUrl}/api/v1/accounts/${accountID}/default-proxy`, {
        method: 'PUT',
        headers: authHeaders,
        body: { proxy_id: usableProxies.data[0].id },
      })
      expect(withProxy.code).toBe(0)
      expect(withProxy.data.default_proxy_configured).toBe(true)
    }

    const clientRequestID = `mock-mixed-idempotency-${suffix}`
    const firstSubmit = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [firstAccountID],
        action: 'login',
        client_request_id: clientRequestID,
      },
    })
    expect(firstSubmit.code).toBe(0)
    expect(firstSubmit.data).toMatchObject({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
    })
    expect(firstSubmit.data.logs[0]).toMatchObject({
      social_account_id: firstAccountID,
      status: 'success',
      charged: true,
    })

    const mixedReplay = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [firstAccountID, secondAccountID],
        action: 'login',
        client_request_id: clientRequestID,
      },
    })
    expect(mixedReplay.code).toBe(0)
    expect(mixedReplay.data).toMatchObject({
      submitted: 2,
      enqueued: 1,
      failed_closed: 0,
    })
    expect(mixedReplay.data.logs.map((log) => log.social_account_id)).toEqual([firstAccountID, secondAccountID])
    expect(mixedReplay.data.logs[0].id).toBe(firstSubmit.data.logs[0].id)
    expect(mixedReplay.data.logs[1].id).not.toBe(firstSubmit.data.logs[0].id)

    const usage = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: authHeaders,
    })
    expect(usage.code).toBe(0)
    expect(usage.data.items.filter((item) => item.social_account_id === firstAccountID)).toHaveLength(1)
    expect(usage.data.items.filter((item) => item.social_account_id === secondAccountID)).toHaveLength(1)
  })

  it('allows mock login tasks for pending imported accounts with a password and usable default proxy', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const importedName = `mock_login_pending_${Date.now()}`

    const imported = await requestJson<{ succeeded: number; items: Array<{ id?: number; status: string; reason?: string; error?: string }>; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        accounts: [{
          name: importedName,
          platform: 'x_twitter',
          password: 'account-secret',
          two_factor: 'H6X33U477GHC22AR',
        }],
      },
    })
    expect(imported.code).toBe(0)
    expect(imported.data.succeeded).toBe(1)
    const pendingAccountID = imported.data.items[0].id
    expect(pendingAccountID).toBeGreaterThan(0)
    const importedAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=100`, {
      headers: authHeaders,
    })
    const pendingAccount = importedAccounts.data.items.find((account) => account.id === pendingAccountID)
    expect(pendingAccount).toMatchObject({
      account_status: 'not_stored',
      default_proxy_configured: false,
    })

    const usableProxies = await requestJson<Array<{ id: number }>>(`${baseUrl}/api/v1/proxies/usable`, { headers: authHeaders })
    expect(usableProxies.data.length).toBeGreaterThan(0)
    const withProxy = await requestJson<MockAccount>(`${baseUrl}/api/v1/accounts/${pendingAccountID}/default-proxy`, {
      method: 'PUT',
      headers: authHeaders,
      body: { proxy_id: usableProxies.data[0].id },
    })
    expect(withProxy.data.default_proxy_configured).toBe(true)

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [pendingAccountID],
        action: 'login',
        client_request_id: 'mock-login-pending-with-proxy',
      },
    })

    expect(result.code).toBe(0)
    expect(result.data).toMatchObject({
      submitted: 1,
      enqueued: 1,
      failed_closed: 0,
    })
    expect(result.data.logs[0]).toMatchObject({
      social_account_id: pendingAccountID,
      action: 'login',
      status: 'success',
      charged: true,
      charged_amount: 0.1,
      charge_status: 'charged',
    })

    const refreshed = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=100`, {
      headers: authHeaders,
    })
    expect(refreshed.data.items.find((account) => account.id === pendingAccountID)).toMatchObject({
      account_status: 'available',
      task_status: 'success',
    })
  })

  it('rejects mock login tasks for imported accounts after the password is cleared', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const importedName = `mock_login_no_password_${Date.now()}`

    const imported = await requestJson<{ succeeded: number; items: Array<{ id?: number; status: string; reason?: string; error?: string }>; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        accounts: [{
          name: importedName,
          platform: 'x_twitter',
          password: 'temporary-secret',
          two_factor: 'H6X33U477GHC22AR',
        }],
      },
    })
    expect(imported.code).toBe(0)
    expect(imported.data.succeeded).toBe(1)
    const pendingAccountID = imported.data.items[0].id
    expect(pendingAccountID).toBeGreaterThan(0)

    const clearedPassword = await requestJson<MockAccount>(`${baseUrl}/api/v1/accounts/${pendingAccountID}`, {
      method: 'PUT',
      headers: authHeaders,
      body: { password: '' },
    })
    expect(clearedPassword.code).toBe(0)

    const usableProxies = await requestJson<Array<{ id: number }>>(`${baseUrl}/api/v1/proxies/usable`, { headers: authHeaders })
    expect(usableProxies.data.length).toBeGreaterThan(0)
    const withProxy = await requestJson<MockAccount>(`${baseUrl}/api/v1/accounts/${pendingAccountID}/default-proxy`, {
      method: 'PUT',
      headers: authHeaders,
      body: { proxy_id: usableProxies.data[0].id },
    })
    expect(withProxy.data.default_proxy_configured).toBe(true)

    const rejected = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [pendingAccountID],
        action: 'login',
        client_request_id: 'mock-login-pending-without-password',
      },
    })

    expect(rejected.code).toBe('SOCIAL_TASK_LOGIN_PASSWORD_REQUIRED')
    const usage = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: authHeaders,
    })
    expect(usage.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ idempotency_key: 'mock-login-pending-without-password' }),
    ]))
  })

  it('keeps mock user account edits scoped to delivery fields', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    expect(accounts.code).toBe(0)
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const updated = await requestJson<MockAccount>(`${baseUrl}/api/v1/accounts/${account!.id}`, {
      method: 'PUT',
      headers: authHeaders,
      body: {
        name: '@mock_identity_should_not_change',
        platform_user_id: 'mock-rest-should-not-change',
        account_status: 'invalid',
        task_status: 'manual_review',
        default_proxy_snapshot: '{"id":999,"endpoint":"http://should-not-change.example:8080"}',
        password: 'mock-edited-password',
        registration_ip: '203.0.113.90',
        remark: 'mock editable remark',
      },
    })

    expect(updated.code).toBe(0)
    expect(updated.data).toMatchObject({
      id: account!.id,
      name: account!.name,
      platform_user_id: account!.platform_user_id,
      account_status: account!.account_status,
      task_status: account!.task_status,
      default_proxy_snapshot: account!.default_proxy_snapshot,
      password: 'mock-edited-password',
      registration_ip: '203.0.113.90',
      remark: 'mock editable remark',
    })
  })

  it('rejects mock user task submission before logging when the account has no default proxy', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const clearProxy = await requestJson<unknown>(`${baseUrl}/api/v1/accounts/${account!.id}/default-proxy`, {
      method: 'PUT',
      headers: authHeaders,
      body: { proxy_id: null },
    })
    expect(clearProxy.code).toBe(0)

    const template = await requestJson<{ id: string }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Missing proxy follow',
        type: 'follow',
        params: { targets: ['@target'] },
      },
    })
    expect(template.code).toBe(0)

    const rejected = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        template_id: template.data.id,
        client_request_id: 'mock-missing-default-proxy',
      },
    })

    expect(rejected.code).toBe('SOCIAL_IP_NOT_AVAILABLE')

    const list = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=20`, {
      headers: authHeaders,
    })
    expect(list.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ idempotency_key: 'mock-missing-default-proxy' }),
    ]))
  })

  it('keeps mock batch imports in the user workbench until admin upload', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const importedName = `mock_staging_${Date.now()}`

    const importResult = await requestJson<{ accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: importedName,
          platform: 'x_twitter',
          password: 'account-secret',
          two_factor: 'H6X33U477GHC22AR',
        }],
      },
    })
    expect(importResult.code).toBe(0)
    expect(importResult.data.accounts[0]).toMatchObject({
      name: importedName,
      account_status: 'not_stored',
      task_status: 'pending',
    })

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=50`, {
      headers: userHeaders,
    })
    expect(userAccounts.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        name: importedName,
        account_status: 'not_stored',
        task_status: 'pending',
      }),
    ]))

    const totalPoolBeforeUpload = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolBeforeUpload.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ name: importedName }),
    ]))

    const uploadResult = await requestJson<{
      total: number
      succeeded: number
      skipped: number
      failed: number
      items: Array<{ id?: number; name?: string; status: string }>
    }>(`${baseUrl}/api/v1/admin/accounts/store-workbench`, {
      method: 'POST',
      headers: adminHeaders,
      body: { account_ids: [importResult.data.accounts[0].id] },
    })
    expect(uploadResult.code).toBe(0)
    expect(uploadResult.data).toMatchObject({
      total: 1,
      succeeded: 1,
      skipped: 0,
      failed: 0,
    })

    const totalPoolAfterUpload = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolAfterUpload.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        name: importedName,
        account_status: 'pending_check',
        task_status: 'stored',
      }),
    ]))
  })

  it('binds matching unassigned mock total-pool accounts during user batch import', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const importedName = `mock_pool_match_${Date.now()}`

    const poolAccount = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${importedName}`,
        platform: 'x_twitter',
        password: 'pool-secret',
        phone: '+15550001234',
        two_factor: 'POOL2FA',
        auth_cookie: 'ct0=pool; auth_token=pool',
        default_proxy_snapshot: JSON.stringify({ id: 999, name: 'pool-proxy', endpoint: 'http://pool-proxy.example:8080', status: 'online' }),
        remark: 'pool delivery note',
        assigned_user_id: null,
      },
    })
    expect(poolAccount.code).toBe(0)
    expect(poolAccount.data.assigned_user_id).toBeNull()
    expect(poolAccount.data.default_proxy_configured).toBe(true)

    const importResult = await requestJson<{ imported: number; accounts: MockAccount[]; items: Array<{ id?: number; status: string; reason?: string }> }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: importedName,
          platform: 'x_twitter',
          password: 'typed-secret',
          two_factor: 'TYPED2FA',
          auth_cookie: 'ct0=typed; auth_token=typed',
          remark: 'typed import note',
        }],
      },
    })
    expect(importResult.code).toBe(0)
    expect(importResult.data).toMatchObject({
      imported: 1,
      accounts: [expect.objectContaining({
        id: poolAccount.data.id,
        name: `@${importedName}`,
        password: 'pool-secret',
        phone: '+15550001234',
        auth_cookie: 'ct0=pool; auth_token=pool',
        remark: 'pool delivery note',
        task_status: 'stored',
        default_proxy_configured: false,
        default_proxy_snapshot: '',
      })],
      items: [expect.objectContaining({
        id: poolAccount.data.id,
        status: 'succeeded',
      })],
    })

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=50`, {
      headers: userHeaders,
    })
    expect(userAccounts.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: poolAccount.data.id,
        password: 'pool-secret',
        auth_cookie: 'ct0=pool; auth_token=pool',
      }),
    ]))

    const totalPool = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPool.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: poolAccount.data.id,
        assigned_user_id: 2,
        assigned_user_email: 'operator@example.test',
        default_proxy_configured: false,
      }),
    ]))
  })

  it('exports matched mock total-pool delivery fields from the user workbench', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const importedName = `mock_export_match_${Date.now()}`

    const poolAccount = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${importedName}`,
        platform: 'x_twitter',
        password: 'export-pool-secret',
        phone: '+15550004321',
        email: 'export-match@example.test',
        email_password: 'mail-export-secret',
        two_factor: 'EXPORT2FA',
        auth_cookie: 'ct0=export-pool; auth_token=export-pool',
        execution_auth: 'encrypted-export-execution-auth',
        remark: 'export pool delivery note',
        assigned_user_id: null,
      },
    })
    expect(poolAccount.code).toBe(0)

    const importResult = await requestJson<{ imported: number; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: importedName,
          platform: 'x_twitter',
          password: 'typed-secret',
          two_factor: 'TYPED2FA',
          auth_cookie: 'ct0=typed; auth_token=typed',
        }],
      },
    })
    expect(importResult.code).toBe(0)
    expect(importResult.data.imported).toBe(1)
    expect(importResult.data.accounts[0].id).toBe(poolAccount.data.id)

    const exported = await fetch(`${baseUrl}/api/v1/accounts/export`, {
      headers: userHeaders,
    })
    expect(exported.status).toBe(200)
    expect(exported.headers.get('content-type')).toContain('text/csv')
    const csv = await exported.text()

    expect(csv).toContain('platform,username,name,platform_user_id,password,phone,email,email_password,two_factor,backup_code,email_client_id,email_token,registration_ip,auth_cookie,execution_auth,default_proxy_snapshot,account_status,task_status,remark,created_at,updated_at')
    expect(csv).toContain(`"@${importedName}"`)
    expect(csv).toContain('"export-pool-secret"')
    expect(csv).toContain('"+15550004321"')
    expect(csv).toContain('"export-match@example.test"')
    expect(csv).toContain('"mail-export-secret"')
    expect(csv).toContain('"EXPORT2FA"')
    expect(csv).toContain('"ct0=export-pool; auth_token=export-pool"')
    expect(csv).toContain('"encrypted-export-execution-auth"')
    expect(csv).toContain('"export pool delivery note"')
    expect(csv).not.toContain('"typed-secret"')
    expect(csv).not.toContain('"TYPED2FA"')
  })

  it('hard deletes user workbench accounts from the mock total account pool', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: userHeaders,
    })
    expect(userAccounts.code).toBe(0)
    const account = userAccounts.data.items[0]
    expect(account).toBeTruthy()

    const totalPoolBeforeDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolBeforeDelete.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const deleted = await requestJson<null>(`${baseUrl}/api/v1/accounts/${account.id}`, {
      method: 'DELETE',
      headers: userHeaders,
    })
    expect(deleted.code).toBe(0)

    const userAccountsAfterDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: userHeaders,
    })
    expect(userAccountsAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const totalPoolAfterDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const missingAdminAccount = await fetch(`${baseUrl}/api/v1/admin/accounts/${account.id}`, {
      headers: adminHeaders,
    })
    expect(missingAdminAccount.status).toBe(404)
  })

  it('removes usage projections for mock task logs when a user account is hard deleted', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: userHeaders,
    })
    expect(userAccounts.code).toBe(0)
    const account = userAccounts.data.items.find((item) => item.account_status === 'available' && item.name !== 'x_busy_ops_03')
    expect(account).toBeDefined()

    const task = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        account_ids: [account!.id],
        action: 'login',
        client_request_id: 'mock-delete-clears-usage',
      },
    })
    expect(task.code).toBe(0)
    expect(task.data.logs).toEqual(expect.arrayContaining([
      expect.objectContaining({
        social_account_id: account!.id,
        status: 'success',
        charge_status: 'charged',
      }),
    ]))

    const usageBeforeDelete = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: userHeaders,
    })
    expect(usageBeforeDelete.code).toBe(0)
    expect(usageBeforeDelete.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        social_account_id: account!.id,
        status: 'success',
        charge_status: 'charged',
      }),
    ]))

    const deleted = await requestJson<null>(`${baseUrl}/api/v1/accounts/${account!.id}`, {
      method: 'DELETE',
      headers: userHeaders,
    })
    expect(deleted.code).toBe(0)

    const usageAfterDelete = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=100`, {
      headers: userHeaders,
    })
    expect(usageAfterDelete.code).toBe(0)
    expect(usageAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ social_account_id: account!.id }),
    ]))

    const totalPoolAfterDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account!.id }),
    ]))
  })

  it('hard deletes user workbench batch removals from the mock total account pool', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: userHeaders,
    })
    expect(userAccounts.code).toBe(0)
    const account = userAccounts.data.items[0]
    expect(account).toBeTruthy()

    const totalPoolBeforeDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolBeforeDelete.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const deleted = await requestJson<{
      total: number
      succeeded: number
      removed: number
      skipped: number
      failed: number
      items: Array<{ id?: number; status: string; reason?: string; error?: string }>
    }>(`${baseUrl}/api/v1/accounts/batch-delete`, {
      method: 'POST',
      headers: userHeaders,
      body: { ids: [account.id, account.id, 0, 0] },
    })
    expect(deleted.code).toBe(0)
    expect(deleted.data).toMatchObject({
      total: 4,
      succeeded: 1,
      removed: 1,
      skipped: 1,
      failed: 2,
    })
    expect(deleted.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id, status: 'succeeded' }),
      expect.objectContaining({ id: account.id, status: 'skipped', reason: 'duplicate_in_batch' }),
      expect.objectContaining({ id: 0, status: 'failed', reason: 'invalid_id' }),
    ]))
    expect(deleted.data.items.filter((item) => item.id === 0 && item.status === 'failed' && item.reason === 'invalid_id')).toHaveLength(2)

    const userAccountsAfterDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: userHeaders,
    })
    expect(userAccountsAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const totalPoolAfterDelete = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    expect(totalPoolAfterDelete.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id }),
    ]))

    const missingAdminAccount = await fetch(`${baseUrl}/api/v1/admin/accounts/${account.id}`, {
      headers: adminHeaders,
    })
    expect(missingAdminAccount.status).toBe(404)
  })

  it('requires an explicit admin token for mock total account pool access', async () => {
    const baseUrl = await startMockApi()

    const unauthenticated = await fetch(`${baseUrl}/api/v1/admin/total-accounts`)
    expect(unauthenticated.status).toBe(401)
    await expect(unauthenticated.json()).resolves.toMatchObject({
      code: 'UNAUTHENTICATED',
    })

    const userResponse = await fetch(`${baseUrl}/api/v1/admin/total-accounts`, {
      headers: { Authorization: 'Bearer dev-mock-user-token' },
    })
    expect(userResponse.status).toBe(403)
    await expect(userResponse.json()).resolves.toMatchObject({
      code: 'ADMIN_ONLY',
    })

    const adminResponse = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts`, {
      headers: { Authorization: 'Bearer dev-mock-admin-token' },
    })
    expect(adminResponse.code).toBe(0)
    expect(Array.isArray(adminResponse.data.items)).toBe(true)
  })

  it('requires an authenticated mock user for core user social operations', async () => {
    const baseUrl = await startMockApi()
    const protectedPaths = [
      '/api/v1/accounts',
      '/api/v1/accounts/tasks',
      '/api/v1/proxies',
      '/api/v1/proxies/usable',
      '/api/v1/task-settings/templates',
      '/api/v1/task-settings/media?storage_key=social-task/2/avatar.png',
    ]

    for (const path of protectedPaths) {
      const unauthenticated = await fetch(`${baseUrl}${path}`)
      expect(unauthenticated.status).toBe(401)
      await expect(unauthenticated.json()).resolves.toMatchObject({
        code: 'UNAUTHENTICATED',
      })

      const invalidToken = await fetch(`${baseUrl}${path}`, {
        headers: { Authorization: 'Bearer invalid-mock-token' },
      })
      expect(invalidToken.status).toBe(401)
      await expect(invalidToken.json()).resolves.toMatchObject({
        code: 'UNAUTHENTICATED',
      })
    }

    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    await expect(requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts`, {
      headers: userHeaders,
    })).resolves.toMatchObject({ code: 0 })
    await expect(requestJson<MockPage<unknown>>(`${baseUrl}/api/v1/proxies`, {
      headers: userHeaders,
    })).resolves.toMatchObject({ code: 0 })
    await expect(requestJson<unknown[]>(`${baseUrl}/api/v1/task-settings/templates`, {
      headers: userHeaders,
    })).resolves.toMatchObject({ code: 0 })
  })

  it('matches backend batch result semantics for mock total account pool operations', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const initialPool = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?page=1&page_size=50`, {
      headers: adminHeaders,
    })
    const unassignedAccount = initialPool.data.items.find(account => !Number(account.assigned_user_id))
    const assignedAccount = initialPool.data.items.find(account => Number(account.assigned_user_id) > 0)
    expect(unassignedAccount).toBeTruthy()
    expect(assignedAccount).toBeTruthy()
    expect(assignedAccount!.assigned_user_email).toBeTruthy()
    expect(assignedAccount!.assigned_user_email).not.toBe(`#${assignedAccount!.assigned_user_id}`)
    expect(typeof assignedAccount!.default_proxy_configured).toBe('boolean')

    const missingUserResult = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [unassignedAccount!.id], user_id: 404404 },
    })
    expect(missingUserResult.data).toMatchObject({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
    })
    expect(missingUserResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: unassignedAccount!.id, status: 'failed', reason: 'target_user_not_found' }),
    ]))

    const assignResult = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [unassignedAccount!.id, unassignedAccount!.id, assignedAccount!.id, 0], user_id: 2 },
    })
    expect(assignResult.data).toMatchObject({
      total: 4,
      succeeded: 1,
      skipped: 3,
      failed: 0,
    })
    expect(assignResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: unassignedAccount!.id, status: 'succeeded' }),
      expect.objectContaining({ id: unassignedAccount!.id, status: 'skipped', reason: 'duplicate_in_batch' }),
      expect.objectContaining({ id: assignedAccount!.id, status: 'skipped', reason: 'already_assigned' }),
      expect.objectContaining({ id: 0, status: 'skipped', reason: 'invalid_input' }),
    ]))

    const reclaimResult = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-reclaim`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [unassignedAccount!.id, unassignedAccount!.id, 0, 999999] },
    })
    expect(reclaimResult.data).toMatchObject({
      total: 4,
      succeeded: 1,
      skipped: 2,
      failed: 1,
    })
    expect(reclaimResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: unassignedAccount!.id, status: 'succeeded' }),
      expect.objectContaining({ id: unassignedAccount!.id, status: 'skipped', reason: 'duplicate_in_batch' }),
      expect.objectContaining({ id: 0, status: 'skipped', reason: 'invalid_id' }),
      expect.objectContaining({ id: 999999, status: 'failed', reason: 'reclaim_failed' }),
    ]))

    const deleteResult = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-delete`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [assignedAccount!.id, assignedAccount!.id, 0, 999999] },
    })
    expect(deleteResult.data).toMatchObject({
      total: 4,
      succeeded: 1,
      skipped: 2,
      failed: 1,
    })
    expect(deleteResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: assignedAccount!.id, status: 'succeeded' }),
      expect.objectContaining({ id: assignedAccount!.id, status: 'skipped', reason: 'duplicate_in_batch' }),
      expect.objectContaining({ id: 0, status: 'skipped', reason: 'invalid_id' }),
      expect.objectContaining({ id: 999999, status: 'failed', reason: 'delete_failed' }),
    ]))
  })

  it('rejects mock workbench staging accounts from total account pool ownership operations', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const stagingName = `mock_total_staging_guard_${Date.now()}`

    const stagingImport = await requestJson<{ accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: stagingName,
          platform: 'x_twitter',
          password: 'staging-secret',
          two_factor: 'H6X33U477GHC22AR',
        }],
      },
    })
    expect(stagingImport.code).toBe(0)
    const stagingAccount = stagingImport.data.accounts[0]
    expect(stagingAccount).toMatchObject({
      account_status: 'not_stored',
      task_status: 'pending',
    })

    const assign = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [stagingAccount.id], user_id: 1 },
    })
    expect(assign.data).toMatchObject({ succeeded: 0, failed: 1 })
    expect(assign.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: stagingAccount.id, status: 'failed', reason: 'assign_failed' }),
    ]))

    const reclaim = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-reclaim`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [stagingAccount.id] },
    })
    expect(reclaim.data).toMatchObject({ succeeded: 0, failed: 1 })
    expect(reclaim.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: stagingAccount.id, status: 'failed', reason: 'reclaim_failed' }),
    ]))

    const singleAssign = await fetch(`${baseUrl}/api/v1/admin/total-accounts/${stagingAccount.id}/assign`, {
      method: 'POST',
      headers: {
        ...adminHeaders,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ user_id: 1 }),
    })
    expect(singleAssign.status).toBe(404)
    await expect(singleAssign.json()).resolves.toMatchObject({ code: 'SOCIAL_ACCOUNT_NOT_FOUND' })

    const singleReclaim = await fetch(`${baseUrl}/api/v1/admin/total-accounts/${stagingAccount.id}/reclaim`, {
      method: 'POST',
      headers: adminHeaders,
    })
    expect(singleReclaim.status).toBe(404)
    await expect(singleReclaim.json()).resolves.toMatchObject({ code: 'SOCIAL_ACCOUNT_NOT_FOUND' })

    const deleteResult = await requestJson<MockBatchResult>(`${baseUrl}/api/v1/admin/total-accounts/batch-delete`, {
      method: 'POST',
      headers: adminHeaders,
      body: { ids: [stagingAccount.id] },
    })
    expect(deleteResult.data).toMatchObject({ succeeded: 0, failed: 1 })
    expect(deleteResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: stagingAccount.id, status: 'failed', reason: 'delete_failed' }),
    ]))

    const userAccounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?search=${encodeURIComponent(stagingName)}`, {
      headers: userHeaders,
    })
    expect(userAccounts.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: stagingAccount.id,
        account_status: 'not_stored',
        task_status: 'pending',
      }),
    ]))
  })

  it('applies mock total account pool search and assignment filters like the backend endpoint', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const suffix = Date.now()
    const unassignedName = `mock_total_filter_free_${suffix}`
    const assignedName = `mock_total_filter_owned_${suffix}`

    const unassigned = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: unassignedName,
        platform: 'x_twitter',
        assigned_user_id: null,
      },
    })
    const assigned = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: assignedName,
        platform: 'x_twitter',
        assigned_user_id: 2,
      },
    })
    expect(unassigned.code).toBe(0)
    expect(assigned.code).toBe(0)

    const searchUnassigned = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?search=${encodeURIComponent(unassignedName)}`, {
      headers: adminHeaders,
    })
    expect(searchUnassigned.data.items.map(item => item.id)).toEqual([unassigned.data.id])

    const assignedOnly = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?assigned=true&search=${encodeURIComponent(assignedName)}`, {
      headers: adminHeaders,
    })
    expect(assignedOnly.data.items).toEqual([
      expect.objectContaining({
        id: assigned.data.id,
        assigned_user_id: 2,
        assigned_user_email: 'operator@example.test',
      }),
    ])

    const unassignedOnly = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?unassigned=true&search=${encodeURIComponent(assignedName)}`, {
      headers: adminHeaders,
    })
    expect(unassignedOnly.data.items).toHaveLength(0)
  })

  it('exports only mock total-pool accounts through the total account pool export endpoint', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const poolName = `mock_total_pool_export_${Date.now()}`
    const stagingName = `mock_workbench_staging_export_${Date.now()}`

    const poolAccount = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: poolName,
        platform: 'x_twitter',
        password: 'pool-export-secret',
        auth_cookie: 'ct0=pool-export; auth_token=pool-export',
        assigned_user_id: null,
      },
    })
    expect(poolAccount.code).toBe(0)

    const stagingImport = await requestJson<{ imported: number; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: stagingName,
          platform: 'x_twitter',
          password: 'staging-export-secret',
          auth_cookie: 'ct0=staging-export; auth_token=staging-export',
        }],
      },
    })
    expect(stagingImport.code).toBe(0)
    expect(stagingImport.data.accounts[0]).toMatchObject({
      name: stagingName,
      task_status: 'pending',
    })

    const response = await fetch(`${baseUrl}/api/v1/admin/total-accounts/export`, {
      headers: adminHeaders,
    })
    expect(response.status).toBe(200)
    expect(response.headers.get('content-type')).toContain('text/csv')
    const csv = await response.text()
    expect(csv).toContain(poolName)
    expect(csv).toContain('pool-export-secret')
    expect(csv).not.toContain(stagingName)
    expect(csv).not.toContain('staging-export-secret')
  })

  it('keeps mock total-pool single-account ownership changes aligned with service rules', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const accountName = `mock_single_assign_${Date.now()}`

    const created = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        platform: 'x_twitter',
        name: accountName,
        password: 'single-assign-password',
        default_proxy_snapshot: JSON.stringify({ id: 8080, name: 'stale-pool-proxy', endpoint: 'http://stale.example:8080' }),
        assigned_user_id: null,
      },
    })
    expect(created.code).toBe(0)
    expect(created.data).toMatchObject({
      assigned_user_id: null,
      default_proxy_configured: true,
    })

    const missingUser = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${created.data.id}/assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { user_id: 404404 },
    })
    expect(missingUser.code).toBe('USER_NOT_FOUND')

    const assigned = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${created.data.id}/assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { user_id: 2 },
    })
    expect(assigned.data).toMatchObject({
      id: created.data.id,
      assigned_user_id: 2,
      assigned_user_email: 'operator@example.test',
      default_proxy_configured: false,
      task_status: 'stored',
    })
    expect(assigned.data.default_proxy_snapshot ?? '').toBe('')

    const alreadyAssigned = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${created.data.id}/assign`, {
      method: 'POST',
      headers: adminHeaders,
      body: { user_id: 1 },
    })
    expect(alreadyAssigned.code).toBe('SOCIAL_ACCOUNT_ALREADY_ASSIGNED')

    const fetched = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts/${created.data.id}`, {
      headers: adminHeaders,
    })
    expect(fetched.data).toMatchObject({
      assigned_user_id: 2,
      assigned_user_email: 'operator@example.test',
      default_proxy_configured: false,
    })

    const reclaimed = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${created.data.id}/reclaim`, {
      method: 'POST',
      headers: adminHeaders,
    })
    expect(reclaimed.data).toMatchObject({
      id: created.data.id,
      assigned_user_id: null,
      assigned_user_email: '',
      default_proxy_configured: false,
    })
    expect(reclaimed.data.default_proxy_snapshot ?? '').toBe('')
  })

  it('deduplicates mock workbench batch imports by normalized platform and account name', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const importedName = `mock_dupe_${Date.now()}`

    const importResult = await requestJson<{
      total: number
      succeeded: number
      imported: number
      skipped: number
      failed: number
      duplicates: number
      errors: string[]
      items: Array<{ id?: number; name?: string; status: string; reason?: string; error?: string }>
      accounts: MockAccount[]
    }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [
          {
            name: `@${importedName}`,
            platform: 'x_twitter',
            password: 'account-secret',
            two_factor: 'H6X33U477GHC22AR',
          },
          {
            name: importedName,
            platform: 'X_Twitter',
            password: 'account-secret',
            auth_cookie: 'ct0=mock',
          },
        ],
      },
    })

    expect(importResult.code).toBe(0)
    expect(importResult.data.total).toBe(2)
    expect(importResult.data.succeeded).toBe(1)
    expect(importResult.data.imported).toBe(1)
    expect(importResult.data.skipped).toBe(1)
    expect(importResult.data.failed).toBe(0)
    expect(importResult.data.duplicates).toBe(1)
    expect(importResult.data.errors).toEqual(['account is duplicated in this import batch'])
    expect(importResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ status: 'succeeded', reason: 'staged_not_stored' }),
      expect.objectContaining({ status: 'duplicate', reason: 'duplicate_in_batch', error: 'account is duplicated in this import batch' }),
    ]))
    expect(importResult.data.accounts).toHaveLength(1)
  })

  it('returns backend-aligned reason codes for mock user workbench batch import matching', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const suffix = Date.now()
    const matchedName = `mock_reason_matched_${suffix}`
    const stagedName = `mock_reason_staged_${suffix}`
    const assignedName = `mock_reason_assigned_${suffix}`
    const ambiguousName = `mock_reason_ambiguous_${suffix}`
    const invalidName = `mock_reason_invalid_${suffix}`

    await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${matchedName}`,
        platform: 'x_twitter',
        password: 'pool-secret',
        two_factor: 'POOL-TOTP',
        assigned_user_id: null,
      },
    })
    await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${assignedName}`,
        platform: 'x_twitter',
        password: 'assigned-secret',
        two_factor: 'ASSIGNED-TOTP',
      },
    })
    await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${ambiguousName}`,
        platform: 'x_twitter',
        password: 'ambiguous-x-secret',
        two_factor: 'AMBIGUOUS-X-TOTP',
        assigned_user_id: null,
      },
    })
    await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        name: `@${ambiguousName}`,
        platform: 'instagram',
        password: 'ambiguous-instagram-secret',
        two_factor: 'AMBIGUOUS-INSTAGRAM-TOTP',
        assigned_user_id: null,
      },
    })

    const importResult = await requestJson<{
      total: number
      succeeded: number
      imported: number
      skipped: number
      failed: number
      duplicates: number
      errors: string[]
      items: Array<{ id?: number; name?: string; status: string; reason?: string; error?: string }>
      accounts: MockAccount[]
    }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [
          {
            name: `@${matchedName}`,
            platform: 'x_twitter',
            password: 'user-secret',
            two_factor: 'USER-TOTP',
          },
          {
            name: `@${stagedName}`,
            platform: 'x_twitter',
            password: 'staged-secret',
            auth_cookie: 'ct0=staged',
          },
          {
            name: `@${invalidName}`,
            platform: 'x_twitter',
            two_factor: 'INVALID-TOTP',
          },
          {
            name: `@${assignedName}`,
            platform: 'x_twitter',
            password: 'assigned-secret',
            two_factor: 'ASSIGNED-TOTP',
          },
          {
            name: `@${ambiguousName}`,
            password: 'ambiguous-secret',
            two_factor: 'AMBIGUOUS-TOTP',
          },
        ],
      },
    })

    expect(importResult.code).toBe(0)
    expect(importResult.data).toMatchObject({
      total: 5,
      succeeded: 2,
      imported: 2,
      skipped: 3,
      failed: 2,
      duplicates: 1,
    })
    expect(importResult.data.errors).toEqual(expect.arrayContaining([
      'account import data is invalid',
      'account is already assigned to a workbench',
      'multiple total-pool accounts match this username',
    ]))
    expect(importResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ name: `@${matchedName}`, status: 'succeeded', reason: 'matched_total_pool' }),
      expect.objectContaining({ name: `@${stagedName}`, status: 'succeeded', reason: 'staged_not_stored' }),
      expect.objectContaining({ name: `@${invalidName}`, status: 'failed', reason: 'invalid_input', error: 'account import data is invalid' }),
      expect.objectContaining({ name: `@${assignedName}`, status: 'duplicate', reason: 'already_assigned', error: 'account is already assigned to a workbench' }),
      expect.objectContaining({ name: `@${ambiguousName}`, status: 'failed', reason: 'ambiguous_total_pool_match', error: 'multiple total-pool accounts match this username' }),
    ]))
    expect(importResult.data.accounts).toHaveLength(2)

    const duplicateWorkbenchResult = await requestJson<{
      duplicates: number
      items: Array<{ id?: number; name?: string; status: string; reason?: string; error?: string }>
    }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [
          {
            name: `@${matchedName}`,
            platform: 'x_twitter',
            password: 'user-secret',
            two_factor: 'USER-TOTP',
          },
        ],
      },
    })

    expect(duplicateWorkbenchResult.code).toBe(0)
    expect(duplicateWorkbenchResult.data.duplicates).toBe(1)
    expect(duplicateWorkbenchResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ name: `@${matchedName}`, status: 'duplicate', reason: 'already_in_workbench', error: 'account already exists in your workbench' }),
    ]))
  })

  it('deduplicates admin total-pool xlsx imports by normalized username when platform user ID is absent', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const importedName = `mock_xlsx_dupe_${Date.now()}`
    const file = createAdminImportXlsx([
      ['账号', '密码', '2FA', '备份码', '邮箱账号', '邮箱密码', '邮箱客户端ID', '邮箱令牌', '注册IP', 'Cookie'],
      [`@${importedName}`, 'account-secret', 'TOTP-SECRET', '', 'mail@example.test', 'mail-secret', 'client-id', 'mail-token', '127.0.0.1', 'ct0=xlsx; auth_token=xlsx'],
    ])

    const firstImport = await requestMultipart<MockImportResult>(`${baseUrl}/api/v1/admin/accounts/import`, file, adminHeaders)
    expect(firstImport.code).toBe(0)
    expect(firstImport.data).toMatchObject({
      total: 1,
      succeeded: 1,
      created: 1,
      skipped: 0,
      duplicates: 0,
    })
    expect(firstImport.data.accounts[0]).toMatchObject({
      name: `@${importedName}`,
      platform: 'x_twitter',
      auth_cookie: 'ct0=xlsx; auth_token=xlsx',
    })
    expect(firstImport.data.accounts[0].name).not.toMatch(/^file_upload_/)

    const secondImport = await requestMultipart<MockImportResult>(`${baseUrl}/api/v1/admin/accounts/import`, file, adminHeaders)
    expect(secondImport.code).toBe(0)
    expect(secondImport.data).toMatchObject({
      total: 1,
      succeeded: 0,
      created: 0,
      skipped: 1,
      duplicates: 1,
    })
    expect(secondImport.data.accounts).toHaveLength(0)
    expect(secondImport.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        name: `@${importedName}`,
        status: 'duplicate',
        reason: 'duplicate_in_database',
      }),
    ]))
  })

  it('imports admin total-pool accounts through the total-accounts API namespace', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const importedName = `mock_total_import_${Date.now()}`
    const file = createAdminImportXlsx([
      ['账号', '密码', '2FA', '备份码', '邮箱账号', '邮箱密码', '邮箱客户端ID', '邮箱令牌', '注册IP', 'Cookie'],
      [`@${importedName}`, 'account-secret', 'TOTP-SECRET', '', 'mail@example.test', 'mail-secret', 'client-id', 'mail-token', '127.0.0.1', 'ct0=total; auth_token=total'],
    ])

    const firstImport = await requestMultipart<MockImportResult>(`${baseUrl}/api/v1/admin/total-accounts/import`, file, adminHeaders)
    expect(firstImport.code).toBe(0)
    expect(firstImport.data).toMatchObject({
      total: 1,
      succeeded: 1,
      created: 1,
      skipped: 0,
      duplicates: 0,
    })
    expect(firstImport.data.accounts[0]).toMatchObject({
      name: `@${importedName}`,
      platform: 'x_twitter',
      auth_cookie: 'ct0=total; auth_token=total',
    })

    const totalPool = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/admin/total-accounts?search=${encodeURIComponent(importedName)}`, {
      headers: adminHeaders,
    })
    expect(totalPool.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: firstImport.data.accounts[0].id,
        name: `@${importedName}`,
      }),
    ]))

    const duplicateImport = await requestMultipart<MockImportResult>(`${baseUrl}/api/v1/admin/total-accounts/import`, file, adminHeaders)
    expect(duplicateImport.code).toBe(0)
    expect(duplicateImport.data).toMatchObject({
      total: 1,
      succeeded: 0,
      created: 0,
      skipped: 1,
      duplicates: 1,
    })
    expect(duplicateImport.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        name: `@${importedName}`,
        status: 'duplicate',
        reason: 'duplicate_in_database',
      }),
    ]))
  })

  it('updates only total-pool accounts through the total-accounts API namespace', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const accountName = `mock_total_update_${Date.now()}`
    const renamedPayload = `@${accountName}_renamed`

    const created = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/accounts`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        platform: 'x_twitter',
        name: accountName,
        password: 'update-secret',
        account_status: 'available',
        task_status: 'stored',
      },
    })
    expect(created.code).toBe(0)

    const updated = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${created.data.id}`, {
      method: 'PUT',
      headers: adminHeaders,
      body: {
        name: renamedPayload,
        password: 'updated-secret',
        remark: 'updated through total namespace',
        account_status: 'error',
      },
    })
    expect(updated.code).toBe(0)
    expect(updated.data).toMatchObject({
      id: created.data.id,
      name: accountName,
      password: 'updated-secret',
      remark: 'updated through total namespace',
      account_status: 'error',
    })
    expect(updated.data.name).not.toBe(renamedPayload)

    const staging = await requestJson<{ accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        accounts: [{
          name: `${accountName}_staging`,
          platform: 'x_twitter',
          password: 'staging-secret',
          two_factor: 'H6X33U477GHC22AR',
        }],
      },
    })
    expect(staging.code).toBe(0)

    const rejected = await requestJson<MockAccount>(`${baseUrl}/api/v1/admin/total-accounts/${staging.data.accounts[0].id}`, {
      method: 'PUT',
      headers: adminHeaders,
      body: {
        password: 'should-not-update',
      },
    })
    expect(rejected.code).toBe('SOCIAL_ACCOUNT_NOT_FOUND')
  })

  it('rejects non-available mock accounts before creating usage-visible task logs', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const importResult = await requestJson<{ imported: number; accounts: MockAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        accounts: [{
          name: 'x_failed_usage_projection',
          platform: 'x_twitter',
          password: 'account-secret',
          auth_cookie: 'ct0=mock-failed',
        }],
      },
    })
    expect(importResult.code).toBe(0)
    expect(importResult.data.imported).toBe(1)
    const importedAccount = importResult.data.accounts[0]
    expect(importedAccount).toBeTruthy()

    const failedSubmit = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [importedAccount.id],
        action: 'login_check',
        client_request_id: 'mock-usage-failed',
      },
    })
    expect(failedSubmit.code).toBe('SOCIAL_ACCOUNT_NOT_AVAILABLE')

    const list = await requestJson<MockPage<MockUsageLog>>(`${baseUrl}/api/v1/usage?page=1&page_size=20`, {
      headers: authHeaders,
    })
    expect(list.code).toBe(0)
    expect(list.data.total).toBe(1)
    expect(list.data.items).not.toEqual(expect.arrayContaining([
      expect.objectContaining({ social_account_id: importedAccount.id }),
    ]))

    const stats = await requestJson<MockUsageStats>(`${baseUrl}/api/v1/usage/stats`, {
      headers: authHeaders,
    })
    expect(stats.code).toBe(0)
    expect(stats.data).toMatchObject({
      total_operations: 1,
      success_count: 1,
      failed_count: 0,
      total_charged: 0.1,
    })
    expect(stats.data).not.toHaveProperty('total_requests')
    expect(stats.data).not.toHaveProperty('total_tokens')
    expect(stats.data).not.toHaveProperty('total_actual_cost')

    const failedStats = await requestJson<MockUsageStats>(`${baseUrl}/api/v1/usage/stats?status=failed`, {
      headers: authHeaders,
    })
    expect(failedStats.code).toBe(0)
    expect(failedStats.data).toMatchObject({
      total_operations: 0,
      success_count: 0,
      failed_count: 0,
      total_charged: 0,
    })

    const dashboardStats = await requestJson<MockDashboardStats>(`${baseUrl}/api/v1/usage/dashboard/stats`, {
      headers: authHeaders,
    })
    expect(dashboardStats.code).toBe(0)
    expect(dashboardStats.data.total_operations).toBe(1)
    expect(dashboardStats.data.total_charged).toBe(0.1)
    expect(dashboardStats.data.by_platform).toEqual(expect.arrayContaining([
      expect.objectContaining({
        platform: 'x_twitter',
        total_operations: 1,
        total_charged: 0.1,
      }),
    ]))
    expect(dashboardStats.data).not.toHaveProperty('total_requests')
    expect(dashboardStats.data).not.toHaveProperty('total_tokens')
    expect(dashboardStats.data).not.toHaveProperty('total_actual_cost')
    expect(dashboardStats.data).not.toHaveProperty('rpm')
    expect(dashboardStats.data.by_platform?.[0]).not.toHaveProperty('total_requests')
    expect(dashboardStats.data.by_platform?.[0]).not.toHaveProperty('total_actual_cost')

    const trend = await requestJson<MockDashboardTrendPoint[]>(`${baseUrl}/api/v1/usage/dashboard/trend?granularity=day`, {
      headers: authHeaders,
    })
    expect(trend.code).toBe(0)
    expect(sumBy(trend.data, 'operations')).toBe(1)
    expect(sumBy(trend.data, 'charged')).toBeCloseTo(0.1)
    expect(trend.data[0]).not.toHaveProperty('requests')
    expect(trend.data[0]).not.toHaveProperty('actual_cost')
  })

  it('projects mock social task logs through the existing admin dashboard endpoints', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const stats = await requestJson<MockAdminDashboardStats>(`${baseUrl}/api/v1/admin/dashboard/stats`, {
      headers: authHeaders,
    })
    expect(stats.code).toBe(0)
    expect(stats.data).toMatchObject({
      total_users: 2,
      total_accounts: 4,
      normal_accounts: 2,
      error_accounts: 0,
      ratelimit_accounts: 1,
      overload_accounts: 1,
      total_operations: 1,
      total_charged: 0.1,
    })
    expect(stats.data).not.toHaveProperty('total_api_keys')
    expect(stats.data).not.toHaveProperty('active_api_keys')
    expect(stats.data).not.toHaveProperty('total_tokens')
    expect(stats.data).not.toHaveProperty('total_cost')
    expect(stats.data).not.toHaveProperty('today_tokens')
    expect(stats.data).not.toHaveProperty('today_cost')
    expect(stats.data).not.toHaveProperty('total_requests')
    expect(stats.data).not.toHaveProperty('total_actual_cost')
    expect(stats.data).not.toHaveProperty('rpm')
    expect(stats.data).not.toHaveProperty('tpm')

    const trend = await requestJson<MockDashboardTrendPoint[]>(`${baseUrl}/api/v1/admin/dashboard/trend?granularity=day`, {
      headers: authHeaders,
    })
    expect(trend.code).toBe(0)
    expect(sumBy(trend.data, 'operations')).toBe(1)
    expect(sumBy(trend.data, 'charged')).toBeCloseTo(0.1)
    expect(trend.data[0]).not.toHaveProperty('requests')
    expect(trend.data[0]).not.toHaveProperty('actual_cost')
    expect(trend.data[0]).not.toHaveProperty('total_tokens')
    expect(trend.data[0]).not.toHaveProperty('cost')
    expect(trend.data[0]).not.toHaveProperty('input_tokens')
    expect(trend.data[0]).not.toHaveProperty('output_tokens')

    const userTrend = await requestJson<MockAdminUserTrendPoint[]>(`${baseUrl}/api/v1/admin/dashboard/users-trend?limit=5`, {
      headers: authHeaders,
    })
    expect(userTrend.code).toBe(0)
    expect(userTrend.data).toEqual(expect.arrayContaining([
      expect.objectContaining({
        user_id: 2,
        email: 'operator@example.test',
        operations: 1,
        charged: 0.1,
      }),
    ]))
    expect(userTrend.data[0]).not.toHaveProperty('requests')
    expect(userTrend.data[0]).not.toHaveProperty('actual_cost')
    expect(userTrend.data[0]).not.toHaveProperty('tokens')
    expect(userTrend.data[0]).not.toHaveProperty('cost')

    const ranking = await requestJson<MockAdminRanking>(`${baseUrl}/api/v1/admin/dashboard/users-ranking?limit=5`, {
      headers: authHeaders,
    })
    expect(ranking.code).toBe(0)
    expect(ranking.data).toMatchObject({
      total_operations: 1,
      total_charged: 0.1,
    })
    expect(ranking.data).not.toHaveProperty('total_requests')
    expect(ranking.data).not.toHaveProperty('total_actual_cost')
    expect(ranking.data).not.toHaveProperty('total_tokens')
    expect(ranking.data.ranking).toEqual(expect.arrayContaining([
      expect.objectContaining({
        user_id: 2,
        email: 'operator@example.test',
        operations: 1,
        charged: 0.1,
      }),
    ]))
    expect(ranking.data.ranking[0]).not.toHaveProperty('requests')
    expect(ranking.data.ranking[0]).not.toHaveProperty('actual_cost')
    expect(ranking.data.ranking[0]).not.toHaveProperty('tokens')

    const userUsageStats = await requestJson<MockAdminUserUsageStats>(`${baseUrl}/api/v1/admin/users/2/usage?period=month`, {
      headers: authHeaders,
    })
    expect(userUsageStats.code).toBe(0)
    expect(userUsageStats.data).toMatchObject({
      total_operations: 1,
      total_charged: 0.1,
    })
    expect(userUsageStats.data).not.toHaveProperty('period')
    expect(userUsageStats.data).not.toHaveProperty('total_requests')
    expect(userUsageStats.data).not.toHaveProperty('total_cost')
    expect(userUsageStats.data).not.toHaveProperty('total_tokens')
  })
})

async function startMockApi(): Promise<string> {
  const port = await getFreePort()
  const projectRoot = resolve(__dirname, '../../../..')
  const mockPath = resolve(projectRoot, 'tools/mock-api.mjs')
  const stdout: string[] = []
  const stderr: string[] = []
  mockServer = spawn(process.execPath, [mockPath], {
    cwd: projectRoot,
    env: { ...process.env, MOCK_API_PORT: String(port) },
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  mockServer.stdout.on('data', (chunk: Buffer) => stdout.push(chunk.toString('utf8')))
  mockServer.stderr.on('data', (chunk: Buffer) => stderr.push(chunk.toString('utf8')))

  const baseUrl = `http://127.0.0.1:${port}`
  const deadline = Date.now() + 5000
  while (Date.now() < deadline) {
    if (mockServer.exitCode !== null) {
      throw new Error(`mock API exited early: ${stderr.join('') || stdout.join('')}`)
    }
    try {
      const response = await fetch(`${baseUrl}/api/v1/settings/public`)
      if (response.ok) return baseUrl
    } catch {
      await delay(50)
    }
  }
  throw new Error(`mock API did not start: ${stderr.join('') || stdout.join('')}`)
}

async function requestJson<T>(
  url: string,
  options: { method?: string; headers?: Record<string, string>; body?: Record<string, unknown> } = {}
): Promise<MockEnvelope<T>> {
  const response = await fetch(url, {
    method: options.method ?? 'GET',
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.headers ?? {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  })
  return response.json() as Promise<MockEnvelope<T>>
}

async function requestMultipart<T>(
  url: string,
  file: TestUploadFile,
  headers: Record<string, string> = {}
): Promise<MockEnvelope<T>> {
  const boundary = `mock-api-test-${Date.now()}-${Math.random().toString(16).slice(2)}`
  const fileBytes = file.bytes
  const prefix = new TextEncoder().encode(
    `--${boundary}\r\n` +
    `Content-Disposition: form-data; name="file"; filename="${file.name}"\r\n` +
    `Content-Type: ${file.type || 'application/octet-stream'}\r\n\r\n`
  )
  const suffix = new TextEncoder().encode(`\r\n--${boundary}--\r\n`)
  const body = new Uint8Array(prefix.length + fileBytes.length + suffix.length)
  body.set(prefix, 0)
  body.set(fileBytes, prefix.length)
  body.set(suffix, prefix.length + fileBytes.length)
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      ...headers,
      'Content-Type': `multipart/form-data; boundary=${boundary}`,
    },
    body,
  })
  return response.json() as Promise<MockEnvelope<T>>
}

function createAdminImportXlsx(rows: string[][]): TestUploadFile {
  const workbook = utils.book_new()
  utils.book_append_sheet(workbook, utils.aoa_to_sheet(rows), 'accounts')
  const bytes = write(workbook, { bookType: 'xlsx', type: 'array' }) as ArrayBuffer
  return {
    name: 'accounts.xlsx',
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    bytes: new Uint8Array(bytes),
  }
}

async function getFreePort(): Promise<number> {
  return new Promise((resolvePort, reject) => {
    const server = createServer()
    server.on('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        server.close()
        reject(new Error('failed to allocate mock API port'))
        return
      }
      const { port } = address
      server.close(() => resolvePort(port))
    })
  })
}

function delay(ms: number): Promise<void> {
  return new Promise((resolveDelay) => {
    setTimeout(resolveDelay, ms)
  })
}

function sumBy<T extends Record<string, unknown>>(items: T[], key: keyof T): number {
  return items.reduce((sum, item) => sum + Number(item[key] || 0), 0)
}
