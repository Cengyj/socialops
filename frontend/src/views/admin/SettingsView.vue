<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div
        v-if="settingsLoadError"
        class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
      >
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span>{{ settingsLoadError }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadSettings">
            {{ t('common.retry') }}
          </button>
        </div>
      </div>

      <div class="space-y-6">
        <div class="settings-tabs-shell">
          <nav class="settings-tabs-scroll" role="tablist" :aria-label="t('admin.settings.title')">
            <div class="settings-tabs">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                :id="`settings-tab-${tab.key}`"
                type="button"
                role="tab"
                :class="['settings-tab', activeTab === tab.key && 'settings-tab-active']"
                :aria-selected="activeTab === tab.key"
                :aria-controls="`settings-panel-${tab.key}`"
                :tabindex="activeTab === tab.key ? 0 : -1"
                @click="selectTab(tab.key)"
                @keydown="handleTabKeydown($event, tab.key)"
              >
                <span class="settings-tab-icon">
                  <Icon :name="tab.icon" size="sm" />
                </span>
                <span class="settings-tab-label">{{ tab.label }}</span>
              </button>
            </div>
          </nav>
        </div>

        <div v-if="activeTab === 'general'" id="settings-panel-general" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-general">
          <section class="card p-6 space-y-5">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.general.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.siteName') }}
                <input v-model="form.site_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.siteSubtitle') }}
                <input v-model="form.site_subtitle" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.frontendUrl') }}
                <input v-model="form.frontend_url" type="url" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.apiBaseUrl') }}
                <input v-model="form.api_base_url" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.docUrl') }}
                <input v-model="form.doc_url" type="url" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.contactInfo') }}
                <input v-model="form.contact_info" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.tableDefaultPageSize') }}
                <input v-model.number="form.table_default_page_size" type="number" min="1" max="500" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.general.tablePageSizeOptions') }}
                <input v-model="tablePageSizeOptionsInput" type="text" class="input mt-1" />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.backend_mode_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.general.backendMode') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.general.homeContent') }}
                <textarea v-model="form.home_content" rows="4" class="input mt-1 resize-y"></textarea>
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

          <section class="card p-6 space-y-5">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.customMenu.title') }}</h2>
              <button type="button" class="btn btn-secondary btn-sm" @click="addCustomMenuItem">
                <Icon name="plus" size="sm" />
                {{ t('admin.settings.customMenu.add') }}
              </button>
            </div>
            <div v-if="form.custom_menu_items.length === 0" class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
              {{ t('admin.settings.customMenu.empty') }}
            </div>
            <div v-else class="space-y-3">
              <div
                v-for="(item, index) in form.custom_menu_items"
                :key="item.id || `menu-${index}`"
                class="grid gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-[1fr_1fr_1.2fr_120px_auto]"
              >
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customMenu.label') }}
                  <input v-model="item.label" type="text" class="input mt-1" />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customMenu.pageSlug') }}
                  <input v-model="item.page_slug" type="text" class="input mt-1" placeholder="help/intro" />
                  <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.customMenu.pageSlugHint') }}
                  </span>
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customMenu.url') }}
                  <input v-model="item.url" type="text" class="input mt-1" />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customMenu.visibility') }}
                  <select v-model="item.visibility" class="input mt-1">
                    <option value="user">{{ t('admin.settings.customMenu.user') }}</option>
                    <option value="admin">{{ t('admin.settings.customMenu.admin') }}</option>
                  </select>
                </label>
                <button type="button" class="btn btn-secondary self-end px-3" @click="removeCustomMenuItem(index)">
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </section>

          <section class="card p-6 space-y-5">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.customEndpoints.title') }}</h2>
              <button type="button" class="btn btn-secondary btn-sm" @click="addCustomEndpoint">
                <Icon name="plus" size="sm" />
                {{ t('admin.settings.customEndpoints.add') }}
              </button>
            </div>
            <div v-if="form.custom_endpoints.length === 0" class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
              {{ t('admin.settings.customEndpoints.empty') }}
            </div>
            <div v-else class="space-y-3">
              <div
                v-for="(endpoint, index) in form.custom_endpoints"
                :key="`${endpoint.name}-${index}`"
                class="grid gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-[1fr_1.3fr_1fr_auto]"
              >
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customEndpoints.name') }}
                  <input v-model="endpoint.name" type="text" class="input mt-1" />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customEndpoints.endpoint') }}
                  <input v-model="endpoint.endpoint" type="url" class="input mt-1" />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.customEndpoints.description') }}
                  <input v-model="endpoint.description" type="text" class="input mt-1" />
                </label>
                <button type="button" class="btn btn-secondary self-end px-3" @click="removeCustomEndpoint(index)">
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </section>
        </div>

        <div v-if="activeTab === 'agreement'" id="settings-panel-agreement" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-agreement">
          <section class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.loginAgreement.title') }}</h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.loginAgreement.description') }}</p>
                </div>
                <div class="flex items-center gap-3">
                  <span class="text-sm text-gray-600 dark:text-gray-300">
                    {{ form.login_agreement_enabled ? t('common.enabled') : t('common.disabled') }}
                  </span>
                  <label class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center">
                    <span class="sr-only">{{ t('admin.settings.loginAgreement.enabled') }}</span>
                    <input v-model="form.login_agreement_enabled" type="checkbox" class="peer sr-only" />
                    <span class="absolute inset-0 rounded-full bg-gray-200 transition-colors duration-200 peer-checked:bg-primary-600 peer-focus-visible:ring-2 peer-focus-visible:ring-primary-500 peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-white dark:bg-dark-600 dark:peer-focus-visible:ring-offset-dark-800"></span>
                    <span class="pointer-events-none absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform duration-200 peer-checked:translate-x-5"></span>
                  </label>
                </div>
              </div>
            </div>

            <div class="space-y-6 p-6">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.loginAgreement.mode') }}
                </label>
                <div class="grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-700">
                  <button
                    type="button"
                    class="inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
                    :class="form.login_agreement_mode === 'modal'
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                    @click="form.login_agreement_mode = 'modal'"
                  >
                    <Icon name="shield" size="sm" />
                    {{ t('admin.settings.loginAgreement.modal') }}
                  </button>
                  <button
                    type="button"
                    class="inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
                    :class="form.login_agreement_mode === 'checkbox'
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                    @click="form.login_agreement_mode = 'checkbox'"
                  >
                    <Icon name="checkCircle" size="sm" />
                    {{ t('admin.settings.loginAgreement.checkbox') }}
                  </button>
                </div>
              </div>

              <div>
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ t('admin.settings.loginAgreement.documents') }}
                    </h3>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.loginAgreement.documentsHint') }}
                    </p>
                  </div>
                  <button type="button" class="btn btn-primary btn-sm inline-flex items-center gap-1.5" @click="addLoginAgreementDocument">
                    <Icon name="plus" size="sm" />
                    {{ t('admin.settings.loginAgreement.add') }}
                  </button>
                </div>

                <div v-if="form.login_agreement_documents.length === 0" class="mt-4 rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                  {{ t('admin.settings.loginAgreement.empty') }}
                </div>
                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="(document, index) in form.login_agreement_documents"
                    :key="document.id || `agreement-${index}`"
                    class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800/60"
                  >
                    <div class="mb-3 flex items-center justify-between gap-3">
                      <div class="flex min-w-0 items-center gap-3">
                        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                          <Icon name="document" size="sm" />
                        </span>
                        <div class="min-w-0">
                          <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                            {{ document.title || t('admin.settings.loginAgreement.untitled') }}
                          </p>
                          <p class="truncate text-xs text-gray-500 dark:text-gray-400">
                            {{ document.id }}
                          </p>
                        </div>
                      </div>
                      <button type="button" class="rounded-md p-2 text-red-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" @click="removeLoginAgreementDocument(index)">
                        <Icon name="trash" size="sm" />
                      </button>
                    </div>
                    <div class="space-y-3">
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('admin.settings.loginAgreement.documentTitle') }}
                        <input v-model="document.title" type="text" class="input mt-1" />
                      </label>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('admin.settings.loginAgreement.content') }}
                        <textarea v-model="document.content_md" rows="6" class="input mt-1 resize-y font-mono text-sm"></textarea>
                      </label>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>

        <div v-if="activeTab === 'security'" id="settings-panel-security" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-security">
          <section class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.registration.title') }}</h2>
            </div>
            <div class="grid gap-4 p-6 md:grid-cols-2">
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
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.password_reset_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.registration.passwordReset') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.promo_code_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.registration.promoCode') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.registration.emailSuffixWhitelist') }}
                <input v-model="emailSuffixWhitelistInput" type="text" class="input mt-1" />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-5">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.security.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.totp_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.security.totp') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.turnstile_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.turnstile.enabled') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.turnstile.siteKey') }}
                <input v-model="form.turnstile_site_key" type="text" class="input mt-1" />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-turnstile"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.turnstile.secretKey') }}
                  <input
                    v-model="form.turnstile_secret_key"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="turnstileSecretConfigured
                      ? t('admin.settings.secretConfiguredPlaceholder')
                      : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
            </div>
          </section>

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
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-wechat-connect"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.wechatConnect.appSecretLabel') }}
                  <input
                    v-model="form.wechat_connect_mp_app_secret"
                    data-testid="wechat-connect-mp-app-secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="wechatMpSecretConfigured
                      ? t('admin.settings.wechatConnect.appSecretConfiguredPlaceholder')
                      : t('admin.settings.wechatConnect.appSecretPlaceholder')"
                  />
                </label>
              </form>
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
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.linuxdo.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.linuxdo.description') }}</p>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.linuxdo_connect_enabled"
                  data-testid="linuxdo-connect-enabled"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.linuxdo.enabled') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.linuxdo.clientId') }}
                <input
                  v-model="form.linuxdo_connect_client_id"
                  data-testid="linuxdo-connect-client-id"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-linuxdo-connect"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.linuxdo.clientSecret') }}
                  <input
                    v-model="form.linuxdo_connect_client_secret"
                    data-testid="linuxdo-connect-client-secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="linuxdoSecretConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.linuxdo.redirectUrl') }}
                <input
                  v-model="form.linuxdo_connect_redirect_url"
                  data-testid="linuxdo-connect-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.oauth.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.github_oauth_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.oauth.github') }}
              </label>
              <a
                data-testid="github-oauth-apps-guide-link"
                href="https://github.com/settings/developers"
                target="_blank"
                rel="noopener noreferrer"
                class="self-center text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                OAuth Apps
              </a>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oauth.githubClientId') }}
                <input v-model="form.github_oauth_client_id" type="text" class="input mt-1" />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-github-oauth"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.oauth.githubSecret') }}
                  <input
                    v-model="form.github_oauth_client_secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="githubSecretConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oauth.githubRedirect') }}
                <input v-model="form.github_oauth_redirect_url" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oauth.githubFrontendRedirect') }}
                <input v-model="form.github_oauth_frontend_redirect_url" type="text" class="input mt-1" />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.google.title') }}</h2>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.google_oauth_enabled"
                  data-testid="google-oauth-enabled"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.google.enabled') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.google.clientId') }}
                <input
                  v-model="form.google_oauth_client_id"
                  data-testid="google-oauth-client-id"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-google-oauth"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.google.clientSecret') }}
                  <input
                    v-model="form.google_oauth_client_secret"
                    data-testid="google-oauth-client-secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="googleSecretConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.google.redirectUrl') }}
                <input
                  v-model="form.google_oauth_redirect_url"
                  data-testid="google-oauth-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.google.frontendRedirectUrl') }}
                <input
                  v-model="form.google_oauth_frontend_redirect_url"
                  data-testid="google-oauth-frontend-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.dingtalk.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.dingtalk.description') }}</p>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.dingtalk_connect_enabled"
                  data-testid="dingtalk-connect-enabled"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.dingtalk.enabled') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.clientId') }}
                <input
                  v-model="form.dingtalk_connect_client_id"
                  data-testid="dingtalk-connect-client-id"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-dingtalk-connect"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.dingtalk.clientSecret') }}
                  <input
                    v-model="form.dingtalk_connect_client_secret"
                    data-testid="dingtalk-connect-client-secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="dingtalkSecretConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.redirectUrl') }}
                <input
                  v-model="form.dingtalk_connect_redirect_url"
                  data-testid="dingtalk-connect-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.corpRestrictionPolicy') }}
                <select
                  v-model="form.dingtalk_connect_corp_restriction_policy"
                  data-testid="dingtalk-connect-corp-restriction-policy"
                  class="input mt-1"
                >
                  <option value="none">{{ t('common.none') }}</option>
                  <option value="internal_only">{{ t('admin.settings.dingtalk.internalOnly') }}</option>
                </select>
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.internalCorpId') }}
                <input
                  v-model="form.dingtalk_connect_internal_corp_id"
                  data-testid="dingtalk-connect-internal-corp-id"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.dingtalk_connect_bypass_registration"
                  data-testid="dingtalk-connect-bypass-registration"
                  type="checkbox"
                  class="checkbox"
                  :disabled="form.dingtalk_connect_corp_restriction_policy !== 'internal_only'"
                />
                {{ t('admin.settings.dingtalk.bypassRegistration') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.dingtalk_connect_sync_corp_email" type="checkbox" class="checkbox" />
                {{ t('admin.settings.dingtalk.syncCorpEmail') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.dingtalk_connect_sync_display_name" type="checkbox" class="checkbox" />
                {{ t('admin.settings.dingtalk.syncDisplayName') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.dingtalk_connect_sync_dept" type="checkbox" class="checkbox" />
                {{ t('admin.settings.dingtalk.syncDept') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncCorpEmailAttrKey') }}
                <input v-model="form.dingtalk_connect_sync_corp_email_attr_key" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncCorpEmailAttrName') }}
                <input v-model="form.dingtalk_connect_sync_corp_email_attr_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncDisplayNameAttrKey') }}
                <input v-model="form.dingtalk_connect_sync_display_name_attr_key" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncDisplayNameAttrName') }}
                <input v-model="form.dingtalk_connect_sync_display_name_attr_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncDeptAttrKey') }}
                <input v-model="form.dingtalk_connect_sync_dept_attr_key" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.dingtalk.syncDeptAttrName') }}
                <input v-model="form.dingtalk_connect_sync_dept_attr_name" type="text" class="input mt-1" />
              </label>
            </div>
          </section>

          <section class="card p-6 space-y-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.oidc.title') }}</h2>
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
                {{ t('admin.settings.oidc.validateIdToken') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.providerName') }}
                <input v-model="form.oidc_connect_provider_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.clientId') }}
                <input v-model="form.oidc_connect_client_id" type="text" class="input mt-1" />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-oidc-connect"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.oidc.clientSecret') }}
                  <input
                    v-model="form.oidc_connect_client_secret"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="oidcSecretConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.issuerUrl') }}
                <input v-model="form.oidc_connect_issuer_url" type="url" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.scopes') }}
                <input v-model="form.oidc_connect_scopes" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.discoveryUrl') }}
                <input
                  v-model="form.oidc_connect_discovery_url"
                  data-testid="oidc-connect-discovery-url"
                  type="url"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.authorizeUrl') }}
                <input
                  v-model="form.oidc_connect_authorize_url"
                  data-testid="oidc-connect-authorize-url"
                  type="url"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.tokenUrl') }}
                <input
                  v-model="form.oidc_connect_token_url"
                  data-testid="oidc-connect-token-url"
                  type="url"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.userinfoUrl') }}
                <input
                  v-model="form.oidc_connect_userinfo_url"
                  data-testid="oidc-connect-userinfo-url"
                  type="url"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.jwksUrl') }}
                <input
                  v-model="form.oidc_connect_jwks_url"
                  data-testid="oidc-connect-jwks-url"
                  type="url"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.redirectUrl') }}
                <input
                  v-model="form.oidc_connect_redirect_url"
                  data-testid="oidc-connect-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.frontendRedirectUrl') }}
                <input
                  v-model="form.oidc_connect_frontend_redirect_url"
                  data-testid="oidc-connect-frontend-redirect-url"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.tokenAuthMethod') }}
                <select
                  v-model="form.oidc_connect_token_auth_method"
                  data-testid="oidc-connect-token-auth-method"
                  class="input mt-1"
                >
                  <option value="client_secret_post">client_secret_post</option>
                  <option value="client_secret_basic">client_secret_basic</option>
                  <option value="none">none</option>
                </select>
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.allowedSigningAlgs') }}
                <input
                  v-model="form.oidc_connect_allowed_signing_algs"
                  data-testid="oidc-connect-allowed-signing-algs"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.clockSkewSeconds') }}
                <input
                  v-model.number="form.oidc_connect_clock_skew_seconds"
                  data-testid="oidc-connect-clock-skew-seconds"
                  type="number"
                  min="0"
                  class="input mt-1"
                />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input
                  v-model="form.oidc_connect_require_email_verified"
                  data-testid="oidc-connect-require-email-verified"
                  type="checkbox"
                  class="checkbox"
                />
                {{ t('admin.settings.oidc.requireEmailVerified') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.userinfoEmailPath') }}
                <input
                  v-model="form.oidc_connect_userinfo_email_path"
                  data-testid="oidc-connect-userinfo-email-path"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.userinfoIdPath') }}
                <input
                  v-model="form.oidc_connect_userinfo_id_path"
                  data-testid="oidc-connect-userinfo-id-path"
                  type="text"
                  class="input mt-1"
                />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.oidc.userinfoUsernamePath') }}
                <input
                  v-model="form.oidc_connect_userinfo_username_path"
                  data-testid="oidc-connect-userinfo-username-path"
                  type="text"
                  class="input mt-1"
                />
              </label>
            </div>
          </section>

          <section class="card overflow-hidden">
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.adminApiKey.title') }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.adminApiKey.status', { status: adminApiKey.exists ? t('common.enabled') : t('common.disabled') }) }}</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-primary btn-sm" :disabled="adminApiKey.loading" @click="regenerateAdminApiKey">
                  {{ t('admin.settings.adminApiKey.regenerate') }}
                </button>
                <button type="button" class="btn btn-danger btn-sm" :disabled="adminApiKey.loading || !adminApiKey.exists" @click="deleteAdminApiKey">
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
            <div class="grid gap-4 p-6 md:grid-cols-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.adminApiKey.masked') }}
                <input :value="adminApiKey.masked_key || '-'" type="text" class="input mt-1" readonly />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.adminApiKey.generated') }}
                <input :value="adminApiKey.generated" type="text" class="input mt-1 font-mono text-xs" readonly />
              </label>
            </div>
          </section>

        </div>

        <div v-if="activeTab === 'users'" id="settings-panel-users" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-users">
          <section class="card p-6 space-y-5">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.authSourceDefaults.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.authSourceDefaults.description') }}</p>
            </div>
            <div class="grid gap-4 md:grid-cols-3">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.defaultBalance') }}
                <input v-model.number="form.default_balance" type="number" min="0" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.defaultConcurrency') }}
                <input v-model.number="form.default_concurrency" type="number" min="1" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.defaultUserRpmLimit') }}
                <input v-model.number="form.default_user_rpm_limit" type="number" min="0" class="input mt-1" />
              </label>
            </div>
            <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.force_email_on_third_party_signup" type="checkbox" class="checkbox" />
              {{ t('admin.settings.authSourceDefaults.requireEmailLabel') }}
            </label>

            <div v-if="subscriptionPlansError" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
              {{ subscriptionPlansError }}
            </div>

            <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.settings.authSourceDefaults.globalSubscriptionsLabel') }}
                  </h3>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.authSourceDefaults.defaultSubscriptionsHint') }}
                  </p>
                </div>
                <button
                  type="button"
                  class="btn btn-secondary px-3 py-1.5 text-xs"
                  :disabled="subscriptionPlansLoading || subscriptionPlanOptions.length === 0"
                  @click="addGlobalDefaultSubscription"
                >
                  {{ t('admin.settings.authSourceDefaults.addSubscription') }}
                </button>
              </div>
              <div v-if="!form.default_subscriptions.length" class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.authSourceDefaults.noSubscriptions') }}
              </div>
              <div v-else class="mt-3 space-y-2">
                <div
                  v-for="(item, index) in form.default_subscriptions"
                  :key="`global-${index}`"
                  class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_120px_auto]"
                >
                  <Select
                    v-model="item.plan_id"
                    :options="subscriptionPlanOptions"
                    :placeholder="t('admin.settings.authSourceDefaults.packageLabel')"
                    searchable
                    @change="() => syncDefaultSubscriptionPlan(item)"
                  >
                    <template #option="{ option, selected }">
                      <div class="flex min-w-0 items-start justify-between gap-3">
                        <SubscriptionPackageBadge
                          :name="String(option.label)"
                          :platform="String(option.platform || 'social')"
                          :description="String(option.description || '')"
                          :quota-display="String(option.quotaDisplay || '')"
                          :validity-label="String(option.validityLabel || '')"
                          :hidden="option.hidden === true"
                          :hidden-label="t('payment.admin.hidden')"
                          compact
                        />
                        <Icon v-if="selected" name="check" size="sm" class="mt-1 shrink-0 text-primary-600 dark:text-primary-400" />
                      </div>
                    </template>
                  </Select>
                  <input
                    v-model.number="item.validity_days"
                    type="number"
                    min="1"
                    max="36500"
                    class="input"
                    :placeholder="t('admin.settings.authSourceDefaults.validityDaysLabel')"
                  />
                  <button type="button" class="btn btn-secondary px-3" @click="removeDefaultSubscription(form.default_subscriptions, index)">
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </div>
            </div>

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
                  <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                    <div class="mb-2 flex items-center justify-between gap-3">
                      <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('admin.settings.authSourceDefaults.defaultSubscriptionsLabel') }}
                      </span>
                      <button
                        type="button"
                        class="btn btn-secondary px-2 py-1 text-xs"
                        :disabled="subscriptionPlansLoading || subscriptionPlanOptions.length === 0"
                        @click="addDefaultSubscription(authSourceDefaults[source].subscriptions)"
                      >
                        {{ t('admin.settings.authSourceDefaults.addSubscription') }}
                      </button>
                    </div>
                    <div v-if="authSourceDefaults[source].subscriptions.length === 0" class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.authSourceDefaults.noSubscriptions') }}
                    </div>
                    <div v-else class="space-y-2">
                      <div
                        v-for="(item, index) in authSourceDefaults[source].subscriptions"
                        :key="`${source}-${index}`"
                        class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_110px_auto]"
                      >
                        <Select
                          v-model="item.plan_id"
                          :options="subscriptionPlanOptions"
                          :placeholder="t('admin.settings.authSourceDefaults.packageLabel')"
                          searchable
                          @change="() => syncDefaultSubscriptionPlan(item)"
                        />
                        <input
                          v-model.number="item.validity_days"
                          type="number"
                          min="1"
                          max="36500"
                          class="input"
                          :placeholder="t('admin.settings.authSourceDefaults.validityDaysLabel')"
                        />
                        <button type="button" class="btn btn-secondary px-3" @click="removeDefaultSubscription(authSourceDefaults[source].subscriptions, index)">
                          <Icon name="trash" size="sm" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.affiliate.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.affiliate.description') }}</p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex flex-col gap-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <label class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ t('admin.settings.affiliate.enabled') }}
                  </label>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.affiliate.enabledHint') }}
                  </p>
                </div>
                <label class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center self-start sm:self-center">
                  <span class="sr-only">{{ t('admin.settings.affiliate.enabled') }}</span>
                  <input v-model="form.affiliate_enabled" data-testid="affiliate-enabled" type="checkbox" class="peer sr-only" />
                  <span class="absolute inset-0 rounded-full bg-gray-200 transition-colors duration-200 peer-checked:bg-primary-600 peer-focus-visible:ring-2 peer-focus-visible:ring-primary-500 peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-white dark:bg-dark-600 dark:peer-focus-visible:ring-offset-dark-800"></span>
                  <span class="pointer-events-none absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform duration-200 peer-checked:translate-x-5"></span>
                </label>
              </div>
              <div v-if="form.affiliate_enabled" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.rate') }}
                  <input v-model.number="form.affiliate_rebate_rate" type="number" min="0" max="1" step="0.01" class="input mt-1" />
                  <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-gray-400">{{ t('admin.settings.affiliate.rateHint') }}</span>
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.freezeHours') }}
                  <input v-model.number="form.affiliate_rebate_freeze_hours" type="number" min="0" class="input mt-1" />
                  <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-gray-400">{{ t('admin.settings.affiliate.freezeHoursHint') }}</span>
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.durationDays') }}
                  <input v-model.number="form.affiliate_rebate_duration_days" type="number" min="0" class="input mt-1" />
                  <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-gray-400">{{ t('admin.settings.affiliate.durationDaysHint') }}</span>
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.perInviteeCap') }}
                  <input
                    v-model.number="form.affiliate_rebate_per_invitee_cap"
                    data-testid="affiliate-rebate-per-invitee-cap"
                    type="number"
                    min="0"
                    step="0.01"
                    class="input mt-1"
                  />
                  <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-gray-400">{{ t('admin.settings.affiliate.perInviteeCapHint') }}</span>
                </label>
              </div>
            </div>
          </section>
        </div>

        <div v-if="activeTab === 'email'" id="settings-panel-email" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-email">
          <section class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.smtp.title') }}</h2>
            </div>
            <div class="space-y-5 p-6">
              <div class="grid gap-4 md:grid-cols-3">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.host') }}
                <input v-model="form.smtp_host" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.port') }}
                <input v-model.number="form.smtp_port" type="number" min="1" max="65535" class="input mt-1" />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.smtp_use_tls" type="checkbox" class="checkbox" />
                {{ t('admin.settings.smtp.useTls') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.username') }}
                <input v-model="form.smtp_username" type="text" class="input mt-1" />
              </label>
              <form class="contents" autocomplete="off" @submit.prevent="saveSettings">
                <input
                  type="text"
                  autocomplete="username"
                  class="sr-only"
                  tabindex="-1"
                  aria-hidden="true"
                  value="settings-smtp"
                  readonly
                />
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.smtp.password') }}
                  <input
                    v-model="form.smtp_password"
                    type="password"
                    autocomplete="new-password"
                    class="input mt-1"
                    :placeholder="smtpPasswordConfigured ? t('admin.settings.secretConfiguredPlaceholder') : t('admin.settings.secretPlaceholder')"
                  />
                </label>
              </form>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.fromEmail') }}
                <input v-model="form.smtp_from_email" type="email" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.fromName') }}
                <input v-model="form.smtp_from_name" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.smtp.testEmail') }}
                <input v-model="testEmail" type="email" class="input mt-1" />
              </label>
              </div>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="testingSmtp" @click="testSmtpConnection">
                  {{ testingSmtp ? t('admin.settings.smtp.testing') : t('admin.settings.smtp.testConnection') }}
                </button>
                <button type="button" class="btn btn-primary btn-sm" :disabled="sendingTestEmail || !testEmail" @click="sendTestEmail">
                  {{ sendingTestEmail ? t('admin.settings.smtp.sending') : t('admin.settings.smtp.sendTestEmail') }}
                </button>
              </div>
            </div>
          </section>

          <EmailTemplateEditor />

          <section class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.notifications.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.notifications.description') }}</p>
            </div>
            <div class="grid gap-4 p-6 md:grid-cols-2">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.balance_low_notify_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.notifications.balanceLow') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.notifications.balanceThreshold') }}
                <input v-model.number="form.balance_low_notify_threshold" type="number" min="0" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.notifications.rechargeUrl') }}
                <input v-model="form.balance_low_notify_recharge_url" type="url" class="input mt-1" />
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.subscription_expiry_notify_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.notifications.subscriptionExpiry') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.account_quota_notify_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.notifications.accountQuota') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.notifications.accountQuotaEmails') }}
                <input v-model="accountQuotaEmailsInput" type="text" class="input mt-1" />
              </label>
            </div>
          </section>
        </div>

        <div v-if="activeTab === 'payment'" id="settings-panel-payment" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-payment">
          <section class="card p-6 space-y-5">
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
                  href="https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md#supported-payment-methods"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                >
                  {{ t('admin.settings.payment.findProvider') }}
                </a>
              </div>
            </div>
            <div class="grid gap-4 md:grid-cols-3">
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.payment_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.payment.enabled') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.payment_balance_disabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.payment.balanceDisabled') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.purchase_subscription_enabled" type="checkbox" class="checkbox" />
                {{ t('admin.settings.payment.purchaseEntry') }}
              </label>
              <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="form.payment_alipay_force_qrcode" type="checkbox" class="checkbox" />
                {{ t('admin.settings.payment.alipayForceQr') }}
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 md:col-span-2">
                {{ t('admin.settings.payment.purchaseUrl') }}
                <input v-model="form.purchase_subscription_url" type="url" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.minAmount') }}
                <input v-model.number="form.payment_min_amount" type="number" min="0" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.maxAmount') }}
                <input v-model.number="form.payment_max_amount" type="number" min="0" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.dailyLimit') }}
                <input v-model.number="form.payment_daily_limit" type="number" min="0" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.orderTimeout') }}
                <input v-model.number="form.payment_order_timeout_minutes" type="number" min="1" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.maxPendingOrders') }}
                <input v-model.number="form.payment_max_pending_orders" type="number" min="1" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.rechargeMultiplier') }}
                <input v-model.number="form.payment_balance_recharge_multiplier" type="number" min="0" step="0.01" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.rechargeFeeRate') }}
                <input v-model.number="form.payment_recharge_fee_rate" type="number" min="0" step="0.01" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.productPrefix') }}
                <input v-model="form.payment_product_name_prefix" type="text" class="input mt-1" />
              </label>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.payment.productSuffix') }}
                <input v-model="form.payment_product_name_suffix" type="text" class="input mt-1" />
              </label>
            </div>
            <div class="border-t border-gray-200 pt-5 dark:border-dark-700">
              <div class="grid gap-4 md:grid-cols-3">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.payment.loadBalanceStrategy') }}
                  <select
                    v-model="form.payment_load_balance_strategy"
                    data-testid="payment-load-balance-strategy"
                    class="input mt-1"
                  >
                    <option
                      v-for="option in paymentLoadBalanceOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </option>
                  </select>
                </label>
                <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
                  <input
                    v-model="form.payment_cancel_rate_limit_enabled"
                    data-testid="payment-cancel-rate-limit-enabled"
                    type="checkbox"
                    class="checkbox"
                  />
                  {{ t('admin.settings.payment.cancelRateLimitEnabled') }}
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.payment.cancelRateLimitMax') }}
                  <input
                    v-model.number="form.payment_cancel_rate_limit_max"
                    data-testid="payment-cancel-rate-limit-max"
                    type="number"
                    min="1"
                    class="input mt-1"
                  />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.payment.cancelRateLimitWindow') }}
                  <input
                    v-model.number="form.payment_cancel_rate_limit_window"
                    data-testid="payment-cancel-rate-limit-window"
                    type="number"
                    min="1"
                    class="input mt-1"
                  />
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.payment.cancelRateLimitUnit') }}
                  <select
                    v-model="form.payment_cancel_rate_limit_unit"
                    data-testid="payment-cancel-rate-limit-unit"
                    class="input mt-1"
                  >
                    <option
                      v-for="option in paymentCancelRateLimitUnitOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </option>
                  </select>
                </label>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.payment.cancelRateLimitWindowMode') }}
                  <select
                    v-model="form.payment_cancel_rate_limit_window_mode"
                    data-testid="payment-cancel-rate-limit-window-mode"
                    class="input mt-1"
                  >
                    <option
                      v-for="option in paymentCancelRateLimitWindowModeOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </option>
                  </select>
                </label>
              </div>
            </div>
            <div>
              <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.enabledTypes') }}</p>
              <div class="flex flex-wrap gap-3">
                <label
                  v-for="option in paymentTypeOptions"
                  :key="option.value"
                  class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300"
                >
                  <input v-model="form.payment_enabled_types" type="checkbox" class="checkbox" :value="String(option.value)" />
                  {{ option.label }}
                </label>
              </div>
            </div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.payment.helpText') }}
              <textarea v-model="form.payment_help_text" rows="3" class="input mt-1 resize-y"></textarea>
            </label>
            <ImageUpload
              v-model="paymentHelpImageUrl"
              :upload-label="t('admin.settings.site.uploadImage')"
              :remove-label="t('admin.settings.site.remove')"
              :placeholder="t('admin.settings.payment.helpImagePlaceholder')"
            />
          </section>

          <div v-if="paymentProvidersError" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
            {{ paymentProvidersError }}
          </div>

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

        <div v-if="activeTab === 'backup'" id="settings-panel-backup" class="space-y-6" role="tabpanel" aria-labelledby="settings-tab-backup">
          <BackupSettingsSection />
        </div>

        <div v-if="activeTab !== 'backup'" class="flex justify-end">
          <button type="button" class="btn btn-primary" :disabled="saving || loading || Boolean(settingsLoadError)" @click="saveSettings">
            <span v-if="saving">{{ t('common.saving') }}</span>
            <span v-else>{{ t('common.save') }}</span>
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Select from '@/components/common/Select.vue'
import PaymentProviderList from '@/components/payment/PaymentProviderList.vue'
import SubscriptionPackageBadge from '@/components/payment/SubscriptionPackageBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import BackupSettingsSection from './settings/BackupSettingsSection.vue'
import EmailTemplateEditor from './settings/EmailTemplateEditor.vue'
import { adminAPI } from '@/api'
import {
  appendAuthSourceDefaultsToUpdateRequest,
  buildAuthSourceDefaultsState,
  deriveWeChatConnectStoredMode,
  resolveWeChatConnectModeCapabilities,
  type AuthSourceDefaultsState,
  type AuthSourceType,
  type DefaultSubscriptionSetting,
  type SystemSettings,
  type UpdateSettingsRequest,
} from '@/api/admin/settings'
import type { CustomEndpoint, CustomMenuItem, LoginAgreementDocument, NotifyEmailEntry } from '@/types'
import type { ProviderInstance, SubscriptionPlan } from '@/types/payment'
import type { TypeOption } from '@/components/payment/providerConfig'
import { computePlanValidityDays, getPlanQuotaAmount } from '@/utils/subscriptionQuotaPlans'
import {
  formatSubscriptionPlanValidity,
  formatSubscriptionQuotaAmount,
} from '@/utils/subscriptionPlanDisplay'
import { useAppStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { extractApiErrorMessage } from '@/utils/apiError'

type TabKey = 'general' | 'agreement' | 'security' | 'users' | 'payment' | 'email' | 'backup'

interface SettingsForm extends UpdateSettingsRequest {
  site_name: string
  site_logo: string
  site_subtitle: string
  frontend_url: string
  api_base_url: string
  contact_info: string
  doc_url: string
  home_content: string
  table_default_page_size: number
  table_page_size_options: number[]
  backend_mode_enabled: boolean
  purchase_subscription_enabled: boolean
  purchase_subscription_url: string
  custom_menu_items: CustomMenuItem[]
  custom_endpoints: CustomEndpoint[]
  registration_enabled: boolean
  email_verify_enabled: boolean
  registration_email_suffix_whitelist: string[]
  promo_code_enabled: boolean
  password_reset_enabled: boolean
  invitation_code_enabled: boolean
  totp_enabled: boolean
  login_agreement_enabled: boolean
  login_agreement_mode: string
  login_agreement_documents: LoginAgreementDocument[]
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_use_tls: boolean
  turnstile_enabled: boolean
  turnstile_site_key: string
  turnstile_secret_key: string
  linuxdo_connect_enabled: boolean
  linuxdo_connect_client_id: string
  linuxdo_connect_client_secret: string
  linuxdo_connect_redirect_url: string
  dingtalk_connect_enabled: boolean
  dingtalk_connect_client_id: string
  dingtalk_connect_client_secret: string
  dingtalk_connect_redirect_url: string
  dingtalk_connect_corp_restriction_policy: string
  dingtalk_connect_internal_corp_id: string
  dingtalk_connect_bypass_registration: boolean
  dingtalk_connect_sync_corp_email: boolean
  dingtalk_connect_sync_display_name: boolean
  dingtalk_connect_sync_dept: boolean
  dingtalk_connect_sync_corp_email_attr_key: string
  dingtalk_connect_sync_display_name_attr_key: string
  dingtalk_connect_sync_dept_attr_key: string
  dingtalk_connect_sync_corp_email_attr_name: string
  dingtalk_connect_sync_display_name_attr_name: string
  dingtalk_connect_sync_dept_attr_name: string
  wechat_connect_enabled: boolean
  wechat_connect_app_id: string
  wechat_connect_mp_app_id: string
  wechat_connect_mp_app_secret: string
  wechat_connect_open_enabled: boolean
  wechat_connect_mp_enabled: boolean
  wechat_connect_mobile_enabled: boolean
  wechat_connect_mode: string
  wechat_connect_scopes: string
  wechat_connect_redirect_url: string
  wechat_connect_frontend_redirect_url: string
  oidc_connect_enabled: boolean
  oidc_connect_provider_name: string
  oidc_connect_client_id: string
  oidc_connect_client_secret: string
  oidc_connect_issuer_url: string
  oidc_connect_discovery_url: string
  oidc_connect_authorize_url: string
  oidc_connect_token_url: string
  oidc_connect_userinfo_url: string
  oidc_connect_jwks_url: string
  oidc_connect_scopes: string
  oidc_connect_redirect_url: string
  oidc_connect_frontend_redirect_url: string
  oidc_connect_token_auth_method: string
  oidc_connect_use_pkce: boolean
  oidc_connect_validate_id_token: boolean
  oidc_connect_allowed_signing_algs: string
  oidc_connect_clock_skew_seconds: number
  oidc_connect_require_email_verified: boolean
  oidc_connect_userinfo_email_path: string
  oidc_connect_userinfo_id_path: string
  oidc_connect_userinfo_username_path: string
  github_oauth_enabled: boolean
  github_oauth_client_id: string
  github_oauth_client_secret: string
  github_oauth_redirect_url: string
  github_oauth_frontend_redirect_url: string
  google_oauth_enabled: boolean
  google_oauth_client_id: string
  google_oauth_client_secret: string
  google_oauth_redirect_url: string
  google_oauth_frontend_redirect_url: string
  default_balance: number
  default_concurrency: number
  default_user_rpm_limit: number
  default_subscriptions: DefaultSubscriptionSetting[]
  force_email_on_third_party_signup: boolean
  affiliate_enabled: boolean
  affiliate_rebate_rate: number
  affiliate_rebate_freeze_hours: number
  affiliate_rebate_duration_days: number
  affiliate_rebate_per_invitee_cap: number
  payment_enabled: boolean
  payment_enabled_types: string[]
  payment_min_amount: number
  payment_max_amount: number
  payment_daily_limit: number
  payment_order_timeout_minutes: number
  payment_max_pending_orders: number
  payment_balance_disabled: boolean
  payment_balance_recharge_multiplier: number
  payment_recharge_fee_rate: number
  payment_load_balance_strategy: string
  payment_product_name_prefix: string
  payment_product_name_suffix: string
  payment_help_image_url: string
  payment_help_text: string
  payment_cancel_rate_limit_enabled: boolean
  payment_cancel_rate_limit_max: number
  payment_cancel_rate_limit_window: number
  payment_cancel_rate_limit_unit: string
  payment_cancel_rate_limit_window_mode: string
  payment_alipay_force_qrcode: boolean
  balance_low_notify_enabled: boolean
  balance_low_notify_threshold: number
  balance_low_notify_recharge_url: string
  subscription_expiry_notify_enabled: boolean
  account_quota_notify_enabled: boolean
  account_quota_notify_emails: NotifyEmailEntry[]
}

const { t, locale } = useI18n()
const appStore = useAppStore()
const adminSettingsStore = useAdminSettingsStore()
const route = useRoute() as ReturnType<typeof useRoute> | undefined

const tabs = [
  { key: 'general' as const, label: t('admin.settings.tabs.general'), icon: 'cog' as const },
  { key: 'agreement' as const, label: t('admin.settings.tabs.agreement'), icon: 'document' as const },
  { key: 'security' as const, label: t('admin.settings.tabs.security'), icon: 'shield' as const },
  { key: 'users' as const, label: t('admin.settings.tabs.users'), icon: 'users' as const },
  { key: 'payment' as const, label: t('admin.settings.tabs.payment'), icon: 'creditCard' as const },
  { key: 'email' as const, label: t('admin.settings.tabs.email'), icon: 'mail' as const },
  { key: 'backup' as const, label: t('admin.settings.tabs.backup'), icon: 'database' as const },
]
const tabKeys = tabs.map((tab) => tab.key)
const activeTab = ref<TabKey>(resolveInitialTab(route?.query?.tab))

function resolveInitialTab(value: unknown): TabKey {
  const candidate = Array.isArray(value) ? value[0] : value
  return tabKeys.includes(candidate as TabKey) ? candidate as TabKey : 'general'
}

function selectTab(tab: TabKey) {
  activeTab.value = tab
}

async function focusSelectedTab(tab: TabKey) {
  await nextTick()
  document.getElementById(`settings-tab-${tab}`)?.focus()
}

function handleTabKeydown(event: KeyboardEvent, tab: TabKey) {
  const currentIndex = tabs.findIndex((item) => item.key === tab)
  if (currentIndex < 0) return

  let nextIndex = currentIndex
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % tabs.length
  } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
  } else if (event.key === 'Home') {
    nextIndex = 0
  } else if (event.key === 'End') {
    nextIndex = tabs.length - 1
  } else {
    return
  }

  event.preventDefault()
  const nextTab = tabs[nextIndex]?.key
  if (nextTab) {
    selectTab(nextTab)
    void focusSelectedTab(nextTab)
  }
}

const authSourceTypes: AuthSourceType[] = ['email', 'linuxdo', 'oidc', 'wechat', 'github', 'google', 'dingtalk']

const form = reactive<SettingsForm>({
  site_name: '',
  site_logo: '',
  site_subtitle: '',
  frontend_url: '',
  api_base_url: '',
  contact_info: '',
  doc_url: '',
  home_content: '',
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  backend_mode_enabled: false,
  purchase_subscription_enabled: false,
  purchase_subscription_url: '',
  custom_menu_items: [],
  custom_endpoints: [],
  registration_enabled: true,
  email_verify_enabled: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: false,
  password_reset_enabled: false,
  invitation_code_enabled: false,
  totp_enabled: false,
  login_agreement_enabled: false,
  login_agreement_mode: 'modal',
  login_agreement_documents: [],
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  smtp_from_email: '',
  smtp_from_name: '',
  smtp_use_tls: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  turnstile_secret_key: '',
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: '',
  linuxdo_connect_client_secret: '',
  linuxdo_connect_redirect_url: '',
  dingtalk_connect_enabled: false,
  dingtalk_connect_client_id: '',
  dingtalk_connect_client_secret: '',
  dingtalk_connect_redirect_url: '',
  dingtalk_connect_corp_restriction_policy: 'none',
  dingtalk_connect_internal_corp_id: '',
  dingtalk_connect_bypass_registration: false,
  dingtalk_connect_sync_corp_email: false,
  dingtalk_connect_sync_display_name: false,
  dingtalk_connect_sync_dept: false,
  dingtalk_connect_sync_corp_email_attr_key: 'dingtalk_email',
  dingtalk_connect_sync_display_name_attr_key: 'dingtalk_name',
  dingtalk_connect_sync_dept_attr_key: 'dingtalk_department',
  dingtalk_connect_sync_corp_email_attr_name: '钉钉企业邮箱',
  dingtalk_connect_sync_display_name_attr_name: '钉钉姓名',
  dingtalk_connect_sync_dept_attr_name: '钉钉部门',
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
  oidc_connect_provider_name: 'OIDC',
  oidc_connect_client_id: '',
  oidc_connect_client_secret: '',
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
  github_oauth_enabled: false,
  github_oauth_client_id: '',
  github_oauth_client_secret: '',
  github_oauth_redirect_url: '',
  github_oauth_frontend_redirect_url: '/auth/github/callback',
  google_oauth_enabled: false,
  google_oauth_client_id: '',
  google_oauth_client_secret: '',
  google_oauth_redirect_url: '',
  google_oauth_frontend_redirect_url: '/auth/oauth/callback',
  default_balance: 0,
  default_concurrency: 1,
  default_user_rpm_limit: 0,
  default_subscriptions: [],
  force_email_on_third_party_signup: false,
  affiliate_enabled: false,
  affiliate_rebate_rate: 0,
  affiliate_rebate_freeze_hours: 0,
  affiliate_rebate_duration_days: 0,
  affiliate_rebate_per_invitee_cap: 0,
  payment_enabled: false,
  payment_enabled_types: [],
  payment_min_amount: 0,
  payment_max_amount: 0,
  payment_daily_limit: 0,
  payment_order_timeout_minutes: 30,
  payment_max_pending_orders: 3,
  payment_balance_disabled: false,
  payment_balance_recharge_multiplier: 1,
  payment_recharge_fee_rate: 0,
  payment_load_balance_strategy: 'round-robin',
  payment_product_name_prefix: '',
  payment_product_name_suffix: '',
  payment_help_image_url: '',
  payment_help_text: '',
  payment_cancel_rate_limit_enabled: false,
  payment_cancel_rate_limit_max: 10,
  payment_cancel_rate_limit_window: 1,
  payment_cancel_rate_limit_unit: 'day',
  payment_cancel_rate_limit_window_mode: 'rolling',
  payment_alipay_force_qrcode: false,
  balance_low_notify_enabled: false,
  balance_low_notify_threshold: 0,
  balance_low_notify_recharge_url: '',
  subscription_expiry_notify_enabled: false,
  account_quota_notify_enabled: false,
  account_quota_notify_emails: [],
})

const authSourceDefaults = reactive<AuthSourceDefaultsState>(buildAuthSourceDefaultsState({}))
const loading = ref(false)
const saving = ref(false)
const settingsLoadError = ref('')
const paymentProvidersError = ref('')
const subscriptionPlansError = ref('')
const wechatMpSecretConfigured = ref(false)
const smtpPasswordConfigured = ref(false)
const turnstileSecretConfigured = ref(false)
const linuxdoSecretConfigured = ref(false)
const dingtalkSecretConfigured = ref(false)
const githubSecretConfigured = ref(false)
const googleSecretConfigured = ref(false)
const oidcSecretConfigured = ref(false)
const paymentProviders = ref<ProviderInstance[]>([])
const paymentProvidersLoading = ref(false)
const subscriptionPlans = ref<SubscriptionPlan[]>([])
const subscriptionPlansLoading = ref(false)
const tablePageSizeOptionsInput = ref('10, 20, 50, 100')
const emailSuffixWhitelistInput = ref('')
const accountQuotaEmailsInput = ref('')
const testEmail = ref('')
const testingSmtp = ref(false)
const sendingTestEmail = ref(false)
const adminApiKey = reactive({
  loading: false,
  exists: false,
  masked_key: '',
  generated: '',
})
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
const paymentTypeOptions = computed<TypeOption[]>(() => [
  { value: 'alipay', label: t('admin.settings.payment.types.alipay') },
  { value: 'wxpay', label: t('admin.settings.payment.types.wxpay') },
  { value: 'stripe', label: t('admin.settings.payment.types.stripe') },
  { value: 'airwallex', label: t('admin.settings.payment.types.airwallex') },
])
const paymentLoadBalanceOptions = computed<TypeOption[]>(() => [
  { value: 'round-robin', label: t('admin.settings.payment.loadBalanceRoundRobin') },
  { value: 'least-amount', label: t('admin.settings.payment.loadBalanceLeastAmount') },
])
const paymentCancelRateLimitUnitOptions = computed<TypeOption[]>(() => [
  { value: 'minute', label: t('admin.settings.payment.cancelRateLimitUnitMinute') },
  { value: 'hour', label: t('admin.settings.payment.cancelRateLimitUnitHour') },
  { value: 'day', label: t('admin.settings.payment.cancelRateLimitUnitDay') },
])
const paymentCancelRateLimitWindowModeOptions = computed<TypeOption[]>(() => [
  { value: 'rolling', label: t('admin.settings.payment.cancelRateLimitModeRolling') },
  { value: 'fixed', label: t('admin.settings.payment.cancelRateLimitModeFixed') },
])

onMounted(async () => {
  await loadSettings()
  await Promise.all([loadPaymentProviders(), loadSubscriptionPlans(), loadAdminApiKey()])
})

async function loadSettings() {
  loading.value = true
  settingsLoadError.value = ''
  try {
    const settings = await adminAPI.settings.getSettings()
    applySettings(settings)
  } catch (error: unknown) {
    settingsLoadError.value = extractApiErrorMessage(error, t('admin.settings.failedToLoad'))
    appStore.showError(settingsLoadError.value)
  } finally {
    loading.value = false
  }
}

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
    site_subtitle: settings.site_subtitle || '',
    frontend_url: settings.frontend_url || '',
    api_base_url: settings.api_base_url || '',
    contact_info: settings.contact_info || '',
    doc_url: settings.doc_url || '',
    home_content: settings.home_content || '',
    table_default_page_size: settings.table_default_page_size || 20,
    table_page_size_options: Array.isArray(settings.table_page_size_options) ? settings.table_page_size_options : [10, 20, 50, 100],
    backend_mode_enabled: settings.backend_mode_enabled ?? false,
    purchase_subscription_enabled: settings.purchase_subscription_enabled ?? false,
    purchase_subscription_url: settings.purchase_subscription_url || '',
    custom_menu_items: cloneCustomMenuItems(settings.custom_menu_items),
    custom_endpoints: cloneCustomEndpoints(settings.custom_endpoints),
    registration_enabled: settings.registration_enabled ?? true,
    email_verify_enabled: settings.email_verify_enabled ?? false,
    registration_email_suffix_whitelist: settings.registration_email_suffix_whitelist || [],
    promo_code_enabled: settings.promo_code_enabled ?? false,
    password_reset_enabled: settings.password_reset_enabled ?? false,
    invitation_code_enabled: settings.invitation_code_enabled ?? false,
    totp_enabled: settings.totp_enabled ?? false,
    login_agreement_enabled: settings.login_agreement_enabled ?? false,
    login_agreement_mode: settings.login_agreement_mode || 'modal',
    login_agreement_documents: cloneLoginAgreementDocuments(settings.login_agreement_documents),
    smtp_host: settings.smtp_host || '',
    smtp_port: settings.smtp_port || 587,
    smtp_username: settings.smtp_username || '',
    smtp_password: '',
    smtp_from_email: settings.smtp_from_email || '',
    smtp_from_name: settings.smtp_from_name || '',
    smtp_use_tls: settings.smtp_use_tls ?? true,
    turnstile_enabled: settings.turnstile_enabled ?? false,
    turnstile_site_key: settings.turnstile_site_key || '',
    turnstile_secret_key: '',
    linuxdo_connect_enabled: settings.linuxdo_connect_enabled ?? false,
    linuxdo_connect_client_id: settings.linuxdo_connect_client_id || '',
    linuxdo_connect_client_secret: '',
    linuxdo_connect_redirect_url: settings.linuxdo_connect_redirect_url || '',
    dingtalk_connect_enabled: settings.dingtalk_connect_enabled ?? false,
    dingtalk_connect_client_id: settings.dingtalk_connect_client_id || '',
    dingtalk_connect_client_secret: '',
    dingtalk_connect_redirect_url: settings.dingtalk_connect_redirect_url || '',
    dingtalk_connect_corp_restriction_policy: settings.dingtalk_connect_corp_restriction_policy || 'none',
    dingtalk_connect_internal_corp_id: settings.dingtalk_connect_internal_corp_id || '',
    dingtalk_connect_bypass_registration: settings.dingtalk_connect_bypass_registration ?? false,
    dingtalk_connect_sync_corp_email: settings.dingtalk_connect_sync_corp_email ?? false,
    dingtalk_connect_sync_display_name: settings.dingtalk_connect_sync_display_name ?? false,
    dingtalk_connect_sync_dept: settings.dingtalk_connect_sync_dept ?? false,
    dingtalk_connect_sync_corp_email_attr_key: settings.dingtalk_connect_sync_corp_email_attr_key || 'dingtalk_email',
    dingtalk_connect_sync_display_name_attr_key: settings.dingtalk_connect_sync_display_name_attr_key || 'dingtalk_name',
    dingtalk_connect_sync_dept_attr_key: settings.dingtalk_connect_sync_dept_attr_key || 'dingtalk_department',
    dingtalk_connect_sync_corp_email_attr_name: settings.dingtalk_connect_sync_corp_email_attr_name || '钉钉企业邮箱',
    dingtalk_connect_sync_display_name_attr_name: settings.dingtalk_connect_sync_display_name_attr_name || '钉钉姓名',
    dingtalk_connect_sync_dept_attr_name: settings.dingtalk_connect_sync_dept_attr_name || '钉钉部门',
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
    oidc_connect_provider_name: settings.oidc_connect_provider_name || 'OIDC',
    oidc_connect_client_id: settings.oidc_connect_client_id || '',
    oidc_connect_client_secret: '',
    oidc_connect_issuer_url: settings.oidc_connect_issuer_url || '',
    oidc_connect_discovery_url: settings.oidc_connect_discovery_url || '',
    oidc_connect_authorize_url: settings.oidc_connect_authorize_url || '',
    oidc_connect_token_url: settings.oidc_connect_token_url || '',
    oidc_connect_userinfo_url: settings.oidc_connect_userinfo_url || '',
    oidc_connect_jwks_url: settings.oidc_connect_jwks_url || '',
    oidc_connect_scopes: settings.oidc_connect_scopes || 'openid email profile',
    oidc_connect_redirect_url: settings.oidc_connect_redirect_url || '',
    oidc_connect_frontend_redirect_url: settings.oidc_connect_frontend_redirect_url || '/auth/oidc/callback',
    oidc_connect_token_auth_method: settings.oidc_connect_token_auth_method || 'client_secret_post',
    oidc_connect_use_pkce: settings.oidc_connect_use_pkce ?? true,
    oidc_connect_validate_id_token: settings.oidc_connect_validate_id_token ?? true,
    oidc_connect_allowed_signing_algs: settings.oidc_connect_allowed_signing_algs || 'RS256,ES256,PS256',
    oidc_connect_clock_skew_seconds: settings.oidc_connect_clock_skew_seconds ?? 120,
    oidc_connect_require_email_verified: settings.oidc_connect_require_email_verified ?? false,
    oidc_connect_userinfo_email_path: settings.oidc_connect_userinfo_email_path || '',
    oidc_connect_userinfo_id_path: settings.oidc_connect_userinfo_id_path || '',
    oidc_connect_userinfo_username_path: settings.oidc_connect_userinfo_username_path || '',
    github_oauth_enabled: settings.github_oauth_enabled ?? false,
    github_oauth_client_id: settings.github_oauth_client_id || '',
    github_oauth_client_secret: '',
    github_oauth_redirect_url: settings.github_oauth_redirect_url || '',
    github_oauth_frontend_redirect_url: settings.github_oauth_frontend_redirect_url || '/auth/github/callback',
    google_oauth_enabled: settings.google_oauth_enabled ?? false,
    google_oauth_client_id: settings.google_oauth_client_id || '',
    google_oauth_client_secret: '',
    google_oauth_redirect_url: settings.google_oauth_redirect_url || '',
    google_oauth_frontend_redirect_url: settings.google_oauth_frontend_redirect_url || '/auth/oauth/callback',
    default_balance: settings.default_balance ?? 0,
    default_concurrency: settings.default_concurrency ?? 1,
    default_user_rpm_limit: settings.default_user_rpm_limit ?? 0,
    default_subscriptions: normalizeSettingsSubscriptions(settings.default_subscriptions),
    force_email_on_third_party_signup: settings.force_email_on_third_party_signup ?? false,
    affiliate_enabled: settings.affiliate_enabled ?? false,
    affiliate_rebate_rate: settings.affiliate_rebate_rate ?? 0,
    affiliate_rebate_freeze_hours: settings.affiliate_rebate_freeze_hours ?? 0,
    affiliate_rebate_duration_days: settings.affiliate_rebate_duration_days ?? 0,
    affiliate_rebate_per_invitee_cap: settings.affiliate_rebate_per_invitee_cap ?? 0,
    payment_enabled: settings.payment_enabled ?? false,
    payment_enabled_types: settings.payment_enabled_types || [],
    payment_min_amount: settings.payment_min_amount ?? 0,
    payment_max_amount: settings.payment_max_amount ?? 0,
    payment_daily_limit: settings.payment_daily_limit ?? 0,
    payment_order_timeout_minutes: settings.payment_order_timeout_minutes ?? 30,
    payment_max_pending_orders: settings.payment_max_pending_orders ?? 3,
    payment_balance_disabled: settings.payment_balance_disabled ?? false,
    payment_balance_recharge_multiplier: settings.payment_balance_recharge_multiplier ?? 1,
    payment_recharge_fee_rate: settings.payment_recharge_fee_rate ?? 0,
    payment_load_balance_strategy: normalizePaymentLoadBalanceStrategy(settings.payment_load_balance_strategy),
    payment_product_name_prefix: settings.payment_product_name_prefix || '',
    payment_product_name_suffix: settings.payment_product_name_suffix || '',
    payment_help_image_url: settings.payment_help_image_url || '',
    payment_help_text: settings.payment_help_text || '',
    payment_cancel_rate_limit_enabled: settings.payment_cancel_rate_limit_enabled ?? false,
    payment_cancel_rate_limit_max: settings.payment_cancel_rate_limit_max ?? 10,
    payment_cancel_rate_limit_window: settings.payment_cancel_rate_limit_window ?? 1,
    payment_cancel_rate_limit_unit: settings.payment_cancel_rate_limit_unit || 'day',
    payment_cancel_rate_limit_window_mode: settings.payment_cancel_rate_limit_window_mode || 'rolling',
    payment_alipay_force_qrcode: settings.payment_alipay_force_qrcode ?? false,
    balance_low_notify_enabled: settings.balance_low_notify_enabled ?? false,
    balance_low_notify_threshold: settings.balance_low_notify_threshold ?? 0,
    balance_low_notify_recharge_url: settings.balance_low_notify_recharge_url || '',
    subscription_expiry_notify_enabled: settings.subscription_expiry_notify_enabled ?? false,
    account_quota_notify_enabled: settings.account_quota_notify_enabled ?? false,
    account_quota_notify_emails: cloneNotifyEmailEntries(settings.account_quota_notify_emails),
  })
  tablePageSizeOptionsInput.value = form.table_page_size_options.join(', ')
  emailSuffixWhitelistInput.value = form.registration_email_suffix_whitelist.join(', ')
  accountQuotaEmailsInput.value = form.account_quota_notify_emails.map((item) => item.email).filter(Boolean).join(', ')
  wechatMpSecretConfigured.value =
    settings.wechat_connect_mp_app_secret_configured ?? settings.wechat_connect_app_secret_configured ?? false
  smtpPasswordConfigured.value = settings.smtp_password_configured ?? false
  turnstileSecretConfigured.value = settings.turnstile_secret_key_configured ?? false
  linuxdoSecretConfigured.value = settings.linuxdo_connect_client_secret_configured ?? false
  dingtalkSecretConfigured.value = settings.dingtalk_connect_client_secret_configured ?? false
  githubSecretConfigured.value = settings.github_oauth_client_secret_configured ?? false
  googleSecretConfigured.value = settings.google_oauth_client_secret_configured ?? false
  oidcSecretConfigured.value = settings.oidc_connect_client_secret_configured ?? false

  Object.assign(authSourceDefaults, buildAuthSourceDefaultsState(settings))
}

