package handler

import (
	"github.com/Wei-Shaw/socialops/internal/handler/admin"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/google/wire"
)

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	announcementHandler *admin.AnnouncementHandler,
	backupHandler *admin.BackupHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	paymentHandler *admin.PaymentHandler,
	affiliateHandler *admin.AffiliateHandler,
	accountWorkbenchAdminHandler *admin.AccountWorkbenchAdminHandler,
	totalAccountsHandler *admin.TotalAccountsHandler,
	globalProxyHandler *admin.GlobalProxyHandler,
) *AdminHandlers {
	return &AdminHandlers{
		Dashboard:        dashboardHandler,
		User:             userHandler,
		Group:            groupHandler,
		Announcement:     announcementHandler,
		Backup:           backupHandler,
		Redeem:           redeemHandler,
		Promo:            promoHandler,
		Setting:          settingHandler,
		System:           systemHandler,
		Subscription:     subscriptionHandler,
		UserAttribute:    userAttributeHandler,
		Payment:          paymentHandler,
		Affiliate:        affiliateHandler,
		AccountWorkbench: accountWorkbenchAdminHandler,
		TotalAccounts:    totalAccountsHandler,
		GlobalProxies:    globalProxyHandler,
	}
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo, notificationEmailService *service.NotificationEmailService) *SettingHandler {
	h := NewSettingHandler(settingService, buildInfo.Version)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvideAdminSettingHandler creates admin.SettingHandler without OpsService dependency.
func ProvideAdminSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, paymentConfigService *service.PaymentConfigService, paymentService *service.PaymentService, userAttributeService *service.UserAttributeService, notificationEmailService *service.NotificationEmailService) *admin.SettingHandler {
	h := admin.NewSettingHandler(settingService, emailService, turnstileService, paymentConfigService, paymentService, userAttributeService)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvidePaymentHandler wires the user-facing payment handler.
func ProvidePaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return NewPaymentHandler(paymentService, configService)
}

// ProvideAccountWorkbenchAdminHandler wires admin account pool dependencies.
func ProvideAccountWorkbenchAdminHandler(svc *service.SocialAccountService, ipSvc *service.SocialIPService, billing *service.SocialBillingService, executor *service.SocialTaskExecutor) *admin.AccountWorkbenchAdminHandler {
	return admin.NewAccountWorkbenchAdminHandler(svc, ipSvc, billing, executor)
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	announcementHandler *AnnouncementHandler,
	adminHandlers *AdminHandlers,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	paymentHandler *PaymentHandler,
	paymentWebhookHandler *PaymentWebhookHandler,
	accountWorkbenchHandler *AccountWorkbenchHandler,
	proxyHandler *ProxyHandler,
	taskSettingsHandler *TaskSettingsHandler,
) *Handlers {
	return &Handlers{
		Auth:             authHandler,
		User:             userHandler,
		Usage:            usageHandler,
		Redeem:           redeemHandler,
		Subscription:     subscriptionHandler,
		Announcement:     announcementHandler,
		Admin:            adminHandlers,
		Setting:          settingHandler,
		Totp:             totpHandler,
		Payment:          paymentHandler,
		PaymentWebhook:   paymentWebhookHandler,
		AccountWorkbench: accountWorkbenchHandler,
		Proxy:            proxyHandler,
		TaskSettings:     taskSettingsHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	NewTotpHandler,
	ProvideSettingHandler,
	ProvidePaymentHandler,
	NewPaymentWebhookHandler,
	NewAccountWorkbenchHandlerWithGlobalProxies,
	NewProxyHandler,
	NewTaskSettingsHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	admin.NewGroupHandler,
	admin.NewAnnouncementHandler,
	admin.NewBackupHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	ProvideAdminSettingHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUserAttributeHandler,
	admin.NewPaymentHandler,
	admin.NewAffiliateHandler,
	ProvideAccountWorkbenchAdminHandler,
	admin.NewTotalAccountsHandler,
	admin.NewGlobalProxyHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
