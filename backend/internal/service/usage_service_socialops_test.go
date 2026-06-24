//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/domain"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageService_GetByID_DoesNotFabricateMissingUsage(t *testing.T) {
	svc := NewUsageService(nil)

	usage, err := svc.GetByID(context.Background(), 42, 1001)

	require.Nil(t, usage)
	require.True(t, infraerrors.IsNotFound(err), "expected not-found for missing usage, got %v", err)
	require.Equal(t, "USAGE_NOT_FOUND", infraerrors.Reason(err))
}

func TestUsageServiceDoesNotFabricateUsageWhenRepositoryMissing(t *testing.T) {
	svc := NewUsageService(nil)

	items, page, err := svc.List(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{UserID: 42})
	require.Nil(t, items)
	require.Nil(t, page)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing usage repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	stats, err := svc.Stats(context.Background(), usagestats.UsageLogFilters{UserID: 42})
	require.Nil(t, stats)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing usage stats repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	dashboardStats, err := svc.GetUserDashboardStats(context.Background(), 42)
	require.Nil(t, dashboardStats)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing user dashboard repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	trend, err := svc.GetUserUsageTrendByUserID(context.Background(), 42, time.Now().Add(-time.Hour), time.Now(), "day")
	require.Nil(t, trend)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing user usage trend repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))
}

func TestUsageServiceNormalizesUsageLogFiltersBeforeRepository(t *testing.T) {
	repo := &socialOpsUsageDashboardRepoStub{
		stats: &usagestats.UsageStats{TotalOperations: 1},
	}
	svc := NewUsageService(repo)

	items, page, err := svc.List(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    42,
		Operation: " UPDATE_PROFILE ",
		Status:    " SUCCESS ",
	})

	require.NoError(t, err)
	require.NotNil(t, items)
	require.NotNil(t, page)
	require.Equal(t, 1, repo.listCalls)
	require.Equal(t, SocialTaskActionUpdateProfile, repo.listFilters.Operation)
	require.Equal(t, SocialTaskLogStatusSuccess, repo.listFilters.Status)

	stats, err := svc.Stats(context.Background(), usagestats.UsageLogFilters{
		UserID:    42,
		Operation: " LIKE ",
		Status:    " FAILED ",
	})

	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Equal(t, 1, repo.statsCalls)
	require.Equal(t, SocialTaskActionLike, repo.statsFilters.Operation)
	require.Equal(t, SocialTaskLogStatusFailed, repo.statsFilters.Status)
}

func TestUsageServiceRejectsUnsupportedUsageLogFiltersBeforeRepository(t *testing.T) {
	tests := []struct {
		name    string
		filters usagestats.UsageLogFilters
		reason  string
	}{
		{
			name:    "old tweet action",
			filters: usagestats.UsageLogFilters{UserID: 42, Operation: "tweet"},
			reason:  "SOCIAL_TASK_UNSUPPORTED_ACTION",
		},
		{
			name:    "unsupported direct message action",
			filters: usagestats.UsageLogFilters{UserID: 42, Operation: "message"},
			reason:  "SOCIAL_TASK_UNSUPPORTED_ACTION",
		},
		{
			name:    "non-final status",
			filters: usagestats.UsageLogFilters{UserID: 42, Status: SocialTaskLogStatusRunning},
			reason:  "USAGE_STATUS_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &socialOpsUsageDashboardRepoStub{}
			svc := NewUsageService(repo)

			items, page, err := svc.List(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, tt.filters)

			require.Nil(t, items)
			require.Nil(t, page)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
			require.Zero(t, repo.listCalls)

			stats, err := svc.Stats(context.Background(), tt.filters)

			require.Nil(t, stats)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
			require.Zero(t, repo.statsCalls)
		})
	}
}

