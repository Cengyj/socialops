import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageDetailDialog from '../UsageDetailDialog.vue'

const messages: Record<string, string> = {
  'usage.detailTitle': 'Task Detail',
  'usage.detailDescription': 'Review the submitted configuration and execution summary.',
  'usage.detailLoading': 'Loading detail...',
  'usage.detailEmpty': 'No structured detail is available for this record.',
  'usage.detailSections.summary': 'Summary',
  'usage.detailSections.proxy': 'Execution Proxy',
  'usage.detailSections.payload': 'Execution Payload',
  'usage.detailSections.profile': 'Profile Fields',
  'usage.detailSections.media': 'Media',
  'usage.detailSections.template': 'Template Snapshot',
  'usage.detailSections.technical': 'Technical Info',
  'common.close': 'Close',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const detail = {
  id: 1,
  user_id: 7,
  social_account_id: 9,
  platform: 'x_twitter',
  account_name: 'x-main',
  operation: 'post',
  status: 'success',
  quantity: 1,
  cost: 0.1,
  charge_status: 'charged',
  created_at: '2026-06-01T00:00:00Z',
}

const baseProps = {
  show: true,
  loading: false,
  detail,
  overviewRows: [
    { label: 'Operation', value: 'Post' },
    { label: 'Platform', value: 'Twitter / X' },
    { label: 'Account', value: 'x-main' },
    { label: 'Status', value: 'Success', badgeClass: 'badge-success' },
    { label: 'Cost', value: '$0.10', valueTone: 'money' },
    { label: 'Completed at', value: '2026-06-01 00:00:00' },
  ],
  resultRows: [{ label: 'Result', value: 'post succeeded', span: 'full' }],
  proxyRows: [{ label: 'Proxy endpoint', value: 'http://proxy.local:8080' }],
  payloadRows: [{ label: 'Content', value: 'hello world' }],
  payloadProfileRows: [{ label: 'Display name', value: 'SocialOps' }],
  payloadMediaCards: [
    {
      title: 'Media #1',
      previewSrc: 'blob:usage-media-preview',
      previewTestId: 'usage-media-preview-payload-post-0',
      rows: [{ label: 'File name', value: 'post.png' }],
    },
  ],
  templateSummaryRows: [{ label: 'Template name', value: 'Launch post' }],
  templatePoolCards: [{ title: 'Targets', values: ['@northwind'] }],
  templateProfileRows: [{ label: 'Location', value: 'Earth' }],
  templateMediaCards: [],
  technicalRows: [{ label: 'Idempotency key', value: 'usage-detail-1' }],
}

const dialogStubs = {
  BaseDialog: {
    props: ['show', 'title'],
    emits: ['close'],
    template: '<div v-if="show"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></div>',
  },
}

describe('UsageDetailDialog', () => {
  it('renders structured usage detail sections without flattening the record', () => {
    const wrapper = mount(UsageDetailDialog, {
      props: baseProps,
      global: {
        stubs: dialogStubs,
      },
    })

    expect(wrapper.text()).toContain('Task Detail')
    const overview = wrapper.get('[data-testid="usage-detail-overview"]')
    expect(overview.text()).toContain('Post')
    expect(overview.text()).toContain('Twitter / X')
    expect(overview.text()).toContain('x-main')
    expect(overview.get('.badge-success').text()).toContain('Success')
    expect(overview.get('.text-green-600').text()).toContain('$0.10')
    expect(wrapper.get('[data-testid="usage-detail-result"]').text()).toContain('post succeeded')
    expect(wrapper.get('[data-testid="usage-detail-proxy"]').text()).toContain('http://proxy.local:8080')
    expect(wrapper.get('[data-testid="usage-detail-payload"]').text()).toContain('hello world')
    expect(wrapper.text()).toContain('Profile Fields')
    expect(wrapper.text()).toContain('Media #1')
    expect(wrapper.text()).toContain('Launch post')
    expect(wrapper.text()).toContain('@northwind')
    expect(wrapper.get('[data-testid="usage-detail-technical"]').text()).toContain('usage-detail-1')
    expect(wrapper.get('[data-testid="usage-media-preview-payload-post-0"]').attributes('src')).toBe('blob:usage-media-preview')
  })

  it('renders loading and empty states and emits close from the footer', async () => {
    const loading = mount(UsageDetailDialog, {
      props: {
        ...baseProps,
        loading: true,
        detail: null,
      },
      global: {
        stubs: dialogStubs,
      },
    })

    expect(loading.text()).toContain('Loading detail...')

    const empty = mount(UsageDetailDialog, {
      props: {
        ...baseProps,
        detail: null,
        overviewRows: [],
        resultRows: [],
        proxyRows: [],
        payloadRows: [],
        payloadProfileRows: [],
        payloadMediaCards: [],
        templateSummaryRows: [],
        templatePoolCards: [],
        templateProfileRows: [],
        templateMediaCards: [],
        technicalRows: [],
      },
      global: {
        stubs: dialogStubs,
      },
    })

    expect(empty.text()).toContain('No structured detail is available for this record.')
    await empty.get('[data-testid="usage-detail-close"]').trigger('click')
    expect(empty.emitted('close')).toHaveLength(1)
  })
})
