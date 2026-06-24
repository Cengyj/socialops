import { mount, type DOMWrapper } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import ProxyTable from '../components/ProxyTable.vue'
import type { ProxyRow } from '../useProxyManagement'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.actions': 'Actions',
    'common.delete': 'Delete',
    'common.edit': 'Edit',
    'common.processing': 'Processing',
    'proxies.addProxy': 'Add proxy',
    'proxies.columns.endpoint': 'Endpoint',
    'proxies.columns.lastCheck': 'Last check',
    'proxies.columns.latency': 'Latency',
    'proxies.columns.name': 'Name',
    'proxies.columns.status': 'Status',
    'proxies.columns.type': 'Type',
    'proxies.empty.description': 'Add a proxy.',
    'proxies.empty.title': 'No proxies',
    'proxies.noResults.description': 'Adjust filters.',
    'proxies.noResults.title': 'No results',
    'proxies.test': 'Test',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
  },
  methods: {
    rowId(row: unknown) {
      return (row as { id?: number } | null)?.id ?? ''
    },
    rowValue(row: unknown, key: string) {
      return (row as Record<string, unknown> | null)?.[key]
    },
  },
  template: `
    <div data-testid="proxy-table-stub">
      <div data-testid="proxy-header">
        <slot name="header-select" />
      </div>
      <div v-for="row in data" :key="rowId(row)" :data-testid="\`proxy-row-\${rowId(row)}\`">
        <slot name="cell-select" :row="row" />
        <slot name="cell-name" :row="row" />
        <slot name="cell-type" :row="row" :value="rowValue(row, 'type')" />
        <slot name="cell-endpoint" :row="row" :value="rowValue(row, 'endpoint')" />
        <slot name="cell-status" :row="row" :value="rowValue(row, 'status')" />
        <slot name="cell-latency" :row="row" :value="rowValue(row, 'latency')" />
        <slot name="cell-actions" :row="row" />
      </div>
      <slot v-if="!loading && data.length === 0" name="empty" />
    </div>
  `,
})

const row: ProxyRow = {
  id: 7,
  name: 'Tokyo proxy',
  type: 'residential',
  endpoint: 'http://proxy.example.com:8080',
  status: 'online',
  latency: 42,
  lastCheck: '2026-06-06 10:00',
  remark: 'primary',
  updatedAt: '2026-06-06T01:00:00Z',
}

function mountTable(testing: boolean, proxies: ProxyRow[] = [row], loading = false, hasActiveProxyFilters = false) {
  return mount(ProxyTable, {
    props: {
      allVisibleSelected: false,
      hasActiveProxyFilters,
      isSelected: () => false,
      loading,
      proxies,
      proxyStatusLabel: (status: string) => status,
      proxyTypeLabel: (type: string) => type,
      someVisibleSelected: false,
      statusBadgeClass: () => 'badge-success',
      testing,
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        Icon: true,
      },
    },
  })
}

