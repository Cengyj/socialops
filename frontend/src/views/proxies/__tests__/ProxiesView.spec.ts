import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ProxiesView from '../ProxiesView.vue'

const { listProxies } = vi.hoisted(() => ({
  listProxies: vi.fn(),
}))

vi.mock('@/api/proxies', () => ({
  default: {
    list: listProxies,
    listUsable: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    test: vi.fn(),
    testAll: vi.fn(),
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
    'common.edit': 'Edit',
    'common.delete': 'Delete',
    'common.clear': 'Clear',
    'common.saving': 'Saving',
    'common.processing': 'Processing',
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
        SearchInput: { props: ['modelValue', 'placeholder'], template: '<input :placeholder="placeholder" :value="modelValue" />' },
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
        SearchInput: { props: ['modelValue', 'placeholder'], template: '<input :placeholder="placeholder" :value="modelValue" />' },
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
    listProxies.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
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
})
