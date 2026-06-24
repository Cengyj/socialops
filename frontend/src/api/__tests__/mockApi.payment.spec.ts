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

interface MockPaymentOrder {
  id: number
  amount: number
  pay_amount: number
  fee_rate: number
  currency: string
  payment_type: string
  out_trade_no: string
  status: string
  order_type: string
  created_at: string
  expires_at: string
  refund_amount: number
  provider_instance_id?: string
  user_email?: string
  payment_trade_no?: string
}

interface MockCreateOrderResult {
  order_id: number
  amount: number
  pay_amount: number
  fee_rate: number
  expires_at: string
  payment_type: string
  out_trade_no: string
  currency: string
}

interface MockPaymentConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  daily_limit: number
  max_pending_orders: number
  order_timeout_minutes: number
  balance_disabled: boolean
  balance_recharge_multiplier: number
  recharge_fee_rate: number
  load_balance_strategy: string
  enabled_payment_types: string[]
  cancel_rate_limit_enabled: boolean
  cancel_rate_limit_max: number
  cancel_rate_limit_window: number
  cancel_rate_limit_unit: string
  cancel_rate_limit_window_mode: string
  alipay_force_qrcode: boolean
}

interface MockPaymentDashboard {
  today_amount: number
  total_amount: number
  today_count: number
  total_count: number
  avg_amount: number
  pending_orders: number
  daily_series: Array<{ date: string; amount: number; count: number }>
  payment_methods: Array<{ type: string; amount: number; count: number }>
  top_users: Array<{ user_id: number; email: string; amount: number }>
}

interface MockAdminOrderDetail {
  order: MockPaymentOrder
  auditLogs: Array<{ id: number; action: string; detail: string | null; operator: string | null; created_at: string }>
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

describe('mock API payment contract', () => {
  it('serves backend-compatible user and admin payment order shapes', async () => {
    const baseUrl = await startMockApi()
    const userHeaders = { Authorization: 'Bearer dev-mock-user-token' }
    const adminHeaders = { Authorization: 'Bearer dev-mock-admin-token' }

    const userConfig = await requestJson<MockPaymentConfig>(`${baseUrl}/api/v1/payment/config`, {
      headers: userHeaders,
    })
    expect(userConfig.code).toBe(0)
    expect(userConfig.data).toMatchObject({
      enabled: true,
      min_amount: 1,
      max_amount: 99999,
      daily_limit: 99999,
      max_pending_orders: 10,
      order_timeout_minutes: 30,
      balance_disabled: false,
      balance_recharge_multiplier: 1,
      recharge_fee_rate: 0,
      load_balance_strategy: 'round-robin',
      enabled_payment_types: ['alipay', 'wxpay'],
      cancel_rate_limit_enabled: false,
      cancel_rate_limit_max: 10,
      cancel_rate_limit_window: 1,
      cancel_rate_limit_unit: 'day',
      cancel_rate_limit_window_mode: 'rolling',
      alipay_force_qrcode: false,
    })
    expect(userConfig.data).not.toHaveProperty('payment_enabled')

    const removedChannels = await fetch(`${baseUrl}/api/v1/payment/channels`, {
      headers: userHeaders,
    })
    expect(removedChannels.status).toBe(404)
    await expect(removedChannels.json()).resolves.toMatchObject({
      code: 'MOCK_NOT_FOUND',
    })

    const adminConfig = await requestJson<MockPaymentConfig>(`${baseUrl}/api/v1/admin/payment/config`, {
      headers: adminHeaders,
    })
    expect(adminConfig.code).toBe(0)
    expect(adminConfig.data).toMatchObject({
      enabled: true,
      recharge_fee_rate: 0,
      cancel_rate_limit_enabled: false,
      alipay_force_qrcode: false,
    })
    expect(adminConfig.data).not.toHaveProperty('payment_enabled')

    const created = await requestJson<MockCreateOrderResult>(`${baseUrl}/api/v1/payment/orders`, {
      method: 'POST',
      headers: userHeaders,
      body: {
        amount: 12.5,
        payment_type: 'alipay',
        order_type: 'balance',
      },
    })
    expect(created.code).toBe(0)
    expect(created.data).toMatchObject({
      order_id: expect.any(Number),
      amount: 12.5,
      pay_amount: 12.5,
      fee_rate: 0,
      payment_type: 'alipay',
      out_trade_no: expect.any(String),
      currency: 'CNY',
    })

    const userOrder = await requestJson<MockPaymentOrder>(`${baseUrl}/api/v1/payment/orders/${created.data.order_id}`, {
      headers: userHeaders,
    })
    expect(userOrder.code).toBe(0)
    expect(userOrder.data).toMatchObject({
      id: created.data.order_id,
      user_id: 2,
      out_trade_no: created.data.out_trade_no,
      status: 'PENDING',
      provider_instance_id: expect.any(String),
    })
    expect(userOrder.data).not.toHaveProperty('user_email')
    expect(userOrder.data).not.toHaveProperty('payment_trade_no')

    const dashboard = await requestJson<MockPaymentDashboard>(`${baseUrl}/api/v1/admin/payment/dashboard?days=30`, {
      headers: adminHeaders,
    })
    expect(dashboard.code).toBe(0)
    expect(dashboard.data).toMatchObject({
      today_amount: expect.any(Number),
      total_amount: expect.any(Number),
      today_count: expect.any(Number),
      total_count: expect.any(Number),
      avg_amount: expect.any(Number),
      pending_orders: 1,
      daily_series: expect.any(Array),
      payment_methods: expect.any(Array),
      top_users: expect.any(Array),
    })
    expect(dashboard.data).not.toHaveProperty('items')

    const adminOrders = await requestJson<MockPage<MockPaymentOrder>>(`${baseUrl}/api/v1/admin/payment/orders?page=1&page_size=20`, {
      headers: adminHeaders,
    })
    expect(adminOrders.code).toBe(0)
    expect(adminOrders.data.total).toBe(1)
    expect(adminOrders.data.items[0]).toMatchObject({
      id: created.data.order_id,
      user_id: 2,
      user_email: 'operator@example.test',
      out_trade_no: created.data.out_trade_no,
      payment_trade_no: expect.any(String),
    })

    const detail = await requestJson<MockAdminOrderDetail>(`${baseUrl}/api/v1/admin/payment/orders/${created.data.order_id}`, {
      headers: adminHeaders,
    })
    expect(detail.code).toBe(0)
    expect(detail.data.order).toMatchObject({
      id: created.data.order_id,
      user_email: 'operator@example.test',
    })
    expect(Array.isArray(detail.data.auditLogs)).toBe(true)

    const refundEligible = await requestJson<{ provider_instance_ids: string[] }>(`${baseUrl}/api/v1/payment/orders/refund-eligible-providers`, {
      headers: userHeaders,
    })
    expect(refundEligible.code).toBe(0)
    expect(refundEligible.data.provider_instance_ids).toContain('1')

    const cancelled = await requestJson<{ message: string }>(`${baseUrl}/api/v1/admin/payment/orders/${created.data.order_id}/cancel`, {
      method: 'POST',
      headers: adminHeaders,
    })
    expect(cancelled.code).toBe(0)
    expect(cancelled.data.message).toBeTruthy()

    const afterCancel = await requestJson<MockPaymentOrder>(`${baseUrl}/api/v1/payment/orders/${created.data.order_id}`, {
      headers: userHeaders,
    })
    expect(afterCancel.data.status).toBe('CANCELLED')
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
