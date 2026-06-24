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

interface MockRedeemCode {
  id: number
  code: string
  type: string
  value: number
  status: string
  used_by: number | null
  used_at: string | null
  created_at: string
  expires_at?: string | null
  group_id?: number | null
  plan_id?: number | null
  validity_days?: number
}

interface MockRedeemStats {
  total_codes: number
  active_codes: number
  unused_codes: number
  used_codes: number
  expired_codes: number
  disabled_codes: number
  total_value_distributed: number
  total_value: number
  by_type: Record<string, number>
}

interface MockPromoCode {
  id: number
  code: string
  bonus_amount: number
  max_uses: number
  used_count: number
  status: string
  expires_at: string | null
  notes: string | null
  created_at: string
  updated_at: string
}

interface MockPromoUsage {
  id: number
  promo_code_id: number
  user_id: number
  bonus_amount: number
  used_at: string
  user?: {
    id: number
    email: string
  }
}

interface MockAffiliateDetail {
  user_id: number
  aff_code: string
  aff_count: number
  aff_quota: number
  aff_frozen_quota: number
  aff_history_quota: number
  effective_rebate_rate_percent: number
  invitees: Array<{
    user_id: number
    email: string
    username: string
    total_rebate: number
  }>
}

interface MockAffiliateTransfer {
  transferred_quota: number
  balance: number
}

