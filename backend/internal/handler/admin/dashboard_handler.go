package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
	settingService   *service.SettingService
}

func NewDashboardHandler(dashboardService *service.DashboardService, settingService *service.SettingService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService, settingService: settingService}
}

func (h *DashboardHandler) GetSnapshotV2(c *gin.Context) {
	response.Success(c, gin.H{"stats": gin.H{}, "updated_at": time.Now().UTC()})
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.dashboardService.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *DashboardHandler) GetRealtimeMetrics(c *gin.Context) {
	response.Success(c, gin.H{"online_users": 0, "active_tasks": 0})
}

func (h *DashboardHandler) GetUsageTrend(c *gin.Context) {
	start, end := dashboardDateRange()
	trend, err := h.dashboardService.GetUsageTrend(c.Request.Context(), start, end, c.DefaultQuery("granularity", "day"), nil, nil)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, trend)
}

func (h *DashboardHandler) GetGroupStats(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *DashboardHandler) GetAPIKeyUsageTrend(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *DashboardHandler) GetUserUsageTrend(c *gin.Context) {
	start, end := dashboardDateRange()
	limit := parseDashboardLimit(c.DefaultQuery("limit", "20"), 50)
	trend, err := h.dashboardService.GetUserUsageTrend(c.Request.Context(), start, end, c.DefaultQuery("granularity", "day"), limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, trend)
}

func (h *DashboardHandler) GetUserSpendingRanking(c *gin.Context) {
	start, end := dashboardDateRange()
	limit := parseDashboardLimit(c.DefaultQuery("limit", "20"), 50)
	ranking, err := h.dashboardService.GetUserSpendingRanking(c.Request.Context(), start, end, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ranking)
}

func (h *DashboardHandler) GetBatchUsersUsage(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *DashboardHandler) GetBatchAPIKeysUsage(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *DashboardHandler) GetUserBreakdown(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *DashboardHandler) BackfillAggregation(c *gin.Context) {
	response.Success(c, gin.H{"queued": false, "message": "dashboard aggregation is not configured yet"})
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
