package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/domain"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestUsageLogRepositoryListsSocialTaskLogs(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social@example.com").SetPasswordHash("hash").SaveX(ctx)
	otherUser := client.User.Create().SetEmail("usage-social-other@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("usage_social_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_other").
		SetAssignedUserID(otherUser.ID).
		SaveX(ctx)
	executedAt := time.Now().UTC().Truncate(time.Second)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetResultMessage("follow succeeded").
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(executedAt).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(otherAccount.ID).
		SetUserID(otherUser.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: user.ID})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, user.ID, items[0].UserID)
	require.Equal(t, account.ID, items[0].SocialAccountID)
	require.Equal(t, "x_twitter", items[0].Platform)
	require.Equal(t, "usage_social", items[0].AccountName)
	require.Equal(t, service.SocialTaskActionFollow, items[0].Operation)
	require.Equal(t, service.SocialTaskLogStatusSuccess, items[0].Status)
	require.Equal(t, service.SocialTaskChargeStatusCharged, items[0].ChargeStatus)
	require.NotNil(t, items[0].ResultMessage)
	require.Equal(t, "follow succeeded", *items[0].ResultMessage)
	require.Equal(t, int64(1), items[0].Quantity)
	require.InEpsilon(t, service.SocialTaskUnitPrice, items[0].Cost, 0.000001)
	require.NotNil(t, items[0].CompletedAt)

	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.TotalOperations)
	require.Equal(t, int64(1), stats.SuccessCount)
	require.Equal(t, int64(1), stats.FailedCount)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)
}

func TestUsageLogRepositorySortsSocialTaskLogsByDisplayFields(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-sort@example.com").SetPasswordHash("hash").SaveX(ctx)
	now := time.Now().UTC().Truncate(time.Second)

	xAccount := client.SocialAccount.Create().
		SetName("z-account").
		SetPlatform("x").
		SetPlatformKey("x_twitter").
		SetNameKey("z-account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	blueskyAccount := client.SocialAccount.Create().
		SetName("a-account").
		SetPlatform("bluesky").
		SetPlatformKey("bluesky").
		SetNameKey("a-account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	threadsAccount := client.SocialAccount.Create().
		SetName("m-account").
		SetPlatform("threads").
		SetPlatformKey("threads").
		SetNameKey("m-account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)

	xLog := client.SocialTaskLog.Create().
		SetSocialAccountID(xAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now.Add(2 * time.Second)).
		SetCreatedAt(now.Add(2 * time.Second)).
		SaveX(ctx)
	blueskyLog := client.SocialTaskLog.Create().
		SetSocialAccountID(blueskyAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	threadsLog := client.SocialTaskLog.Create().
		SetSocialAccountID(threadsAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionPost).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now.Add(time.Second)).
		SetCreatedAt(now.Add(time.Second)).
		SaveX(ctx)

	listIDs := func(sortBy string, sortOrder string) []int64 {
		items, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20, SortBy: sortBy, SortOrder: sortOrder}, usagestats.UsageLogFilters{UserID: user.ID})
		require.NoError(t, err)
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		return ids
	}

	require.Equal(t, []int64{blueskyLog.ID, threadsLog.ID, xLog.ID}, listIDs("platform", "asc"))
	require.Equal(t, []int64{xLog.ID, threadsLog.ID, blueskyLog.ID}, listIDs("platform", "desc"))
	require.Equal(t, []int64{blueskyLog.ID, threadsLog.ID, xLog.ID}, listIDs("account", "asc"))
	require.Equal(t, []int64{blueskyLog.ID, threadsLog.ID, xLog.ID}, listIDs("account_name", "asc"))
	require.Equal(t, []int64{blueskyLog.ID, xLog.ID, threadsLog.ID}, listIDs("result", "asc"))
	require.Equal(t, []int64{blueskyLog.ID, xLog.ID, threadsLog.ID}, listIDs("status", "asc"))
}

func TestUsageLogRepositoryOmitsIntermediateSocialTaskLogsFromUsage(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-final-only@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_final_only").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_final_only").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	now := time.Now().UTC().Truncate(time.Second)

	successLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	pendingLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusPending).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(now.Add(time.Second)).
		SaveX(ctx)

	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: user.ID})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, service.SocialTaskLogStatusSuccess, items[0].Status)

	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{UserID: user.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalOperations)
	require.Equal(t, int64(1), stats.SuccessCount)
	require.Zero(t, stats.FailedCount)

	detail, err := repo.GetByID(ctx, successLog.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, service.SocialTaskLogStatusSuccess, detail.Status)

	_, err = repo.GetByID(ctx, pendingLog.ID, user.ID)
	require.ErrorIs(t, err, service.ErrUsageLogNotFound)

	pendingItems, pendingResult, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID: user.ID,
		Status: service.SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), pendingResult.Total)
	require.Empty(t, pendingItems)

	pendingStats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{
		UserID: user.ID,
		Status: service.SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	require.Zero(t, pendingStats.TotalOperations)
	require.Zero(t, pendingStats.SuccessCount)
	require.Zero(t, pendingStats.FailedCount)
	require.Zero(t, pendingStats.TotalCharged)
}

