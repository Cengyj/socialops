<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.settings.title') }}</h1>

      <form class="space-y-6" novalidate @submit.prevent="saveSettings">
        <div class="border-b border-gray-200 dark:border-dark-700">
          <nav class="flex flex-wrap gap-4" role="tablist">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              type="button"
              class="border-b-2 px-3 py-2 text-sm font-medium transition-colors"
              :class="activeTab === tab.key
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'"
              @click="activeTab = tab.key"
            >
              {{ tab.label }}
            </button>
          </nav>
        </div>

        <div v-show="activeTab === 'general'" class="space-y-4">
          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.general.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.siteName') }}
                <input v-model="form.site_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.siteUrl') }}
                <input v-model="form.api_base_url" type="text" class="input mt-1" />
              </label>
            </div>
            <div class="pt-2">
              <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.siteLogo') }}
              </p>
              <ImageUpload
                v-model="siteLogo"
                mode="image"
                :max-size="300 * 1024"
                :upload-label="t('admin.settings.site.uploadImage')"
                :remove-label="t('admin.settings.site.remove')"
                :hint="t('admin.settings.general.siteLogoHint')"
              />
            </div>
          </section>
        </div>

        <div v-show="activeTab === 'security'" class="space-y-4">
          <section class="card p-6 space-y-5">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.wechatConnect.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.wechatConnect.description') }}</p>
            </div>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.wechat_connect_enabled" type="checkbox" class="checkbox" />
              {{ t('admin.settings.wechatConnect.enabledLabel') }}
            </label>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.wechatConnect.appIdLabel') }}
                <input
                  v-model="form.wechat_connect_mp_app_id"
                  data-testid="wechat-connect-mp-app-id"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.wechatConnect.appSecretLabel') }}
                <input
                  v-model="form.wechat_connect_mp_app_secret"
                  data-testid="wechat-connect-mp-app-secret"
                  type="password"
                  class="input mt-1"
                  :placeholder="wechatMpSecretConfigured
                    ? t('admin.settings.wechatConnect.appSecretConfiguredPlaceholder')
                    : t('admin.settings.wechatConnect.appSecretPlaceholder')"
                />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.wechat_connect_open_enabled"
                  data-testid="wechat-connect-open-enabled"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.wechatConnect.openModeLabel') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.wechat_connect_mp_enabled"
                  data-testid="wechat-connect-mp-enabled"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.wechatConnect.mpModeLabel') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.wechatConnect.redirectUrlLabel') }}
                <input
                  v-model="form.wechat_connect_redirect_url"
                  data-testid="wechat-connect-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.wechatConnect.frontendRedirectUrlLabel') }}
                <input
                  v-model="form.wechat_connect_frontend_redirect_url"
                  data-testid="wechat-connect-frontend-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">GitHub OAuth</h2>
            <a
              data-testid="github-oauth-apps-guide-link"
              href="https://github.com/settings/developers"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              OAuth Apps
            </a>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.github_oauth_enabled" type="checkbox" class="checkbox" />
              GitHub OAuth
            </label>
          </section>

          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">OIDC</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.oidc_connect_enabled" type="checkbox" class="checkbox" />
                OIDC
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.oidc_connect_use_pkce" type="checkbox" class="checkbox" />
                PKCE
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.oidc_connect_validate_id_token" type="checkbox" class="checkbox" />
                Validate ID token
              </label>
            </div>
          </section>
        </div>

        <div v-show="activeTab === 'users'" class="space-y-4">
          <section class="card p-6 space-y-5">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.authSourceDefaults.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.authSourceDefaults.description') }}</p>
            </div>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.force_email_on_third_party_signup" type="checkbox" class="checkbox" />
              {{ t('admin.settings.authSourceDefaults.requireEmailLabel') }}
            </label>
            <div class="grid gap-4 md:grid-cols-2">
              <div
                v-for="source in authSourceTypes"
                :key="source"
                class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
              >
                <label class="flex items-center gap-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <input
                    v-model="authSourceDefaults[source].grant_on_signup"
                    :data-testid="`auth-source-${source}-enabled`"
                    type="checkbox"
                    class="checkbox"
                  />
                  {{ t(`admin.settings.authSourceDefaults.sources.${source}.title`) }}
                </label>
                <div
                  v-if="authSourceDefaults[source].grant_on_signup"
                  :data-testid="`auth-source-${source}-panel`"
                  class="mt-4 space-y-3"
                >
                  <label class="block text-sm text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.defaultBalance') }}
                    <input v-model.number="authSourceDefaults[source].balance" type="number" class="input mt-1" />
                  </label>
                  <label class="block text-sm text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.defaultConcurrency') }}
                    <input v-model.number="authSourceDefaults[source].concurrency" type="number" min="1" class="input mt-1" />
                  </label>
                  <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="authSourceDefaults[source].grant_on_first_bind" type="checkbox" class="checkbox" />
                    {{ t('admin.settings.authSourceDefaults.grantOnFirstBindLabel') }}
                  </label>
                </div>
              </div>
            </div>
          </section>
        </div>

        <div v-show="activeTab === 'social'" class="space-y-4">
          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.social.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.social.taskWorkers') }}
                <input v-model.number="form.task_executor_workers" type="number" min="1" max="20" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.social.taskInterval') }}
                <input v-model.number="form.task_min_interval_ms" type="number" min="500" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.social.ipCheckInterval') }}
                <input v-model.number="form.ip_check_interval_minutes" type="number" min="5" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.social.maxAccountsPerUser') }}
                <input v-model.number="form.max_accounts_per_user" type="number" min="1" class="input mt-1" />
              </label>
            </div>
          </section>
        </div>

        <div v-show="activeTab === 'registration'" class="space-y-4">
          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.registration.title') }}</h2>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.registration_enabled" type="checkbox" class="checkbox" />
              {{ t('admin.settings.registration.enabled') }}
            </label>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.email_verify_enabled" type="checkbox" class="checkbox" />
              {{ t('admin.settings.registration.emailVerify') }}
            </label>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.invitation_code_enabled" type="checkbox" class="checkbox" />
              {{ t('admin.settings.registration.invitationCode') }}
            </label>
          </section>
        </div>

        <div v-show="activeTab === 'payment'" class="space-y-4">
          <section class="card p-6 space-y-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.payment.title') }}</h2>
              <div class="flex flex-wrap gap-3 text-sm">
                <a
                  href="https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                >
                  {{ t('admin.settings.payment.configGuide') }}
                </a>
                <a
                  href="https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md#支持的支付方式"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                >
                  {{ t('admin.settings.payment.findProvider') }}
                </a>
              </div>
            </div>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.payment_enabled" type="checkbox" class="checkbox" />
              {{ t('admin.settings.payment.enabled') }}
            </label>
            <ImageUpload
              v-model="paymentHelpImageUrl"
              :upload-label="t('admin.settings.site.uploadImage')"
              :remove-label="t('admin.settings.site.remove')"
              :placeholder="t('admin.settings.payment.helpImagePlaceholder')"
            />
          </section>

          <PaymentProviderList
            :providers="paymentProviders"
            :loading="paymentProvidersLoading"
            :can-create="true"
            :enabled-payment-types="form.payment_enabled_types"
            :all-payment-types="paymentTypeOptions"
            redirect-label=""
            @refresh="loadPaymentProviders"
            @toggle-field="toggleProviderField"
          />
        </div>

        <div class="flex justify-end">
          <button class="btn btn-primary" :disabled="saving || loading" type="submit">
            <span v-if="saving">{{ t('common.saving') }}</span>
            <span v-else>{{ t('common.save') }}</span>
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import PaymentProviderList from '@/components/payment/PaymentProviderList.vue'
import { adminAPI } from '@/api'
import {
  appendAuthSourceDefaultsToUpdateRequest,
  buildAuthSourceDefaultsState,
  resolveWeChatConnectModeCapabilities,
  type AuthSourceDefaultsState,
  type AuthSourceType,
  type SystemSettings,
  type UpdateSettingsRequest,
} from '@/api/admin/settings'
import type { ProviderInstance } from '@/types/payment'
import type { TypeOption } from '@/components/payment/providerConfig'