interface MockAffiliateEntry {
  user_id: number
  email: string
  username: string
  aff_code: string
  aff_code_custom: boolean
  aff_rebate_rate_percent?: number | null
  aff_count: number
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

describe('mock API redeem, promo, and affiliate contract', () => {
  it('serves the existing user and admin redeem-code endpoints with backend-compatible shapes', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const generated = await requestJson<MockRedeemCode[]>(`${baseUrl}/api/v1/admin/redeem-codes/generate`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        count: 2,
        type: 'balance',
        value: 12.5,
        expires_in_days: 7,
      },
    })
    expect(generated.code).toBe(0)
    expect(generated.data).toHaveLength(2)
    expect(generated.data[0]).toMatchObject({
      id: expect.any(Number),
      code: expect.any(String),
      type: 'balance',
      value: 12.5,
      status: 'unused',
      used_by: null,
      used_at: null,
      expires_at: expect.any(String),
    })

    const redeemed = await requestJson<MockRedeemCode>(`${baseUrl}/api/v1/redeem`, {
      method: 'POST',
      headers: userHeaders,
      body: { code: generated.data[0].code },
    })
    expect(redeemed.code).toBe(0)
    expect(redeemed.data).toMatchObject({
      id: generated.data[0].id,
      status: 'used',
      used_by: 2,
      used_at: expect.any(String),
    })

    const history = await requestJson<MockRedeemCode[]>(`${baseUrl}/api/v1/redeem/history`, {
      headers: userHeaders,
    })
    expect(history.code).toBe(0)
    expect(history.data.map((item) => item.id)).toContain(generated.data[0].id)

    const stats = await requestJson<MockRedeemStats>(`${baseUrl}/api/v1/admin/redeem-codes/stats`, {
      headers: adminHeaders,
    })
    expect(stats.code).toBe(0)
    expect(stats.data).toMatchObject({
      total_codes: 2,
      active_codes: 1,
      unused_codes: 1,
      used_codes: 1,
      expired_codes: 0,
      disabled_codes: 0,
      total_value_distributed: 12.5,
      total_value: 25,
    })
    expect(stats.data.by_type.balance).toBe(2)

    const expired = await requestJson<MockRedeemCode>(`${baseUrl}/api/v1/admin/redeem-codes/${generated.data[1].id}/expire`, {
      method: 'POST',
      headers: adminHeaders,
    })
    expect(expired.code).toBe(0)
    expect(expired.data.status).toBe('expired')

    const batchUpdate = await requestJson<{ updated: number; message: string }>(`${baseUrl}/api/v1/admin/redeem-codes/batch-update`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        ids: [generated.data[1].id],
        fields: { notes: 'maintenance' },
      },
    })
    expect(batchUpdate.data.updated).toBe(1)

    const detail = await requestJson<MockRedeemCode>(`${baseUrl}/api/v1/admin/redeem-codes/${generated.data[1].id}`, {
      headers: adminHeaders,
    })
    expect(detail.data).toMatchObject({ id: generated.data[1].id, status: 'expired' })

    const exported = await fetch(`${baseUrl}/api/v1/admin/redeem-codes/export`, {
      headers: adminHeaders,
    })
    expect(exported.status).toBe(200)
    expect(exported.headers.get('content-type')).toContain('text/csv')
    expect(await exported.text()).toContain('id,code,type,value,status,used_by,used_by_email,used_at,expires_at,created_at')
  })

  it('serves the existing promo-code validation, CRUD, and usage endpoints', async () => {
    const baseUrl = await startMockApi()
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const disabledValidation = await requestJson<{ valid: boolean; error_code: string }>(
      `${baseUrl}/api/v1/auth/validate-promo-code`,
      { method: 'POST', body: { code: 'WELCOME' } },
    )
    expect(disabledValidation.code).toBe(0)
    expect(disabledValidation.data).toMatchObject({
      valid: false,
      error_code: 'PROMO_CODE_DISABLED',
    })

    const created = await requestJson<MockPromoCode>(`${baseUrl}/api/v1/admin/promo-codes`, {
      method: 'POST',
      headers: adminHeaders,
      body: {
        code: 'welcome',
        bonus_amount: 5,
        max_uses: 2,
        expires_at: Math.floor((Date.now() + 86400000) / 1000),
        notes: 'launch',
      },
    })
    expect(created.code).toBe(0)
    expect(created.data).toMatchObject({
      code: 'WELCOME',
      bonus_amount: 5,
      max_uses: 2,
      used_count: 0,
      status: 'active',
      notes: 'launch',
    })

    const list = await requestJson<MockPage<MockPromoCode>>(`${baseUrl}/api/v1/admin/promo-codes?search=wel&page=1&page_size=20`, {
      headers: adminHeaders,
    })
    expect(list.code).toBe(0)
    expect(list.data.items.map((item) => item.id)).toContain(created.data.id)

    const updated = await requestJson<MockPromoCode>(`${baseUrl}/api/v1/admin/promo-codes/${created.data.id}`, {
      method: 'PUT',
      headers: adminHeaders,
      body: {
        expires_at: 0,
        status: 'disabled',
        notes: '',
      },
    })
    expect(updated.data).toMatchObject({
      id: created.data.id,
      status: 'disabled',
      expires_at: null,
      notes: '',
    })

    const usages = await requestJson<MockPage<MockPromoUsage>>(`${baseUrl}/api/v1/admin/promo-codes/${created.data.id}/usages`, {
      headers: adminHeaders,
    })
    expect(usages.code).toBe(0)
    expect(usages.data).toMatchObject({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const deleted = await requestJson<{ message: string }>(`${baseUrl}/api/v1/admin/promo-codes/${created.data.id}`, {
      method: 'DELETE',
      headers: adminHeaders,
    })
    expect(deleted.code).toBe(0)
    expect(deleted.data.message).toBeTruthy()
  })

  it('serves the existing user and admin affiliate endpoints', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const detail = await requestJson<MockAffiliateDetail>(`${baseUrl}/api/v1/user/aff`, {
      headers: userHeaders,
    })
    expect(detail.code).toBe(0)
    expect(detail.data).toMatchObject({
      user_id: 2,
      aff_code: expect.any(String),
      aff_quota: expect.any(Number),
      aff_frozen_quota: expect.any(Number),
      aff_history_quota: expect.any(Number),
      effective_rebate_rate_percent: expect.any(Number),
      invitees: expect.any(Array),
    })

    const transferred = await requestJson<MockAffiliateTransfer>(`${baseUrl}/api/v1/user/aff/transfer`, {
      method: 'POST',
      headers: userHeaders,
    })
    expect(transferred.code).toBe(0)
    expect(transferred.data).toMatchObject({
      transferred_quota: 3.5,
      balance: expect.any(Number),
    })

    const lookup = await requestJson<Array<{ id: number; email: string }>>(`${baseUrl}/api/v1/admin/affiliates/users/lookup?q=operator`, {
      headers: adminHeaders,
    })
    expect(lookup.code).toBe(0)
    expect(lookup.data.map((user) => user.id)).toContain(2)

    const updated = await requestJson<{ user_id: number }>(`${baseUrl}/api/v1/admin/affiliates/users/2`, {
      method: 'PUT',
      headers: adminHeaders,
      body: {
        aff_code: 'vip2026',
        aff_rebate_rate_percent: 12.5,
      },
    })
    expect(updated.data.user_id).toBe(2)

    const customUsers = await requestJson<MockPage<MockAffiliateEntry>>(`${baseUrl}/api/v1/admin/affiliates/users?search=vip&page=1&page_size=20`, {
      headers: adminHeaders,
    })
    expect(customUsers.data.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        user_id: 2,
        aff_code: 'VIP2026',
        aff_code_custom: true,
        aff_rebate_rate_percent: 12.5,
      }),
    ]))

    const overview = await requestJson<{ user_id: number; aff_code: string; rebate_rate_percent: number }>(
      `${baseUrl}/api/v1/admin/affiliates/users/2/overview`,
      { headers: adminHeaders },
    )
    expect(overview.data).toMatchObject({
      user_id: 2,
      aff_code: 'VIP2026',
      rebate_rate_percent: 12.5,
    })

    for (const path of ['invites', 'rebates', 'transfers']) {
      const records = await requestJson<MockPage<Record<string, unknown>>>(`${baseUrl}/api/v1/admin/affiliates/${path}?page=1&page_size=20`, {
        headers: adminHeaders,
      })
      expect(records.code).toBe(0)
      expect(records.data).toMatchObject({
        items: expect.any(Array),
        total: expect.any(Number),
        page: 1,
        page_size: 20,
      })
    }

    const batchRate = await requestJson<{ affected: number }>(`${baseUrl}/api/v1/admin/affiliates/users/batch-rate`, {
      method: 'POST',
      headers: adminHeaders,
      body: { user_ids: [2], clear: true },
    })
    expect(batchRate.data.affected).toBe(1)

    const cleared = await requestJson<{ user_id: number }>(`${baseUrl}/api/v1/admin/affiliates/users/2`, {
      method: 'DELETE',
      headers: adminHeaders,
    })
    expect(cleared.data.user_id).toBe(2)
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
  options: { method?: string; headers?: Record<string, string>; body?: Record<string, unknown> } = {},
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
