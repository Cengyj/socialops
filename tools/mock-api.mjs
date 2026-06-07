import http from 'node:http'
import { URL } from 'node:url'
import { createRequire } from 'node:module'

const port = Number(process.env.MOCK_API_PORT || 8080)
const adminEmail = process.env.ADMIN_EMAIL || '3081794680@qq.com'
const adminPassword = process.env.ADMIN_PASSWORD || '668435li'
const dataManagementAgentEnabled = process.env.MOCK_DATA_MANAGEMENT_AGENT_ENABLED === '1'
const socialTaskUnitPrice = 0.1
const require = createRequire(import.meta.url)
let mockXlsx

const now = () => new Date().toISOString()
const daysFromNow = (days) => new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString()
const daysAgo = (days) => new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString()

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

const adminUser = {
  id: 1,
  username: adminEmail,
  email: adminEmail,
  avatar_url: null,
  role: 'admin',
  balance: 128.5,
  concurrency: 30,
  rpm_limit: 0,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  subscriptions: [],
  last_active_at: now(),
  created_at: now(),
  updated_at: now(),
  run_mode: 'standard',
}

const regularUser = {
  id: 2,
  username: 'operator@example.test',
  email: 'operator@example.test',
  avatar_url: null,
  role: 'user',
  balance: 24.75,
  concurrency: 10,
  rpm_limit: 0,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  subscriptions: [],
  last_active_at: now(),
  created_at: now(),
  updated_at: now(),
  run_mode: 'standard',
}

function currentMockUser(req) {
  const token = String(req.headers.authorization || '').replace(/^Bearer\s+/i, '')
  return token === 'dev-mock-user-token' ? regularUser : adminUser
}

const mockIdentityProviders = ['email', 'linuxdo', 'oidc', 'wechat', 'dingtalk']
const mockTotpStates = new Map()
const mockTaskTemplatesByUser = new Map()
let nextMockTaskTemplateId = 1
const mockTaskTemplateTypes = ['login_check', 'post', 'like', 'retweet', 'follow']
const maxMockTaskTemplatePoolValues = 500
const maxMockTaskTemplateValueLength = 2048

function normalizeMockIdentityProvider(provider) {
  const value = String(provider || '').trim().toLowerCase()
  return mockIdentityProviders.includes(value) ? value : ''
}

function mockProviderBindStartPath(provider, redirectTo = '/settings/profile') {
  const normalized = normalizeMockIdentityProvider(provider)
  if (!normalized || normalized === 'email') return ''
  const redirect = String(redirectTo || '').trim() || '/settings/profile'
  const params = new URLSearchParams({
    redirect,
    intent: 'bind_current_user',
  })
  return `/api/v1/auth/oauth/${normalized}/bind/start?${params.toString()}`
}

function isReservedMockEmail(email) {
  const value = String(email || '').trim().toLowerCase()
  return [
    '@linuxdo-connect.invalid',
    '@oidc-connect.invalid',
    '@wechat-connect.invalid',
    '@dingtalk-connect.invalid',
  ].some((suffix) => value.endsWith(suffix))
}

function mockEmailIdentitySummary(user) {
  const summary = {
    provider: 'email',
    bound: false,
    bound_count: 0,
    can_bind: false,
    can_unbind: false,
    note_key: 'profile.authBindings.notes.emailManagedFromProfile',
    note: 'Primary account email is managed from the profile form.',
  }
  const email = String(user?.email || '').trim()
  if (!email || isReservedMockEmail(email)) {
    return summary
  }
  return {
    ...summary,
    bound: true,
    bound_count: 1,
    display_name: email,
    subject_hint: maskEmailForMock(email),
    provider_key: 'email',
  }
}

function mockProviderIdentitySummary(provider) {
  return {
    provider,
    bound: false,
    bound_count: 0,
    bind_start_path: mockProviderBindStartPath(provider),
    can_bind: true,
    can_unbind: false,
  }
}

function mockIdentitySummarySet(user) {
  return {
    email: mockEmailIdentitySummary(user),
    linuxdo: mockProviderIdentitySummary('linuxdo'),
    oidc: mockProviderIdentitySummary('oidc'),
    wechat: mockProviderIdentitySummary('wechat'),
    dingtalk: mockProviderIdentitySummary('dingtalk'),
  }
}

function mockProfileBindingMap(identities) {
  return {
    email: identities.email,
    linuxdo: identities.linuxdo,
    oidc: identities.oidc,
    wechat: identities.wechat,
    dingtalk: identities.dingtalk,
  }
}

function mockProfileResponse(user, { includeRunMode = false } = {}) {
  const response = clone(user)
  if (includeRunMode) {
    response.run_mode = response.run_mode || 'standard'
  } else {
    delete response.run_mode
  }
  const identities = mockIdentitySummarySet(user)
  const bindings = mockProfileBindingMap(identities)
  response.avatar_url = user.avatar_url || null
  response.identities = identities
  response.auth_bindings = bindings
  response.identity_bindings = clone(bindings)
  response.email_bound = identities.email.bound
  response.linuxdo_bound = identities.linuxdo.bound
  response.oidc_bound = identities.oidc.bound
  response.wechat_bound = identities.wechat.bound
  response.dingtalk_bound = identities.dingtalk.bound
  return response
}

function mockTotpState(user) {
  const key = Number(user?.id || 0)
  if (!mockTotpStates.has(key)) {
    mockTotpStates.set(key, {
      enabled: false,
      enabled_at: null,
      setup_token: '',
      secret: '',
    })
  }
  return mockTotpStates.get(key)
}

function mockTotpEnabledAtUnix(state) {
  const millis = Date.parse(String(state.enabled_at || ''))
  return Number.isFinite(millis) ? Math.floor(millis / 1000) : null
}

const publicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  totp_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: false,
  password_reset_enabled: false,
  invitation_code_enabled: false,
  login_agreement_enabled: false,
  login_agreement_mode: 'modal',
  login_agreement_documents: [],
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: 'SocialOps',
  site_logo: '/logo.png',
  site_subtitle: 'Website account pool and social execution billing platform',
  api_base_url: '/api/v1',
  contact_info: '',
  doc_url: '',
  home_content: '',
  hide_ccs_import_button: false,
  payment_enabled: true,
  risk_control_enabled: false,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  wechat_oauth_open_enabled: false,
  wechat_oauth_mp_enabled: false,
  wechat_oauth_mobile_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  backend_mode_enabled: false,
  version: 'dev-mock',
  balance_low_notify_enabled: false,
  account_quota_notify_enabled: false,
  balance_low_notify_threshold: 0,
  affiliate_enabled: false,
}

const adminSettings = {
  ...clone(publicSettings),
  frontend_url: 'http://127.0.0.1:3000',
  smtp_host: 'smtp.example.test',
  smtp_port: 587,
  smtp_username: 'notify@example.test',
  smtp_password_configured: true,
  smtp_from_email: 'notify@example.test',
  smtp_from_name: 'SocialOps',
  smtp_use_tls: true,
  turnstile_secret_key_configured: false,
  api_key_acl_trust_forwarded_ip: false,
  purchase_subscription_enabled: true,
  purchase_subscription_url: '/purchase',
  totp_enabled: false,
  totp_encryption_key_configured: false,
  github_oauth_enabled: false,
  github_oauth_client_id: '',
  github_oauth_client_secret_configured: false,
  github_oauth_redirect_url: '',
  github_oauth_frontend_redirect_url: '/auth/github/callback',
  oidc_connect_enabled: false,
  oidc_connect_provider_name: 'OIDC',
  oidc_connect_client_id: '',
  oidc_connect_client_secret_configured: false,
  oidc_connect_issuer_url: '',
  oidc_connect_discovery_url: '',
  oidc_connect_authorize_url: '',
  oidc_connect_token_url: '',
  oidc_connect_userinfo_url: '',
  oidc_connect_jwks_url: '',
  oidc_connect_scopes: 'openid email profile',
  oidc_connect_redirect_url: '',
  oidc_connect_frontend_redirect_url: '/auth/oidc/callback',
  oidc_connect_token_auth_method: 'client_secret_post',
  oidc_connect_use_pkce: true,
  oidc_connect_validate_id_token: true,
  oidc_connect_allowed_signing_algs: 'RS256,ES256,PS256',
  oidc_connect_clock_skew_seconds: 120,
  oidc_connect_require_email_verified: false,
  oidc_connect_userinfo_email_path: '',
  oidc_connect_userinfo_id_path: '',
  oidc_connect_userinfo_username_path: '',
  wechat_connect_enabled: false,
  wechat_connect_app_id: '',
  wechat_connect_app_secret_configured: false,
  wechat_connect_open_enabled: false,
  wechat_connect_mp_enabled: false,
  wechat_connect_mobile_enabled: false,
  wechat_connect_open_app_id: '',
  wechat_connect_mp_app_id: '',
  wechat_connect_mobile_app_id: '',
  wechat_connect_open_app_secret_configured: false,
  wechat_connect_mp_app_secret_configured: false,
  wechat_connect_mobile_app_secret_configured: false,
  wechat_connect_mode: 'open',
  wechat_connect_scopes: 'snsapi_login',
  wechat_connect_redirect_url: '',
  wechat_connect_frontend_redirect_url: '/auth/wechat/callback',
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: '',
  linuxdo_connect_client_secret_configured: false,
  linuxdo_connect_redirect_url: '',
  dingtalk_connect_enabled: false,
  dingtalk_connect_client_id: '',
  dingtalk_connect_client_secret_configured: false,
  dingtalk_connect_redirect_url: '',
  dingtalk_connect_corp_restriction_policy: 'none',
  dingtalk_connect_internal_corp_id: '',
  dingtalk_connect_bypass_registration: false,
  dingtalk_connect_sync_corp_email: false,
  dingtalk_connect_sync_display_name: true,
  dingtalk_connect_sync_dept: false,
  dingtalk_connect_sync_corp_email_attr_key: '',
  dingtalk_connect_sync_display_name_attr_key: '',
  dingtalk_connect_sync_dept_attr_key: '',
  dingtalk_connect_sync_corp_email_attr_name: '',
  dingtalk_connect_sync_display_name_attr_name: '',
  dingtalk_connect_sync_dept_attr_name: '',
  default_balance: 0,
  default_concurrency: 5,
  default_user_rpm_limit: 0,
  default_subscriptions: [],
  force_email_on_third_party_signup: false,
  affiliate_rebate_rate: 0,
  affiliate_rebate_freeze_hours: 0,
  affiliate_rebate_duration_days: 0,
  affiliate_rebate_per_invitee_cap: 0,
  payment_enabled: true,
  payment_min_amount: 1,
  payment_max_amount: 99999,
  payment_daily_limit: 99999,
  payment_order_timeout_minutes: 30,
  payment_max_pending_orders: 10,
  payment_enabled_types: ['alipay', 'wxpay'],
  payment_balance_disabled: false,
  payment_balance_recharge_multiplier: 1,
  payment_recharge_fee_rate: 0,
  payment_load_balance_strategy: 'round-robin',
  payment_product_name_prefix: 'SocialOps',
  payment_product_name_suffix: '',
  payment_help_image_url: '',
  payment_help_text: 'Mock payment environment',
  payment_cancel_rate_limit_enabled: false,
  payment_cancel_rate_limit_max: 10,
  payment_cancel_rate_limit_window: 1,
  payment_cancel_rate_limit_unit: 'day',
  payment_cancel_rate_limit_window_mode: 'rolling',
  payment_alipay_force_qrcode: false,
  balance_low_notify_threshold: 0,
  balance_low_notify_recharge_url: '/purchase',
  subscription_expiry_notify_enabled: true,
  account_quota_notify_emails: [],
  auth_source_default_email_balance: 0,
  auth_source_default_email_concurrency: 5,
  auth_source_default_email_subscriptions: [],
  auth_source_default_email_grant_on_signup: false,
  auth_source_default_email_grant_on_first_bind: false,
}

for (const source of ['linuxdo', 'oidc', 'wechat', 'github', 'google', 'dingtalk']) {
  adminSettings[`auth_source_default_${source}_balance`] = 0
  adminSettings[`auth_source_default_${source}_concurrency`] = 5
  adminSettings[`auth_source_default_${source}_subscriptions`] = []
  adminSettings[`auth_source_default_${source}_grant_on_signup`] = false
  adminSettings[`auth_source_default_${source}_grant_on_first_bind`] = false
}

const mockAdminSettingSecretFields = [
  ['smtp_password', 'smtp_password_configured'],
  ['turnstile_secret_key', 'turnstile_secret_key_configured'],
  ['linuxdo_connect_client_secret', 'linuxdo_connect_client_secret_configured'],
  ['dingtalk_connect_client_secret', 'dingtalk_connect_client_secret_configured'],
  ['wechat_connect_app_secret', 'wechat_connect_app_secret_configured'],
  ['wechat_connect_open_app_secret', 'wechat_connect_open_app_secret_configured'],
  ['wechat_connect_mp_app_secret', 'wechat_connect_mp_app_secret_configured'],
  ['wechat_connect_mobile_app_secret', 'wechat_connect_mobile_app_secret_configured'],
  ['oidc_connect_client_secret', 'oidc_connect_client_secret_configured'],
  ['github_oauth_client_secret', 'github_oauth_client_secret_configured'],
  ['google_oauth_client_secret', 'google_oauth_client_secret_configured'],
]

const mockPublicSettingDirectKeys = [
  'registration_enabled',
  'email_verify_enabled',
  'totp_enabled',
  'force_email_on_third_party_signup',
  'registration_email_suffix_whitelist',
  'promo_code_enabled',
  'password_reset_enabled',
  'invitation_code_enabled',
  'login_agreement_enabled',
  'login_agreement_mode',
  'login_agreement_documents',
  'turnstile_enabled',
  'turnstile_site_key',
  'site_name',
  'site_logo',
  'site_subtitle',
  'api_base_url',
  'contact_info',
  'doc_url',
  'home_content',
  'payment_enabled',
  'risk_control_enabled',
  'purchase_subscription_enabled',
  'purchase_subscription_url',
  'table_default_page_size',
  'table_page_size_options',
  'custom_endpoints',
  'backend_mode_enabled',
  'version',
  'balance_low_notify_enabled',
  'account_quota_notify_enabled',
  'balance_low_notify_threshold',
  'balance_low_notify_recharge_url',
  'affiliate_enabled',
]

function stripMockAdminSettingSecrets() {
  for (const [secretKey, configuredKey] of mockAdminSettingSecretFields) {
    if (adminSettings[secretKey]) {
      adminSettings[configuredKey] = true
    }
    delete adminSettings[secretKey]
  }
}

function mockUserVisibleMenuItems(items) {
  if (!Array.isArray(items)) return []
  return items.filter((item) => item && item.visibility === 'user').map(clone)
}

function resolveMockWeChatPublicCapabilities() {
  const enabled = adminSettings.wechat_connect_enabled === true
  const hasExplicitCapabilities = ['wechat_connect_open_enabled', 'wechat_connect_mp_enabled', 'wechat_connect_mobile_enabled']
    .some((key) => typeof adminSettings[key] === 'boolean')
  if (hasExplicitCapabilities) {
    return {
      open: adminSettings.wechat_connect_open_enabled === true,
      mp: adminSettings.wechat_connect_mp_enabled === true,
      mobile: adminSettings.wechat_connect_mobile_enabled === true,
    }
  }
  if (!enabled) {
    return { open: false, mp: false, mobile: false }
  }
  switch (String(adminSettings.wechat_connect_mode || 'open')) {
    case 'mp':
      return { open: false, mp: true, mobile: false }
    case 'mobile':
      return { open: false, mp: false, mobile: true }
    default:
      return { open: true, mp: false, mobile: false }
  }
}

function syncMockPublicSettingsFromAdminSettings() {
  for (const key of mockPublicSettingDirectKeys) {
    if (adminSettings[key] !== undefined) {
      publicSettings[key] = clone(adminSettings[key])
    }
  }
  publicSettings.custom_menu_items = mockUserVisibleMenuItems(adminSettings.custom_menu_items)
  publicSettings.linuxdo_oauth_enabled = adminSettings.linuxdo_connect_enabled === true
  publicSettings.dingtalk_oauth_enabled = adminSettings.dingtalk_connect_enabled === true
  publicSettings.oidc_oauth_enabled = adminSettings.oidc_connect_enabled === true
  publicSettings.oidc_oauth_provider_name = String(adminSettings.oidc_connect_provider_name || 'OIDC')
  publicSettings.github_oauth_enabled = adminSettings.github_oauth_enabled === true
  publicSettings.google_oauth_enabled = adminSettings.google_oauth_enabled === true
  const wechat = resolveMockWeChatPublicCapabilities()
  publicSettings.wechat_oauth_enabled = wechat.open || wechat.mp
  publicSettings.wechat_oauth_open_enabled = wechat.open
  publicSettings.wechat_oauth_mp_enabled = wechat.mp
  publicSettings.wechat_oauth_mobile_enabled = wechat.mobile
}

stripMockAdminSettingSecrets()
syncMockPublicSettingsFromAdminSettings()

let adminApiKey = {
  exists: true,
  masked_key: 'sk-mock...7f2a',
}

const emailTemplateEvents = [
  { value: 'auth.verify_code', label: 'Email Verification Code', category: 'auth' },
  { value: 'subscription.purchase_success', label: 'Subscription Activated', category: 'subscription' },
]

const emailTemplateStore = new Map()

function emailTemplateKey(event, locale) {
  return `${event}::${locale}`
}

function defaultEmailTemplate(event, locale) {
  const lang = String(locale || '').toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const subject = lang === 'zh'
    ? 'SocialOps 通知 - {{site_name}}'
    : 'SocialOps notification - {{site_name}}'
  const html = lang === 'zh'
    ? '<p>您好，{{recipient_name}}。</p><p>这是一封来自 {{site_name}} 的系统通知。</p>'
    : '<p>Hello {{recipient_name}},</p><p>This is a system notification from {{site_name}}.</p>'
  return {
    event,
    locale,
    subject,
    html,
    is_custom: false,
    updated_at: now(),
    placeholders: ['site_name', 'recipient_name', 'recipient_email', 'verification_code'],
  }
}

function getStoredEmailTemplate(event, locale) {
  return emailTemplateStore.get(emailTemplateKey(event, locale)) || defaultEmailTemplate(event, locale)
}

const adminPaymentConfig = {
  enabled: true,
  min_amount: 1,
  max_amount: 99999,
  daily_limit: 99999,
  order_timeout_minutes: 30,
  max_pending_orders: 10,
  enabled_payment_types: ['alipay', 'wxpay'],
  balance_disabled: false,
  balance_recharge_multiplier: 1,
  recharge_fee_rate: 0,
  load_balance_strategy: 'round-robin',
  product_name_prefix: 'SocialOps',
  product_name_suffix: '',
  help_image_url: '',
  help_text: 'Mock payment environment',
  stripe_publishable_key: '',
  cancel_rate_limit_enabled: false,
  cancel_rate_limit_max: 10,
  cancel_rate_limit_window: 1,
  cancel_rate_limit_unit: 'day',
  cancel_rate_limit_window_mode: 'rolling',
  alipay_force_qrcode: false,
}

const paymentConfig = {
  enabled: true,
  min_amount: 1,
  max_amount: 99999,
  daily_limit: 99999,
  max_pending_orders: 10,
  order_timeout_minutes: 30,
  balance_disabled: false,
  balance_recharge_multiplier: 1,
  recharge_fee_rate: 0,
  load_balance_strategy: 'round-robin',
  enabled_payment_types: ['alipay', 'wxpay'],
  product_name_prefix: 'SocialOps',
  product_name_suffix: '',
  help_image_url: '',
  help_text: 'Mock payment environment',
  stripe_publishable_key: '',
  cancel_rate_limit_enabled: false,
  cancel_rate_limit_max: 10,
  cancel_rate_limit_window: 1,
  cancel_rate_limit_unit: 'day',
  cancel_rate_limit_window_mode: 'rolling',
  alipay_force_qrcode: false,
}

const dataManagementConfig = {
  source_mode: 'docker_exec',
  backup_root: '/var/lib/socialops/backups',
  retention_days: 14,
  keep_last: 10,
  active_postgres_profile_id: 'mock-postgres',
  active_redis_profile_id: 'mock-redis',
  active_s3_profile_id: 'mock-r2',
  postgres: {
    host: 'postgres',
    port: 5432,
    user: 'socialops',
    password_configured: true,
    database: 'socialops',
    ssl_mode: 'disable',
    container_name: 'socialops-postgres',
  },
  redis: {
    addr: 'redis:6379',
    username: '',
    password_configured: true,
    db: 0,
    container_name: 'socialops-redis',
  },
  s3: {
    enabled: true,
    endpoint: 'https://example.r2.cloudflarestorage.com',
    region: 'auto',
    bucket: 'socialops-backups',
    access_key_id: 'mock-access-key',
    secret_access_key_configured: true,
    prefix: 'backups/',
    force_path_style: false,
    use_ssl: true,
  },
}

const mockSourceProfiles = {
  postgres: [
    {
      source_type: 'postgres',
      profile_id: 'mock-postgres',
      name: 'Mock PostgreSQL',
      is_active: true,
      password_configured: true,
      config: {
        host: 'postgres',
        port: 5432,
        user: 'socialops',
        database: 'socialops',
        ssl_mode: 'disable',
        addr: '',
        username: '',
        db: 0,
        container_name: 'socialops-postgres',
      },
      created_at: now(),
      updated_at: now(),
    },
  ],
  redis: [
    {
      source_type: 'redis',
      profile_id: 'mock-redis',
      name: 'Mock Redis',
      is_active: true,
      password_configured: true,
      config: {
        host: '',
        port: 0,
        user: '',
        database: '',
        ssl_mode: '',
        addr: 'redis:6379',
        username: '',
        db: 0,
        container_name: 'socialops-redis',
      },
      created_at: now(),
      updated_at: now(),
    },
  ],
}

const mockS3Profiles = [
  {
    profile_id: 'mock-r2',
    name: 'Mock R2',
    is_active: true,
    s3: clone(dataManagementConfig.s3),
    secret_access_key_configured: true,
    created_at: now(),
    updated_at: now(),
  },
]

let mockBackupJobs = []
let mockBackupS3Config = {
  endpoint: dataManagementConfig.s3.endpoint,
  region: dataManagementConfig.s3.region,
  bucket: dataManagementConfig.s3.bucket,
  access_key_id: dataManagementConfig.s3.access_key_id,
  secret_access_key: 'mock-secret-key',
  prefix: dataManagementConfig.s3.prefix,
  force_path_style: dataManagementConfig.s3.force_path_style,
}
let mockBackupSchedule = {
  enabled: false,
  cron_expr: '0 2 * * *',
  retain_days: dataManagementConfig.retention_days,
  retain_count: dataManagementConfig.keep_last,
}
let mockBackupRecords = []
let nextProviderId = 20
let mockPaymentProviders = [
  {
    id: 1,
    provider_key: 'alipay',
    name: 'Mock Alipay',
    config: {},
    supported_types: ['alipay'],
    enabled: true,
    payment_mode: '',
    refund_enabled: true,
    allow_user_refund: true,
    limits: '',
    sort_order: 1,
    created_at: now(),
    updated_at: now(),
  },
  {
    id: 2,
    provider_key: 'wxpay',
    name: 'Mock WeChat Pay',
    config: {},
    supported_types: ['wxpay'],
    enabled: true,
    payment_mode: '',
    refund_enabled: false,
    allow_user_refund: false,
    limits: '',
    sort_order: 2,
    created_at: now(),
    updated_at: now(),
  },
]

const methodLimits = {
  alipay: {
    currency: 'CNY',
    daily_limit: 0,
    daily_used: 0,
    daily_remaining: 0,
    single_min: 1,
    single_max: 99999,
    fee_rate: 0,
    available: true,
  },
  wxpay: {
    currency: 'CNY',
    daily_limit: 0,
    daily_used: 0,
    daily_remaining: 0,
    single_min: 1,
    single_max: 99999,
    fee_rate: 0,
    available: true,
  },
}

const mockGroups = [
  {
    id: 1,
    name: 'X Execution Pool',
    description: 'Shared execution pool for X actions',
    platform: 'x_twitter',
    rate_multiplier: 1,
    rpm_limit: 0,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'subscription',
    daily_limit_usd: 10,
    weekly_limit_usd: 50,
    monthly_limit_usd: 200,
    sort_order: 1,
    created_at: now(),
    updated_at: now(),
  },
  {
    id: 2,
    name: 'Instagram Execution Pool',
    description: 'Shared execution pool for Instagram actions',
    platform: 'instagram',
    rate_multiplier: 1,
    rpm_limit: 0,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'subscription',
    daily_limit_usd: 8,
    weekly_limit_usd: 35,
    monthly_limit_usd: 120,
    sort_order: 2,
    created_at: now(),
    updated_at: now(),
  },
]

