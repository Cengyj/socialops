import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RiskControlView from '../RiskControlView.vue'

const { getStatus, listLogs, showError } = vi.hoisted(() => ({
  getStatus: vi.fn(),
  listLogs: vi.fn(),
  showError: vi.fn(),
}))

const messages: Record<string, string> = {
  'admin.riskControl.title': 'Risk Control',
  'admin.riskControl.description': 'Review SocialOps account risk signals',
  'admin.riskControl.runtimeStatus': 'Runtime Status',
  'admin.riskControl.accountRules': 'Account Rules',
  'admin.riskControl.recentEvents': 'Recent Events',
  'admin.riskControl.skeletonMessage': 'SocialOps risk control backend is not configured yet',
  'admin.riskControl.records': 'Risk Events',
  'admin.riskControl.scope': 'Scope',
  'admin.riskControl.target': 'Target',
  'admin.riskControl.status': 'Status',
  'admin.riskControl.enabled': 'Enabled',
  'admin.riskControl.disabled': 'Disabled',
  'admin.riskControl.empty': 'No risk events yet',
  'admin.riskControl.loadFailed': 'Failed to load risk control',
  'usage.time': 'Time',
  'common.refresh': 'Refresh',
}

vi.mock('@/api/admin/riskControl', () => ({
  default: {
    getStatus,
    listLogs,
  },
  adminRiskControlAPI: {
    getStatus,
    listLogs,
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

describe('admin RiskControlView', () => {
  beforeEach(() => {
    getStatus.mockReset()
    listLogs.mockReset()
    showError.mockReset()

    getStatus.mockResolvedValue({
      enabled: false,
      status: 'disabled',
      message: 'SocialOps risk control backend is not configured yet',
    })
    listLogs.mockResolvedValue({
      items: [
        {
          id: 1,
          scope: 'account',
          target: '@socialops-demo',
          status: 'observed',
          created_at: '2026-05-31T02:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('renders the SocialOps-safe risk-control skeleton', async () => {
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(getStatus).toHaveBeenCalledTimes(1)
    expect(listLogs).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(wrapper.text()).toContain('Disabled')
    expect(wrapper.text()).toContain('SocialOps risk control backend is not configured yet')
    expect(wrapper.text()).toContain('account')
    expect(wrapper.text()).toContain('@socialops-demo')
    expect(wrapper.text()).toContain('observed')
  })

  it('reports risk-control load failures', async () => {
    getStatus.mockRejectedValue(new Error('network'))

    mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to load risk control')
  })
})
