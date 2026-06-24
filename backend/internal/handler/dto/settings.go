package dto

import (
	"encoding/json"
	"strings"
)

type CustomMenuItem struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	IconSVG    string `json:"icon_svg"`
	URL        string `json:"url"`
	PageSlug   string `json:"page_slug,omitempty"`
	Visibility string `json:"visibility"`
	SortOrder  int    `json:"sort_order"`
}

type CustomEndpoint struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type SystemSettings struct {
	RegistrationEnabled              bool                     `json:"registration_enabled"`
	EmailVerifyEnabled               bool                     `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string                 `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool                     `json:"promo_code_enabled"`
	PasswordResetEnabled             bool                     `json:"password_reset_enabled"`
	FrontendURL                      string                   `json:"frontend_url"`
	InvitationCodeEnabled            bool                     `json:"invitation_code_enabled"`
	TotpEnabled                      bool                     `json:"totp_enabled"`
	TotpEncryptionKeyConfigured      bool                     `json:"totp_encryption_key_configured"`
	LoginAgreementEnabled            bool                     `json:"login_agreement_enabled"`
	LoginAgreementMode               string                   `json:"login_agreement_mode"`
	LoginAgreementUpdatedAt          string                   `json:"login_agreement_updated_at"`
	LoginAgreementDocuments          []LoginAgreementDocument `json:"login_agreement_documents"`

	SMTPHost               string `json:"smtp_host"`
	SMTPPort               int    `json:"smtp_port"`
	SMTPUsername           string `json:"smtp_username"`
	SMTPPasswordConfigured bool   `json:"smtp_password_configured"`
	SMTPFrom               string `json:"smtp_from_email"`
	SMTPFromName           string `json:"smtp_from_name"`
	SMTPUseTLS             bool   `json:"smtp_use_tls"`

	TurnstileEnabled             bool   `json:"turnstile_enabled"`
	TurnstileSiteKey             string `json:"turnstile_site_key"`
	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`

	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`

	DingTalkConnectEnabled                 bool   `json:"dingtalk_connect_enabled"`
	DingTalkConnectClientID                string `json:"dingtalk_connect_client_id"`
	DingTalkConnectClientSecretConfigured  bool   `json:"dingtalk_connect_client_secret_configured"`
	DingTalkConnectRedirectURL             string `json:"dingtalk_connect_redirect_url"`
	DingTalkConnectCorpRestrictionPolicy   string `json:"dingtalk_connect_corp_restriction_policy"`
	DingTalkConnectInternalCorpID          string `json:"dingtalk_connect_internal_corp_id"`
	DingTalkConnectBypassRegistration      bool   `json:"dingtalk_connect_bypass_registration"`
	DingTalkConnectSyncCorpEmail           bool   `json:"dingtalk_connect_sync_corp_email"`
	DingTalkConnectSyncDisplayName         bool   `json:"dingtalk_connect_sync_display_name"`
	DingTalkConnectSyncDept                bool   `json:"dingtalk_connect_sync_dept"`
	DingTalkConnectSyncCorpEmailAttrKey    string `json:"dingtalk_connect_sync_corp_email_attr_key"`
	DingTalkConnectSyncDisplayNameAttrKey  string `json:"dingtalk_connect_sync_display_name_attr_key"`
	DingTalkConnectSyncDeptAttrKey         string `json:"dingtalk_connect_sync_dept_attr_key"`
	DingTalkConnectSyncCorpEmailAttrName   string `json:"dingtalk_connect_sync_corp_email_attr_name"`
	DingTalkConnectSyncDisplayNameAttrName string `json:"dingtalk_connect_sync_display_name_attr_name"`
	DingTalkConnectSyncDeptAttrName        string `json:"dingtalk_connect_sync_dept_attr_name"`

	WeChatConnectEnabled                   bool   `json:"wechat_connect_enabled"`
	WeChatConnectAppID                     string `json:"wechat_connect_app_id"`
	WeChatConnectAppSecretConfigured       bool   `json:"wechat_connect_app_secret_configured"`
	WeChatConnectOpenAppID                 string `json:"wechat_connect_open_app_id"`
	WeChatConnectOpenAppSecretConfigured   bool   `json:"wechat_connect_open_app_secret_configured"`
	WeChatConnectMPAppID                   string `json:"wechat_connect_mp_app_id"`
	WeChatConnectMPAppSecretConfigured     bool   `json:"wechat_connect_mp_app_secret_configured"`
	WeChatConnectMobileAppID               string `json:"wechat_connect_mobile_app_id"`
	WeChatConnectMobileAppSecretConfigured bool   `json:"wechat_connect_mobile_app_secret_configured"`
	WeChatConnectOpenEnabled               bool   `json:"wechat_connect_open_enabled"`
	WeChatConnectMPEnabled                 bool   `json:"wechat_connect_mp_enabled"`
	WeChatConnectMobileEnabled             bool   `json:"wechat_connect_mobile_enabled"`
	WeChatConnectMode                      string `json:"wechat_connect_mode"`
	WeChatConnectScopes                    string `json:"wechat_connect_scopes"`
	WeChatConnectRedirectURL               string `json:"wechat_connect_redirect_url"`
	WeChatConnectFrontendRedirectURL       string `json:"wechat_connect_frontend_redirect_url"`

	OIDCConnectEnabled                bool   `json:"oidc_connect_enabled"`
	OIDCConnectProviderName           string `json:"oidc_connect_provider_name"`
	OIDCConnectClientID               string `json:"oidc_connect_client_id"`
	OIDCConnectClientSecretConfigured bool   `json:"oidc_connect_client_secret_configured"`
	OIDCConnectIssuerURL              string `json:"oidc_connect_issuer_url"`
	OIDCConnectDiscoveryURL           string `json:"oidc_connect_discovery_url"`
	OIDCConnectAuthorizeURL           string `json:"oidc_connect_authorize_url"`
	OIDCConnectTokenURL               string `json:"oidc_connect_token_url"`
	OIDCConnectUserInfoURL            string `json:"oidc_connect_userinfo_url"`
	OIDCConnectJWKSURL                string `json:"oidc_connect_jwks_url"`
	OIDCConnectScopes                 string `json:"oidc_connect_scopes"`
	OIDCConnectRedirectURL            string `json:"oidc_connect_redirect_url"`
	OIDCConnectFrontendRedirectURL    string `json:"oidc_connect_frontend_redirect_url"`
	OIDCConnectTokenAuthMethod        string `json:"oidc_connect_token_auth_method"`
	OIDCConnectUsePKCE                bool   `json:"oidc_connect_use_pkce"`
	OIDCConnectValidateIDToken        bool   `json:"oidc_connect_validate_id_token"`
	OIDCConnectAllowedSigningAlgs     string `json:"oidc_connect_allowed_signing_algs"`
	OIDCConnectClockSkewSeconds       int    `json:"oidc_connect_clock_skew_seconds"`
	OIDCConnectRequireEmailVerified   bool   `json:"oidc_connect_require_email_verified"`
	OIDCConnectUserInfoEmailPath      string `json:"oidc_connect_userinfo_email_path"`
	OIDCConnectUserInfoIDPath         string `json:"oidc_connect_userinfo_id_path"`
	OIDCConnectUserInfoUsernamePath   string `json:"oidc_connect_userinfo_username_path"`

	GitHubOAuthEnabled                bool   `json:"github_oauth_enabled"`
	GitHubOAuthClientID               string `json:"github_oauth_client_id"`
	GitHubOAuthClientSecretConfigured bool   `json:"github_oauth_client_secret_configured"`
	GitHubOAuthRedirectURL            string `json:"github_oauth_redirect_url"`
	GitHubOAuthFrontendRedirectURL    string `json:"github_oauth_frontend_redirect_url"`
	GoogleOAuthEnabled                bool   `json:"google_oauth_enabled"`
	GoogleOAuthClientID               string `json:"google_oauth_client_id"`
	GoogleOAuthClientSecretConfigured bool   `json:"google_oauth_client_secret_configured"`
	GoogleOAuthRedirectURL            string `json:"google_oauth_redirect_url"`
	GoogleOAuthFrontendRedirectURL    string `json:"google_oauth_frontend_redirect_url"`

	SiteName                    string           `json:"site_name"`
	SiteLogo                    string           `json:"site_logo"`
	SiteSubtitle                string           `json:"site_subtitle"`
	APIBaseURL                  string           `json:"api_base_url"`
	ContactInfo                 string           `json:"contact_info"`
	DocURL                      string           `json:"doc_url"`
	HomeContent                 string           `json:"home_content"`
	PurchaseSubscriptionEnabled bool             `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string           `json:"purchase_subscription_url"`
	TableDefaultPageSize        int              `json:"table_default_page_size"`
	TablePageSizeOptions        []int            `json:"table_page_size_options"`
	CustomMenuItems             []CustomMenuItem `json:"custom_menu_items"`
	CustomEndpoints             []CustomEndpoint `json:"custom_endpoints"`

	DefaultConcurrency           int                          `json:"default_concurrency"`
	DefaultBalance               float64                      `json:"default_balance"`
	AffiliateRebateRate          float64                      `json:"affiliate_rebate_rate"`
	AffiliateRebateFreezeHours   int                          `json:"affiliate_rebate_freeze_hours"`
	AffiliateRebateDurationDays  int                          `json:"affiliate_rebate_duration_days"`
	AffiliateRebatePerInviteeCap float64                      `json:"affiliate_rebate_per_invitee_cap"`
	DefaultUserRPMLimit          int                          `json:"default_user_rpm_limit"`
	DefaultSubscriptions         []DefaultSubscriptionSetting `json:"default_subscriptions"`

	BackendModeEnabled bool `json:"backend_mode_enabled"`

	PaymentVisibleMethodAlipaySource  string `json:"payment_visible_method_alipay_source"`
	PaymentVisibleMethodWxpaySource   string `json:"payment_visible_method_wxpay_source"`
	PaymentVisibleMethodAlipayEnabled bool   `json:"payment_visible_method_alipay_enabled"`
	PaymentVisibleMethodWxpayEnabled  bool   `json:"payment_visible_method_wxpay_enabled"`

	BalanceLowNotifyEnabled         bool               `json:"balance_low_notify_enabled"`
	BalanceLowNotifyThreshold       float64            `json:"balance_low_notify_threshold"`
	BalanceLowNotifyRechargeURL     string             `json:"balance_low_notify_recharge_url"`
	SubscriptionExpiryNotifyEnabled bool               `json:"subscription_expiry_notify_enabled"`
	AccountQuotaNotifyEnabled       bool               `json:"account_quota_notify_enabled"`
	AccountQuotaNotifyEmails        []NotifyEmailEntry `json:"account_quota_notify_emails"`

	PaymentEnabled                   bool     `json:"payment_enabled"`
	PaymentMinAmount                 float64  `json:"payment_min_amount"`
	PaymentMaxAmount                 float64  `json:"payment_max_amount"`
	PaymentDailyLimit                float64  `json:"payment_daily_limit"`
	PaymentOrderTimeoutMin           int      `json:"payment_order_timeout_minutes"`
	PaymentMaxPendingOrders          int      `json:"payment_max_pending_orders"`
	PaymentEnabledTypes              []string `json:"payment_enabled_types"`
	PaymentBalanceDisabled           bool     `json:"payment_balance_disabled"`
	PaymentBalanceRechargeMultiplier float64  `json:"payment_balance_recharge_multiplier"`
	PaymentRechargeFeeRate           float64  `json:"payment_recharge_fee_rate"`
	PaymentLoadBalanceStrat          string   `json:"payment_load_balance_strategy"`
	PaymentProductNamePrefix         string   `json:"payment_product_name_prefix"`
	PaymentProductNameSuffix         string   `json:"payment_product_name_suffix"`
	PaymentHelpImageURL              string   `json:"payment_help_image_url"`
	PaymentHelpText                  string   `json:"payment_help_text"`
	PaymentCancelRateLimitEnabled    bool     `json:"payment_cancel_rate_limit_enabled"`
	PaymentCancelRateLimitMax        int      `json:"payment_cancel_rate_limit_max"`
	PaymentCancelRateLimitWindow     int      `json:"payment_cancel_rate_limit_window"`
	PaymentCancelRateLimitUnit       string   `json:"payment_cancel_rate_limit_unit"`
	PaymentCancelRateLimitMode       string   `json:"payment_cancel_rate_limit_window_mode"`
	PaymentAlipayForceQRCode         bool     `json:"payment_alipay_force_qrcode"`

	AffiliateEnabled bool `json:"affiliate_enabled"`
}

type DefaultSubscriptionSetting struct {
	PlanID       int64 `json:"plan_id,omitempty"`
	GroupID      int64 `json:"group_id,omitempty"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool                     `json:"registration_enabled"`
	EmailVerifyEnabled               bool                     `json:"email_verify_enabled"`
	ForceEmailOnThirdPartySignup     bool                     `json:"force_email_on_third_party_signup"`
	RegistrationEmailSuffixWhitelist []string                 `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool                     `json:"promo_code_enabled"`
	PasswordResetEnabled             bool                     `json:"password_reset_enabled"`
	InvitationCodeEnabled            bool                     `json:"invitation_code_enabled"`
	TotpEnabled                      bool                     `json:"totp_enabled"`
	LoginAgreementEnabled            bool                     `json:"login_agreement_enabled"`
	LoginAgreementMode               string                   `json:"login_agreement_mode"`
	LoginAgreementUpdatedAt          string                   `json:"login_agreement_updated_at"`
	LoginAgreementRevision           string                   `json:"login_agreement_revision"`
	LoginAgreementDocuments          []LoginAgreementDocument `json:"login_agreement_documents"`
	TurnstileEnabled                 bool                     `json:"turnstile_enabled"`
	TurnstileSiteKey                 string                   `json:"turnstile_site_key"`
	SiteName                         string                   `json:"site_name"`
	SiteLogo                         string                   `json:"site_logo"`
	SiteSubtitle                     string                   `json:"site_subtitle"`
	APIBaseURL                       string                   `json:"api_base_url"`
	ContactInfo                      string                   `json:"contact_info"`
	DocURL                           string                   `json:"doc_url"`
	HomeContent                      string                   `json:"home_content"`
	PurchaseSubscriptionEnabled      bool                     `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL          string                   `json:"purchase_subscription_url"`
	TableDefaultPageSize             int                      `json:"table_default_page_size"`
	TablePageSizeOptions             []int                    `json:"table_page_size_options"`
	CustomMenuItems                  []CustomMenuItem         `json:"custom_menu_items"`
	CustomEndpoints                  []CustomEndpoint         `json:"custom_endpoints"`
	DingTalkOAuthEnabled             bool                     `json:"dingtalk_oauth_enabled"`
	LinuxDoOAuthEnabled              bool                     `json:"linuxdo_oauth_enabled"`
	WeChatOAuthEnabled               bool                     `json:"wechat_oauth_enabled"`
	WeChatOAuthOpenEnabled           bool                     `json:"wechat_oauth_open_enabled"`
	WeChatOAuthMPEnabled             bool                     `json:"wechat_oauth_mp_enabled"`
	WeChatOAuthMobileEnabled         bool                     `json:"wechat_oauth_mobile_enabled"`
	OIDCOAuthEnabled                 bool                     `json:"oidc_oauth_enabled"`
	OIDCOAuthProviderName            string                   `json:"oidc_oauth_provider_name"`
	GitHubOAuthEnabled               bool                     `json:"github_oauth_enabled"`
	GoogleOAuthEnabled               bool                     `json:"google_oauth_enabled"`
	BackendModeEnabled               bool                     `json:"backend_mode_enabled"`
	PaymentEnabled                   bool                     `json:"payment_enabled"`
	Version                          string                   `json:"version"`
	BalanceLowNotifyEnabled          bool                     `json:"balance_low_notify_enabled"`
	AccountQuotaNotifyEnabled        bool                     `json:"account_quota_notify_enabled"`
	BalanceLowNotifyThreshold        float64                  `json:"balance_low_notify_threshold"`
	BalanceLowNotifyRechargeURL      string                   `json:"balance_low_notify_recharge_url"`

	AffiliateEnabled bool `json:"affiliate_enabled"`
}

