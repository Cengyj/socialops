import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const { list, getStats, createCleanupTask, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  getStats: vi.fn(),
  createCleanupTask: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

const messages: Record<string, string> = {
  'admin.usage.title': 'Usage Records',
  'admin.usage.description': 'Review SocialOps operation usage',
  'admin.usage.records': 'Operation Records',
  'admin.usage.user': 'User',
  'admin.usage.empty': 'No usage records yet',
  'admin.usage.failedToLoad': 'Failed to load admin usage records',
  'admin.usage.cleanup.button': 'Cleanup',
  'admin.usage.cleanupQueued': 'Cleanup queued',
  'admin.usage.cleanupFailed': 'Failed to queue cleanup',
  'usage.operation': 'Operation',
  'usage.status': 'Status',
  'usage.quantity': 'Quantity',
  'usage.time': 'Time',
  'usage.totalOperations': 'Total Operations',
  'usage.totalQuantity': 'Total Quantity',
  'usage.successCount': 'Successful',
  'usage.failedCount': 'Failed',
  'common.refresh': 'Refresh',
}

vi.mock('@/api/admin/usage', () => ({
  default: {
    list,
    getStats,
    createCleanupTask,
  },
  adminUsageAPI: {
    list,
    getStats,
    createCleanupTask,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
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

describe('admin UsageView', () => {
  beforeEach(() => {
    list.mockReset()
    getStats.mockReset()
    createCleanupTask.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    list.mockResolvedValue({
      items: [
        {
          id: 10,
          user_id: 42,
          operation: 'dm',
          status: 'failed',
          quantity: 1,
          cost: 0,
          created_at: '2026-05-31T01:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getStats.mockResolvedValue({
      total_operations: 1,
      total_quantity: 1,
      success_count: 0,
      failed_count: 1,
    })
    createCleanupTask.mockResolvedValue({ created: false, message: 'Cleanup not configured' })
  })

  it('renders admin SocialOps operation usage records', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(list).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(getStats).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('#42')
    expect(wrapper.text()).toContain('dm')
    expect(wrapper.text()).toContain('failed')
    expect(wrapper.text()).toContain('Failed')
  })

  it('queues usage cleanup through the SocialOps skeleton endpoint', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'Cleanup')?.trigger('click')
    await flushPromises()

    expect(createCleanupTask).toHaveBeenCalledWith({})
    expect(showSuccess).toHaveBeenCalledWith('Cleanup not configured')
  })

  it('reports admin usage load failures', async () => {
    list.mockRejectedValue(new Error('network'))

    mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to load admin usage records')
  })
})
