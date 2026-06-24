package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

type adminDashboardStatsResponse struct {
	TotalUsers                int64   `json:"total_users"`
	TodayNewUsers             int64   `json:"today_new_users"`
	ActiveUsers               int64   `json:"active_users"`
	HourlyActiveUsers         int64   `json:"hourly_active_users"`
	TotalAccounts             int64   `json:"total_accounts"`
	NormalAccounts            int64   `json:"normal_accounts"`
	ErrorAccounts             int64   `json:"error_accounts"`
	RateLimitAccounts         int64   `json:"ratelimit_accounts"`
	OverloadAccounts          int64   `json:"overload_accounts"`
	TotalOperations           int64   `json:"total_operations"`
	TodayOperations           int64   `json:"today_operations"`
	TotalCharged              float64 `json:"total_charged"`
	TodayCharged              float64 `json:"today_charged"`
	AverageDurationMs         float64 `json:"average_duration_ms"`
	RecentOperationsPerMinute int64   `json:"recent_operations_per_minute"`
	StatsUpdatedAt            string  `json:"stats_updated_at"`
	StatsStale                bool    `json:"stats_stale"`
}

type adminDashboardTrendPointResponse struct {
	Date       string  `json:"date"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"`
}

type adminUserUsageTrendPointResponse struct {
	Date       string  `json:"date"`
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Username   string  `json:"username"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"`
}

type adminUserSpendingRankingItemResponse struct {
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"`
}

type adminUserSpendingRankingResponse struct {
	Ranking         []adminUserSpendingRankingItemResponse `json:"ranking"`
	TotalCharged    float64                                `json:"total_charged"`
	TotalOperations int64                                  `json:"total_operations"`
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.dashboardService.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, adminDashboardStatsResponseFromUsageStats(stats))
}

func (h *DashboardHandler) GetUsageTrend(c *gin.Context) {
	start, end := dashboardDateRange()
	trend, err := h.dashboardService.GetUsageTrend(c.Request.Context(), start, end, c.DefaultQuery("granularity", "day"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, adminDashboardTrendResponseFromUsageTrend(trend))
}

func (h *DashboardHandler) GetUserUsageTrend(c *gin.Context) {
	start, end := dashboardDateRange()
	limit := parseDashboardLimit(c.DefaultQuery("limit", "20"), 50)
	trend, err := h.dashboardService.GetUserUsageTrend(c.Request.Context(), start, end, c.DefaultQuery("granularity", "day"), limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, adminUserUsageTrendResponseFromUsageTrend(trend))
}

func (h *DashboardHandler) GetUserSpendingRanking(c *gin.Context) {
	start, end := dashboardDateRange()
	limit := parseDashboardLimit(c.DefaultQuery("limit", "20"), 50)
	ranking, err := h.dashboardService.GetUserSpendingRanking(c.Request.Context(), start, end, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, adminUserSpendingRankingResponseFromUsageStats(ranking))
}

func dashboardDateRange() (time.Time, time.Time) {
	end := time.Now().UTC()
	return end.AddDate(0, 0, -30), end
}

func parseDashboardLimit(raw string, max int) int {
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > max {
		return max
	}
	return limit
}

func adminDashboardStatsResponseFromUsageStats(stats *usagestats.DashboardStats) adminDashboardStatsResponse {
	if stats == nil {
		return adminDashboardStatsResponse{}
	}
	return adminDashboardStatsResponse{
		TotalUsers:                stats.TotalUsers,
		TodayNewUsers:             stats.TodayNewUsers,
		ActiveUsers:               stats.ActiveUsers,
		HourlyActiveUsers:         stats.HourlyActiveUsers,
		TotalAccounts:             stats.TotalAccounts,
		NormalAccounts:            stats.NormalAccounts,
		ErrorAccounts:             stats.ErrorAccounts,
		RateLimitAccounts:         stats.RateLimitAccounts,
		OverloadAccounts:          stats.OverloadAccounts,
		TotalOperations:           stats.TotalOperations,
		TodayOperations:           stats.TodayOperations,
		TotalCharged:              stats.TotalCharged,
		TodayCharged:              stats.TodayCharged,
		AverageDurationMs:         stats.AverageDurationMs,
		RecentOperationsPerMinute: stats.RecentOperationsPerMinute,
		StatsUpdatedAt:            stats.StatsUpdatedAt,
		StatsStale:                stats.StatsStale,
	}
}

func adminDashboardTrendResponseFromUsageTrend(trend []usagestats.TrendDataPoint) []adminDashboardTrendPointResponse {
	if len(trend) == 0 {
		return []adminDashboardTrendPointResponse{}
	}
	out := make([]adminDashboardTrendPointResponse, 0, len(trend))
	for _, point := range trend {
		out = append(out, adminDashboardTrendPointResponse{
			Date:       point.Date,
			Operations: point.Operations,
			Charged:    point.Charged,
		})
	}
	return out
}

func adminUserUsageTrendResponseFromUsageTrend(trend []usagestats.UserUsageTrendPoint) []adminUserUsageTrendPointResponse {
	if len(trend) == 0 {
		return []adminUserUsageTrendPointResponse{}
	}
	out := make([]adminUserUsageTrendPointResponse, 0, len(trend))
	for _, point := range trend {
		out = append(out, adminUserUsageTrendPointResponse{
			Date:       point.Date,
			UserID:     point.UserID,
			Email:      point.Email,
			Username:   point.Username,
			Operations: point.Operations,
			Charged:    point.Charged,
		})
	}
	return out
}

func adminUserSpendingRankingResponseFromUsageStats(ranking *usagestats.UserSpendingRankingResponse) adminUserSpendingRankingResponse {
	out := adminUserSpendingRankingResponse{Ranking: []adminUserSpendingRankingItemResponse{}}
	if ranking == nil {
		return out
	}
	out.TotalCharged = ranking.TotalCharged
	out.TotalOperations = ranking.TotalOperations
	if len(ranking.Ranking) == 0 {
		return out
	}
	out.Ranking = make([]adminUserSpendingRankingItemResponse, 0, len(ranking.Ranking))
	for _, item := range ranking.Ranking {
		out.Ranking = append(out.Ranking, adminUserSpendingRankingItemResponse{
			UserID:     item.UserID,
			Email:      item.Email,
			Operations: item.Operations,
			Charged:    item.Charged,
		})
	}
	return out
}
