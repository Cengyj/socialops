import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, h } from "vue";
import { flushPromises, mount } from "@vue/test-utils";

import SettingsView from "../SettingsView.vue";

const {
  getSettings,
  updateSettings,
  getAdminApiKey,
  getGroups,
  getProviders,
  getPlans,
  updateProvider,
  createProvider,
  deleteProvider,
  fetchPublicSettings,
  adminSettingsFetch,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  getAdminApiKey: vi.fn(),
  getGroups: vi.fn(),
  getProviders: vi.fn(),
  getPlans: vi.fn(),
  updateProvider: vi.fn(),
  createProvider: vi.fn(),
  deleteProvider: vi.fn(),
  fetchPublicSettings: vi.fn(),
  adminSettingsFetch: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}));

const localeRef = vi.hoisted(() => ({ value: "zh-CN" }));

vi.mock("@/api", () => ({
  adminAPI: {
    settings: {
      getSettings,
      updateSettings,
      getAdminApiKey,
    },
    groups: {
      getAll: getGroups,
    },
    payment: {
      getProviders,
      getPlans,
      updateProvider,
      createProvider,
      deleteProvider,
    },
  },
}));

vi.mock("@/stores", () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning: vi.fn(),
    showInfo: vi.fn(),
    fetchPublicSettings,
  }),
}));

vi.mock("@/stores/adminSettings", () => ({
  useAdminSettingsStore: () => ({
    fetch: adminSettingsFetch,
  }),
}));

vi.mock("@/composables/useClipboard", () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}));

vi.mock("vue-router", async () => {
  const actual = await vi.importActual<typeof import("vue-router")>("vue-router");
  return {
    ...actual,
    useRoute: () => ({ query: {} }),
  };
});

vi.mock("@/utils/apiError", () => ({
  extractApiErrorMessage: () => "error",
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  const translations: Record<string, string> = {
    "admin.settings.wechatConnect.title": "微信登录",
    "admin.settings.wechatConnect.description": "用于微信开放平台或公众号/小程序的第三方登录配置。",
    "admin.settings.wechatConnect.enabledLabel": "启用微信登录",
    "admin.settings.wechatConnect.enabledHint": "开启后可使用微信第三方登录回调与授权配置。",
    "admin.settings.wechatConnect.appIdLabel": "AppID",
    "admin.settings.wechatConnect.appIdPlaceholder": "微信开放平台 AppID",
    "admin.settings.wechatConnect.appSecretLabel": "AppSecret",
    "admin.settings.wechatConnect.appSecretConfiguredPlaceholder": "密钥已配置，留空以保留当前值。",
    "admin.settings.wechatConnect.appSecretPlaceholder": "微信开放平台 AppSecret",
    "admin.settings.wechatConnect.appSecretConfiguredHint": "密钥已配置，留空以保留当前值。",
    "admin.settings.wechatConnect.appSecretHint": "填写后会覆盖当前微信密钥。",
    "admin.settings.wechatConnect.modeLabel": "模式",
    "admin.settings.wechatConnect.openModeLabel": "非微信环境使用开放平台",
    "admin.settings.wechatConnect.openModeHint": "浏览器不在微信内时，自动走开放平台扫码授权。",
    "admin.settings.wechatConnect.mpModeLabel": "微信环境使用公众号",
    "admin.settings.wechatConnect.mpModeHint": "浏览器在微信内时，自动走公众号授权。",
    "admin.settings.wechatConnect.redirectUrlLabel": "回调地址",
    "admin.settings.wechatConnect.redirectUrlPlaceholder": "https://your-site.com/api/v1/auth/oauth/wechat/callback",
    "admin.settings.wechatConnect.generateAndCopy": "使用当前站点生成并复制",
    "admin.settings.wechatConnect.redirectUrlSetAndCopied": "已使用当前站点生成回调地址并复制到剪贴板",
    "admin.settings.wechatConnect.frontendRedirectUrlLabel": "前端回调地址",
    "admin.settings.wechatConnect.frontendRedirectUrlPlaceholder": "/auth/wechat/callback",
    "admin.settings.wechatConnect.frontendRedirectUrlHint": "通常用于前端路由回调地址，需与后端配置保持一致。",
    "admin.settings.authSourceDefaults.title": "认证来源默认值",
    "admin.settings.authSourceDefaults.description": "按注册来源配置新用户默认余额、并发、订阅与授权策略。",
    "admin.settings.authSourceDefaults.requireEmailLabel": "第三方注册强制补充邮箱",
    "admin.settings.authSourceDefaults.requireEmailHint": "启用后，Linux DO、OIDC、微信注册缺少邮箱时必须先补充邮箱地址。",
    "admin.settings.authSourceDefaults.enabledHint": "以下默认值会在该来源注册新用户时发放；首次绑定时授权仅作用于已有账号绑定该来源。",
    "admin.settings.authSourceDefaults.sources.email.title": "邮箱注册",
    "admin.settings.authSourceDefaults.sources.email.description": "适用于邮箱密码注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.linuxdo.title": "Linux DO 登录",
    "admin.settings.authSourceDefaults.sources.linuxdo.description": "适用于 Linux DO 第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.oidc.title": "OIDC 登录",
    "admin.settings.authSourceDefaults.sources.oidc.description": "适用于 OIDC 第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.sources.wechat.title": "微信登录",
    "admin.settings.authSourceDefaults.sources.wechat.description": "适用于微信第三方注册的新用户默认配额。",
    "admin.settings.authSourceDefaults.grantOnFirstBindLabel": "首次绑定时授权",
    "admin.settings.authSourceDefaults.grantOnFirstBindHint": "已有账号首次绑定该来源时发放默认权益。",
    "admin.settings.authSourceDefaults.defaultSubscriptionsLabel": "默认订阅",
    "admin.settings.authSourceDefaults.defaultSubscriptionsHint": "仅对当前认证来源生效，未配置时不追加来源专属订阅。",
    "admin.settings.authSourceDefaults.noSourceSubscriptions": "当前来源未配置专属默认订阅。",
    "admin.settings.affiliate.perInviteeCap": "单个受邀人返利上限",
    "admin.settings.paymentVisibleMethods.methodLabel": "{title} 可见方式",
    "admin.settings.paymentVisibleMethods.methodHint": "控制前台结算页是否展示该方式，以及展示时使用的来源键。",
    ["admin.settings.paymentVisibleMethods." + "source" + "Label"]: "支付来源",
    "admin.settings.paymentVisibleMethods.sourceHint": "启用后必须明确选择一个来源；未配置状态不会对外展示该支付方式。",
    "admin.settings.paymentVisibleMethods.sourceRequiredError": "{title} 已启用，请先选择支付来源。",
    "admin.settings.payment.configGuide": "查看支付配置说明",
    "admin.settings.payment.findProvider": "查看支持的支付方式",
    "admin.settings.site.uploadImage": "上传图片",
    "admin.settings.site.remove": "移除",
  };
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) =>
        (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, token) => params?.[token] ?? `{${token}}`),
      locale: localeRef,
    }),
  };
});