let mockPlans = [
  {
    id: 101,
    group_id: 1,
    platform: 'x_twitter',
    name: 'X Starter Monthly',
    description: 'Entry package for login, follow, like, repost, and posting tasks.',
    price: 29,
    original_price: 39,
    validity_days: 1,
    validity_unit: 'months',
    daily_limit_usd: 6,
    weekly_limit_usd: 28,
    monthly_limit_usd: 100,
    features: ['X login', 'Follow tasks', 'Like tasks', 'Repost tasks', 'Posting tasks'],
    product_name: 'X Starter Monthly',
    for_sale: true,
    sort_order: 1,
  },
  {
    id: 102,
    group_id: 1,
    platform: 'x_twitter',
    name: 'X Growth Monthly',
    description: 'Higher-volume X package with more execution allowance.',
    price: 59,
    original_price: 79,
    validity_days: 1,
    validity_unit: 'months',
    daily_limit_usd: 12,
    weekly_limit_usd: 55,
    monthly_limit_usd: 220,
    features: ['All Starter actions', 'Higher daily allowance', 'Higher weekly allowance'],
    product_name: 'X Growth Monthly',
    for_sale: true,
    sort_order: 2,
  },
  {
    id: 201,
    group_id: 2,
    platform: 'instagram',
    name: 'Instagram Action Monthly',
    description: 'Instagram execution package for follow, like, and posting workflows.',
    price: 49,
    original_price: 65,
    validity_days: 1,
    validity_unit: 'months',
    daily_limit_usd: 8,
    weekly_limit_usd: 35,
    monthly_limit_usd: 140,
    features: ['Instagram login', 'Follow tasks', 'Like tasks', 'Posting tasks'],
    product_name: 'Instagram Action Monthly',
    for_sale: true,
    sort_order: 3,
  },
]

function findGroup(groupId) {
  return mockGroups.find((group) => group.id === Number(groupId)) || null
}

function decoratePlan(plan) {
  const group = findGroup(plan.group_id)
  const quotaUSD = normalizePositiveLimit(plan.quota_usd ?? plan.monthly_limit_usd)
  return {
    ...clone(plan),
    quota_usd: quotaUSD,
    monthly_limit_usd: quotaUSD,
    group_platform: group?.platform || plan.platform || 'social',
    group_name: group?.name || '',
    group_status: group?.status || 'active',
    subscription_type: group?.subscription_type || 'subscription',
  }
}

function firstPlanForGroup(groupId) {
  return mockPlans
    .filter((plan) => plan.group_id === Number(groupId))
    .sort((a, b) => a.sort_order - b.sort_order)[0] || null
}

function normalizePositiveLimit(value) {
  const num = Number(value)
  return Number.isFinite(num) && num > 0 ? num : null
}

function quotaFieldProvided(value) {
  return value !== undefined && value !== null && value !== ''
}

function resolvePlanQuotaInput(body = {}, { required = false } = {}) {
  const hasQuota = quotaFieldProvided(body.quota_usd)
  const hasMonthly = quotaFieldProvided(body.monthly_limit_usd)
  if (!hasQuota && !hasMonthly) {
    return required
      ? { error: { code: 'PLAN_QUOTA_REQUIRED', message: 'quota amount must be greater than 0' } }
      : { touched: false, value: null }
  }
  const quota = hasQuota ? Number(body.quota_usd) : null
  const monthly = hasMonthly ? Number(body.monthly_limit_usd) : null
  if (hasQuota && hasMonthly && Math.abs(quota - monthly) >= 1e-9) {
    return { error: { code: 'PLAN_QUOTA_CONFLICT', message: 'quota_usd must match monthly_limit_usd when both are provided' } }
  }
  const value = hasQuota ? quota : monthly
  if (!Number.isFinite(value) || value <= 0) {
    return { error: { code: 'PLAN_QUOTA_REQUIRED', message: 'quota amount must be greater than 0' } }
  }
  return { touched: true, value }
}

function buildSubscription({
  id,
  userId,
  group,
  plan,
  status = 'active',
  startsAt = daysAgo(3),
  expiresAt = daysFromNow(27),
  dailyUsage = 1.6,
  weeklyUsage = 6.4,
  monthlyUsage = 18.2,
}) {
  const quotaUSD = normalizePositiveLimit(plan?.quota_usd ?? plan?.monthly_limit_usd ?? group.monthly_limit_usd)
  return {
    id,
    user_id: userId,
    group_id: group.id,
    plan_id: plan?.id ?? null,
    plan_name: plan?.name || group.name,
    plan_platform: plan?.platform || group.platform,
    quota_usd: quotaUSD,
    daily_limit_usd: normalizePositiveLimit(plan?.daily_limit_usd ?? group.daily_limit_usd),
    weekly_limit_usd: normalizePositiveLimit(plan?.weekly_limit_usd ?? group.weekly_limit_usd),
    monthly_limit_usd: quotaUSD,
    status,
    starts_at: startsAt,
    expires_at: expiresAt,
    daily_usage_usd: dailyUsage,
    weekly_usage_usd: weeklyUsage,
    monthly_usage_usd: monthlyUsage,
    daily_window_start: daysAgo(0),
    weekly_window_start: daysAgo(2),
    monthly_window_start: daysAgo(10),
    created_at: startsAt,
    updated_at: now(),
    user: clone(adminUser),
    group: clone(group),
  }
}

let nextSubscriptionId = 1000
let nextPlanId = 300
let nextOrderId = 5000
let nextRedeemCodeId = 8000

let mockSubscriptions = [
  buildSubscription({
    id: nextSubscriptionId++,
    userId: adminUser.id,
    group: findGroup(1),
    plan: decoratePlan(firstPlanForGroup(1)),
    status: 'active',
    startsAt: daysAgo(4),
    expiresAt: daysFromNow(26),
    dailyUsage: 2.4,
    weeklyUsage: 11.2,
    monthlyUsage: 34.6,
  }),
  buildSubscription({
    id: nextSubscriptionId++,
    userId: adminUser.id,
    group: findGroup(2),
    plan: decoratePlan(firstPlanForGroup(2)),
    status: 'expired',
    startsAt: daysAgo(40),
    expiresAt: daysAgo(5),
    dailyUsage: 0.5,
    weeklyUsage: 2.4,
    monthlyUsage: 12.8,
  }),
]

adminUser.subscriptions = mockSubscriptions.filter((subscription) => subscription.status === 'active')

const mockChannels = []
let mockOrders = []
let mockRedeemCodes = []
let nextPromoCodeId = 8500
let mockPromoCodes = []
let mockPromoUsages = []
let nextAffiliateLedgerId = 8700
let mockAffiliateProfiles = [
  {
    user_id: adminUser.id,
    aff_code: 'ADMINMOCK',
    aff_code_custom: false,
    aff_rebate_rate_percent: null,
    inviter_id: null,
    aff_count: 1,
    aff_quota: 0,
    aff_frozen_quota: 0,
    aff_history_quota: 0,
    created_at: daysAgo(12),
    updated_at: now(),
  },
  {
    user_id: regularUser.id,
    aff_code: 'OPSMOCK',
    aff_code_custom: false,
    aff_rebate_rate_percent: null,
    inviter_id: adminUser.id,
    aff_count: 0,
    aff_quota: 3.5,
    aff_frozen_quota: 1.25,
    aff_history_quota: 8.75,
    created_at: daysAgo(9),
    updated_at: now(),
  },
]
let mockAffiliateTransfers = []
let nextAnnouncementId = 7000
let mockAnnouncements = [
  {
    id: nextAnnouncementId++,
    title: 'Mock commercial readiness notice',
    content: 'This mock announcement verifies user-side announcement loading without leaking internal errors.',
    status: 'active',
    notify_mode: 'silent',
    targeting: { any_of: [] },
    starts_at: daysAgo(1),
    ends_at: daysFromNow(30),
    created_at: daysAgo(1),
    updated_at: now(),
  },
]
const mockAnnouncementReads = new Map()
let nextSocialAccountId = 3000
let nextSocialTaskLogId = 9000
let nextMockProxyId = 5000

const regularUserSeedProxy = {
  id: nextMockProxyId++,
  user_id: regularUser.id,
  name: 'Mock execution proxy',
  ip_type: 'residential',
  endpoint: 'http://8.8.8.8:8080',
  status: 'online',
  latency_ms: 80,
  last_check_at: now(),
  remark: 'Seed proxy for mock execution readiness',
  created_at: daysAgo(8),
  updated_at: now(),
}

let mockSocialAccounts = [
  {
    id: nextSocialAccountId++,
    name: 'x_growth_ops_01',
    platform: 'x_twitter',
    username: 'x_growth_ops_01',
    platform_user_id: 'x-10001',
    password: 'mock-pass-01',
    phone: '+1 555 0101',
    email: 'x-growth-01@example.test',
    email_password: 'mail-pass-01',
    two_factor: 'JBSWY3DPEHPK3PXP',
    backup_code: '',
    email_client_id: '',
    email_token: '',
    registration_ip: '203.0.113.10',
    auth_cookie: 'ct0=mock; auth_token=mock-token-01',
    execution_auth: '{"access_token":"mock-token-01","token_secret":"mock-secret-01"}',
    account_status: 'available',
    task_status: 'stored',
    task_message: 'Ready for execution',
    remark: 'Seed account for admin workbench smoke testing',
    assigned_user_id: regularUser.id,
    proxy_id: regularUserSeedProxy.id,
    default_proxy_snapshot: JSON.stringify({
      id: regularUserSeedProxy.id,
      name: regularUserSeedProxy.name,
      ip_type: regularUserSeedProxy.ip_type,
      endpoint: regularUserSeedProxy.endpoint,
      status: regularUserSeedProxy.status,
    }),
    created_at: daysAgo(8),
    updated_at: now(),
  },
  {
    id: nextSocialAccountId++,
    name: 'ig_brand_ops_01',
    platform: 'instagram',
    username: 'ig_brand_ops_01',
    platform_user_id: 'ig-20001',
    password: 'mock-pass-02',
    phone: '+1 555 0102',
    email: 'ig-brand-01@example.test',
    email_password: 'mail-pass-02',
    two_factor: '',
    backup_code: '',
    email_client_id: '',
    email_token: '',
    registration_ip: '',
    auth_cookie: '',
    execution_auth: '',
    account_status: 'pending_check',
    task_status: 'stored',
    task_message: 'Waiting for login check',
    remark: 'Imported from mock CSV',
    assigned_user_id: null,
    proxy_id: null,
    default_proxy_snapshot: '',
    created_at: daysAgo(5),
    updated_at: now(),
  },
  {
    id: nextSocialAccountId++,
    name: 'x_limited_ops_02',
    platform: 'x_twitter',
    username: 'x_limited_ops_02',
    platform_user_id: 'x-10002',
    password: 'mock-pass-03',
    phone: '+1 555 0103',
    email: 'x-limited-02@example.test',
    email_password: 'mail-pass-03',
    two_factor: '',
    backup_code: '',
    email_client_id: '',
    email_token: '',
    registration_ip: '',
    auth_cookie: 'ct0=mock-limited',
    execution_auth: '{"access_token":"mock-limited","token_secret":"mock-limited-secret"}',
    account_status: 'limited',
    task_status: 'failed',
    task_message: 'Platform rate limited, retry later',
    remark: 'Shows non-executable state in the workbench',
    assigned_user_id: adminUser.id,
    proxy_id: null,
    default_proxy_snapshot: '',
    created_at: daysAgo(2),
    updated_at: now(),
  },
]

let mockSocialTaskLogs = [
  {
    id: nextSocialTaskLogId++,
    social_account_id: mockSocialAccounts[0].id,
    user_id: regularUser.id,
    action: 'login_check',
    platform: mockSocialAccounts[0].platform,
    account_name: mockSocialAccounts[0].name,
    target: '',
    content: '',
    status: 'success',
    result_message: 'Login check succeeded',
    charged: true,
    charged_amount: socialTaskUnitPrice,
    price: socialTaskUnitPrice,
    charge_status: 'charged',
    charge_source: 'subscription',
    proxy_id: null,
    proxy_snapshot: '',
    billing_request_id: 'mock-billing-1',
    idempotency_key: 'mock-task-1',
    executed_at: daysAgo(1),
    created_at: daysAgo(1),
  },
]

let mockProxies = [regularUserSeedProxy]

function qrDataUrl(label) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="280" height="280"><rect width="100%" height="100%" fill="#ffffff"/><rect x="24" y="24" width="232" height="232" rx="16" fill="#111827"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="#ffffff" font-family="Arial" font-size="18">${label}</text></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

function send(res, status, payload) {
  res.writeHead(status, {
    'Content-Type': 'application/json; charset=utf-8',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization, Accept-Language, X-Idempotency-Key',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
  })
  res.end(JSON.stringify(payload))
}

function ok(res, data) {
  send(res, 200, { code: 0, message: 'success', data })
}

function accepted(res, data) {
  send(res, 202, { code: 0, message: 'accepted', data })
}

function paginated(res, items = [], page = 1, pageSize = 20, total = items.length) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  ok(res, {
    items,
    total,
    page,
    page_size: pageSize,
    pages,
  })
}

function paginatedFromUrl(res, url, items = []) {
  const page = Math.max(1, Number(url.searchParams.get('page') || 1) || 1)
  const pageSize = Math.max(1, Math.min(1000, Number(url.searchParams.get('page_size') || 20) || 20))
  const start = (page - 1) * pageSize
  paginated(res, items.slice(start, start + pageSize), page, pageSize, items.length)
}

function mockAnnouncementReadKey(announcementId, userId) {
  return `${Number(announcementId)}:${Number(userId)}`
}

function mockAnnouncementReadAt(announcementId, userId) {
  return mockAnnouncementReads.get(mockAnnouncementReadKey(announcementId, userId)) || null
}

function normalizeMockAnnouncementTargeting(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return { any_of: [] }
  }
  const anyOf = Array.isArray(value.any_of) ? value.any_of : []
  return { any_of: clone(anyOf) }
}

function mockAnnouncementTime(value) {
  if (value === undefined || value === null || value === '') return null
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value > 0 ? new Date(value * 1000).toISOString() : null
  }
  const trimmed = String(value).trim()
  if (!trimmed || trimmed === '0') return null
  if (/^\d+$/.test(trimmed)) {
    return new Date(Number(trimmed) * 1000).toISOString()
  }
  return trimmed
}

function mockAnnouncementIsActive(item) {
  if (!item || item.status !== 'active') return false
  const nowMs = Date.now()
  if (item.starts_at && new Date(item.starts_at).getTime() > nowMs) return false
  if (item.ends_at && new Date(item.ends_at).getTime() <= nowMs) return false
  return true
}

function mockAnnouncementConditionMatches(condition, user) {
  if (!condition || typeof condition !== 'object') return false
  if (condition.type === 'balance') {
    const balance = Number(user?.balance || 0)
    const value = Number(condition.value || 0)
    switch (condition.operator) {
      case 'gt': return balance > value
      case 'gte': return balance >= value
      case 'lt': return balance < value
      case 'lte': return balance <= value
      case 'eq': return balance === value
      default: return false
    }
  }
  if (condition.type === 'subscription') {
    const groupIds = Array.isArray(condition.group_ids) ? condition.group_ids.map(Number) : []
    const subscriptions = Array.isArray(user?.subscriptions) ? user.subscriptions : []
    return condition.operator === 'in' && subscriptions.some((sub) => groupIds.includes(Number(sub.group_id)))
  }
  return false
}

function mockAnnouncementVisibleToUser(item, user) {
  const targeting = normalizeMockAnnouncementTargeting(item?.targeting)
  if (targeting.any_of.length === 0) return true
  return targeting.any_of.some((group) => {
    const allOf = Array.isArray(group?.all_of) ? group.all_of : []
    return allOf.length > 0 && allOf.every((condition) => mockAnnouncementConditionMatches(condition, user))
  })
}

function mockAdminAnnouncementForResponse(item) {
  return {
    id: item.id,
    title: item.title,
    content: item.content,
    status: item.status,
    notify_mode: item.notify_mode || 'silent',
    targeting: normalizeMockAnnouncementTargeting(item.targeting),
    starts_at: item.starts_at || null,
    ends_at: item.ends_at || null,
    created_at: item.created_at,
    updated_at: item.updated_at,
  }
}

function mockUserAnnouncementForResponse(item, user) {
  return {
    id: item.id,
    title: item.title,
    content: item.content,
    notify_mode: item.notify_mode || 'silent',
    starts_at: item.starts_at || null,
    ends_at: item.ends_at || null,
    read_at: mockAnnouncementReadAt(item.id, user.id),
    created_at: item.created_at,
    updated_at: item.updated_at,
  }
}

function mockAnnouncementReadStatusRows(announcement) {
  return [adminUser, regularUser].map((user) => ({
    user_id: user.id,
    email: user.email,
    username: user.username,
    balance: user.balance,
    eligible: mockAnnouncementVisibleToUser(announcement, user),
    read_at: mockAnnouncementReadAt(announcement.id, user.id),
  }))
}

function mockAnnouncementFromCreateBody(body) {
  const timestamp = now()
  return {
    id: nextAnnouncementId++,
    title: String(body.title || 'Mock announcement'),
    content: String(body.content || ''),
    status: String(body.status || 'draft'),
    notify_mode: String(body.notify_mode || 'silent'),
    targeting: normalizeMockAnnouncementTargeting(body.targeting),
    starts_at: mockAnnouncementTime(body.starts_at),
    ends_at: mockAnnouncementTime(body.ends_at),
    created_at: timestamp,
    updated_at: timestamp,
  }
}

function applyMockAnnouncementUpdate(announcement, body) {
  if (body.title !== undefined) announcement.title = String(body.title)
  if (body.content !== undefined) announcement.content = String(body.content)
  if (body.status !== undefined) announcement.status = String(body.status)
  if (body.notify_mode !== undefined) announcement.notify_mode = String(body.notify_mode)
  if (body.targeting !== undefined) announcement.targeting = normalizeMockAnnouncementTargeting(body.targeting)
  if (body.starts_at !== undefined) announcement.starts_at = mockAnnouncementTime(body.starts_at)
  if (body.ends_at !== undefined) announcement.ends_at = mockAnnouncementTime(body.ends_at)
  announcement.updated_at = now()
}

function roundMoney(value) {
  return Math.round(Number(value || 0) * 100) / 100
}

function paymentProviderForType(paymentType) {
  const normalized = String(paymentType || '').trim()
  const baseType = normalized.split('_')[0] || normalized
  return mockPaymentProviders.find((provider) => {
    if (!provider.enabled) return false
    if (provider.provider_key === normalized || provider.provider_key === baseType) return true
    return Array.isArray(provider.supported_types) && provider.supported_types.includes(normalized)
  }) || null
}

function buildMockPaymentOrder(req, body = {}) {
  const user = currentMockUser(req)
  const amount = roundMoney(body.amount)
  const orderType = String(body.order_type || 'balance')
  const paymentType = String(body.payment_type || 'alipay')
  const provider = paymentProviderForType(paymentType)
  return {
    id: nextOrderId++,
    user_id: user.id,
    user_email: user.email,
    user_name: user.username,
    user_notes: user.notes || null,
    amount,
    pay_amount: amount,
    currency: 'CNY',
    fee_rate: 0,
    payment_type: paymentType,
    payment_trade_no: '',
    out_trade_no: `MOCK-${Date.now()}-${nextOrderId}`,
    status: 'PENDING',
    order_type: orderType,
    created_at: now(),
    updated_at: now(),
    expires_at: daysFromNow(1 / 48),
    paid_at: null,
    completed_at: null,
    failed_at: null,
    failed_reason: null,
    refund_amount: 0,
    refund_reason: null,
    refund_at: null,
    refund_requested_at: null,
    refund_requested_by: null,
    refund_request_reason: null,
    plan_id: orderType === 'subscription' ? Number(body.plan_id || 0) || undefined : undefined,
    subscription_group_id: undefined,
    subscription_days: undefined,
    provider_instance_id: provider ? String(provider.id) : undefined,
    provider_key: provider?.provider_key || paymentType,
    pay_url: `https://example.test/pay/mock`,
    qr_code: qrDataUrl(orderType === 'subscription' ? 'Mock Subscription Pay' : 'Mock Recharge Pay'),
    qr_code_img: null,
    recharge_code: '',
    client_ip: req.socket?.remoteAddress || '',
    src_host: req.headers.host || '',
    src_url: req.headers.referer || null,
    force_refund: false,
  }
}

function sanitizeUserPaymentOrder(order) {
  if (!order) return null
  return {
    id: order.id,
    user_id: order.user_id,
    amount: order.amount,
    pay_amount: order.pay_amount,
    fee_rate: order.fee_rate,
    currency: order.currency,
    payment_type: order.payment_type,
    out_trade_no: order.out_trade_no,
    status: order.status,
    order_type: order.order_type,
    created_at: order.created_at,
    expires_at: order.expires_at,
    paid_at: order.paid_at || undefined,
    completed_at: order.completed_at || undefined,
    refund_amount: order.refund_amount,
    refund_reason: order.refund_reason || undefined,
    refund_requested_at: order.refund_requested_at || undefined,
    refund_requested_by: order.refund_requested_by || undefined,
    refund_request_reason: order.refund_request_reason || undefined,
    plan_id: order.plan_id,
    provider_instance_id: order.provider_instance_id,
  }
}

function sanitizePublicPaymentOrder(order) {
  if (!order) return null
  const userOrder = sanitizeUserPaymentOrder(order)
  delete userOrder.user_id
  delete userOrder.provider_instance_id
  return userOrder
}

function sanitizeAdminPaymentOrder(order) {
  return order ? clone(order) : null
}

function findPaymentOrderByID(id) {
  return mockOrders.find((order) => order.id === Number(id)) || null
}

function filterPaymentOrders(url, { userId = null } = {}) {
  let orders = mockOrders.slice()
  if (userId !== null && userId !== undefined) {
    orders = orders.filter((order) => order.user_id === Number(userId))
  }
  const queryUserID = Number(url.searchParams.get('user_id') || 0) || 0
  if (queryUserID > 0) {
    orders = orders.filter((order) => order.user_id === queryUserID)
  }
  for (const field of ['status', 'order_type', 'payment_type']) {
    const value = String(url.searchParams.get(field) || '').trim()
    if (value) {
      orders = orders.filter((order) => String(order[field] || '') === value)
    }
  }
  const keyword = String(url.searchParams.get('keyword') || '').trim().toLowerCase()
  if (keyword) {
    orders = orders.filter((order) => {
      return [order.out_trade_no, order.user_email, order.user_name]
        .some((value) => String(value || '').toLowerCase().includes(keyword))
    })
  }
  return orders.sort((a, b) => {
    const timeDelta = Date.parse(b.created_at) - Date.parse(a.created_at)
    return timeDelta || b.id - a.id
  })
}

