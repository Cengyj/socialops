import { afterEach, describe, expect, it } from 'vitest'
import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process'
import { once } from 'node:events'
import { createServer } from 'node:net'
import { resolve } from 'node:path'

interface MockEnvelope<T> {
  code: number | string
  message: string
  data: T
}

interface MockPublicSettings {
  site_name: string
  payment_enabled: boolean
  purchase_subscription_enabled: boolean
  purchase_subscription_url: string
  force_email_on_third_party_signup: boolean
  custom_menu_items: Array<{ id: string; visibility: string }>
  smtp_password?: string
  turnstile_secret_key?: string
  linuxdo_connect_client_secret?: string
}

interface MockEmailTemplateEvent {
  value: string
  label: string
  category: string
  optional?: boolean
  placeholders?: string[]
}

interface MockEmailTemplateList {
  events: MockEmailTemplateEvent[]
  locales: string[]
  placeholders: string[]
}

function removedGatewaySettingKey(parts: string[]): string {
  return parts.join('_')
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

describe('mock API settings contract', () => {
  it('keeps admin-updated public settings in sync without exposing system secrets', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const updated = await requestJson<Record<string, unknown>>(`${baseUrl}/api/v1/admin/settings`, {
      method: 'PUT',
      headers: adminHeaders,
      body: {
        site_name: 'Mock Operations Console',
        payment_enabled: false,
        purchase_subscription_enabled: false,
        purchase_subscription_url: '/purchase/internal',
        force_email_on_third_party_signup: true,
        smtp_password: 'smtp-secret',
        turnstile_secret_key: 'turnstile-secret',
        linuxdo_connect_client_secret: 'linuxdo-secret',
        custom_menu_items: [
          { id: 'user-page', label: 'User Page', url: 'md:user', visibility: 'user', sort_order: 1 },
          { id: 'admin-page', label: 'Admin Page', url: 'md:admin', visibility: 'admin', sort_order: 2 },
        ],
      },
    })
    expect(updated.code).toBe(0)
    expect(updated.data).not.toHaveProperty('smtp_password')
    expect(updated.data).not.toHaveProperty('turnstile_secret_key')
    expect(updated.data).not.toHaveProperty('linuxdo_connect_client_secret')
    expect(updated.data).not.toHaveProperty(removedGatewaySettingKey(['api', 'key', 'acl', 'trust', 'forwarded', 'ip']))
    expect(updated.data).not.toHaveProperty('risk_control_enabled')

    const publicSettings = await requestJson<MockPublicSettings>(`${baseUrl}/api/v1/settings/public`)
    expect(publicSettings.code).toBe(0)
    expect(publicSettings.data).toMatchObject({
      site_name: 'Mock Operations Console',
      payment_enabled: false,
      purchase_subscription_enabled: false,
      purchase_subscription_url: '/purchase/internal',
      force_email_on_third_party_signup: true,
    })
    expect(publicSettings.data.custom_menu_items).toEqual([
      expect.objectContaining({ id: 'user-page', visibility: 'user' }),
    ])
    expect(publicSettings.data).not.toHaveProperty('smtp_password')
    expect(publicSettings.data).not.toHaveProperty('turnstile_secret_key')
    expect(publicSettings.data).not.toHaveProperty('linuxdo_connect_client_secret')
    expect(publicSettings.data).not.toHaveProperty(removedGatewaySettingKey(['api', 'key', 'acl', 'trust', 'forwarded', 'ip']))
    expect(publicSettings.data).not.toHaveProperty('risk_control_enabled')
  })

  it('exposes only current notification email template events in the mock API', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const response = await requestJson<MockEmailTemplateList>(
      `${baseUrl}/api/v1/admin/settings/email-templates`,
      { headers: adminHeaders },
    )

    expect(response.code).toBe(0)
    expect(response.data.events.map((event) => event.value)).toEqual([
      'auth.verify_code',
      'auth.password_reset',
      'notification_email.verify_code',
      'subscription.purchase_success',
      'subscription.expiry_reminder',
      'balance.low',
      'balance.recharge_success',
    ])
    expect(response.data.events.find((event) => event.value === 'balance.low')).toMatchObject({
      category: 'billing',
      optional: true,
      placeholders: expect.arrayContaining(['current_balance', 'balance', 'threshold']),
    })
    expect(response.data.placeholders).toEqual(
      expect.arrayContaining([
        'verification_code',
        'expires_in_minutes',
        'reset_url',
        'subscription_group',
        'plan_name',
        'expires_at',
        'current_balance',
        'recharge_amount',
      ])
    )
    expect(response.data.events.map((event) => event.value)).not.toEqual(
      expect.arrayContaining([
        'account.quota_alert',
        'content_moderation.violation_notice',
        'content_moderation.account_disabled',
        'ops.alert',
        'ops.scheduled_report',
      ])
    )

    const unsupported = await requestJson<Record<string, unknown>>(
      `${baseUrl}/api/v1/admin/settings/email-templates/ops.alert/en-US`,
      { headers: adminHeaders },
    )
    expect(unsupported.code).toBe('EMAIL_TEMPLATE_EVENT_UNSUPPORTED')
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