type LoginAgreementDocument struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
}

type EmailTemplateEventOption struct {
	Value       string `json:"value"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
}

type EmailTemplateSummary struct {
	Event     string `json:"event"`
	Locale    string `json:"locale"`
	Subject   string `json:"subject"`
	IsCustom  bool   `json:"is_custom,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type EmailTemplateListResponse struct {
	Events       []EmailTemplateEventOption `json:"events"`
	Locales      []string                   `json:"locales"`
	Templates    []EmailTemplateSummary     `json:"templates,omitempty"`
	Placeholders []string                   `json:"placeholders,omitempty"`
}

type EmailTemplateDetail struct {
	Event        string   `json:"event"`
	Locale       string   `json:"locale"`
	Subject      string   `json:"subject"`
	HTML         string   `json:"html"`
	IsCustom     bool     `json:"is_custom,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	Placeholders []string `json:"placeholders,omitempty"`
}

type UpdateEmailTemplateRequest struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type PreviewEmailTemplateRequest struct {
	Event     string            `json:"event"`
	Locale    string            `json:"locale"`
	Subject   string            `json:"subject"`
	HTML      string            `json:"html"`
	Variables map[string]string `json:"variables,omitempty"`
}

type EmailTemplatePreviewResponse struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func ParseCustomMenuItems(raw string) []CustomMenuItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CustomMenuItem{}
	}
	var items []CustomMenuItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomMenuItem{}
	}
	return items
}

func ParseUserVisibleMenuItems(raw string) []CustomMenuItem {
	items := ParseCustomMenuItems(raw)
	filtered := make([]CustomMenuItem, 0, len(items))
	for _, item := range items {
		if item.Visibility == "user" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func ParseCustomEndpoints(raw string) []CustomEndpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CustomEndpoint{}
	}
	var items []CustomEndpoint
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomEndpoint{}
	}
	return items
}
