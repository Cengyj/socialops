import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const { listUsage, getStats, showError } = vi.hoisted(() => ({
  listUsage: vi.fn(),
  getStats: vi.fn(),
  showError: vi.fn(),
}))

const messages: Record<string, string> = {
  'usage.title': 'Usage Records',
  'usage.description': 'Track SocialOps operation history',
  'usage.records': 'Operation Records',
  'usage.operation': 'Operation',
  'usage.platform': 'Platform',
  'usage.account': 'Account',
  'usage.status': 'Status',
  'usage.quantity': 'Quantity',
  'usage.chargeStatus': 'Charge Status',
  'usage.cost': 'Cost',
  'usage.result': 'Result',
  'usage.time': 'Time',
  'usage.empty': 'No SocialOps operations yet',
  'usage.totalOperations': 'Total Operations',
  'usage.totalQuantity': 'Total Quantity',
  'usage.successCount': 'Successful',
  'usage.totalCost': 'Total Cost',
  'usage.failedToLoad': 'Failed to load usage records',
  'usage.safeResult': 'Task failed; diagnostic details are hidden',
  'usage.taskResults.proxyUnavailable': 'Execution proxy unavailable; not charged',
  'usage.platforms.x_twitter': 'Twitter / X',
  'usage.actions.follow': 'Follow',
  'usage.statuses.success': 'Succeeded',
  'usage.chargeStatuses.charged': 'Charged',
  'common.refresh': 'Refresh',
}

vi.mock('@/api/usage', () => ({
  usageAPI: {
    list: listUsage,
    getStats,
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
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }

describe('user UsageView', () => {
  beforeEach(() => {
    listUsage.mockReset()
    getStats.mockReset()
    showError.mockReset()

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
    getStats.mockResolvedValueOnce({
      total_requests: 34,
      total_tokens: 34,
      total_actual_cost: 8.25,
    })
    getStats.mockResolvedValueOnce({
      total_requests: 21,
    })
  })

  it('renders SocialOps operation usage records and stats', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(listUsage).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(getStats).toHaveBeenNthCalledWith(1)
    expect(getStats).toHaveBeenNthCalledWith(2, { status: 'success' })
    expect(wrapper.text()).toContain('Total Operations')
    expect(wrapper.text()).toContain('Total Quantity')
    expect(wrapper.text()).toContain('34')
    expect(wrapper.text()).toContain('21')
    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Succeeded')
    expect(wrapper.text()).toContain('Charged')
    expect(wrapper.text()).toContain('follow succeeded')
    expect(wrapper.text()).toContain('$8.25')
  })

  it('shows an empty SocialOps usage state', async () => {
    listUsage.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    getStats.mockResolvedValue({ total_requests: 0, total_tokens: 0, total_actual_cost: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('No SocialOps operations yet')
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
    getStats.mockResolvedValue({ total_requests: 0, total_tokens: 0, total_actual_cost: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Not Charged')
    expect(wrapper.text()).toContain('Task failed; diagnostic details are hidden')
    expect(wrapper.text()).not.toContain('authorization')
    expect(wrapper.text()).not.toContain('Bearer abc')
    expect(wrapper.text()).not.toContain('token=secret')
    expect(wrapper.text()).not.toContain('127.0.0.1')
    expect(wrapper.text()).not.toContain('trace-123')
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
    getStats.mockResolvedValue({ total_requests: 0, total_tokens: 0, total_actual_cost: 0 })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Execution proxy unavailable; not charged')
    expect(wrapper.text()).not.toContain('执行代理不可用，本次未扣费')
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
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).not.toContain('Invalid Date')
  })

  it('reports load failures through the app store', async () => {
    listUsage.mockRejectedValue(new Error('network'))
    getStats.mockRejectedValue(new Error('network'))

    mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to load usage records')
  })
})
