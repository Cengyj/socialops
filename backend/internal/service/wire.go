package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/payment"
	"github.com/google/wire"
)

type BuildInfo struct {
	Version   string
	BuildType string
}

func ProvideUpdateService(cache UpdateCache, githubClient GitHubReleaseClient, buildInfo BuildInfo) *UpdateService {
	return NewUpdateService(cache, githubClient, buildInfo.Version, buildInfo.BuildType)
}

func ProvideEmailQueueService(emailService *EmailService) *EmailQueueService {
	return NewEmailQueueService(emailService, 4)
}

func ProvideIdempotencyCleanupService(_ *config.Config) *IdempotencyCleanupService {
	return &IdempotencyCleanupService{}
}

func ProvideSystemOperationLockService(cfg *config.Config) *SystemOperationLockService {
	return NewSystemOperationLockService(nil, DefaultIdempotencyConfig())
}

func ProvideBackupService(settingRepo SettingRepository, cfg *config.Config, encryptor SecretEncryptor, storeFactory BackupObjectStoreFactory, dumper DBDumper) *BackupService {
	svc := NewBackupService(settingRepo, cfg, encryptor, storeFactory, dumper)
	svc.Start()
	return svc
}

func ProvideSettingService(settingRepo SettingRepository, groupRepo GroupRepository, cfg *config.Config) *SettingService {
	svc := NewSettingService(settingRepo, cfg)
	svc.SetDefaultSubscriptionGroupReader(groupRepo)
	_ = svc.LoadAPIKeyACLTrustForwardedIPSetting(context.Background())
	return svc
}

func ProvidePaymentConfigService(entClient *dbent.Client, settingRepo SettingRepository, key payment.EncryptionKey) *PaymentConfigService {
	return NewPaymentConfigService(entClient, settingRepo, key)
}

func ProvideBalanceNotifyService(_ *EmailService, _ SettingRepository, _ *NotificationEmailService) *BalanceNotifyService {
	return nil
}

func ProvidePaymentService(entClient *dbent.Client, registry *payment.Registry, loadBalancer payment.LoadBalancer, redeemService *RedeemService, subscriptionSvc *SubscriptionService, configService *PaymentConfigService, userRepo UserRepository, groupRepo GroupRepository, affiliateService *AffiliateService, notificationEmailService *NotificationEmailService) *PaymentService {
	svc := NewPaymentService(entClient, registry, loadBalancer, redeemService, subscriptionSvc, configService, userRepo, groupRepo, affiliateService)
	svc.SetNotificationEmailService(notificationEmailService)
	return svc
}

func ProvidePaymentOrderExpiryService(paymentSvc *PaymentService) *PaymentOrderExpiryService {
	svc := NewPaymentOrderExpiryService(paymentSvc, time.Minute)
	svc.Start()
	return svc
}

func ProvideSubscriptionService(groupRepo GroupRepository, userSubRepo UserSubscriptionRepository, billingCacheService *BillingCacheService, entClient *dbent.Client, cfg *config.Config) *SubscriptionService {
	return NewSubscriptionService(groupRepo, userSubRepo, billingCacheService, entClient, cfg)
}

func ProvideSubscriptionExpiryService(userSubRepo UserSubscriptionRepository, settingRepo SettingRepository, notificationEmailService *NotificationEmailService) *SubscriptionExpiryService {
	svc := NewSubscriptionExpiryService(userSubRepo, time.Minute)
	svc.SetSettingRepository(settingRepo)
	svc.SetNotificationEmailService(notificationEmailService)
	return svc
}

func ProvideBillingCacheService() *BillingCacheService {
	return &BillingCacheService{}
}

func ProvideAPIKeyService(apiKeyRepo APIKeyRepository, userRepo UserRepository, groupRepo GroupRepository, userSubRepo UserSubscriptionRepository, userGroupRateRepo UserGroupRateRepository, cache APIKeyCache, cfg *config.Config, billingCacheService *BillingCacheService) *APIKeyService {
	svc := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, userSubRepo, userGroupRateRepo, cache, cfg)
	svc.SetRateLimitCacheInvalidator(billingCacheService)
	return svc
}

