import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { resolveCompletedSetupRedirectPath } from '@/router/setupRedirect'

const routerSource = readFileSync(resolve(__dirname, '../index.ts'), 'utf8')

function sourceBetween(start: string, end: string): string {
  const startIndex = routerSource.indexOf(start)
  const endIndex = routerSource.indexOf(end)

  expect(startIndex).toBeGreaterThanOrEqual(0)
  expect(endIndex).toBeGreaterThan(startIndex)

  return routerSource.slice(startIndex, endIndex)
}

// Mock 导航加载状态
vi.mock('@/composables/useNavigationLoading', () => {
  const mockStart = vi.fn()
  const mockEnd = vi.fn()
  return {
    useNavigationLoadingState: () => ({
      startNavigation: mockStart,
      endNavigation: mockEnd,
      isLoading: { value: false },
    }),
    useNavigationLoading: () => ({
      startNavigation: mockStart,
      endNavigation: mockEnd,
      isLoading: { value: false },
    }),
  }
})

// Mock 路由预加载
vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

// Mock API 相关模块
vi.mock('@/api', () => ({
  authAPI: {
    getCurrentUser: vi.fn().mockResolvedValue({ data: {} }),
    logout: vi.fn(),
  },
  isTotp2FARequired: () => false,
}))

vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn(),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))


// 用于测试的 auth 状态
interface MockAuthState {
  isAuthenticated: boolean
  isAdmin: boolean
  isSimpleMode: boolean
  backendModeEnabled: boolean
  hasPendingAuthSession: boolean
  paymentEnabled?: boolean
  featureFlags?: Partial<Record<FeatureFlagName, boolean>>
  setupNeedsSetup?: boolean
}

type FeatureFlagName = 'payment' | 'affiliate'

const featureFlagModes: Record<FeatureFlagName, 'opt-in' | 'opt-out'> = {
  payment: 'opt-out',
  affiliate: 'opt-in',
}

function isMockFeatureFlagEnabled(flag: FeatureFlagName, authState: MockAuthState): boolean {
  const value = authState.featureFlags?.[flag]
  if (typeof value === 'boolean') return value
  return featureFlagModes[flag] === 'opt-out'
}

function matchesMockBackendModePublicPath(path: string, allowedPath: string): boolean {
  return path === allowedPath || path.startsWith(`${allowedPath}/`)
}

function matchesMockRoutePathPrefix(path: string, prefix: string): boolean {
  return path === prefix || path.startsWith(`${prefix}/`)
}

function isMockBackendModePublicRouteAllowed(path: string, hasPendingAuthSession: boolean): boolean {
  const allowed = [
    '/login',
    '/setup',
    '/payment/result',
    '/payment/stripe',
    '/payment/stripe-popup',
    '/payment/airwallex',
    '/legal',
  ]
  const callbackPaths = [
    '/auth/callback',
    '/auth/oauth/callback',
    '/auth/linuxdo/callback',
    '/auth/dingtalk/callback',
    '/auth/dingtalk/email-completion',
    '/auth/oidc/callback',
    '/auth/wechat/callback',
    '/auth/wechat/payment/callback',
  ]
  const pendingAuthPaths = ['/register', '/email-verify']

  return (
    allowed.some((allowedPath) => matchesMockBackendModePublicPath(path, allowedPath)) ||
    callbackPaths.includes(path) ||
    (hasPendingAuthSession && pendingAuthPaths.includes(path))
  )
}

/**
 * 将 router/index.ts 中 beforeEach 守卫的核心逻辑提取为可测试的函数
 */
