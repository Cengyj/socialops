import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageRecordsTable from '../UsageRecordsTable.vue'

const messages: Record<string, string> = {
  'usage.records': 'Operation Records',
  'usage.platform': 'Platform',
  'usage.operation': 'Operation',
  'usage.account': 'Account',
  'usage.result': 'Result',
  'usage.cost': 'Cost',
  'usage.summary': 'Summary',
  'usage.time': 'Time',
  'usage.loading': 'Loading usage records...',
  'usage.failedToLoad': 'Failed to load usage records',
  'usage.empty': 'No SocialOps operations yet',
  'usage.emptyFiltered': 'No usage records match the current filters.',
  'usage.filters.clear': 'Clear filters',
  'usage.actions.viewDetails': 'View details',
  'usage.actions.follow': 'Follow',
  'usage.statuses.success': 'Success',
  'usage.statuses.failed': 'Failed',
  'usage.platforms.x_twitter': 'Twitter / X',
  'usage.safeResult': 'Task failed; diagnostic details are hidden',
  'usage.taskSummaryTarget': 'Target: {value}',
  'usage.taskSummaryNoDetails': 'No structured details',
  'common.actions': 'Actions',
  'common.refresh': 'Refresh',
  'common.unknown': 'Unknown',
}

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

const PaginationStub = {
  name: 'Pagination',
  props: ['total', 'page', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<button type="button" data-testid="usage-pagination" @click="$emit(\'update:page\', 2); $emit(\'update:pageSize\', 50)">Pagination</button>',
}

const baseProps = {
  loading: false,
  loadError: false,
  hasActiveFilters: false,
  totalRows: 1,
  page: 1,
  pageSize: 20,
  sortBy: 'time' as const,
  sortOrder: 'desc' as const,
  detailLoading: false,
  activeDetailId: null,
}

const usageRow = {
  id: 1,
  user_id: 7,
  social_account_id: 9,
  platform: 'x_twitter',
  account_name: 'x-main',
  operation: 'follow',
  status: 'success',
  quantity: 1,
  cost: 0.1,
  charge_status: 'charged',
  target: '@northwind',
  result_message: 'follow succeeded',
  created_at: '2026-06-01T00:00:00Z',
  completed_at: '2026-06-01T00:00:01Z',
}

describe('UsageRecordsTable', () => {
  it('renders the SocialOps usage table columns and row content without charge status', async () => {
    const wrapper = mount(UsageRecordsTable, {
      props: {
        ...baseProps,
        rows: [usageRow],
      },
      global: {
        stubs: { Pagination: PaginationStub },
      },
    })

    const headers = wrapper.findAll('[data-testid="usage-records-table"] thead th').map(header => header.text())
    expect(headers).toEqual([
      'Platform',
      'Operation',
      'Account',
      'Result',
      'Cost',
      'Summary',
      'Time',
      'Actions',
    ])
    expect(headers).not.toContain('Charge Status')
    expect(wrapper.text()).toContain('Twitter / X')
    expect(wrapper.text()).toContain('Follow')
    expect(wrapper.text()).toContain('x-main')
    expect(wrapper.text()).toContain('Success')
    expect(wrapper.text()).toContain('$0.10')
    expect(wrapper.text()).toContain('follow succeeded')
    expect(wrapper.text()).not.toContain('Charged')
    const costCell = wrapper.findAll('[data-testid="usage-records-table"] tbody td')[4]
    expect(costCell.classes()).toEqual(expect.arrayContaining([
      'font-medium',
      'tabular-nums',
      'text-green-600',
      'dark:text-green-400',
    ]))
    expect(wrapper.get('[data-testid="usage-records-table"] thead th').classes()).toContain('align-middle')
    expect(wrapper.get('[data-testid="usage-records-table"] thead th').classes()).toContain('font-semibold')
    expect(wrapper.get('[data-testid="usage-records-table"] tbody td').classes()).toContain('align-middle')
    expect(wrapper.find('[data-testid="usage-sort-platform"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-sort-operation"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-sort-account"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-sort-status"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-sort-cost"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="usage-sort-time"]').exists()).toBe(true)

    await wrapper.get('[data-testid="usage-detail-button-1"]').trigger('click')
    await wrapper.get('[data-testid="usage-pagination"]').trigger('click')

    expect(wrapper.emitted('open-detail')?.[0]).toEqual([1])
    expect(wrapper.emitted('update:page')?.[0]).toEqual([2])
    expect(wrapper.emitted('update:pageSize')?.[0]).toEqual([50])
  })

  it('uses existing danger badge styling for failed usage rows', () => {
    const wrapper = mount(UsageRecordsTable, {
      props: {
        ...baseProps,
        rows: [{
          ...usageRow,
          status: 'failed',
          cost: 0,
          result_message: 'follow failed',
        }],
      },
      global: {
        stubs: { Pagination: PaginationStub },
      },
    })

    const statusBadge = wrapper.get('[data-testid="usage-records-table"] tbody td:nth-child(4) .badge')
    expect(statusBadge.classes()).toContain('badge-danger')
  })

  it('emits stable server-side sort changes for supported SocialOps columns', async () => {
    const wrapper = mount(UsageRecordsTable, {
      props: {
        ...baseProps,
        rows: [usageRow],
      },
      global: {
        stubs: { Pagination: PaginationStub },
      },
    })

    await wrapper.get('[data-testid="usage-sort-platform"]').trigger('click')
    await wrapper.get('[data-testid="usage-sort-operation"]').trigger('click')
    await wrapper.setProps({ sortBy: 'operation', sortOrder: 'asc' })
    await wrapper.get('[data-testid="usage-sort-operation"]').trigger('click')
    await wrapper.get('[data-testid="usage-sort-account"]').trigger('click')
    await wrapper.get('[data-testid="usage-sort-status"]').trigger('click')
    await wrapper.get('[data-testid="usage-sort-cost"]').trigger('click')

    expect(wrapper.emitted('sort-change')).toEqual([
      ['platform', 'asc'],
      ['operation', 'asc'],
      ['operation', 'desc'],
      ['account', 'asc'],
      ['status', 'asc'],
      ['cost', 'asc'],
    ])
  })

  it('renders error and filtered empty states through reusable table states', async () => {
    const error = mount(UsageRecordsTable, {
      props: {
        ...baseProps,
        rows: [],
        totalRows: 0,
        loadError: true,
      },
      global: {
        stubs: { Pagination: PaginationStub },
      },
    })

    expect(error.get('[data-testid="usage-error-state"]').text()).toContain('Failed to load usage records')
    await error.get('[data-testid="usage-retry-load"]').trigger('click')
    expect(error.emitted('retry')).toHaveLength(1)

    const filteredEmpty = mount(UsageRecordsTable, {
      props: {
        ...baseProps,
        rows: [],
        totalRows: 0,
        hasActiveFilters: true,
      },
      global: {
        stubs: { Pagination: PaginationStub },
      },
    })

    expect(filteredEmpty.get('[data-testid="usage-empty-state"]').text()).toContain('No usage records match the current filters.')
    await filteredEmpty.get('[data-testid="usage-empty-clear-filters"]').trigger('click')
    expect(filteredEmpty.emitted('clear')).toHaveLength(1)
  })
})