func ProvideAPIKeyAuthCacheInvalidator(apiKeyService *APIKeyService) APIKeyAuthCacheInvalidator {
	apiKeyService.StartAuthCacheInvalidationSubscriber(context.Background())
	return apiKeyService
}

func ProvideAdminService(
	userRepo UserRepository,
	groupRepo GroupRepository,
	apiKeyRepo APIKeyRepository,
	redeemCodeRepo RedeemCodeRepository,
	userGroupRateRepo UserGroupRateRepository,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	entClient *dbent.Client,
	settingService *SettingService,
	defaultSubAssigner DefaultSubscriptionAssigner,
	userSubRepo UserSubscriptionRepository,
) AdminService {
	return NewAdminService(
		userRepo,
		groupRepo,
		apiKeyRepo,
		redeemCodeRepo,
		userGroupRateRepo,
		billingCacheService,
		authCacheInvalidator,
		entClient,
		settingService,
		defaultSubAssigner,
		userSubRepo,
	)
}

type GroupCapacityService struct{}

func ProvideGroupCapacityService() *GroupCapacityService { return &GroupCapacityService{} }

func ProvideUsageCleanupService(repo UsageCleanupRepository, cfg *config.Config) *UsageCleanupService {
	return NewUsageCleanupService(repo, nil, nil, cfg)
}

type DashboardService struct {
	repo UsageLogRepository
}
type DashboardStatsCache any

func ProvideDashboardService(repo UsageLogRepository, _ DashboardStatsCache, _ *config.Config) *DashboardService {
	return &DashboardService{repo: repo}
}

type socialOpsDashboardStatsCacheSkeleton struct{}

func ProvideDashboardStatsCache() DashboardStatsCache {
	return socialOpsDashboardStatsCacheSkeleton{}
}

func ProvideConcurrencyService() *ConcurrencyService {
	return NewConcurrencyService(nil)
}

func ProvideSocialTaskExecutor(entClient *dbent.Client, billing *SocialBillingService) *SocialTaskExecutor {
	svc := NewSocialTaskExecutor(entClient, billing, SocialTaskExecutorConfig{})
	svc.Start()
	return svc
}

func ProvideSocialAccountService(entClient *dbent.Client) *SocialAccountService {
	return NewSocialAccountService(entClient)
}

var ProviderSet = wire.NewSet(
	NewAuthService,
	NewUserService,
	ProvideAPIKeyService,
	ProvideAPIKeyAuthCacheInvalidator,
	NewRedeemService,
	NewPromoService,
	NewUsageService,
	ProvideDashboardService,
	ProvideBillingCacheService,
	NewAnnouncementService,
	ProvideAdminService,
	NewDataManagementService,
	NewEmailService,
	ProvideEmailQueueService,
	NewTurnstileService,
	ProvideUpdateService,
	ProvideSettingService,
	NewTotpService,
	ProvideBackupService,
	ProvidePaymentConfigService,
	ProvidePaymentService,
	ProvidePaymentOrderExpiryService,
	ProvideSubscriptionService,
	ProvideSubscriptionExpiryService,
	ProvideBalanceNotifyService,
	NewAffiliateService,
	NewUserAttributeService,
	NewNotificationEmailService,
	ProvideIdempotencyCleanupService,
	ProvideSystemOperationLockService,
	ProvideGroupCapacityService,
	ProvideUsageCleanupService,
	ProvideDashboardStatsCache,
	ProvideConcurrencyService,
	ProvideSocialAccountService,
	NewSocialBillingService,
	NewSocialIPService,
	ProvideSocialTaskExecutor,
	NewSocialIPChecker,
	NewPlanService,
	wire.Bind(new(DefaultSubscriptionAssigner), new(*SubscriptionService)),
)
