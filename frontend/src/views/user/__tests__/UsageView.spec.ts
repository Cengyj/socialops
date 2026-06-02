import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const { listMyTaskLogs, showError } = vi.hoisted(() => ({
  listMyTaskLogs: vi.fn(),
  showError: vi.fn(),
}))

const messages: Record<string, string> = {
  'usage.title': 'Usage Records',
  'usage.description': 'Track SocialOps operation history',
  'usage.records': 'Operation Records',
  'usage.operation': 'Operation',
  'usage.status': 'Status',
  'usage.quantity': 'Quantity',
  'usage.chargeStatus': 'Charge Status',
  'usage.cost': 'Cost',
  'usage.time': 'Time',
  'usage.empty': 'No SocialOps operations yet',
  'usage.totalOperations': 'Total Operations',
  'usage.totalQuantity': 'Total Quantity',
  'usage.successCount': 'Successful',
  'usage.totalCost': 'Total Cost',
  'usage.failedToLoad': 'Failed to load usage records',
  'common.refresh': 'Refresh',
}

vi.mock('@/api/socialAccounts', () => ({
  default: {
    listMyTaskLogs,
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
    listMyTaskLogs.mockReset()
    showError.mockReset()

    listMyTaskLogs.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 2,
          action: 'follow',
          status: 'success',
          charged_amount: 1.5,
          charge_status: 'charged',
          price: 1.5,
          created_at: '2026-05-31T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
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

    expect(listMyTaskLogs).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(wrapper.text()).toContain('Total Operations')
    expect(wrapper.text()).toContain('Total Quantity')
    expect(wrapper.text()).toContain('follow')
    expect(wrapper.text()).toContain('success')
    expect(wrapper.text()).toContain('$1.50')
  })

  it('shows an empty SocialOps usage state', async () => {
    listMyTaskLogs.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })

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

  it('reports load failures through the app store', async () => {
    listMyTaskLogs.mockRejectedValue(new Error('network'))

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
