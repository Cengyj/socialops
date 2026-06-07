import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import CustomPageView from '../CustomPageView.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { apiClient } from '@/api/client'

const { getSettings, getPaymentConfig } = vi.hoisted(() => ({
  getSettings: vi.fn(),
  getPaymentConfig: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings,
    },
    payment: {
      getConfig: getPaymentConfig,
    },
  },
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
  },
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({
      params: { id: 'admin-docs' },
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'en-US' },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }

function publicSettings() {
  return {
    registration_enabled: false,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: false,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    totp_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'SocialOps',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    payment_enabled: false,
    risk_control_enabled: false,
    purchase_subscription_enabled: false,
    purchase_subscription_url: '',
    table_default_page_size: 20,
    table_page_size_options: [10, 20, 50, 100],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    dingtalk_oauth_enabled: false,
    wechat_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: '',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    balance_low_notify_recharge_url: '',
    affiliate_enabled: false,
  }
}

describe('CustomPageView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getSettings.mockReset()
    getPaymentConfig.mockReset()
    getPaymentConfig.mockResolvedValue({ data: { enabled: true } })
    vi.mocked(apiClient.get).mockReset()
  })

  it('loads admin-only custom menu settings before resolving a direct custom page route', async () => {
    getSettings.mockResolvedValue({
      custom_menu_items: [
        {
          id: 'admin-docs',
          label: 'Admin Docs',
          icon_svg: '',
          url: 'https://docs.example.com/admin',
          visibility: 'admin',
          sort_order: 0,
        },
      ],
    })

    const appStore = useAppStore()
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = publicSettings() as any

    const authStore = useAuthStore()
    authStore.user = {
      id: 7,
      email: 'admin@example.com',
      username: 'admin',
      role: 'admin',
      balance: 0,
      concurrency: 1,
      status: 'active',
      created_at: '2026-06-04T00:00:00Z',
      updated_at: '2026-06-04T00:00:00Z',
    } as any
    authStore.token = 'admin-token'

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Icon: true,
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(getSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).not.toContain('customPage.notFoundTitle')

    const iframe = wrapper.get('iframe')
    const iframeURL = new URL(iframe.attributes('src'))
    expect(iframeURL.origin).toBe('https://docs.example.com')
    expect(iframeURL.pathname).toBe('/admin')
    expect(iframeURL.searchParams.get('user_id')).toBe('7')
    expect(iframeURL.searchParams.get('token')).toBe('admin-token')
    expect(iframeURL.searchParams.get('ui_mode')).toBe('embedded')
  })

  it('loads markdown pages through the shared API client so auth refresh remains available', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: '# Admin Guide\n\n![logo](images/logo.png)',
    } as any)

    const appStore = useAppStore()
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = {
      ...publicSettings(),
      custom_menu_items: [
        {
          id: 'admin-docs',
          label: 'Admin Docs',
          icon_svg: '',
          url: 'md:help/intro',
          page_slug: 'help/intro',
          visibility: 'user',
          sort_order: 0,
        },
      ],
    } as any

    const authStore = useAuthStore()
    authStore.token = 'stale-token'

    mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Icon: true,
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(apiClient.get).toHaveBeenCalledWith('/pages/help%252Fintro', {
      responseType: 'text',
    })
  })

  it('shows the not-found markdown state when the page endpoint returns 404', async () => {
    vi.mocked(apiClient.get).mockRejectedValue({
      response: { status: 404 },
    })

    const appStore = useAppStore()
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = {
      ...publicSettings(),
      custom_menu_items: [
        {
          id: 'admin-docs',
          label: 'Admin Docs',
          icon_svg: '',
          url: 'md:missing',
          page_slug: 'missing',
          visibility: 'user',
          sort_order: 0,
        },
      ],
    } as any

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Icon: true,
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('customPage.markdownNotFoundTitle')
    expect(wrapper.text()).not.toContain('customPage.markdownLoadFailedTitle')
  })
})