func TestUsageLogRepositoryProjectsNormalizedPlatformKey(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-platform-key@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_platform_key").
		SetPlatform("Twitter / X").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_platform_key").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(time.Now().UTC()).
		SaveX(ctx)

	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: user.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, "x_twitter", items[0].Platform)

	stats, err := repo.GetUserDashboardStats(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, stats.ByPlatform, 1)
	require.Equal(t, "x_twitter", stats.ByPlatform[0].Platform)
}

func TestUsageLogRepositoryFiltersSocialTaskLogsByPlatformAlias(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-platform-alias@example.com").SetPasswordHash("hash").SaveX(ctx)
	xAccount := client.SocialAccount.Create().
		SetName("usage_social_platform_alias").
		SetPlatform("twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_platform_alias").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	instagramAccount := client.SocialAccount.Create().
		SetName("usage_social_platform_alias_other").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("usage_social_platform_alias_other").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(xAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(time.Now().UTC()).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(instagramAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(time.Now().UTC()).
		SaveX(ctx)

	filters := usagestats.UsageLogFilters{UserID: user.ID, Platform: "twitter"}
	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, "x_twitter", items[0].Platform)
	require.Equal(t, xAccount.ID, items[0].SocialAccountID)

	stats, err := repo.GetStatsWithFilters(ctx, filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)
}

func TestUsageLogRepositoryFiltersSocialTaskLogs(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-filters@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_filters").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_filters").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	windowStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	matchingTime := windowStart.Add(12 * time.Hour)
	windowEnd := windowStart.Add(24 * time.Hour)
	target := "@target"

	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(matchingTime).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetTarget(target).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(windowEnd.Add(time.Hour)).
		SaveX(ctx)

	filters := usagestats.UsageLogFilters{
		UserID:    user.ID,
		Operation: service.SocialTaskActionFollow,
		Status:    service.SocialTaskLogStatusSuccess,
		StartTime: &windowStart,
		EndTime:   &windowEnd,
	}
	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, filters)

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, service.SocialTaskActionFollow, items[0].Operation)
	require.Equal(t, service.SocialTaskLogStatusSuccess, items[0].Status)
	require.InEpsilon(t, service.SocialTaskUnitPrice, items[0].Cost, 0.000001)

	stats, err := repo.GetStatsWithFilters(ctx, filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalOperations)
	require.Equal(t, int64(1), stats.SuccessCount)
	require.Zero(t, stats.FailedCount)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)
}