const AppLayoutStub = { template: "<div><slot /></div>" };
const ToggleStub = defineComponent({
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["update:modelValue"],
  inheritAttrs: false,
  setup(props, { attrs, emit }) {
    return () =>
      h("input", {
        ...attrs,
        class: "toggle-stub",
        type: "checkbox",
        checked: props.modelValue,
        onChange: (event: Event) => {
          emit("update:modelValue", (event.target as HTMLInputElement).checked);
        },
      });
  },
});

const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: "",
    },
    options: {
      type: Array,
      default: () => [],
    },
    placeholder: {
      type: String,
      default: "",
    },
  },
  emits: ["update:modelValue", "change"],
  setup(props, { emit }) {
    const onChange = (event: Event) => {
      const target = event.target as HTMLSelectElement;
      emit("update:modelValue", target.value);
      const option =
        (props.options as Array<Record<string, unknown>>).find(
          (item) => String(item.value ?? "") === target.value,
        ) ?? null;
      emit("change", target.value, option);
    };

    return () =>
      h(
        "select",
        {
          class: "select-stub",
          value: props.modelValue ?? "",
          "data-placeholder": props.placeholder,
          onChange,
        },
        (props.options as Array<Record<string, unknown>>).map((option) =>
          h(
            "option",
            {
              key: `${String(option.value ?? "")}:${String(option.label ?? "")}`,
              value: option.value as string,
            },
            String(option.label ?? ""),
          ),
        ),
      );
  },
});

const ImageUploadStub = defineComponent({
  props: {
    modelValue: {
      type: String,
      default: "",
    },
    uploadLabel: {
      type: String,
      default: "",
    },
    removeLabel: {
      type: String,
      default: "",
    },
    placeholder: {
      type: String,
      default: "",
    },
  },
  setup(props) {
    return () =>
      h("div", {
        class: "image-upload-stub",
        "data-model-value": props.modelValue,
        "data-upload-label": props.uploadLabel,
        "data-remove-label": props.removeLabel,
        "data-placeholder": props.placeholder,
      });
  },
});