function simulateGuard(
  toPath: string,
  toMeta: Record<string, any>,
  authState: MockAuthState
): string | null {
  const requiresAuth = toMeta.requiresAuth !== false
  const requiresAdmin = toMeta.requiresAdmin === true

  if (toPath === '/setup' && authState.setupNeedsSetup === false) {
    return resolveCompletedSetupRedirectPath(authState.isAuthenticated, authState.isAdmin)
  }

  // 不需要认证的路由
  if (!requiresAuth) {
    if (
      authState.isAuthenticated &&
      (toPath === '/login' || toPath === '/register')
    ) {
      if (authState.backendModeEnabled && !authState.isAdmin) {
        return toPath === '/login' ? null : '/login'
      }
      return authState.isAdmin ? '/admin/dashboard' : '/dashboard'
    }
    if (authState.backendModeEnabled && authState.isAuthenticated && !authState.isAdmin) {
      if (!isMockBackendModePublicRouteAllowed(toPath, authState.hasPendingAuthSession)) {
        return '/login'
      }
    }
    if (authState.backendModeEnabled && !authState.isAuthenticated) {
      if (!isMockBackendModePublicRouteAllowed(toPath, authState.hasPendingAuthSession)) {
        return '/login'
      }
    }
    return null // 允许通过
  }

  // 需要认证但未登录
  if (!authState.isAuthenticated) {
    return '/login'
  }

  // 需要管理员但不是管理员
  if (requiresAdmin && !authState.isAdmin) {
    return '/dashboard'
  }

  // 公开功能开关路由守卫应与 sidebar 的 feature flag registry 保持一致。
  if (toMeta.requiresFeatureFlag) {
    const featureFlag = toMeta.requiresFeatureFlag as FeatureFlagName
    if (!isMockFeatureFlagEnabled(featureFlag, authState)) {
      return authState.isAdmin ? '/admin/dashboard' : '/dashboard'
    }
  }

  // 支付功能开关采用 opt-out 语义：未加载 public settings 时不应提前拦截。
  if (toMeta.requiresPayment) {
    const paymentEnabled = authState.paymentEnabled
    if (paymentEnabled === false) {
      return authState.isAdmin ? '/admin/dashboard' : '/dashboard'
    }
  }

  // 简易模式限制
  if (authState.isSimpleMode) {
    const userRestrictedPaths = [
      '/accounts',
      '/usage',
      '/subscriptions',
      '/purchase',
      '/payment/qrcode',
      '/orders',
      '/redeem',
      '/affiliate',
    ]
    const adminRestrictedPaths = [
      '/usage',
      '/subscriptions',
      '/purchase',
      '/payment/qrcode',
      '/orders',
      '/redeem',
      '/affiliate',
      '/admin/subscriptions',
      '/admin/users',
      '/admin/redeem',
      '/admin/promo-codes',
      '/admin/affiliates',
      '/admin/orders',
    ]
    const restrictedPaths = authState.isAdmin ? adminRestrictedPaths : userRestrictedPaths
    if (restrictedPaths.some((path) => matchesMockRoutePathPrefix(toPath, path))) {
      return authState.isAdmin ? '/admin/dashboard' : '/dashboard'
    }
  }

  // Backend mode: admin gets full access, non-admin blocked
  if (authState.backendModeEnabled) {
    if (authState.isAuthenticated && authState.isAdmin) {
      return null
    }
    if (!isMockBackendModePublicRouteAllowed(toPath, authState.hasPendingAuthSession)) {
      return '/login'
    }
  }

  return null // 允许通过
}