async function loadPaymentProviders() {
  paymentProvidersLoading.value = true
  paymentProvidersError.value = ''
  try {
    const response = await adminAPI.payment.getProviders()
    paymentProviders.value = Array.isArray(response) ? response : response.data || []
  } catch (error: unknown) {
    paymentProvidersError.value = extractApiErrorMessage(error, t('admin.settings.payment.failedToLoadProviders'))
    paymentProviders.value = []
  } finally {
    paymentProvidersLoading.value = false
  }
}

async function loadSubscriptionPlans() {
  subscriptionPlansLoading.value = true
  subscriptionPlansError.value = ''
  try {
    const response = await adminAPI.payment.getPlans()
    subscriptionPlans.value = (response.data || []).map((plan: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...plan,
      features: Array.isArray(plan.features)
        ? plan.features
        : String(plan.features || '').split('\n').map((item) => item.trim()).filter(Boolean),
    }))
    hydrateDefaultSubscriptionPlanBindings()
  } catch (error: unknown) {
    subscriptionPlansError.value = extractApiErrorMessage(error, t('admin.settings.authSourceDefaults.failedToLoadPlans'))
    subscriptionPlans.value = []
  } finally {
    subscriptionPlansLoading.value = false
  }
}

async function toggleProviderField(provider: ProviderInstance, field: 'enabled' | 'refund_enabled' | 'allow_user_refund') {
  try {
    await adminAPI.payment.updateProvider(provider.id, { [field]: !provider[field] })
    await loadPaymentProviders()
    appStore.showSuccess(t('admin.settings.payment.providerUpdated'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  }
}

function buildPayload(): UpdateSettingsRequest {
  const wechatConnectMode = deriveWeChatConnectStoredMode(
    form.wechat_connect_open_enabled,
    form.wechat_connect_mp_enabled,
    form.wechat_connect_mobile_enabled,
    form.wechat_connect_mode,
  )
  const payload: UpdateSettingsRequest = {
    site_name: form.site_name,
    site_logo: form.site_logo,
    site_subtitle: form.site_subtitle,
    frontend_url: form.frontend_url,
    api_base_url: form.api_base_url,
    contact_info: form.contact_info,
    doc_url: form.doc_url,
    home_content: form.home_content,
    table_default_page_size: normalizePositiveInteger(form.table_default_page_size, 20),
    table_page_size_options: normalizePageSizeOptions(tablePageSizeOptionsInput.value),
    backend_mode_enabled: form.backend_mode_enabled,
    purchase_subscription_enabled: form.purchase_subscription_enabled,
    purchase_subscription_url: form.purchase_subscription_url,
    custom_menu_items: normalizeCustomMenuItems(form.custom_menu_items),
    custom_endpoints: normalizeCustomEndpoints(form.custom_endpoints),
    registration_enabled: form.registration_enabled,
    email_verify_enabled: form.email_verify_enabled,
    registration_email_suffix_whitelist: splitListInput(emailSuffixWhitelistInput.value),
    promo_code_enabled: form.promo_code_enabled,
    password_reset_enabled: form.password_reset_enabled,
    invitation_code_enabled: form.invitation_code_enabled,
    totp_enabled: form.totp_enabled,
    login_agreement_enabled: form.login_agreement_enabled,
    login_agreement_mode: form.login_agreement_mode,
    login_agreement_documents: normalizeLoginAgreementDocuments(form.login_agreement_documents),
    smtp_host: form.smtp_host,
    smtp_port: normalizePositiveInteger(form.smtp_port, 587),
    smtp_username: form.smtp_username,
    smtp_from_email: form.smtp_from_email,
    smtp_from_name: form.smtp_from_name,
    smtp_use_tls: form.smtp_use_tls,
    turnstile_enabled: form.turnstile_enabled,
    turnstile_site_key: form.turnstile_site_key,
    linuxdo_connect_enabled: form.linuxdo_connect_enabled,
    linuxdo_connect_client_id: form.linuxdo_connect_client_id,
    linuxdo_connect_redirect_url: form.linuxdo_connect_redirect_url,
    dingtalk_connect_enabled: form.dingtalk_connect_enabled,
    dingtalk_connect_client_id: form.dingtalk_connect_client_id,
    dingtalk_connect_redirect_url: form.dingtalk_connect_redirect_url,
    dingtalk_connect_corp_restriction_policy: form.dingtalk_connect_corp_restriction_policy,
    dingtalk_connect_internal_corp_id: form.dingtalk_connect_internal_corp_id,
    dingtalk_connect_bypass_registration: form.dingtalk_connect_bypass_registration,
    dingtalk_connect_sync_corp_email: form.dingtalk_connect_sync_corp_email,
    dingtalk_connect_sync_display_name: form.dingtalk_connect_sync_display_name,
    dingtalk_connect_sync_dept: form.dingtalk_connect_sync_dept,
    dingtalk_connect_sync_corp_email_attr_key: form.dingtalk_connect_sync_corp_email_attr_key,
    dingtalk_connect_sync_display_name_attr_key: form.dingtalk_connect_sync_display_name_attr_key,
    dingtalk_connect_sync_dept_attr_key: form.dingtalk_connect_sync_dept_attr_key,
    dingtalk_connect_sync_corp_email_attr_name: form.dingtalk_connect_sync_corp_email_attr_name,
    dingtalk_connect_sync_display_name_attr_name: form.dingtalk_connect_sync_display_name_attr_name,
    dingtalk_connect_sync_dept_attr_name: form.dingtalk_connect_sync_dept_attr_name,
    wechat_connect_enabled: form.wechat_connect_enabled,
    wechat_connect_app_id: form.wechat_connect_mp_app_id || form.wechat_connect_app_id,
    wechat_connect_mp_app_id: form.wechat_connect_mp_app_id,
    wechat_connect_open_enabled: form.wechat_connect_open_enabled,
    wechat_connect_mp_enabled: form.wechat_connect_mp_enabled,
    wechat_connect_mobile_enabled: form.wechat_connect_mobile_enabled,
    wechat_connect_mode: wechatConnectMode,
    wechat_connect_redirect_url: form.wechat_connect_redirect_url,
    wechat_connect_frontend_redirect_url: form.wechat_connect_frontend_redirect_url,
    oidc_connect_enabled: form.oidc_connect_enabled,
    oidc_connect_provider_name: form.oidc_connect_provider_name,
    oidc_connect_client_id: form.oidc_connect_client_id,
    oidc_connect_issuer_url: form.oidc_connect_issuer_url,
    oidc_connect_discovery_url: form.oidc_connect_discovery_url,
    oidc_connect_authorize_url: form.oidc_connect_authorize_url,
    oidc_connect_token_url: form.oidc_connect_token_url,
    oidc_connect_userinfo_url: form.oidc_connect_userinfo_url,
    oidc_connect_jwks_url: form.oidc_connect_jwks_url,
    oidc_connect_scopes: form.oidc_connect_scopes,
    oidc_connect_redirect_url: form.oidc_connect_redirect_url,
    oidc_connect_frontend_redirect_url: form.oidc_connect_frontend_redirect_url,
    oidc_connect_token_auth_method: form.oidc_connect_token_auth_method,
    oidc_connect_use_pkce: form.oidc_connect_use_pkce,
    oidc_connect_validate_id_token: form.oidc_connect_validate_id_token,
    oidc_connect_allowed_signing_algs: form.oidc_connect_allowed_signing_algs,
    oidc_connect_clock_skew_seconds: Math.max(0, Math.floor(Number(form.oidc_connect_clock_skew_seconds) || 0)),
    oidc_connect_require_email_verified: form.oidc_connect_require_email_verified,
    oidc_connect_userinfo_email_path: form.oidc_connect_userinfo_email_path,
    oidc_connect_userinfo_id_path: form.oidc_connect_userinfo_id_path,
    oidc_connect_userinfo_username_path: form.oidc_connect_userinfo_username_path,
    github_oauth_enabled: form.github_oauth_enabled,
    github_oauth_client_id: form.github_oauth_client_id,
    github_oauth_redirect_url: form.github_oauth_redirect_url,
    github_oauth_frontend_redirect_url: form.github_oauth_frontend_redirect_url,
    google_oauth_enabled: form.google_oauth_enabled,
    google_oauth_client_id: form.google_oauth_client_id,
    google_oauth_redirect_url: form.google_oauth_redirect_url,
    google_oauth_frontend_redirect_url: form.google_oauth_frontend_redirect_url,
    default_balance: normalizeNonNegativeNumber(form.default_balance),
    default_concurrency: normalizePositiveInteger(form.default_concurrency, 1),
    default_user_rpm_limit: Math.max(0, Math.floor(Number(form.default_user_rpm_limit) || 0)),
    default_subscriptions: normalizeSettingsSubscriptions(form.default_subscriptions),
    force_email_on_third_party_signup: form.force_email_on_third_party_signup,
    affiliate_enabled: form.affiliate_enabled,
    affiliate_rebate_rate: normalizeNonNegativeNumber(form.affiliate_rebate_rate),
    affiliate_rebate_freeze_hours: Math.max(0, Math.floor(Number(form.affiliate_rebate_freeze_hours) || 0)),
    affiliate_rebate_duration_days: Math.max(0, Math.floor(Number(form.affiliate_rebate_duration_days) || 0)),
    affiliate_rebate_per_invitee_cap: normalizeNonNegativeNumber(form.affiliate_rebate_per_invitee_cap),
    payment_enabled: form.payment_enabled,
    payment_enabled_types: [...form.payment_enabled_types],
    payment_min_amount: normalizeNonNegativeNumber(form.payment_min_amount),
    payment_max_amount: normalizeNonNegativeNumber(form.payment_max_amount),
    payment_daily_limit: normalizeNonNegativeNumber(form.payment_daily_limit),
    payment_order_timeout_minutes: normalizePositiveInteger(form.payment_order_timeout_minutes, 30),
    payment_max_pending_orders: normalizePositiveInteger(form.payment_max_pending_orders, 3),
    payment_balance_disabled: form.payment_balance_disabled,
    payment_balance_recharge_multiplier: normalizeNonNegativeNumber(form.payment_balance_recharge_multiplier),
    payment_recharge_fee_rate: normalizeNonNegativeNumber(form.payment_recharge_fee_rate),
    payment_load_balance_strategy: normalizePaymentLoadBalanceStrategy(form.payment_load_balance_strategy),
    payment_product_name_prefix: form.payment_product_name_prefix,
    payment_product_name_suffix: form.payment_product_name_suffix,
    payment_help_image_url: form.payment_help_image_url,
    payment_help_text: form.payment_help_text,
    payment_cancel_rate_limit_enabled: form.payment_cancel_rate_limit_enabled,
    payment_cancel_rate_limit_max: normalizePositiveInteger(form.payment_cancel_rate_limit_max, 10),
    payment_cancel_rate_limit_window: normalizePositiveInteger(form.payment_cancel_rate_limit_window, 1),
    payment_cancel_rate_limit_unit: form.payment_cancel_rate_limit_unit,
    payment_cancel_rate_limit_window_mode: form.payment_cancel_rate_limit_window_mode,
    payment_alipay_force_qrcode: form.payment_alipay_force_qrcode,
    balance_low_notify_enabled: form.balance_low_notify_enabled,
    balance_low_notify_threshold: normalizeNonNegativeNumber(form.balance_low_notify_threshold),
    balance_low_notify_recharge_url: form.balance_low_notify_recharge_url,
    subscription_expiry_notify_enabled: form.subscription_expiry_notify_enabled,
    account_quota_notify_enabled: form.account_quota_notify_enabled,
    account_quota_notify_emails: normalizeNotifyEmailEntries(accountQuotaEmailsInput.value),
  }

  if (form.smtp_password) {
    payload.smtp_password = form.smtp_password
  }
  if (form.turnstile_secret_key) {
    payload.turnstile_secret_key = form.turnstile_secret_key
  }
  if (form.linuxdo_connect_client_secret) {
    payload.linuxdo_connect_client_secret = form.linuxdo_connect_client_secret
  }
  if (form.dingtalk_connect_client_secret) {
    payload.dingtalk_connect_client_secret = form.dingtalk_connect_client_secret
  }
  if (form.wechat_connect_mp_app_secret) {
    payload.wechat_connect_mp_app_secret = form.wechat_connect_mp_app_secret
  }
  if (form.github_oauth_client_secret) {
    payload.github_oauth_client_secret = form.github_oauth_client_secret
  }
  if (form.google_oauth_client_secret) {
    payload.google_oauth_client_secret = form.google_oauth_client_secret
  }
  if (form.oidc_connect_client_secret) {
    payload.oidc_connect_client_secret = form.oidc_connect_client_secret
  }

  return appendAuthSourceDefaultsToUpdateRequest(payload, authSourceDefaults)
}

const subscriptionPlanOptions = computed(() =>
  subscriptionPlans.value
    .filter((plan) => plan.subscription_type !== 'standard' && plan.group_status === 'active')
    .sort((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id))
    .map((plan) => ({
      value: plan.id,
      label: plan.name,
      platform: plan.platform || plan.group_platform || 'social',
      description: plan.description || '',
      quotaDisplay: formatQuota(getPlanQuotaAmount(plan)),
      validityLabel: formatPlanValidity(plan),
      hidden: plan.for_sale === false,
    }))
)

function normalizeSettingsSubscriptions(items: DefaultSubscriptionSetting[] | null | undefined): DefaultSubscriptionSetting[] {
  if (!Array.isArray(items)) return []
  return items
    .filter((item) => Number(item.plan_id || item.group_id || 0) > 0 && Number(item.validity_days) > 0)
    .map((item) => {
      const planID = item.plan_id != null && Number(item.plan_id) > 0 ? Math.floor(Number(item.plan_id)) : undefined
      const groupID = planID == null && item.group_id != null && Number(item.group_id) > 0 ? Math.floor(Number(item.group_id)) : undefined
      const normalized: DefaultSubscriptionSetting = {
        plan_id: planID,
        validity_days: Math.min(36500, Math.max(1, Math.floor(Number(item.validity_days)))),
      }
      if (groupID != null) normalized.group_id = groupID
      return normalized
    })
}

function addDefaultSubscription(target: DefaultSubscriptionSetting[]) {
  const usedPlanIds = new Set(target.map((item) => item.plan_id).filter(Boolean))
  const option = subscriptionPlanOptions.value.find((item) => !usedPlanIds.has(Number(item.value))) || subscriptionPlanOptions.value[0]
  if (!option) return
  const plan = subscriptionPlans.value.find((item) => item.id === Number(option.value))
  target.push({
    plan_id: Number(option.value),
    validity_days: plan ? computePlanValidityDays(plan) : 30,
  })
}

function addGlobalDefaultSubscription() {
  addDefaultSubscription(form.default_subscriptions)
}

function removeDefaultSubscription(target: DefaultSubscriptionSetting[], index: number) {
  target.splice(index, 1)
}

function syncDefaultSubscriptionPlan(item: DefaultSubscriptionSetting) {
  const plan = subscriptionPlans.value.find((candidate) => candidate.id === Number(item.plan_id))
  if (!plan) return
  item.plan_id = plan.id
  item.group_id = plan.group_id
  if (!item.validity_days || item.validity_days < 1) {
    item.validity_days = computePlanValidityDays(plan)
  }
}

function hydrateDefaultSubscriptionPlanBindings() {
  hydrateSubscriptionList(form.default_subscriptions)
  for (const source of authSourceTypes) {
    hydrateSubscriptionList(authSourceDefaults[source].subscriptions)
  }
}

function hydrateSubscriptionList(items: DefaultSubscriptionSetting[]) {
  for (const item of items) {
    if (item.plan_id) {
      syncDefaultSubscriptionPlan(item)
      continue
    }
    if (!item.group_id) continue
    const plan = subscriptionPlans.value.find((candidate) => candidate.group_id === item.group_id)
    if (!plan) continue
    item.plan_id = plan.id
    item.group_id = plan.group_id
  }
}

function addCustomMenuItem() {
  form.custom_menu_items.push({
    id: '',
    label: '',
    icon_svg: '',
    url: '',
    page_slug: '',
    visibility: 'admin',
    sort_order: form.custom_menu_items.length,
  })
}

function removeCustomMenuItem(index: number) {
  form.custom_menu_items.splice(index, 1)
}

function addCustomEndpoint() {
  form.custom_endpoints.push({
    name: '',
    endpoint: '',
    description: '',
  })
}

function removeCustomEndpoint(index: number) {
  form.custom_endpoints.splice(index, 1)
}

function addLoginAgreementDocument() {
  form.login_agreement_documents.push({
    id: `agreement-${Date.now()}`,
    title: '',
    content_md: '',
  })
}

function removeLoginAgreementDocument(index: number) {
  form.login_agreement_documents.splice(index, 1)
}

async function testSmtpConnection() {
  testingSmtp.value = true
  try {
    await adminAPI.settings.testSmtpConnection({
      smtp_host: form.smtp_host,
      smtp_port: normalizePositiveInteger(form.smtp_port, 587),
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password,
      smtp_use_tls: form.smtp_use_tls,
    })
    appStore.showSuccess(t('admin.settings.smtp.testSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.settings.smtp.testFailed')))
  } finally {
    testingSmtp.value = false
  }
}

async function sendTestEmail() {
  if (!testEmail.value) return
  sendingTestEmail.value = true
  try {
    await adminAPI.settings.sendTestEmail({
      email: testEmail.value,
      smtp_host: form.smtp_host,
      smtp_port: normalizePositiveInteger(form.smtp_port, 587),
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password,
      smtp_from_email: form.smtp_from_email,
      smtp_from_name: form.smtp_from_name,
      smtp_use_tls: form.smtp_use_tls,
    })
    appStore.showSuccess(t('admin.settings.smtp.sendSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.settings.smtp.sendFailed')))
  } finally {
    sendingTestEmail.value = false
  }
}

