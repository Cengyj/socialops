import { describe, expect, it } from 'vitest'

import type { UsageLog } from '@/api/usage'
import {
  buildUsageExportQueryParams,
  buildUsageCsv,
  buildUsageFilterParams,
  buildUsageListQueryParams,
  buildUsageStatsQueryParams,
  collectDetailMediaPreviewLocators,
  csvCell,
  defaultUsageEndDate,
  defaultUsageStartDate,
  formatDate,
  formatDateInputValue,
  formatMediaDimensions,
  formatPercentage,
  isFinalUsageStatus,
  normalizeUsageFilterValue,
  normalizeUsageOptionValue,
  normalizeUsageSelectValue,
  normalizeUsageStatusFilterValue,
  parseProxySnapshotValue,
} from '@/utils/usageRecords'

const csvLabels = {
  platform: 'Platform',
  operation: 'Operation',
  account: 'Account',
  result: 'Result',
  cost: 'Cost',
  summary: 'Summary',
  time: 'Time',
  target: 'Target',
  content: 'Content',
}

const csvFormatters = {
  actionLabel: (value?: string | null) => value === 'post' ? 'Post' : String(value || ''),
  platformLabel: (value?: string | null) => value === 'x_twitter' ? 'Twitter / X' : String(value || ''),
  statusLabel: (value?: string | null) => value === 'success' ? 'Succeeded' : String(value || ''),
  resultSummary: () => 'Post - Text ready',
  resultMessage: () => 'post succeeded',
}

