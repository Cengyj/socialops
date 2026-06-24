import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { usageAPI } from '@/api/usage'

describe('user usage api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('unwraps paginated SocialOps usage records', async () => {
    const page = {
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          quantity: 1,
          cost: 0.1,
          charge_status: 'charged',
          created_at: '2026-06-01T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.list({ page: 1, page_size: 20, operation: 'follow' })

    expect(get).toHaveBeenCalledWith('/usage', { params: { page: 1, page_size: 20, operation: 'follow' } })
    expect(result).toEqual(page)
  })

  it('keeps structured task fields while dropping stale bookkeeping on usage list responses', async () => {
    const page = {
      items: [
        {
          id: 2,
          user_id: 7,
          api_key_id: 123,
          group_id: 456,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'post',
          model: 'stale-bookkeeping-field',
          input_tokens: 100,
          output_tokens: 200,
          total_tokens: 300,
          service_tier: 'priority',
          status: 'failed',
          quantity: 1,
          cost: 0,
          charge_status: 'not_charged',
          target: 'https://x.com/northwind/status/1',
          content: 'hello world',
          payload: {
            target: 'https://x.com/northwind/status/1',
            post: {
              text: 'hello world',
              quote_post_url: 'https://x.com/northwind/status/2',
              media: [
                {
                  source: 'inline',
                  file_name: 'post-image-1.png',
                  content_type: 'image/png',
                  storage_key: 'social-task/private/post-image-1.png',
                  url: 'https://storage.example/private/post-image-1.png',
                  sha256: 'secret-sha',
                  byte_size: 1234,
                  width: 400,
                  height: 400,
                },
              ],
            },
          },
          template_snapshot: {
            template_id: 'tmpl_1',
            template_name: 'Rich post',
            template_type: 'post',
            params: {
              targets: ['https://x.com/northwind/status/1'],
              contents: ['hello world'],
              quote_post_url: 'https://x.com/northwind/status/2',
              media: [
                {
                  source: 'inline',
                  file_name: 'post-image-1.png',
                  content_type: 'image/png',
                  storage_key: 'social-task/private/template-post-image-1.png',
                  url: 'https://storage.example/private/template-post-image-1.png',
                  sha256: 'template-secret-sha',
                  byte_size: 1234,
                  width: 400,
                  height: 400,
                },
              ],
            },
          },
          created_at: '2026-06-01T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.list()

    expect(result.items[0].payload?.post?.media?.[0]?.file_name).toBe('post-image-1.png')
    expect(result.items[0].payload?.post?.media?.[0]?.url).toBeUndefined()
    expect(result.items[0].payload?.post?.media?.[0]?.storage_key).toBeUndefined()
    expect(result.items[0].payload?.post?.media?.[0]?.sha256).toBeUndefined()
    expect(result.items[0].template_snapshot?.params?.media?.[0]?.storage_key).toBeUndefined()
    expect(result.items[0].template_snapshot?.template_name).toBe('Rich post')
    expect(result.items[0]).not.toHaveProperty('api_key_id')
    expect(result.items[0]).not.toHaveProperty('group_id')
    expect(result.items[0]).not.toHaveProperty('model')
    expect(result.items[0]).not.toHaveProperty('input_tokens')
    expect(result.items[0]).not.toHaveProperty('output_tokens')
    expect(result.items[0]).not.toHaveProperty('total_tokens')
    expect(result.items[0]).not.toHaveProperty('service_tier')
  })

  it('omits non-final task states from usage list responses', async () => {
    const page = {
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          quantity: 1,
          cost: 0.1,
          charge_status: 'charged',
          created_at: '2026-06-01T00:00:00Z',
        },
        {
          id: 2,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'post',
          status: 'running',
          quantity: 1,
          cost: 0,
          charge_status: 'not_charged',
          created_at: '2026-06-01T00:00:00Z',
        },
        {
          id: 3,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'like',
          status: 'failed',
          quantity: 1,
          cost: 0,
          charge_status: 'not_charged',
          created_at: '2026-06-01T00:00:00Z',
        },
      ],
      total: 3,
      page: 1,
      page_size: 20,
      pages: 1,
    }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.list()

    expect(result.items.map(item => item.status)).toEqual(['success', 'failed'])
    expect(result.items.map(item => item.operation)).toEqual(['follow', 'like'])
    expect(result.total).toBe(3)
    expect(result.pages).toBe(1)
  })

  it('unwraps one usage record by id', async () => {
    const record = {
      id: 7,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'like',
      status: 'success',
      quantity: 1,
      cost: 0.1,
      charge_status: 'charged',
      created_at: '2026-06-01T00:00:00Z',
    }
    get.mockResolvedValue({ data: record })

    const result = await usageAPI.getById(7)

    expect(get).toHaveBeenCalledWith('/usage/7')
    expect(result).toEqual(record)
  })

  it('keeps structured task detail fields while dropping stale bookkeeping on detail responses', async () => {
    const record = {
      id: 8,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'post',
      model: 'stale-bookkeeping-field',
      service_tier: 'priority',
      status: 'success',
      quantity: 1,
      cost: 0.1,
      charge_status: 'charged',
      target: 'https://x.com/northwind/status/1',
      content: 'hello world',
      payload: {
        target: 'https://x.com/northwind/status/1',
        post: {
          text: 'hello world',
          quote_post_url: 'https://x.com/northwind/status/2',
          media: [
            {
              source: 'inline',
              file_name: 'post-image-1.png',
              content_type: 'image/png',
              byte_size: 1234,
              width: 400,
              height: 400,
            },
          ],
        },
        profile: {
          display_name: 'Northwind Updates',
        },
      },
      template_snapshot: {
        template_id: 'tmpl_1',
        template_name: 'Rich post',
        template_type: 'post',
        params: {
          contents: ['hello world'],
          media: [
            {
              source: 'inline',
              file_name: 'post-image-1.png',
              content_type: 'image/png',
              byte_size: 1234,
              width: 400,
              height: 400,
            },
          ],
        },
      },
      created_at: '2026-06-01T00:00:00Z',
    }
    get.mockResolvedValue({ data: record })

    const result = await usageAPI.getById(8)
    const expectedRecord = { ...record }
    delete (expectedRecord as Record<string, unknown>).model
    delete (expectedRecord as Record<string, unknown>).service_tier

    expect(get).toHaveBeenCalledWith('/usage/8')
    expect(result).toEqual(expectedRecord)
    expect(result.payload?.post?.media?.[0]?.file_name).toBe('post-image-1.png')
    expect(result.template_snapshot?.template_name).toBe('Rich post')
    expect(result).not.toHaveProperty('model')
    expect(result).not.toHaveProperty('service_tier')
  })

  it('streams task-history media previews through the dedicated usage media endpoint', async () => {
    const blob = new Blob(['preview'], { type: 'image/png' })
    get.mockResolvedValue({ data: blob })

    const result = await usageAPI.previewTaskMedia(8, { scope: 'payload', section: 'post', index: 0 })

    expect(get).toHaveBeenCalledWith('/usage/8/media', {
      params: { scope: 'payload', section: 'post', index: 0 },
      responseType: 'blob',
    })
    expect(result).toBe(blob)
  })

  it('passes SocialOps usage stat filters to the stats endpoint', async () => {
    const stats = {
      total_operations: 3,
      success_count: 2,
      failed_count: 1,
      total_charged: 0.3,
      total_requests: 99,
      total_tokens: 99,
      total_actual_cost: 9.9,
    }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getStats({ status: 'success', operation: 'follow', platform: 'x_twitter', account: 'main' })

    expect(get).toHaveBeenCalledWith('/usage/stats', { params: { status: 'success', operation: 'follow', platform: 'x_twitter', account: 'main' } })
    expect(result).toEqual({
      total_operations: 3,
      success_count: 2,
      failed_count: 1,
      total_charged: 0.3,
    })
    expect(result).not.toHaveProperty('total_requests')
    expect(result).not.toHaveProperty('total_tokens')
    expect(result).not.toHaveProperty('total_actual_cost')
  })

  it('sanitizes user dashboard stats through the SocialOps operation contract', async () => {
    const stats = {
      total_operations: 12,
      today_operations: 3,
      total_charged: 1.2,
      today_charged: 0.3,
      recent_operations_per_minute: 2,
      total_requests: 99,
      total_tokens: 12,
      total_actual_cost: 9.9,
      rpm: 99,
      by_platform: [
        {
          platform: 'x_twitter',
          total_operations: 10,
          today_operations: 2,
          total_charged: 1.0,
          today_charged: 0.2,
          total_requests: 99,
          total_actual_cost: 9.9,
        },
      ],
    }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getDashboardStats()

    expect(get).toHaveBeenCalledWith('/usage/dashboard/stats')
    expect(result).toEqual({
      total_operations: 12,
      today_operations: 3,
      total_charged: 1.2,
      today_charged: 0.3,
      recent_operations_per_minute: 2,
      by_platform: [
        { platform: 'x_twitter', total_operations: 10, today_operations: 2, total_charged: 1.0, today_charged: 0.2 },
      ],
    })
    expect(result).not.toHaveProperty('total_requests')
    expect(result).not.toHaveProperty('total_tokens')
    expect(result).not.toHaveProperty('rpm')
  })

  it('sanitizes dashboard trend arrays from the SocialOps backend response', async () => {
    const trend = [{ date: '2026-06-01', operations: 2, charged: 0.2, requests: 99, actual_cost: 9.9 }]
    get.mockResolvedValue({ data: trend })

    const result = await usageAPI.getDashboardTrend({ granularity: 'day' })

    expect(get).toHaveBeenCalledWith('/usage/dashboard/trend', { params: { granularity: 'day' } })
    expect(result).toEqual([{ date: '2026-06-01', operations: 2, charged: 0.2 }])
    expect(result[0]).not.toHaveProperty('requests')
    expect(result[0]).not.toHaveProperty('actual_cost')
  })
})