const baseSettingsResponse = {
  registration_enabled: true,
  email_verify_enabled: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  invitation_code_enabled: false,
  password_reset_enabled: false,
  totp_enabled: false,
  totp_encryption_key_configured: false,
  default_balance: 0,
  default_concurrency: 1,
  default_subscriptions: [],
  site_name: "SocialOps",
  site_logo: "",
  site_subtitle: "",
  api_base_url: "",
  contact_info: "",
  doc_url: "",
  home_content: "",
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  backend_mode_enabled: false,
  custom_menu_items: [],
  custom_endpoints: [],
  frontend_url: "",
  smtp_host: "",
  smtp_port: 587,
  smtp_username: "",
  smtp_password_configured: false,
  smtp_from_email: "",
  smtp_from_name: "",
  smtp_use_tls: true,
  turnstile_enabled: false,
  turnstile_site_key: "",
  turnstile_secret_key_configured: false,
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: "",
  linuxdo_connect_client_secret_configured: false,
  linuxdo_connect_redirect_url: "",
  wechat_connect_enabled: true,
  wechat_connect_app_id: "wx-app-id-123",
  wechat_connect_app_secret_configured: true,
  wechat_connect_open_enabled: false,
  wechat_connect_mp_enabled: true,
  wechat_connect_mode: "mp",
  wechat_connect_scopes: "",
  wechat_connect_redirect_url:
    "https://admin.example.com/api/v1/auth/oauth/wechat/callback",
  wechat_connect_frontend_redirect_url: "/auth/wechat/callback",
  oidc_connect_enabled: false,
  oidc_connect_provider_name: "OIDC",
  oidc_connect_client_id: "",
  oidc_connect_client_secret_configured: false,
  oidc_connect_issuer_url: "",
  oidc_connect_discovery_url: "",
  oidc_connect_authorize_url: "",
  oidc_connect_token_url: "",
  oidc_connect_userinfo_url: "",
  oidc_connect_jwks_url: "",
  oidc_connect_scopes: "openid email profile",
  oidc_connect_redirect_url: "",
  oidc_connect_frontend_redirect_url: "/auth/oidc/callback",
  oidc_connect_token_auth_method: "client_secret_post",
  oidc_connect_use_pkce: true,
  oidc_connect_validate_id_token: true,
  oidc_connect_allowed_signing_algs: "RS256,ES256,PS256",
  oidc_connect_clock_skew_seconds: 120,
  oidc_connect_require_email_verified: false,
  oidc_connect_userinfo_email_path: "",
  oidc_connect_userinfo_id_path: "",
  oidc_connect_userinfo_username_path: "",
  payment_enabled: true,
  payment_min_amount: 1,
  payment_max_amount: 10000,
  payment_daily_limit: 50000,
  payment_order_timeout_minutes: 30,
  payment_max_pending_orders: 3,
  payment_enabled_types: [],
  payment_balance_disabled: false,
  payment_balance_recharge_multiplier: 1,
  payment_recharge_fee_rate: 0,
  payment_load_balance_strategy: "round-robin",
  payment_product_name_prefix: "",
  payment_product_name_suffix: "",
  payment_help_image_url: "",
  payment_help_text: "",
  payment_cancel_rate_limit_enabled: false,
  payment_cancel_rate_limit_max: 10,
  payment_cancel_rate_limit_window: 1,
  payment_cancel_rate_limit_unit: "day",
  payment_cancel_rate_limit_window_mode: "rolling",
  payment_visible_method_alipay_source: "alipay_direct",
  payment_visible_method_wxpay_source: "invalid-source",
  payment_visible_method_alipay_enabled: true,
  payment_visible_method_wxpay_enabled: true,
  balance_low_notify_enabled: false,
  balance_low_notify_threshold: 0,
  balance_low_notify_recharge_url: "",
  subscription_expiry_notify_enabled: true,
  account_quota_notify_enabled: false,
  account_quota_notify_emails: [],
  affiliate_enabled: false,
  affiliate_rebate_rate: 0,
  affiliate_rebate_freeze_hours: 0,
  affiliate_rebate_duration_days: 0,
  affiliate_rebate_per_invitee_cap: 0,
};

const sampleSubscriptionPlans = [
  {
    id: 101,
    group_id: 11,
    platform: "x_twitter",
    group_platform: "x_twitter",
    group_status: "active",
    subscription_type: "subscription",
    quota_usd: 100,
    monthly_limit_usd: 100,
    daily_limit_usd: 10,
    weekly_limit_usd: 50,
    name: "X 100",
    description: "100 USD execution quota",
    price: 88,
    original_price: 100,
    validity_days: 1,
    validity_unit: "months",
    features: [],
    for_sale: true,
    sort_order: 1,
  },
];

