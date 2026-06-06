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

interface MockUserAuthBinding {
  provider: string
  bound: boolean
  bound_count: number
  display_name?: string
  subject_hint?: string
  can_bind: boolean
  can_unbind: boolean
  note_key?: string
}

interface MockProfileUser {
  id: number
  email: string
  username: string
  run_mode?: string
  email_bound: boolean
  linuxdo_bound: boolean
  oidc_bound: boolean
  wechat_bound: boolean
  dingtalk_bound: boolean
  identities: {
    email: MockUserAuthBinding
    linuxdo: MockUserAuthBinding
    oidc: MockUserAuthBinding
    wechat: MockUserAuthBinding
    dingtalk: MockUserAuthBinding
  }
  auth_bindings: Record<string, MockUserAuthBinding>
  identity_bindings: Record<string, MockUserAuthBinding>
  balance_notify_enabled: boolean
  balance_notify_threshold: number | null
  balance_notify_extra_emails: Array<{
    email: string
    disabled: boolean
    verified: boolean
  }>
}

interface MockTotpStatus {
  enabled: boolean
  enabled_at: number | null
  feature_enabled: boolean
}

interface MockPublicSettings {
  totp_enabled: boolean
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

describe('mock API auth and profile contract', () => {
  it('serves auth/me and user/profile with backend-compatible profile shapes', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const publicSettings = await requestJson<MockPublicSettings>(`${baseUrl}/api/v1/settings/public`)
    expect(publicSettings.code).toBe(0)
    expect(publicSettings.data.totp_enabled).toBe(false)

    const currentUser = await requestJson<MockProfileUser>(`${baseUrl}/api/v1/auth/me`, {
      headers: userHeaders,
    })
    expect(currentUser.code).toBe(0)
    expect(currentUser.data.run_mode).toBe('standard')
    expect(currentUser.data.identities.email.bound).toBe(true)
    expect(currentUser.data.auth_bindings.email).toMatchObject({
      provider: 'email',
      bound: true,
      can_bind: false,
    })
    expect(currentUser.data.identity_bindings.email.bound).toBe(true)
    expect(currentUser.data.email_bound).toBe(true)
    expect(currentUser.data.dingtalk_bound).toBe(false)

    const profile = await requestJson<MockProfileUser>(`${baseUrl}/api/v1/user/profile`, {
      headers: userHeaders,
    })
    expect(profile.code).toBe(0)
    expect(profile.data.run_mode).toBeUndefined()
    expect(profile.data.identities.email.bound).toBe(true)
    expect(profile.data.auth_bindings.email.note_key).toBe(
      'profile.authBindings.notes.emailManagedFromProfile'
    )
  })

  it('serves existing profile update, email binding, notify email, and TOTP routes', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }

    const updated = await requestJson<MockProfileUser>(`${baseUrl}/api/v1/user`, {
      method: 'PUT',
      headers: userHeaders,
      body: {
        username: 'operator-updated',
        balance_notify_enabled: true,
        balance_notify_threshold: 12.5,
      },
    })
    expect(updated.code).toBe(0)
    expect(updated.data.username).toBe('operator-updated')
    expect(updated.data.balance_notify_enabled).toBe(true)
    expect(updated.data.balance_notify_threshold).toBe(12.5)
    expect(updated.data.auth_bindings.email.bound).toBe(true)

    await expect(
      requestJson<{ message: string }>(`${baseUrl}/api/v1/user/account-bindings/email/send-code`, {
        method: 'POST',
        headers: userHeaders,
        body: { email: 'operator-bound@example.test' },
      })
    ).resolves.toMatchObject({ code: 0 })

    const bound = await requestJson<MockProfileUser>(`${baseUrl}/api/v1/user/account-bindings/email`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        email: 'operator-bound@example.test',
        verify_code: '123456',
        password: 'new-password',
      },
    })
    expect(bound.code).toBe(0)
    expect(bound.data.email).toBe('operator-bound@example.test')
    expect(bound.data.email_bound).toBe(true)
    expect(bound.data.auth_bindings.email.display_name).toBe('operator-bound@example.test')

    await expect(
      requestJson<{ message: string }>(`${baseUrl}/api/v1/user/notify-email/send-code`, {
        method: 'POST',
        headers: userHeaders,
        body: { email: 'notify@example.test' },
      })
    ).resolves.toMatchObject({ code: 0 })

    const verifiedNotifyEmail = await requestJson<MockProfileUser>(
      `${baseUrl}/api/v1/user/notify-email/verify`,
      {
        method: 'POST',
        headers: userHeaders,
        body: { email: 'notify@example.test', code: '123456' },
      }
    )
    expect(verifiedNotifyEmail.data.balance_notify_extra_emails).toContainEqual({
      email: 'notify@example.test',
      disabled: false,
      verified: true,
    })

    const toggledNotifyEmail = await requestJson<MockProfileUser>(
      `${baseUrl}/api/v1/user/notify-email/toggle`,
      {
        method: 'PUT',
        headers: userHeaders,
        body: { email: 'notify@example.test', disabled: true },
      }
    )
    expect(toggledNotifyEmail.data.balance_notify_extra_emails).toContainEqual({
      email: 'notify@example.test',
      disabled: true,
      verified: true,
    })

    const removedNotifyEmail = await requestJson<MockProfileUser>(
      `${baseUrl}/api/v1/user/notify-email`,
      {
        method: 'DELETE',
        headers: userHeaders,
        body: { email: 'notify@example.test' },
      }
    )
    expect(removedNotifyEmail.data.balance_notify_extra_emails).toEqual([])

    const totpStatus = await requestJson<MockTotpStatus>(`${baseUrl}/api/v1/user/totp/status`, {
      headers: userHeaders,
    })
    expect(totpStatus.data).toEqual({
      enabled: false,
      enabled_at: null,
      feature_enabled: false,
    })

    const totpMethod = await requestJson<{ method: 'email' | 'password' }>(
      `${baseUrl}/api/v1/user/totp/verification-method`,
      {
        headers: userHeaders,
      }
    )
    expect(totpMethod.data.method).toBe('password')

    const disabledSetup = await requestRaw(`${baseUrl}/api/v1/user/totp/setup`, {
      method: 'POST',
      headers: userHeaders,
      body: { password: 'new-password' },
    })
    expect(disabledSetup.status).toBe(400)
    expect(disabledSetup.body.code).toBe('TOTP_NOT_ENABLED')
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

async function requestRaw(
  url: string,
  options: { method?: string; headers?: Record<string, string>; body?: Record<string, unknown> } = {}
): Promise<{ status: number; body: { code?: string; message?: string } }> {
  const response = await fetch(url, {
    method: options.method ?? 'GET',
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.headers ?? {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  })
  return {
    status: response.status,
    body: (await response.json()) as { code?: string; message?: string },
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
