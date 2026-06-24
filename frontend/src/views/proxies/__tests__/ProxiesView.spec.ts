import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ProxiesView from '../ProxiesView.vue'

const { listProxies, testAllProxies } = vi.hoisted(() => ({
  listProxies: vi.fn(),
  testAllProxies: vi.fn(),
}))

vi.mock('@/api/proxies', () => ({
  default: {
    list: listProxies,
    listUsable: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    test: vi.fn(),
    testAll: testAllProxies,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.actions': 'Actions',
    'common.cancel': 'Cancel',
    'common.confirm': 'Confirm',
    'common.refresh': 'Refresh',
    'common.retry': 'Retry',
    'common.edit': 'Edit',
    'common.delete': 'Delete',
    'common.clear': 'Clear',
    'common.saving': 'Saving',
    'common.processing': 'Processing',
    'proxies.failedToLoad': 'Failed to load proxies',
    'proxies.addProxy': 'Add proxy',
    'proxies.editTitle': 'Edit proxy',
    'proxies.searchPlaceholder': 'Search proxies',
    'proxies.testSelected': 'Test selected',
    'proxies.testAll': 'Test all',
    'proxies.test': 'Test',
    'proxies.empty.title': 'No proxies yet',
    'proxies.empty.description': 'Add and test your own proxy resources.',
    'proxies.noResults.title': 'No matching proxies',
    'proxies.noResults.description': 'Adjust filters and try again.',
    'proxies.stats.total': 'Total',
    'proxies.stats.online': 'Online',
    'proxies.stats.offline': 'Offline',
    'proxies.stats.unknown': 'Unknown',
    'proxies.filters.allStatus': 'All statuses',
    'proxies.filters.allTypes': 'All types',
    'proxies.status.online': 'Online',
    'proxies.status.offline': 'Offline',
    'proxies.status.unknown': 'Unknown',
    'proxies.types.residential': 'Residential',
    'proxies.types.static': 'Static',
    'proxies.types.dynamic': 'Dynamic',
    'proxies.types.mobile': 'Mobile',
    'proxies.types.datacenter': 'Datacenter',
    'proxies.columns.name': 'Name',
    'proxies.columns.type': 'Type',
    'proxies.columns.endpoint': 'Endpoint',
    'proxies.columns.status': 'Status',
    'proxies.columns.latency': 'Latency',
    'proxies.columns.lastCheck': 'Last check',
    'proxies.form.name': 'Name',
    'proxies.form.namePlaceholder': 'Proxy name',
    'proxies.form.type': 'Type',
    'proxies.form.endpoint': 'Endpoint',
    'proxies.form.endpointPlaceholder': 'Proxy endpoint',
    'proxies.form.endpointHint': 'Only online proxies with valid endpoints can be assigned.',
    'proxies.form.remark': 'Remark',
    'proxies.form.remarkPlaceholder': 'Optional remark',
    'proxies.deleteDialog.title': 'Delete proxy',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: string) => messages[key] ?? fallback ?? key,
    }),
  }
})

function mountView() {
  return mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: { template: '<section><slot name="filters" /><slot name="table" /></section>' },
        DataTable: {
          props: ['data', 'loading'],
          template: '<div><slot v-if="data.length === 0 && !loading" name="empty" /></div>',
        },
        Icon: true,
        SearchInput: { props: ['modelValue', 'placeholder'], template: '<input :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
        Select: { props: ['modelValue', 'options'], template: '<select :value="modelValue"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>' },
      },
    },
  })
}

function mountViewWithRealDialog() {
  return mount(ProxiesView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: { template: '<section><slot name="filters" /><slot name="table" /></section>' },
        DataTable: {
          props: ['data', 'loading'],
          template: '<div><slot v-if="data.length === 0 && !loading" name="empty" /></div>',
        },
        Icon: true,
        SearchInput: { props: ['modelValue', 'placeholder'], template: '<input :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
        Select: { props: ['modelValue', 'options'], template: '<select :value="modelValue"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>' },
      },
    },
  })
}