func TestUsageLogRepositoryListAndStatsShareSocialTaskFilters(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-shared-filters@example.com").SetPasswordHash("hash").SaveX(ctx)
	mainAccount := client.SocialAccount.Create().
		SetName("Main Delivery").
		SetPlatform("Twitter / X").
		SetPlatformKey("x_twitter").
		SetNameKey("main_delivery").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	backupAccount := client.SocialAccount.Create().
		SetName("Backup Delivery").
		SetPlatform("Twitter / X").
		SetPlatformKey("x_twitter").
		SetNameKey("backup_delivery").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	otherPlatformAccount := client.SocialAccount.Create().
		SetName("Main Delivery Instagram").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("main_delivery_instagram").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	windowStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	matchingTime := windowStart.Add(3 * time.Hour)
	windowEnd := windowStart.Add(24 * time.Hour)

	client.SocialTaskLog.Create().
		SetSocialAccountID(mainAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(matchingTime).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(mainAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(matchingTime.Add(time.Minute)).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(mainAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusPending).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(matchingTime.Add(2 * time.Minute)).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(backupAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(otherPlatformAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(mainAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(matchingTime).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(mainAccount.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(windowEnd.Add(time.Hour)).
		SaveX(ctx)

	filters := usagestats.UsageLogFilters{
		UserID:      user.ID,
		Operation:   service.SocialTaskActionFollow,
		Platform:    "x_twitter",
		AccountName: "MAIN",
		StartTime:   &windowStart,
		EndTime:     &windowEnd,
	}
	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, filters)

	require.NoError(t, err)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, items, 2)
	for _, item := range items {
		require.Equal(t, mainAccount.ID, item.SocialAccountID)
		require.Equal(t, service.SocialTaskActionFollow, item.Operation)
		require.Contains(t, []string{service.SocialTaskLogStatusSuccess, service.SocialTaskLogStatusFailed}, item.Status)
	}

	stats, err := repo.GetStatsWithFilters(ctx, filters)
	require.NoError(t, err)
	require.Equal(t, result.Total, stats.TotalOperations)
	require.Equal(t, int64(1), stats.SuccessCount)
	require.Equal(t, int64(1), stats.FailedCount)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)
}

func TestUsageLogRepositoryCostsOnlyFinalSuccessfulCharges(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-final-charge@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_final_charge").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_final_charge").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-time.Hour)
	windowEnd := now.Add(time.Hour)

	success := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	failedWithStaleAmount := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(9.99).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetExecutedAt(now).
		SetCreatedAt(now.Add(time.Second)).
		SaveX(ctx)
	successWithoutCharge := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLoginCheck).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(7.77).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetExecutedAt(now).
		SetCreatedAt(now.Add(2 * time.Second)).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionPost).
		SetStatus(service.SocialTaskLogStatusPending).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(12.34).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(now.Add(3 * time.Second)).
		SaveX(ctx)

	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: user.ID})

	require.NoError(t, err)
	require.Equal(t, int64(3), result.Total)
	costByID := map[int64]float64{}
	for _, item := range items {
		costByID[item.ID] = item.Cost
	}
	require.InEpsilon(t, service.SocialTaskUnitPrice, costByID[success.ID], 0.000001)
	require.Zero(t, costByID[failedWithStaleAmount.ID])
	require.Zero(t, costByID[successWithoutCharge.ID])

	failedItem, err := repo.GetByID(ctx, failedWithStaleAmount.ID, user.ID)
	require.NoError(t, err)
	require.Zero(t, failedItem.Cost)

	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{UserID: user.ID})
	require.NoError(t, err)
	require.Equal(t, int64(3), stats.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)

	dashboard, err := repo.GetUserDashboardStats(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), dashboard.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, dashboard.TotalCharged, 0.000001)
	require.Len(t, dashboard.ByPlatform, 1)
	require.InEpsilon(t, service.SocialTaskUnitPrice, dashboard.ByPlatform[0].TotalCharged, 0.000001)

	trend, err := repo.GetUserUsageTrendByUserID(ctx, user.ID, windowStart, windowEnd, "day")
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, int64(3), trend[0].Operations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, trend[0].Charged, 0.000001)

	adminTrend, err := repo.GetUsageTrend(ctx, windowStart, windowEnd, "day")
	require.NoError(t, err)
	require.Len(t, adminTrend, 1)
	require.Equal(t, int64(3), adminTrend[0].Operations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, adminTrend[0].Charged, 0.000001)

	ranking, err := repo.GetUserSpendingRanking(ctx, windowStart, windowEnd, 10)
	require.NoError(t, err)
	require.Equal(t, int64(3), ranking.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, ranking.TotalCharged, 0.000001)
	require.NotNil(t, findRankingItem(ranking.Ranking, user.ID))
	require.InEpsilon(t, service.SocialTaskUnitPrice, findRankingItem(ranking.Ranking, user.ID).Charged, 0.000001)
}