describe('路由守卫逻辑', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  // --- 未认证用户 ---

  describe('未认证用户', () => {
    const authState: MockAuthState = {
      isAuthenticated: false,
      isAdmin: false,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问需要认证的页面重定向到 /login', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBe('/login')
    })

    it('访问管理页面重定向到 /login', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/login')
    })

    it('访问公开页面允许通过', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('访问 /home 公开页面允许通过', () => {
      const redirect = simulateGuard('/home', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })
  })

  // --- 已认证普通用户 ---

  describe('已认证普通用户', () => {
    const authState: MockAuthState = {
      isAuthenticated: true,
      isAdmin: false,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问 /login 重定向到 /dashboard', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /register 重定向到 /dashboard', () => {
      const redirect = simulateGuard('/register', { requiresAuth: false }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /dashboard 允许通过', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })

    it('访问管理页面被拒绝，重定向到 /dashboard', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('访问 /admin/users 被拒绝', () => {
      const redirect = simulateGuard('/admin/users', { requiresAdmin: true }, authState)
      expect(redirect).toBe('/dashboard')
    })
  })

  // --- 已认证管理员 ---

  describe('已认证管理员', () => {
    const authState: MockAuthState = {
      isAuthenticated: true,
      isAdmin: true,
      isSimpleMode: false,
      backendModeEnabled: false,
      hasPendingAuthSession: false,
    }

    it('访问 /login 重定向到 /admin/dashboard', () => {
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/dashboard')
    })

    it('访问管理页面允许通过', () => {
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBeNull()
    })

    it('访问用户页面允许通过', () => {
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })
  })

  // --- 简易模式 ---

  describe('公开功能开关路由守卫', () => {
    it('源码路由用已有 feature flag 标记 sidebar 已隐藏的功能页面', () => {
      expect(routerSource).toContain("requiresFeatureFlag: 'affiliate'")
    })

    it('源码守卫复用 feature flag registry，而不是复制开关默认值', () => {
      const featureGuard = sourceBetween(
        '// Check feature-flag requirements',
        '// Check payment requirement'
      )

      expect(featureGuard).toContain('requiresFeatureFlag')
      expect(featureGuard).toContain('isRequiredFeatureFlagEnabled')
      expect(routerSource).toContain('isFeatureFlagEnabled(FeatureFlags[flag])')
    })

    it('源码守卫在读取公开功能开关前先加载 public settings', () => {
      const guardInitialization = sourceBetween(
        'const appStore = useAppStore()',
        '// Check if route requires authentication'
      )

      expect(guardInitialization).toContain('await appStore.fetchPublicSettings()')
    })

    it('affiliate 开关未加载时按 opt-in 语义拦截用户直达页面', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }

      const redirect = simulateGuard('/affiliate', { requiresFeatureFlag: 'affiliate' }, authState)

      expect(redirect).toBe('/dashboard')
    })

    it('affiliate 明确关闭时拦截管理员直达记录页面', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
        featureFlags: { affiliate: false },
      }

      const redirect = simulateGuard(
        '/admin/affiliates/invites',
        { requiresAdmin: true, requiresFeatureFlag: 'affiliate' },
        authState
      )

      expect(redirect).toBe('/admin/dashboard')
    })

  })

  describe('支付路由守卫', () => {
    it('public settings 尚未加载时允许进入现有支付路由', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }

      const redirect = simulateGuard('/purchase', { requiresPayment: true }, authState)

      expect(redirect).toBeNull()
    })

    it('public settings 明确关闭支付时才拦截现有支付路由', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
        paymentEnabled: false,
      }

      const redirect = simulateGuard('/purchase', { requiresPayment: true }, authState)

      expect(redirect).toBe('/dashboard')
    })

    it('源码守卫使用 payment opt-out 语义，避免设置未注入时误跳转', () => {
      const paymentGuard = sourceBetween(
        '// Check payment requirement',
        '// 简易模式下限制访问某些页面'
      )

      expect(paymentGuard).toContain('paymentEnabled === false')
      expect(paymentGuard).not.toContain('if (!paymentEnabled)')
    })
  })

  describe('简易模式受限路由', () => {
    it('源码 simple mode 守卫覆盖侧边栏隐藏的用户和管理端路径', () => {
      const simpleModeGuard = sourceBetween(
        '// 简易模式下限制访问某些页面',
        '// Backend mode: admin gets full access, non-admin blocked'
      )

      expect(simpleModeGuard).toContain('SIMPLE_MODE_USER_RESTRICTED_PATHS')
      expect(simpleModeGuard).toContain('SIMPLE_MODE_ADMIN_RESTRICTED_PATHS')
      expect(simpleModeGuard).toContain('matchesRoutePathPrefix')

      const restrictedPaths = [
        "'/accounts'",
        "'/usage'",
        "'/subscriptions'",
        "'/purchase'",
        "'/payment/qrcode'",
        "'/orders'",
        "'/redeem'",
        "'/affiliate'",
        "'/admin/users'",
        "'/admin/subscriptions'",
        "'/admin/redeem'",
        "'/admin/promo-codes'",
        "'/admin/affiliates'",
        "'/admin/orders'",
      ]

      for (const path of restrictedPaths) {
        expect(routerSource).toContain(path)
      }
    })

    it('普通用户简易模式访问 /subscriptions 重定向到 /dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/subscriptions', {}, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('普通用户简易模式访问 /redeem 重定向到 /dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/redeem', {}, authState)
      expect(redirect).toBe('/dashboard')
    })

    it('普通用户简易模式访问侧边栏隐藏的工作台、用量、订单和返利页面会重定向', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }

      for (const path of ['/accounts', '/usage', '/purchase', '/payment/qrcode', '/orders', '/affiliate']) {
        expect(simulateGuard(path, {}, authState)).toBe('/dashboard')
      }
    })

    it('管理员简易模式访问 /admin/subscriptions 重定向', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard(
        '/admin/subscriptions',
        { requiresAdmin: true },
        authState
      )
      expect(redirect).toBe('/admin/dashboard')
    })

    it('管理员简易模式访问侧边栏隐藏的管理页面会重定向', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }

      const paths = [
        '/admin/users',
        '/admin/promo-codes',
        '/admin/affiliates/invites',
        '/admin/orders/dashboard',
        '/admin/orders/plans',
      ]

      for (const path of paths) {
        expect(simulateGuard(path, { requiresAdmin: true }, authState)).toBe('/admin/dashboard')
      }
    })

    it('简易模式下非受限页面正常访问', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBeNull()
    })

    it('管理员简易模式下保留当前侧边栏仍展示的 SocialOps 管理页面', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: true,
        backendModeEnabled: false,
        hasPendingAuthSession: false,
      }

      for (const path of ['/admin/dashboard', '/accounts', '/task-settings', '/proxies', '/admin/total-accounts', '/admin/global-proxies', '/admin/announcements', '/admin/settings']) {
        expect(simulateGuard(path, { requiresAdmin: path.startsWith('/admin') }, authState)).toBeNull()
      }
    })
  })

  describe('SocialOps proxy route contract', () => {
    it('exposes user-scoped proxy and task settings routes and removes admin proxy route', () => {
      expect(routerSource).toContain("path: '/proxies'")
      expect(routerSource).toContain("name: 'Proxies'")
      expect(routerSource).toContain("component: () => import('@/views/proxies/ProxiesView.vue')")
      expect(routerSource).toContain("path: '/task-settings'")
      expect(routerSource).toContain("name: 'TaskSettings'")
      expect(routerSource).toContain("component: () => import('@/views/task-settings/TaskSettingsView.vue')")
      expect(routerSource).toContain("path: '/admin/global-proxies'")
      expect(routerSource).toContain("name: 'AdminGlobalProxies'")
      expect(routerSource).toContain("component: () => import('@/views/admin/GlobalProxiesView.vue')")
      expect(routerSource).not.toContain("path: '/admin/login-proxies'")
      const removedAdminProxyPath = "'/admin" + "/proxies'"
      const removedAdminProxyName = "'Admin" + "Proxies'"
      expect(routerSource).not.toContain(`path: ${removedAdminProxyPath}`)
      expect(routerSource).not.toContain(`name: ${removedAdminProxyName}`)
      expect(routerSource).not.toContain("admin." + "proxies")
    })
  })

  describe('Backend Mode', () => {
    it('白名单覆盖当前公开支付续接路由', () => {
      const publicRouteWhitelist = sourceBetween(
        'const BACKEND_MODE_ALLOWED_PATHS',
        'const BACKEND_MODE_CALLBACK_PATHS'
      )

      expect(publicRouteWhitelist).toContain("'/payment/result'")
      expect(publicRouteWhitelist).toContain("'/payment/stripe'")
      expect(publicRouteWhitelist).toContain("'/payment/stripe-popup'")
      expect(publicRouteWhitelist).toContain("'/payment/airwallex'")
    })

    it('白名单覆盖当前 OAuth 回调路由和别名', () => {
      const callbackWhitelist = sourceBetween(
        'const BACKEND_MODE_CALLBACK_PATHS',
        'const BACKEND_MODE_PENDING_AUTH_PATHS'
      )

      expect(callbackWhitelist).toContain("'/auth/callback'")
      expect(callbackWhitelist).toContain("'/auth/oauth/callback'")
      expect(callbackWhitelist).toContain("'/auth/dingtalk/callback'")
    })

    it('unauthenticated: /home redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/home', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('unauthenticated: /login is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: /payment/airwallex is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/payment/airwallex', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: Stripe payment continuation routes are allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }

      expect(simulateGuard('/payment/stripe', { requiresAuth: false }, authState)).toBeNull()
      expect(simulateGuard('/payment/stripe-popup', { requiresAuth: false }, authState)).toBeNull()
    })

    it('unauthenticated: /setup is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: initialized /setup redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        setupNeedsSetup: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })

    it('admin: initialized /setup redirects to /admin/dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
        setupNeedsSetup: false,
      }
      const redirect = simulateGuard('/setup', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/dashboard')
    })

    it('admin: /admin/dashboard is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/admin/dashboard', { requiresAdmin: true }, authState)
      expect(redirect).toBeNull()
    })

    it('admin: /login redirects to /admin/dashboard', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: true,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBe('/admin/dashboard')
    })

    it('non-admin authenticated: /dashboard redirects to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/dashboard', {}, authState)
      expect(redirect).toBe('/login')
    })

    it('non-admin authenticated: /login is allowed (no redirect loop)', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/login', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('non-admin authenticated: public self-service pages redirect to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }

      for (const path of ['/home', '/register', '/forgot-password', '/reset-password']) {
        expect(simulateGuard(path, { requiresAuth: false }, authState)).toBe('/login')
      }
    })

    it('non-admin authenticated: backend-mode public continuation pages remain allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }

      for (const path of ['/legal/privacy', '/payment/result', '/auth/oauth/callback']) {
        expect(simulateGuard(path, { requiresAuth: false }, authState)).toBeNull()
      }
    })

    it('non-admin authenticated: /payment/airwallex is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/payment/airwallex', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('non-admin authenticated: Stripe payment continuation routes are allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: true,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }

      expect(simulateGuard('/payment/stripe', { requiresAuth: false }, authState)).toBeNull()
      expect(simulateGuard('/payment/stripe-popup', { requiresAuth: false }, authState)).toBeNull()
    })

    it('unauthenticated: /auth/oauth/callback alias is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/auth/oauth/callback', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: callback routes are allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/auth/wechat/callback', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: WeChat payment callback route is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/auth/wechat/payment/callback', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: /payment/result is allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/payment/result', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: legal document child routes remain allowed', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/legal/privacy', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: same-prefix public route lookalikes redirect to /login', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }

      const lookalikePaths = [
        '/login-extra',
        '/setup-old',
        '/payment/resultx',
        '/payment/stripe-callback',
        '/payment/stripe-popup-extra',
        '/payment/airwallex-callback',
        '/legalish',
      ]

      for (const path of lookalikePaths) {
        expect(simulateGuard(path, { requiresAuth: false }, authState)).toBe('/login')
      }
    })

    it('unauthenticated: /register is allowed when a pending auth session exists', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: true,
      }
      const redirect = simulateGuard('/register', { requiresAuth: false }, authState)
      expect(redirect).toBeNull()
    })

    it('unauthenticated: /email-verify is blocked without a pending auth session', () => {
      const authState: MockAuthState = {
        isAuthenticated: false,
        isAdmin: false,
        isSimpleMode: false,
        backendModeEnabled: true,
        hasPendingAuthSession: false,
      }
      const redirect = simulateGuard('/email-verify', { requiresAuth: false }, authState)
      expect(redirect).toBe('/login')
    })
  })
})
