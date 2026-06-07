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
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	paymentHandler *admin.PaymentHandler,
	affiliateHandler *admin.AffiliateHandler,
	accountWorkbenchAdminHandler *admin.AccountWorkbenchAdminHandler,
	totalAccountsHandler *admin.TotalAccountsHandler,
) *AdminHandlers {
	return &AdminHandlers{
		Dashboard:        dashboardHandler,
		User:             userHandler,
		Group:            groupHandler,
		Announcement:     announcementHandler,
		DataManagement:   dataManagementHandler,
		Backup:           backupHandler,
		Redeem:           redeemHandler,
		Promo:            promoHandler,
		Setting:          settingHandler,
		System:           systemHandler,
		Subscription:     subscriptionHandler,
		UserAttribute:    userAttributeHandler,
		APIKey:           apiKeyHandler,
		Payment:          paymentHandler,
		Affiliate:        affiliateHandler,
		AccountWorkbench: accountWorkbenchAdminHandler,
		TotalAccounts:    totalAccountsHandler,
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

// ProvidePaymentHandler avoids exposing NewPaymentHandler's optional legacy
// variadic argument to Wire.
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
	apiKeyHandler *APIKeyHandler,
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
	planHandler *PlanHandler,
) *Handlers {
	return &Handlers{
		Auth:             authHandler,
		User:             userHandler,
		APIKey:           apiKeyHandler,
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
		Plan:             planHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	NewTotpHandler,
	ProvideSettingHandler,
	ProvidePaymentHandler,
	NewPaymentWebhookHandler,
	NewAccountWorkbenchHandler,
	NewProxyHandler,
	NewTaskSettingsHandler,
	NewPlanHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	admin.NewGroupHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	ProvideAdminSettingHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUserAttributeHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewPaymentHandler,
	admin.NewAffiliateHandler,
	ProvideAccountWorkbenchAdminHandler,
	admin.NewTotalAccountsHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
