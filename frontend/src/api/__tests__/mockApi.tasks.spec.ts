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
  auth_cookie?: string
  account_status: string
  task_status?: string
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

interface TestUploadFile {
  name: string
  type: string
  bytes: Uint8Array
}

interface MockTaskLog {
  id: number
  status: string
  charged: boolean
  charged_amount: number
  charge_status: string
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
  total_requests: number
  total_tokens: number
  total_cost: number
  total_actual_cost: number
}

interface MockDashboardStats extends MockUsageStats {
  today_requests: number
  today_tokens: number
  today_cost: number
  today_actual_cost: number
  by_platform?: Array<{
    platform: string
    total_requests: number
    total_tokens: number
    total_actual_cost: number
    today_requests: number
    today_tokens: number
    today_actual_cost: number
  }>
}

interface MockTrendPoint {
  date: string
  requests: number
  total_tokens: number
  cost: number
  actual_cost: number
}

interface MockAdminDashboardStats extends MockUsageStats {
  total_users: number
  active_users: number
  total_accounts: number
  normal_accounts: number
  error_accounts: number
  today_requests: number
  today_actual_cost: number
}

interface MockAdminRanking {
  ranking: Array<{
    user_id: number
    email: string
    requests: number
    tokens: number
    actual_cost: number
  }>
  total_requests: number
  total_tokens: number
  total_actual_cost: number
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

  it('charges successful mock social tasks with the current SocialOps unit price', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available')
    expect(account).toBeDefined()

    const directSubmit = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        action: 'login_check',
        client_request_id: 'mock-task-unit-price',
      },
    })
    expect(directSubmit.code).toBe('TASK_TEMPLATE_REQUIRED')

    const template = await requestJson<{ id: string }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Login check',
        type: 'login_check',
        params: {},
      },
    })
    expect(template.code).toBe(0)

    const result = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [account!.id],
        template_id: template.data.id,
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

  it('rejects mock user task submission before logging when the account has no default proxy', async () => {
    const baseUrl = await startMockApi()
    const authHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const accounts = await requestJson<MockPage<MockAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=20`, {
      headers: authHeaders,
    })
    const account = accounts.data.items.find((item) => item.account_status === 'available')
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
    expect(importResult.data.errors).toEqual(['account could not be imported'])
    expect(importResult.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ status: 'succeeded' }),
      expect.objectContaining({ status: 'duplicate', reason: 'duplicate_in_batch', error: 'account could not be imported' }),
    ]))
    expect(importResult.data.accounts).toHaveLength(1)
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

    const template = await requestJson<{ id: string }>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        name: 'Failed usage login check',
        type: 'login_check',
        params: {},
      },
    })
    expect(template.code).toBe(0)

    const failedSubmit = await requestJson<MockSubmitTaskResponse>(`${baseUrl}/api/v1/accounts/tasks`, {
      method: 'POST',
      headers: authHeaders,
      body: {
        account_ids: [importedAccount.id],
        template_id: template.data.id,
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
      total_requests: 1,
      total_tokens: 1,
      total_cost: 0.1,
      total_actual_cost: 0.1,
    })

    const failedStats = await requestJson<MockUsageStats>(`${baseUrl}/api/v1/usage/stats?status=failed`, {
      headers: authHeaders,
    })
    expect(failedStats.code).toBe(0)
    expect(failedStats.data).toMatchObject({
      total_requests: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
    })

    const dashboardStats = await requestJson<MockDashboardStats>(`${baseUrl}/api/v1/usage/dashboard/stats`, {
      headers: authHeaders,
    })
    expect(dashboardStats.code).toBe(0)
    expect(dashboardStats.data.total_requests).toBe(1)
    expect(dashboardStats.data.total_actual_cost).toBe(0.1)
    expect(dashboardStats.data.by_platform).toEqual(expect.arrayContaining([
      expect.objectContaining({
        platform: 'x_twitter',
        total_requests: 1,
        total_actual_cost: 0.1,
      }),
    ]))

    const trend = await requestJson<MockTrendPoint[]>(`${baseUrl}/api/v1/usage/dashboard/trend?granularity=day`, {
      headers: authHeaders,
    })
    expect(trend.code).toBe(0)
    expect(sumBy(trend.data, 'requests')).toBe(1)
    expect(sumBy(trend.data, 'actual_cost')).toBeCloseTo(0.1)
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
      total_accounts: 3,
      normal_accounts: 1,
      error_accounts: 0,
      ratelimit_accounts: 1,
      overload_accounts: 1,
      total_requests: 1,
      total_tokens: 1,
      total_actual_cost: 0.1,
    })

    const trend = await requestJson<MockTrendPoint[]>(`${baseUrl}/api/v1/admin/dashboard/trend?granularity=day`, {
      headers: authHeaders,
    })
    expect(trend.code).toBe(0)
    expect(sumBy(trend.data, 'requests')).toBe(1)
    expect(sumBy(trend.data, 'actual_cost')).toBeCloseTo(0.1)

    const ranking = await requestJson<MockAdminRanking>(`${baseUrl}/api/v1/admin/dashboard/users-ranking?limit=5`, {
      headers: authHeaders,
    })
    expect(ranking.code).toBe(0)
    expect(ranking.data).toMatchObject({
      total_requests: 1,
      total_tokens: 1,
      total_actual_cost: 0.1,
    })
    expect(ranking.data.ranking).toEqual(expect.arrayContaining([
      expect.objectContaining({
        user_id: 2,
        email: 'operator@example.test',
        requests: 1,
        tokens: 1,
        actual_cost: 0.1,
      }),
    ]))
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