function mountView() {
  return mount(SettingsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        RouterLink: { template: "<a><slot /></a>" },
        Select: SelectStub,
        Toggle: ToggleStub,
        Icon: true,
        ConfirmDialog: true,
        PaymentProviderList: true,
        PaymentProviderDialog: true,
        SubscriptionPackageBadge: true,
        GroupBadge: true,
        GroupOptionItem: true,
        ProxySelector: true,
        ImageUpload: ImageUploadStub,
        BackupSettingsSection: true,
      },
    },
  });
}

async function openPaymentTab(wrapper: ReturnType<typeof mountView>) {
  const paymentTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.payment"));

  expect(paymentTabButton).toBeDefined();
  await paymentTabButton?.trigger("click");
  await flushPromises();
}

async function openUsersTab(wrapper: ReturnType<typeof mountView>) {
  const usersTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.users"));

  expect(usersTabButton).toBeDefined();
  await usersTabButton?.trigger("click");
  await flushPromises();
}

async function openSecurityTab(wrapper: ReturnType<typeof mountView>) {
  const securityTabButton = wrapper
    .findAll("button")
    .find((node) => node.text().includes("admin.settings.tabs.security"));

  expect(securityTabButton).toBeDefined();
  await securityTabButton?.trigger("click");
  await flushPromises();
}