describe('usageRecords utilities', () => {
  it('formats date inputs and hides invalid display dates', () => {
    const date = new Date(2026, 5, 8)

    expect(formatDateInputValue(date)).toBe('2026-06-08')
    expect(defaultUsageEndDate(date)).toBe('2026-06-08')
    expect(defaultUsageStartDate(date)).toBe('2026-05-10')
    expect(formatDate('not-a-date')).toBe('-')
    expect(formatDate()).toBe('-')
  })

  it('formats usage percentages without noisy decimals', () => {
    expect(formatPercentage(21, 34)).toBe('61.8%')
    expect(formatPercentage(2, 4)).toBe('50%')
    expect(formatPercentage(0, 0)).toBe('0%')
  })

  it('normalizes usage filter values consistently', () => {
    expect(normalizeUsageSelectValue(' Post ')).toBe('post')
    expect(normalizeUsageSelectValue('')).toBe('all')
    expect(normalizeUsageSelectValue(null)).toBe('all')
    expect(normalizeUsageOptionValue(' X_TWITTER ')).toBe('x_twitter')
    expect(normalizeUsageFilterValue('all')).toBe('')
    expect(normalizeUsageFilterValue(' Failed ')).toBe('failed')
    expect(normalizeUsageStatusFilterValue(' Failed ')).toBe('failed')
    expect(normalizeUsageStatusFilterValue('running')).toBe('')
    expect(normalizeUsageStatusFilterValue('all')).toBe('')
  })

  it('builds shared SocialOps usage query filters for list, stats, and export', () => {
    const filters = {
      startDate: '2026-06-01',
      endDate: '2026-06-08',
      operation: ' Follow ',
      platform: ' X_TWITTER ',
      account: ' main account ',
      status: ' Failed ',
    }
    const expectedFilters = {
      start_date: '2026-06-01',
      end_date: '2026-06-08',
      operation: 'follow',
      platform: 'x_twitter',
      account: 'main account',
      status: 'failed',
    }

    expect(buildUsageFilterParams(filters)).toEqual(expectedFilters)
    expect(buildUsageStatsQueryParams(filters)).toEqual(expectedFilters)
    expect(buildUsageListQueryParams(filters, { page: 3, pageSize: 50 })).toEqual({
      page: 3,
      page_size: 50,
      sort_by: 'time',
      sort_order: 'desc',
      ...expectedFilters,
    })
    expect(buildUsageExportQueryParams(filters, { page: 2, pageSize: 100 })).toEqual({
      page: 2,
      page_size: 100,
      sort_by: 'time',
      sort_order: 'desc',
      ...expectedFilters,
    })
    expect(buildUsageExportQueryParams(filters, { page: 2, pageSize: 100, sortBy: 'cost', sortOrder: 'asc' })).toEqual({
      page: 2,
      page_size: 100,
      sort_by: 'cost',
      sort_order: 'asc',
      ...expectedFilters,
    })
  })

  it('omits inactive SocialOps usage filters from query params', () => {
    expect(buildUsageFilterParams({
      startDate: '',
      endDate: null,
      operation: 'all',
      platform: 'all',
      account: '   ',
      status: 'all',
    })).toEqual({})
  })

  it('omits non-final task statuses from usage query params', () => {
    expect(buildUsageFilterParams({
      status: 'running',
    })).toEqual({})
    expect(buildUsageListQueryParams({
      status: 'pending',
    }, { page: 1, pageSize: 20 })).toEqual({
      page: 1,
      page_size: 20,
      sort_by: 'time',
      sort_order: 'desc',
    })
    expect(buildUsageStatsQueryParams({
      status: 'failed',
    })).toEqual({ status: 'failed' })
  })

  it('recognizes only completed task outcomes as Usage statuses', () => {
    expect(isFinalUsageStatus('success')).toBe(true)
    expect(isFinalUsageStatus(' Failed ')).toBe(true)
    expect(isFinalUsageStatus('running')).toBe(false)
    expect(isFinalUsageStatus('queued')).toBe(false)
    expect(isFinalUsageStatus('')).toBe(false)
  })

  it('escapes CSV cells against formulas, quotes, and line breaks', () => {
    expect(csvCell('=SUM(1,2)')).toBe('"\'=SUM(1,2)"')
    expect(csvCell('+1')).toBe("\"'+1\"")
    expect(csvCell('hello "world"\nnext')).toBe('"hello ""world"" next"')
  })

  it('builds SocialOps usage CSV without leaking media storage internals', () => {
    const row: UsageLog = {
      id: 1,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: '=main',
      operation: 'post',
      status: 'success',
      quantity: 1,
      cost: 1.5,
      charge_status: 'charged',
      target: 'https://x.com/northwind/status/1',
      content: '=HYPERLINK("https://example.com")',
      payload: {
        post: {
          text: '=HYPERLINK("https://example.com")',
          media: [
            {
              source: 'inline',
              file_name: 'post-image.png',
              content_type: 'image/png',
              storage_key: 'social-task/private/post-image.png',
            },
          ],
        },
      },
      created_at: '2026-05-31T00:00:00Z',
      completed_at: '2026-05-31T00:00:01Z',
    }

    const csv = buildUsageCsv([row], csvLabels, csvFormatters)

    expect(csv).toContain('"Platform","Operation","Account","Result","Cost","Summary","Time","Target","Content"')
    expect(csv).not.toContain('Charge Status')
    expect(csv).not.toContain('"Charged"')
    expect(csv).toContain('"Post"')
    expect(csv).toContain('"Twitter / X"')
    expect(csv).toContain('"\'=main"')
    expect(csv).toContain('"\'=HYPERLINK(""https://example.com"")"')
    expect(csv).not.toContain('storage_key')
    expect(csv).not.toContain('social-task/private')
  })

  it('parses safe proxy snapshot summaries without exposing raw JSON by default', () => {
    expect(parseProxySnapshotValue('http://proxy.local:8080')).toEqual({
      kind: 'endpoint',
      endpoint: 'http://proxy.local:8080',
    })
    expect(parseProxySnapshotValue('{"name":"proxy-a","endpoint":"http://proxy.local:8080","status":"online","password":"secret"}')).toEqual({
      kind: 'structured',
      name: 'proxy-a',
      endpoint: 'http://proxy.local:8080',
      status: 'online',
    })
    expect(parseProxySnapshotValue('not json')).toBeNull()
  })

  it('collects only previewable task-history media locators', () => {
    const row: UsageLog = {
      id: 1,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'main',
      operation: 'post',
      status: 'success',
      quantity: 1,
      cost: 1.5,
      charge_status: 'charged',
      payload: {
        post: {
          media: [
            { source: 'inline', content_type: 'image/png', file_name: 'image.png' },
            { source: 'inline', content_type: 'video/mp4', file_name: 'video.mp4' },
          ],
        },
        avatar: { source: 'inline', content_type: '', file_name: 'avatar' },
      },
      template_snapshot: {
        template_id: 'tmpl_1',
        template_name: 'Template',
        template_type: 'post',
        params: {
          banner: { source: 'inline', content_type: 'image/jpeg', file_name: 'banner.jpg' },
        },
      },
      created_at: '2026-05-31T00:00:00Z',
    }

    expect(collectDetailMediaPreviewLocators(row)).toEqual([
      { key: 'payload:post:0', locator: { scope: 'payload', section: 'post', index: 0 } },
      { key: 'payload:avatar', locator: { scope: 'payload', section: 'avatar' } },
      { key: 'template:banner', locator: { scope: 'template', section: 'banner' } },
    ])
  })

  it('formats media dimensions only when both sides are known', () => {
    expect(formatMediaDimensions({ width: 1500, height: 500 })).toBe('1,500 \u00d7 500')
    expect(formatMediaDimensions({ width: 1500 })).toBe('')
  })
})