async function loadAdminApiKey() {
  adminApiKey.loading = true
  try {
    const status = await adminAPI.settings.getAdminApiKey()
    adminApiKey.exists = status.exists
    adminApiKey.masked_key = status.masked_key || ''
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.settings.adminApiKey.failedToLoad')))
  } finally {
    adminApiKey.loading = false
  }
}

async function regenerateAdminApiKey() {
  if (!window.confirm(t('admin.settings.adminApiKey.regenerateConfirm'))) return
  adminApiKey.loading = true
  try {
    const response = await adminAPI.settings.regenerateAdminApiKey()
    adminApiKey.generated = response.key
    adminApiKey.exists = true
    adminApiKey.masked_key = response.key ? `${response.key.slice(0, 8)}...${response.key.slice(-4)}` : ''
    appStore.showSuccess(t('admin.settings.adminApiKey.regenerated'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    adminApiKey.loading = false
  }
}

async function deleteAdminApiKey() {
  if (!window.confirm(t('admin.settings.adminApiKey.deleteConfirm'))) return
  adminApiKey.loading = true
  try {
    await adminAPI.settings.deleteAdminApiKey()
    adminApiKey.exists = false
    adminApiKey.masked_key = ''
    adminApiKey.generated = ''
    appStore.showSuccess(t('admin.settings.adminApiKey.deleted'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    adminApiKey.loading = false
  }
}

function formatQuota(value: number | null | undefined): string {
  return formatSubscriptionQuotaAmount(value, t('payment.admin.unlimited'))
}

function formatPlanValidity(plan: SubscriptionPlan): string {
  return formatSubscriptionPlanValidity(plan, t, locale.value)
}

async function saveSettings() {
  saving.value = true
  try {
    const updated = await adminAPI.settings.updateSettings(buildPayload())
    clearSecretInputs()
    if (updated) {
      applySettings({ ...form, ...updated })
    }
    await Promise.all([
      appStore.fetchPublicSettings(true),
      adminSettingsStore.fetch(true),
    ])
    appStore.showSuccess(t('admin.settings.settingsSaved'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.settings.failedToSave')))
  } finally {
    saving.value = false
  }
}

function clearSecretInputs() {
  if (form.smtp_password) smtpPasswordConfigured.value = true
  if (form.turnstile_secret_key) turnstileSecretConfigured.value = true
  if (form.linuxdo_connect_client_secret) linuxdoSecretConfigured.value = true
  if (form.dingtalk_connect_client_secret) dingtalkSecretConfigured.value = true
  if (form.wechat_connect_mp_app_secret) wechatMpSecretConfigured.value = true
  if (form.github_oauth_client_secret) githubSecretConfigured.value = true
  if (form.google_oauth_client_secret) googleSecretConfigured.value = true
  if (form.oidc_connect_client_secret) oidcSecretConfigured.value = true
  form.smtp_password = ''
  form.turnstile_secret_key = ''
  form.linuxdo_connect_client_secret = ''
  form.dingtalk_connect_client_secret = ''
  form.wechat_connect_mp_app_secret = ''
  form.github_oauth_client_secret = ''
  form.google_oauth_client_secret = ''
  form.oidc_connect_client_secret = ''
}

function splitListInput(value: string): string[] {
  return value
    .split(/[\n,;]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function normalizePageSizeOptions(value: string): number[] {
  const values = splitListInput(value)
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item) && item > 0)
    .map((item) => Math.floor(item))
  return Array.from(new Set(values)).sort((a, b) => a - b)
}

function normalizePositiveInteger(value: unknown, fallback: number): number {
  const normalized = Math.floor(Number(value))
  return Number.isFinite(normalized) && normalized > 0 ? normalized : fallback
}

function normalizePaymentLoadBalanceStrategy(value: unknown): string {
  const normalized = String(value || '').trim()
  if (normalized === 'least-amount') return normalized
  if (normalized === 'round-robin' || normalized === 'round_robin') return 'round-robin'
  return 'round-robin'
}

function normalizeNonNegativeNumber(value: unknown): number {
  const normalized = Number(value)
  return Number.isFinite(normalized) && normalized >= 0 ? normalized : 0
}

function cloneCustomMenuItems(items: CustomMenuItem[] | null | undefined): CustomMenuItem[] {
  if (!Array.isArray(items)) return []
  return items.map((item, index) => ({
    id: item.id || '',
    label: item.label || '',
    icon_svg: item.icon_svg || '',
    url: item.url || '',
    page_slug: item.page_slug || '',
    visibility: item.visibility === 'user' ? 'user' : 'admin',
    sort_order: Number(item.sort_order ?? index),
  }))
}

function normalizeCustomMenuItems(items: CustomMenuItem[]): CustomMenuItem[] {
  return cloneCustomMenuItems(items)
    .filter((item) => item.label.trim() || item.url.trim() || item.page_slug?.trim())
    .map((item, index) => ({
      ...item,
      url: customMenuItemURLForPayload(item),
      sort_order: index,
    }))
}

function customMenuItemURLForPayload(item: CustomMenuItem): string {
  return item.page_slug?.trim() ? `md:${item.page_slug.trim()}` : item.url
}

function cloneCustomEndpoints(items: CustomEndpoint[] | null | undefined): CustomEndpoint[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    name: item.name || '',
    endpoint: item.endpoint || '',
    description: item.description || '',
  }))
}

function normalizeCustomEndpoints(items: CustomEndpoint[]): CustomEndpoint[] {
  return cloneCustomEndpoints(items).filter((item) => item.name.trim() || item.endpoint.trim())
}

function cloneLoginAgreementDocuments(items: LoginAgreementDocument[] | null | undefined): LoginAgreementDocument[] {
  if (!Array.isArray(items)) return []
  return items.map((item, index) => ({
    id: item.id || `agreement-${index}`,
    title: item.title || '',
    content_md: item.content_md || '',
  }))
}

function normalizeLoginAgreementDocuments(items: LoginAgreementDocument[]): LoginAgreementDocument[] {
  return cloneLoginAgreementDocuments(items).filter((item) => item.title.trim() || item.content_md.trim())
}

function cloneNotifyEmailEntries(items: NotifyEmailEntry[] | null | undefined): NotifyEmailEntry[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({
    email: item.email || '',
    disabled: item.disabled === true,
    verified: item.verified === true,
  }))
}

function normalizeNotifyEmailEntries(value: string): NotifyEmailEntry[] {
  return splitListInput(value).map((email) => ({
    email,
    disabled: false,
    verified: false,
  }))
}
</script>

<style scoped>
.settings-tabs-shell {
  @apply sticky z-20 -mx-1 rounded-2xl border border-white/80 bg-white/90 p-1.5 backdrop-blur-xl;
  top: 4.75rem;
  box-shadow:
    0 12px 28px rgb(15 23 42 / 0.07),
    0 1px 0 rgb(255 255 255 / 0.9) inset;
}

.settings-tabs-scroll {
  @apply overflow-x-auto;
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.settings-tabs-scroll::-webkit-scrollbar {
  display: none;
}

.settings-tabs {
  @apply flex min-w-max items-center gap-1;
}

.settings-tab {
  @apply relative isolate flex h-10 min-w-[6.75rem] shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-xl border border-transparent px-3 text-sm font-medium text-gray-600 outline-none transition-colors duration-200 ease-out dark:text-gray-300;
}

@media (min-width: 768px) {
  .settings-tabs {
    @apply min-w-full;
  }

  .settings-tab {
    @apply min-w-0 flex-1 basis-0 overflow-hidden px-2 text-[13px];
  }

  .settings-tab-icon {
    @apply h-6 w-6;
  }
}

.settings-tab::before {
  @apply absolute inset-0 -z-10 rounded-xl opacity-0 transition-opacity duration-200;
  content: "";
  background: linear-gradient(135deg, rgb(248 250 252 / 0.95), rgb(241 245 249 / 0.8));
}

.settings-tab:hover::before,
.settings-tab:focus-visible::before {
  opacity: 1;
}

.settings-tab:focus-visible {
  @apply ring-2 ring-primary-500/40 ring-offset-2 ring-offset-white dark:ring-offset-dark-900;
}

.settings-tab-active {
  @apply border-primary-200/80 bg-white text-primary-700 shadow-sm dark:border-primary-400/30 dark:bg-dark-700/95 dark:text-primary-200;
  box-shadow:
    0 8px 18px rgb(15 23 42 / 0.08),
    0 1px 0 rgb(255 255 255 / 0.92) inset;
}

.settings-tab-active::before {
  opacity: 0;
}

.settings-tab-active::after {
  position: absolute;
  right: 0.75rem;
  bottom: 0.25rem;
  left: 0.75rem;
  height: 2px;
  border-radius: 9999px;
  content: "";
  background: linear-gradient(90deg, #14b8a6, #0ea5e9);
}

.settings-tab-icon {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors duration-200 dark:text-gray-400;
}

.settings-tab:hover .settings-tab-icon,
.settings-tab:focus-visible .settings-tab-icon {
  @apply text-gray-700 dark:text-gray-200;
}

.settings-tab-active .settings-tab-icon {
  @apply bg-primary-50 text-primary-600 dark:bg-primary-400/10 dark:text-primary-300;
}

.settings-tab-label {
  @apply min-w-0 overflow-hidden text-ellipsis whitespace-nowrap leading-none;
}

</style>

<style>
.dark .settings-tabs-shell {
  border-color: rgb(51 65 85 / 0.65);
  background: rgb(15 23 42 / 0.86);
  box-shadow:
    0 16px 36px rgb(0 0 0 / 0.28),
    0 1px 0 rgb(255 255 255 / 0.06) inset;
}

.dark .settings-tab::before {
  background: linear-gradient(135deg, rgb(30 41 59 / 0.9), rgb(51 65 85 / 0.62));
}

.dark .settings-tab-active {
  box-shadow:
    0 12px 26px rgb(0 0 0 / 0.22),
    0 1px 0 rgb(255 255 255 / 0.08) inset;
}
</style>
