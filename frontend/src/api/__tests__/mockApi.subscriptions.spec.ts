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

interface MockSubscriptionProgressWindow {
  limit_usd: number
  used_usd: number
  remaining_usd: number
  percentage: number
  window_start: string
  resets_at: string
  resets_in_seconds: number
}

interface MockSubscriptionProgress {
  id: number
  group_name: string
  expires_at: string
  expires_in_days: number
  daily?: MockSubscriptionProgressWindow | null
  weekly?: MockSubscriptionProgressWindow | null
  monthly?: MockSubscriptionProgressWindow | null
}

interface MockSubscriptionProgressEntry {
  subscription: {
    id: number
    quota_usd: number | null
    daily_limit_usd: number | null
    weekly_limit_usd: number | null
    monthly_limit_usd: number | null
  }
  progress: MockSubscriptionProgress
}

interface MockSubscriptionSummary {
  active_count: number
  subscriptions: Array<{
    id: number
    group_name: string
    status: string
    daily_progress: number | null
    weekly_progress: number | null
    monthly_progress: number | null
    expires_at: string | null
    days_remaining: number | null
    user_id?: number
  }>
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

describe('mock API user subscription contract', () => {
  it('serves the existing progress and summary endpoints with backend-compatible shapes', async () => {
    const baseUrl = await startMockApi()

    const progress = await requestJson<MockSubscriptionProgressEntry[]>(`${baseUrl}/api/v1/subscriptions/progress`)
    expect(progress.code).toBe(0)
    expect(progress.data.length).toBeGreaterThan(0)
    expect(progress.data[0]).toMatchObject({
      subscription: expect.objectContaining({
        id: expect.any(Number),
        daily_limit_usd: expect.any(Number),
        weekly_limit_usd: expect.any(Number),
        monthly_limit_usd: expect.any(Number),
      }),
      progress: expect.objectContaining({
        id: expect.any(Number),
        group_name: expect.any(String),
        expires_at: expect.any(String),
        expires_in_days: expect.any(Number),
      }),
    })
    expect(progress.data[0].progress.monthly).toMatchObject({
      limit_usd: expect.any(Number),
      used_usd: expect.any(Number),
      remaining_usd: expect.any(Number),
      resets_in_seconds: expect.any(Number),
    })
    expect(progress.data[0].progress.monthly).not.toHaveProperty('limit')
    expect(progress.data[0].progress.monthly).not.toHaveProperty('reset_in_seconds')

    const summary = await requestJson<MockSubscriptionSummary>(`${baseUrl}/api/v1/subscriptions/summary`)
    expect(summary.code).toBe(0)
    expect(summary.data.active_count).toBe(progress.data.length)
    expect(summary.data.subscriptions[0]).toMatchObject({
      id: progress.data[0].subscription.id,
      group_name: progress.data[0].progress.group_name,
      status: 'active',
      expires_at: expect.any(String),
      days_remaining: expect.any(Number),
    })
    expect(summary.data.subscriptions[0]).not.toHaveProperty('user_id')
  })

  it('does not serve the removed user per-subscription progress endpoint', async () => {
    const baseUrl = await startMockApi()

    const response = await fetch(`${baseUrl}/api/v1/subscriptions/1000/progress`)

    expect(response.status).toBe(404)
  })

  it('serves the existing admin per-subscription progress endpoint', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const progress = await requestJson<MockSubscriptionProgress>(`${baseUrl}/api/v1/admin/subscriptions/1000/progress`, {
      headers: adminHeaders,
    })

    expect(progress.code).toBe(0)
    expect(progress.data).toMatchObject({
      id: 1000,
      group_name: expect.any(String),
      expires_at: expect.any(String),
      expires_in_days: expect.any(Number),
    })
    expect(progress.data.monthly).toMatchObject({
      limit_usd: expect.any(Number),
      used_usd: expect.any(Number),
      remaining_usd: expect.any(Number),
      resets_in_seconds: expect.any(Number),
    })
  })

  it('does not serve removed admin subscription assign endpoints', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    for (const path of ['/api/v1/admin/subscriptions/assign', '/api/v1/admin/subscriptions/bulk-assign']) {
      const response = await fetch(`${baseUrl}${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...adminHeaders },
        body: JSON.stringify({ user_id: 1, user_ids: [1], plan_id: 1, validity_days: 30 }),
      })

      expect(response.status).toBe(404)
      await expect(response.json()).resolves.toMatchObject({
        code: 'MOCK_NOT_FOUND',
      })
    }
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

async function requestJson<T>(url: string, options: { headers?: Record<string, string> } = {}): Promise<MockEnvelope<T>> {
  const response = await fetch(url, {
    headers: options.headers,
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
