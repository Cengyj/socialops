import { afterEach, describe, expect, it } from 'vitest'
import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process'
import { once } from 'node:events'
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

    const updated = await requestJson<MockProxy>(`${baseUrl}/api/v1/proxies/${created.data.id}`, {
      method: 'PUT',
      headers: auth,
      body: {
        name: 'qa proxy renamed',
        ip_type: 'static',
        endpoint: '',
        remark: '',
      },
    })
    expect(updated.data).toMatchObject({
      id: created.data.id,
      name: 'qa proxy renamed',
      ip_type: 'static',
      endpoint: '',
      status: 'unknown',
      remark: '',
    })

    const deleted = await requestJson<null>(`${baseUrl}/api/v1/proxies/${created.data.id}`, { method: 'DELETE', headers: auth })
    expect(deleted.code).toBe(0)

    const accountAfterDelete = await requestJson<MockPage<MockSocialAccount>>(`${baseUrl}/api/v1/accounts?page=1&page_size=1`, { headers: auth })
    expect(accountAfterDelete.data.items[0].default_proxy_snapshot).toBe('')
    expect(accountAfterDelete.data.items[0].default_proxy_configured).toBe(false)

    const missingTest = await fetch(`${baseUrl}/api/v1/proxies/${created.data.id}/test`, { method: 'POST', headers: auth })
    expect(missingTest.status).toBe(404)
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