func TestUsageLogRepositoryGetByIDProjectsStructuredTaskDetails(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-detail@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_detail").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_detail").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	target := "https://x.com/northwind/status/1"
	content := "hello world"
	chargeSource := service.SocialTaskChargeSourceSubscription
	proxySnapshot := `{"id":8,"name":"proxy-a","endpoint":"http://user:pass@proxy.local:8080","status":"online"}`
	billingRequestID := "sub:detail-1"
	idempotencyKey := "usage-repo-detail-1"
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionPost).
		SetTarget(target).
		SetContent(content).
		SetPayload(domain.SocialTaskPayload{
			Post: &domain.SocialPostPayload{
				Text:         "hello world",
				QuotePostURL: "https://x.com/northwind/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
				}},
			},
		}).
		SetTemplateSnapshot(domain.SocialTaskTemplateSnapshot{
			TemplateID:   "tmpl_1",
			TemplateName: "Rich post",
			TemplateType: service.SocialTaskActionPost,
			Params: domain.SocialTaskTemplateParams{
				Contents:     []string{"hello world"},
				QuotePostURL: "https://x.com/northwind/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
				}},
			},
		}).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetChargeSource(chargeSource).
		SetProxySnapshot(proxySnapshot).
		SetBillingRequestID(billingRequestID).
		SetIdempotencyKey(idempotencyKey).
		SetExecutedAt(time.Now().UTC()).
		SaveX(ctx)

	item, err := repo.GetByID(ctx, log.ID, user.ID)

	require.NoError(t, err)
	require.NotNil(t, item)
	require.NotNil(t, item.ChargeSource)
	require.Equal(t, chargeSource, *item.ChargeSource)
	require.NotNil(t, item.ProxySnapshot)
	require.Equal(t, proxySnapshot, *item.ProxySnapshot)
	require.NotNil(t, item.BillingRequestID)
	require.Equal(t, billingRequestID, *item.BillingRequestID)
	require.NotNil(t, item.IdempotencyKey)
	require.Equal(t, idempotencyKey, *item.IdempotencyKey)
	require.NotNil(t, item.CompletedAt)
	require.NotNil(t, item.Target)
	require.Equal(t, target, *item.Target)
	require.NotNil(t, item.Content)
	require.Equal(t, content, *item.Content)
	require.NotNil(t, item.Payload)
	require.NotNil(t, item.Payload.Post)
	require.Equal(t, "hello world", item.Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/2", item.Payload.Post.QuotePostURL)
	require.Len(t, item.Payload.Post.Media, 1)
	require.Equal(t, "post-image-1.png", item.Payload.Post.Media[0].FileName)
	require.NotNil(t, item.TemplateSnapshot)
	require.Equal(t, "tmpl_1", item.TemplateSnapshot.TemplateID)
	require.Equal(t, "Rich post", item.TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, item.TemplateSnapshot.TemplateType)
	require.Len(t, item.TemplateSnapshot.Params.Contents, 1)
	require.Equal(t, "hello world", item.TemplateSnapshot.Params.Contents[0])
	require.Len(t, item.TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "post-image-1.png", item.TemplateSnapshot.Params.Media[0].FileName)
}