func TestDashboardServiceDoesNotFabricateUsageWhenRepositoryMissing(t *testing.T) {
	svc := ProvideDashboardService(nil)
	start := time.Now().Add(-time.Hour)
	end := time.Now()

	stats, err := svc.GetStats(context.Background())
	require.Nil(t, stats)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing dashboard stats repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	trend, err := svc.GetUsageTrend(context.Background(), start, end, "day")
	require.Nil(t, trend)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing dashboard trend repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	userTrend, err := svc.GetUserUsageTrend(context.Background(), start, end, "day", 20)
	require.Nil(t, userTrend)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing user trend repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))

	ranking, err := svc.GetUserSpendingRanking(context.Background(), start, end, 20)
	require.Nil(t, ranking)
	require.True(t, infraerrors.IsServiceUnavailable(err), "expected service unavailable for missing ranking repository, got %v", err)
	require.Equal(t, "USAGE_SERVICE_UNAVAILABLE", infraerrors.Reason(err))
}

func TestUsageService_GetUserDashboardStatsDelegatesToRepository(t *testing.T) {
	repo := &socialOpsUsageDashboardRepoStub{
		userStats: &usagestats.UserDashboardStats{
			TotalOperations: 7,
			TotalCharged:    0.7,
			TodayOperations: 3,
			TodayCharged:    0.3,
		},
	}
	svc := NewUsageService(repo)

	stats, err := svc.GetUserDashboardStats(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, int64(42), repo.userStatsUserID)
	require.Equal(t, int64(7), stats.TotalOperations)
	require.Equal(t, int64(3), stats.TodayOperations)
	require.InEpsilon(t, 0.7, stats.TotalCharged, 0.000001)
}

func TestUsageService_GetUserUsageTrendByUserIDDelegatesToRepository(t *testing.T) {
	repo := &socialOpsUsageDashboardRepoStub{
		userTrend: []usagestats.TrendDataPoint{{Date: "2026-06-01", Operations: 4, Charged: 0.4}},
	}
	svc := NewUsageService(repo)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	trend, err := svc.GetUserUsageTrendByUserID(context.Background(), 42, start, end, "day")

	require.NoError(t, err)
	require.Equal(t, int64(42), repo.userTrendUserID)
	require.Equal(t, "day", repo.userTrendGranularity)
	require.Equal(t, repo.userTrend, trend)
}

func TestUsageServicePreviewTaskMediaResolvesStoredPayloadAndTemplateAssets(t *testing.T) {
	repo := &usagePreviewRepoStub{
		item: &UsageLog{
			ID:     501,
			UserID: 42,
			Payload: &domain.SocialTaskPayload{
				Post: &domain.SocialPostPayload{
					Media: []domain.SocialTaskMediaRef{{
						Source:      "library",
						StorageKey:  "social-task/42/post.png",
						ContentType: "image/png",
						FileName:    "post.png",
					}},
				},
			},
			TemplateSnapshot: &domain.SocialTaskTemplateSnapshot{
				Params: domain.SocialTaskTemplateParams{
					Avatar: &domain.SocialTaskMediaRef{
						Source:      "library",
						StorageKey:  "social-task/42/avatar.png",
						ContentType: "image/png",
						FileName:    "avatar.png",
					},
				},
			},
		},
	}
	resolver := &usageMediaResolverStub{
		resolved: &ResolvedSocialTaskMedia{
			ContentType: "image/png",
			FileName:    "preview.png",
			Body:        []byte("preview"),
		},
	}
	svc := NewUsageService(repo).WithMediaResolver(resolver)

	postResolved, err := svc.PreviewTaskMedia(context.Background(), 501, 42, UsageTaskMediaLocator{
		Scope:   "payload",
		Section: "post",
		Index:   0,
	})

	require.NoError(t, err)
	require.NotNil(t, postResolved)
	require.Equal(t, int64(42), resolver.userID)
	require.NotNil(t, resolver.ref)
	require.Equal(t, "social-task/42/post.png", resolver.ref.StorageKey)

	templateResolved, err := svc.PreviewTaskMedia(context.Background(), 501, 42, UsageTaskMediaLocator{
		Scope:   "template",
		Section: "avatar",
		Index:   -1,
	})

	require.NoError(t, err)
	require.NotNil(t, templateResolved)
	require.NotNil(t, resolver.ref)
	require.Equal(t, "social-task/42/avatar.png", resolver.ref.StorageKey)
}