function paymentDashboardStats(daysInput) {
  const days = Math.max(1, Number(daysInput || 30) || 30)
  const paidStatuses = new Set(['COMPLETED', 'PAID', 'RECHARGING'])
  const today = new Date()
  const todayKey = today.toISOString().slice(0, 10)
  const since = new Date(Date.now() - days * 24 * 60 * 60 * 1000)
  const paidOrders = mockOrders.filter((order) => {
    if (!paidStatuses.has(order.status)) return false
    if (!order.paid_at) return false
    return Date.parse(order.paid_at) >= since.getTime()
  })
  const todayOrders = paidOrders.filter((order) => String(order.paid_at || '').slice(0, 10) === todayKey)
  const totalAmount = roundMoney(paidOrders.reduce((sum, order) => sum + Number(order.pay_amount || 0), 0))
  const todayAmount = roundMoney(todayOrders.reduce((sum, order) => sum + Number(order.pay_amount || 0), 0))
  const methodMap = new Map()
  const userMap = new Map()
  const dailyMap = new Map()
  for (const order of paidOrders) {
    const method = methodMap.get(order.payment_type) || { type: order.payment_type, amount: 0, count: 0 }
    method.amount = roundMoney(method.amount + Number(order.pay_amount || 0))
    method.count += 1
    methodMap.set(order.payment_type, method)

    const user = userMap.get(order.user_id) || { user_id: order.user_id, email: order.user_email || '', amount: 0 }
    user.amount = roundMoney(user.amount + Number(order.pay_amount || 0))
    userMap.set(order.user_id, user)

    const date = String(order.paid_at || '').slice(0, 10)
    const day = dailyMap.get(date) || { date, amount: 0, count: 0 }
    day.amount = roundMoney(day.amount + Number(order.pay_amount || 0))
    day.count += 1
    dailyMap.set(date, day)
  }
  const dailySeries = Array.from({ length: days }, (_, index) => {
    const date = new Date(since.getTime() + (index + 1) * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    return dailyMap.get(date) || { date, amount: 0, count: 0 }
  })
  return {
    today_amount: todayAmount,
    total_amount: totalAmount,
    today_count: todayOrders.length,
    total_count: paidOrders.length,
    avg_amount: paidOrders.length ? roundMoney(totalAmount / paidOrders.length) : 0,
    pending_orders: mockOrders.filter((order) => order.status === 'PENDING').length,
    daily_series: dailySeries,
    payment_methods: Array.from(methodMap.values()),
    top_users: Array.from(userMap.values()).sort((a, b) => b.amount - a.amount).slice(0, 10),
  }
}

function cancelMockPaymentOrder(order) {
  if (!order) return false
  if (order.status !== 'PENDING') return false
  order.status = 'CANCELLED'
  order.updated_at = now()
  return true
}

function activeSubscriptions() {
  const currentTime = Date.now()
  return mockSubscriptions.filter((subscription) => {
    if (subscription.status !== 'active') return false
    if (!subscription.expires_at) return true
    return new Date(subscription.expires_at).getTime() > currentTime
  })
}

function subscriptionDaysRemaining(expiresAt) {
  if (!expiresAt) return 0
  const expiresAtMs = new Date(expiresAt).getTime()
  if (!Number.isFinite(expiresAtMs)) return 0
  return Math.max(0, Math.floor((expiresAtMs - Date.now()) / (24 * 60 * 60 * 1000)))
}

function subscriptionProgressPercentage(used, limit) {
  const amount = normalizePositiveLimit(limit)
  if (amount === null) return null
  const percentage = (Number(used || 0) / amount) * 100
  return Math.max(0, Math.min(100, percentage))
}

function subscriptionWindowProgress(limit, used, windowStart, durationMs) {
  const amount = normalizePositiveLimit(limit)
  if (amount === null || !windowStart) return null
  const windowStartMs = new Date(windowStart).getTime()
  if (!Number.isFinite(windowStartMs)) return null
  const resetsAtMs = windowStartMs + durationMs
  const usedAmount = Number(used || 0)
  return {
    limit_usd: amount,
    used_usd: usedAmount,
    remaining_usd: Math.max(0, amount - usedAmount),
    percentage: Math.max(0, Math.min(100, (usedAmount / amount) * 100)),
    window_start: new Date(windowStartMs).toISOString(),
    resets_at: new Date(resetsAtMs).toISOString(),
    resets_in_seconds: Math.max(0, Math.floor((resetsAtMs - Date.now()) / 1000)),
  }
}

function subscriptionProgress(subscription) {
  return {
    id: subscription.id,
    group_name: subscription.plan_name || subscription.group?.name || '',
    expires_at: subscription.expires_at,
    expires_in_days: subscriptionDaysRemaining(subscription.expires_at),
    daily: subscriptionWindowProgress(
      subscription.daily_limit_usd ?? subscription.group?.daily_limit_usd,
      subscription.daily_usage_usd,
      subscription.daily_window_start,
      24 * 60 * 60 * 1000
    ),
    weekly: subscriptionWindowProgress(
      subscription.weekly_limit_usd ?? subscription.group?.weekly_limit_usd,
      subscription.weekly_usage_usd,
      subscription.weekly_window_start,
      7 * 24 * 60 * 60 * 1000
    ),
    monthly: subscriptionWindowProgress(
      subscription.monthly_limit_usd ?? subscription.quota_usd ?? subscription.group?.monthly_limit_usd,
      subscription.monthly_usage_usd,
      subscription.monthly_window_start,
      30 * 24 * 60 * 60 * 1000
    ),
  }
}

function subscriptionSummaryItem(subscription) {
  return {
    id: subscription.id,
    group_name: subscription.plan_name || subscription.group?.name || '',
    status: subscription.status,
    daily_progress: subscriptionProgressPercentage(
      subscription.daily_usage_usd,
      subscription.daily_limit_usd ?? subscription.group?.daily_limit_usd
    ),
    weekly_progress: subscriptionProgressPercentage(
      subscription.weekly_usage_usd,
      subscription.weekly_limit_usd ?? subscription.group?.weekly_limit_usd
    ),
    monthly_progress: subscriptionProgressPercentage(
      subscription.monthly_usage_usd,
      subscription.monthly_limit_usd ?? subscription.quota_usd ?? subscription.group?.monthly_limit_usd
    ),
    expires_at: subscription.expires_at || null,
    days_remaining: subscriptionDaysRemaining(subscription.expires_at),
  }
}

function sortedPlans(forSaleOnly = false) {
  return mockPlans
    .filter((plan) => !forSaleOnly || plan.for_sale)
    .sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id))
    .map(decoratePlan)
}

function currentCheckoutInfo() {
  return {
    methods: clone(methodLimits),
    global_min: 1,
    global_max: 99999,
    plans: sortedPlans(true),
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    recharge_fee_rate: 0,
    help_text: 'Mock checkout for subscription package preview.',
    help_image_url: '',
    stripe_publishable_key: '',
    alipay_force_qrcode: false,
  }
}

function buildMockRedeemCode(body = {}) {
  const type = String(body.type || 'balance')
  const rawPlan = body.plan_id != null ? mockPlans.find((item) => item.id === Number(body.plan_id)) : null
  const plan = rawPlan ? decoratePlan(rawPlan) : null
  const groupID = plan?.group_id ?? (body.group_id != null ? Number(body.group_id) : null)
  const group = groupID ? findGroup(groupID) : null
  const expiresInDays = Number(body.expires_in_days || 0)

  return {
    id: nextRedeemCodeId++,
    code: `MOCK-${Math.random().toString(36).slice(2, 10).toUpperCase()}`,
    type,
    value: type === 'subscription' ? 0 : Number(body.value || 0),
    status: 'unused',
    used_by: null,
    used_at: null,
    created_at: now(),
    expires_at: expiresInDays > 0 ? daysFromNow(expiresInDays) : null,
    notes: '',
    group_id: groupID,
    plan_id: plan?.id ?? null,
    validity_days: Number(body.validity_days || plan?.validity_days || 30),
    group: group ? clone(group) : null,
  }
}

function findMockUser(userID) {
  const id = Number(userID)
  if (id === adminUser.id) return adminUser
  if (id === regularUser.id) return regularUser
  return null
}

function userShallow(user) {
  if (!user) return null
  return {
    id: user.id,
    email: user.email,
    username: user.username,
    role: user.role,
    balance: user.balance,
    concurrency: user.concurrency,
    status: user.status,
    allowed_groups: user.allowed_groups,
    created_at: user.created_at,
    updated_at: user.updated_at,
  }
}

function normalizeMockCode(value) {
  return String(value || '').trim().toUpperCase()
}

function findRedeemCodeByID(id) {
  return mockRedeemCodes.find((code) => code.id === Number(id)) || null
}

function findRedeemCodeByCode(rawCode) {
  const code = normalizeMockCode(rawCode)
  return mockRedeemCodes.find((item) => normalizeMockCode(item.code) === code) || null
}

function effectiveRedeemStatus(code, currentTime = Date.now()) {
  if (!code) return ''
  if (code.status === 'unused' && code.expires_at && Date.parse(code.expires_at) <= currentTime) {
    return 'expired'
  }
  return code.status
}

function filterRedeemCodes(url) {
  const type = String(url.searchParams.get('type') || '').trim()
  const status = String(url.searchParams.get('status') || '').trim()
  const search = String(url.searchParams.get('search') || '').trim().toLowerCase()
  let items = mockRedeemCodes.slice()
  if (type) {
    items = items.filter((code) => code.type === type)
  }
  if (status) {
    if (status === 'active') {
      items = items.filter((code) => effectiveRedeemStatus(code) === 'unused')
    } else {
      items = items.filter((code) => effectiveRedeemStatus(code) === status)
    }
  }
  if (search) {
    items = items.filter((code) => {
      const user = code.used_by ? findMockUser(code.used_by) : null
      return [code.code, code.type, code.notes, user?.email, user?.username]
        .some((value) => String(value || '').toLowerCase().includes(search))
    })
  }
  return sortRedeemCodes(url, items)
}

function sortRedeemCodes(url, items) {
  const sortBy = String(url.searchParams.get('sort_by') || 'id').trim()
  const desc = String(url.searchParams.get('sort_order') || 'desc') !== 'asc'
  return items.slice().sort((a, b) => {
    const av = redeemSortValue(a, sortBy)
    const bv = redeemSortValue(b, sortBy)
    let delta
    if (typeof av === 'number' && typeof bv === 'number') {
      delta = av - bv
    } else {
      delta = String(av || '').localeCompare(String(bv || ''))
    }
    if (delta === 0) delta = a.id - b.id
    return desc ? -delta : delta
  })
}

function redeemSortValue(code, sortBy) {
  switch (sortBy) {
    case 'code':
    case 'type':
    case 'status':
      return String(code[sortBy] || '')
    case 'value':
      return Number(code.value || 0)
    case 'created_at':
      return Date.parse(code.created_at || '') || 0
    case 'expires_at':
      return code.expires_at ? Date.parse(code.expires_at) || 0 : 0
    default:
      return Number(code.id || 0)
  }
}

function redeemCodeForResponse(code) {
  if (!code) return null
  const copy = clone(code)
  copy.status = effectiveRedeemStatus(copy)
  if (copy.used_by) {
    copy.user = userShallow(findMockUser(copy.used_by))
  }
  return copy
}

function redeemStats() {
  const stats = {
    total_codes: 0,
    active_codes: 0,
    unused_codes: 0,
    used_codes: 0,
    expired_codes: 0,
    disabled_codes: 0,
    total_value_distributed: 0,
    total_value: 0,
    by_type: {
      balance: 0,
      concurrency: 0,
      subscription: 0,
      invitation: 0,
    },
  }
  for (const code of mockRedeemCodes) {
    const status = effectiveRedeemStatus(code)
    stats.total_codes += 1
    stats.total_value = roundMoney(stats.total_value + Number(code.value || 0))
    if (code.type) {
      stats.by_type[code.type] = (stats.by_type[code.type] || 0) + 1
    }
    if (status === 'used') {
      stats.used_codes += 1
      if (Number(code.value || 0) > 0) {
        stats.total_value_distributed = roundMoney(stats.total_value_distributed + Number(code.value || 0))
      }
    } else if (status === 'disabled') {
      stats.disabled_codes += 1
    } else if (status === 'expired') {
      stats.expired_codes += 1
    } else {
      stats.unused_codes += 1
      stats.active_codes += 1
    }
  }
  return stats
}

function redeemCsvPayload(items) {
  const columns = ['id', 'code', 'type', 'value', 'status', 'used_by', 'used_by_email', 'used_at', 'expires_at', 'created_at']
  const rows = [
    columns.join(','),
    ...items.map((code) => {
      const user = code.used_by ? findMockUser(code.used_by) : null
      const row = {
        ...redeemCodeForResponse(code),
        used_by_email: user?.email || '',
      }
      return columns
        .map((column) => `"${String(row[column] ?? '').replace(/"/g, '""')}"`)
        .join(',')
    }),
  ]
  return rows.join('\n')
}

function redeemMockCode(req, rawCode) {
  const code = findRedeemCodeByCode(rawCode)
  if (!code) {
    return { error: { status: 404, payload: { code: 'REDEEM_CODE_NOT_FOUND', message: 'redeem code not found' } } }
  }
  const status = effectiveRedeemStatus(code)
  if (status !== 'unused') {
    return { error: { status: 400, payload: { code: 'REDEEM_CODE_UNAVAILABLE', message: 'redeem code is unavailable' } } }
  }

  const user = currentMockUser(req)
  code.status = 'used'
  code.used_by = user.id
  code.used_at = now()
  code.updated_at = now()
  if (code.type === 'balance') {
    user.balance = roundMoney(Number(user.balance || 0) + Number(code.value || 0))
  } else if (code.type === 'concurrency') {
    user.concurrency = Number(user.concurrency || 0) + Number(code.value || 0)
  } else if (code.type === 'subscription') {
    const subscription =
      code.plan_id != null
        ? createSubscriptionFromPlan(user.id, Number(code.plan_id), Number(code.validity_days || 30))
        : code.group_id != null
          ? createSubscriptionFromGroup(user.id, Number(code.group_id), Number(code.validity_days || 30))
          : null
    if (subscription) {
      user.subscriptions = activeSubscriptions().filter((item) => item.user_id === user.id)
    }
  }
  return { code }
}

function buildMockPromoCode(body = {}) {
  const expiresAt = body.expires_at == null || Number(body.expires_at) === 0
    ? null
    : new Date(Number(body.expires_at) * 1000).toISOString()
  return {
    id: nextPromoCodeId++,
    code: normalizeMockCode(body.code) || `PROMO${nextPromoCodeId}`,
    bonus_amount: Number(body.bonus_amount || 0),
    max_uses: Math.max(0, Number(body.max_uses || 0)),
    used_count: 0,
    status: 'active',
    expires_at: expiresAt,
    notes: body.notes == null ? '' : String(body.notes),
    created_at: now(),
    updated_at: now(),
  }
}

function findPromoCodeByID(id) {
  return mockPromoCodes.find((code) => code.id === Number(id)) || null
}

function findPromoCodeByCode(rawCode) {
  const code = normalizeMockCode(rawCode)
  return mockPromoCodes.find((item) => normalizeMockCode(item.code) === code) || null
}

function filterPromoCodes(url) {
  const status = String(url.searchParams.get('status') || '').trim()
  const search = String(url.searchParams.get('search') || '').trim().toLowerCase()
  let items = mockPromoCodes.slice()
  if (status) {
    items = items.filter((code) => code.status === status)
  }
  if (search) {
    items = items.filter((code) => String(code.code || '').toLowerCase().includes(search))
  }
  return sortPromoCodes(url, items)
}

function sortPromoCodes(url, items) {
  const sortBy = String(url.searchParams.get('sort_by') || 'created_at').trim()
  const desc = String(url.searchParams.get('sort_order') || 'desc') !== 'asc'
  return items.slice().sort((a, b) => {
    const av = promoSortValue(a, sortBy)
    const bv = promoSortValue(b, sortBy)
    let delta
    if (typeof av === 'number' && typeof bv === 'number') {
      delta = av - bv
    } else {
      delta = String(av || '').localeCompare(String(bv || ''))
    }
    if (delta === 0) delta = a.id - b.id
    return desc ? -delta : delta
  })
}

function promoSortValue(code, sortBy) {
  switch (sortBy) {
    case 'code':
    case 'status':
      return String(code[sortBy] || '')
    case 'bonus_amount':
      return Number(code.bonus_amount || 0)
    case 'expires_at':
      return code.expires_at ? Date.parse(code.expires_at) || 0 : 0
    case 'created_at':
      return Date.parse(code.created_at || '') || 0
    default:
      return Number(code.id || 0)
  }
}

function updateMockPromoCode(code, body = {}) {
  if (body.code !== undefined) {
    code.code = normalizeMockCode(body.code)
  }
  if (body.bonus_amount !== undefined) {
    code.bonus_amount = Number(body.bonus_amount || 0)
  }
  if (body.max_uses !== undefined) {
    code.max_uses = Math.max(0, Number(body.max_uses || 0))
  }
  if (body.status !== undefined) {
    code.status = String(body.status || 'active')
  }
  if (body.expires_at !== undefined) {
    code.expires_at = Number(body.expires_at) === 0 || body.expires_at === null
      ? null
      : new Date(Number(body.expires_at) * 1000).toISOString()
  }
  if (body.notes !== undefined) {
    code.notes = body.notes == null ? '' : String(body.notes)
  }
  code.updated_at = now()
  return code
}

function promoValidationResponse(rawCode) {
  if (!publicSettings.promo_code_enabled) {
    return { valid: false, error_code: 'PROMO_CODE_DISABLED' }
  }
  const promo = findPromoCodeByCode(rawCode)
  if (!promo) {
    return { valid: false, error_code: 'PROMO_CODE_NOT_FOUND' }
  }
  if (promo.status !== 'active') {
    return { valid: false, error_code: 'PROMO_CODE_DISABLED' }
  }
  if (promo.expires_at && Date.parse(promo.expires_at) <= Date.now()) {
    return { valid: false, error_code: 'PROMO_CODE_EXPIRED' }
  }
  if (promo.max_uses > 0 && promo.used_count >= promo.max_uses) {
    return { valid: false, error_code: 'PROMO_CODE_MAX_USED' }
  }
  return { valid: true, bonus_amount: promo.bonus_amount }
}

function promoUsageForResponse(usage) {
  if (!usage) return null
  return {
    ...clone(usage),
    user: userShallow(findMockUser(usage.user_id)),
  }
}

function ensureAffiliateProfile(userID) {
  const id = Number(userID)
  let profile = mockAffiliateProfiles.find((item) => item.user_id === id)
  if (profile) return profile
  profile = {
    user_id: id,
    aff_code: `AFF${String(id).padStart(6, '0')}`,
    aff_code_custom: false,
    aff_rebate_rate_percent: null,
    inviter_id: null,
    aff_count: 0,
    aff_quota: 0,
    aff_frozen_quota: 0,
    aff_history_quota: 0,
    created_at: now(),
    updated_at: now(),
  }
  mockAffiliateProfiles.push(profile)
  return profile
}

function globalAffiliateRate() {
  return Number(adminSettings.affiliate_rebate_rate || 0)
}

function affiliateRate(profile) {
  return profile && profile.aff_rebate_rate_percent != null
    ? Number(profile.aff_rebate_rate_percent)
    : globalAffiliateRate()
}

function affiliateInvitees(inviterID) {
  return mockAffiliateProfiles
    .filter((profile) => profile.inviter_id === Number(inviterID))
    .map((profile) => {
      const user = findMockUser(profile.user_id)
      return {
        user_id: profile.user_id,
        email: user ? maskEmailForMock(user.email) : '',
        username: user?.username || '',
        created_at: profile.created_at,
        total_rebate: profile.aff_history_quota,
      }
    })
}

function maskEmailForMock(email) {
  const value = String(email || '').trim()
  const at = value.indexOf('@')
  if (at <= 0) return value ? `${value[0]}***` : ''
  const local = value.slice(0, at)
  const domain = value.slice(at + 1)
  const dot = domain.lastIndexOf('.')
  const maskedDomain = dot > 0 ? `${domain[0]}***${domain.slice(dot)}` : `${domain[0] || ''}***`
  return `${local[0] || ''}***@${maskedDomain}`
}

function affiliateDetailForUser(userID) {
  const profile = ensureAffiliateProfile(userID)
  return {
    user_id: profile.user_id,
    aff_code: profile.aff_code,
    inviter_id: profile.inviter_id,
    aff_count: affiliateInvitees(profile.user_id).length,
    aff_quota: profile.aff_quota,
    aff_frozen_quota: profile.aff_frozen_quota,
    aff_history_quota: profile.aff_history_quota,
    effective_rebate_rate_percent: affiliateRate(profile),
    invitees: affiliateInvitees(profile.user_id),
  }
}

function affiliateOverviewForUser(userID) {
  const profile = ensureAffiliateProfile(userID)
  const user = findMockUser(profile.user_id) || { email: '', username: '' }
  const invitees = affiliateInvitees(profile.user_id)
  return {
    user_id: profile.user_id,
    email: user.email,
    username: user.username,
    aff_code: profile.aff_code,
    rebate_rate_percent: affiliateRate(profile),
    invited_count: invitees.length,
    rebated_invitee_count: invitees.filter((item) => Number(item.total_rebate || 0) > 0).length,
    available_quota: profile.aff_quota,
    history_quota: profile.aff_history_quota,
  }
}

function affiliateAdminEntry(profile) {
  const user = findMockUser(profile.user_id) || { email: '', username: '' }
  return {
    user_id: profile.user_id,
    email: user.email,
    username: user.username,
    aff_code: profile.aff_code,
    aff_code_custom: profile.aff_code_custom === true,
    aff_rebate_rate_percent: profile.aff_rebate_rate_percent,
    aff_count: affiliateInvitees(profile.user_id).length,
  }
}

function filterAffiliateAdminEntries(url) {
  const search = String(url.searchParams.get('search') || '').trim().toLowerCase()
  let entries = mockAffiliateProfiles
    .filter((profile) => profile.aff_code_custom === true || profile.aff_rebate_rate_percent != null)
    .map(affiliateAdminEntry)
  if (search) {
    entries = entries.filter((entry) =>
      [entry.email, entry.username, entry.aff_code]
        .some((value) => String(value || '').toLowerCase().includes(search))
    )
  }
  return entries.sort((a, b) => a.user_id - b.user_id)
}

function affiliateInviteRecords() {
  return mockAffiliateProfiles
    .filter((profile) => profile.inviter_id)
    .map((profile) => {
      const inviterProfile = ensureAffiliateProfile(profile.inviter_id)
      const inviter = findMockUser(inviterProfile.user_id) || { email: '', username: '' }
      const invitee = findMockUser(profile.user_id) || { email: '', username: '' }
      return {
        inviter_id: inviterProfile.user_id,
        inviter_email: inviter.email,
        inviter_username: inviter.username,
        invitee_id: profile.user_id,
        invitee_email: invitee.email,
        invitee_username: invitee.username,
        aff_code: inviterProfile.aff_code,
        total_rebate: profile.aff_history_quota,
        created_at: profile.created_at,
      }
    })
}

function affiliateRebateRecords() {
  return mockAffiliateProfiles
    .filter((profile) => profile.inviter_id && profile.aff_history_quota > 0)
    .map((profile, index) => {
      const inviterProfile = ensureAffiliateProfile(profile.inviter_id)
      const inviter = findMockUser(inviterProfile.user_id) || { email: '', username: '' }
      const invitee = findMockUser(profile.user_id) || { email: '', username: '' }
      return {
        order_id: 90000 + index,
        out_trade_no: `MOCK-AFF-${profile.user_id}`,
        inviter_id: inviterProfile.user_id,
        inviter_email: inviter.email,
        inviter_username: inviter.username,
        invitee_id: profile.user_id,
        invitee_email: invitee.email,
        invitee_username: invitee.username,
        order_amount: 35,
        pay_amount: 35,
        currency: 'CNY',
        rebate_amount: profile.aff_history_quota,
        payment_type: 'alipay',
        order_status: 'COMPLETED',
        created_at: profile.created_at,
      }
    })
}

function filterAffiliateRecords(url, items) {
  const search = String(url.searchParams.get('search') || '').trim().toLowerCase()
  let records = items.slice()
  if (search) {
    records = records.filter((record) => Object.values(record)
      .some((value) => String(value || '').toLowerCase().includes(search)))
  }
  const startAt = Date.parse(String(url.searchParams.get('start_at') || ''))
  if (Number.isFinite(startAt)) {
    records = records.filter((record) => Date.parse(record.created_at || '') >= startAt)
  }
  const endRaw = String(url.searchParams.get('end_at') || '')
  const endAt = Date.parse(endRaw)
  if (Number.isFinite(endAt)) {
    const endBoundary = /^\d{4}-\d{2}-\d{2}$/.test(endRaw) ? endAt + 24 * 60 * 60 * 1000 - 1 : endAt
    records = records.filter((record) => Date.parse(record.created_at || '') <= endBoundary)
  }
  const desc = String(url.searchParams.get('sort_order') || 'desc') !== 'asc'
  return records.sort((a, b) => {
    const delta = (Date.parse(a.created_at || '') || 0) - (Date.parse(b.created_at || '') || 0)
    return desc ? -delta : delta
  })
}

async function readJson(req) {
  const raw = (await readRequestBuffer(req)).toString('utf8')
  if (!raw.trim()) return {}
  try {
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

async function readRequestBuffer(req) {
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  return Buffer.concat(chunks)
}

async function readMultipartFile(req) {
  const contentType = String(req.headers['content-type'] || '')
  const boundaryMatch = /boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType)
  const boundaryText = boundaryMatch?.[1] || boundaryMatch?.[2]
  if (!boundaryText) return null

  const body = await readRequestBuffer(req)
  const boundary = Buffer.from(`--${boundaryText}`)
  let offset = body.indexOf(boundary)
  while (offset >= 0) {
    const partStart = offset + boundary.length
    if (body.slice(partStart, partStart + 2).toString('utf8') === '--') return null

    const headerStart = body.slice(partStart, partStart + 2).toString('utf8') === '\r\n'
      ? partStart + 2
      : partStart
    const headerEnd = body.indexOf(Buffer.from('\r\n\r\n'), headerStart)
    if (headerEnd < 0) return null

    const headers = body.slice(headerStart, headerEnd).toString('utf8')
    const dataStart = headerEnd + 4
    const nextBoundary = body.indexOf(boundary, dataStart)
    if (nextBoundary < 0) return null

    const dataEnd = body.slice(nextBoundary - 2, nextBoundary).toString('utf8') === '\r\n'
      ? nextBoundary - 2
      : nextBoundary
    const disposition = /content-disposition:\s*([^\r\n]+)/i.exec(headers)?.[1] || ''
    const name = /name="([^"]+)"/i.exec(disposition)?.[1] || ''
    const filename = /filename="([^"]*)"/i.exec(disposition)?.[1] || ''
    if (name === 'file') {
      const partContentType = /content-type:\s*([^\r\n]+)/i.exec(headers)?.[1] || ''
      return {
        filename,
        contentType: partContentType.trim(),
        data: body.slice(dataStart, dataEnd),
      }
    }
    offset = body.indexOf(boundary, nextBoundary + boundary.length)
  }
  return null
}

function loadMockXlsx() {
  if (!mockXlsx) {
    try {
      mockXlsx = require('xlsx')
    } catch {
      mockXlsx = require('../frontend/node_modules/xlsx')
    }
  }
  return mockXlsx
}

function parseMockCSVRows(text) {
  const rows = []
  let row = []
  let value = ''
  let quoted = false

  for (let i = 0; i < text.length; i += 1) {
    const char = text[i]
    if (quoted) {
      if (char === '"') {
        if (text[i + 1] === '"') {
          value += '"'
          i += 1
        } else {
          quoted = false
        }
      } else {
        value += char
      }
      continue
    }
    if (char === '"') {
      quoted = true
    } else if (char === ',') {
      row.push(value)
      value = ''
    } else if (char === '\n') {
      row.push(value)
      rows.push(row)
      row = []
      value = ''
    } else if (char !== '\r') {
      value += char
    }
  }
  row.push(value)
  if (row.some((cell) => String(cell || '').trim() !== '')) rows.push(row)
  return rows
}

