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

interface MockPage<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

interface MockAnnouncement {
  id: number
  title: string
  content: string
  status: 'draft' | 'active' | 'archived'
  notify_mode: 'silent' | 'popup'
  targeting: { any_of?: unknown[] }
  read_at?: string | null
}

interface MockReadStatus {
  user_id: number
  email: string
  username: string
  balance: number
  eligible: boolean
  read_at?: string | null
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

describe('mock API announcements contract', () => {
  it('keeps read state and targeting aligned with backend announcement APIs', async () => {
    const baseUrl = await startMockApi()

    const created = await requestJson<MockAnnouncement>(`${baseUrl}/api/v1/admin/announcements`, {
      method: 'POST',
      body: {
        title: 'Announcement contract',
        content: 'Contract body',
        status: 'active',
        notify_mode: 'popup',
        targeting: { any_of: [] },
      },
    })
    expect(created.code).toBe(0)
    expect(created.data.targeting).toEqual({ any_of: [] })

    const list = await requestJson<MockAnnouncement[]>(`${baseUrl}/api/v1/announcements`, {
      token: 'dev-mock-user-token',
    })
    const announcement = list.data.find((item) => item.id === created.data.id)
    expect(announcement).toMatchObject({
      id: created.data.id,
      notify_mode: 'popup',
    })

    const marked = await requestJson<{ message: string }>(
      `${baseUrl}/api/v1/announcements/${created.data.id}/read`,
      { method: 'POST', token: 'dev-mock-user-token' },
    )
    expect(marked.code).toBe(0)

    const unread = await requestJson<MockAnnouncement[]>(
      `${baseUrl}/api/v1/announcements?unread_only=1`,
      { token: 'dev-mock-user-token' },
    )
    expect(unread.data.some((item) => item.id === created.data.id)).toBe(false)

    const readStatus = await requestJson<MockPage<MockReadStatus>>(
      `${baseUrl}/api/v1/admin/announcements/${created.data.id}/read-status`,
    )
    expect(readStatus.code).toBe(0)
    expect(readStatus.data.items).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          user_id: 2,
          email: 'operator@example.test',
          balance: expect.any(Number),
          eligible: true,
          read_at: expect.any(String),
        }),
      ]),
    )
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
  options: { method?: string; body?: Record<string, unknown>; token?: string } = {},
): Promise<MockEnvelope<T>> {
  const response = await fetch(url, {
    method: options.method ?? 'GET',
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
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
