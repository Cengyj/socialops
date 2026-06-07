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

  it('keeps sanitized structured task fields on usage list responses', async () => {
    const page = {
      items: [
        {
          id: 2,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'post',
          status: 'failed',
          quantity: 1,
          cost: 0,
          charge_status: 'not_charged',
          target: 'https://x.com/openai/status/1',
          content: 'hello world',
          payload: {
            target: 'https://x.com/openai/status/1',
            post: {
              text: 'hello world',
              quote_post_url: 'https://x.com/openai/status/2',
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
          template_snapshot: {
            template_id: 'tmpl_1',
            template_name: 'Rich post',
            template_type: 'post',
            params: {
              targets: ['https://x.com/openai/status/1'],
              contents: ['hello world'],
              quote_post_url: 'https://x.com/openai/status/2',
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
    expect(result.items[0].template_snapshot?.params?.media?.[0]?.storage_key).toBeUndefined()
    expect(result.items[0].template_snapshot?.template_name).toBe('Rich post')
  })

  it('keeps query as the same usage list contract', async () => {
    const page = { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    get.mockResolvedValue({ data: page })

    const result = await usageAPI.query({ status: 'success' })

    expect(get).toHaveBeenCalledWith('/usage', { params: { status: 'success' } })
    expect(result).toEqual(page)
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

  it('keeps structured task detail fields on usage detail responses', async () => {
    const record = {
      id: 8,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'post',
      status: 'success',
      quantity: 1,
      cost: 0.1,
      charge_status: 'charged',
      target: 'https://x.com/openai/status/1',
      content: 'hello world',
      payload: {
        target: 'https://x.com/openai/status/1',
        post: {
          text: 'hello world',
          quote_post_url: 'https://x.com/openai/status/2',
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
          display_name: 'OpenAI News',
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

    expect(get).toHaveBeenCalledWith('/usage/8')
    expect(result).toEqual(record)
    expect(result.payload?.post?.media?.[0]?.file_name).toBe('post-image-1.png')
    expect(result.template_snapshot?.template_name).toBe('Rich post')
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
    const stats = { total_requests: 3, total_actual_cost: 0.3 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getStats({ status: 'success', operation: 'follow' })

    expect(get).toHaveBeenCalledWith('/usage/stats', { params: { status: 'success', operation: 'follow' } })
    expect(result).toEqual(stats)
  })

  it('keeps the date-range stats helper on the same stats endpoint', async () => {
    const stats = { total_requests: 3, total_actual_cost: 0.3 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getStatsByDateRange('2026-06-01', '2026-06-02')

    expect(get).toHaveBeenCalledWith('/usage/stats', {
      params: {
        start_date: '2026-06-01',
        end_date: '2026-06-02',
      },
    })
    expect(result).toEqual(stats)
  })

  it('unwraps user dashboard usage stats', async () => {
    const stats = { total_requests: 12, today_requests: 3, total_actual_cost: 1.2 }
    get.mockResolvedValue({ data: stats })

    const result = await usageAPI.getDashboardStats()

    expect(get).toHaveBeenCalledWith('/usage/dashboard/stats')
    expect(result).toEqual(stats)
  })

  it('normalizes dashboard trend arrays from the current backend response', async () => {
    const trend = [{ date: '2026-06-01', requests: 2, actual_cost: 0.2 }]
    get.mockResolvedValue({ data: trend })

    const result = await usageAPI.getDashboardTrend({ granularity: 'day' })

    expect(get).toHaveBeenCalledWith('/usage/dashboard/trend', { params: { granularity: 'day' } })
    expect(result).toEqual(trend)
  })

  it('normalizes dashboard trend wrapper responses from compatible callers', async () => {
    const trend = [{ date: '2026-06-01', requests: 2, actual_cost: 0.2 }]
    get.mockResolvedValue({ data: { trend } })

    const result = await usageAPI.getDashboardTrend()

    expect(get).toHaveBeenCalledWith('/usage/dashboard/trend', { params: undefined })
    expect(result).toEqual(trend)
  })
})
