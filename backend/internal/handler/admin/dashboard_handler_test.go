package admin

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestAdminDashboardStatsResponseExcludesRemovedGatewayFields(t *testing.T) {
	stats := &usagestats.DashboardStats{
		TotalUsers:                12,
		TodayNewUsers:             2,
		ActiveUsers:               7,
		HourlyActiveUsers:         3,
		StatsUpdatedAt:            "2026-06-09T00:00:00Z",
		StatsStale:                true,
		TotalAccounts:             20,
		NormalAccounts:            16,
		ErrorAccounts:             1,
		RateLimitAccounts:         2,
		OverloadAccounts:          1,
		TotalOperations:           123,
		TotalCharged:              8.8,
		TodayOperations:           9,
		TodayCharged:              1.1,
		AverageDurationMs:         456.7,
		RecentOperationsPerMinute: 6,
	}

	payload := mustMarshalAdminDashboardResponse(t, adminDashboardStatsResponseFromUsageStats(stats))

	require.Contains(t, payload, `"total_accounts":20`)
	require.Contains(t, payload, `"today_charged":1.1`)
	require.Contains(t, payload, `"total_operations":123`)
	require.Contains(t, payload, `"recent_operations_per_minute":6`)
	require.NotContains(t, payload, "api_key")
	require.NotContains(t, payload, "token")
	require.NotContains(t, payload, `"total_requests"`)
	require.NotContains(t, payload, `"today_requests"`)
	require.NotContains(t, payload, `"total_actual_cost"`)
	require.NotContains(t, payload, `"today_actual_cost"`)
	require.NotContains(t, payload, `"rpm"`)
	require.NotContains(t, payload, "total_cost")
	require.NotContains(t, payload, "account_cost")
	require.NotContains(t, payload, `"tpm"`)
}

func TestAdminDashboardTrendResponsesExcludeRemovedGatewayFields(t *testing.T) {
	trend := adminDashboardTrendResponseFromUsageTrend([]usagestats.TrendDataPoint{{
		Date:       "2026-06-09",
		Operations: 15,
		Charged:    2.3,
	}})
	userTrend := adminUserUsageTrendResponseFromUsageTrend([]usagestats.UserUsageTrendPoint{{
		Date:       "2026-06-09",
		UserID:     42,
		Email:      "operator@example.com",
		Username:   "operator",
		Operations: 7,
		Charged:    3.2,
	}})
	ranking := adminUserSpendingRankingResponseFromUsageStats(&usagestats.UserSpendingRankingResponse{
		Ranking: []usagestats.UserSpendingRankingItem{{
			UserID:     42,
			Email:      "operator@example.com",
			Operations: 7,
			Charged:    3.2,
		}},
		TotalCharged:    3.2,
		TotalOperations: 7,
	})

	payload := mustMarshalAdminDashboardResponse(t, map[string]any{
		"trend":     trend,
		"userTrend": userTrend,
		"ranking":   ranking,
	})

	require.Contains(t, payload, `"charged":2.3`)
	require.Contains(t, payload, `"total_charged":3.2`)
	require.Contains(t, payload, `"operations":15`)
	require.Contains(t, payload, `"total_operations":7`)
	require.NotContains(t, payload, "token")
	require.NotContains(t, payload, `"requests"`)
	require.NotContains(t, payload, `"actual_cost"`)
	require.NotContains(t, payload, `"total_requests"`)
	require.NotContains(t, payload, `"total_actual_cost"`)
	require.NotContains(t, payload, `"cost":`)
	require.NotContains(t, payload, "total_tokens")
}

func TestAdminDashboardEmptyResponsesUseStableArrays(t *testing.T) {
	require.Equal(t, []adminDashboardTrendPointResponse{}, adminDashboardTrendResponseFromUsageTrend(nil))
	require.Equal(t, []adminUserUsageTrendPointResponse{}, adminUserUsageTrendResponseFromUsageTrend(nil))
	require.Equal(t, []adminUserSpendingRankingItemResponse{}, adminUserSpendingRankingResponseFromUsageStats(nil).Ranking)
}

func mustMarshalAdminDashboardResponse(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}
