import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import UsageView from '../UsageView.vue'

const { listUsage, getById, getStats, previewTaskMedia, showError } = vi.hoisted(() => ({
  listUsage: vi.fn(),
  getById: vi.fn(),
  getStats: vi.fn(),
  previewTaskMedia: vi.fn(),
  showError: vi.fn(),
}))

const messages: Record<string, string> = {
  'usage.title': 'Usage Records',
  'usage.description': 'Track SocialOps operation history',
  'usage.filters.operation': 'Operation filter',
  'usage.filters.status': 'Status filter',
  'usage.filters.allOperations': 'All operations',
  'usage.filters.allStatuses': 'All statuses',
  'usage.filters.allPlatforms': 'All platforms',
  'usage.filters.clear': 'Clear filters',
  'usage.records': 'Operation Records',
  'usage.operation': 'Operation',
  'usage.platform': 'Platform',
  'usage.account': 'Account',
  'usage.status': 'Status',
  'usage.quantity': 'Quantity',
  'usage.cost': 'Cost',
  'usage.result': 'Result',
  'usage.time': 'Time',
  'usage.actions.viewDetails': 'View details',
  'usage.detailTitle': 'Task Detail',
  'usage.detailDescription': 'Review the submitted configuration and execution summary.',
  'usage.detailSections.summary': 'Summary',
  'usage.detailSections.proxy': 'Execution Proxy',
  'usage.detailSections.payload': 'Execution Payload',
  'usage.detailSections.template': 'Template Snapshot',
  'usage.detailSections.profile': 'Profile Fields',
  'usage.detailSections.media': 'Media',
  'usage.detailSections.targets': 'Targets',
  'usage.detailSections.contents': 'Contents',
  'usage.detailSections.technical': 'Technical Info',
  'usage.detailLabels.target': 'Target',
  'usage.detailLabels.content': 'Content',
  'usage.detailLabels.quotePostUrl': 'Quote link',
  'usage.detailLabels.operation': 'Operation',
  'usage.detailLabels.platform': 'Platform',
  'usage.detailLabels.account': 'Account',
  'usage.detailLabels.status': 'Status',
  'usage.detailLabels.chargeStatus': 'Charge status',
  'usage.detailLabels.cost': 'Cost',
  'usage.detailLabels.result': 'Result',
  'usage.detailLabels.quantity': 'Quantity',
  'usage.detailLabels.createdAt': 'Created at',
  'usage.detailLabels.completedAt': 'Completed at',
  'usage.detailLabels.chargeSource': 'Charge source',
  'usage.detailLabels.proxySnapshot': 'Execution proxy',
  'usage.detailLabels.proxyName': 'Proxy name',
  'usage.detailLabels.proxyEndpoint': 'Proxy endpoint',
  'usage.detailLabels.proxyStatus': 'Proxy status',
  'usage.detailLabels.billingRequestId': 'Billing request',
  'usage.detailLabels.idempotencyKey': 'Idempotency key',
  'usage.detailLabels.templateName': 'Template name',
  'usage.detailLabels.templateType': 'Template type',
  'usage.detailLabels.displayName': 'Display name',
  'usage.detailLabels.screenName': 'Screen name',
  'usage.detailLabels.description': 'Description',
  'usage.detailLabels.location': 'Location',
  'usage.detailLabels.url': 'URL',
  'usage.detailLabels.avatar': 'Avatar',
  'usage.detailLabels.banner': 'Banner',
  'usage.detailLabels.mediaItem': 'Media #{index}',
  'usage.detailLabels.fileName': 'File name',
  'usage.detailLabels.contentType': 'Content type',
  'usage.detailLabels.dimensions': 'Dimensions',
  'usage.detailLabels.byteSize': 'Size',
  'usage.detailLabels.source': 'Source',
  'usage.detailLabels.count': 'Count',
  'usage.detailEmpty': 'No structured detail is available for this record.',
  'usage.detailLoading': 'Loading detail...',
  'usage.detailLoadFailed': 'Failed to load task detail',
  'usage.loading': 'Loading usage records...',
  'usage.empty': 'No SocialOps operations yet',
  'usage.emptyFiltered': 'No usage records match the current filters.',
  'usage.totalOperations': 'Total Operations',
  'usage.successCount': 'Successful',
  'usage.failedCount': 'Failed',
  'usage.successRate': 'Success Rate',
  'usage.totalCharged': 'Total Charged',
  'usage.inSelectedRange': 'In selected range',
  'usage.successOnlyBilling': 'Charged successful tasks',
  'usage.summary': 'Summary',
  'usage.exportCsv': 'Export CSV',
  'usage.exportFailed': 'Failed to export usage records',
  'usage.exportEmpty': 'No usage records match the current filters.',
  'usage.exporting': 'Exporting...',
  'usage.failedToLoad': 'Failed to load usage records',
  'usage.safeResult': 'Task failed; diagnostic details are hidden',
  'usage.taskResults.proxyUnavailable': 'Execution proxy unavailable; not charged',
  'usage.taskResults.queueBusy': 'Task queue is busy; not charged',
  'usage.taskResults.avatarSizeInvalid': 'Avatar image must be 400 × 400; not charged',
  'usage.taskResults.bannerSizeInvalid': 'Banner image must be 1500 × 500; not charged',
  'usage.taskSummaryTarget': 'Target: {value}',
  'usage.taskSummaryContent': 'Text: {value}',
  'usage.taskSummaryQuote': 'Quote: {value}',
  'usage.taskSummaryMedia': '{count} media item(s)',
  'usage.taskSummaryProfile': '{count} profile field(s)',
  'usage.taskSummaryAvatar': 'Avatar image ready',
  'usage.taskSummaryBanner': 'Banner image ready',
  'usage.taskSummaryNoDetails': 'No structured details',
  'usage.platforms.x_twitter': 'Twitter / X',
  'usage.actions.follow': 'Follow',
  'usage.actions.post': 'Post',
  'usage.statuses.success': 'Success',
  'usage.statuses.failed': 'Failed',
  'usage.chargeStatuses.not_charged': 'Not Charged',
  'usage.chargeStatuses.charged': 'Charged',
  'usage.chargeSources.subscription': 'Subscription',
  'usage.chargeSources.wallet': 'Wallet',
  'usage.chargeSources.mixed': 'Subscription + Wallet',
  'usage.proxyStatuses.online': 'Online',
  'usage.proxyStatuses.offline': 'Offline',
  'common.refresh': 'Refresh',
  'common.actions': 'Actions',
  'common.close': 'Close',
  'common.unknown': 'Unknown',
  'common.none': 'None',
}

