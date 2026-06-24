import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageFiltersToolbar from '../UsageFiltersToolbar.vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

const messages: Record<string, string> = {
  'common.refresh': 'Refresh',
  'usage.platform': 'Platform',
  'usage.operation': 'Operation',
  'usage.result': 'Result',
  'usage.timeRange': 'Time Range',
  'usage.filters.clear': 'Clear filters',
  'usage.exportCsv': 'Export CSV',
  'usage.exporting': 'Exporting...',
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

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<button type="button" class="select-stub" @click="$emit(\'update:modelValue\', options[1]?.value)">{{ options[0]?.label }}</button>',
}

const DateRangePickerStub = {
  name: 'DateRangePicker',
  props: ['startDate', 'endDate'],
  emits: ['update:startDate', 'update:endDate', 'change'],
  template: '<button type="button" data-testid="usage-date-range" @click="$emit(\'change\', { startDate, endDate, preset: null })">Date range</button>',
}

function mountToolbar(overrides = {}) {
  return mount(UsageFiltersToolbar, {
    props: {
      platform: 'all',
      operation: 'all',
      status: 'all',
      startDate: '2026-06-01',
      endDate: '2026-06-09',
      platformOptions: [
        { value: 'all', label: 'All platforms' },
        { value: 'x_twitter', label: 'Twitter / X' },
      ],
      operationOptions: [
        { value: 'all', label: 'All operations' },
        { value: 'post', label: 'Post' },
      ],
      statusOptions: [
        { value: 'all', label: 'All statuses' },
        { value: 'failed', label: 'Failed' },
      ],
      hasActiveFilters: false,
      loading: false,
      exporting: false,
      ...overrides,
    },
    global: {
      stubs: {
        Select: SelectStub,
        DateRangePicker: DateRangePickerStub,
      },
    },
  })
}

describe('UsageFiltersToolbar', () => {
  it('keeps the time range label defined in real usage locales', () => {
    expect(en.usage.timeRange).toBe('Time Range')
    expect(zh.usage.timeRange).toBe('时间范围')
  })

  it('orders filters as platform, operation, status, then time and keeps actions together', () => {
    const wrapper = mountToolbar()
    const selects = wrapper.findAllComponents({ name: 'Select' })

    expect(selects).toHaveLength(3)
    expect(wrapper.text()).toContain('Platform')
    expect(wrapper.text()).toContain('Operation')
    expect(wrapper.text()).toContain('Result')
    expect(wrapper.text()).toContain('Time Range')
    expect((selects[0].props('options') as Array<{ label: string }>)[0].label).toBe('All platforms')
    expect((selects[1].props('options') as Array<{ label: string }>)[0].label).toBe('All operations')
    expect((selects[2].props('options') as Array<{ label: string }>)[0].label).toBe('All statuses')

    const toolbar = wrapper.get('[data-testid="usage-filter-toolbar"]')
    expect(toolbar.html().indexOf('All platforms')).toBeLessThan(toolbar.html().indexOf('All operations'))
    expect(toolbar.html().indexOf('All operations')).toBeLessThan(toolbar.html().indexOf('All statuses'))
    expect(toolbar.html().indexOf('All statuses')).toBeLessThan(toolbar.html().indexOf('usage-date-range'))

    const actions = wrapper.get('[data-testid="usage-filter-actions"]')
    expect(actions.html().indexOf('Refresh')).toBeLessThan(actions.html().indexOf('Clear filters'))
    expect(actions.html().indexOf('Clear filters')).toBeLessThan(actions.html().indexOf('Export CSV'))
  })

  it('emits filter, date, and action events from the toolbar controls', async () => {
    const wrapper = mountToolbar({ hasActiveFilters: true })
    const selects = wrapper.findAllComponents({ name: 'Select' })

    await selects[0].trigger('click')
    await selects[1].trigger('click')
    await selects[2].trigger('click')
    await wrapper.get('[data-testid="usage-date-range"]').trigger('click')
    await wrapper.get('[data-testid="usage-refresh"]').trigger('click')
    await wrapper.get('[data-testid="usage-clear-filters"]').trigger('click')
    await wrapper.get('[data-testid="usage-export-csv"]').trigger('click')

    expect(wrapper.emitted('update:platform')?.[0]).toEqual(['x_twitter'])
    expect(wrapper.emitted('update:operation')?.[0]).toEqual(['post'])
    expect(wrapper.emitted('update:status')?.[0]).toEqual(['failed'])
    expect(wrapper.emitted('date-change')).toHaveLength(1)
    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(wrapper.emitted('clear')).toHaveLength(1)
    expect(wrapper.emitted('export-csv')).toHaveLength(1)
  })

  it('disables actions while loading and shows exporting text while exporting', () => {
    const loading = mountToolbar({ loading: true, hasActiveFilters: true })

    expect((loading.get('[data-testid="usage-refresh"]').element as HTMLButtonElement).disabled).toBe(true)
    expect((loading.get('[data-testid="usage-clear-filters"]').element as HTMLButtonElement).disabled).toBe(true)
    expect((loading.get('[data-testid="usage-export-csv"]').element as HTMLButtonElement).disabled).toBe(true)

    const exporting = mountToolbar({ exporting: true })
    expect(exporting.get('[data-testid="usage-export-csv"]').text()).toContain('Exporting...')
  })
})
