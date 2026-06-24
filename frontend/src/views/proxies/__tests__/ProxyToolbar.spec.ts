import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ProxyToolbar from '../components/ProxyToolbar.vue'
import type { SelectOption } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.processing': 'Processing...',
    'common.refresh': 'Refresh',
    'proxies.addProxy': 'Add proxy',
    'proxies.noProxiesToTest': 'No proxies to test.',
    'proxies.searchPlaceholder': 'Search proxies',
    'proxies.selection.noneSelected': 'Select at least one proxy to test.',
    'proxies.selection.selectedCount': '{count} selected',
    'proxies.testAll': 'Test all',
    'proxies.testSelected': 'Test selected',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const message = messages[key] ?? key
        return message.replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`))
      },
    }),
  }
})

const statusOptions: SelectOption[] = [
  { value: 'all', label: 'All statuses' },
  { value: 'online', label: 'Online' },
]

const typeOptions: SelectOption[] = [
  { value: 'all', label: 'All types' },
  { value: 'residential', label: 'Residential' },
]

type ToolbarPropOverrides = Partial<InstanceType<typeof ProxyToolbar>['$props']>

function mountToolbar(testing: boolean, loading = false, overrides: ToolbarPropOverrides = {}) {
  return mount(ProxyToolbar, {
    props: {
      hasProxies: true,
      loading,
      searchQuery: '',
      selectedCount: 1,
      statusFilter: 'all',
      statusOptions,
      testing,
      typeFilter: 'all',
      typeOptions,
      ...overrides,
    },
    global: {
      stubs: {
        Icon: true,
        SearchInput: {
          props: ['modelValue', 'placeholder'],
          template: '<input :placeholder="placeholder" :value="modelValue" />',
        },
        Select: {
          props: ['modelValue', 'options'],
          template: '<select :value="modelValue"><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>',
        },
      },
    },
  })
}

function buttonByText(wrapper: ReturnType<typeof mountToolbar>, text: string) {
  const button = wrapper.findAll('button').find(item => item.text() === text)
  expect(button, `${text} button`).toBeTruthy()
  return button!
}

describe('ProxyToolbar', () => {
  it('shows the current proxy selection count without adding another action', async () => {
    const wrapper = mountToolbar(false)
    const selectedCount = wrapper.get('[data-testid="proxy-selected-count"]')

    expect(selectedCount.text()).toBe('1 selected')
    expect(selectedCount.attributes('title')).toBe('1 selected')
    expect(selectedCount.attributes('role')).toBe('status')
    expect(selectedCount.attributes('aria-live')).toBe('polite')
    expect(selectedCount.attributes('aria-atomic')).toBe('true')
    expect(wrapper.findAll('button')).toHaveLength(4)

    await wrapper.setProps({ selectedCount: 3 })

    expect(wrapper.get('[data-testid="proxy-selected-count"]').text()).toBe('3 selected')
  })

  it('keeps toolbar action labels inspectable and constrained on narrow layouts', () => {
    const wrapper = mountToolbar(false)

    for (const label of ['Refresh', 'Test selected', 'Test all', 'Add proxy']) {
      const button = buttonByText(wrapper, label)
      expect(button.attributes('aria-label')).toBe(label)
      expect(button.attributes('title')).toBe(label)
      expect(button.classes()).toEqual(expect.arrayContaining(['h-10', 'min-w-0', 'max-w-full', 'justify-center']))
      const text = button.get('span')
      expect(text.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    }
  })

  it('locks mutating and test actions while connectivity checks are running', async () => {
    const wrapper = mountToolbar(true)

    for (const label of ['Test selected', 'Test all', 'Add proxy']) {
      const button = buttonByText(wrapper, label)
      expect(button.attributes('disabled')).toBeDefined()
      expect(button.attributes('aria-label')).toBe('Processing...')
      expect(button.attributes('title')).toBe('Processing...')
      await button.trigger('click')
    }

    expect(wrapper.emitted('testSelected')).toBeUndefined()
    expect(wrapper.emitted('testAll')).toBeUndefined()
    expect(wrapper.emitted('create')).toBeUndefined()
  })

  it('keeps create and test actions available when checks are idle', async () => {
    const wrapper = mountToolbar(false)

    for (const label of ['Test selected', 'Test all', 'Add proxy']) {
      const button = buttonByText(wrapper, label)
      expect(button.attributes('disabled')).toBeUndefined()
      await button.trigger('click')
    }

    expect(wrapper.emitted('testSelected')).toHaveLength(1)
    expect(wrapper.emitted('testAll')).toHaveLength(1)
    expect(wrapper.emitted('create')).toHaveLength(1)
  })

  it('locks test actions while the proxy list is refreshing but keeps create available', async () => {
    const wrapper = mountToolbar(false, true)

    const refreshButton = buttonByText(wrapper, 'Refresh')
    expect(refreshButton.attributes('disabled')).toBeDefined()
    expect(refreshButton.attributes('aria-label')).toBe('Processing...')
    expect(refreshButton.attributes('title')).toBe('Processing...')

    for (const label of ['Test selected', 'Test all']) {
      const button = buttonByText(wrapper, label)
      expect(button.attributes('disabled')).toBeDefined()
      expect(button.attributes('aria-label')).toBe('Processing...')
      expect(button.attributes('title')).toBe('Processing...')
      await button.trigger('click')
    }

    const createButton = buttonByText(wrapper, 'Add proxy')
    expect(createButton.attributes('disabled')).toBeUndefined()
    expect(createButton.attributes('title')).toBe('Add proxy')
    await createButton.trigger('click')

    expect(wrapper.emitted('testSelected')).toBeUndefined()
    expect(wrapper.emitted('testAll')).toBeUndefined()
    expect(wrapper.emitted('create')).toHaveLength(1)
  })

  it('explains disabled connectivity actions when selection or proxy data is missing', async () => {
    const wrapper = mountToolbar(false, false, {
      hasProxies: false,
      selectedCount: 0,
    })

    const selectedButton = buttonByText(wrapper, 'Test selected')
    expect(selectedButton.attributes('disabled')).toBeDefined()
    expect(selectedButton.attributes('aria-label')).toBe('Select at least one proxy to test.')
    expect(selectedButton.attributes('title')).toBe('Select at least one proxy to test.')
    await selectedButton.trigger('click')

    const testAllButton = buttonByText(wrapper, 'Test all')
    expect(testAllButton.attributes('disabled')).toBeDefined()
    expect(testAllButton.attributes('aria-label')).toBe('No proxies to test.')
    expect(testAllButton.attributes('title')).toBe('No proxies to test.')
    await testAllButton.trigger('click')

    expect(wrapper.emitted('testSelected')).toBeUndefined()
    expect(wrapper.emitted('testAll')).toBeUndefined()
  })
})
