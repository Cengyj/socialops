package handler

import (
	"github.com/Wei-Shaw/socialops/internal/handler/admin"
)

// AdminHandlers contains all admin-related HTTP handlers
type AdminHandlers struct {
	Dashboard        *admin.DashboardHandler
	User             *admin.UserHandler
	Group            *admin.GroupHandler
	Announcement     *admin.AnnouncementHandler
	Backup           *admin.BackupHandler
	Redeem           *admin.RedeemHandler
	Promo            *admin.PromoHandler
	Setting          *admin.SettingHandler
	System           *admin.SystemHandler
	Subscription     *admin.SubscriptionHandler
	UserAttribute    *admin.UserAttributeHandler
	Payment          *admin.PaymentHandler
	Affiliate        *admin.AffiliateHandler
	AccountWorkbench *admin.AccountWorkbenchAdminHandler
	TotalAccounts    *admin.TotalAccountsHandler
	GlobalProxies     *admin.GlobalProxyHandler
}

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth             *AuthHandler
	User             *UserHandler
	Usage            *UsageHandler
	Redeem           *RedeemHandler
	Subscription     *SubscriptionHandler
	Announcement     *AnnouncementHandler
	Admin            *AdminHandlers
	Setting          *SettingHandler
	Totp             *TotpHandler
	Payment          *PaymentHandler
	PaymentWebhook   *PaymentWebhookHandler
	AccountWorkbench *AccountWorkbenchHandler
	Proxy            *ProxyHandler
	TaskSettings     *TaskSettingsHandler
}

// BuildInfo contains build-time information
type BuildInfo struct {
	Version   string
	BuildType string // "source" for manual builds, "release" for CI builds
}
