import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import DashboardView from '../DashboardView.vue'

const {
  getDashboardStats,
  getDashboardTrend,
  listMyAccounts,
  listUsage,
  showError
} = vi.hoisted(() => ({
  getDashboardStats: vi.fn(),
  getDashboardTrend: vi.fn(),
  listMyAccounts: vi.fn(),
  listUsage: vi.fn(),
  showError: vi.fn()
}))

const messages: Record<string, string> = {
  'dashboard.title': 'Dashboard',
  'dashboard.eyebrow': 'My SocialOps',
  'dashboard.description': 'Review account bindings, task execution results, and successful charges.',
  'dashboard.viewLogs': 'View logs',
  'dashboard.loadingMine': 'Loading my dashboard...',
  'dashboard.trendTitle': '30-day task trend',
  'dashboard.trendDescription': 'Aggregated by your social task execution time.',
  'dashboard.granularityDay': 'Granularity: day',
  'dashboard.trendEmpty': 'No trend data',
  'dashboard.trendCount': '{count} tasks',
  'dashboard.chargedAmount': 'Successful charges {amount}',
  'dashboard.platformDistribution': 'Platform distribution',
  'dashboard.platformDistributionDescription': 'Uses task records first, then assigned accounts when no tasks exist.',
  'dashboard.platformDistributionEmpty': 'No platform data',
  'dashboard.recentUsage': 'Recent usage records',
  'dashboard.recentUsageDescription': 'Shows recent operations, statuses, quantity, and cost.',
  'dashboard.allRecords': 'All records',
  'dashboard.noUsageRecords': 'No usage records',
  'dashboard.quickEntries': 'Quick entries',
  'dashboard.quickEntriesDescription': 'Go to common account, log, and billing pages.',
  'dashboard.boundAccounts': 'Assigned accounts',
  'dashboard.sampledAccounts': 'Sampled {count}',
  'dashboard.coveredPlatforms': '{count} platforms covered',
  'dashboard.executableAccounts': 'Executable accounts',
  'dashboard.totalPoolAccounts': 'Assigned from the total account pool',
  'dashboard.todayRequests': 'Tasks today',
  'dashboard.totalRequests': 'Total tasks',
  'dashboard.recentRpm': 'Last 5 min {count}/min',
  'dashboard.totalCharged': 'Total charged',
  'dashboard.successOnlyBilling': 'Only successful tasks are charged',
  'dashboard.recentSuccessRate': 'Recent success rate',
  'dashboard.successFailureMeta': '{success} succeeded / {failed} failed',
  'dashboard.unknownPlatform': 'Unknown platform',
  'dashboard.loadIncomplete': 'My dashboard data is incomplete. Please refresh later.',
  'dashboard.quickLinks.taskLogs.label': 'Task logs',
  'dashboard.quickLinks.taskLogs.description': 'Review success, failure, and charge records',
  'dashboard.quickLinks.purchase.label': 'Recharge',
  'dashboard.quickLinks.purchase.description': 'Add task execution balance or plans',
  'dashboard.quickLinks.subscriptions.label': 'My subscriptions',
  'dashboard.quickLinks.subscriptions.description': 'Review available benefits and validity',
  'dashboard.quickLinks.profile.label': 'Profile',
  'dashboard.quickLinks.profile.description': 'Maintain account information and security settings',
  'usage.operation': 'Operation',
  'usage.platform': 'Platform',
  'usage.account': 'Account',
  'usage.status': 'Status',
  'usage.quantity': 'Quantity',
  'usage.chargeStatus': 'Charge Status',
  'usage.cost': 'Cost',
  'usage.result': 'Result',
  'usage.time': 'Time',
  'usage.safeResult': 'Task failed; diagnostic details are hidden',
  'usage.taskResults.proxyUnavailable': 'Execution proxy unavailable; not charged',
  'usage.taskResults.queueBusy': 'Task queue is busy; not charged',
  'usage.taskResults.avatarSizeInvalid': 'Avatar image must be 400 × 400; not charged',
  'usage.taskResults.bannerSizeInvalid': 'Banner image must be 1500 × 500; not charged',
  'usage.actions.like': 'Like',
  'usage.actions.follow': 'Follow',
  'usage.actions.login_check': 'Login Check',
  'usage.actions.post': 'Post',
  'usage.statuses.success': 'Succeeded',
  'usage.statuses.failed': 'Failed',
  'usage.platforms.x_twitter': 'Twitter / X',
  'usage.chargeStatuses.charged': 'Charged',
  'usage.chargeStatuses.not_charged': 'Not Charged',
  'usage.taskSummaryTarget': 'Target: {value}',
  'usage.taskSummaryContent': 'Text: {value}',
  'usage.taskSummaryQuote': 'Quote: {value}',
  'usage.taskSummaryMedia': '{count} media item(s)',
  'usage.taskSummaryProfile': '{count} profile field(s)',
  'usage.taskSummaryAvatar': 'Avatar image ready',
  'usage.taskSummaryBanner': 'Banner image ready',
  'usage.taskSummaryNoDetails': 'No structured details',
  'common.unknown': 'Unknown',
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

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardStats,
    getDashboardTrend,
    list: listUsage
  }
}))