describe("admin SettingsView payment visible method controls", () => {
  beforeEach(() => {
    getSettings.mockReset();
    updateSettings.mockReset();
    getAdminApiKey.mockReset();
    getGroups.mockReset();
    getProviders.mockReset();
    getPlans.mockReset();
    updateProvider.mockReset();
    createProvider.mockReset();
    deleteProvider.mockReset();
    fetchPublicSettings.mockReset();
    adminSettingsFetch.mockReset();
    showError.mockReset();
    showSuccess.mockReset();
    localeRef.value = "zh-CN";

    getSettings.mockResolvedValue({ ...baseSettingsResponse });
    updateSettings.mockImplementation(async (payload) => ({
      ...baseSettingsResponse,
      ...payload,
    }));
    getAdminApiKey.mockResolvedValue({
      exists: false,
      masked_key: "",
    });
    getGroups.mockResolvedValue([]);
    getProviders.mockResolvedValue({
      data: [],
    });
    getPlans.mockResolvedValue({
      data: sampleSubscriptionPlans,
    });
    fetchPublicSettings.mockResolvedValue(undefined);
    adminSettingsFetch.mockResolvedValue(undefined);
  });

  it("allows every real settings tab to be selected without falling back to hidden options", async () => {
    const wrapper = mountView();

    await flushPromises();

    const tabKeys = [
      "general",
      "agreement",
      "security",
      "users",
      "payment",
      "email",
      "backup",
    ];

    for (const tabKey of tabKeys) {
      const tabButton = wrapper
        .findAll("button")
        .find((node) => node.text().includes(`admin.settings.tabs.${tabKey}`));

      expect(tabButton, `missing settings tab: ${tabKey}`).toBeDefined();
      await tabButton?.trigger("click");
      await flushPromises();
      expect(tabButton?.attributes("aria-selected")).toBe("true");
    }
  });

  it("mounts only the active settings panel so hidden controls cannot intercept tab interactions", async () => {
    const wrapper = mountView();

    await flushPromises();

    expect(wrapper.text()).toContain("admin.settings.general.title");
    expect(wrapper.text()).not.toContain("admin.settings.payment.title");

    await openPaymentTab(wrapper);

    expect(wrapper.text()).toContain("admin.settings.payment.title");
    expect(wrapper.text()).not.toContain("admin.settings.general.title");
  });

  it("keeps a save action visible for email settings because SMTP and templates are editable", async () => {
    const wrapper = mountView();

    await flushPromises();
    const emailTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.email"));

    expect(emailTabButton).toBeDefined();
    await emailTabButton?.trigger("click");
    await flushPromises();

    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton, "email settings must not be a dead-end tab without a save action").toBeDefined();
  });

  it("removes the duplicate features tab and keeps switches in owner tabs", async () => {
    const wrapper = mountView();

    await flushPromises();
    const featuresTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.features"));
    const authTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.auth"));
    const affiliateTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.affiliate"));
    const notificationsTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.notifications"));

    expect(featuresTabButton).toBeUndefined();
    expect(authTabButton).toBeUndefined();
    expect(affiliateTabButton).toBeUndefined();
    expect(notificationsTabButton).toBeUndefined();

    await openPaymentTab(wrapper);
    expect(wrapper.text()).toContain("admin.settings.payment.enabled");
    expect(wrapper.text()).toContain("admin.settings.payment.balanceDisabled");
    expect(wrapper.text()).toContain("admin.settings.payment.purchaseEntry");

    await openUsersTab(wrapper);
    await flushPromises();

    expect(wrapper.text()).not.toContain("admin.settings.features.payment");
    expect(wrapper.text()).not.toContain("admin.settings.features.disableBalanceRecharge");
    expect(wrapper.text()).not.toContain("admin.settings.features.purchaseEntry");
    expect(wrapper.text()).not.toContain("admin.settings.features.promoCode");
    expect(wrapper.text()).not.toContain("admin.settings.features.invitationCode");
    expect(wrapper.text()).not.toContain("admin.settings.features.backendMode");
    expect(wrapper.text()).not.toContain("admin.settings.features.affiliate");
    expect(wrapper.text()).toContain("admin.settings.affiliate.title");
  });

  it("refreshes shared settings caches after a successful save", async () => {
    const wrapper = mountView();

    await flushPromises();
    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton).toBeDefined();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(fetchPublicSettings).toHaveBeenCalledWith(true);
    expect(adminSettingsFetch).toHaveBeenCalledWith(true);
  });

  it("keeps backup operations inside system settings", async () => {
    const wrapper = mountView();

    await flushPromises();
    const backupTabButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("admin.settings.tabs.backup"));

    expect(backupTabButton).toBeDefined();
    await backupTabButton?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("admin.settings.tabs.backup");
    expect(wrapper.text()).not.toContain("admin.settings.features.title");
  });

  it("does not render removed visible payment method controls", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    expect(wrapper.text()).not.toContain("可见方式");
    expect(wrapper.text()).not.toContain("支付来源");
  });

  it("links payment guidance to README sections instead of removed payment docs", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    const paymentLinks = wrapper
      .findAll("a")
      .filter((node) =>
        ["查看支付配置说明", "查看支持的支付方式"].includes(node.text()),
      );

    expect(paymentLinks).toHaveLength(2);
    expect(paymentLinks[0]?.attributes("href")).toBe(
      "https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md",
    );
    expect(paymentLinks[1]?.attributes("href")).toBe(
      "https://github.com/Wei-Shaw/socialops/blob/main/docs/PAYMENT_CN.md#supported-payment-methods",
    );
    for (const link of paymentLinks) {
      expect(link.attributes("href")).toContain("docs/PAYMENT");
    }
  });

  it("does not submit removed visible payment method settings", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);
    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton).toBeDefined();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    const payload = updateSettings.mock.calls[0]?.[0];
    expect(payload).not.toHaveProperty("payment_visible_method_alipay_source");
    expect(payload).not.toHaveProperty("payment_visible_method_wxpay_source");
    expect(payload).not.toHaveProperty("payment_visible_method_alipay_enabled");
    expect(payload).not.toHaveProperty("payment_visible_method_wxpay_enabled");
  });

  it("updates provider enablement immediately and reloads providers", async () => {
    const provider = {
      id: 7,
      provider_key: "alipay",
      name: "Official Alipay",
      config: {},
      supported_types: ["alipay"],
      enabled: false,
      payment_mode: "",
      refund_enabled: false,
      allow_user_refund: false,
      limits: "",
      sort_order: 0,
    };
    getProviders.mockReset();
    getProviders
      .mockResolvedValueOnce({ data: [provider] })
      .mockResolvedValueOnce({ data: [{ ...provider, enabled: true }] });
    updateProvider.mockResolvedValue({ data: { ...provider, enabled: true } });

    const PaymentProviderListStub = defineComponent({
      emits: ["toggleField"],
      setup(_, { emit }) {
        return () =>
          h(
            "button",
            {
              class: "provider-toggle-stub",
              onClick: () => emit("toggleField", provider, "enabled"),
            },
            "toggle provider",
          );
      },
    });

    const wrapper = mount(SettingsView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          RouterLink: { template: "<a><slot /></a>" },
          Select: SelectStub,
          Toggle: ToggleStub,
          Icon: true,
          ConfirmDialog: true,
          PaymentProviderList: PaymentProviderListStub,
          PaymentProviderDialog: true,
          SubscriptionPackageBadge: true,
          GroupBadge: true,
          GroupOptionItem: true,
          ProxySelector: true,
          ImageUpload: ImageUploadStub,
          BackupSettingsSection: true,
        },
      },
    });

    await flushPromises();
    await openPaymentTab(wrapper);
    await wrapper.get(".provider-toggle-stub").trigger("click");
    await flushPromises();

    expect(updateProvider).toHaveBeenCalledWith(7, { enabled: true });
    expect(getProviders).toHaveBeenCalledTimes(2);
  });

  it("passes translated upload and remove labels to the payment help image uploader", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    const imageUploads = wrapper.findAll(".image-upload-stub");
    expect(imageUploads.length).toBeGreaterThan(0);

    const paymentHelpImageUpload = imageUploads.find(
      (node) => node.attributes("data-placeholder") === "admin.settings.payment.helpImagePlaceholder",
    );

    expect(paymentHelpImageUpload).toBeDefined();
    expect(paymentHelpImageUpload?.attributes("data-upload-label")).toBe("上传图片");
    expect(paymentHelpImageUpload?.attributes("data-remove-label")).toBe("移除");
  });
});

