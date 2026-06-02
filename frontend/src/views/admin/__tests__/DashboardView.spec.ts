import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import DashboardView from '../DashboardView.vue'

const { getStats, getSocialAccountStats } = vi.hoisted(() => ({
  getStats: vi.fn(),
  getSocialAccountStats: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getStats
    },
    socialAccounts: {
      getStats: getSocialAccountStats
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('admin DashboardView', () => {
  beforeEach(() => {
    getStats.mockReset()
    getSocialAccountStats.mockReset()

    getStats.mockResolvedValue({
      total_users: 42,
      active_users: 8,
      today_requests: 17
    })
    getSocialAccountStats.mockResolvedValue({
      total: 12,
      stored: 10,
      available: 7
    })
  })

  it('loads SocialOps dashboard and social account summary stats', async () => {
    const wrapper = mount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
        }
      }
    })

    await flushPromises()

    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getSocialAccountStats).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('7')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('17')
  })
})
