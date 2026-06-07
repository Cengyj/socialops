import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'SocialOps',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('admin payment maintenance routes', () => {
  it('keeps admin plan and payment maintenance pages reachable when user payment is disabled', async () => {
    const { default: router } = await import('@/router')
    const routeNames = ['AdminPaymentDashboard', 'AdminOrders', 'AdminPaymentPlans']

    for (const routeName of routeNames) {
      const route = router.getRoutes().find((record) => record.name === routeName)
      expect(route?.meta.requiresAdmin).toBe(true)
      expect(route?.meta.requiresPayment).not.toBe(true)
    }
  })

  it('redirects legacy backup links into the settings backup tab', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.path === '/admin/backups')

    expect(route?.path).toBe('/admin/backups')
    expect(route?.redirect).toEqual({ path: '/admin/settings', query: { tab: 'backup' } })
    expect(route?.name).toBeUndefined()
  })

  it('redirects legacy data management links into the settings backup tab', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.path === '/admin/data-management')

    expect(route?.path).toBe('/admin/data-management')
    expect(route?.redirect).toEqual({ path: '/admin/settings', query: { tab: 'backup' } })
    expect(route?.name).toBeUndefined()
  })
})