type TabKey = 'general' | 'security' | 'users' | 'social' | 'registration' | 'payment'

const { t } = useI18n()

const activeTab = ref<TabKey>('general')
const tabs = [
  { key: 'general' as const, label: t('admin.settings.tabs.general') },
  { key: 'security' as const, label: t('admin.settings.tabs.security') },
  { key: 'users' as const, label: t('admin.settings.tabs.users') },
  { key: 'social' as const, label: t('admin.settings.tabs.social') },
  { key: 'registration' as const, label: t('admin.settings.tabs.registration') },
  { key: 'payment' as const, label: t('admin.settings.tabs.payment') },
]

const authSourceTypes: AuthSourceType[] = ['email', 'linuxdo', 'oidc', 'wechat', 'github', 'google', 'dingtalk']

const form = reactive<UpdateSettingsRequest & {
  task_executor_workers: number
  task_min_interval_ms: number
  ip_check_interval_minutes: number
  max_accounts_per_user: number
  wechat_connect_mp_app_secret?: string
  payment_enabled_types: string[]
}>({
  site_name: '',
  api_base_url: '',
  task_executor_workers: 3,
  task_min_interval_ms: 2000,
  ip_check_interval_minutes: 30,
  max_accounts_per_user: 10,
  registration_enabled: true,
  email_verify_enabled: false,
  invitation_code_enabled: false,
  force_email_on_third_party_signup: false,
  wechat_connect_enabled: false,
  wechat_connect_app_id: '',
  wechat_connect_mp_app_id: '',
  wechat_connect_mp_app_secret: '',
  wechat_connect_open_enabled: true,
  wechat_connect_mp_enabled: false,
  wechat_connect_mobile_enabled: false,
  wechat_connect_mode: 'open',
  wechat_connect_scopes: '',
  wechat_connect_redirect_url: '',
  wechat_connect_frontend_redirect_url: '/auth/wechat/callback',
  oidc_connect_enabled: false,
  oidc_connect_use_pkce: true,
  oidc_connect_validate_id_token: true,
  github_oauth_enabled: false,
  payment_enabled: false,
  payment_enabled_types: [],
  payment_help_image_url: '',
  payment_help_text: '',
  site_logo: '',
})