describe("admin SettingsView wechat connect controls", () => {
  beforeEach(() => {
    getSettings.mockReset();
    updateSettings.mockReset();
    getAdminApiKey.mockReset();
    getGroups.mockReset();
    getProviders.mockReset();
    getPlans.mockReset();
    updateProvider.mockReset();
    createProvider.mockReset();
    deleteProvider.mockReset();
    fetchPublicSettings.mockReset();
    adminSettingsFetch.mockReset();
    showError.mockReset();
    showSuccess.mockReset();

    getSettings.mockResolvedValue({
      ...baseSettingsResponse,
      payment_visible_method_wxpay_source: "official_wxpay",
    });
    updateSettings.mockImplementation(async (payload) => ({
      ...baseSettingsResponse,
      payment_visible_method_wxpay_source: "official_wxpay",
      ...payload,
    }));
    getAdminApiKey.mockResolvedValue({
      exists: false,
      masked_key: "",
    });
    getGroups.mockResolvedValue([]);
    getProviders.mockResolvedValue({
      data: [],
    });
    getPlans.mockResolvedValue({
      data: sampleSubscriptionPlans,
    });
    fetchPublicSettings.mockResolvedValue(undefined);
    adminSettingsFetch.mockResolvedValue(undefined);
  });

  it("loads and echoes WeChat Connect fields from the backend payload", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-app-id"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("wx-app-id-123");
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-open-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(false);
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(true);
    expect(wrapper.find('[data-testid="wechat-connect-scopes"]').exists()).toBe(
      false,
    );
    expect(
      wrapper
        .get('[data-testid="wechat-connect-mp-app-secret"]')
        .attributes("placeholder"),
    ).toContain("密钥已配置");
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-frontend-redirect-url"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("/auth/wechat/callback");
  });

  it("links GitHub OAuth Apps guide to GitHub developer settings", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      github_oauth_enabled: true,
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    const link = wrapper.get('[data-testid="github-oauth-apps-guide-link"]');
    expect(link.text()).toContain("OAuth Apps");
    expect(link.attributes("href")).toBe("https://github.com/settings/developers");
    expect(link.attributes("target")).toBe("_blank");
    expect(link.attributes("rel")).toContain("noopener");
  });

  it("saves WeChat Connect fields using the backend contract and clears the secret after save", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    await wrapper
      .get('[data-testid="wechat-connect-mp-app-id"]')
      .setValue("wx-app-id-updated");
    await wrapper
      .get('[data-testid="wechat-connect-mp-app-secret"]')
      .setValue("new-secret");
    await wrapper
      .get('[data-testid="wechat-connect-open-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="wechat-connect-mp-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="wechat-connect-redirect-url"]')
      .setValue("https://admin.example.com/api/v1/auth/oauth/wechat/callback");
    await wrapper
      .get('[data-testid="wechat-connect-frontend-redirect-url"]')
      .setValue("/auth/wechat/callback");
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        wechat_connect_enabled: true,
        wechat_connect_app_id: "wx-app-id-updated",
        wechat_connect_open_enabled: true,
        wechat_connect_mp_enabled: true,
        wechat_connect_mp_app_id: "wx-app-id-updated",
        wechat_connect_mp_app_secret: "new-secret",
        wechat_connect_redirect_url:
          "https://admin.example.com/api/v1/auth/oauth/wechat/callback",
        wechat_connect_frontend_redirect_url: "/auth/wechat/callback",
      }),
    );
    expect(
      (
        wrapper.get('[data-testid="wechat-connect-mp-app-secret"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("");
    expect(
      wrapper
        .get('[data-testid="wechat-connect-mp-app-secret"]')
        .attributes("placeholder"),
    ).toContain("密钥已配置");
  });

  it("derives the stored WeChat Connect mode from the selected channel capabilities", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    await wrapper
      .get('[data-testid="wechat-connect-open-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="wechat-connect-mp-enabled"]')
      .setValue(false);
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        wechat_connect_open_enabled: true,
        wechat_connect_mp_enabled: false,
        wechat_connect_mode: "open",
      }),
    );
  });

  it("exposes payment routing and cancellation safety controls and saves selected values", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    await wrapper
      .get('[data-testid="payment-load-balance-strategy"]')
      .setValue("least-amount");
    await wrapper
      .get('[data-testid="payment-cancel-rate-limit-enabled"]')
      .setValue(true);
    await wrapper
      .get('[data-testid="payment-cancel-rate-limit-max"]')
      .setValue(5);
    await wrapper
      .get('[data-testid="payment-cancel-rate-limit-window"]')
      .setValue(2);
    await wrapper
      .get('[data-testid="payment-cancel-rate-limit-unit"]')
      .setValue("hour");
    await wrapper
      .get('[data-testid="payment-cancel-rate-limit-window-mode"]')
      .setValue("fixed");

    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton).toBeDefined();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        payment_load_balance_strategy: "least-amount",
        payment_cancel_rate_limit_enabled: true,
        payment_cancel_rate_limit_max: 5,
        payment_cancel_rate_limit_window: 2,
        payment_cancel_rate_limit_unit: "hour",
        payment_cancel_rate_limit_window_mode: "fixed",
      }),
    );
  });

  it("normalizes the stored underscore payment routing alias before displaying and saving settings", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      payment_load_balance_strategy: "round_robin",
    });
    const wrapper = mountView();

    await flushPromises();
    await openPaymentTab(wrapper);

    expect(
      (
        wrapper.get('[data-testid="payment-load-balance-strategy"]')
          .element as HTMLSelectElement
      ).value,
    ).toBe("round-robin");

    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton).toBeDefined();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        payment_load_balance_strategy: "round-robin",
      }),
    );
  });

  it("collapses auth source defaults until the source is enabled", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openUsersTab(wrapper);

    expect(
      (
        wrapper.get('[data-testid="auth-source-email-enabled"]')
          .element as HTMLInputElement
      ).checked,
    ).toBe(false);
    expect(
      wrapper.find('[data-testid="auth-source-email-panel"]').exists(),
    ).toBe(false);
    expect(wrapper.text()).not.toContain("注册即授权");

    await wrapper
      .get('[data-testid="auth-source-email-enabled"]')
      .setValue(true);

    expect(
      wrapper.find('[data-testid="auth-source-email-panel"]').exists(),
    ).toBe(true);
    expect(wrapper.text()).toContain("首次绑定时授权");
  });

  it("saves affiliate per-invitee rebate cap from the user defaults tab", async () => {
    const wrapper = mountView();

    await flushPromises();
    await openUsersTab(wrapper);
    await wrapper.get('[data-testid="affiliate-enabled"]').setValue(true);
    await flushPromises();
    await wrapper.get('[data-testid="affiliate-rebate-per-invitee-cap"]').setValue(12.5);

    const saveButton = wrapper
      .findAll("button")
      .find((node) => node.text().includes("common.save"));

    expect(saveButton).toBeDefined();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        affiliate_rebate_per_invitee_cap: 12.5,
      }),
    );
  });

  it("preserves optional OIDC security flags instead of forcing them on save", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      oidc_connect_enabled: true,
      oidc_connect_use_pkce: false,
      oidc_connect_validate_id_token: false,
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        oidc_connect_use_pkce: false,
        oidc_connect_validate_id_token: false,
      }),
    );
  });

  it("renders LinuxDo, DingTalk, Google, and advanced OIDC settings from the backend contract", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      linuxdo_connect_enabled: true,
      linuxdo_connect_client_id: "linuxdo-client",
      linuxdo_connect_client_secret_configured: true,
      linuxdo_connect_redirect_url: "https://admin.example.com/api/v1/auth/oauth/linuxdo/callback",
      dingtalk_connect_enabled: true,
      dingtalk_connect_client_id: "dingtalk-client",
      dingtalk_connect_client_secret_configured: true,
      dingtalk_connect_redirect_url: "https://admin.example.com/api/v1/auth/oauth/dingtalk/callback",
      dingtalk_connect_corp_restriction_policy: "internal_only",
      dingtalk_connect_internal_corp_id: "ding-corp",
      dingtalk_connect_bypass_registration: true,
      dingtalk_connect_sync_corp_email: true,
      dingtalk_connect_sync_display_name: true,
      dingtalk_connect_sync_dept: true,
      dingtalk_connect_sync_corp_email_attr_key: "corp_email",
      dingtalk_connect_sync_display_name_attr_key: "corp_name",
      dingtalk_connect_sync_dept_attr_key: "corp_dept",
      dingtalk_connect_sync_corp_email_attr_name: "企业邮箱",
      dingtalk_connect_sync_display_name_attr_name: "姓名",
      dingtalk_connect_sync_dept_attr_name: "部门",
      google_oauth_enabled: true,
      google_oauth_client_id: "google-client",
      google_oauth_client_secret_configured: true,
      google_oauth_redirect_url: "https://admin.example.com/api/v1/auth/oauth/google/callback",
      google_oauth_frontend_redirect_url: "/auth/oauth/callback",
      oidc_connect_enabled: true,
      oidc_connect_discovery_url: "https://issuer.example.com/.well-known/openid-configuration",
      oidc_connect_authorize_url: "https://issuer.example.com/auth",
      oidc_connect_token_url: "https://issuer.example.com/token",
      oidc_connect_userinfo_url: "https://issuer.example.com/userinfo",
      oidc_connect_jwks_url: "https://issuer.example.com/jwks",
      oidc_connect_redirect_url: "https://admin.example.com/api/v1/auth/oauth/oidc/callback",
      oidc_connect_token_auth_method: "client_secret_basic",
      oidc_connect_allowed_signing_algs: "RS256",
      oidc_connect_clock_skew_seconds: 90,
      oidc_connect_require_email_verified: true,
      oidc_connect_userinfo_email_path: "email",
      oidc_connect_userinfo_id_path: "sub",
      oidc_connect_userinfo_username_path: "preferred_username",
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    expect((wrapper.get('[data-testid="linuxdo-connect-client-id"]').element as HTMLInputElement).value).toBe("linuxdo-client");
    expect(wrapper.get('[data-testid="linuxdo-connect-client-secret"]').attributes("placeholder")).toContain("admin.settings.secretConfiguredPlaceholder");
    expect((wrapper.get('[data-testid="dingtalk-connect-internal-corp-id"]').element as HTMLInputElement).value).toBe("ding-corp");
    expect((wrapper.get('[data-testid="dingtalk-connect-bypass-registration"]').element as HTMLInputElement).checked).toBe(true);
    expect((wrapper.get('[data-testid="google-oauth-client-id"]').element as HTMLInputElement).value).toBe("google-client");
    expect(wrapper.get('[data-testid="google-oauth-client-secret"]').attributes("placeholder")).toContain("admin.settings.secretConfiguredPlaceholder");
    expect((wrapper.get('[data-testid="oidc-connect-discovery-url"]').element as HTMLInputElement).value).toContain(".well-known");
    expect((wrapper.get('[data-testid="oidc-connect-token-auth-method"]').element as HTMLSelectElement).value).toBe("client_secret_basic");
    expect((wrapper.get('[data-testid="oidc-connect-require-email-verified"]').element as HTMLInputElement).checked).toBe(true);
    expect((wrapper.get('[data-testid="oidc-connect-userinfo-username-path"]').element as HTMLInputElement).value).toBe("preferred_username");
  });

  it("saves restored OAuth provider settings while leaving blank secrets out of the payload", async () => {
    getSettings.mockResolvedValueOnce({
      ...baseSettingsResponse,
      linuxdo_connect_client_secret_configured: true,
      dingtalk_connect_client_secret_configured: true,
      google_oauth_client_secret_configured: true,
      oidc_connect_client_secret_configured: true,
    });

    const wrapper = mountView();

    await flushPromises();
    await openSecurityTab(wrapper);

    await wrapper.get('[data-testid="linuxdo-connect-enabled"]').setValue(true);
    await wrapper.get('[data-testid="linuxdo-connect-client-id"]').setValue("linuxdo-updated");
    await wrapper.get('[data-testid="dingtalk-connect-enabled"]').setValue(true);
    await wrapper.get('[data-testid="dingtalk-connect-client-id"]').setValue("dingtalk-updated");
    await wrapper.get('[data-testid="google-oauth-enabled"]').setValue(true);
    await wrapper.get('[data-testid="google-oauth-client-id"]').setValue("google-updated");
    await wrapper.get('[data-testid="oidc-connect-discovery-url"]').setValue("https://issuer.example.com/.well-known/openid-configuration");
    await wrapper.get('[data-testid="oidc-connect-token-auth-method"]').setValue("client_secret_basic");
    await wrapper.find("form").trigger("submit.prevent");
    await flushPromises();

    const payload = updateSettings.mock.calls[0]?.[0];
    expect(payload).toEqual(expect.objectContaining({
      linuxdo_connect_enabled: true,
      linuxdo_connect_client_id: "linuxdo-updated",
      dingtalk_connect_enabled: true,
      dingtalk_connect_client_id: "dingtalk-updated",
      google_oauth_enabled: true,
      google_oauth_client_id: "google-updated",
      oidc_connect_discovery_url: "https://issuer.example.com/.well-known/openid-configuration",
      oidc_connect_token_auth_method: "client_secret_basic",
    }));
    expect(payload).not.toHaveProperty("linuxdo_connect_client_secret");
    expect(payload).not.toHaveProperty("dingtalk_connect_client_secret");
    expect(payload).not.toHaveProperty("google_oauth_client_secret");
    expect(payload).not.toHaveProperty("oidc_connect_client_secret");
  });
});
