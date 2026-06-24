import { afterEach, describe, expect, it } from 'vitest'
import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process'
import { once } from 'node:events'
import { readFileSync } from 'node:fs'
import { createServer } from 'node:net'
import { resolve } from 'node:path'

interface MockEnvelope<T> {
  code: number
  message: string
  data: T
}

interface MockProxy {
  id: number
  user_id: number
  name: string
  ip_type: string
  endpoint: string
  status: string
  latency_ms?: number | null
  error?: string
  remark: string
}

interface MockPage<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

interface MockSocialAccount {
  id: number
  default_proxy_snapshot: string
  default_proxy_configured: boolean
}

interface MockBatchDefaultProxyResult {
  total: number
  succeeded: number
  skipped: number
  failed: number
  errors?: string[]
  items: Array<{ id: number; status: string; reason?: string; error?: string }>
}

interface MockTaskTemplate {
  id: string
  name: string
  type: string
  params: {
    targets?: string[]
    contents?: string[]
  }
  is_default: boolean
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

describe('mock API user proxy contract', () => {
  it('keeps proxy deletion aligned with backend task-log reference cleanup', () => {
    const projectRoot = resolve(__dirname, '../../../..')
    const source = readFileSync(resolve(projectRoot, 'tools/mock-api.mjs'), 'utf8')
    const deleteStart = source.indexOf('function deleteMockProxy(id)')
    const deleteEnd = source.indexOf('function createMockSocialAccount', deleteStart)
    const deleteSource = source.slice(deleteStart, deleteEnd)

    expect(deleteSource).toContain('for (const log of mockSocialTaskLogs)')
    expect(deleteSource).toContain('if (Number(log.proxy_id) === numericId)')
    expect(deleteSource).toContain('log.proxy_id = null')
    expect(deleteSource).not.toContain('log.proxy_snapshot =')
  })

  it('serves user-scoped proxy CRUD, usable proxies, connectivity tests, and snapshot cleanup', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }

    const emptyList = await requestJson<MockPage<MockProxy>>(`${baseUrl}/api/v1/proxies?page=1&page_size=20`, { headers: auth })
    expect(emptyList.code).toBe(0)
    expect(Array.isArray(emptyList.data.items)).toBe(true)

    const rejected = await fetch(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: 1,
        name: 'bad proxy',
        ip_type: 'residential',
        endpoint: 'http://8.8.8.8:8080',
      }),
    })
    expect(rejected.status).toBe(400)
    const rejectedPayload = await rejected.json()
    expect(rejectedPayload.code).toBe('SOCIAL_IP_USER_ID_NOT_ACCEPTED')

    const rejectedBlankName = await fetch(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: '   ',
        ip_type: 'residential',
        endpoint: 'http://8.8.8.8:8080',
      }),
    })
    expect(rejectedBlankName.status).toBe(400)
    const rejectedBlankNamePayload = await rejectedBlankName.json()
    expect(rejectedBlankNamePayload.code).toBe('SOCIAL_IP_NAME_REQUIRED')

    const created = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'qa proxy',
        ip_type: 'residential',
        endpoint: 'http://8.8.8.8:8080',
        remark: 'initial',
      },
    })
    expect(created.data).toMatchObject({
      user_id: 2,
      name: 'qa proxy',
      ip_type: 'residential',
      endpoint: 'http://8.8.8.8:8080',
      status: 'unknown',
      remark: 'initial',
    })

    const filteredList = await requestJson<MockPage<MockProxy>>(`${baseUrl}/api/v1/proxies?search=qa&page=1&page_size=20`, { headers: auth })
    expect(filteredList.data.items.map((proxy) => proxy.id)).toContain(created.data.id)

    const testResult = await requestJson<Pick<MockProxy, 'id' | 'status' | 'latency_ms'>>(`${baseUrl}/api/v1/proxies/${created.data.id}/test`, {
      method: 'POST',
      headers: auth,
    })
    expect(testResult.data.id).toBe(created.data.id)
    expect(testResult.data.status).toBe('online')

    const usable = await requestJson<MockProxy[]>(`${baseUrl}/api/v1/proxies/usable`, { headers: auth })
    expect(usable.data.map((proxy) => proxy.id)).toContain(created.data.id)

    const accounts = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accounts.data.items[0]
    const accountWithProxy = await requestJson<MockSocialAccount>(`${baseUrl}/api/v1/accounts/${account.id}/default-proxy`, {
      method: 'PUT',
      headers: auth,
      body: { proxy_id: created.data.id },
    })
    expect(JSON.parse(accountWithProxy.data.default_proxy_snapshot)).toMatchObject({ id: created.data.id })
    expect(accountWithProxy.data.default_proxy_configured).toBe(true)

    const rejectedUpdate = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: 1,
        name: 'bad proxy owner update',
      }),
    })
    expect(rejectedUpdate.status).toBe(400)
    const rejectedUpdatePayload = await rejectedUpdate.json()
    expect(rejectedUpdatePayload.code).toBe('SOCIAL_IP_USER_ID_NOT_ACCEPTED')

    const rejectedBlankNameUpdate = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: '   ',
      }),
    })
    expect(rejectedBlankNameUpdate.status).toBe(400)
    const rejectedBlankNameUpdatePayload = await rejectedBlankNameUpdate.json()
    expect(rejectedBlankNameUpdatePayload.code).toBe('SOCIAL_IP_NAME_REQUIRED')

    const updated = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: auth,
      body: {
        name: 'qa proxy renamed',
        ip_type: 'static',
        endpoint: 'http://8.8.8.8:8080',
        remark: '',
      },
    })
    expect(updated.data).toMatchObject({
      id: created.data.id,
      name: 'qa proxy renamed',
      ip_type: 'static',
      endpoint: 'http://8.8.8.8:8080',
      status: 'online',
      remark: '',
    })

    const accountAfterUpdate = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    expect(JSON.parse(accountAfterUpdate.data.items[0].default_proxy_snapshot)).toMatchObject({
      id: created.data.id,
      name: 'qa proxy renamed',
      ip_type: 'static',
      endpoint: 'http://8.8.8.8:8080',
      status: 'online',
    })
    expect(accountAfterUpdate.data.items[0].default_proxy_configured).toBe(true)

    const endpointCleared = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: auth,
      body: {
        endpoint: '',
      },
    })
    expect(endpointCleared.data).toMatchObject({
      id: created.data.id,
      endpoint: '',
      status: 'unknown',
    })

    const accountAfterEndpointClear = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    expect(JSON.parse(accountAfterEndpointClear.data.items[0].default_proxy_snapshot)).toMatchObject({
      id: created.data.id,
      endpoint: '',
      status: 'unknown',
    })
    expect(accountAfterEndpointClear.data.items[0].default_proxy_configured).toBe(true)

    const deleted = await requestJson<null>(`${baseUrl}/api/v1/proxies/${created.data.id}`, { method: 'DELETE', headers: auth })
    expect(deleted.code).toBe(0)

    const accountAfterDelete = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    expect(accountAfterDelete.data.items[0].default_proxy_snapshot).toBe('')
    expect(accountAfterDelete.data.items[0].default_proxy_configured).toBe(false)

    const missingTest = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}/test`, { method: 'POST', headers: auth })
    expect(missingTest.status).toBe(404)
    await expect(missingTest.json()).resolves.toMatchObject({ code: 'SOCIAL_IP_NOT_FOUND' })

    const missingUpdate = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'deleted proxy' }),
    })
    expect(missingUpdate.status).toBe(404)
    await expect(missingUpdate.json()).resolves.toMatchObject({ code: 'SOCIAL_IP_NOT_FOUND' })

    const missingDelete = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}`, { method: 'DELETE', headers: auth })
    expect(missingDelete.status).toBe(404)
    await expect(missingDelete.json()).resolves.toMatchObject({ code: 'SOCIAL_IP_NOT_FOUND' })
  })

  it('uses backend-aligned safe errors for mock proxy tests without endpoints', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }

    const created = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'no endpoint proxy',
        ip_type: 'residential',
      },
    })
    expect(created.data).toMatchObject({
      name: 'no endpoint proxy',
      endpoint: '',
      status: 'unknown',
    })

    const singleResult = await requestJson<Pick<MockProxy, 'id' | 'status' | 'latency_ms' | 'error'>>(`${baseUrl}/api/v1/proxies/${created.data.id}/test`, {
      method: 'POST',
      headers: auth,
    })
    expect(singleResult.data).toMatchObject({
      id: created.data.id,
      status: 'unknown',
      latency_ms: 0,
      error: 'proxy endpoint is not ready for connectivity check',
    })
    expect(singleResult.data.error).not.toContain('no endpoint configured')

    const allResults = await requestJson<Array<Pick<MockProxy, 'id' | 'status' | 'latency_ms' | 'error'>>>(`${baseUrl}/api/v1/proxies/test`, {
      method: 'POST',
      headers: auth,
    })
    expect(allResults.data).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: created.data.id,
        status: 'unknown',
        latency_ms: 0,
        error: 'proxy endpoint is not ready for connectivity check',
      }),
    ]))
    expect(JSON.stringify(allResults.data)).not.toContain('no endpoint configured')
  })

  it('ignores proxy fields during user workbench import', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }

    const imported = await requestJson<{ accounts: MockSocialAccount[] }>(`${baseUrl}/api/v1/accounts/batch-import`, {
      method: 'POST',
      headers: auth,
      body: {
        accounts: [{
          platform: 'x_twitter',
          name: `@import_proxy_bypass_${Date.now()}`,
          password: 'account-secret',
          auth_cookie: 'auth_token=mock',
          proxy_id: 999,
          default_proxy_snapshot: '{"id":999,"endpoint":"http://bypass.example:8080","status":"online"}',
        }],
      },
    })

    expect(imported.code).toBe(0)
    expect(imported.data.accounts).toHaveLength(1)
    expect(imported.data.accounts[0].default_proxy_snapshot).toBe('')
    expect(imported.data.accounts[0].default_proxy_configured).toBe(false)
  })

  it('rejects invalid batch default-proxy assignment modes', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const accounts = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accounts.data.items[0]

    const response = await fetch(`${baseUrl}/api/v1/accounts/default-proxy`, {
      method: 'POST',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_ids: [account.id],
        mode: 'bogus',
        proxy_id: null,
      }),
    })
    const payload = await response.json()

    expect(response.status).toBe(400)
    expect(payload.code).toBe('SOCIAL_IP_ASSIGNMENT_MODE_INVALID')
  })

  it('rejects specific batch default-proxy assignment without proxy_id', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const accounts = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accounts.data.items[0]

    const response = await fetch(`${baseUrl}/api/v1/accounts/default-proxy`, {
      method: 'POST',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_ids: [account.id],
        mode: 'specific',
      }),
    })
    const payload = await response.json()

    expect(response.status).toBe(400)
    expect(payload.code).toBe('SOCIAL_IP_REQUIRED')
  })

  it('reports unavailable specific batch proxy assignment as row failures', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const accounts = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accounts.data.items[0]
    const proxy = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'untested batch proxy',
        ip_type: 'residential',
        endpoint: 'http://8.8.4.4:8080',
      },
    })
    expect(proxy.data.status).toBe('unknown')

    const result = await requestJson<MockBatchDefaultProxyResult>(`${baseUrl}/api/v1/accounts/default-proxy`, {
      method: 'POST',
      headers: auth,
      body: {
        account_ids: [account.id],
        mode: 'specific',
        proxy_id: proxy.data.id,
      },
    })

    expect(result.code).toBe(0)
    expect(result.data).toMatchObject({
      total: 1,
      succeeded: 0,
      skipped: 0,
      failed: 1,
    })
    expect(result.data.items).toEqual([
      expect.objectContaining({
        id: account.id,
        status: 'failed',
        reason: 'proxy_not_available',
      }),
    ])
  })

  it('uses the backend contract code when random batch assignment has no online proxy pool', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const usable = await requestJson<MockProxy[]>(`${baseUrl}/api/v1/proxies/usable`, { headers: auth })
    for (const proxy of usable.data) {
      await requestJson<null>(`${baseUrl}/api/v1/proxies/${proxy.id}`, { method: 'DELETE', headers: auth })
    }
    const accounts = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accounts.data.items[0]

    const response = await fetch(`${baseUrl}/api/v1/accounts/default-proxy`, {
      method: 'POST',
      headers: { ...auth, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_ids: [account.id],
        mode: 'random',
        proxy_id: null,
      }),
    })
    const payload = await response.json()

    expect(response.status).toBe(400)
    expect(payload.code).toBe('SOCIAL_IP_POOL_EMPTY')
  })

  it('tests all current-user proxies even when query filters are present', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const first = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'bulk test first',
        ip_type: 'residential',
        endpoint: 'http://8.8.8.8:8080',
      },
    })
    const second = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'bulk test second',
        ip_type: 'static',
        endpoint: 'http://1.1.1.1:8080',
      },
    })

    const result = await requestJson<Array<Pick<MockProxy, 'id' | 'status' | 'latency_ms'>>>(
      `${baseUrl}/api/v1/proxies/test?status=offline&search=definitely-no-match`,
      { method: 'POST', headers: auth },
    )

    expect(result.data.map((proxy) => proxy.id)).toEqual(expect.arrayContaining([first.data.id, second.data.id]))
    expect(result.data.map((proxy) => proxy.id)).toEqual([...result.data.map((proxy) => proxy.id)].sort((a, b) => a - b))
    expect(result.data.find((proxy) => proxy.id === first.data.id)?.status).toBe('online')
    expect(result.data.find((proxy) => proxy.id === second.data.id)?.status).toBe('online')
  })

  it('serves all current-user usable proxies independent of accidental query filters', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const first = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'usable first',
        ip_type: 'residential',
        endpoint: 'http://8.8.4.4:8080',
      },
    })
    const second = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'usable second',
        ip_type: 'static',
        endpoint: 'http://1.0.0.1:8080',
      },
    })
    await requestJson<Pick<MockProxy, 'id' | 'status' | 'latency_ms'>>(`${baseUrl}/api/v1/proxies/${first.data.id}/test`, {
      method: 'POST',
      headers: auth,
    })
    await requestJson<Pick<MockProxy, 'id' | 'status' | 'latency_ms'>>(`${baseUrl}/api/v1/proxies/${second.data.id}/test`, {
      method: 'POST',
      headers: auth,
    })

    const usable = await requestJson<MockProxy[]>(
      `${baseUrl}/api/v1/proxies/usable?status=offline&search=definitely-no-match`,
      { headers: auth },
    )

    expect(usable.data.map((proxy) => proxy.id)).toEqual(expect.arrayContaining([first.data.id, second.data.id]))
    expect(usable.data.map((proxy) => proxy.id)).toEqual([...usable.data.map((proxy) => proxy.id)].sort((a, b) => a - b))
    expect(usable.data.find((proxy) => proxy.id === first.data.id)).toMatchObject({ status: 'online', endpoint: 'http://8.8.4.4:8080' })
    expect(usable.data.find((proxy) => proxy.id === second.data.id)).toMatchObject({ status: 'online', endpoint: 'http://1.0.0.1:8080' })
  })

  it('reports duplicate and invalid account IDs in batch default-proxy assignment', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }
    const accountPage = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    const account = accountPage.data.items[0]
    const proxy = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies`, {
      method: 'POST',
      headers: auth,
      body: {
        name: 'batch proxy',
        ip_type: 'residential',
        endpoint: 'http://8.8.4.4:8080',
      },
    })
    await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies/${proxy.data.id}/test`, {
      method: 'POST',
      headers: auth,
    })

    const result = await requestJson<MockBatchDefaultProxyResult>(`${baseUrl}/api/v1/accounts/default-proxy`, {
      method: 'POST',
      headers: auth,
      body: {
        account_ids: [account.id, account.id, 0],
        mode: 'specific',
        proxy_id: proxy.data.id,
      },
    })

    expect(result.data).toMatchObject({
      total: 3,
      succeeded: 1,
      skipped: 1,
      failed: 1,
    })
    expect(result.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: account.id, status: 'succeeded' }),
      expect.objectContaining({ id: account.id, status: 'skipped', reason: 'duplicate_in_batch' }),
      expect.objectContaining({ id: 0, status: 'failed', reason: 'invalid_id' }),
    ]))

    const refreshed = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    expect(JSON.parse(refreshed.data.items[0].default_proxy_snapshot)).toMatchObject({ id: proxy.data.id })
    expect(refreshed.data.items[0].default_proxy_configured).toBe(true)
  })

  it('serves user-scoped task setting templates used by accounts', async () => {
    const baseUrl = await startMockApi()
    const auth = { Authorization: 'Bearer dev-mock-user-token' }

    const empty = await requestJson<MockTaskTemplate[]>(`${baseUrl}/api/v1/task-settings/templates`, { headers: auth })
    expect(empty.code).toBe(0)
    expect(empty.data).toEqual([])

    const invalid = await requestJson<{ valid: boolean; errors: string[] }>(`${baseUrl}/api/v1/task-settings/templates/validate`, {
      method: 'POST',
      headers: auth,
      body: { name: 'Empty follow', type: 'follow', params: { targets: [] } },
    })
    expect(invalid.data.valid).toBe(false)
    expect(invalid.data.errors.length).toBeGreaterThan(0)

    const saved = await requestJson<MockTaskTemplate>(`${baseUrl}/api/v1/task-settings/templates`, {
      method: 'POST',
      headers: auth,
      body: { name: 'Follow batch A', type: 'follow', params: { targets: ['@target'] }, is_default: true },
    })
    expect(saved.data).toMatchObject({
      name: 'Follow batch A',
      type: 'follow',
      params: { targets: ['@target'] },
      is_default: true,
    })

    const listed = await requestJson<MockTaskTemplate[]>(`${baseUrl}/api/v1/task-settings/templates`, { headers: auth })
    expect(listed.data.map((template) => template.id)).toContain(saved.data.id)
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

async function requestJson<T>(url: string, options: { method?: string; body?: Record<string, unknown>; headers?: Record<string, string> } = {}): Promise<MockEnvelope<T>> {
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