func TestUsageLogRepositoryListProjectsStructuredTaskDetails(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-list-structured@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_list_structured").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_list_structured").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	target := "https://x.com/northwind/status/1"
	content := "hello world"
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionPost).
		SetTarget(target).
		SetContent(content).
		SetPayload(domain.SocialTaskPayload{
			Target: target,
			Post: &domain.SocialPostPayload{
				Text:         content,
				QuotePostURL: "https://x.com/northwind/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/post-image-1.png",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
					SHA256:      "secret-sha",
					ByteSize:    1234,
					Width:       400,
					Height:      400,
				}},
			},
			Avatar: &domain.SocialTaskMediaRef{
				Source:      "inline",
				URL:         "data:image/png;base64,QUJD",
				StorageKey:  "media/private/avatar.png",
				ContentType: "image/png",
				FileName:    "avatar.png",
				SHA256:      "avatar-sha",
				ByteSize:    2048,
				Width:       400,
				Height:      400,
			},
		}).
		SetTemplateSnapshot(domain.SocialTaskTemplateSnapshot{
			TemplateID:   "tmpl_1",
			TemplateName: "Rich post",
			TemplateType: service.SocialTaskActionPost,
			Params: domain.SocialTaskTemplateParams{
				Targets:      []string{target},
				Contents:     []string{content},
				QuotePostURL: "https://x.com/northwind/status/2",
				Media: []domain.SocialTaskMediaRef{{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/post-image-1.png",
					ContentType: "image/png",
					FileName:    "post-image-1.png",
					SHA256:      "secret-sha",
					ByteSize:    1234,
					Width:       400,
					Height:      400,
				}},
				Avatar: &domain.SocialTaskMediaRef{
					Source:      "inline",
					URL:         "data:image/png;base64,QUJD",
					StorageKey:  "media/private/avatar.png",
					ContentType: "image/png",
					FileName:    "avatar.png",
					SHA256:      "avatar-sha",
					ByteSize:    2048,
					Width:       400,
					Height:      400,
				},
			},
		}).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(time.Now().UTC()).
		SaveX(ctx)

	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: user.ID})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].Target)
	require.Equal(t, target, *items[0].Target)
	require.NotNil(t, items[0].Content)
	require.Equal(t, content, *items[0].Content)
	require.NotNil(t, items[0].Payload)
	require.Equal(t, target, items[0].Payload.Target)
	require.NotNil(t, items[0].Payload.Post)
	require.Equal(t, content, items[0].Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/2", items[0].Payload.Post.QuotePostURL)
	require.Len(t, items[0].Payload.Post.Media, 1)
	require.Equal(t, "post-image-1.png", items[0].Payload.Post.Media[0].FileName)
	require.Equal(t, "data:image/png;base64,QUJD", items[0].Payload.Post.Media[0].URL)
	require.Equal(t, "media/private/post-image-1.png", items[0].Payload.Post.Media[0].StorageKey)
	require.Equal(t, "secret-sha", items[0].Payload.Post.Media[0].SHA256)
	require.NotNil(t, items[0].Payload.Avatar)
	require.Equal(t, "avatar.png", items[0].Payload.Avatar.FileName)
	require.Equal(t, "media/private/avatar.png", items[0].Payload.Avatar.StorageKey)
	require.NotNil(t, items[0].TemplateSnapshot)
	require.Equal(t, "tmpl_1", items[0].TemplateSnapshot.TemplateID)
	require.Equal(t, "Rich post", items[0].TemplateSnapshot.TemplateName)
	require.Equal(t, service.SocialTaskActionPost, items[0].TemplateSnapshot.TemplateType)
	require.Equal(t, []string{target}, items[0].TemplateSnapshot.Params.Targets)
	require.Equal(t, []string{content}, items[0].TemplateSnapshot.Params.Contents)
	require.Equal(t, "https://x.com/northwind/status/2", items[0].TemplateSnapshot.Params.QuotePostURL)
	require.Len(t, items[0].TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "media/private/post-image-1.png", items[0].TemplateSnapshot.Params.Media[0].StorageKey)
	require.NotNil(t, items[0].TemplateSnapshot.Params.Avatar)
	require.Equal(t, "media/private/avatar.png", items[0].TemplateSnapshot.Params.Avatar.StorageKey)
}