func TestUsageServicePreviewTaskMediaFailsClosedForUnsupportedOrMissingRefs(t *testing.T) {
	svc := NewUsageService(&usagePreviewRepoStub{
		item: &UsageLog{
			ID:     501,
			UserID: 42,
			Payload: &domain.SocialTaskPayload{
				Post: &domain.SocialPostPayload{
					Media: []domain.SocialTaskMediaRef{{
						Source:      "library",
						StorageKey:  "media/post.png",
						ContentType: "image/png",
						FileName:    "post.png",
					}},
				},
			},
		},
	}).WithMediaResolver(&usageMediaResolverStub{})

	_, err := svc.PreviewTaskMedia(context.Background(), 501, 42, UsageTaskMediaLocator{
		Scope:   "payload",
		Section: "post",
		Index:   0,
	})
	require.True(t, infraerrors.IsBadRequest(err), "expected bad request for unsupported media ref, got %v", err)
	require.Equal(t, "USAGE_TASK_MEDIA_SOURCE_UNSUPPORTED", infraerrors.Reason(err))

	_, err = svc.PreviewTaskMedia(context.Background(), 501, 42, UsageTaskMediaLocator{
		Scope:   "payload",
		Section: "post",
		Index:   1,
	})
	require.True(t, infraerrors.IsNotFound(err), "expected not found for missing media ref, got %v", err)
	require.Equal(t, "USAGE_TASK_MEDIA_NOT_FOUND", infraerrors.Reason(err))
}

func TestDashboardServiceDelegatesSocialOpsAggregatesToRepository(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	repo := &socialOpsUsageDashboardRepoStub{
		dashboardStats: &usagestats.DashboardStats{TotalUsers: 5, ActiveUsers: 2, TodayOperations: 9},
		adminTrend:     []usagestats.TrendDataPoint{{Date: "2026-06-01", Operations: 9}},
		userUsageTrend: []usagestats.UserUsageTrendPoint{{Date: "2026-06-01", UserID: 42, Email: "u@example.com", Operations: 9}},
		ranking: &usagestats.UserSpendingRankingResponse{
			Ranking:         []usagestats.UserSpendingRankingItem{{UserID: 42, Email: "u@example.com", Charged: 0.9, Operations: 9}},
			TotalCharged:    0.9,
			TotalOperations: 9,
		},
	}
	svc := ProvideDashboardService(repo)

	stats, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(5), stats.TotalUsers)
	require.Equal(t, int64(9), stats.TodayOperations)

	trend, err := svc.GetUsageTrend(context.Background(), start, end, "day")
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

	listCalls   int
	listFilters usagestats.UsageLogFilters

	stats        *usagestats.UsageStats
	statsCalls   int
	statsFilters usagestats.UsageLogFilters

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

type usagePreviewRepoStub struct {
	item *UsageLog
}

func (r *usagePreviewRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *usagePreviewRepoStub) GetStatsWithFilters(context.Context, usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	return nil, nil
}

func (r *usagePreviewRepoStub) GetByID(context.Context, int64, int64) (*UsageLog, error) {
	if r.item == nil {
		return nil, ErrUsageLogNotFound
	}
	return r.item, nil
}

type usageMediaResolverStub struct {
	userID   int64
	ref      *domain.SocialTaskMediaRef
	resolved *ResolvedSocialTaskMedia
	err      error
}

func (r *usageMediaResolverStub) Resolve(_ context.Context, userID int64, ref *domain.SocialTaskMediaRef) (*ResolvedSocialTaskMedia, error) {
	r.userID = userID
	if ref != nil {
		cloned := *ref
		r.ref = &cloned
	} else {
		r.ref = nil
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.resolved, nil
}

func (r *socialOpsUsageDashboardRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	r.listCalls++
	r.listFilters = filters
	return []UsageLog{}, &pagination.PaginationResult{Page: 1, PageSize: 20}, nil
}

func (r *socialOpsUsageDashboardRepoStub) GetStatsWithFilters(_ context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	r.statsCalls++
	r.statsFilters = filters
	if r.stats != nil {
		return r.stats, nil
	}
	return &usagestats.UsageStats{}, nil
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

func (r *socialOpsUsageDashboardRepoStub) GetUsageTrend(_ context.Context, _ time.Time, _ time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
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