const authSourceDefaults = reactive<AuthSourceDefaultsState>(buildAuthSourceDefaultsState({}))
const loading = ref(false)
const saving = ref(false)
const wechatMpSecretConfigured = ref(false)
const paymentProviders = ref<ProviderInstance[]>([])
const paymentProvidersLoading = ref(false)
const paymentHelpImageUrl = computed({
	get: () => form.payment_help_image_url || '',
	set: (value: string) => {
		form.payment_help_image_url = value
	},
})
const siteLogo = computed<string>({
	get: () => form.site_logo || '',
	set: (value: string) => {
		form.site_logo = value
	},
})
const paymentTypeOptions: TypeOption[] = [
  { value: 'alipay', label: 'Alipay' },
  { value: 'wxpay', label: 'WeChat Pay' },
  { value: 'stripe', label: 'Stripe' },
  { value: 'airwallex', label: 'Airwallex' },
]

onMounted(async () => {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getSettings()
    applySettings(settings)
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    loading.value = false
  }
  await loadPaymentProviders()
})

function applySettings(settings: Partial<SystemSettings>) {
  const wechatModes = resolveWeChatConnectModeCapabilities(
    settings.wechat_connect_open_enabled,
    settings.wechat_connect_mp_enabled,
    settings.wechat_connect_mobile_enabled,
    settings.wechat_connect_mode,
  )

  Object.assign(form, {
    site_name: settings.site_name || '',
    site_logo: settings.site_logo || '',
    api_base_url: settings.api_base_url || '',
    registration_enabled: settings.registration_enabled ?? true,
    email_verify_enabled: settings.email_verify_enabled ?? false,
    invitation_code_enabled: settings.invitation_code_enabled ?? false,
    force_email_on_third_party_signup: settings.force_email_on_third_party_signup ?? false,
    wechat_connect_enabled: settings.wechat_connect_enabled ?? false,
    wechat_connect_app_id: settings.wechat_connect_app_id || '',
    wechat_connect_mp_app_id: settings.wechat_connect_mp_app_id || settings.wechat_connect_app_id || '',
    wechat_connect_mp_app_secret: '',
    wechat_connect_open_enabled: wechatModes.openEnabled,
    wechat_connect_mp_enabled: wechatModes.mpEnabled,
    wechat_connect_mobile_enabled: wechatModes.mobileEnabled,
    wechat_connect_mode: settings.wechat_connect_mode || 'open',
    wechat_connect_scopes: settings.wechat_connect_scopes || '',
    wechat_connect_redirect_url: settings.wechat_connect_redirect_url || '',
    wechat_connect_frontend_redirect_url: settings.wechat_connect_frontend_redirect_url || '/auth/wechat/callback',
    oidc_connect_enabled: settings.oidc_connect_enabled ?? false,
    oidc_connect_use_pkce: settings.oidc_connect_use_pkce ?? true,
    oidc_connect_validate_id_token: settings.oidc_connect_validate_id_token ?? true,
    github_oauth_enabled: settings.github_oauth_enabled ?? false,
    payment_enabled: settings.payment_enabled ?? false,
    payment_enabled_types: settings.payment_enabled_types || [],
    payment_help_image_url: settings.payment_help_image_url || '',
    payment_help_text: settings.payment_help_text || '',
  })
  wechatMpSecretConfigured.value =
    settings.wechat_connect_mp_app_secret_configured ?? settings.wechat_connect_app_secret_configured ?? false

  Object.assign(authSourceDefaults, buildAuthSourceDefaultsState(settings))
}

