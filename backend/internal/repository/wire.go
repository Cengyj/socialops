package repository

import (
	"database/sql"
	"errors"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// ProvideGitHubReleaseClient 创建 GitHub Release 客户端
func ProvideGitHubReleaseClient(cfg *config.Config) service.GitHubReleaseClient {
	return NewGitHubReleaseClient(cfg.Update.ProxyURL, cfg.Security.ProxyFallback.AllowDirectOnError)
}

// ProviderSet is the Wire provider set for all repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewAPIKeyRepository,
	NewRedeemCodeRepository,
	NewPromoCodeRepository,
	NewAnnouncementRepository,
	NewAnnouncementReadRepository,
	NewSettingRepository,
	NewGroupRepository,
	NewUserSubscriptionRepository,
	NewUserAttributeDefinitionRepository,
	NewUserAttributeValueRepository,
	NewUserGroupRateRepository,
	NewAffiliateRepository,
	NewUsageCleanupRepository,
	NewUsageLogRepository,

	// Cache implementations
	NewBillingCache,
	NewAPIKeyCache,
	NewEmailCache,
	NewIdentityCache,
	NewRedeemCache,
	NewUpdateCache,
	NewTotpCache,
	NewRefreshTokenCache,

	// Encryptors
	NewAESEncryptor,

	// Backup infrastructure
	NewPgDumper,
	NewS3BackupStoreFactory,

	// HTTP service ports
	NewTurnstileVerifier,
	ProvideGitHubReleaseClient,

	ProvideEnt,
	ProvideSQLDB,
	ProvideRedis,
)

// ProvideEnt 为依赖注入提供 Ent 客户端。
func ProvideEnt(cfg *config.Config) (*ent.Client, error) {
	client, _, err := InitEnt(cfg)
	return client, err
}

// ProvideSQLDB 从 Ent 客户端提取底层的 *sql.DB 连接。
func ProvideSQLDB(client *ent.Client) (*sql.DB, error) {
	if client == nil {
		return nil, errors.New("nil ent client")
	}
	drv, ok := client.Driver().(*entsql.Driver)
	if !ok {
		return nil, errors.New("ent driver does not expose *sql.DB")
	}
	return drv.DB(), nil
}

// ProvideRedis 为依赖注入提供 Redis 客户端。
func ProvideRedis(cfg *config.Config) *redis.Client {
	return InitRedis(cfg)
}