func TestUsageLogRepositorySpendingRankingOmitsUsersWithoutCharges(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	chargedUser := client.User.Create().SetEmail("usage-ranking-charged@example.com").SetPasswordHash("hash").SaveX(ctx)
	failedOnlyUser := client.User.Create().SetEmail("usage-ranking-failed-only@example.com").SetPasswordHash("hash").SaveX(ctx)
	chargedAccount := client.SocialAccount.Create().
		SetName("usage_ranking_charged").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_ranking_charged").
		SetAssignedUserID(chargedUser.ID).
		SaveX(ctx)
	failedOnlyAccount := client.SocialAccount.Create().
		SetName("usage_ranking_failed_only").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_ranking_failed_only").
		SetAssignedUserID(failedOnlyUser.ID).
		SaveX(ctx)
	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-time.Hour)
	windowEnd := now.Add(time.Hour)

	client.SocialTaskLog.Create().
		SetSocialAccountID(chargedAccount.ID).
		SetUserID(chargedUser.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(failedOnlyAccount.ID).
		SetUserID(failedOnlyUser.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetExecutedAt(now).
		SetCreatedAt(now.Add(time.Second)).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(failedOnlyAccount.ID).
		SetUserID(failedOnlyUser.ID).
		SetAction(service.SocialTaskActionPost).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(9.99).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetExecutedAt(now).
		SetCreatedAt(now.Add(2 * time.Second)).
		SaveX(ctx)

	ranking, err := repo.GetUserSpendingRanking(ctx, windowStart, windowEnd, 10)

	require.NoError(t, err)
	require.Equal(t, int64(3), ranking.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, ranking.TotalCharged, 0.000001)
	require.NotNil(t, findRankingItem(ranking.Ranking, chargedUser.ID))
	require.Nil(t, findRankingItem(ranking.Ranking, failedOnlyUser.ID))
}

func TestUsageLogRepositoryUsesExecutionTimeForSocialTaskWindows(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	user := client.User.Create().SetEmail("usage-social-executed-window@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_executed_window").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_executed_window").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	windowStart := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)

	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(windowStart.Add(-2 * time.Hour)).
		SetExecutedAt(windowStart.Add(2 * time.Hour)).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0.35).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(windowStart.Add(3 * time.Hour)).
		SetExecutedAt(windowEnd.Add(2 * time.Hour)).
		SaveX(ctx)

	filters := usagestats.UsageLogFilters{
		UserID:    user.ID,
		StartTime: &windowStart,
		EndTime:   &windowEnd,
	}
	items, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, filters)

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, service.SocialTaskActionFollow, items[0].Operation)
	require.InEpsilon(t, service.SocialTaskUnitPrice, items[0].Cost, 0.000001)

	stats, err := repo.GetStatsWithFilters(ctx, filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalCharged, 0.000001)

	trend, err := repo.GetUserUsageTrendByUserID(ctx, user.ID, windowStart, windowEnd, "day")
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, "2026-06-02", trend[0].Date)
	require.Equal(t, int64(1), trend[0].Operations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, trend[0].Charged, 0.000001)
}

func TestUsageLogRepositoryUserDashboardUsesExecutionTimeForToday(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	todayStart := utcDayStart(now)
	user := client.User.Create().SetEmail("usage-social-dashboard-window@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("usage_social_dashboard_window").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("usage_social_dashboard_window").
		SetAssignedUserID(user.ID).
		SaveX(ctx)

	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(todayStart.Add(-2 * time.Hour)).
		SetExecutedAt(now).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0.35).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(now).
		SetExecutedAt(now.Add(24 * time.Hour)).
		SaveX(ctx)

	stats, err := repo.GetUserDashboardStats(ctx, user.ID)

	require.NoError(t, err)
	require.Equal(t, int64(2), stats.TotalOperations)
	require.InEpsilon(t, 0.45, stats.TotalCharged, 0.000001)
	require.Equal(t, int64(1), stats.TodayOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TodayCharged, 0.000001)
	require.Len(t, stats.ByPlatform, 1)
	require.Equal(t, int64(1), stats.ByPlatform[0].TodayOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.ByPlatform[0].TodayCharged, 0.000001)
}

