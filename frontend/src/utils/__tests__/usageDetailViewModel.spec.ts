import { describe, expect, it } from 'vitest'
import type { UsageLog } from '@/api/usage'
import {
  buildUsageDetailViewModel,
  type UsageDetailViewModelFormatters,
} from '@/utils/usageDetailViewModel'

const labels: Record<string, string> = {
  'usage.detailLabels.operation': 'Operation',
  'usage.detailLabels.platform': 'Platform',
  'usage.detailLabels.account': 'Account',
  'usage.detailLabels.status': 'Status',
  'usage.detailLabels.chargeStatus': 'Charge status',
  'usage.detailLabels.chargeSource': 'Charge source',
  'usage.detailLabels.cost': 'Cost',
  'usage.detailLabels.quantity': 'Quantity',
  'usage.detailLabels.result': 'Result',
  'usage.detailLabels.createdAt': 'Created at',
  'usage.detailLabels.completedAt': 'Completed at',
  'usage.detailLabels.target': 'Target',
  'usage.detailLabels.content': 'Content',
  'usage.detailLabels.quotePostUrl': 'Quote link',
  'usage.detailLabels.billingRequestId': 'Billing request',
  'usage.detailLabels.idempotencyKey': 'Idempotency key',
  'usage.detailLabels.proxySnapshot': 'Proxy snapshot',
  'usage.detailLabels.proxyName': 'Proxy name',
  'usage.detailLabels.proxyEndpoint': 'Proxy endpoint',
  'usage.detailLabels.proxyStatus': 'Proxy status',
  'usage.detailLabels.displayName': 'Display name',
  'usage.detailLabels.screenName': 'Screen name',
  'usage.detailLabels.description': 'Description',
  'usage.detailLabels.location': 'Location',
  'usage.detailLabels.url': 'URL',
  'usage.detailLabels.avatar': 'Avatar',
  'usage.detailLabels.banner': 'Banner',
  'usage.detailLabels.templateName': 'Template name',
  'usage.detailLabels.templateType': 'Template type',
  'usage.detailLabels.fileName': 'File name',
  'usage.detailLabels.contentType': 'Content type',
  'usage.detailLabels.dimensions': 'Dimensions',
  'usage.detailLabels.byteSize': 'Size',
  'usage.detailLabels.source': 'Source',
  'usage.detailSections.targets': 'Targets',
  'usage.detailSections.contents': 'Contents',
}

const formatters: UsageDetailViewModelFormatters = {
  t: (key, params) => {
    if (key === 'usage.detailLabels.mediaItem') return `Media ${params?.index ?? ''}`.trim()
    return labels[key] ?? key
  },
  actionLabel: value => value === 'post' ? 'Post' : String(value || 'Unknown'),
  platformLabel: value => value === 'x_twitter' ? 'Twitter / X' : String(value || '-'),
  statusLabel: value => value === 'success' ? 'Success' : String(value || '-'),
  chargeStatusLabel: value => value === 'charged' ? 'Charged' : String(value || '-'),
  chargeSourceLabel: value => value === 'subscription' ? 'Subscription' : String(value || '-'),
  proxyStatusLabel: value => value === 'online' ? 'Online' : String(value || ''),
  resultMessage: row => row.result_message || '',
}

const richDetail: UsageLog = {
  id: 1,
  user_id: 7,
  social_account_id: 9,
  platform: 'x_twitter',
  account_name: 'x-main',
  operation: 'post',
  status: 'success',
  charge_status: 'charged',
  charge_source: 'subscription',
  proxy_snapshot: '{"id":8,"name":"proxy-a","endpoint":"http://proxy.local:8080","status":"online","password":"secret"}',
  billing_request_id: 'sub:task-1',
  idempotency_key: 'usage-detail-1',
  result_message: 'post succeeded',
  quantity: 1,
  cost: 1.5,
  created_at: '2026-05-31T00:00:00Z',
  completed_at: '2026-05-31T00:00:01Z',
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
          storage_key: 'social-task/private/post-image-1.png',
        } as never,
      ],
    },
    profile: {
      display_name: 'Northwind Updates',
      location: 'San Francisco',
      url: 'https://example.com/northwind',
    },
    avatar: {
      source: 'inline',
      file_name: 'avatar.png',
      content_type: 'image/png',
      byte_size: 2048,
      width: 400,
      height: 400,
      storage_key: 'social-task/private/avatar.png',
    } as never,
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
          file_name: 'template-image-1.png',
          content_type: 'image/png',
          byte_size: 4096,
          width: 500,
          height: 500,
          storage_key: 'social-task/private/template-image-1.png',
        } as never,
      ],
    },
  },
}

