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

func ProvideIdempotencyCoordinator(repo IdempotencyRepository, _ *config.Config) *IdempotencyCoordinator {
	coordinator := NewIdempotencyCoordinator(repo, DefaultIdempotencyConfig())
	SetDefaultIdempotencyCoordinator(coordinator)
	return coordinator
}

func ProvideSystemOperationLockService(repo IdempotencyRepository, _ *config.Config) *SystemOperationLockService {
	return NewSystemOperationLockService(repo, DefaultIdempotencyConfig())
}

func ProvideBackupService(settingRepo SettingRepository, cfg *config.Config, encryptor SecretEncryptor, storeFactory BackupObjectStoreFactory, dumper DBDumper) *BackupService {
	svc := NewBackupService(settingRepo, cfg, encryptor, storeFactory, dumper)
	svc.Start()
	return svc
}

func ProvideSettingService(settingRepo SettingRepository, groupRepo GroupRepository, paymentConfigService *PaymentConfigService, cfg *config.Config) *SettingService {
	svc := NewSettingService(settingRepo, cfg)
	svc.SetDefaultSubscriptionGroupReader(groupRepo)
	svc.SetDefaultSubscriptionPlanReader(paymentConfigService)
	return svc
}

func ProvidePaymentConfigService(entClient *dbent.Client, settingRepo SettingRepository, key payment.EncryptionKey) *PaymentConfigService {
	return NewPaymentConfigService(entClient, settingRepo, key)
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

func ProvideAdminService(
	userRepo UserRepository,
	groupRepo GroupRepository,
	redeemCodeRepo RedeemCodeRepository,
	userGroupRateRepo UserGroupRateRepository,
	billingCacheService *BillingCacheService,
	entClient *dbent.Client,
	settingService *SettingService,
	defaultSubAssigner DefaultSubscriptionAssigner,
) AdminService {
	return NewAdminService(
		userRepo,
		groupRepo,
		redeemCodeRepo,
		userGroupRateRepo,
		billingCacheService,
		entClient,
		settingService,
		defaultSubAssigner,
	)
}

type DashboardService struct {
	repo UsageLogRepository
}

func ProvideDashboardService(repo UsageLogRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func ProvideUsageService(repo UsageLogRepository, entClient *dbent.Client) *UsageService {
	return NewUsageService(repo).WithMediaResolver(NewSocialTaskMediaService(entClient))
}

func ProvideConcurrencyService() *ConcurrencyService {
	return NewConcurrencyService(nil)
}

func ProvideSocialTaskExecutor(entClient *dbent.Client, billing *SocialBillingService, cfg *config.Config, encryptor ExecutionAuthEncryptor) *SocialTaskExecutor {
	svc := NewSocialTaskExecutor(entClient, billing, SocialTaskExecutorConfig{}).WithCredentialEncryptor(encryptor)
	ipSvc := NewSocialIPService(entClient)
	proxyHealthReporter := func(ctx context.Context, proxyID int64) {
		_ = ipSvc.MarkExecutionReachable(ctx, proxyID)
	}
	registrar := NewTwitterAccountCredentialRegistrar().
		WithDeviceParamProvider(NewHTTPDeviceParamProvider(TwitterDeviceParamConfig{
			URL:        cfg.TwitterLogin.DeviceParamsURL,
			Collection: cfg.TwitterLogin.DeviceParamsCollection,
		})).
		WithEmailCodeResolver(NewHTTPEmailCodeResolver(TwitterEmailCodeConfig{
			URL: cfg.TwitterLogin.EmailCodeURL,
		})).
		WithProxyHealthReporter(proxyHealthReporter)
	twitter := NewTwitterExecutor().
		WithMediaResolver(NewSocialTaskMediaService(entClient)).
		WithLoginRegistrar(registrar).
		WithCredentialEncryptor(encryptor).
		WithProxyHealthReporter(proxyHealthReporter)
	svc.RegisterPlatformExecutor("x_twitter", twitter)
	svc.Start()
	return svc
}

func ProvideSocialAccountService(entClient *dbent.Client, encryptor ExecutionAuthEncryptor) *SocialAccountService {
	return NewSocialAccountServiceWithCredentialEncryptor(entClient, encryptor)
}

var ProviderSet = wire.NewSet(
	NewAuthService,
	NewUserService,
	NewRedeemService,
	NewPromoService,
	ProvideUsageService,
	ProvideDashboardService,
	ProvideBillingCacheService,
	NewAnnouncementService,
	ProvideAdminService,
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
	NewAffiliateService,
	NewUserAttributeService,
	NewNotificationEmailService,
	ProvideIdempotencyCoordinator,
	ProvideSystemOperationLockService,
	ProvideConcurrencyService,
	ProvideSocialAccountService,
	NewSocialBillingService,
	NewSocialIPService,
	NewGlobalProxyService,
	ProvideSocialTaskExecutor,
	NewSocialIPChecker,
	wire.Bind(new(DefaultSubscriptionAssigner), new(*SubscriptionService)),
)