func TestUsageLogRepositoryAggregatesSocialOpsDashboardStats(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	yesterday := now.Add(-24 * time.Hour)

	activeUser := client.User.Create().
		SetEmail("dashboard-active@example.com").
		SetPasswordHash("hash").
		SetCreatedAt(now).
		SaveX(ctx)
	inactiveUser := client.User.Create().
		SetEmail("dashboard-inactive@example.com").
		SetPasswordHash("hash").
		SetCreatedAt(yesterday).
		SaveX(ctx)

	availableAccount := client.SocialAccount.Create().
		SetName("dashboard_available").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("dashboard_available").
		SetAssignedUserID(activeUser.ID).
		SetAccountStatus("available").
		SaveX(ctx)
	limitedAccount := client.SocialAccount.Create().
		SetName("dashboard_limited").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("dashboard_limited").
		SetAssignedUserID(activeUser.ID).
		SetAccountStatus("limited").
		SaveX(ctx)
	invalidAccount := client.SocialAccount.Create().
		SetName("dashboard_invalid").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("dashboard_invalid").
		SetAssignedUserID(inactiveUser.ID).
		SetAccountStatus("invalid").
		SaveX(ctx)

	client.SocialTaskLog.Create().
		SetSocialAccountID(availableAccount.ID).
		SetUserID(activeUser.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(limitedAccount.ID).
		SetUserID(activeUser.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(now).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(invalidAccount.ID).
		SetUserID(inactiveUser.ID).
		SetAction(service.SocialTaskActionLoginCheck).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(yesterday).
		SaveX(ctx)

	stats, err := repo.GetDashboardStats(ctx)

	require.NoError(t, err)
	require.Equal(t, int64(2), stats.TotalUsers)
	require.Equal(t, int64(1), stats.TodayNewUsers)
	require.Equal(t, int64(1), stats.ActiveUsers)
	require.Equal(t, int64(3), stats.TotalAccounts)
	require.Equal(t, int64(1), stats.NormalAccounts)
	require.Equal(t, int64(1), stats.ErrorAccounts)
	require.Equal(t, int64(1), stats.RateLimitAccounts)
	require.Equal(t, int64(3), stats.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice*2, stats.TotalCharged, 0.000001)
	require.Equal(t, int64(2), stats.TodayOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TodayCharged, 0.000001)
	require.GreaterOrEqual(t, stats.RecentOperationsPerMinute, int64(0))
}

func TestUsageLogRepositoryAggregatesUserDashboardAndAdminBreakdowns(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	yesterday := now.Add(-24 * time.Hour)

	user := client.User.Create().SetEmail("dashboard-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	otherUser := client.User.Create().SetEmail("dashboard-other@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("dashboard_user_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("dashboard_user_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus("available").
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("dashboard_other_account").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("dashboard_other_account").
		SetAssignedUserID(otherUser.ID).
		SetAccountStatus("available").
		SaveX(ctx)

	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionFollow).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(now).
		SetCreatedAt(now).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusFailed).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(service.SocialTaskChargeStatusNotCharged).
		SetCreatedAt(yesterday).
		SaveX(ctx)
	client.SocialTaskLog.Create().
		SetSocialAccountID(otherAccount.ID).
		SetUserID(otherUser.ID).
		SetAction(service.SocialTaskActionPost).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(service.SocialTaskUnitPrice).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetCreatedAt(now).
		SaveX(ctx)

	userStats, err := repo.GetUserDashboardStats(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), userStats.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TotalCharged, 0.000001)
	require.Equal(t, int64(1), userStats.TodayOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TodayCharged, 0.000001)
	require.Len(t, userStats.ByPlatform, 1)
	require.Equal(t, "x_twitter", userStats.ByPlatform[0].Platform)
	require.Equal(t, int64(2), userStats.ByPlatform[0].TotalOperations)
	require.Equal(t, int64(1), userStats.ByPlatform[0].TodayOperations)

	trend, err := repo.GetUsageTrend(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), "day")
	require.NoError(t, err)
	require.NotEmpty(t, trend)
	require.Equal(t, int64(3), sumTrendOperations(trend))

	userTrend, err := repo.GetUserUsageTrend(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), "day", 10)
	require.NoError(t, err)
	require.NotEmpty(t, userTrend)
	require.Equal(t, int64(3), sumUserTrendOperations(userTrend))

	ranking, err := repo.GetUserSpendingRanking(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, ranking.Ranking, 2)
	require.Equal(t, int64(3), ranking.TotalOperations)
	require.InEpsilon(t, service.SocialTaskUnitPrice*2, ranking.TotalCharged, 0.000001)
	require.NotNil(t, findRankingItem(ranking.Ranking, user.ID))
	require.NotNil(t, findRankingItem(ranking.Ranking, otherUser.ID))
	require.InEpsilon(t, service.SocialTaskUnitPrice, findRankingItem(ranking.Ranking, otherUser.ID).Charged, 0.000001)
}