vi.mock('@/api/accountWorkbench', () => ({
  accountWorkbenchAdminAPI: {},
  default: {
    listMyAccounts
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: translate
    })
  }
})

describe('user DashboardView', () => {
  beforeEach(() => {
    getDashboardStats.mockReset()
    getDashboardTrend.mockReset()
    listMyAccounts.mockReset()
    listUsage.mockReset()
    showError.mockReset()

    getDashboardStats.mockResolvedValue({
      total_requests: 12,
      today_requests: 3,
      total_actual_cost: 1.25,
      today_actual_cost: 0.25,
      rpm: 1,
      by_platform: [
        { platform: 'x_twitter', total_requests: 10, today_requests: 3, total_actual_cost: 1.2 }
      ]
    })
    getDashboardTrend.mockResolvedValue([
      { date: '2026-06-01', requests: 5, actual_cost: 0.5 },
      { date: '2026-06-02', requests: 7, actual_cost: 0.75 }
    ])
    listMyAccounts.mockResolvedValue({
      items: [
        { id: 1, name: 'x-main', platform: 'x_twitter', account_status: 'available', task_status: 'idle', created_at: '', updated_at: '' },
        { id: 2, name: 'x-backup', platform: 'x_twitter', account_status: 'limited', task_status: 'idle', created_at: '', updated_at: '' }
      ],
      total: 2,
      page: 1,
      page_size: 100,
      pages: 1
    })
    listUsage.mockResolvedValue({
      items: [
        {
          id: 1,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'like',
          status: 'success',
          charge_status: 'charged',
          result_message: 'like succeeded',
          quantity: 1,
          cost: 0.2,
          created_at: '2026-06-02T08:00:00Z',
          completed_at: '2026-06-02T08:00:01Z'
        },
        {
          id: 2,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'follow',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: 'target not found',
          quantity: 1,
          cost: 0,
          created_at: '2026-06-02T08:01:00Z',
          completed_at: '2026-06-02T08:01:01Z'
        },
        {
          id: 3,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'login_check',
          status: 'success',
          charge_status: 'charged',
          result_message: 'login check succeeded',
          quantity: 1,
          cost: 0,
          created_at: '2026-06-02T08:02:00Z',
          completed_at: '2026-06-02T08:02:01Z'
        }
      ],
      total: 2,
      page: 1,
      page_size: 8,
      pages: 1
    })
  })

  it('loads personal SocialOps task dashboard data', async () => {
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

    expect(getDashboardStats).toHaveBeenCalledTimes(1)
    expect(getDashboardTrend).toHaveBeenCalledWith({ granularity: 'day' })
    expect(listMyAccounts).toHaveBeenCalledWith({ page: 1, page_size: 100 })
    expect(listUsage).toHaveBeenCalledWith({ page: 1, page_size: 8 })
    expect(wrapper.text()).toContain('My SocialOps')
    expect(wrapper.text()).toContain('Assigned accounts')
    expect(wrapper.text()).toContain('Executable accounts')
    expect(wrapper.text()).toContain('30-day task trend')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Like')
    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Login Check')
    expect(wrapper.text()).toContain('Charged')
    expect(wrapper.text()).toContain('like succeeded')
    expect(wrapper.text()).toContain('$0.20')
    expect(wrapper.text()).not.toContain('我的 SocialOps')
    expect(wrapper.text()).not.toContain('近 30 天任务趋势')
    expect(wrapper.text()).not.toContain('快捷入口')
  })

  it('renders recent usage rows without task diagnostic payloads', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 3,
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
          created_at: '2026-06-02T08:01:00Z',
          completed_at: '2026-06-02T08:01:01Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 8,
      pages: 1
    })

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

    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Not Charged')
    expect(wrapper.text()).toContain('Task failed; diagnostic details are hidden')
    expect(wrapper.text()).not.toContain('authorization')
    expect(wrapper.text()).not.toContain('Bearer abc')
    expect(wrapper.text()).not.toContain('token=secret')
    expect(wrapper.text()).not.toContain('127.0.0.1')
    expect(wrapper.text()).not.toContain('trace_id=trace-123')
  })

  it('localizes backend safe task result messages in recent usage rows', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 3,
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
          created_at: '2026-06-02T08:01:00Z',
          completed_at: '2026-06-02T08:01:01Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 8,
      pages: 1
    })

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

    expect(wrapper.text()).toContain('Execution proxy unavailable; not charged')
    expect(wrapper.text()).not.toContain('执行代理不可用，本次未扣费')
  })

  it('localizes exact banner dimension failure messages in recent usage rows', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 4,
          user_id: 7,
          social_account_id: 9,
          platform: 'x_twitter',
          account_name: 'x-main',
          operation: 'update_banner',
          status: 'failed',
          charge_status: 'not_charged',
          result_message: '背景图图片尺寸必须为 1500x500，本次未扣费',
          quantity: 1,
          cost: 0,
          created_at: '2026-06-02T08:01:00Z',
          completed_at: '2026-06-02T08:01:01Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 8,
      pages: 1
    })

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

    expect(wrapper.text()).toContain('Banner image must be 1500 × 500; not charged')
    expect(wrapper.text()).not.toContain('背景图图片尺寸必须为 1500x500，本次未扣费')
  })

  it('shows structured task summaries in recent usage rows when safe list payloads are present', async () => {
    listUsage.mockResolvedValue({
      items: [
        {
          id: 3,
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
          created_at: '2026-06-02T08:01:00Z',
          completed_at: '2026-06-02T08:01:01Z',
          payload: {
            post: {
              text: 'hello world',
              quote_post_url: 'https://x.com/openai/status/2',
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
        }
      ],
      total: 1,
      page: 1,
      page_size: 8,
      pages: 1
    })

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

    expect(wrapper.text()).toContain('Post · Text: hello world · Quote: https://x.com/openai/status/2 · 1 media item(s)')
    expect(wrapper.text()).toContain('Task queue is busy; not charged')
    expect(wrapper.text()).not.toContain('No structured details')
  })

  it('keeps recent usage table constrained inside the dashboard grid', async () => {
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

    const recentUsageSection = wrapper.find('section.card.overflow-hidden')
    expect(recentUsageSection.exists()).toBe(true)
    expect(recentUsageSection.classes()).toContain('min-w-0')

    const cardSections = wrapper.findAll('section.card')
    const quickEntriesSection = cardSections[cardSections.length - 1]
    expect(quickEntriesSection.classes()).toContain('min-w-0')
  })

  it('reports dashboard load failures through localized copy', async () => {
    getDashboardStats.mockRejectedValue(new Error('network'))
    getDashboardTrend.mockRejectedValue(new Error('network'))
    listMyAccounts.mockRejectedValue(new Error('network'))
    listUsage.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 8, pages: 0 })

    mount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          RouterLink: { props: ['to'], template: '<a><slot /></a>' },
        }
      }
    })

    await flushPromises()

    expect(showError).toHaveBeenCalledWith('My dashboard data is incomplete. Please refresh later.')
  })
})