async function loadPaymentProviders() {
  paymentProvidersLoading.value = true
  try {
    const response = await adminAPI.payment.getProviders()
    paymentProviders.value = Array.isArray(response) ? response : response.data || []
  } catch (error) {
    console.error('Failed to load payment providers:', error)
    paymentProviders.value = []
  } finally {
    paymentProvidersLoading.value = false
  }
}

async function toggleProviderField(provider: ProviderInstance, field: 'enabled' | 'refund_enabled' | 'allow_user_refund') {
  await adminAPI.payment.updateProvider(provider.id, { [field]: !provider[field] })
  await loadPaymentProviders()
}

function buildPayload(): UpdateSettingsRequest {
  const payload: UpdateSettingsRequest = {
    site_name: form.site_name,
    site_logo: form.site_logo,
    api_base_url: form.api_base_url,
    registration_enabled: form.registration_enabled,
    email_verify_enabled: form.email_verify_enabled,
    invitation_code_enabled: form.invitation_code_enabled,
    force_email_on_third_party_signup: form.force_email_on_third_party_signup,
    wechat_connect_enabled: form.wechat_connect_enabled,
    wechat_connect_app_id: form.wechat_connect_mp_app_id || form.wechat_connect_app_id,
    wechat_connect_mp_app_id: form.wechat_connect_mp_app_id,
    wechat_connect_open_enabled: form.wechat_connect_open_enabled,
    wechat_connect_mp_enabled: form.wechat_connect_mp_enabled,
    wechat_connect_mobile_enabled: form.wechat_connect_mobile_enabled,
    wechat_connect_redirect_url: form.wechat_connect_redirect_url,
    wechat_connect_frontend_redirect_url: form.wechat_connect_frontend_redirect_url,
    oidc_connect_enabled: form.oidc_connect_enabled,
    oidc_connect_use_pkce: form.oidc_connect_use_pkce,
    oidc_connect_validate_id_token: form.oidc_connect_validate_id_token,
    github_oauth_enabled: form.github_oauth_enabled,
    payment_enabled: form.payment_enabled,
    payment_enabled_types: form.payment_enabled_types,
    payment_help_image_url: form.payment_help_image_url,
    payment_help_text: form.payment_help_text,
  }

  if (form.wechat_connect_mp_app_secret) {
    payload.wechat_connect_mp_app_secret = form.wechat_connect_mp_app_secret
  }

  return appendAuthSourceDefaultsToUpdateRequest(payload, authSourceDefaults)
}

async function saveSettings() {
  saving.value = true
  try {
    const updated = await adminAPI.settings.updateSettings(buildPayload())
    if (form.wechat_connect_mp_app_secret) {
      form.wechat_connect_mp_app_secret = ''
      wechatMpSecretConfigured.value = true
    }
    if (updated) {
      applySettings({ ...form, ...updated })
    }
  } catch (error) {
    console.error('Failed to save settings:', error)
  } finally {
    saving.value = false
  }
}
</script>