func TestUsageLogRepositoryUserUsageTrendLimitKeepsCompleteTopUserSeries(t *testing.T) {
	ctx := context.Background()
	repo, client := newUsageLogSocialOpsRepo(t)
	dayOne := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	dayTwo := dayOne.AddDate(0, 0, 1)

	topUser := client.User.Create().SetEmail("dashboard-trend-top@example.com").SetPasswordHash("hash").SaveX(ctx)
	otherUser := client.User.Create().SetEmail("dashboard-trend-other@example.com").SetPasswordHash("hash").SaveX(ctx)
	topAccount := client.SocialAccount.Create().
		SetName("dashboard_trend_top").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("dashboard_trend_top").
		SetAssignedUserID(topUser.ID).
		SetAccountStatus("available").
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("dashboard_trend_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("dashboard_trend_other").
		SetAssignedUserID(otherUser.ID).
		SetAccountStatus("available").
		SaveX(ctx)

	for _, executedAt := range []time.Time{dayOne, dayTwo} {
		client.SocialTaskLog.Create().
			SetSocialAccountID(topAccount.ID).
			SetUserID(topUser.ID).
			SetAction(service.SocialTaskActionFollow).
			SetStatus(service.SocialTaskLogStatusSuccess).
			SetPrice(service.SocialTaskUnitPrice).
			SetChargedAmount(0.40).
			SetChargeStatus(service.SocialTaskChargeStatusCharged).
			SetExecutedAt(executedAt).
			SetCreatedAt(executedAt).
			SaveX(ctx)
	}
	client.SocialTaskLog.Create().
		SetSocialAccountID(otherAccount.ID).
		SetUserID(otherUser.ID).
		SetAction(service.SocialTaskActionLike).
		SetStatus(service.SocialTaskLogStatusSuccess).
		SetPrice(service.SocialTaskUnitPrice).
		SetChargedAmount(0.70).
		SetChargeStatus(service.SocialTaskChargeStatusCharged).
		SetExecutedAt(dayOne).
		SetCreatedAt(dayOne).
		SaveX(ctx)

	trend, err := repo.GetUserUsageTrend(ctx, dayOne.Add(-time.Hour), dayTwo.Add(time.Hour), "day", 1)

	require.NoError(t, err)
	require.Len(t, trend, 2)
	require.Equal(t, "2026-06-01", trend[0].Date)
	require.Equal(t, topUser.ID, trend[0].UserID)
	require.Equal(t, "2026-06-02", trend[1].Date)
	require.Equal(t, topUser.ID, trend[1].UserID)
	require.InEpsilon(t, 0.80, sumUserTrendCharged(trend), 0.000001)
}

func sumTrendOperations(points []usagestats.TrendDataPoint) int64 {
	var total int64
	for _, point := range points {
		total += point.Operations
	}
	return total
}

func sumUserTrendOperations(points []usagestats.UserUsageTrendPoint) int64 {
	var total int64
	for _, point := range points {
		total += point.Operations
	}
	return total
}

func sumUserTrendCharged(points []usagestats.UserUsageTrendPoint) float64 {
	var total float64
	for _, point := range points {
		total += point.Charged
	}
	return total
}

func findRankingItem(items []usagestats.UserSpendingRankingItem, userID int64) *usagestats.UserSpendingRankingItem {
	for i := range items {
		if items[i].UserID == userID {
			return &items[i]
		}
	}
	return nil
}

func newUsageLogSocialOpsRepo(t *testing.T) (*usageLogRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	repo, ok := NewUsageLogRepository(client, db).(*usageLogRepository)
	require.True(t, ok)
	return repo, client
}
