package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
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
	require.Nil(t, items[0].APIKeyID)
	require.Nil(t, items[0].GroupID)
	require.Equal(t, service.SocialTaskActionFollow, items[0].Operation)
	require.Equal(t, service.SocialTaskLogStatusSuccess, items[0].Status)
	require.Equal(t, int64(1), items[0].Quantity)
	require.InEpsilon(t, service.SocialTaskUnitPrice, items[0].Cost, 0.000001)
	require.NotNil(t, items[0].CompletedAt)

	stats, err := repo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.TotalRequests)
	require.Equal(t, int64(2), stats.TotalTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TotalActualCost, 0.000001)
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
	require.Equal(t, int64(3), stats.TotalRequests)
	require.Equal(t, int64(3), stats.TotalTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice*2, stats.TotalCost, 0.000001)
	require.InEpsilon(t, service.SocialTaskUnitPrice*2, stats.TotalActualCost, 0.000001)
	require.Equal(t, int64(2), stats.TodayRequests)
	require.Equal(t, int64(2), stats.TodayTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TodayCost, 0.000001)
	require.InEpsilon(t, service.SocialTaskUnitPrice, stats.TodayActualCost, 0.000001)
	require.GreaterOrEqual(t, stats.Rpm, int64(0))
	require.GreaterOrEqual(t, stats.Tpm, int64(0))
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
	require.Equal(t, int64(2), userStats.TotalRequests)
	require.Equal(t, int64(2), userStats.TotalTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TotalCost, 0.000001)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TotalActualCost, 0.000001)
	require.Equal(t, int64(1), userStats.TodayRequests)
	require.Equal(t, int64(1), userStats.TodayTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TodayCost, 0.000001)
	require.InEpsilon(t, service.SocialTaskUnitPrice, userStats.TodayActualCost, 0.000001)
	require.Len(t, userStats.ByPlatform, 1)
	require.Equal(t, "x_twitter", userStats.ByPlatform[0].Platform)
	require.Equal(t, int64(2), userStats.ByPlatform[0].TotalRequests)
	require.Equal(t, int64(1), userStats.ByPlatform[0].TodayRequests)

	trend, err := repo.GetUsageTrend(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), "day", nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, trend)
	require.Equal(t, int64(3), sumTrendRequests(trend))

	userTrend, err := repo.GetUserUsageTrend(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), "day", 10)
	require.NoError(t, err)
	require.NotEmpty(t, userTrend)
	require.Equal(t, int64(3), sumUserTrendRequests(userTrend))

	ranking, err := repo.GetUserSpendingRanking(ctx, yesterday.Add(-time.Hour), now.Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, ranking.Ranking, 2)
	require.Equal(t, int64(3), ranking.TotalRequests)
	require.Equal(t, int64(3), ranking.TotalTokens)
	require.InEpsilon(t, service.SocialTaskUnitPrice*2, ranking.TotalActualCost, 0.000001)
	require.NotNil(t, findRankingItem(ranking.Ranking, user.ID))
	require.NotNil(t, findRankingItem(ranking.Ranking, otherUser.ID))
	require.InEpsilon(t, service.SocialTaskUnitPrice, findRankingItem(ranking.Ranking, otherUser.ID).ActualCost, 0.000001)
}

func sumTrendRequests(points []usagestats.TrendDataPoint) int64 {
	var total int64
	for _, point := range points {
		total += point.Requests
	}
	return total
}

func sumUserTrendRequests(points []usagestats.UserUsageTrendPoint) int64 {
	var total int64
	for _, point := range points {
		total += point.Requests
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
