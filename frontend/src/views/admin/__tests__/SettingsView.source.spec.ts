import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const sourcePath = resolve(__dirname, "../SettingsView.vue");
const source = readFileSync(sourcePath, "utf8");

describe("admin SettingsView SaaS operations coverage", () => {
  it("exposes core SaaS settings modules without adding duplicate pages or APIs", () => {
    expect(source).toContain("EmailTemplateEditor");
    expect(source).toContain("adminAPI.settings.testSmtpConnection");
    expect(source).toContain("adminAPI.settings.sendTestEmail");
    expect(source).toContain("adminAPI.settings.getAdminApiKey");
    expect(source).toContain("adminAPI.settings.regenerateAdminApiKey");
    expect(source).toContain("adminAPI.settings.deleteAdminApiKey");
    expect(source).toContain("custom_menu_items");
    expect(source).toContain("custom_endpoints");
    expect(source).toContain("login_agreement_documents");
  });

  it("exposes custom page slug editing through custom menu items", () => {
    const customMenuSection = source.slice(
      source.indexOf("admin.settings.customMenu.title"),
      source.indexOf("admin.settings.customEndpoints.title"),
    );
    const normalizeMenuSource = source.slice(
      source.indexOf("function normalizeCustomMenuItems"),
      source.indexOf("function cloneCustomEndpoints"),
    );

    expect(customMenuSection).toContain("item.page_slug");
    expect(customMenuSection).toContain("admin.settings.customMenu.pageSlug");
    expect(customMenuSection).toContain("admin.settings.customMenu.pageSlugHint");
    expect(normalizeMenuSource).toContain("customMenuItemURLForPayload(item)");
    expect(normalizeMenuSource).toContain("md:${item.page_slug.trim()}");
  });

  it("persists visible fields while preserving omitted secrets and hidden modules", () => {
    const buildPayloadSource = source.slice(
      source.indexOf("function buildPayload"),
      source.indexOf("const subscriptionPlanOptions"),
    );

    expect(buildPayloadSource).toContain("site_subtitle: form.site_subtitle");
    expect(buildPayloadSource).toContain("frontend_url: form.frontend_url");
    expect(buildPayloadSource).toContain("smtp_host: form.smtp_host");
    expect(buildPayloadSource).toContain("turnstile_enabled: form.turnstile_enabled");
    expect(buildPayloadSource).toContain("balance_low_notify_enabled: form.balance_low_notify_enabled");
    expect(buildPayloadSource).toContain("payment_min_amount: normalizeNonNegativeNumber(form.payment_min_amount)");
    expect(buildPayloadSource).toContain("if (form.smtp_password)");
    expect(buildPayloadSource).toContain("if (form.turnstile_secret_key)");
    expect(buildPayloadSource).not.toContain("smtp_password: form.smtp_password");
    expect(buildPayloadSource).not.toContain("turnstile_secret_key: form.turnstile_secret_key");
  });

  it("keeps commercial OAuth providers and advanced OIDC fields in the security settings tab", () => {
    const securitySection = source.slice(
      source.indexOf("activeTab === 'security'"),
      source.indexOf("activeTab === 'users'"),
    );
    const buildPayloadSource = source.slice(
      source.indexOf("function buildPayload"),
      source.indexOf("const subscriptionPlanOptions"),
    );

    expect(securitySection).toContain("form.linuxdo_connect_enabled");
    expect(securitySection).toContain("form.dingtalk_connect_enabled");
    expect(securitySection).toContain("form.google_oauth_enabled");
    expect(securitySection).toContain("form.oidc_connect_discovery_url");
    expect(securitySection).toContain("form.oidc_connect_authorize_url");
    expect(securitySection).toContain("form.oidc_connect_token_url");
    expect(securitySection).toContain("form.oidc_connect_userinfo_url");
    expect(securitySection).toContain("form.oidc_connect_jwks_url");
    expect(securitySection).toContain("form.oidc_connect_token_auth_method");
    expect(securitySection).toContain("form.oidc_connect_require_email_verified");
    expect(securitySection).toContain("form.oidc_connect_userinfo_email_path");
    expect(buildPayloadSource).toContain("if (form.linuxdo_connect_client_secret)");
    expect(buildPayloadSource).toContain("payload.linuxdo_connect_client_secret = form.linuxdo_connect_client_secret");
    expect(buildPayloadSource).toContain("if (form.dingtalk_connect_client_secret)");
    expect(buildPayloadSource).toContain("payload.dingtalk_connect_client_secret = form.dingtalk_connect_client_secret");
    expect(buildPayloadSource).toContain("if (form.google_oauth_client_secret)");
    expect(buildPayloadSource).toContain("payload.google_oauth_client_secret = form.google_oauth_client_secret");
    expect(buildPayloadSource).not.toContain("linuxdo_connect_client_secret: form.linuxdo_connect_client_secret,");
    expect(buildPayloadSource).not.toContain("dingtalk_connect_client_secret: form.dingtalk_connect_client_secret,");
    expect(buildPayloadSource).not.toContain("google_oauth_client_secret: form.google_oauth_client_secret,");
  });

  it("uses keyboard-accessible roving tab navigation for settings modules", () => {
    const tabNavigationSource = source.slice(
      source.indexOf("<nav"),
      source.indexOf("id=\"settings-panel-general\""),
    );

    expect(tabNavigationSource).toContain('role="tab"');
    expect(tabNavigationSource).toContain(":tabindex=\"activeTab === tab.key ? 0 : -1\"");
    expect(tabNavigationSource).toContain("@keydown=\"handleTabKeydown($event, tab.key)\"");
    expect(source).toContain("function selectTab(tab: TabKey)");
    expect(source).toContain("function handleTabKeydown(event: KeyboardEvent, tab: TabKey)");
  });

  it("handles settings failures through user-visible state instead of console noise", () => {
    expect(source).not.toContain("console.error");
    expect(source).toContain("useAppStore");
    expect(source).toContain("extractApiErrorMessage");
    expect(source).toContain("settingsLoadError");
    expect(source).toContain("paymentProvidersError");
  });

  it("removes SocialOps and operations shortcut cards from system settings", () => {
    expect(source).not.toContain("activeTab === 'social'");
    expect(source).not.toContain("activeTab === 'operations'");
    expect(source).not.toContain('RouterLink to="/admin/total-accounts"');
    expect(source).not.toContain('RouterLink to="/accounts"');
    expect(source).not.toContain('RouterLink to="/admin/data-management"');
    expect(source).not.toContain('RouterLink to="/admin/backups"');
    expect(source).not.toContain("admin.settings.tabs.social");
    expect(source).not.toContain("admin.settings.tabs.operations");
    expect(source).not.toContain("admin.settings.operations.title");
  });

  it("keeps settings switches scoped to owner tabs without a feature-switch tab", () => {
    expect(source).not.toContain("activeTab === 'features'");
    expect(source).not.toContain("admin.settings.tabs.features");
    expect(source).not.toContain("activeTab === 'registration'");
    expect(source).not.toContain("admin.settings.tabs.registration");
    expect(source).not.toContain("activeTab === 'auth'");
    expect(source).not.toContain("admin.settings.tabs.auth");
    expect(source).not.toContain("activeTab === 'affiliate'");
    expect(source).not.toContain("admin.settings.tabs.affiliate");
    expect(source).not.toContain("activeTab === 'notifications'");
    expect(source).not.toContain("admin.settings.tabs.notifications");

    const paymentSection = source.slice(
      source.indexOf("activeTab === 'payment'"),
      source.indexOf("activeTab === 'backup'"),
    );
    const agreementSection = source.slice(
      source.indexOf("activeTab === 'agreement'"),
      source.indexOf("activeTab === 'security'"),
    );
    const generalSection = source.slice(
      source.indexOf("activeTab === 'general'"),
      source.indexOf("activeTab === 'agreement'"),
    );
    const securitySection = source.slice(
      source.indexOf("activeTab === 'security'"),
      source.indexOf("activeTab === 'users'"),
    );
    const userDefaultsSection = source.slice(
      source.indexOf("activeTab === 'users'"),
      source.indexOf("activeTab === 'email'"),
    );
    const emailSection = source.slice(
      source.indexOf("activeTab === 'email'"),
      source.indexOf("activeTab === 'payment'"),
    );

    expect(paymentSection).toContain("form.payment_enabled");
    expect(paymentSection).toContain("form.payment_balance_disabled");
    expect(paymentSection).toContain("form.purchase_subscription_enabled");
    expect(paymentSection).toContain("form.purchase_subscription_url");
    expect(generalSection).not.toContain("form.purchase_subscription_enabled");
    expect(generalSection).not.toContain("form.purchase_subscription_url");
    expect(generalSection).toContain("form.backend_mode_enabled");
    expect(agreementSection).toContain("form.login_agreement_enabled");
    expect(agreementSection).toContain("form.login_agreement_mode");
    expect(securitySection).toContain("form.registration_enabled");
    expect(securitySection).toContain("form.promo_code_enabled");
    expect(securitySection).toContain("form.invitation_code_enabled");
    expect(securitySection).toContain("form.wechat_connect_enabled");
    expect(securitySection).toContain("form.linuxdo_connect_enabled");
    expect(securitySection).toContain("form.oidc_connect_discovery_url");
    expect(userDefaultsSection).toContain("admin.settings.authSourceDefaults.title");
    expect(userDefaultsSection).toContain("form.affiliate_enabled");
    expect(userDefaultsSection).toContain("form.affiliate_rebate_per_invitee_cap");
    expect(emailSection).toContain("form.smtp_host");
    expect(emailSection).toContain("EmailTemplateEditor");
    expect(emailSection).toContain("form.balance_low_notify_enabled");
    expect(emailSection).toContain("form.subscription_expiry_notify_enabled");
    expect(emailSection).toContain("form.account_quota_notify_enabled");
  });

  it("makes user defaults and affiliate rebate settings discoverable in the user defaults tab", () => {
    const userDefaultsSection = source.slice(
      source.indexOf("activeTab === 'users'"),
      source.indexOf("activeTab === 'email'"),
    );

    expect(userDefaultsSection).toContain("admin.settings.authSourceDefaults.title");
    expect(userDefaultsSection).toContain("admin.settings.authSourceDefaults.globalSubscriptionsLabel");
    expect(userDefaultsSection).toContain("admin.settings.authSourceDefaults.defaultSubscriptionsLabel");
    expect(userDefaultsSection).toContain("admin.settings.affiliate.title");
    expect(userDefaultsSection).toContain("form.affiliate_rebate_per_invitee_cap");
    expect(userDefaultsSection).toContain('data-testid="affiliate-rebate-per-invitee-cap"');
  });

  it("keeps backup operations inside the settings center", () => {
    expect(source).toContain("BackupSettingsSection");
    expect(source).toContain("activeTab === 'backup'");
    expect(source).toContain("admin.settings.tabs.backup");
    expect(source).toContain("activeTab !== 'backup'");
    expect(source).not.toContain("adminAPI.settings.getOverloadCooldownSettings");
    expect(source).not.toContain("adminAPI.settings.updateOverloadCooldownSettings");
    expect(source).not.toContain("adminAPI.settings.updateRateLimit429CooldownSettings");
    expect(source).not.toContain("adminAPI.settings.updateStreamTimeoutSettings");
  });

  it("does not wrap the whole settings workspace in a nested form", () => {
    expect(source).not.toContain('<form class="space-y-6"');
    expect(source).not.toContain('type="submit"');
    expect(source).toContain('<form class="contents" autocomplete="off" @submit.prevent="saveSettings">');
    expect(source).toContain('autocomplete="username"');
    expect(source).toContain('autocomplete="new-password"');
    expect(source).toContain('@click="saveSettings"');
    expect(source).not.toContain("activeTab !== 'email'");
  });

  it("keeps payment guide links as complete attributes", () => {
    expect(source).toContain('href="https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md"');
    expect(source).toContain('href="https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md#supported-payment-methods"');
    expect(source).not.toContain('docs/PAYMENT_CN.md#支持的支付方式');
    expect(source.match(/rel="noopener noreferrer"/g)?.length ?? 0).toBeGreaterThanOrEqual(2);
  });

  it("keeps the settings console free of removed gateway business semantics", () => {
    const removedGatewayMarkers = [
      ["Open", "AI"].join(""),
      ["Clau", "de"].join(""),
      ["Bed", "rock"].join(""),
      ["Web", "Search"].join(""),
      ["web", "search"].join("_"),
      ["So", "ra"].join(""),
      ["anti", "gravity"].join(""),
    ];
    for (const forbidden of removedGatewayMarkers) {
      expect(source).not.toContain(forbidden);
    }
  });
});