function installDesktopViewport() {
  const mediaQuery = {
    matches: true,
    media: '(min-width: 768px)',
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }
  vi.stubGlobal('matchMedia', vi.fn().mockReturnValue(mediaQuery))
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: globalThis.matchMedia,
  })
}

function mountViewWithRealPageChrome() {
  installDesktopViewport()
  return mount(ProxiesView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
      },
    },
  })
}

describe('ProxiesView', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.unstubAllGlobals()
    listProxies.mockReset()
    testAllProxies.mockReset()
    listProxies.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    testAllProxies.mockResolvedValue([])
  })

  it('keeps proxy load errors readable in the existing retry panel', async () => {
    listProxies.mockRejectedValue({})

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load proxies')
    expect(wrapper.text().match(/Failed to load proxies/g)).toHaveLength(1)
    expect(wrapper.find('p[title="Failed to load proxies"]').exists()).toBe(false)
    const retryButton = wrapper.get('button[aria-label="Retry"]')
    expect(retryButton.attributes('title')).toBe('Retry')
    expect(retryButton.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'shrink-0', 'justify-center']))
    expect(retryButton.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))

    listProxies.mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    await retryButton.trigger('click')
    await flushPromises()

    expect(listProxies).toHaveBeenCalledTimes(2)
    expect(wrapper.find('p[title="Failed to load proxies"]').exists()).toBe(false)
  })

  it('opens the create proxy dialog from the toolbar action', async () => {
    const wrapper = mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button').filter(button => button.text().includes('Add proxy'))
    expect(buttons.length).toBeGreaterThan(0)

    await buttons[0].trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.textContent).toContain('Add proxy')
    expect(document.body.querySelector('#proxy-name')).not.toBeNull()
  })

  it('opens the attached create proxy dialog from the toolbar action', async () => {
    const wrapper = mountViewWithRealDialog()
    await flushPromises()

    const buttons = wrapper.findAll('button').filter(button => button.text().includes('Add proxy'))
    expect(buttons.length).toBeGreaterThan(0)

    await buttons[0].trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.textContent).toContain('Add proxy')
    expect(document.body.querySelector('#proxy-name')).not.toBeNull()
  })

  it('opens the create proxy dialog with the real page layout and table', async () => {
    const wrapper = mountViewWithRealPageChrome()
    await flushPromises()

    const buttons = wrapper.findAll('button').filter(button => button.text().includes('Add proxy'))
    expect(buttons.length).toBeGreaterThan(0)

    await buttons[0].trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.textContent).toContain('Add proxy')
    expect(document.body.querySelector('#proxy-name')).not.toBeNull()
  })

  it('keeps the test-all action available when filters currently match no rows', async () => {
    listProxies
      .mockResolvedValueOnce({
        items: [
          {
            id: 7,
            user_id: 10,
            name: 'Tokyo proxy',
            ip_type: 'residential',
            endpoint: 'http://proxy.example.com:8080',
            status: 'unknown',
            latency_ms: null,
            last_check_at: null,
            remark: null,
            created_at: '2026-06-06T00:00:00Z',
            updated_at: '2026-06-06T01:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 200,
        pages: 1,
      })

    const wrapper = mountView()
    await flushPromises()

    const searchInput = wrapper.find('input[placeholder="Search proxies"]')
    await searchInput.setValue('no-match')
    await new Promise(resolve => setTimeout(resolve, 300))
    await flushPromises()

    const testSelectedButton = wrapper.findAll('button').find(button => button.text().includes('Test selected'))
    const testAllButton = wrapper.findAll('button').find(button => button.text().includes('Test all'))
    expect(testSelectedButton?.attributes('disabled')).toBeDefined()
    expect(testAllButton?.attributes('disabled')).toBeUndefined()

    await testAllButton?.trigger('click')
    await flushPromises()

    expect(testAllProxies).toHaveBeenCalledTimes(1)
  })
})