function expectConstrainedActionButton(button: DOMWrapper<Element>, label: string) {
  expect(button.attributes('aria-label')).toBe(label)
  expect(button.attributes('title')).toBe(label)
  expect(button.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
  const text = button.findAll('span').find(item => item.text() === label)
  expect(text, `${label} button text`).toBeTruthy()
  expect(text!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
}

describe('ProxyTable', () => {
  it('locks selection controls and empty create action while proxy tests are running', async () => {
    const wrapper = mountTable(true)

    const headerCheckbox = wrapper.get('[data-testid="proxy-header"] input[type="checkbox"]')
    const rowCheckbox = wrapper.get('[data-testid="proxy-row-7"] input[type="checkbox"]')
    expect(headerCheckbox.attributes('disabled')).toBeDefined()
    expect(rowCheckbox.attributes('disabled')).toBeDefined()
    await headerCheckbox.trigger('change')
    await rowCheckbox.trigger('change')

    expect(wrapper.emitted('toggleAllVisible')).toBeUndefined()
    expect(wrapper.emitted('toggleSelection')).toBeUndefined()

    const emptyWrapper = mountTable(true, [])
    const createButton = emptyWrapper.findAll('button').find(item => item.text() === 'Add proxy')
    expect(createButton).toBeTruthy()
    expect(createButton!.attributes('aria-label')).toBe('Processing')
    expect(createButton!.attributes('title')).toBe('Processing')
    expect(createButton!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    const createButtonText = createButton!.findAll('span').find(item => item.text() === 'Add proxy')
    expect(createButtonText).toBeTruthy()
    expect(createButtonText!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(createButton!.attributes('disabled')).toBeDefined()
    await createButton!.trigger('click')
    expect(emptyWrapper.emitted('create')).toBeUndefined()
  })

  it('keeps selection controls and empty create action available when proxy tests are idle', async () => {
    const wrapper = mountTable(false)

    const headerCheckbox = wrapper.get('[data-testid="proxy-header"] input[type="checkbox"]')
    const rowCheckbox = wrapper.get('[data-testid="proxy-row-7"] input[type="checkbox"]')
    expect(headerCheckbox.attributes('disabled')).toBeUndefined()
    expect(rowCheckbox.attributes('disabled')).toBeUndefined()
    await headerCheckbox.trigger('change')
    await rowCheckbox.trigger('change')

    expect(wrapper.emitted('toggleAllVisible')).toHaveLength(1)
    expect(wrapper.emitted('toggleSelection')).toEqual([[7]])

    const emptyWrapper = mountTable(false, [])
    const createButton = emptyWrapper.findAll('button').find(item => item.text() === 'Add proxy')
    expect(createButton).toBeTruthy()
    expectConstrainedActionButton(createButton!, 'Add proxy')
    expect(createButton!.attributes('disabled')).toBeUndefined()
    await createButton!.trigger('click')
    expect(emptyWrapper.emitted('create')).toHaveLength(1)
  })

  it('shows the filtered empty state without the create action when filters match no proxies', async () => {
    const wrapper = mountTable(false, [], false, true)

    expect(wrapper.text()).toContain('No results')
    expect(wrapper.text()).toContain('Adjust filters.')
    expect(wrapper.text()).not.toContain('No proxies')
    expect(wrapper.text()).not.toContain('Add a proxy.')
    expect(wrapper.findAll('button').some(button => button.text() === 'Add proxy')).toBe(false)
  })

  it('keeps long proxy names from widening mobile card rows', () => {
    const longName = 'stage104-mobile-proxy-table-name-with-a-very-long-unbroken-identifier-0123456789abcdef'
    const wrapper = mountTable(false, [{ ...row, name: longName }])
    const nameButton = wrapper.get('[data-testid="proxy-row-7"] button')

    expect(nameButton.classes()).toEqual(expect.arrayContaining(['max-w-full', 'break-all', 'sm:break-normal']))
    expect(nameButton.attributes('title')).toBe(longName)
  })

  it('keeps long proxy endpoints readable in mobile card rows and truncated on desktop', () => {
    const longEndpoint = 'http://stage122-proxy-endpoint-with-a-really-long-unbroken-hostname-0123456789abcdef.example.test:18080'
    const wrapper = mountTable(false, [{ ...row, endpoint: longEndpoint }])
    const endpoint = wrapper.get(`span[title="${longEndpoint}"]`)

    expect(endpoint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-all', 'text-right', 'sm:max-w-[360px]', 'sm:truncate', 'sm:break-normal']))
    expect(endpoint.text()).toBe(longEndpoint)
  })

  it('keeps row-level action labels inspectable and constrained on narrow layouts', () => {
    const wrapper = mountTable(false)
    const rowEl = wrapper.get('[data-testid="proxy-row-7"]')

    for (const label of ['Test', 'Edit', 'Delete']) {
      const button = rowEl.get(`button[aria-label="${label}"]`)
      expectConstrainedActionButton(button, label)
      expect(button.attributes('disabled')).toBeUndefined()
    }
  })

  it('disables every row-level action while proxy tests are running', async () => {
    const wrapper = mountTable(true)
    const rowEl = wrapper.get('[data-testid="proxy-row-7"]')

    for (const label of ['Tokyo proxy', 'Test', 'Edit', 'Delete']) {
      const button = rowEl.findAll('button').find(item => item.text() === label)
      expect(button, `${label} button`).toBeTruthy()
      expect(button!.attributes('disabled')).toBeDefined()
      if (label !== 'Tokyo proxy') {
        expect(button!.attributes('aria-label')).toBe('Processing')
        expect(button!.attributes('title')).toBe('Processing')
      }
      await button!.trigger('click')
    }

    expect(wrapper.emitted('edit')).toBeUndefined()
    expect(wrapper.emitted('test')).toBeUndefined()
    expect(wrapper.emitted('delete')).toBeUndefined()
  })

  it('emits row actions normally when proxy tests are idle', async () => {
    const wrapper = mountTable(false)
    const rowEl = wrapper.get('[data-testid="proxy-row-7"]')

    for (const label of ['Tokyo proxy', 'Test', 'Edit', 'Delete']) {
      const button = rowEl.findAll('button').find(item => item.text() === label)
      expect(button, `${label} button`).toBeTruthy()
      expect(button!.attributes('disabled')).toBeUndefined()
      await button!.trigger('click')
    }

    expect(wrapper.emitted('edit')).toEqual([[row], [row]])
    expect(wrapper.emitted('test')).toEqual([[7]])
    expect(wrapper.emitted('delete')).toEqual([[row]])
  })

  it('locks selection and row actions while the proxy list is refreshing', async () => {
    const wrapper = mountTable(false, [row], true)
    const rowEl = wrapper.get('[data-testid="proxy-row-7"]')

    const headerCheckbox = wrapper.get('[data-testid="proxy-header"] input[type="checkbox"]')
    const rowCheckbox = rowEl.get('input[type="checkbox"]')
    expect(headerCheckbox.attributes('disabled')).toBeDefined()
    expect(rowCheckbox.attributes('disabled')).toBeDefined()
    await headerCheckbox.trigger('change')
    await rowCheckbox.trigger('change')

    for (const label of ['Tokyo proxy', 'Test', 'Edit', 'Delete']) {
      const button = rowEl.findAll('button').find(item => item.text() === label)
      expect(button, `${label} button`).toBeTruthy()
      expect(button!.attributes('disabled')).toBeDefined()
      if (label !== 'Tokyo proxy') {
        expect(button!.attributes('aria-label')).toBe('Processing')
        expect(button!.attributes('title')).toBe('Processing')
      }
      await button!.trigger('click')
    }

    expect(wrapper.emitted('toggleAllVisible')).toBeUndefined()
    expect(wrapper.emitted('toggleSelection')).toBeUndefined()
    expect(wrapper.emitted('edit')).toBeUndefined()
    expect(wrapper.emitted('test')).toBeUndefined()
    expect(wrapper.emitted('delete')).toBeUndefined()
  })
})