const mockImportHeaderAliases = {
  name: ['name', 'username', 'screen_name', 'account', '账号', '用户名'],
  password: ['password', '密码'],
  phone: ['phone', 'mobile', '手机号', '手机'],
  email: ['email', 'mail', '邮箱', '邮箱账号'],
  email_password: ['email_password', 'emailpassword', 'mail_password', 'mailpassword', '邮箱密码'],
  two_factor: ['two_factor', 'twofactor', '2fa', 'totp', 'otp', '两步验证', '二次验证码'],
  backup_code: ['backup_code', 'backupcode', 'backup', '备份码'],
  email_client_id: ['email_client_id', 'emailclientid', 'client_id', 'clientid', '邮箱客户端id', '客户端id'],
  email_token: ['email_token', 'emailtoken', 'mail_token', 'mailtoken', 'token', '邮箱令牌', '邮箱token', '邮箱授权码'],
  registration_ip: ['registration_ip', 'registrationip', 'register_ip', 'registerip', 'bound_ip', 'boundip', '注册ip', '注册IP'],
  auth_cookie: ['auth_cookie', 'authcookie', 'cookie', 'Cookie', '认证cookie', '认证Cookie'],
  execution_auth: ['execution_auth', 'executionauth', '执行凭证'],
  default_proxy_snapshot: ['default_proxy_snapshot', 'defaultproxysnapshot', '默认代理快照'],
  remark: ['remark', 'note', '备注'],
}

const mockImportAliasToField = Object.entries(mockImportHeaderAliases).reduce((acc, [field, aliases]) => {
  for (const alias of aliases) acc[normalizeMockImportHeader(alias)] = field
  return acc
}, {})

function normalizeMockImportHeader(value) {
  return String(value || '').trim().toLowerCase().replace(/[\s_-]+/g, '')
}

function mockImportString(value) {
  return String(value ?? '').trim()
}

function mockImportPointerValue(value) {
  const text = mockImportString(value)
  return text === '' ? undefined : text
}

function mockAdminImportRowToAccount(row, headerIndex, hasHeader) {
  const fallback = {
    name: 0,
    password: 1,
    phone: 2,
    email: 3,
    email_password: 4,
    auth_cookie: 5,
    execution_auth: 6,
    default_proxy_snapshot: 7,
    remark: 8,
  }
  const value = (field) => {
    let index = hasHeader ? headerIndex[field] : fallback[field]
    if (index == null || index < 0 || index >= row.length) return ''
    return mockImportString(row[index])
  }
  return {
    name: value('name'),
    platform: 'x_twitter',
    password: mockImportPointerValue(value('password')),
    phone: mockImportPointerValue(value('phone')),
    email: mockImportPointerValue(value('email')),
    email_password: mockImportPointerValue(value('email_password')),
    two_factor: mockImportPointerValue(value('two_factor')),
    backup_code: mockImportPointerValue(value('backup_code')),
    email_client_id: mockImportPointerValue(value('email_client_id')),
    email_token: mockImportPointerValue(value('email_token')),
    registration_ip: mockImportPointerValue(value('registration_ip')),
    auth_cookie: mockImportPointerValue(value('auth_cookie')),
    execution_auth: mockImportPointerValue(value('execution_auth')),
    default_proxy_snapshot: mockImportPointerValue(value('default_proxy_snapshot')),
    remark: mockImportPointerValue(value('remark')),
    account_status: 'pending_check',
    task_status: 'stored',
    task_message: 'Stored in mock account pool',
    assigned_user_id: null,
  }
}

function mockAdminImportObjectToAccount(record = {}) {
  const normalized = {}
  for (const [key, rawValue] of Object.entries(record || {})) {
    const field = mockImportAliasToField[normalizeMockImportHeader(key)]
    if (field) normalized[field] = rawValue
  }
  return {
    name: mockImportString(normalized.name),
    platform: 'x_twitter',
    password: mockImportPointerValue(normalized.password),
    phone: mockImportPointerValue(normalized.phone),
    email: mockImportPointerValue(normalized.email),
    email_password: mockImportPointerValue(normalized.email_password),
    two_factor: mockImportPointerValue(normalized.two_factor),
    backup_code: mockImportPointerValue(normalized.backup_code),
    email_client_id: mockImportPointerValue(normalized.email_client_id),
    email_token: mockImportPointerValue(normalized.email_token),
    registration_ip: mockImportPointerValue(normalized.registration_ip),
    auth_cookie: mockImportPointerValue(normalized.auth_cookie),
    execution_auth: mockImportPointerValue(normalized.execution_auth),
    default_proxy_snapshot: mockImportPointerValue(normalized.default_proxy_snapshot),
    remark: mockImportPointerValue(normalized.remark),
    account_status: 'pending_check',
    task_status: 'stored',
    task_message: 'Stored in mock account pool',
    assigned_user_id: null,
  }
}

function mockAdminImportRowsToAccounts(rows) {
  const cleanedRows = rows
    .map((row) => Array.isArray(row) ? row.map((cell) => mockImportString(cell)) : [])
    .filter((row) => row.some((cell) => cell !== ''))
  if (cleanedRows.length === 0) return []

  const headerIndex = {}
  for (let i = 0; i < cleanedRows[0].length; i += 1) {
    const field = mockImportAliasToField[normalizeMockImportHeader(cleanedRows[0][i])]
    if (field && headerIndex[field] == null) headerIndex[field] = i
  }
  const hasHeader = Object.keys(headerIndex).length > 0
  const records = hasHeader ? cleanedRows.slice(1) : cleanedRows
  return records.map((row) => mockAdminImportRowToAccount(row, headerIndex, hasHeader))
}

function parseMockAdminImportFile(file) {
  const filename = String(file?.filename || '').toLowerCase()
  const contentType = String(file?.contentType || '').toLowerCase()
  if (filename.endsWith('.json') || contentType.includes('json')) {
    const parsed = JSON.parse(file.data.toString('utf8') || '[]')
    return Array.isArray(parsed) ? parsed.map(mockAdminImportObjectToAccount) : []
  }
  if (filename.endsWith('.xls')) {
    throw new Error('legacy .xls social account imports are not supported; please use .xlsx, .csv, or .json')
  }
  if (filename.endsWith('.xlsx') || contentType.includes('spreadsheet')) {
    const xlsx = loadMockXlsx()
    const workbook = xlsx.read(file.data, { type: 'buffer' })
    const worksheet = workbook.Sheets[workbook.SheetNames[0]]
    if (!worksheet) return []
    const rows = xlsx.utils.sheet_to_json(worksheet, { header: 1, defval: '', raw: false })
    return mockAdminImportRowsToAccounts(rows)
  }
  return mockAdminImportRowsToAccounts(parseMockCSVRows(file.data.toString('utf8')))
}

function mockSocialAccountImportResult(total = 0) {
  return {
    total,
    succeeded: 0,
    created: 0,
    skipped: 0,
    failed: 0,
    duplicates: 0,
    errors: [],
    items: [],
    accounts: [],
  }
}

function importMockTotalPoolAccounts(inputs) {
  const result = mockSocialAccountImportResult(inputs.length)
  const seen = new Set()

  for (const input of inputs) {
    const name = String(input?.name || '').trim()
    const platform = normalizeMockSocialPlatform(input?.platform || 'x_twitter')
    if (!name || !platform) {
      result.skipped += 1
      result.failed += 1
      result.errors.push('missing platform or name')
      result.items.push({ name, status: 'failed', reason: 'invalid_input', error: 'missing platform or name' })
      continue
    }

    const accountInput = { ...input, name, platform }
    const key = mockSocialAccountDedupKey(accountInput)
    if (key && seen.has(key)) {
      result.skipped += 1
      result.duplicates += 1
      const message = 'duplicate account in import batch'
      result.errors.push(message)
      result.items.push({ name, status: 'duplicate', reason: 'duplicate_in_batch', error: message })
      continue
    }
    if (key) seen.add(key)

    const duplicate = key && mockSocialAccounts.some((account) => mockSocialAccountDedupKey(account) === key && !isWorkbenchStagingAccount(account))
    if (duplicate) {
      result.skipped += 1
      result.duplicates += 1
      result.items.push({ name, status: 'duplicate', reason: 'duplicate_in_database', error: 'duplicate account in total pool' })
      continue
    }

    const account = createMockSocialAccount({
      ...accountInput,
      account_status: 'pending_check',
      task_status: 'stored',
      task_message: 'Stored in mock account pool',
      assigned_user_id: null,
    })
    result.succeeded += 1
    result.created += 1
    result.items.push({ id: account.id, name: account.name, status: 'succeeded' })
    result.accounts.push(clone(account))
  }
  return result
}

function publicBackupS3Config() {
  return {
    ...clone(mockBackupS3Config),
    secret_access_key: '',
  }
}

function isBackupS3Configured(config = mockBackupS3Config) {
  return Boolean(
    String(config.bucket || '').trim() &&
    String(config.access_key_id || '').trim() &&
    String(config.secret_access_key || '').trim()
  )
}

function backupRecordForResponse(record) {
  if (!record) return null
  const copy = clone(record)
  delete copy.__restore_started_at
  return copy
}

function refreshMockBackupRecord(record) {
  if (!record) return null
  if (record.status === 'running') {
    const startedAt = new Date(record.started_at).getTime()
    if (Number.isFinite(startedAt) && Date.now() - startedAt > 1200) {
      record.status = 'completed'
      record.progress = ''
      record.finished_at = now()
      record.size_bytes = record.size_bytes || 1024 * 1024 * 6
    } else if (!record.progress) {
      record.progress = 'uploading'
    }
  }
  if (record.restore_status === 'running') {
    const restoreStartedAt = new Date(record.__restore_started_at || record.started_at).getTime()
    if (Number.isFinite(restoreStartedAt) && Date.now() - restoreStartedAt > 1200) {
      record.restore_status = 'completed'
      record.restore_error = ''
      record.restored_at = now()
      delete record.__restore_started_at
    }
  }
  return record
}

function refreshMockBackupRecords() {
  mockBackupRecords = mockBackupRecords.map((record) => refreshMockBackupRecord(record))
}

function findMockBackupRecord(id) {
  const record = mockBackupRecords.find((item) => item.id === id)
  return refreshMockBackupRecord(record)
}