vi.mock('@/api/usage', () => ({
  usageAPI: {
    list: listUsage,
    getById,
    getStats,
    previewTaskMedia,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        let value = messages[key] ?? key
        if (!params) return value
        Object.entries(params).forEach(([name, replacement]) => {
          value = value.replace(`{${name}}`, String(replacement))
        })
        return value
      },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const DateRangePickerStub = {
  props: ['startDate', 'endDate'],
  emits: ['update:startDate', 'update:endDate', 'change'],
  template: '<button type="button" data-testid="usage-date-range" @click="$emit(\'change\', { startDate, endDate, preset: null })">Date range</button>',
}
const PaginationStub = {
  props: ['total', 'page', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<div data-testid="usage-pagination">{{ total }}</div>',
}
const viewStubs = {
  AppLayout: AppLayoutStub,
  DateRangePicker: DateRangePickerStub,
  Pagination: PaginationStub,
}
const defaultListParams = () => expect.objectContaining({
  page: 1,
  page_size: 20,
  sort_by: 'time',
  sort_order: 'desc',
  start_date: expect.any(String),
  end_date: expect.any(String),
})
const defaultStatsParams = () => expect.objectContaining({
  start_date: expect.any(String),
  end_date: expect.any(String),
})
const readBlobText = (blob: Blob) => new Promise<string>((resolve, reject) => {
  const reader = new FileReader()
  reader.onload = () => resolve(String(reader.result || ''))
  reader.onerror = () => reject(reader.error)
  reader.readAsText(blob)
})
const createDeferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}
const originalCreateObjectURL = globalThis.URL.createObjectURL
const originalRevokeObjectURL = globalThis.URL.revokeObjectURL
const originalAnchorClick = HTMLAnchorElement.prototype.click

describe('user UsageView', () => {
  beforeEach(() => {
    listUsage.mockReset()
    getById.mockReset()
    getStats.mockReset()
    previewTaskMedia.mockReset()
    showError.mockReset()
    previewTaskMedia.mockResolvedValue(new Blob(['preview'], { type: 'image/png' }))
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(() => 'blob:usage-media-preview'),
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(),
    })

    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          charge_status: 'charged',
          result_message: 'follow succeeded',
          quantity: 1,
          cost: 1.5,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({
      total_operations: 34,
      success_count: 21,
      failed_count: 13,
      total_charged: 8.25,
    })
  })

  afterEach(() => {
    globalThis.URL.createObjectURL = originalCreateObjectURL
    globalThis.URL.revokeObjectURL = originalRevokeObjectURL
    HTMLAnchorElement.prototype.click = originalAnchorClick
    document.body.innerHTML = ''
  })

  it('renders SocialOps operation usage records and stats', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(defaultListParams())
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(defaultStatsParams())
    expect(wrapper.text()).toContain('Total Operations')
    expect(wrapper.text()).toContain('Successful')
    expect(wrapper.text()).toContain('Failed')
    expect(wrapper.text()).toContain('Success Rate')
    expect(wrapper.text()).toContain('Total Charged')
    expect(wrapper.text()).not.toContain('Total Quantity')
    expect(wrapper.text()).not.toContain('Total Cost')
    expect(wrapper.text()).toContain('34')
    expect(wrapper.text()).toContain('21')
    expect(wrapper.text()).toContain('13')
    expect(wrapper.text()).toContain('61.8%')
    const statCards = wrapper.findAll('[data-testid="usage-stat-card"]')
    expect(statCards).toHaveLength(5)
    expect(statCards.every(card => card.find('svg').exists())).toBe(true)
    expect(statCards.slice(0, 4).every(card => card.text().includes('In selected range'))).toBe(true)
    expect(statCards[4].text()).toContain('Charged successful tasks')
    const table = wrapper.get('[data-testid="usage-records-table"]')
    expect(table.classes()).toContain('min-w-max')
    expect(table.classes()).not.toContain('table-fixed')
    const tableHeaders = wrapper.findAll('[data-testid="usage-records-table"] thead th').map(header => header.text())
    expect(tableHeaders).toEqual([
      'Platform',
      'Operation',
      'Account',
      'Result',
      'Cost',
      'Summary',
      'Time',
      'Actions',
    ])
    expect(wrapper.findAll('[data-testid="usage-records-table"] thead th').every(header => header.classes().includes('py-3'))).toBe(true)
    expect(wrapper.findAll('[data-testid="usage-records-table"] tbody tr:first-child td').every(cell => cell.classes().includes('py-4'))).toBe(true)
    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Success')
    expect(tableHeaders).not.toContain('Charge Status')
    expect(wrapper.text()).toContain('follow succeeded')
    expect(wrapper.text()).toContain('$8.25')
  })

  it('re-queries usage history when the operation filter changes', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(listUsage).toHaveBeenNthCalledWith(1, defaultListParams())
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(defaultStatsParams())

    listUsage.mockClear()
    getStats.mockClear()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(3)

    await selects[1].vm.$emit('update:modelValue', 'post')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 20, operation: 'post' }))
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ operation: 'post' }))
  })

  it('re-queries usage history when the status filter changes and clears back to defaults', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    listUsage.mockClear()
    getStats.mockClear()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(3)

    await selects[2].vm.$emit('update:modelValue', 'failed')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 20, status: 'failed' }))
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ status: 'failed' }))

    listUsage.mockClear()
    getStats.mockClear()

    await wrapper.get('[data-testid="usage-clear-filters"]').trigger('click')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(defaultListParams())
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(defaultStatsParams())
  })

  it('does not send non-final task statuses to usage history queries', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    listUsage.mockClear()
    getStats.mockClear()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(3)

    await selects[2].vm.$emit('update:modelValue', 'running')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledTimes(1)
    expect(listUsage).toHaveBeenCalledWith(defaultListParams())
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getStats).toHaveBeenCalledWith(defaultStatsParams())
  })

  it('keeps usage records and result filters scoped to final outcomes', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 2,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          charge_status: 'charged',
          quantity: 1,
          cost: 0.1,
          result_message: 'follow succeeded',
          created_at: '2026-05-31T00:00:00Z',
        },
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'running',
          charge_status: 'not_charged',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T00:00:00Z',
        },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    const statusOptions = selects[2].props('options') as Array<{ value: string }>
    expect(statusOptions.map(option => option.value)).toEqual(['all', 'success', 'failed'])
    expect(wrapper.text()).toContain('follow succeeded')
    expect(wrapper.text()).not.toContain('running')
    expect(wrapper.get('[data-testid="usage-pagination"]').text()).toBe('2')
  })

  it('orders filters as platform, operation, status, then time and keeps actions in the filter card', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()
    listUsage.mockClear()
    getStats.mockClear()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(3)
    expect((selects[0].props('options') as Array<{ label: string }>)[0].label).toBe('All platforms')
    expect((selects[1].props('options') as Array<{ label: string }>)[0].label).toBe('All operations')
    expect((selects[2].props('options') as Array<{ label: string }>)[0].label).toBe('All statuses')
    expect(wrapper.find('[data-testid="usage-account-filter"]').exists()).toBe(false)
    expect(wrapper.find('h1').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Track SocialOps operation history')

    const toolbar = wrapper.get('[data-testid="usage-filter-toolbar"]')
    expect(toolbar.html().indexOf('All platforms')).toBeLessThan(toolbar.html().indexOf('All operations'))
    expect(toolbar.html().indexOf('All operations')).toBeLessThan(toolbar.html().indexOf('All statuses'))
    expect(toolbar.html().indexOf('All statuses')).toBeLessThan(toolbar.html().indexOf('usage-date-range'))

    const actions = wrapper.get('[data-testid="usage-filter-actions"]')
    expect(actions.html().indexOf('Refresh')).toBeLessThan(actions.html().indexOf('Clear filters'))
    expect(actions.html().indexOf('Clear filters')).toBeLessThan(actions.html().indexOf('Export CSV'))
    expect(actions.find('[data-testid="usage-refresh"]').exists()).toBe(true)
    expect(actions.find('[data-testid="usage-clear-filters"]').exists()).toBe(true)
    expect(actions.find('[data-testid="usage-export-csv"]').exists()).toBe(true)

    await selects[0].vm.$emit('update:modelValue', 'x_twitter')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({ platform: 'x_twitter' }))
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ platform: 'x_twitter' }))
  })

  it('sorts usage records with server-side params and keeps export order consistent', async () => {
    HTMLAnchorElement.prototype.click = vi.fn()
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()
    listUsage.mockClear()
    getStats.mockClear()

    await wrapper.get('[data-testid="usage-sort-status"]').trigger('click')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'status',
      sort_order: 'asc',
    }))

    listUsage.mockClear()
    getStats.mockClear()

    await wrapper.get('[data-testid="usage-sort-cost"]').trigger('click')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'cost',
      sort_order: 'asc',
    }))
    expect(getStats).toHaveBeenCalledWith(defaultStatsParams())

    listUsage.mockClear()
    ;(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>).mockClear()
    listUsage.mockResolvedValueOnce({
      items: [
        {
          id: 10,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'main',
          operation: 'post',
          status: 'success',
          charge_status: 'charged',
          result_message: 'post succeeded',
          quantity: 1,
          cost: 1.5,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    })

    await wrapper.get('[data-testid="usage-export-csv"]').trigger('click')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 100,
      sort_by: 'cost',
      sort_order: 'asc',
    }))
    expect(HTMLAnchorElement.prototype.click).toHaveBeenCalled()
  })

  it('exports filtered SocialOps usage records as a safe CSV', async () => {
    HTMLAnchorElement.prototype.click = vi.fn()
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()
    listUsage.mockClear()
    ;(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>).mockClear()

    listUsage.mockResolvedValueOnce({
      items: [
        {
          id: 10,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: '=main',
          operation: 'post',
          status: 'success',
          charge_status: 'charged',
          result_message: 'post succeeded',
          quantity: 1,
          cost: 1.5,
          target: 'https://x.com/northwind/status/1',
          content: '=HYPERLINK("https://example.com")',
          payload: {
            post: {
              text: '=HYPERLINK("https://example.com")',
              media: [
                {
                  source: 'inline',
                  file_name: 'post-image-1.png',
                  content_type: 'image/png',
                  storage_key: 'social-task/private/post-image-1.png',
                },
              ],
            },
          },
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    })

    await wrapper.get('[data-testid="usage-export-csv"]').trigger('click')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 100,
      sort_by: 'time',
      sort_order: 'desc',
      start_date: expect.any(String),
      end_date: expect.any(String),
    }))
    expect(HTMLAnchorElement.prototype.click).toHaveBeenCalled()
    const blob = (globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>).mock.calls[0][0] as Blob
    const csv = await readBlobText(blob)

    expect(csv).toContain('"Platform","Operation","Account","Result","Cost","Summary","Time","Target","Content"')
    expect(csv).not.toContain('Charge Status')
    expect(csv).not.toContain('"Charged"')
    expect(csv).toContain('"Post"')
    expect(csv).toContain('"Twitter / X"')
    expect(csv).toContain('"\'=main"')
    expect(csv).toContain('"\'=HYPERLINK(""https://example.com"")"')
    expect(csv).not.toContain('storage_key')
    expect(csv).not.toContain('social-task/private')
    expect(csv).not.toContain('api_key')
    expect(csv).not.toContain('token')
  })

  it('shows an empty SocialOps usage state', async () => {
    listUsage.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    getStats.mockResolvedValue({ total_operations: 0, total_charged: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('No SocialOps operations yet')
    const emptyState = wrapper.get('[data-testid="usage-empty-state"]')
    expect(emptyState.find('svg').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-empty-clear-filters"]').exists()).toBe(false)
  })

  it('shows a loading state while usage records are being fetched', async () => {
    const listDeferred = createDeferred<{ items: unknown[]; total: number; page: number; page_size: number; pages: number }>()
    const statsDeferred = createDeferred<{ total_operations: number; success_count: number; failed_count: number; total_charged: number }>()
    listUsage.mockReturnValue(listDeferred.promise)
    getStats.mockReturnValue(statsDeferred.promise)

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await nextTick()

    expect(wrapper.text()).toContain('Loading usage records...')
    const loadingState = wrapper.get('[data-testid="usage-loading-state"]')
    expect(loadingState.find('svg').exists()).toBe(true)

    listDeferred.resolve({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    statsDeferred.resolve({ total_operations: 0, success_count: 0, failed_count: 0, total_charged: 0 })
    await flushPromises()
  })

  it('distinguishes filtered empty usage results from an empty account history', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()
    listUsage.mockClear()
    getStats.mockClear()
    listUsage.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    getStats.mockResolvedValue({ total_operations: 0, success_count: 0, failed_count: 0, total_charged: 0 })

    const selects = wrapper.findAllComponents({ name: 'Select' })
    await selects[1].vm.$emit('update:modelValue', 'post')
    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith(expect.objectContaining({ operation: 'post' }))
    expect(wrapper.text()).toContain('No usage records match the current filters.')
    const emptyState = wrapper.get('[data-testid="usage-empty-state"]')
    expect(emptyState.find('svg').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-empty-clear-filters"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('No SocialOps operations yet')
  })

  it('renders failed usage rows without task diagnostic payloads', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: 'authorization Bearer abc token=secret proxy=http://127.0.0.1:8080 trace_id=trace-123',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({ total_operations: 0, total_charged: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.get('[data-testid="usage-records-table"]').text()).not.toContain('Not Charged')
    expect(wrapper.text()).toContain('authorization Bearer abc token=secret proxy=http://127.0.0.1:8080 trace_id=trace-123')
  })

  it('localizes backend safe task result messages', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: '执行代理不可用，本次未扣费',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({ total_operations: 0, total_charged: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Execution proxy unavailable; not charged')
    expect(wrapper.text()).not.toContain('执行代理不可用，本次未扣费')
  })

  it('localizes exact avatar dimension failure messages', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 2,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'update_avatar',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: '头像图片尺寸必须为 400x400，本次未扣费',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({ total_operations: 0, total_charged: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Avatar image must be 400 × 400; not charged')
    expect(wrapper.text()).not.toContain('头像图片尺寸必须为 400x400，本次未扣费')
  })

  it('shows structured task summaries in usage rows when safe list payloads are present', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'post',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: '任务队列繁忙，本次未扣费',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: '2026-05-31T00:00:01Z',
          payload: {
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
          },
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({ total_operations: 0, total_charged: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Post · Text: hello world · Quote: https://x.com/northwind/status/2 · 1 media item(s)')
    expect(wrapper.text()).toContain('Task queue is busy; not charged')
    expect(wrapper.text()).not.toContain('No structured details')
  })

  it('does not render invalid task timestamps as dates', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'success',
          charge_status: 'charged',
          result_message: 'follow succeeded',
          quantity: 1,
          cost: 1.5,
          created_at: '2026-05-31T00:00:00Z',
          completed_at: 'not-a-date',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).not.toContain('Invalid Date')
  })

  it('reports load failures through the app store', async () => {
    listUsage.mockRejectedValue(new Error('network'))
    getStats.mockRejectedValue(new Error('network'))

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to load usage records')
    expect(wrapper.text()).toContain('Failed to load usage records')
    const errorState = wrapper.get('[data-testid="usage-error-state"]')
    expect(errorState.find('svg').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-retry-load"]').exists()).toBe(true)

    listUsage.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    getStats.mockResolvedValue({ total_operations: 0, success_count: 0, failed_count: 0, total_charged: 0 })

    await wrapper.get('[data-testid="usage-retry-load"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('No SocialOps operations yet')
    expect(wrapper.text()).not.toContain('Failed to load usage records')
  })

  it('opens a detail dialog with safe structured task history', async () => {
    getById.mockResolvedValue({
      id: 1,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'post',
      status: 'success',
      charge_status: 'charged',
      charge_source: 'subscription',
      proxy_snapshot: '{"id":8,"name":"proxy-a","endpoint":"http://proxy.local:8080","status":"online"}',
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
            },
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
              byte_size: 1234,
              width: 400,
              height: 400,
            },
          ],
          avatar: {
            source: 'inline',
            file_name: 'avatar.png',
            content_type: 'image/png',
            byte_size: 2048,
            width: 400,
            height: 400,
          },
        },
      },
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    await wrapper.get('[data-testid="usage-detail-button-1"]').trigger('click')
    await flushPromises()

    const bodyText = document.body.textContent ?? ''

    expect(getById).toHaveBeenCalledWith(1)
    expect(bodyText).toContain('Task Detail')
    expect(bodyText).toContain('Review the submitted configuration and execution summary.')
    expect(bodyText).toContain('Operation')
    expect(bodyText).toContain('Post')
    expect(bodyText).toContain('Platform')
    expect(bodyText).toContain('Twitter / X')
    expect(bodyText).toContain('Account')
    expect(bodyText).toContain('x-main')
    expect(bodyText).toContain('Status')
    expect(bodyText).toContain('Success')
    expect(bodyText).toContain('Charge status')
    expect(bodyText).toContain('Charged')
    expect(bodyText).toContain('Charge source')
    expect(bodyText).toContain('Subscription')
    expect(bodyText).toContain('Cost')
    expect(bodyText).toContain('$1.50')
    expect(bodyText).toContain('Result')
    expect(bodyText).toContain('post succeeded')
    expect(bodyText).toContain('Quantity')
    expect(bodyText).toContain('1')
    expect(document.body.querySelector('[data-testid="usage-detail-proxy"]')).not.toBeNull()
    expect(bodyText).toContain('Execution Proxy')
    expect(bodyText).toContain('Proxy name')
    expect(bodyText).toContain('proxy-a')
    expect(bodyText).toContain('Proxy endpoint')
    expect(bodyText).toContain('http://proxy.local:8080')
    expect(bodyText).toContain('Proxy status')
    expect(bodyText).toContain('Online')
    expect(document.body.querySelector('[data-testid="usage-detail-technical"]')).not.toBeNull()
    expect(bodyText).toContain('Technical Info')
    expect(bodyText).toContain('Billing request')
    expect(bodyText).toContain('sub:task-1')
    expect(bodyText).toContain('Idempotency key')
    expect(bodyText).toContain('usage-detail-1')
    const overviewText = document.body.querySelector('[data-testid="usage-detail-overview"]')?.textContent ?? ''
    const resultText = document.body.querySelector('[data-testid="usage-detail-result"]')?.textContent ?? ''
    const payloadText = document.body.querySelector('[data-testid="usage-detail-payload"]')?.textContent ?? ''
    expect(overviewText).toContain('Cost')
    expect(overviewText).toContain('$1.50')
    expect(resultText).toContain('post succeeded')
    expect(overviewText).not.toContain('Target')
    expect(overviewText).not.toContain('Content')
    expect(overviewText).not.toContain('Quote link')
    expect(payloadText).toContain('Target')
    expect(payloadText).toContain('https://x.com/northwind/status/1')
    expect(payloadText).toContain('Content')
    expect(payloadText).toContain('hello world')
    expect(payloadText).toContain('Quote link')
    expect(payloadText).toContain('https://x.com/northwind/status/2')
    expect(bodyText).toContain('Target')
    expect(bodyText).toContain('https://x.com/northwind/status/1')
    expect(bodyText).toContain('Content')
    expect(bodyText).toContain('hello world')
    expect(bodyText).toContain('Quote link')
    expect(bodyText).toContain('https://x.com/northwind/status/2')
    expect(bodyText).toContain('Template name')
    expect(bodyText).toContain('Rich post')
    expect(bodyText).toContain('Display name')
    expect(bodyText).toContain('Northwind Updates')
    expect(bodyText).toContain('Avatar')
    expect(bodyText).toContain('post-image-1.png')
    expect(bodyText).toContain('avatar.png')
    expect(bodyText).toContain('400 × 400')
    expect(bodyText).toContain('1,234 B')
    expect(bodyText).not.toContain('{"id":8')
    expect(bodyText).not.toContain('data:image/png;base64')
    expect(bodyText).not.toContain('storage_key')
  })

  it('renders plain endpoint proxy snapshots as a safe proxy endpoint row', async () => {
    getById.mockResolvedValue({
      id: 1,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'follow',
      status: 'failed',
      charge_status: 'not_charged',
      proxy_snapshot: 'http://proxy.local:8080',
      result_message: 'Task failed; diagnostic details are hidden',
      quantity: 1,
      cost: 0,
      created_at: '2026-05-31T00:00:00Z',
      completed_at: '2026-05-31T00:00:01Z',
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    await wrapper.get('[data-testid="usage-detail-button-1"]').trigger('click')
    await flushPromises()

    const bodyText = document.body.textContent ?? ''

    expect(bodyText).toContain('Proxy endpoint')
    expect(bodyText).toContain('http://proxy.local:8080')
    expect(bodyText).not.toContain('Proxy name')
    expect(bodyText).not.toContain('Proxy status')
  })

  it('loads blob previews for task-history media without exposing stored refs in detail JSON', async () => {
    getById.mockResolvedValue({
      id: 1,
      user_id: 7,
      social_account_id: 9,
      platform: 'x_twitter',
      account_name: 'x-main',
      operation: 'post',
      status: 'success',
      charge_status: 'charged',
      result_message: 'post succeeded',
      quantity: 1,
      cost: 1.5,
      created_at: '2026-05-31T00:00:00Z',
      completed_at: '2026-05-31T00:00:01Z',
      payload: {
        post: {
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
        avatar: {
          source: 'inline',
          file_name: 'avatar.png',
          content_type: 'image/png',
          byte_size: 2048,
          width: 400,
          height: 400,
        },
      },
      template_snapshot: {
        template_id: 'tmpl_1',
        template_name: 'Rich post',
        template_type: 'post',
        params: {
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
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: viewStubs,
      },
    })

    await flushPromises()

    await wrapper.get('[data-testid="usage-detail-button-1"]').trigger('click')
    await flushPromises()

    expect(previewTaskMedia).toHaveBeenCalledWith(1, { scope: 'payload', section: 'post', index: 0 })
    expect(previewTaskMedia).toHaveBeenCalledWith(1, { scope: 'payload', section: 'avatar' })
    expect(previewTaskMedia).toHaveBeenCalledWith(1, { scope: 'template', section: 'post', index: 0 })
    expect(globalThis.URL.createObjectURL as unknown as ReturnType<typeof vi.fn>).toHaveBeenCalled()

    const bodyText = document.body.textContent ?? ''
    expect(bodyText).not.toContain('storage_key')
    expect(bodyText).not.toContain('social-task/')
    expect(bodyText).not.toContain('data:image/png;base64')

    const preview = document.body.querySelector('[data-testid="usage-media-preview-payload-post-0"]') as HTMLImageElement | null
    expect(preview).not.toBeNull()
    expect(preview?.getAttribute('src')).toBe('blob:usage-media-preview')
  })
})