describe('usageDetailViewModel', () => {
  it('builds structured SocialOps detail sections without exposing raw proxy or storage refs', () => {
    const viewModel = buildUsageDetailViewModel(richDetail, {
      'payload:post:0': 'blob:payload-post',
      'payload:avatar': 'blob:payload-avatar',
    }, formatters)

    expect(viewModel.overviewRows).toEqual(expect.arrayContaining([
      expect.objectContaining({ label: 'Operation', value: 'Post' }),
      expect.objectContaining({ label: 'Platform', value: 'Twitter / X', valueTone: 'muted' }),
      expect.objectContaining({ label: 'Status', value: 'Success', badgeClass: 'badge-success' }),
      expect.objectContaining({ label: 'Charge status', value: 'Charged', badgeClass: 'badge-success' }),
      expect.objectContaining({ label: 'Cost', value: '$1.50', valueTone: 'money' }),
      expect.objectContaining({ label: 'Completed at' }),
    ]))
    expect(viewModel.resultRows).toEqual(expect.arrayContaining([
      expect.objectContaining({ label: 'Result', value: 'post succeeded', span: 'full' }),
      expect.objectContaining({ label: 'Charge source', value: 'Subscription', valueTone: 'muted' }),
      expect.objectContaining({ label: 'Quantity', value: '1' }),
      expect.objectContaining({ label: 'Created at' }),
    ]))
    expect(viewModel.proxyRows).toEqual([
      { label: 'Proxy name', value: 'proxy-a' },
      { label: 'Proxy endpoint', value: 'http://proxy.local:8080' },
      { label: 'Proxy status', value: 'Online', badgeClass: 'badge-success' },
    ])
    expect(viewModel.payloadRows).toEqual(expect.arrayContaining([
      expect.objectContaining({ label: 'Target', value: 'https://x.com/northwind/status/1' }),
      expect.objectContaining({ label: 'Content', value: 'hello world', span: 'full' }),
      expect.objectContaining({ label: 'Quote link', value: 'https://x.com/northwind/status/2', span: 'full' }),
    ]))
    expect(viewModel.payloadProfileRows).toContainEqual({ label: 'Display name', value: 'Northwind Updates' })
    expect(viewModel.payloadMediaCards[0]).toMatchObject({
      title: 'Media 1',
      previewSrc: 'blob:payload-post',
      previewTestId: 'usage-media-preview-payload-post-0',
    })
    expect(viewModel.payloadMediaCards[1]).toMatchObject({
      title: 'Avatar',
      previewSrc: 'blob:payload-avatar',
      previewTestId: 'usage-media-preview-payload-avatar',
    })
    expect(viewModel.templateSummaryRows).toContainEqual({ label: 'Template name', value: 'Rich post' })
    expect(viewModel.templatePoolCards).toContainEqual({ title: 'Targets', values: ['https://x.com/northwind/status/1'] })
    expect(viewModel.technicalRows).toContainEqual(expect.objectContaining({ label: 'Idempotency key', value: 'usage-detail-1', valueTone: 'technical' }))

    const serialized = JSON.stringify(viewModel)
    expect(serialized).not.toContain('password')
    expect(serialized).not.toContain('storage_key')
    expect(serialized).not.toContain('social-task/private')
    expect(serialized).not.toContain('{"id":8')
  })

  it('normalizes plain endpoint proxy snapshots as endpoint-only rows', () => {
    const viewModel = buildUsageDetailViewModel({
      ...richDetail,
      proxy_snapshot: 'http://proxy.local:8080',
      template_snapshot: null,
      payload: null,
    }, {}, formatters)

    expect(viewModel.proxyRows).toEqual([{ label: 'Proxy endpoint', value: 'http://proxy.local:8080' }])
    expect(viewModel.templateSummaryRows).toEqual([])
    expect(viewModel.payloadRows).toEqual([])
  })

  it('returns empty sections for missing detail', () => {
    const viewModel = buildUsageDetailViewModel(null, {}, formatters)

    expect(Object.values(viewModel).every(section => Array.isArray(section) && section.length === 0)).toBe(true)
  })
})