function buildMockBackupRecord(expireDays = 14) {
  const createdAt = new Date()
  const safeTimestamp = createdAt.toISOString().replace(/[:.]/g, '-')
  const fileName = `socialops-backup-${safeTimestamp}.sql.gz`
  const prefix = String(mockBackupS3Config.prefix || '').replace(/^\/+/, '')
  return {
    id: `backup-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    status: 'running',
    backup_type: 'postgres',
    file_name: fileName,
    s3_key: `${prefix}${fileName}`,
    size_bytes: 0,
    triggered_by: 'manual',
    error_message: '',
    started_at: createdAt.toISOString(),
    finished_at: '',
    expires_at: expireDays > 0 ? daysFromNow(expireDays) : '',
    progress: 'dumping',
    restore_status: '',
    restore_error: '',
    restored_at: '',
  }
}

function createSubscriptionFromGroup(userId, groupId, validityDays = 30) {
  const group = findGroup(groupId)
  if (!group) return null
  const plan = decoratePlan(firstPlanForGroup(groupId) || {
    id: null,
    group_id: group.id,
    platform: group.platform,
    name: group.name,
    description: group.description || '',
    price: 0,
    validity_days: validityDays,
    validity_unit: 'days',
    daily_limit_usd: group.daily_limit_usd,
    weekly_limit_usd: group.weekly_limit_usd,
    monthly_limit_usd: group.monthly_limit_usd,
    features: [],
    product_name: group.name,
    for_sale: false,
    sort_order: 999,
  })

  const startsAt = now()
  const expiresAt = daysFromNow(validityDays)
  const subscription = buildSubscription({
    id: nextSubscriptionId++,
    userId,
    group,
    plan,
    status: 'active',
    startsAt,
    expiresAt,
    dailyUsage: 0,
    weeklyUsage: 0,
    monthlyUsage: 0,
  })
  mockSubscriptions.unshift(subscription)
  adminUser.subscriptions = activeSubscriptions()
  return subscription
}

function createSubscriptionFromPlan(userId, planId, validityDays = 30) {
  const rawPlan = mockPlans.find((item) => item.id === Number(planId))
  if (!rawPlan) return null
  const plan = decoratePlan(rawPlan)
  const group = findGroup(plan.group_id)
  if (!group) return null
  const startsAt = now()
  const expiresAt = daysFromNow(validityDays)
  const subscription = buildSubscription({
    id: nextSubscriptionId++,
    userId,
    group,
    plan,
    status: 'active',
    startsAt,
    expiresAt,
    dailyUsage: 0,
    weeklyUsage: 0,
    monthlyUsage: 0,
  })
  mockSubscriptions.unshift(subscription)
  adminUser.subscriptions = activeSubscriptions()
  return subscription
}

function findSubscription(id) {
  return mockSubscriptions.find((subscription) => subscription.id === Number(id)) || null
}

function filterSubscriptions(url) {
  let items = [...mockSubscriptions]
  const status = url.searchParams.get('status')
  const userId = Number(url.searchParams.get('user_id') || 0)
  const groupId = Number(url.searchParams.get('group_id') || 0)
  const planId = Number(url.searchParams.get('plan_id') || 0)
  const platform = (url.searchParams.get('platform') || '').trim()
  const sortBy = url.searchParams.get('sort_by') || 'created_at'
  const sortOrder = url.searchParams.get('sort_order') === 'asc' ? 'asc' : 'desc'

  if (status) items = items.filter((subscription) => subscription.status === status)
  if (userId > 0) items = items.filter((subscription) => subscription.user_id === userId)
  if (groupId > 0) items = items.filter((subscription) => subscription.group_id === groupId)
  if (planId > 0) items = items.filter((subscription) => subscription.plan_id === planId)
  if (platform) {
    items = items.filter((subscription) => {
      const value = subscription.plan_platform || subscription.group?.platform || ''
      return value === platform
    })
  }

  items.sort((a, b) => {
    const av = a[sortBy]
    const bv = b[sortBy]
    if (av === bv) return 0
    if (sortOrder === 'asc') return av > bv ? 1 : -1
    return av < bv ? 1 : -1
  })

  return items
}

function userSafeSocialAccount(account) {
  const copy = clone(account)
  copy.default_proxy_configured = String(copy.default_proxy_snapshot || '').trim() !== ''
  delete copy.assigned_user_id
  delete copy.proxy_id
  return copy
}

function mockTaskTemplatesForUser(userId) {
  const key = Number(userId)
  if (!mockTaskTemplatesByUser.has(key)) mockTaskTemplatesByUser.set(key, [])
  return mockTaskTemplatesByUser.get(key)
}

function normalizeMockTaskTemplateValues(values) {
  return Array.isArray(values) ? values.map((item) => String(item).trim()).filter(Boolean) : []
}

function normalizeMockTaskTemplateParamsForType(type, params = {}) {
  const normalized = {
    targets: normalizeMockTaskTemplateValues(params.targets),
    contents: normalizeMockTaskTemplateValues(params.contents),
  }
  if (type === 'login_check') return { targets: [], contents: [] }
  if (['follow', 'like', 'retweet'].includes(type)) return { targets: normalized.targets, contents: [] }
  if (type === 'post') return { targets: [], contents: normalized.contents }
  return normalized
}

function normalizeMockTaskTemplateInput(body = {}) {
  const type = String(body.type || 'login_check').trim()
  const params = body.params && typeof body.params === 'object' ? body.params : {}
  return {
    id: body.id ? String(body.id) : '',
    name: String(body.name || '').trim(),
    type: mockTaskTemplateTypes.includes(type) ? type : '',
    params: normalizeMockTaskTemplateParamsForType(type, params),
    is_default: Boolean(body.is_default),
  }
}

function validateMockTaskTemplateInput(input) {
  const errors = []
  if (!input.name) errors.push('template name is required')
  if (!input.type) errors.push('unsupported task type')
  if (input.params.targets.length > maxMockTaskTemplatePoolValues) {
    errors.push(`target list cannot exceed ${maxMockTaskTemplatePoolValues} items`)
  }
  if (input.params.contents.length > maxMockTaskTemplatePoolValues) {
    errors.push(`content pool cannot exceed ${maxMockTaskTemplatePoolValues} items`)
  }
  if (input.params.targets.some((value) => Array.from(value).length > maxMockTaskTemplateValueLength)) {
    errors.push(`target item cannot exceed ${maxMockTaskTemplateValueLength} characters`)
  }
  if (input.params.contents.some((value) => Array.from(value).length > maxMockTaskTemplateValueLength)) {
    errors.push(`content item cannot exceed ${maxMockTaskTemplateValueLength} characters`)
  }
  if (['follow', 'like', 'retweet'].includes(input.type) && input.params.targets.length === 0) {
    errors.push('target list is required')
  }
  if (input.type === 'post' && input.params.contents.length === 0) {
    errors.push('content pool is required')
  }
  return {
    valid: errors.length === 0,
    type: input.type || '',
    targets: input.params.targets.length,
    contents: input.params.contents.length,
    errors,
  }
}

function saveMockTaskTemplate(userId, body = {}) {
  const input = normalizeMockTaskTemplateInput(body)
  const validation = validateMockTaskTemplateInput(input)
  if (!validation.valid) return { template: null, validation }
  const templates = mockTaskTemplatesForUser(userId)
  const existingIndex = input.id ? templates.findIndex((template) => template.id === input.id) : -1
  const timestamp = now()
  const template = {
    id: existingIndex >= 0 ? templates[existingIndex].id : `tmpl_${nextMockTaskTemplateId++}`,
    name: input.name,
    type: input.type,
    params: input.params,
    is_default: input.is_default,
    created_at: existingIndex >= 0 ? templates[existingIndex].created_at : timestamp,
    updated_at: timestamp,
  }
  if (template.is_default) {
    for (const item of templates) {
      if (item.type === template.type) item.is_default = false
    }
  }
  if (existingIndex >= 0) templates[existingIndex] = template
  else templates.unshift(template)
  return { template, validation }
}

function findMockTaskTemplate(userId, id) {
  return mockTaskTemplatesForUser(userId).find((template) => template.id === String(id))
}

function isCompleteWorkbenchImportAccount(account = {}) {
  const hasName = String(account?.name || '').trim() !== ''
  const hasPassword = String(account?.password || '').trim() !== ''
  const hasTwoFactor = String(account?.two_factor || '').trim() !== ''
  const hasAuthCookie = firstMockString(account?.auth_cookie) !== ''
  const hasEmail = String(account?.email || '').trim() !== '' && (
    String(account?.email_password || '').trim() !== '' ||
    String(account?.email_token || '').trim() !== ''
  )
  return hasName && hasPassword && (hasTwoFactor || hasEmail || hasAuthCookie)
}

function normalizeMockSocialPlatform(value) {
  const normalized = String(value || '').trim().toLowerCase().replace(/[-/\s]+/g, '_')
  if (['twitter', 'x', 'x_twitter', 'twitter_x'].includes(normalized)) return 'x_twitter'
  return normalized
}

function normalizeMockSocialUsername(value) {
  return String(value || '').trim().toLowerCase().replace(/^@+/, '').trim()
}

function mockSocialAccountDedupKey(account = {}) {
  const name = normalizeMockSocialUsername(account.name)
  const platform = normalizeMockSocialPlatform(account.platform)
  if (!platform || !name) return ''
  return `${platform}\u0000username\u0000${name}`
}

function firstMockString(...values) {
  for (const value of values) {
    const trimmed = String(value || '').trim()
    if (trimmed) return trimmed
  }
  return ''
}

function mockSocialAccountBatchResult(ids = []) {
  return {
    total: ids.length,
    succeeded: 0,
    skipped: 0,
    failed: 0,
    duplicates: 0,
    errors: [],
    items: [],
  }
}

function mockSocialAccountBatchSuccess(result, account) {
  result.succeeded += 1
  result.items.push({ id: account.id, name: account.name, status: 'succeeded' })
}

function mockSocialAccountBatchSkip(result, id, name, reason) {
  result.skipped += 1
  result.items.push({ id, name, status: 'skipped', reason, error: 'account could not be processed' })
}

function isWorkbenchStagingAccount(account) {
  return !!account?.assigned_user_id &&
    account.account_status === 'not_stored' &&
    account.task_status === 'pending'
}

function findSocialAccount(id) {
  return mockSocialAccounts.find((account) => account.id === Number(id)) || null
}

function filterSocialAccounts(url, { userOnly = false, userId = adminUser.id, totalPoolOnly = false } = {}) {
  let items = [...mockSocialAccounts]
  if (userOnly) {
    items = items.filter((account) => account.assigned_user_id === userId)
  }
  if (totalPoolOnly) {
    items = items.filter((account) => !isWorkbenchStagingAccount(account))
  }

  const platform = (url.searchParams.get('platform') || '').trim()
  const accountStatus = (url.searchParams.get('account_status') || '').trim()
  const taskStatus = (url.searchParams.get('task_status') || '').trim()
  const unassigned = url.searchParams.get('unassigned') === 'true'

  if (platform) items = items.filter((account) => account.platform === platform)
  if (accountStatus) items = items.filter((account) => account.account_status === accountStatus)
  if (taskStatus) items = items.filter((account) => account.task_status === taskStatus)
  if (unassigned) items = items.filter((account) => !account.assigned_user_id)

  return items.sort((a, b) => b.id - a.id)
}

const mockProxyTypes = new Set(['residential', 'static', 'mobile', 'datacenter'])
const mockProxyStatuses = new Set(['online', 'offline', 'unknown'])

function findMockProxy(id) {
  return mockProxies.find((proxy) => proxy.id === Number(id)) || null
}

function normalizeMockProxyType(raw, { defaultIfEmpty = false } = {}) {
  const value = String(raw || '').trim()
  if (!value && defaultIfEmpty) return 'residential'
  return mockProxyTypes.has(value) ? value : null
}

function normalizeMockProxyEndpoint(raw) {
  const endpoint = raw == null ? '' : String(raw).trim()
  if (!endpoint) return { ok: true, endpoint }
  let parsed
  try {
    parsed = new URL(endpoint)
  } catch {
    return { ok: false, message: 'invalid proxy endpoint URL' }
  }
  const scheme = parsed.protocol.replace(':', '')
  if (!['http', 'https', 'socks5', 'socks5h'].includes(scheme)) {
    return { ok: false, message: `unsupported scheme: ${scheme}` }
  }
  if (!parsed.hostname) {
    return { ok: false, message: 'proxy host is required' }
  }
  if (!parsed.port) {
    return { ok: false, message: 'proxy port is required' }
  }
  const port = Number(parsed.port)
  if (!Number.isInteger(port) || port <= 0 || port > 65535) {
    return { ok: false, message: 'proxy port is invalid' }
  }
  if (isBlockedMockProxyHost(parsed.hostname)) {
    return { ok: false, message: 'proxy host points to a private or local address' }
  }
  return { ok: true, endpoint }
}

function isBlockedMockProxyHost(host) {
  const value = String(host || '').toLowerCase().replace(/^\[|\]$/g, '')
  if (!value || value === 'localhost' || value === '::1') return true
  const octets = value.split('.').map(Number)
  if (octets.length !== 4 || octets.some((part) => !Number.isInteger(part) || part < 0 || part > 255)) {
    return false
  }
  const [first, second] = octets
  return first === 10 ||
    first === 127 ||
    (first === 169 && second === 254) ||
    (first === 172 && second >= 16 && second <= 31) ||
    (first === 192 && second === 168) ||
    first === 0 ||
    first >= 224
}

function filterMockProxies(url, { userId } = {}) {
  let items = [...mockProxies]
  const status = String(url.searchParams.get('status') || '').trim()
  const ipType = String(url.searchParams.get('ip_type') || '').trim()
  const search = String(url.searchParams.get('search') || '').trim().toLowerCase()

  if (userId > 0) items = items.filter((proxy) => proxy.user_id === userId)
  if (status) items = mockProxyStatuses.has(status) ? items.filter((proxy) => proxy.status === status) : []
  if (ipType) items = items.filter((proxy) => proxy.ip_type === ipType)
  if (search) {
    items = items.filter((proxy) => [proxy.name, proxy.endpoint, proxy.remark].some((value) => String(value || '').toLowerCase().includes(search)))
  }

  return items.sort((a, b) => b.id - a.id)
}

function createMockProxy(body = {}, userId = 0) {
  if (userId <= 0) return { error: { code: 'BAD_REQUEST', message: 'user_id is required' } }
  if (Object.prototype.hasOwnProperty.call(body, 'user_id')) {
    return { error: { code: 'SOCIAL_IP_USER_ID_NOT_ACCEPTED', message: 'user_id is not accepted' } }
  }
  const name = String(body.name || '').trim()
  if (!name) return { error: { code: 'BAD_REQUEST', message: 'name is required' } }
  const ipType = normalizeMockProxyType(body.ip_type, { defaultIfEmpty: true })
  if (!ipType) return { error: { code: 'SOCIAL_IP_TYPE_INVALID', message: 'social IP type is invalid' } }
  const endpointResult = normalizeMockProxyEndpoint(body.endpoint)
  if (!endpointResult.ok) return { error: { code: 'INVALID_PROXY_ENDPOINT', message: endpointResult.message } }

  const timestamp = now()
  const proxy = {
    id: nextMockProxyId++,
    user_id: userId,
    name,
    ip_type: ipType,
    endpoint: endpointResult.endpoint,
    status: 'unknown',
    latency_ms: null,
    last_check_at: null,
    remark: body.remark == null ? '' : String(body.remark),
    created_at: timestamp,
    updated_at: timestamp,
  }
  mockProxies.unshift(proxy)
  return { proxy }
}

function updateMockProxy(proxy, body = {}) {
  if (body.name !== undefined) {
    const name = String(body.name || '').trim()
    if (!name) return { error: { code: 'BAD_REQUEST', message: 'name is required' } }
    proxy.name = name
  }
  if (body.ip_type !== undefined) {
    const ipType = normalizeMockProxyType(body.ip_type)
    if (!ipType) return { error: { code: 'SOCIAL_IP_TYPE_INVALID', message: 'social IP type is invalid' } }
    proxy.ip_type = ipType
  }
  if (body.endpoint !== undefined) {
    const endpointResult = normalizeMockProxyEndpoint(body.endpoint)
    if (!endpointResult.ok) return { error: { code: 'INVALID_PROXY_ENDPOINT', message: endpointResult.message } }
    proxy.endpoint = endpointResult.endpoint
    proxy.status = 'unknown'
    proxy.latency_ms = null
    proxy.last_check_at = null
  }
  if (body.remark !== undefined) {
    proxy.remark = body.remark == null ? '' : String(body.remark)
  }
  proxy.updated_at = now()
  return { proxy }
}

function testMockProxy(proxy) {
  const checkedAt = now()
  let result
  if (!proxy.endpoint) {
    result = { id: proxy.id, status: 'unknown', latency_ms: 0, error: 'no endpoint configured' }
  } else {
    result = { id: proxy.id, status: 'online', latency_ms: 80 }
  }
  proxy.status = result.status
  proxy.latency_ms = result.latency_ms > 0 ? result.latency_ms : null
  proxy.last_check_at = checkedAt
  proxy.updated_at = checkedAt
  return result
}

function isMockProxyUsableForUser(proxy, userId) {
  return !!proxy && proxy.user_id === userId && proxy.status === 'online' && String(proxy.endpoint || '').trim() !== ''
}

function assignMockDefaultProxy(account, proxyId) {
  updateMockSocialAccount(account, { proxy_id: proxyId })
  return { id: account.id, name: account.name, status: 'succeeded' }
}

function mockProxySnapshot(proxy) {
  if (!proxy) return ''
  return JSON.stringify({
    id: proxy.id,
    name: proxy.name,
    ip_type: proxy.ip_type,
    endpoint: proxy.endpoint || '',
    status: proxy.status,
  })
}

function mockSnapshotProxyID(snapshot) {
  try {
    const payload = JSON.parse(String(snapshot || ''))
    return Number(payload.id || 0)
  } catch {
    return 0
  }
}

function isMockAccountDefaultProxyUsableForUser(account, userId) {
  const proxyId = mockSnapshotProxyID(account?.default_proxy_snapshot)
  if (!proxyId) return false
  return isMockProxyUsableForUser(findMockProxy(proxyId), userId)
}

function deleteMockProxy(id) {
  const numericId = Number(id)
  const before = mockProxies.length
  mockProxies = mockProxies.filter((proxy) => proxy.id !== numericId)
  if (mockProxies.length === before) return false

  for (const account of mockSocialAccounts) {
    if (account.proxy_id === numericId || mockSnapshotProxyID(account.default_proxy_snapshot) === numericId) {
      account.proxy_id = null
      account.default_proxy_snapshot = ''
      account.updated_at = now()
    }
  }
  return true
}

function createMockSocialAccount(body = {}) {
  const account = {
    id: nextSocialAccountId++,
    name: String(body.name || `mock_account_${nextSocialAccountId}`).trim(),
    platform: normalizeMockSocialPlatform(body.platform || 'x_twitter'),
    username: normalizeMockSocialUsername(body.name || `mock_account_${nextSocialAccountId}`),
    platform_user_id: firstMockString(body.platform_user_id),
    password: body.password ? String(body.password) : '',
    phone: body.phone ? String(body.phone) : '',
    email: body.email ? String(body.email) : '',
    email_password: body.email_password ? String(body.email_password) : '',
    two_factor: body.two_factor ? String(body.two_factor) : '',
    backup_code: body.backup_code ? String(body.backup_code) : '',
    email_client_id: body.email_client_id ? String(body.email_client_id) : '',
    email_token: body.email_token ? String(body.email_token) : '',
    registration_ip: body.registration_ip ? String(body.registration_ip) : '',
    auth_cookie: firstMockString(body.auth_cookie),
    execution_auth: firstMockString(body.execution_auth),
    account_status: String(body.account_status || 'pending_check'),
    task_status: String(body.task_status || 'stored'),
    task_message: body.task_message === undefined ? 'Stored in mock account pool' : String(body.task_message),
    remark: body.remark ? String(body.remark) : '',
    assigned_user_id: body.assigned_user_id === null ? null : Number(body.assigned_user_id || adminUser.id),
    proxy_id: body.proxy_id == null ? null : Number(body.proxy_id),
    default_proxy_snapshot: firstMockString(body.default_proxy_snapshot),
    created_at: now(),
    updated_at: now(),
  }
  mockSocialAccounts.unshift(account)
  return account
}

function createMockWorkbenchImportAccount(body = {}, userId = regularUser.id) {
  const account = {
    id: nextSocialAccountId++,
    name: String(body.name || `mock_account_${nextSocialAccountId}`).trim(),
    platform: normalizeMockSocialPlatform(body.platform || 'x_twitter'),
    username: normalizeMockSocialUsername(body.name || `mock_account_${nextSocialAccountId}`),
    platform_user_id: firstMockString(body.platform_user_id),
    password: body.password ? String(body.password) : '',
    phone: body.phone ? String(body.phone) : '',
    email: body.email ? String(body.email) : '',
    email_password: body.email_password ? String(body.email_password) : '',
    two_factor: body.two_factor ? String(body.two_factor) : '',
    backup_code: body.backup_code ? String(body.backup_code) : '',
    email_client_id: body.email_client_id ? String(body.email_client_id) : '',
    email_token: body.email_token ? String(body.email_token) : '',
    registration_ip: body.registration_ip ? String(body.registration_ip) : '',
    auth_cookie: firstMockString(body.auth_cookie),
    execution_auth: firstMockString(body.execution_auth),
    account_status: 'not_stored',
    task_status: 'pending',
    task_message: '',
    remark: body.remark ? String(body.remark) : '',
    assigned_user_id: userId,
    proxy_id: null,
    default_proxy_snapshot: '',
    created_at: now(),
    updated_at: now(),
  }
  mockSocialAccounts.unshift(account)
  return account
}

function updateMockSocialAccount(account, body = {}) {
  const fields = [
    'name',
    'platform_user_id',
    'password',
    'phone',
    'email',
    'email_password',
    'two_factor',
    'backup_code',
    'email_client_id',
    'email_token',
    'registration_ip',
    'auth_cookie',
    'execution_auth',
    'account_status',
    'task_status',
    'task_message',
    'remark',
  ]
  for (const field of fields) {
    if (body[field] !== undefined) account[field] = body[field] == null ? '' : String(body[field])
  }
  if (account.name) account.username = normalizeMockSocialUsername(account.name)
  if (body.proxy_id !== undefined) {
    account.proxy_id = body.proxy_id == null ? null : Number(body.proxy_id)
    const proxy = account.proxy_id ? findMockProxy(account.proxy_id) : null
    account.default_proxy_snapshot = proxy ? mockProxySnapshot(proxy) : (account.proxy_id ? `mock-proxy-${account.proxy_id}` : '')
  }
  account.updated_at = now()
  return account
}

function updateMockUserSocialAccount(account, body = {}) {
  const fields = [
    'password',
    'phone',
    'email',
    'email_password',
    'two_factor',
    'backup_code',
    'email_client_id',
    'email_token',
    'registration_ip',
    'auth_cookie',
    'execution_auth',
    'remark',
  ]
  for (const field of fields) {
    if (body[field] !== undefined) account[field] = body[field] == null ? '' : String(body[field]).trim()
  }
  account.updated_at = now()
  return account
}

function createMockTaskLogs(body = {}, { admin = false } = {}) {
  const ids = Array.isArray(body.account_ids) ? body.account_ids.map(Number).filter(Boolean) : []
  const action = String(body.action || 'login_check')
  const targetPool = Array.isArray(body.target_pool) ? body.target_pool.map((item) => String(item)).filter(Boolean) : []
  const contentPool = Array.isArray(body.content_pool) ? body.content_pool.map((item) => String(item)).filter(Boolean) : []
  const fallbackTarget = body.target ? String(body.target) : ''
  const fallbackContent = body.content ? String(body.content) : ''
  const accounts = ids.map(findSocialAccount).filter(Boolean)

  const logs = accounts.map((account, index) => {
    const target = fallbackTarget || (targetPool.length > 0 ? targetPool[index % targetPool.length] : '')
    const content = fallbackContent || (contentPool.length > 0 ? contentPool[index % contentPool.length] : '')
    const success = account.account_status === 'available'
    account.task_status = success ? 'success' : 'failed'
    account.task_message = success ? `${action} completed in mock executor` : 'Mock executor rejected a non-available account'
    account.updated_at = now()
    const log = {
      id: nextSocialTaskLogId++,
      social_account_id: account.id,
      user_id: account.assigned_user_id || adminUser.id,
      action,
      platform: account.platform,
      account_name: account.name,
      target,
      content,
      status: success ? 'success' : 'failed',
      result_message: account.task_message,
      charged: success,
      charged_amount: success ? socialTaskUnitPrice : 0,
      price: socialTaskUnitPrice,
      charge_status: success ? 'charged' : 'not_charged',
      charge_source: success ? 'subscription' : '',
      proxy_id: account.proxy_id,
      proxy_snapshot: account.default_proxy_snapshot || '',
      billing_request_id: `mock-billing-${Date.now()}-${account.id}`,
      idempotency_key: body.client_request_id || `mock-${Date.now()}-${account.id}`,
      executed_at: now(),
      created_at: now(),
    }
    mockSocialTaskLogs.unshift(log)
    return admin ? clone(log) : userSafeTaskLog(log)
  })

  return logs
}

function usageActivityTime(log) {
  return new Date(log.executed_at || log.created_at || 0)
}

function usageCost(log) {
  if (log.status === 'success' && log.charge_status === 'charged') {
    const amount = Number(log.charged_amount || 0)
    return Number.isFinite(amount) ? amount : 0
  }
  return 0
}

function roundUsageAmount(value) {
  return Math.round(Number(value || 0) * 1000000) / 1000000
}

function parseMockUsageDate(raw, { endOfDay = false } = {}) {
  const value = String(raw || '').trim()
  if (!value) return null
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const suffix = endOfDay ? 'T23:59:59.999Z' : 'T00:00:00.000Z'
    const parsed = new Date(`${value}${suffix}`)
    return Number.isNaN(parsed.getTime()) ? null : parsed
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

function trendBucket(date, granularity = 'day') {
  const iso = date.toISOString()
  if (granularity === 'hour') return iso.slice(0, 13).replace('T', ' ') + ':00'
  if (granularity === 'month') return iso.slice(0, 7)
  return iso.slice(0, 10)
}

function userUsageLogs(userId) {
  return mockSocialTaskLogs.filter((log) => Number(log.user_id) === Number(userId))
}

function filterUsageLogs(url, userId) {
  const operation = String(url.searchParams.get('operation') || url.searchParams.get('model') || '').trim().toLowerCase()
  const status = String(url.searchParams.get('status') || '').trim().toLowerCase()
  const startTime = parseMockUsageDate(url.searchParams.get('start_date'))
  const endTime = parseMockUsageDate(url.searchParams.get('end_date'), { endOfDay: true })

  let items = userUsageLogs(userId)
  if (operation) items = items.filter((log) => String(log.action || '').toLowerCase() === operation)
  if (status) items = items.filter((log) => String(log.status || '').toLowerCase() === status)
  if (startTime) items = items.filter((log) => usageActivityTime(log) >= startTime)
  if (endTime) items = items.filter((log) => usageActivityTime(log) <= endTime)

  const direction = String(url.searchParams.get('sort_order') || 'desc').toLowerCase() === 'asc' ? 1 : -1
  const sortBy = String(url.searchParams.get('sort_by') || 'created_at').toLowerCase()
  return items.sort((a, b) => {
    let av
    let bv
    if (sortBy === 'cost') {
      av = usageCost(a)
      bv = usageCost(b)
    } else if (sortBy === 'operation' || sortBy === 'model') {
      av = String(a.action || '')
      bv = String(b.action || '')
    } else {
      av = usageActivityTime(a).getTime()
      bv = usageActivityTime(b).getTime()
    }
    if (av === bv) return (Number(a.id) - Number(b.id)) * direction
    return av > bv ? direction : -direction
  })
}

function usageLogProjection(log) {
  return {
    id: log.id,
    user_id: log.user_id,
    social_account_id: log.social_account_id,
    platform: log.platform || '',
    account_name: log.account_name || '',
    operation: log.action || '',
    status: log.status || '',
    quantity: 1,
    cost: roundUsageAmount(usageCost(log)),
    charge_status: log.charge_status || '',
    result_message: log.result_message || null,
    created_at: log.created_at,
    completed_at: log.executed_at || null,
  }
}

function usageStatsProjection(logs) {
  const totalCost = roundUsageAmount(logs.reduce((sum, log) => sum + usageCost(log), 0))
  return {
    total_requests: logs.length,
    total_input_tokens: 0,
    total_output_tokens: 0,
    total_cache_tokens: 0,
    total_tokens: logs.length,
    total_cost: totalCost,
    total_actual_cost: totalCost,
    average_duration_ms: 0,
  }
}

function userDashboardUsageStats(userId) {
  const logs = userUsageLogs(userId)
  const nowDate = new Date()
  const todayStart = new Date(Date.UTC(nowDate.getUTCFullYear(), nowDate.getUTCMonth(), nowDate.getUTCDate()))
  const recentStart = new Date(nowDate.getTime() - 5 * 60 * 1000)
  const todayLogs = logs.filter((log) => {
    const activity = usageActivityTime(log)
    return activity >= todayStart && activity <= nowDate
  })
  const recentLogs = logs.filter((log) => {
    const activity = usageActivityTime(log)
    return activity >= recentStart && activity <= nowDate
  })
  const stats = usageStatsProjection(logs)
  const todayStats = usageStatsProjection(todayLogs)
  const byPlatform = new Map()
  for (const log of logs) {
    const platform = log.platform || ''
    const current = byPlatform.get(platform) || {
      platform,
      total_requests: 0,
      total_tokens: 0,
      total_actual_cost: 0,
      today_requests: 0,
      today_tokens: 0,
      today_actual_cost: 0,
    }
    const cost = usageCost(log)
    current.total_requests += 1
    current.total_tokens += 1
    current.total_actual_cost += cost
    const activity = usageActivityTime(log)
    if (activity >= todayStart && activity <= nowDate) {
      current.today_requests += 1
      current.today_tokens += 1
      current.today_actual_cost += cost
    }
    byPlatform.set(platform, current)
  }

  return {
    total_api_keys: 0,
    active_api_keys: 0,
    total_requests: stats.total_requests,
    total_input_tokens: 0,
    total_output_tokens: 0,
    total_cache_creation_tokens: 0,
    total_cache_read_tokens: 0,
    total_tokens: stats.total_tokens,
    total_cost: stats.total_cost,
    total_actual_cost: stats.total_actual_cost,
    today_requests: todayStats.total_requests,
    today_input_tokens: 0,
    today_output_tokens: 0,
    today_cache_creation_tokens: 0,
    today_cache_read_tokens: 0,
    today_tokens: todayStats.total_tokens,
    today_cost: todayStats.total_cost,
    today_actual_cost: todayStats.total_actual_cost,
    average_duration_ms: 0,
    rpm: Math.floor(recentLogs.length / 5),
    tpm: Math.floor(recentLogs.length / 5),
    by_platform: Array.from(byPlatform.values()).map((item) => ({
      ...item,
      total_actual_cost: roundUsageAmount(item.total_actual_cost),
      today_actual_cost: roundUsageAmount(item.today_actual_cost),
    })),
  }
}

function userUsageTrend(userId, granularity = 'day') {
  const end = new Date()
  const start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000)
  const byDate = new Map()
  for (const log of userUsageLogs(userId)) {
    const activity = usageActivityTime(log)
    if (activity < start || activity > end) continue
    const date = trendBucket(activity, granularity)
    const current = byDate.get(date) || {
      date,
      requests: 0,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_read_tokens: 0,
      total_tokens: 0,
      cost: 0,
      actual_cost: 0,
    }
    const cost = usageCost(log)
    current.requests += 1
    current.total_tokens += 1
    current.cost += cost
    current.actual_cost += cost
    byDate.set(date, current)
  }
  return Array.from(byDate.values())
    .map((item) => ({ ...item, cost: roundUsageAmount(item.cost), actual_cost: roundUsageAmount(item.actual_cost) }))
    .sort((a, b) => a.date.localeCompare(b.date))
}

function userById(userId) {
  return [adminUser, regularUser].find((user) => Number(user.id) === Number(userId)) || null
}

function usageTrendFromLogs(logs, granularity = 'day') {
  const end = new Date()
  const start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000)
  const byDate = new Map()
  for (const log of logs) {
    const activity = usageActivityTime(log)
    if (activity < start || activity > end) continue
    const date = trendBucket(activity, granularity)
    const current = byDate.get(date) || {
      date,
      requests: 0,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_read_tokens: 0,
      total_tokens: 0,
      cost: 0,
      actual_cost: 0,
    }
    const cost = usageCost(log)
    current.requests += 1
    current.total_tokens += 1
    current.cost += cost
    current.actual_cost += cost
    byDate.set(date, current)
  }
  return Array.from(byDate.values())
    .map((item) => ({ ...item, cost: roundUsageAmount(item.cost), actual_cost: roundUsageAmount(item.actual_cost) }))
    .sort((a, b) => a.date.localeCompare(b.date))
}

function adminDashboardStats() {
  const logs = [...mockSocialTaskLogs]
  const nowDate = new Date()
  const todayStart = new Date(Date.UTC(nowDate.getUTCFullYear(), nowDate.getUTCMonth(), nowDate.getUTCDate()))
  const hourStart = new Date(nowDate)
  hourStart.setUTCMinutes(0, 0, 0)
  const recentStart = new Date(nowDate.getTime() - 5 * 60 * 1000)
  const users = [adminUser, regularUser]
  const todayLogs = logs.filter((log) => {
    const activity = usageActivityTime(log)
    return activity >= todayStart && activity <= nowDate
  })
  const hourlyUserIds = new Set()
  const activeUserIds = new Set()
  let recentRequests = 0
  for (const log of logs) {
    const activity = usageActivityTime(log)
    if (activity >= todayStart && activity <= nowDate) activeUserIds.add(log.user_id)
    if (activity >= hourStart && activity <= nowDate) hourlyUserIds.add(log.user_id)
    if (activity >= recentStart && activity <= nowDate) recentRequests += 1
  }
  const stats = usageStatsProjection(logs)
  const todayStats = usageStatsProjection(todayLogs)

  return {
    total_users: users.length,
    today_new_users: users.filter((user) => new Date(user.created_at) >= todayStart).length,
    active_users: activeUserIds.size,
    hourly_active_users: hourlyUserIds.size,
    stats_updated_at: now(),
    stats_stale: false,
    total_api_keys: 0,
    active_api_keys: 0,
    total_accounts: mockSocialAccounts.length,
    normal_accounts: mockSocialAccounts.filter((account) => account.account_status === 'available').length,
    error_accounts: mockSocialAccounts.filter((account) => ['invalid', 'not_stored'].includes(account.account_status)).length,
    ratelimit_accounts: mockSocialAccounts.filter((account) => account.account_status === 'limited').length,
    overload_accounts: mockSocialAccounts.filter((account) => account.account_status === 'pending_check').length,
    total_requests: stats.total_requests,
    total_input_tokens: 0,
    total_output_tokens: 0,
    total_cache_creation_tokens: 0,
    total_cache_read_tokens: 0,
    total_tokens: stats.total_tokens,
    total_cost: stats.total_cost,
    total_actual_cost: stats.total_actual_cost,
    total_account_cost: 0,
    today_requests: todayStats.total_requests,
    today_input_tokens: 0,
    today_output_tokens: 0,
    today_cache_creation_tokens: 0,
    today_cache_read_tokens: 0,
    today_tokens: todayStats.total_tokens,
    today_cost: todayStats.total_cost,
    today_actual_cost: todayStats.total_actual_cost,
    today_account_cost: 0,
    average_duration_ms: 0,
    rpm: Math.floor(recentRequests / 5),
    tpm: Math.floor(recentRequests / 5),
  }
}

function adminUserUsageTrend(granularity = 'day', limit = 20) {
  const end = new Date()
  const start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000)
  const byUserDate = new Map()
  for (const log of mockSocialTaskLogs) {
    const activity = usageActivityTime(log)
    if (activity < start || activity > end) continue
    const date = trendBucket(activity, granularity)
    const key = `${date}:${log.user_id}`
    const user = userById(log.user_id)
    const current = byUserDate.get(key) || {
      date,
      user_id: log.user_id,
      email: user?.email || '',
      username: user?.username || '',
      requests: 0,
      tokens: 0,
      cost: 0,
      actual_cost: 0,
    }
    const cost = usageCost(log)
    current.requests += 1
    current.tokens += 1
    current.cost += cost
    current.actual_cost += cost
    byUserDate.set(key, current)
  }
  return Array.from(byUserDate.values())
    .map((item) => ({ ...item, cost: roundUsageAmount(item.cost), actual_cost: roundUsageAmount(item.actual_cost) }))
    .sort((a, b) => {
      if (a.date !== b.date) return a.date.localeCompare(b.date)
      if (a.actual_cost !== b.actual_cost) return b.actual_cost - a.actual_cost
      if (a.requests !== b.requests) return b.requests - a.requests
      return Number(a.user_id) - Number(b.user_id)
    })
    .slice(0, limit)
}

function adminUserSpendingRanking(limit = 20) {
  const end = new Date()
  const start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000)
  const byUser = new Map()
  let totalRequests = 0
  let totalActualCost = 0
  for (const log of mockSocialTaskLogs) {
    const activity = usageActivityTime(log)
    if (activity < start || activity > end) continue
    const user = userById(log.user_id)
    const current = byUser.get(log.user_id) || {
      user_id: log.user_id,
      email: user?.email || '',
      username: user?.username || '',
      actual_cost: 0,
      requests: 0,
      tokens: 0,
    }
    const cost = usageCost(log)
    current.requests += 1
    current.tokens += 1
    current.actual_cost += cost
    totalRequests += 1
    totalActualCost += cost
    byUser.set(log.user_id, current)
  }
  const ranking = Array.from(byUser.values())
    .map((item) => ({ ...item, actual_cost: roundUsageAmount(item.actual_cost) }))
    .sort((a, b) => {
      if (a.actual_cost !== b.actual_cost) return b.actual_cost - a.actual_cost
      if (a.requests !== b.requests) return b.requests - a.requests
      return Number(a.user_id) - Number(b.user_id)
    })
    .slice(0, limit)

  return {
    ranking,
    total_actual_cost: roundUsageAmount(totalActualCost),
    total_requests: totalRequests,
    total_tokens: totalRequests,
  }
}

function userSafeTaskLog(log) {
  const copy = clone(log)
  delete copy.user_id
  delete copy.target
  delete copy.content
  delete copy.price
  delete copy.charge_source
  delete copy.proxy_id
  delete copy.proxy_snapshot
  delete copy.billing_request_id
  delete copy.idempotency_key
  return copy
}

function mockCsvPayload(items) {
  const columns = [
    'platform',
    'username',
    'name',
    'platform_user_id',
    'password',
    'phone',
    'email',
    'email_password',
    'two_factor',
    'backup_code',
    'email_client_id',
    'email_token',
    'registration_ip',
    'auth_cookie',
    'execution_auth',
    'default_proxy_snapshot',
    'account_status',
    'task_status',
    'remark',
    'created_at',
    'updated_at',
  ]
  const rows = [
    columns.join(','),
    ...items.map((account) => columns
      .map((column) => `"${String(account[column] ?? '').replace(/"/g, '""')}"`)
      .join(',')),
  ]
  return rows.join('\n')
}

function sendText(res, status, text, contentType = 'text/plain; charset=utf-8') {
  res.writeHead(status, {
    'Content-Type': contentType,
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization, Accept-Language, X-Idempotency-Key',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
  })
  res.end(text)
}

