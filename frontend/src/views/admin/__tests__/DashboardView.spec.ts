import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import DashboardView from '../DashboardView.vue'

const {
  getStats,
  getUsageTrend,
  getUserSpendingRanking,
  getSocialAccountStats,
  listSocialAccounts
} = vi.hoisted(() => ({
  getStats: vi.fn(),
  getUsageTrend: vi.fn(),
  getUserSpendingRanking: vi.fn(),
  getSocialAccountStats: vi.fn(),
  listSocialAccounts: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getStats,
      getUsageTrend,
      getUserSpendingRanking
    },
    accountWorkbench: {
      getStats: getSocialAccountStats,
      list: listSocialAccounts
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
  const messages: Record<string, string> = {
    'admin.dashboard.title': 'Admin Dashboard',
    'admin.dashboard.opsEyebrow': 'SocialOps operations',
    'admin.dashboard.opsDescription': 'Live summary of the total account pool, task execution, successful charges, and user activity.',
    'admin.dashboard.submitTask': 'Submit task',
    'admin.dashboard.loading': 'Loading dashboard data...',
    'admin.dashboard.taskTrendTitle': '30-day task trend',
    'admin.dashboard.taskTrendDescription': 'Aggregated by task execution count and successful charged amount.',
    'admin.dashboard.granularityDay': 'Granularity: day',
    'admin.dashboard.noTrendData': 'No trend data',
    'admin.dashboard.taskCount': '{count} tasks',
    'admin.dashboard.successfulCharges': 'Successful charges {amount}',
    'admin.dashboard.resourceHealth': 'Resource health',
    'admin.dashboard.resourceHealthDescription': 'Availability based on account pool and proxy status.',
    'admin.dashboard.accountAvailability': 'Account availability',
    'admin.dashboard.availableAccounts': 'Available accounts',
    'admin.dashboard.issueAccounts': 'Issue accounts',
    'admin.dashboard.platformDistribution': 'Platform distribution',
    'admin.dashboard.platformDistributionDescription': 'Uses recent task records first, then account pool samples when no tasks exist.',
    'admin.dashboard.platformDistributionEmpty': 'No platform data',
    'admin.dashboard.accountStatusDistribution': 'Account status distribution',
    'admin.dashboard.accountStatusDistributionDescription': 'Counts current sampled account pool status.',
    'admin.dashboard.accountStatusDistributionEmpty': 'No account status data',
    'admin.dashboard.userSpendingRanking': 'User spending ranking',
    'admin.dashboard.userSpendingRankingDescription': 'Sorted by successful charges in the last 30 days.',
    'admin.dashboard.taskCountShort': '{count} tasks',
    'admin.dashboard.noSpendingRanking': 'No spending ranking',
    'admin.dashboard.recentExecutionSummary': 'Recent execution summary',
    'admin.dashboard.recentExecutionSummaryDescription': 'Daily task count and successful charged amount.',
    'admin.dashboard.date': 'Date',
    'admin.dashboard.taskCountColumn': 'Tasks',
    'admin.dashboard.successfulChargeColumn': 'Successful charges',
    'admin.dashboard.noExecutionSummary': 'No execution summary',
    'admin.dashboard.quickEntries': 'Quick entries',
    'admin.dashboard.totalAccountPool': 'Total account pool',
    'admin.dashboard.assignedAccountsMeta': '{count} assigned',
    'admin.dashboard.allocatableAccounts': 'Allocatable accounts',
    'admin.dashboard.storedAccountsMeta': '{count} stored',
    'admin.dashboard.todayTasks': 'Tasks today',
    'admin.dashboard.recentRpmMeta': 'Last 5 min {count}/min',
    'admin.dashboard.totalTasks': 'Total tasks',
    'admin.dashboard.executionRecordsMeta': 'Successful and failed tasks both count as execution records',
    'admin.dashboard.todaySuccessfulCharges': 'Successful charges today',
    'admin.dashboard.cumulativeChargesMeta': 'Total {amount}',
    'admin.dashboard.activeUsers': 'Active users',
    'admin.dashboard.userGrowthMeta': 'Total users {total}, new today {today}',
    'admin.dashboard.quickLinks.accounts.label': 'Account management',
    'admin.dashboard.quickLinks.accounts.description': 'Match, bind, and submit social tasks',
    'admin.dashboard.quickLinks.totalAccounts.label': 'Total account pool',
    'admin.dashboard.quickLinks.totalAccounts.description': 'Maintain account assignment status',
    'admin.dashboard.loadIncomplete': 'Dashboard data is incomplete. Please refresh later.',
    'admin.dashboard.unknownPlatform': 'Unknown platform',
    'admin.dashboard.accountStatuses.available': 'Healthy',
    'admin.dashboard.accountStatuses.pending_check': 'Pending check',
    'admin.dashboard.accountStatuses.limited': 'Limited',
    'admin.dashboard.accountStatuses.invalid': 'Invalid',
    'admin.dashboard.accountStatuses.not_stored': 'Not stored',
    'admin.dashboard.userFallback': 'User #{id}',
    'common.refresh': 'Refresh'
  }
  function translate(key: string, params?: Record<string, unknown> | string) {
    let message = messages[key] ?? (typeof params === 'string' ? params : key)
    if (params && typeof params === 'object') {
      Object.entries(params).forEach(([name, value]) => {
        message = message.replace(`{${name}}`, String(value))
      })
    }
    return message
  }
  return {
    ...actual,
    useI18n: () => ({
      t: translate
    })
  }
})

describe('admin DashboardView', () => {
  beforeEach(() => {
    getStats.mockReset()
    getUsageTrend.mockReset()
    getUserSpendingRanking.mockReset()
    getSocialAccountStats.mockReset()
    listSocialAccounts.mockReset()

    getStats.mockResolvedValue({
      total_users: 42,
      active_users: 8,
      today_new_users: 3,
      today_requests: 17,
      total_requests: 120,
      today_actual_cost: 2.4,
      total_actual_cost: 18.75,
      rpm: 2
    })
    getUsageTrend.mockResolvedValue([
      { date: '2026-06-01', requests: 9, actual_cost: 1.1 },
      { date: '2026-06-02', requests: 17, actual_cost: 2.4 }
    ])
    getUserSpendingRanking.mockResolvedValue({
      ranking: [{ user_id: 7, email: 'operator@example.com', requests: 11, actual_cost: 3.5 }]
    })
    getSocialAccountStats.mockResolvedValue({
      total: 12,
      stored: 10,
      available: 7
    })
    listSocialAccounts.mockResolvedValue({
      items: [{ id: 88, name: 'x-main', platform: 'x_twitter', account_status: 'available', task_status: 'idle', created_at: '', updated_at: '' }],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1
    })
  })

  it('loads SocialOps operations dashboard data', async () => {
    const wrapper = mount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          RouterLink: { props: ['to'], template: '<a><slot /></a>' },
        }
      }
    })

    await flushPromises()

    expect(getStats).toHaveBeenCalledTimes(1)
    expect(getSocialAccountStats).toHaveBeenCalledTimes(1)
    expect(getUsageTrend).toHaveBeenCalledWith({ granularity: 'day' })
    expect(getUserSpendingRanking).toHaveBeenCalledWith({ limit: 5 })
    expect(listSocialAccounts).toHaveBeenCalledWith({ page: 1, page_size: 100 })
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('7')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('17')
    expect(wrapper.text()).toContain('Total account pool')
    expect(wrapper.text()).toContain('30-day task trend')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('operator@example.com')
    expect(wrapper.text()).not.toContain('Proxy management')
    expect(wrapper.text()).not.toMatch(/[\u4e00-\u9fff]/)
  })
})
