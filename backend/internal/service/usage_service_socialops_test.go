//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageService_GetByID_DoesNotFabricateMissingUsage(t *testing.T) {
	svc := NewUsageService(nil, nil)

	usage, err := svc.GetByID(context.Background(), 42, 1001)

	require.Nil(t, usage)
	require.True(t, infraerrors.IsNotFound(err), "expected not-found for missing usage, got %v", err)
	require.Equal(t, "USAGE_NOT_FOUND", infraerrors.Reason(err))
}

func TestUsageService_GetUserDashboardStatsDelegatesToRepository(t *testing.T) {
	repo := &socialOpsUsageDashboardRepoStub{
		userStats: &usagestats.UserDashboardStats{
			TotalRequests:   7,
			TotalTokens:     7,
			TotalActualCost: 0.7,
			TodayRequests:   3,
			TodayTokens:     3,
			TodayActualCost: 0.3,
		},
	}
	svc := NewUsageService(repo, nil)

	stats, err := svc.GetUserDashboardStats(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, int64(42), repo.userStatsUserID)
	require.Equal(t, int64(7), stats.TotalRequests)
	require.Equal(t, int64(3), stats.TodayRequests)
	require.InEpsilon(t, 0.7, stats.TotalActualCost, 0.000001)
}

func TestUsageService_GetUserUsageTrendByUserIDDelegatesToRepository(t *testing.T) {
	repo := &socialOpsUsageDashboardRepoStub{
		userTrend: []usagestats.TrendDataPoint{{Date: "2026-06-01", Requests: 4, TotalTokens: 4, ActualCost: 0.4}},
	}
	svc := NewUsageService(repo, nil)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	trend, err := svc.GetUserUsageTrendByUserID(context.Background(), 42, start, end, "day")

	require.NoError(t, err)
	require.Equal(t, int64(42), repo.userTrendUserID)
	require.Equal(t, "day", repo.userTrendGranularity)
	require.Equal(t, repo.userTrend, trend)
}

func TestDashboardServiceDelegatesSocialOpsAggregatesToRepository(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	repo := &socialOpsUsageDashboardRepoStub{
		dashboardStats: &usagestats.DashboardStats{TotalUsers: 5, ActiveUsers: 2, TodayRequests: 9},
		adminTrend:     []usagestats.TrendDataPoint{{Date: "2026-06-01", Requests: 9, TotalTokens: 9}},
		userUsageTrend: []usagestats.UserUsageTrendPoint{{Date: "2026-06-01", UserID: 42, Email: "u@example.com", Requests: 9, Tokens: 9}},
		ranking: &usagestats.UserSpendingRankingResponse{
			Ranking:         []usagestats.UserSpendingRankingItem{{UserID: 42, Email: "u@example.com", ActualCost: 0.9, Requests: 9, Tokens: 9}},
			TotalActualCost: 0.9,
			TotalRequests:   9,
			TotalTokens:     9,
		},
	}
	svc := ProvideDashboardService(repo, nil, nil)

	stats, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(5), stats.TotalUsers)
	require.Equal(t, int64(9), stats.TodayRequests)

	trend, err := svc.GetUsageTrend(context.Background(), start, end, "day", nil, nil)
	require.NoError(t, err)
	require.Equal(t, repo.adminTrend, trend)
	require.Equal(t, "day", repo.adminTrendGranularity)

	userTrend, err := svc.GetUserUsageTrend(context.Background(), start, end, "day", 20)
	require.NoError(t, err)
	require.Equal(t, repo.userUsageTrend, userTrend)
	require.Equal(t, 20, repo.userUsageTrendLimit)

	ranking, err := svc.GetUserSpendingRanking(context.Background(), start, end, 20)
	require.NoError(t, err)
	require.Equal(t, repo.ranking, ranking)
	require.Equal(t, 20, repo.rankingLimit)
}

type socialOpsUsageDashboardRepoStub struct {
	userStats       *usagestats.UserDashboardStats
	userStatsUserID int64

	userTrend            []usagestats.TrendDataPoint
	userTrendUserID      int64
	userTrendGranularity string

	dashboardStats *usagestats.DashboardStats

	adminTrend            []usagestats.TrendDataPoint
	adminTrendGranularity string

	userUsageTrend      []usagestats.UserUsageTrendPoint
	userUsageTrendLimit int

	ranking      *usagestats.UserSpendingRankingResponse
	rankingLimit int
}

func (r *socialOpsUsageDashboardRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetStatsWithFilters(context.Context, usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	return nil, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetByID(context.Context, int64, int64) (*UsageLog, error) {
	return nil, ErrUsageLogNotFound
}

func (r *socialOpsUsageDashboardRepoStub) GetUserDashboardStats(_ context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	r.userStatsUserID = userID
	return r.userStats, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetUserUsageTrendByUserID(_ context.Context, userID int64, _ time.Time, _ time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	r.userTrendUserID = userID
	r.userTrendGranularity = granularity
	return r.userTrend, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.dashboardStats, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetUsageTrend(_ context.Context, _ time.Time, _ time.Time, granularity string, _ *int16, _ *bool) ([]usagestats.TrendDataPoint, error) {
	r.adminTrendGranularity = granularity
	return r.adminTrend, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetUserUsageTrend(_ context.Context, _ time.Time, _ time.Time, _ string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	r.userUsageTrendLimit = limit
	return r.userUsageTrend, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetUserSpendingRanking(_ context.Context, _ time.Time, _ time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	r.rankingLimit = limit
	return r.ranking, nil
}