const userTaskPath = '/api/v1/accounts' + '/tasks'
const adminTaskPath = '/api/v1/admin/accounts' + '/tasks'

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`)
  const path = url.pathname

  if (req.method === 'OPTIONS') {
    send(res, 204, {})
    return
  }

  if (path === '/health') {
    ok(res, { status: 'ok', mode: 'mock' })
    return
  }

  if (path === '/setup/status' && req.method === 'GET') {
    ok(res, { needs_setup: false, step: 'complete' })
    return
  }

  if (path.startsWith('/api/v1/admin/') && currentMockUser(req).role !== 'admin') {
    send(res, 403, { code: 'ADMIN_ONLY', message: 'Admin permission is required' })
    return
  }

  if (path === '/api/v1/admin/data-management/agent/health' && req.method === 'GET') {
    ok(res, {
      enabled: dataManagementAgentEnabled,
      reason: dataManagementAgentEnabled ? '' : 'DATA_MANAGEMENT_DEPRECATED',
      socket_path: '/var/run/socialops/datamanagementd.sock',
      agent: dataManagementAgentEnabled
        ? {
            status: 'ready',
            version: 'mock-agent',
            uptime_seconds: 1860,
          }
        : undefined,
    })
    return
  }

  if (path.startsWith('/api/v1/admin/data-management/') && !dataManagementAgentEnabled) {
    send(res, 503, {
      code: 'DATA_MANAGEMENT_DEPRECATED',
      message: 'Data management agent is disabled in mock mode',
      metadata: { socket_path: '/var/run/socialops/datamanagementd.sock' },
    })
    return
  }

  if (path === '/api/v1/admin/data-management/config' && req.method === 'GET') {
    ok(res, clone(dataManagementConfig))
    return
  }

  if (/^\/api\/v1\/admin\/data-management\/sources\/(postgres|redis)\/profiles$/.test(path) && req.method === 'GET') {
    const sourceType = path.split('/')[6]
    ok(res, { items: clone(mockSourceProfiles[sourceType] || []) })
    return
  }

  if (path === '/api/v1/admin/data-management/s3/profiles' && req.method === 'GET') {
    ok(res, { items: clone(mockS3Profiles) })
    return
  }

  if (path === '/api/v1/admin/data-management/backups' && req.method === 'GET') {
    ok(res, { items: clone(mockBackupJobs), next_page_token: '' })
    return
  }

  if (path === '/api/v1/admin/data-management/backups' && req.method === 'POST') {
    const body = await readJson(req)
    const job = {
      job_id: `mock-backup-${Date.now()}`,
      backup_type: body.backup_type || 'full',
      status: 'queued',
      triggered_by: 'admin:1',
      s3_profile_id: body.upload_to_s3 ? dataManagementConfig.active_s3_profile_id : '',
      postgres_profile_id: dataManagementConfig.active_postgres_profile_id,
      redis_profile_id: dataManagementConfig.active_redis_profile_id,
      started_at: now(),
      finished_at: '',
      error_message: '',
    }
    mockBackupJobs.unshift(job)
    ok(res, { job_id: job.job_id, status: job.status })
    return
  }

  if (path === '/api/v1/admin/backups/s3-config' && req.method === 'GET') {
    ok(res, publicBackupS3Config())
    return
  }

  if (path === '/api/v1/admin/backups/s3-config' && req.method === 'PUT') {
    const body = await readJson(req)
    const nextConfig = {
      ...mockBackupS3Config,
      endpoint: body.endpoint ?? mockBackupS3Config.endpoint,
      region: body.region ?? mockBackupS3Config.region,
      bucket: body.bucket ?? mockBackupS3Config.bucket,
      access_key_id: body.access_key_id ?? mockBackupS3Config.access_key_id,
      prefix: body.prefix ?? mockBackupS3Config.prefix,
      force_path_style: body.force_path_style === undefined
        ? mockBackupS3Config.force_path_style
        : Boolean(body.force_path_style),
    }
    if (typeof body.secret_access_key === 'string' && body.secret_access_key.trim()) {
      nextConfig.secret_access_key = body.secret_access_key
    }
    mockBackupS3Config = nextConfig
    ok(res, publicBackupS3Config())
    return
  }

  if (path === '/api/v1/admin/backups/s3-config/test' && req.method === 'POST') {
    const body = await readJson(req)
    const testConfig = {
      ...mockBackupS3Config,
      ...body,
      secret_access_key: typeof body.secret_access_key === 'string' && body.secret_access_key.trim()
        ? body.secret_access_key
        : mockBackupS3Config.secret_access_key,
    }
    if (!isBackupS3Configured(testConfig)) {
      ok(res, {
        ok: false,
        message: 'incomplete S3 config: bucket, access_key_id, secret_access_key are required',
      })
      return
    }
    ok(res, { ok: true, message: 'connection successful' })
    return
  }

  if (path === '/api/v1/admin/backups/schedule' && req.method === 'GET') {
    ok(res, clone(mockBackupSchedule))
    return
  }

  if (path === '/api/v1/admin/backups/schedule' && req.method === 'PUT') {
    const body = await readJson(req)
    const cronExpr = String(body.cron_expr ?? mockBackupSchedule.cron_expr ?? '').trim()
    if (body.enabled && !cronExpr) {
      send(res, 400, { code: 'INVALID_CRON', message: 'cron expression is required when schedule is enabled' })
      return
    }
    if (cronExpr && cronExpr.split(/\s+/).length !== 5) {
      send(res, 400, { code: 'INVALID_CRON', message: 'invalid cron expression' })
      return
    }
    mockBackupSchedule = {
      enabled: body.enabled === undefined ? mockBackupSchedule.enabled : Boolean(body.enabled),
      cron_expr: cronExpr,
      retain_days: Math.max(0, Number(body.retain_days ?? mockBackupSchedule.retain_days) || 0),
      retain_count: Math.max(0, Number(body.retain_count ?? mockBackupSchedule.retain_count) || 0),
    }
    ok(res, clone(mockBackupSchedule))
    return
  }

  if (path === '/api/v1/admin/backups' && req.method === 'GET') {
    refreshMockBackupRecords()
    ok(res, { items: mockBackupRecords.map(backupRecordForResponse) })
    return
  }

  if (path === '/api/v1/admin/backups' && req.method === 'POST') {
    if (!isBackupS3Configured()) {
      send(res, 400, { code: 'BACKUP_S3_NOT_CONFIGURED', message: 'backup S3 storage is not configured' })
      return
    }
    refreshMockBackupRecords()
    if (mockBackupRecords.some((record) => record.status === 'running')) {
      send(res, 409, { code: 'BACKUP_IN_PROGRESS', message: 'a backup is already in progress' })
      return
    }
    const body = await readJson(req)
    const expireDays = Number(body.expire_days ?? 14)
    const record = buildMockBackupRecord(Number.isFinite(expireDays) ? Math.max(0, expireDays) : 14)
    mockBackupRecords = [record, ...mockBackupRecords].slice(0, 100)
    accepted(res, backupRecordForResponse(record))
    return
  }

  {
    const downloadMatch = path.match(/^\/api\/v1\/admin\/backups\/([^/]+)\/download-url$/)
    if (downloadMatch && req.method === 'GET') {
      const record = findMockBackupRecord(decodeURIComponent(downloadMatch[1]))
      if (!record) {
        send(res, 404, { code: 'BACKUP_NOT_FOUND', message: 'backup record not found' })
        return
      }
      if (record.status !== 'completed') {
        send(res, 400, { code: 'BACKUP_NOT_READY', message: 'backup file is not ready for download' })
        return
      }
      ok(res, { url: `https://mock.local/downloads/${encodeURIComponent(record.file_name)}` })
      return
    }
  }

  {
    const restoreMatch = path.match(/^\/api\/v1\/admin\/backups\/([^/]+)\/restore$/)
    if (restoreMatch && req.method === 'POST') {
      const record = findMockBackupRecord(decodeURIComponent(restoreMatch[1]))
      if (!record) {
        send(res, 404, { code: 'BACKUP_NOT_FOUND', message: 'backup record not found' })
        return
      }
      if (record.status !== 'completed') {
        send(res, 400, { code: 'BACKUP_NOT_READY', message: 'only completed backups can be restored' })
        return
      }
      if (mockBackupRecords.some((item) => item.restore_status === 'running')) {
        send(res, 409, { code: 'RESTORE_IN_PROGRESS', message: 'a restore is already in progress' })
        return
      }
      const body = await readJson(req)
      if (!body.password) {
        send(res, 400, { code: 'BAD_REQUEST', message: 'password is required for restore operation' })
        return
      }
      if (body.password !== adminPassword) {
        send(res, 400, { code: 'INVALID_PASSWORD', message: 'incorrect admin password' })
        return
      }
      record.restore_status = 'running'
      record.restore_error = ''
      record.restored_at = ''
      record.__restore_started_at = now()
      accepted(res, backupRecordForResponse(record))
      return
    }
  }

  {
    const backupMatch = path.match(/^\/api\/v1\/admin\/backups\/([^/]+)$/)
    if (backupMatch && req.method === 'GET') {
      const record = findMockBackupRecord(decodeURIComponent(backupMatch[1]))
      if (!record) {
        send(res, 404, { code: 'BACKUP_NOT_FOUND', message: 'backup record not found' })
        return
      }
      ok(res, backupRecordForResponse(record))
      return
    }
    if (backupMatch && req.method === 'DELETE') {
      const backupID = decodeURIComponent(backupMatch[1])
      const before = mockBackupRecords.length
      mockBackupRecords = mockBackupRecords.filter((record) => record.id !== backupID)
      if (mockBackupRecords.length === before) {
        send(res, 404, { code: 'BACKUP_NOT_FOUND', message: 'backup record not found' })
        return
      }
      ok(res, { deleted: true })
      return
    }
  }

  if (path === '/api/v1/settings/public' && req.method === 'GET') {
    ok(res, publicSettings)
    return
  }

  if (path === '/api/v1/admin/settings' && req.method === 'GET') {
    ok(res, clone(adminSettings))
    return
  }

  if (path === '/api/v1/admin/settings' && req.method === 'PUT') {
    const body = await readJson(req)
    Object.assign(adminSettings, body || {})
    stripMockAdminSettingSecrets()
    syncMockPublicSettingsFromAdminSettings()
    ok(res, clone(adminSettings))
    return
  }

  if (path === '/api/v1/admin/settings/test-smtp' && req.method === 'POST') {
    ok(res, { message: 'SMTP test succeeded in mock mode' })
    return
  }

  if (path === '/api/v1/admin/settings/send-test-email' && req.method === 'POST') {
    ok(res, { message: 'Test email sent in mock mode' })
    return
  }

  if (path === '/api/v1/auth/validate-promo-code' && req.method === 'POST') {
    const body = await readJson(req)
    ok(res, promoValidationResponse(body.code))
    return
  }

  if (path === '/api/v1/admin/settings/admin-api-key' && req.method === 'GET') {
    ok(res, clone(adminApiKey))
    return
  }

  if (path === '/api/v1/admin/settings/admin-api-key/regenerate' && req.method === 'POST') {
    const key = `sk-mock-${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`
    adminApiKey = {
      exists: true,
      masked_key: `${key.slice(0, 8)}...${key.slice(-4)}`,
    }
    ok(res, { key })
    return
  }

  if (path === '/api/v1/admin/settings/admin-api-key' && req.method === 'DELETE') {
    adminApiKey = { exists: false, masked_key: '' }
    ok(res, { message: 'Deleted' })
    return
  }

  if (path === '/api/v1/admin/settings/email-templates' && req.method === 'GET') {
    ok(res, {
      events: clone(emailTemplateEvents),
      locales: ['zh-CN', 'en-US'],
      templates: emailTemplateEvents.flatMap((event) =>
        ['zh-CN', 'en-US'].map((locale) => {
          const template = getStoredEmailTemplate(event.value, locale)
          return {
            event: event.value,
            locale,
            subject: template.subject,
            is_custom: template.is_custom,
            updated_at: template.updated_at,
          }
        }),
      ),
      placeholders: ['site_name', 'recipient_name', 'recipient_email', 'verification_code'],
    })
    return
  }

  const emailTemplateMatch = path.match(/^\/api\/v1\/admin\/settings\/email-templates\/([^/]+)\/([^/]+)(?:\/restore-official)?$/)
  if (emailTemplateMatch) {
    const event = decodeURIComponent(emailTemplateMatch[1])
    const locale = decodeURIComponent(emailTemplateMatch[2])
    const restore = path.endsWith('/restore-official')
    if (req.method === 'GET') {
      ok(res, clone(getStoredEmailTemplate(event, locale)))
      return
    }
    if (req.method === 'PUT') {
      const body = await readJson(req)
      const template = {
        ...defaultEmailTemplate(event, locale),
        subject: String(body.subject || ''),
        html: String(body.html || ''),
        is_custom: true,
        updated_at: now(),
      }
      emailTemplateStore.set(emailTemplateKey(event, locale), template)
      ok(res, clone(template))
      return
    }
    if (req.method === 'POST' && restore) {
      const template = defaultEmailTemplate(event, locale)
      emailTemplateStore.delete(emailTemplateKey(event, locale))
      ok(res, clone(template))
      return
    }
  }

  if (
    (path === '/api/v1/admin/settings/email-template-preview' ||
      path === '/api/v1/admin/settings/email-templates/preview') &&
    req.method === 'POST'
  ) {
    const body = await readJson(req)
    ok(res, {
      subject: String(body.subject || ''),
      html: String(body.html || ''),
    })
    return
  }

  if (path === '/api/v1/auth/login' && req.method === 'POST') {
    const body = await readJson(req)
    if (body.email === regularUser.email) {
      ok(res, {
        access_token: 'dev-mock-user-token',
        refresh_token: 'dev-mock-user-refresh-token',
        expires_in: 86400,
        token_type: 'Bearer',
        user: regularUser,
      })
      return
    }
    if (body.email !== adminEmail || body.password !== adminPassword) {
      send(res, 401, { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password' })
      return
    }
    ok(res, {
      access_token: 'dev-mock-access-token',
      refresh_token: 'dev-mock-refresh-token',
      expires_in: 86400,
      token_type: 'Bearer',
      user: adminUser,
    })
    return
  }

  if (path === '/api/v1/auth/me' && req.method === 'GET') {
    ok(res, mockProfileResponse(currentMockUser(req), { includeRunMode: true }))
    return
  }

  if (path === '/api/v1/user/profile' && req.method === 'GET') {
    ok(res, mockProfileResponse(currentMockUser(req)))
    return
  }

  if (path === '/api/v1/user' && req.method === 'PUT') {
    const body = await readJson(req)
    const user = currentMockUser(req)
    if (body.username !== undefined) {
      user.username = String(body.username || '').trim()
    }
    if (body.avatar_url !== undefined) {
      user.avatar_url = body.avatar_url == null ? null : String(body.avatar_url || '').trim()
    }
    if (body.balance_notify_enabled !== undefined) {
      user.balance_notify_enabled = body.balance_notify_enabled === true
    }
    if (body.balance_notify_threshold !== undefined) {
      if (body.balance_notify_threshold === null || body.balance_notify_threshold === '') {
        user.balance_notify_threshold = null
      } else {
        const threshold = Number(body.balance_notify_threshold)
        user.balance_notify_threshold = Number.isFinite(threshold) ? threshold : null
      }
    }
    user.updated_at = now()
    ok(res, mockProfileResponse(user))
    return
  }

  if (path === '/api/v1/user/password' && req.method === 'PUT') {
    ok(res, { message: 'Password changed successfully' })
    return
  }

  if (path === '/api/v1/user/account-bindings/email/send-code' && req.method === 'POST') {
    ok(res, { message: 'Verification code sent successfully' })
    return
  }

  if (path === '/api/v1/user/account-bindings/email' && req.method === 'POST') {
    const body = await readJson(req)
    const email = String(body.email || '').trim()
    if (!email) {
      send(res, 400, { code: 'INVALID_EMAIL', message: 'email is required' })
      return
    }
    const user = currentMockUser(req)
    const previousEmail = user.email
    user.email = email
    if (!user.username || user.username === previousEmail) {
      user.username = email
    }
    user.updated_at = now()
    ok(res, mockProfileResponse(user))
    return
  }

  if (/^\/api\/v1\/user\/account-bindings\/[^/]+$/.test(path) && req.method === 'DELETE') {
    const provider = normalizeMockIdentityProvider(path.split('/').pop())
    if (!provider || provider === 'email') {
      send(res, 400, { code: 'IDENTITY_PROVIDER_INVALID', message: 'identity provider is invalid' })
      return
    }
    ok(res, mockProfileResponse(currentMockUser(req)))
    return
  }

  if (path === '/api/v1/user/auth-identities/bind/start' && req.method === 'POST') {
    const body = await readJson(req)
    const provider = normalizeMockIdentityProvider(body.provider)
    if (!provider || provider === 'email') {
      send(res, 400, { code: 'IDENTITY_PROVIDER_INVALID', message: 'identity provider is invalid' })
      return
    }
    const redirectTo = String(body.redirect_to || '').trim() || '/settings/profile'
    if (!redirectTo.startsWith('/') || redirectTo.startsWith('//')) {
      send(res, 400, { code: 'IDENTITY_REDIRECT_INVALID', message: 'identity redirect is invalid' })
      return
    }
    ok(res, {
      provider,
      authorize_url: mockProviderBindStartPath(provider, redirectTo),
      method: 'GET',
      use_browser_redirect: true,
    })
    return
  }

  if (path === '/api/v1/user/notify-email/send-code' && req.method === 'POST') {
    ok(res, { message: 'Verification code sent successfully' })
    return
  }

  if (path === '/api/v1/user/notify-email/verify' && req.method === 'POST') {
    const body = await readJson(req)
    const email = String(body.email || '').trim()
    if (!email) {
      send(res, 400, { code: 'INVALID_EMAIL', message: 'email is required' })
      return
    }
    const user = currentMockUser(req)
    const existing = user.balance_notify_extra_emails.find((entry) => entry.email === email)
    if (existing) {
      existing.disabled = false
      existing.verified = true
    } else {
      user.balance_notify_extra_emails.push({ email, disabled: false, verified: true })
    }
    user.updated_at = now()
    ok(res, mockProfileResponse(user))
    return
  }

  if (path === '/api/v1/user/notify-email/toggle' && req.method === 'PUT') {
    const body = await readJson(req)
    const email = String(body.email || '').trim()
    const user = currentMockUser(req)
    const entry = user.balance_notify_extra_emails.find((item) => item.email === email)
    if (!entry) {
      send(res, 400, { code: 'NOTIFY_EMAIL_NOT_FOUND', message: 'notification email not found' })
      return
    }
    entry.disabled = body.disabled === true
    user.updated_at = now()
    ok(res, mockProfileResponse(user))
    return
  }

  if (path === '/api/v1/user/notify-email' && req.method === 'DELETE') {
    const body = await readJson(req)
    const email = String(body.email || '').trim()
    const user = currentMockUser(req)
    const before = user.balance_notify_extra_emails.length
    user.balance_notify_extra_emails = user.balance_notify_extra_emails.filter((entry) => entry.email !== email)
    if (user.balance_notify_extra_emails.length === before) {
      send(res, 400, { code: 'NOTIFY_EMAIL_NOT_FOUND', message: 'notification email not found' })
      return
    }
    user.updated_at = now()
    ok(res, mockProfileResponse(user))
    return
  }

  if (path === '/api/v1/user/totp/status' && req.method === 'GET') {
    const state = mockTotpState(currentMockUser(req))
    ok(res, {
      enabled: state.enabled,
      enabled_at: mockTotpEnabledAtUnix(state),
      feature_enabled: publicSettings.totp_enabled === true,
    })
    return
  }

  if (path === '/api/v1/user/totp/verification-method' && req.method === 'GET') {
    ok(res, { method: publicSettings.email_verify_enabled === true ? 'email' : 'password' })
    return
  }

  if (path === '/api/v1/user/totp/send-code' && req.method === 'POST') {
    if (publicSettings.email_verify_enabled !== true) {
      send(res, 400, { code: 'EMAIL_VERIFY_NOT_ENABLED', message: 'email verification is not enabled' })
      return
    }
    ok(res, { success: true })
    return
  }

  if (path === '/api/v1/user/totp/setup' && req.method === 'POST') {
    if (publicSettings.totp_enabled !== true) {
      send(res, 400, { code: 'TOTP_NOT_ENABLED', message: 'totp feature is not enabled' })
      return
    }
    const body = await readJson(req)
    const state = mockTotpState(currentMockUser(req))
    if (state.enabled) {
      send(res, 400, { code: 'TOTP_ALREADY_ENABLED', message: 'totp is already enabled for this account' })
      return
    }
    if (publicSettings.email_verify_enabled === true && !String(body.email_code || '').trim()) {
      send(res, 400, { code: 'VERIFY_CODE_REQUIRED', message: 'email verification code is required' })
      return
    }
    if (publicSettings.email_verify_enabled !== true && !String(body.password || '').trim()) {
      send(res, 400, { code: 'PASSWORD_REQUIRED', message: 'password is required' })
      return
    }
    state.secret = 'JBSWY3DPEHPK3PXP'
    state.setup_token = `mock-totp-setup-${currentMockUser(req).id}`
    ok(res, {
      secret: state.secret,
      qr_code_url: `otpauth://totp/SocialOps:${encodeURIComponent(currentMockUser(req).email)}?secret=${state.secret}&issuer=SocialOps`,
      setup_token: state.setup_token,
      countdown: 300,
    })
    return
  }

  if (path === '/api/v1/user/totp/enable' && req.method === 'POST') {
    if (publicSettings.totp_enabled !== true) {
      send(res, 400, { code: 'TOTP_NOT_ENABLED', message: 'totp feature is not enabled' })
      return
    }
    const body = await readJson(req)
    const state = mockTotpState(currentMockUser(req))
    if (!state.setup_token || body.setup_token !== state.setup_token) {
      send(res, 400, { code: 'TOTP_SETUP_EXPIRED', message: 'totp setup session expired' })
      return
    }
    if (String(body.totp_code || '').trim().length !== 6) {
      send(res, 400, { code: 'TOTP_INVALID_CODE', message: 'invalid totp code' })
      return
    }
    state.enabled = true
    state.enabled_at = now()
    state.setup_token = ''
    ok(res, { success: true })
    return
  }

  if (path === '/api/v1/user/totp/disable' && req.method === 'POST') {
    const body = await readJson(req)
    const state = mockTotpState(currentMockUser(req))
    if (!state.enabled) {
      send(res, 400, { code: 'TOTP_NOT_SETUP', message: 'totp is not set up for this account' })
      return
    }
    if (publicSettings.email_verify_enabled === true && !String(body.email_code || '').trim()) {
      send(res, 400, { code: 'VERIFY_CODE_REQUIRED', message: 'email verification code is required' })
      return
    }
    if (publicSettings.email_verify_enabled !== true && !String(body.password || '').trim()) {
      send(res, 400, { code: 'PASSWORD_REQUIRED', message: 'password is required' })
      return
    }
    state.enabled = false
    state.enabled_at = null
    state.secret = ''
    state.setup_token = ''
    ok(res, { success: true })
    return
  }

  if (path === '/api/v1/auth/logout' && req.method === 'POST') {
    ok(res, null)
    return
  }

  if (path === '/api/v1/auth/refresh' && req.method === 'POST') {
    ok(res, {
      access_token: 'dev-mock-access-token',
      refresh_token: 'dev-mock-refresh-token',
      expires_in: 86400,
      token_type: 'Bearer',
    })
    return
  }

  if (path === '/api/v1/announcements' && req.method === 'GET') {
    const user = currentMockUser(req)
    const unreadOnly = url.searchParams.get('unread_only') === '1'
    let active = mockAnnouncements
      .filter((item) => mockAnnouncementIsActive(item))
      .filter((item) => mockAnnouncementVisibleToUser(item, user))
      .map((item) => mockUserAnnouncementForResponse(item, user))
    if (unreadOnly) active = active.filter((item) => !item.read_at)
    ok(res, clone(active))
    return
  }

  if (/^\/api\/v1\/announcements\/\d+\/read$/.test(path) && req.method === 'POST') {
    const user = currentMockUser(req)
    const item = mockAnnouncements.find((announcement) => announcement.id === Number(path.split('/')[4]))
    if (!item || !mockAnnouncementIsActive(item) || !mockAnnouncementVisibleToUser(item, user)) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock announcement not found' })
      return
    }
    const key = mockAnnouncementReadKey(item.id, user.id)
    if (!mockAnnouncementReads.has(key)) {
      mockAnnouncementReads.set(key, now())
    }
    ok(res, { message: 'Marked as read' })
    return
  }

  if (path === '/api/v1/redeem' && req.method === 'POST') {
    const body = await readJson(req)
    const result = redeemMockCode(req, body.code)
    if (result.error) {
      send(res, result.error.status, result.error.payload)
      return
    }
    ok(res, redeemCodeForResponse(result.code))
    return
  }

  if (path === '/api/v1/redeem/history' && req.method === 'GET') {
    const user = currentMockUser(req)
    const items = mockRedeemCodes
      .filter((code) => code.used_by === user.id)
      .sort((a, b) => (Date.parse(b.used_at || '') || 0) - (Date.parse(a.used_at || '') || 0))
      .slice(0, 25)
      .map(redeemCodeForResponse)
    ok(res, items)
    return
  }

  if (path === '/api/v1/user/aff' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, affiliateDetailForUser(user.id))
    return
  }

  if (path === '/api/v1/user/aff/transfer' && req.method === 'POST') {
    const user = currentMockUser(req)
    const profile = ensureAffiliateProfile(user.id)
    const transferred = roundMoney(profile.aff_quota)
    if (transferred <= 0) {
      send(res, 400, { code: 'AFFILIATE_QUOTA_EMPTY', message: 'no affiliate quota available to transfer' })
      return
    }
    user.balance = roundMoney(Number(user.balance || 0) + transferred)
    profile.aff_quota = 0
    profile.updated_at = now()
    mockAffiliateTransfers.unshift({
      ledger_id: nextAffiliateLedgerId++,
      user_id: user.id,
      user_email: user.email,
      username: user.username,
      amount: transferred,
      balance_after: user.balance,
      available_quota_after: profile.aff_quota,
      frozen_quota_after: profile.aff_frozen_quota,
      history_quota_after: profile.aff_history_quota,
      snapshot_available: true,
      created_at: now(),
    })
    ok(res, {
      transferred_quota: transferred,
      balance: user.balance,
    })
    return
  }

  if (path === '/api/v1/payment/config' && req.method === 'GET') {
    ok(res, paymentConfig)
    return
  }

  if (path === '/api/v1/payment/plans' && req.method === 'GET') {
    ok(res, sortedPlans(true))
    return
  }

  if (path === '/api/v1/payment/channels' && req.method === 'GET') {
    ok(res, mockChannels)
    return
  }

  if (path === '/api/v1/payment/limits' && req.method === 'GET') {
    ok(res, {
      methods: clone(methodLimits),
      global_min: 1,
      global_max: 99999,
    })
    return
  }

  if (path === '/api/v1/payment/checkout-info' && req.method === 'GET') {
    ok(res, currentCheckoutInfo())
    return
  }

  if (path === '/api/v1/payment/orders' && req.method === 'POST') {
    const body = await readJson(req)
    const order = buildMockPaymentOrder(req, body)
    mockOrders.unshift(order)
    ok(res, {
      order_id: order.id,
      amount: order.amount,
      pay_amount: order.pay_amount,
      fee_rate: 0,
      expires_at: order.expires_at,
      payment_type: order.payment_type,
      out_trade_no: order.out_trade_no,
      qr_code: qrDataUrl(order.order_type === 'subscription' ? 'Mock Subscription Pay' : 'Mock Recharge Pay'),
      pay_url: `https://example.test/pay/${order.out_trade_no}`,
      currency: order.currency,
    })
    return
  }

  if (path === '/api/v1/payment/orders/my' && req.method === 'GET') {
    const user = currentMockUser(req)
    paginatedFromUrl(res, url, filterPaymentOrders(url, { userId: user.id }).map(sanitizeUserPaymentOrder))
    return
  }

  if (/^\/api\/v1\/payment\/orders\/\d+$/.test(path) && req.method === 'GET') {
    const order = mockOrders.find((item) => item.id === Number(path.split('/').pop()))
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    const user = currentMockUser(req)
    if (order.user_id !== user.id) {
      send(res, 403, { code: 'MOCK_FORBIDDEN', message: 'No permission for this order' })
      return
    }
    ok(res, sanitizeUserPaymentOrder(order))
    return
  }

  if (/^\/api\/v1\/payment\/orders\/\d+\/cancel$/.test(path) && req.method === 'POST') {
    const order = mockOrders.find((item) => item.id === Number(path.split('/')[5]))
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    const user = currentMockUser(req)
    if (order.user_id !== user.id) {
      send(res, 403, { code: 'MOCK_FORBIDDEN', message: 'No permission for this order' })
      return
    }
    if (!cancelMockPaymentOrder(order)) {
      send(res, 409, { code: 'MOCK_CONFLICT', message: 'Order cannot be cancelled' })
      return
    }
    ok(res, { message: 'cancelled' })
    return
  }

  if (/^\/api\/v1\/payment\/orders\/\d+\/refund-request$/.test(path) && req.method === 'POST') {
    const body = await readJson(req)
    const order = mockOrders.find((item) => item.id === Number(path.split('/')[5]))
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    const user = currentMockUser(req)
    if (order.user_id !== user.id) {
      send(res, 403, { code: 'MOCK_FORBIDDEN', message: 'No permission for this order' })
      return
    }
    if (order.status !== 'COMPLETED' || order.order_type !== 'balance') {
      send(res, 400, { code: 'MOCK_INVALID_STATUS', message: 'Order does not allow refund request' })
      return
    }
    order.status = 'REFUND_REQUESTED'
    order.refund_requested_at = now()
    order.refund_requested_by = String(user.id)
    order.refund_request_reason = String(body.reason || '').trim()
    order.refund_amount = order.amount
    order.updated_at = now()
    ok(res, { message: 'refund requested' })
    return
  }

  if (path === '/api/v1/payment/orders/refund-eligible-providers' && req.method === 'GET') {
    ok(res, {
      provider_instance_ids: mockPaymentProviders
        .filter((provider) => provider.refund_enabled && provider.allow_user_refund)
        .map((provider) => String(provider.id)),
    })
    return
  }

  if (path === '/api/v1/payment/orders/verify' && req.method === 'POST') {
    const body = await readJson(req)
    const order = mockOrders.find((item) => item.out_trade_no === body.out_trade_no) || null
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    const user = currentMockUser(req)
    if (order.user_id !== user.id) {
      send(res, 403, { code: 'MOCK_FORBIDDEN', message: 'No permission for this order' })
      return
    }
    ok(res, sanitizeUserPaymentOrder(order))
    return
  }

  if (path === '/api/v1/payment/public/orders/verify' && req.method === 'POST') {
    const body = await readJson(req)
    const order = mockOrders.find((item) => item.out_trade_no === body.out_trade_no) || null
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    ok(res, sanitizePublicPaymentOrder(order))
    return
  }

  if (path === '/api/v1/payment/public/orders/resolve' && req.method === 'POST') {
    ok(res, sanitizePublicPaymentOrder(mockOrders[0]) || null)
    return
  }

  if (path === '/api/v1/subscriptions' && req.method === 'GET') {
    ok(res, clone(mockSubscriptions))
    return
  }

  if (path === '/api/v1/subscriptions/active' && req.method === 'GET') {
    ok(res, clone(activeSubscriptions()))
    return
  }

  if (path === '/api/v1/subscriptions/progress' && req.method === 'GET') {
    ok(res, activeSubscriptions().map((subscription) => ({
      subscription: clone(subscription),
      progress: subscriptionProgress(subscription),
    })))
    return
  }

  if (path === '/api/v1/subscriptions/summary' && req.method === 'GET') {
    const items = activeSubscriptions()
    ok(res, {
      active_count: items.length,
      subscriptions: items.map(subscriptionSummaryItem),
    })
    return
  }

  if (path === '/api/v1/accounts' && req.method === 'GET') {
    const user = currentMockUser(req)
    paginatedFromUrl(res, url, filterSocialAccounts(url, { userOnly: true, userId: user.id }).map(userSafeSocialAccount))
    return
  }

  if (path === '/api/v1/usage' && req.method === 'GET') {
    const user = currentMockUser(req)
    paginatedFromUrl(res, url, filterUsageLogs(url, user.id).map(usageLogProjection))
    return
  }

  if (path === '/api/v1/usage/stats' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, usageStatsProjection(filterUsageLogs(url, user.id)))
    return
  }

  if (path === '/api/v1/usage/dashboard/stats' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, userDashboardUsageStats(user.id))
    return
  }

  if (path === '/api/v1/usage/dashboard/trend' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, userUsageTrend(user.id, String(url.searchParams.get('granularity') || 'day').trim().toLowerCase()))
    return
  }

  if (/^\/api\/v1\/usage\/\d+$/.test(path) && req.method === 'GET') {
    const user = currentMockUser(req)
    const id = Number(path.split('/').pop())
    const log = userUsageLogs(user.id).find((item) => item.id === id)
    if (!log) {
      send(res, 404, { code: 'USAGE_NOT_FOUND', message: 'usage record not found' })
      return
    }
    ok(res, usageLogProjection(log))
    return
  }

  if (path === '/api/v1/accounts/batch-import' && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const accounts = Array.isArray(body.accounts) ? body.accounts : []
    const seen = new Set()
    const created = []
    const errors = []
    const items = []
    let duplicates = 0
    let failed = 0
    for (const account of accounts) {
      if (!isCompleteWorkbenchImportAccount(account)) {
        failed += 1
        errors.push('account could not be imported')
        items.push({
          name: String(account?.name || '').trim(),
          status: 'failed',
          reason: 'invalid_input',
          error: 'account could not be imported',
        })
        continue
      }
      const key = mockSocialAccountDedupKey(account)
      if (key && seen.has(key)) {
        duplicates += 1
        errors.push('account could not be imported')
        items.push({
          name: String(account?.name || '').trim(),
          status: 'duplicate',
          reason: 'duplicate_in_batch',
          error: 'account could not be imported',
        })
        continue
      }
      if (key) seen.add(key)
      const createdAccount = createMockWorkbenchImportAccount(account, user.id)
      created.push(createdAccount)
      items.push({
        id: createdAccount.id,
        name: createdAccount.name,
        status: 'succeeded',
      })
    }
    ok(res, {
      total: accounts.length,
      succeeded: created.length,
      imported: created.length,
      skipped: Math.max(0, accounts.length - created.length),
      failed,
      duplicates,
      errors,
      items,
      accounts: created.map(userSafeSocialAccount),
    })
    return
  }

  if (path === '/api/v1/task-settings/templates' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, clone(mockTaskTemplatesForUser(user.id)))
    return
  }

  if (path === '/api/v1/task-settings/templates' && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const result = saveMockTaskTemplate(user.id, body)
    if (!result.template) {
      send(res, 400, { code: 'TASK_TEMPLATE_INVALID', message: result.validation.errors.join('; ') || 'invalid task template', data: result.validation })
      return
    }
    ok(res, clone(result.template))
    return
  }

  if (path === '/api/v1/task-settings/templates/validate' && req.method === 'POST') {
    const input = normalizeMockTaskTemplateInput(await readJson(req))
    ok(res, validateMockTaskTemplateInput(input))
    return
  }

  if (/^\/api\/v1\/task-settings\/templates\/[^/]+\/copy$/.test(path) && req.method === 'POST') {
    const user = currentMockUser(req)
    const id = decodeURIComponent(path.split('/').at(-2))
    const template = findMockTaskTemplate(user.id, id)
    if (!template) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock task template not found' })
      return
    }
    const result = saveMockTaskTemplate(user.id, { ...template, id: '', name: `${template.name} Copy`, is_default: false })
    ok(res, clone(result.template))
    return
  }

  if (/^\/api\/v1\/task-settings\/templates\/[^/]+\/default$/.test(path) && req.method === 'POST') {
    const user = currentMockUser(req)
    const id = decodeURIComponent(path.split('/').at(-2))
    const template = findMockTaskTemplate(user.id, id)
    if (!template) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock task template not found' })
      return
    }
    for (const item of mockTaskTemplatesForUser(user.id)) {
      if (item.type === template.type) item.is_default = false
    }
    template.is_default = true
    template.updated_at = now()
    ok(res, clone(template))
    return
  }

  if (/^\/api\/v1\/task-settings\/templates\/[^/]+$/.test(path) && req.method === 'DELETE') {
    const user = currentMockUser(req)
    const id = decodeURIComponent(path.split('/').pop())
    const templates = mockTaskTemplatesForUser(user.id)
    const before = templates.length
    mockTaskTemplatesByUser.set(Number(user.id), templates.filter((template) => template.id !== id))
    if (before === mockTaskTemplatesForUser(user.id).length) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock task template not found' })
      return
    }
    ok(res, null)
    return
  }

  if (/^\/api\/v1\/accounts\/\d+$/.test(path) && req.method === 'PUT') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const id = Number(path.split('/').pop())
    const account = findSocialAccount(id)
    if (!account || account.assigned_user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock assigned account not found' })
      return
    }
    ok(res, userSafeSocialAccount(updateMockUserSocialAccount(account, body)))
    return
  }

  if (/^\/api\/v1\/accounts\/\d+$/.test(path) && req.method === 'DELETE') {
    const user = currentMockUser(req)
    const id = Number(path.split('/').pop())
    const account = findSocialAccount(id)
    if (!account || account.assigned_user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock assigned account not found' })
      return
    }
    account.assigned_user_id = null
    account.updated_at = now()
    ok(res, null)
    return
  }

  if (path === '/api/v1/accounts/batch-delete' && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number).filter(Boolean) : []
    let removed = 0
    for (const id of ids) {
      const account = findSocialAccount(id)
      if (account && account.assigned_user_id === user.id) {
        account.assigned_user_id = null
        account.updated_at = now()
        removed += 1
      }
    }
    ok(res, { total: ids.length, removed, skipped: Math.max(0, ids.length - removed), errors: [] })
    return
  }

  if (path === '/api/v1/accounts/export' && req.method === 'GET') {
    const user = currentMockUser(req)
    sendText(res, 200, mockCsvPayload(filterSocialAccounts(url, { userOnly: true, userId: user.id })), 'text/csv; charset=utf-8')
    return
  }

  if (path === userTaskPath && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const templateId = String(body.template_id || '').trim()
    if (!templateId) {
      send(res, 400, { code: 'TASK_TEMPLATE_REQUIRED', message: 'task template is required' })
      return
    }
    const template = findMockTaskTemplate(user.id, templateId)
    if (!template) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock task template not found' })
      return
    }
    const scopedIds = Array.isArray(body.account_ids)
      ? body.account_ids.map(Number).filter((id) => findSocialAccount(id)?.assigned_user_id === user.id)
      : []
    if (scopedIds.length !== (Array.isArray(body.account_ids) ? body.account_ids.length : 0)) {
      send(res, 403, { code: 'ACCOUNT_SCOPE_DENIED', message: 'Account is outside the current user scope' })
      return
    }
    const unavailableAccount = scopedIds
      .map(findSocialAccount)
      .find((account) => account?.account_status !== 'available')
    if (unavailableAccount) {
      send(res, 400, { code: 'SOCIAL_ACCOUNT_NOT_AVAILABLE', message: 'account is not available for execution' })
      return
    }
    const missingDefaultProxy = scopedIds
      .map(findSocialAccount)
      .find((account) => account?.account_status === 'available' && !isMockAccountDefaultProxyUsableForUser(account, user.id))
    if (missingDefaultProxy) {
      send(res, 400, { code: 'SOCIAL_IP_NOT_AVAILABLE', message: 'default social IP is required for execution' })
      return
    }
    const logs = createMockTaskLogs({
      ...body,
      account_ids: scopedIds,
      action: template.type,
      target: '',
      content: '',
      target_pool: template.params?.targets || [],
      content_pool: template.params?.contents || [],
    }, { admin: false })
    ok(res, {
      submitted: logs.length,
      enqueued: logs.length,
      failed_closed: 0,
      logs,
    })
    return
  }

  if (/^\/api\/v1\/accounts\/\d+\/default-proxy$/.test(path) && req.method === 'PUT') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const id = Number(path.split('/')[4])
    const account = findSocialAccount(id)
    if (!account || account.assigned_user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock assigned account not found' })
      return
    }
    if (body.proxy_id != null && !isMockProxyUsableForUser(findMockProxy(body.proxy_id), user.id)) {
      send(res, 400, { code: 'PROXY_UNAVAILABLE', message: 'proxy must be online and belong to current user' })
      return
    }
    updateMockSocialAccount(account, { proxy_id: body.proxy_id })
    ok(res, userSafeSocialAccount(account))
    return
  }

  if (path === '/api/v1/accounts/default-proxy' && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const ids = Array.isArray(body.account_ids) ? body.account_ids.map(Number).filter(Boolean) : []
    const mode = String(body.mode || '')
    const onlineProxies = filterMockProxies(url, { userId: user.id }).filter((proxy) => isMockProxyUsableForUser(proxy, user.id))
    const result = { total: ids.length, succeeded: 0, skipped: 0, failed: 0, items: [] }
    let randomIndex = 0
    if (!['specific', 'random', 'clear'].includes(mode)) {
      send(res, 400, { code: 'SOCIAL_IP_ASSIGNMENT_MODE_INVALID', message: 'proxy assignment mode is invalid' })
      return
    }
    if (mode === 'specific') {
      const proxyID = Number(body.proxy_id)
      if (!Number.isFinite(proxyID) || proxyID <= 0) {
        send(res, 400, { code: 'SOCIAL_IP_REQUIRED', message: 'proxy is required for this assignment' })
        return
      }
      if (!isMockProxyUsableForUser(findMockProxy(proxyID), user.id)) {
        send(res, 400, { code: 'PROXY_UNAVAILABLE', message: 'proxy must be online and belong to current user' })
        return
      }
    }
    if (mode === 'random' && onlineProxies.length === 0) {
      send(res, 400, { code: 'PROXY_UNAVAILABLE', message: 'no online proxies available' })
      return
    }
    for (const id of ids) {
      const account = findSocialAccount(id)
      if (!account || account.assigned_user_id !== user.id) {
        result.failed += 1
        result.items.push({ id, status: 'failed', reason: 'account is outside current user scope' })
        continue
      }
      if (mode === 'clear') {
        result.items.push(assignMockDefaultProxy(account, null))
      } else if (mode === 'random') {
        const proxy = onlineProxies[randomIndex % onlineProxies.length]
        randomIndex += 1
        result.items.push(assignMockDefaultProxy(account, proxy.id))
      } else {
        result.items.push(assignMockDefaultProxy(account, Number(body.proxy_id)))
      }
      result.succeeded += 1
    }
    ok(res, result)
    return
  }

  if (path === '/api/v1/proxies' && req.method === 'GET') {
    const user = currentMockUser(req)
    paginatedFromUrl(res, url, filterMockProxies(url, { userId: user.id }))
    return
  }

  if (path === '/api/v1/proxies/usable' && req.method === 'GET') {
    const user = currentMockUser(req)
    ok(res, filterMockProxies(url, { userId: user.id }).filter((proxy) => proxy.status === 'online' && String(proxy.endpoint || '').trim() !== ''))
    return
  }

  if (path === '/api/v1/proxies' && req.method === 'POST') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const result = createMockProxy(body, user.id)
    if (result.error) {
      send(res, 400, result.error)
      return
    }
    ok(res, clone(result.proxy))
    return
  }

  if (/^\/api\/v1\/proxies\/\d+$/.test(path) && req.method === 'PUT') {
    const user = currentMockUser(req)
    const body = await readJson(req)
    const proxy = findMockProxy(path.split('/').pop())
    if (!proxy || proxy.user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock proxy not found' })
      return
    }
    const result = updateMockProxy(proxy, body)
    if (result.error) {
      send(res, 400, result.error)
      return
    }
    ok(res, clone(result.proxy))
    return
  }

  if (/^\/api\/v1\/proxies\/\d+$/.test(path) && req.method === 'DELETE') {
    const user = currentMockUser(req)
    const proxy = findMockProxy(path.split('/').pop())
    if (!proxy || proxy.user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock proxy not found' })
      return
    }
    deleteMockProxy(proxy.id)
    ok(res, null)
    return
  }

  if (/^\/api\/v1\/proxies\/\d+\/test$/.test(path) && req.method === 'POST') {
    const user = currentMockUser(req)
    const proxy = findMockProxy(path.split('/')[4])
    if (!proxy || proxy.user_id !== user.id) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock proxy not found' })
      return
    }
    ok(res, testMockProxy(proxy))
    return
  }

  if (path === '/api/v1/proxies/test' && req.method === 'POST') {
    const user = currentMockUser(req)
    ok(res, filterMockProxies(url, { userId: user.id }).map(testMockProxy))
    return
  }

  if (path === '/api/v1/admin/accounts' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterSocialAccounts(url))
    return
  }

  if (path === '/api/v1/admin/total-accounts' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterSocialAccounts(url, { totalPoolOnly: true }))
    return
  }

  if (path === '/api/v1/admin/accounts/stats' && req.method === 'GET') {
    const poolAccounts = mockSocialAccounts.filter((account) => !isWorkbenchStagingAccount(account))
    ok(res, {
      total: poolAccounts.length,
      stored: poolAccounts.filter((account) => account.task_status === 'stored').length,
      available: poolAccounts.filter((account) => account.account_status === 'available').length,
    })
    return
  }

  if (path === '/api/v1/admin/accounts' && req.method === 'POST') {
    const body = await readJson(req)
    ok(res, clone(createMockSocialAccount(body)))
    return
  }

  if (path === '/api/v1/admin/accounts/import' && req.method === 'POST') {
    const file = await readMultipartFile(req)
    if (!file) {
      send(res, 400, { code: 'MOCK_IMPORT_FILE_REQUIRED', message: 'file is required' })
      return
    }
    try {
      ok(res, importMockTotalPoolAccounts(parseMockAdminImportFile(file)))
    } catch (error) {
      send(res, 400, { code: 'MOCK_IMPORT_PARSE_FAILED', message: `invalid import file: ${error?.message || 'parse failed'}` })
    }
    return
  }

  if (path === '/api/v1/admin/accounts/export' && req.method === 'GET') {
    sendText(res, 200, mockCsvPayload(filterSocialAccounts(url)), 'text/csv; charset=utf-8')
    return
  }

  if (path === '/api/v1/admin/accounts/batch-delete' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number).filter(Boolean) : []
    const before = mockSocialAccounts.length
    mockSocialAccounts = mockSocialAccounts.filter((account) => !ids.includes(account.id))
    const deleted = before - mockSocialAccounts.length
    ok(res, { deleted })
    return
  }

  if (path === '/api/v1/admin/total-accounts/batch-assign' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number) : []
    const userId = Number(body.user_id)
    const result = mockSocialAccountBatchResult(ids)
    for (const id of ids) {
      const account = findSocialAccount(id)
      if (!account || !userId || account.assigned_user_id) {
        mockSocialAccountBatchSkip(result, id, account?.name, 'already_assigned')
        continue
      }
      account.assigned_user_id = userId
      account.task_status = 'stored'
      account.updated_at = now()
      mockSocialAccountBatchSuccess(result, account)
    }
    ok(res, result)
    return
  }

  if (path === '/api/v1/admin/total-accounts/batch-reclaim' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number) : []
    const result = mockSocialAccountBatchResult(ids)
    for (const id of ids) {
      const account = findSocialAccount(id)
      if (!account) {
        mockSocialAccountBatchSkip(result, id, '', 'not_found')
        continue
      }
      account.assigned_user_id = null
      account.default_proxy_snapshot = ''
      account.updated_at = now()
      mockSocialAccountBatchSuccess(result, account)
    }
    ok(res, result)
    return
  }

  if (path === '/api/v1/admin/total-accounts/batch-delete' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number) : []
    const result = mockSocialAccountBatchResult(ids)
    for (const id of ids) {
      const index = mockSocialAccounts.findIndex((account) => account.id === id)
      if (index < 0) {
        mockSocialAccountBatchSkip(result, id, '', 'not_found')
        continue
      }
      const [account] = mockSocialAccounts.splice(index, 1)
      mockSocialAccountBatchSuccess(result, account)
    }
    ok(res, result)
    return
  }

  if (path === adminTaskPath && req.method === 'POST') {
    const body = await readJson(req)
    const logs = createMockTaskLogs(body, { admin: true })
    ok(res, {
      submitted: logs.length,
      enqueued: logs.length,
      failed_closed: 0,
      logs,
    })
    return
  }

  if (/^\/api\/v1\/admin\/accounts\/\d+$/.test(path) && req.method === 'GET') {
    const account = findSocialAccount(path.split('/').pop())
    if (!account) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock account not found' })
      return
    }
    ok(res, clone(account))
    return
  }

  if (/^\/api\/v1\/admin\/accounts\/\d+$/.test(path) && req.method === 'PUT') {
    const body = await readJson(req)
    const account = findSocialAccount(path.split('/').pop())
    if (!account) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock account not found' })
      return
    }
    ok(res, clone(updateMockSocialAccount(account, body)))
    return
  }

  if (/^\/api\/v1\/admin\/accounts\/\d+$/.test(path) && req.method === 'DELETE') {
    const id = Number(path.split('/').pop())
    const before = mockSocialAccounts.length
    mockSocialAccounts = mockSocialAccounts.filter((account) => account.id !== id)
    ok(res, { deleted: before - mockSocialAccounts.length })
    return
  }

  if (/^\/api\/v1\/admin\/total-accounts\/\d+\/assign$/.test(path) && req.method === 'POST') {
    const body = await readJson(req)
    const account = findSocialAccount(path.split('/')[5])
    if (!account) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock account not found' })
      return
    }
    account.assigned_user_id = Number(body.user_id || adminUser.id)
    account.task_status = 'stored'
    account.updated_at = now()
    ok(res, clone(account))
    return
  }

  if (/^\/api\/v1\/admin\/total-accounts\/\d+\/reclaim$/.test(path) && req.method === 'POST') {
    const account = findSocialAccount(path.split('/')[5])
    if (!account) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock account not found' })
      return
    }
    account.assigned_user_id = null
    account.updated_at = now()
    ok(res, clone(account))
    return
  }

  if (path === '/api/v1/admin/users' && req.method === 'GET') {
    paginatedFromUrl(res, url, [adminUser, regularUser])
    return
  }

  if (path === '/api/v1/admin/groups' && req.method === 'GET') {
    let groups = [...mockGroups]
    const platform = url.searchParams.get('platform')
    const status = url.searchParams.get('status')
    const subscriptionType = url.searchParams.get('subscription_type')
    if (platform) groups = groups.filter((group) => group.platform === platform)
    if (status) groups = groups.filter((group) => group.status === status)
    if (subscriptionType) groups = groups.filter((group) => group.subscription_type === subscriptionType)
    paginatedFromUrl(res, url, groups)
    return
  }

  if (/^\/api\/v1\/admin\/groups\/\d+$/.test(path) && req.method === 'GET') {
    const group = findGroup(path.split('/').pop())
    if (!group) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock group not found' })
      return
    }
    ok(res, clone(group))
    return
  }

  if (path === '/api/v1/admin/system/check-updates' && req.method === 'GET') {
    ok(res, {
      current_version: 'dev-mock',
      latest_version: 'dev-mock',
      has_update: false,
      cached: true,
      build_type: 'source',
      warning: '',
      release_info: null,
    })
    return
  }

  if (path === '/api/v1/admin/system/version' && req.method === 'GET') {
    ok(res, { version: 'dev-mock' })
    return
  }

  if (path === '/api/v1/admin/announcements' && req.method === 'GET') {
    let items = [...mockAnnouncements]
    const status = url.searchParams.get('status')
    const search = (url.searchParams.get('search') || '').toLowerCase()
    if (status) items = items.filter((item) => item.status === status)
    if (search) items = items.filter((item) => item.title.toLowerCase().includes(search) || item.content.toLowerCase().includes(search))
    paginatedFromUrl(res, url, items.map(mockAdminAnnouncementForResponse))
    return
  }

  if (path === '/api/v1/admin/announcements' && req.method === 'POST') {
    const body = await readJson(req)
    const announcement = mockAnnouncementFromCreateBody(body)
    mockAnnouncements.unshift(announcement)
    ok(res, mockAdminAnnouncementForResponse(announcement))
    return
  }

  if (/^\/api\/v1\/admin\/announcements\/\d+\/read-status$/.test(path) && req.method === 'GET') {
    const id = Number(path.split('/')[5])
    const announcement = mockAnnouncements.find((item) => item.id === id)
    if (!announcement) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock announcement not found' })
      return
    }
    const search = (url.searchParams.get('search') || '').toLowerCase()
    let rows = mockAnnouncementReadStatusRows(announcement)
    if (search) {
      rows = rows.filter((item) => item.email.toLowerCase().includes(search) || item.username.toLowerCase().includes(search))
    }
    paginatedFromUrl(res, url, rows)
    return
  }

  if (/^\/api\/v1\/admin\/announcements\/\d+$/.test(path)) {
    const id = Number(path.split('/').pop())
    const announcement = mockAnnouncements.find((item) => item.id === id)
    if (!announcement) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock announcement not found' })
      return
    }
    if (req.method === 'GET') {
      ok(res, mockAdminAnnouncementForResponse(announcement))
      return
    }
    if (req.method === 'PUT') {
      applyMockAnnouncementUpdate(announcement, await readJson(req))
      ok(res, mockAdminAnnouncementForResponse(announcement))
      return
    }
    if (req.method === 'DELETE') {
      mockAnnouncements = mockAnnouncements.filter((item) => item.id !== id)
      ok(res, { message: 'Deleted' })
      return
    }
  }

  if (path === '/api/v1/admin/subscriptions' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterSubscriptions(url))
    return
  }

  if (/^\/api\/v1\/admin\/subscriptions\/\d+$/.test(path) && req.method === 'GET') {
    const subscription = findSubscription(path.split('/').pop())
    if (!subscription) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock subscription not found' })
      return
    }
    ok(res, clone(subscription))
    return
  }

  if (/^\/api\/v1\/admin\/subscriptions\/\d+\/progress$/.test(path) && req.method === 'GET') {
    const subscription = findSubscription(path.split('/')[5])
    if (!subscription) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock subscription not found' })
      return
    }
    ok(res, subscriptionProgress(subscription))
    return
  }

  if (path === '/api/v1/admin/subscriptions/assign' && req.method === 'POST') {
    const body = await readJson(req)
    const subscription =
      body.plan_id != null
        ? createSubscriptionFromPlan(Number(body.user_id || adminUser.id), Number(body.plan_id), Number(body.validity_days || 30))
        : createSubscriptionFromGroup(Number(body.user_id || adminUser.id), Number(body.group_id), Number(body.validity_days || 30))
    if (!subscription) {
      send(res, 400, { code: 'MOCK_BAD_REQUEST', message: 'Invalid plan or group for mock subscription creation' })
      return
    }
    subscription.notes = String(body.notes || '')
    ok(res, clone(subscription))
    return
  }

  if (path === '/api/v1/admin/subscriptions/bulk-assign' && req.method === 'POST') {
    const body = await readJson(req)
    const userIds = Array.isArray(body.user_ids) ? [...new Set(body.user_ids.map((id) => Number(id)).filter((id) => id > 0))] : []
    const subscriptions = userIds
      .map((userId) =>
        body.plan_id != null
          ? createSubscriptionFromPlan(userId, Number(body.plan_id), Number(body.validity_days || 30))
          : createSubscriptionFromGroup(userId, Number(body.group_id), Number(body.validity_days || 30))
      )
      .filter(Boolean)
      .map((subscription) => ({ ...subscription, notes: String(body.notes || '') }))
    ok(res, {
      success_count: subscriptions.length,
      created_count: subscriptions.length,
      reused_count: 0,
      failed_count: 0,
      subscriptions: clone(subscriptions),
      errors: [],
    })
    return
  }

  if (/^\/api\/v1\/admin\/subscriptions\/\d+\/extend$/.test(path) && req.method === 'POST') {
    const body = await readJson(req)
    const subscription = findSubscription(path.split('/')[5])
    if (!subscription) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock subscription not found' })
      return
    }
    const days = Number(body.days || 0)
    const baseTime = subscription.expires_at ? new Date(subscription.expires_at).getTime() : Date.now()
    subscription.expires_at = new Date(baseTime + days * 24 * 60 * 60 * 1000).toISOString()
    subscription.updated_at = now()
    if (new Date(subscription.expires_at).getTime() > Date.now() && subscription.status !== 'revoked') {
      subscription.status = 'active'
    }
    ok(res, clone(subscription))
    return
  }

  if (/^\/api\/v1\/admin\/subscriptions\/\d+\/reset-quota$/.test(path) && req.method === 'POST') {
    const subscription = findSubscription(path.split('/')[5])
    if (!subscription) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock subscription not found' })
      return
    }
    subscription.daily_usage_usd = 0
    subscription.weekly_usage_usd = 0
    subscription.monthly_usage_usd = 0
    subscription.daily_window_start = now()
    subscription.weekly_window_start = now()
    subscription.monthly_window_start = now()
    subscription.updated_at = now()
    ok(res, clone(subscription))
    return
  }

  if (/^\/api\/v1\/admin\/subscriptions\/\d+$/.test(path) && req.method === 'DELETE') {
    const subscription = findSubscription(path.split('/').pop())
    if (!subscription) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock subscription not found' })
      return
    }
    subscription.status = 'revoked'
    subscription.updated_at = now()
    ok(res, { message: 'Subscription revoked' })
    return
  }

  if (/^\/api\/v1\/admin\/users\/\d+\/subscriptions$/.test(path) && req.method === 'GET') {
    const userId = Number(path.split('/')[5])
    paginatedFromUrl(res, url, mockSubscriptions.filter((subscription) => subscription.user_id === userId))
    return
  }

  if (/^\/api\/v1\/admin\/groups\/\d+\/subscriptions$/.test(path) && req.method === 'GET') {
    const groupId = Number(path.split('/')[5])
    paginatedFromUrl(res, url, mockSubscriptions.filter((subscription) => subscription.group_id === groupId))
    return
  }

  if (path === '/api/v1/admin/payment/config' && req.method === 'GET') {
    ok(res, adminPaymentConfig)
    return
  }

  if (path === '/api/v1/admin/payment/config' && req.method === 'PUT') {
    const body = await readJson(req)
    Object.assign(adminPaymentConfig, body || {})
    ok(res, clone(adminPaymentConfig))
    return
  }

  if (path === '/api/v1/admin/payment/dashboard' && req.method === 'GET') {
    ok(res, paymentDashboardStats(url.searchParams.get('days')))
    return
  }

  if (path === '/api/v1/admin/payment/orders' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterPaymentOrders(url).map(sanitizeAdminPaymentOrder))
    return
  }

  if (/^\/api\/v1\/admin\/payment\/orders\/\d+$/.test(path) && req.method === 'GET') {
    const order = findPaymentOrderByID(path.split('/')[6])
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    ok(res, {
      order: sanitizeAdminPaymentOrder(order),
      auditLogs: [],
    })
    return
  }

  if (/^\/api\/v1\/admin\/payment\/orders\/\d+\/cancel$/.test(path) && req.method === 'POST') {
    const order = findPaymentOrderByID(path.split('/')[6])
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    if (!cancelMockPaymentOrder(order)) {
      send(res, 409, { code: 'MOCK_CONFLICT', message: 'Order cannot be cancelled' })
      return
    }
    ok(res, { message: 'cancelled' })
    return
  }

  if (/^\/api\/v1\/admin\/payment\/orders\/\d+\/retry$/.test(path) && req.method === 'POST') {
    const order = findPaymentOrderByID(path.split('/')[6])
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    order.updated_at = now()
    ok(res, { message: 'fulfillment retried' })
    return
  }

  if (/^\/api\/v1\/admin\/payment\/orders\/\d+\/refund$/.test(path) && req.method === 'POST') {
    const body = await readJson(req)
    const order = findPaymentOrderByID(path.split('/')[6])
    if (!order) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock order not found' })
      return
    }
    if (!['COMPLETED', 'REFUND_REQUESTED', 'REFUND_FAILED'].includes(order.status)) {
      send(res, 400, { code: 'MOCK_INVALID_STATUS', message: 'Order status does not allow refund' })
      return
    }
    const refundAmount = roundMoney(body.amount || order.amount)
    order.status = 'REFUNDED'
    order.refund_amount = refundAmount
    order.refund_reason = String(body.reason || '').trim()
    order.refund_at = now()
    order.force_refund = body.force === true
    order.updated_at = now()
    ok(res, { success: true, balance_deducted: body.deduct_balance ? refundAmount : 0 })
    return
  }

  if (path === '/api/v1/admin/payment/plans' && req.method === 'GET') {
    ok(res, sortedPlans(false))
    return
  }

  if (path === '/api/v1/admin/payment/providers' && req.method === 'GET') {
    ok(res, clone(mockPaymentProviders))
    return
  }

  if (path === '/api/v1/admin/payment/providers' && req.method === 'POST') {
    const body = await readJson(req)
    const provider = {
      id: nextProviderId++,
      provider_key: String(body.provider_key || 'alipay'),
      name: String(body.name || 'Mock Provider'),
      config: body.config || {},
      supported_types: Array.isArray(body.supported_types) ? body.supported_types : [],
      enabled: body.enabled !== false,
      payment_mode: String(body.payment_mode || ''),
      refund_enabled: body.refund_enabled === true,
      allow_user_refund: body.allow_user_refund === true,
      limits: String(body.limits || ''),
      sort_order: mockPaymentProviders.length + 1,
      created_at: now(),
      updated_at: now(),
    }
    mockPaymentProviders.push(provider)
    ok(res, clone(provider))
    return
  }

  if (/^\/api\/v1\/admin\/payment\/providers\/\d+$/.test(path) && req.method === 'PUT') {
    const body = await readJson(req)
    const provider = mockPaymentProviders.find((item) => item.id === Number(path.split('/').pop()))
    if (!provider) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock provider not found' })
      return
    }
    Object.assign(provider, body || {}, { updated_at: now() })
    ok(res, clone(provider))
    return
  }

  if (/^\/api\/v1\/admin\/payment\/providers\/\d+$/.test(path) && req.method === 'DELETE') {
    const id = Number(path.split('/').pop())
    mockPaymentProviders = mockPaymentProviders.filter((provider) => provider.id !== id)
    ok(res, { message: 'Deleted' })
    return
  }

  if (path === '/api/v1/admin/payment/plans' && req.method === 'POST') {
    const body = await readJson(req)
    const quotaInput = resolvePlanQuotaInput(body, { required: true })
    if (quotaInput.error) {
      send(res, 400, quotaInput.error)
      return
    }
    const group = findGroup(body.group_id) || findGroup(body.platform === 'instagram' ? 2 : 1)
    const plan = {
      id: nextPlanId++,
      group_id: group?.id || 1,
      platform: body.platform || group?.platform || 'x_twitter',
      name: String(body.name || 'New Mock Plan').trim(),
      description: String(body.description || '').trim(),
      price: Number(body.price || 0),
      original_price: body.original_price == null ? null : Number(body.original_price || 0),
      validity_days: Number(body.validity_days || 30),
      validity_unit: String(body.validity_unit || 'days'),
      quota_usd: quotaInput.value,
      daily_limit_usd: normalizePositiveLimit(body.daily_limit_usd),
      weekly_limit_usd: normalizePositiveLimit(body.weekly_limit_usd),
      monthly_limit_usd: quotaInput.value,
      features: String(body.features || '')
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean),
      product_name: String(body.product_name || body.name || 'New Mock Plan').trim(),
      for_sale: body.for_sale !== false,
      sort_order: Number(body.sort_order || 0),
    }
    mockPlans.push(plan)
    ok(res, decoratePlan(plan))
    return
  }

  if (/^\/api\/v1\/admin\/payment\/plans\/\d+$/.test(path) && req.method === 'PUT') {
    const body = await readJson(req)
    const plan = mockPlans.find((item) => item.id === Number(path.split('/').pop()))
    if (!plan) {
      send(res, 404, { code: 'MOCK_NOT_FOUND', message: 'Mock plan not found' })
      return
    }
    const quotaInput = resolvePlanQuotaInput(body)
    if (quotaInput.error) {
      send(res, 400, quotaInput.error)
      return
    }
    if (body.features !== undefined) {
      plan.features = Array.isArray(body.features)
        ? body.features.map((item) => String(item).trim()).filter(Boolean)
        : String(body.features || '').split('\n').map((item) => item.trim()).filter(Boolean)
    }
    const fields = [
      'group_id',
      'platform',
      'name',
      'description',
      'price',
      'original_price',
      'validity_days',
      'validity_unit',
      'quota_usd',
      'daily_limit_usd',
      'weekly_limit_usd',
      'monthly_limit_usd',
      'product_name',
      'for_sale',
      'sort_order',
    ]
    for (const field of fields) {
      if (body[field] !== undefined) {
        plan[field] = body[field]
      }
    }
    if (quotaInput.touched) {
      plan.quota_usd = quotaInput.value
      plan.monthly_limit_usd = quotaInput.value
    }
    plan.daily_limit_usd = normalizePositiveLimit(plan.daily_limit_usd)
    plan.weekly_limit_usd = normalizePositiveLimit(plan.weekly_limit_usd)
    plan.monthly_limit_usd = normalizePositiveLimit(plan.monthly_limit_usd ?? plan.quota_usd)
    plan.quota_usd = normalizePositiveLimit(plan.quota_usd ?? plan.monthly_limit_usd)
    ok(res, decoratePlan(plan))
    return
  }

  if (/^\/api\/v1\/admin\/payment\/plans\/\d+$/.test(path) && req.method === 'DELETE') {
    const id = Number(path.split('/').pop())
    mockPlans = mockPlans.filter((plan) => plan.id !== id)
    ok(res, { message: 'Deleted' })
    return
  }

  if (path === '/api/v1/admin/redeem-codes' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterRedeemCodes(url).map(redeemCodeForResponse))
    return
  }

  if (path === '/api/v1/admin/redeem-codes/stats' && req.method === 'GET') {
    ok(res, redeemStats())
    return
  }

  if (path === '/api/v1/admin/redeem-codes/export' && req.method === 'GET') {
    sendText(res, 200, redeemCsvPayload(filterRedeemCodes(url)), 'text/csv; charset=utf-8')
    return
  }

  if (path === '/api/v1/admin/redeem-codes/generate' && req.method === 'POST') {
    const body = await readJson(req)
    const count = Math.max(1, Math.min(100, Number(body.count || 1)))
    const generated = Array.from({ length: count }, () => buildMockRedeemCode(body))
    mockRedeemCodes = [...generated, ...mockRedeemCodes]
    ok(res, generated.map(redeemCodeForResponse))
    return
  }

  if (path === '/api/v1/admin/redeem-codes/create-and-redeem' && req.method === 'POST') {
    const body = await readJson(req)
    const userID = Number(body.user_id || 0)
    if (!findMockUser(userID)) {
      send(res, 404, { code: 'USER_NOT_FOUND', message: 'user not found' })
      return
    }
    let code = findRedeemCodeByCode(body.code)
    if (!code) {
      code = {
        ...buildMockRedeemCode({ ...body, count: 1 }),
        code: String(body.code || '').trim(),
      }
      mockRedeemCodes.unshift(code)
    }
    const fakeReq = { ...req, headers: { ...req.headers, authorization: userID === regularUser.id ? 'Bearer dev-mock-user-token' : '' } }
    const result = redeemMockCode(fakeReq, code.code)
    if (result.error && code.used_by !== userID) {
      send(res, result.error.status, result.error.payload)
      return
    }
    ok(res, { redeem_code: redeemCodeForResponse(code) })
    return
  }

  if (path === '/api/v1/admin/redeem-codes/batch-delete' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number).filter(Boolean) : []
    const before = mockRedeemCodes.length
    mockRedeemCodes = mockRedeemCodes.filter((code) => !ids.includes(code.id))
    ok(res, {
      deleted: before - mockRedeemCodes.length,
      message: 'Redeem codes deleted successfully',
    })
    return
  }

  if (path === '/api/v1/admin/redeem-codes/batch-update' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.ids) ? body.ids.map(Number).filter(Boolean) : []
    const fields = body.fields || {}
    let updated = 0
    for (const code of mockRedeemCodes) {
      if (!ids.includes(code.id)) continue
      if (fields.status !== undefined) code.status = String(fields.status)
      if (fields.expires_at !== undefined) code.expires_at = fields.expires_at ? new Date(fields.expires_at).toISOString() : null
      if (fields.notes !== undefined) code.notes = String(fields.notes || '')
      if (fields.group_id !== undefined) code.group_id = fields.group_id == null ? null : Number(fields.group_id)
      if (fields.plan_id !== undefined) code.plan_id = fields.plan_id == null ? null : Number(fields.plan_id)
      code.updated_at = now()
      updated += 1
    }
    ok(res, {
      updated,
      message: 'Redeem codes updated successfully',
    })
    return
  }

  if (/^\/api\/v1\/admin\/redeem-codes\/\d+\/expire$/.test(path) && req.method === 'POST') {
    const code = findRedeemCodeByID(path.split('/')[5])
    if (!code) {
      send(res, 404, { code: 'REDEEM_CODE_NOT_FOUND', message: 'redeem code not found' })
      return
    }
    code.status = 'expired'
    code.expires_at = daysAgo(1)
    code.updated_at = now()
    ok(res, redeemCodeForResponse(code))
    return
  }

  if (/^\/api\/v1\/admin\/redeem-codes\/\d+$/.test(path)) {
    const code = findRedeemCodeByID(path.split('/').pop())
    if (!code) {
      send(res, 404, { code: 'REDEEM_CODE_NOT_FOUND', message: 'redeem code not found' })
      return
    }
    if (req.method === 'GET') {
      ok(res, redeemCodeForResponse(code))
      return
    }
    if (req.method === 'DELETE') {
      mockRedeemCodes = mockRedeemCodes.filter((item) => item.id !== code.id)
      ok(res, { message: 'Redeem code deleted successfully' })
      return
    }
  }

  if (path === '/api/v1/admin/promo-codes' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterPromoCodes(url).map(clone))
    return
  }

  if (path === '/api/v1/admin/promo-codes' && req.method === 'POST') {
    const body = await readJson(req)
    const promo = buildMockPromoCode(body)
    mockPromoCodes.unshift(promo)
    ok(res, clone(promo))
    return
  }

  if (/^\/api\/v1\/admin\/promo-codes\/\d+\/usages$/.test(path) && req.method === 'GET') {
    const promoID = Number(path.split('/')[5])
    if (!findPromoCodeByID(promoID)) {
      send(res, 404, { code: 'PROMO_CODE_NOT_FOUND', message: 'promo code not found' })
      return
    }
    paginatedFromUrl(res, url, mockPromoUsages
      .filter((usage) => usage.promo_code_id === promoID)
      .map(promoUsageForResponse))
    return
  }

  if (/^\/api\/v1\/admin\/promo-codes\/\d+$/.test(path)) {
    const promo = findPromoCodeByID(path.split('/').pop())
    if (!promo) {
      send(res, 404, { code: 'PROMO_CODE_NOT_FOUND', message: 'promo code not found' })
      return
    }
    if (req.method === 'GET') {
      ok(res, clone(promo))
      return
    }
    if (req.method === 'PUT') {
      ok(res, clone(updateMockPromoCode(promo, await readJson(req))))
      return
    }
    if (req.method === 'DELETE') {
      mockPromoCodes = mockPromoCodes.filter((item) => item.id !== promo.id)
      mockPromoUsages = mockPromoUsages.filter((usage) => usage.promo_code_id !== promo.id)
      ok(res, { message: 'Promo code deleted successfully' })
      return
    }
  }

  if (path === '/api/v1/admin/affiliates/users' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterAffiliateAdminEntries(url))
    return
  }

  if (path === '/api/v1/admin/affiliates/users/lookup' && req.method === 'GET') {
    const keyword = String(url.searchParams.get('q') || '').trim().toLowerCase()
    if (!keyword) {
      ok(res, [])
      return
    }
    ok(res, [adminUser, regularUser]
      .filter((user) => [user.email, user.username].some((value) => String(value || '').toLowerCase().includes(keyword)))
      .map((user) => ({ id: user.id, email: user.email, username: user.username })))
    return
  }

  if (path === '/api/v1/admin/affiliates/users/batch-rate' && req.method === 'POST') {
    const body = await readJson(req)
    const ids = Array.isArray(body.user_ids) ? body.user_ids.map(Number).filter((id) => id > 0) : []
    for (const id of ids) {
      const profile = ensureAffiliateProfile(id)
      profile.aff_rebate_rate_percent = body.clear === true ? null : Number(body.aff_rebate_rate_percent || 0)
      profile.updated_at = now()
    }
    ok(res, { affected: ids.length })
    return
  }

  if (/^\/api\/v1\/admin\/affiliates\/users\/\d+\/overview$/.test(path) && req.method === 'GET') {
    const userID = Number(path.split('/')[6])
    ok(res, affiliateOverviewForUser(userID))
    return
  }

  if (/^\/api\/v1\/admin\/affiliates\/users\/\d+$/.test(path)) {
    const userID = Number(path.split('/').pop())
    const profile = ensureAffiliateProfile(userID)
    if (req.method === 'PUT') {
      const body = await readJson(req)
      if (body.aff_code !== undefined) {
        profile.aff_code = normalizeMockCode(body.aff_code)
        profile.aff_code_custom = true
      }
      if (body.clear_rebate_rate === true) {
        profile.aff_rebate_rate_percent = null
      } else if (body.aff_rebate_rate_percent !== undefined) {
        profile.aff_rebate_rate_percent = body.aff_rebate_rate_percent == null ? null : Number(body.aff_rebate_rate_percent)
      }
      profile.updated_at = now()
      ok(res, { user_id: userID })
      return
    }
    if (req.method === 'DELETE') {
      profile.aff_code = `AFF${String(userID).padStart(6, '0')}`
      profile.aff_code_custom = false
      profile.aff_rebate_rate_percent = null
      profile.updated_at = now()
      ok(res, { user_id: userID })
      return
    }
  }

  if (path === '/api/v1/admin/affiliates/invites' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterAffiliateRecords(url, affiliateInviteRecords()))
    return
  }

  if (path === '/api/v1/admin/affiliates/rebates' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterAffiliateRecords(url, affiliateRebateRecords()))
    return
  }

  if (path === '/api/v1/admin/affiliates/transfers' && req.method === 'GET') {
    paginatedFromUrl(res, url, filterAffiliateRecords(url, mockAffiliateTransfers))
    return
  }

  if (path === '/api/v1/admin/dashboard/stats' && req.method === 'GET') {
    ok(res, adminDashboardStats())
    return
  }

  if (path === '/api/v1/admin/dashboard/trend' && req.method === 'GET') {
    ok(res, usageTrendFromLogs(mockSocialTaskLogs, String(url.searchParams.get('granularity') || 'day').trim().toLowerCase()))
    return
  }

  if (path === '/api/v1/admin/dashboard/users-trend' && req.method === 'GET') {
    const limit = Math.max(1, Math.min(50, Number(url.searchParams.get('limit') || 20) || 20))
    ok(res, adminUserUsageTrend(String(url.searchParams.get('granularity') || 'day').trim().toLowerCase(), limit))
    return
  }

  if (path === '/api/v1/admin/dashboard/users-ranking' && req.method === 'GET') {
    const limit = Math.max(1, Math.min(50, Number(url.searchParams.get('limit') || 20) || 20))
    ok(res, adminUserSpendingRanking(limit))
    return
  }

  send(res, 404, { code: 'MOCK_NOT_FOUND', message: `Mock API has no handler for ${req.method} ${path}` })
})

server.listen(port, '0.0.0.0', () => {
  console.log(`[mock-api] listening on http://localhost:${port}`)
  console.log(`[mock-api] admin ${adminEmail} / ${adminPassword}`)
})

