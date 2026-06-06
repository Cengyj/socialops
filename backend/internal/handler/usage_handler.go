package handler

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type UsageHandler struct {
	usageService  *service.UsageService
	apiKeyService *service.APIKeyService
}

func NewUsageHandler(usageService *service.UsageService, apiKeyService *service.APIKeyService) *UsageHandler {
	return &UsageHandler{usageService: usageService, apiKeyService: apiKeyService}
}

func (h *UsageHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	filters, err := usageLogFiltersFromQuery(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items, result, err := h.usageService.List(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items = sanitizeUsageLogResults(items)
	if result == nil {
		response.Paginated(c, items, int64(len(items)), page, pageSize)
		return
	}
	response.Paginated(c, items, result.Total, page, pageSize)
}

func (h *UsageHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid usage ID")
		return
	}
	item, err := h.usageService.GetByID(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if item != nil {
		item.ResultMessage = shortUserTaskResult(item.ResultMessage)
	}
	response.Success(c, item)
}

func (h *UsageHandler) Stats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	filters, err := usageLogFiltersFromQuery(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	stats, err := h.usageService.Stats(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *UsageHandler) DashboardStats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	stats, err := h.usageService.GetUserDashboardStats(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *UsageHandler) DashboardTrend(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	start, end := parseUsageWindow()
	trend, err := h.usageService.GetUserUsageTrendByUserID(c.Request.Context(), subject.UserID, start, end, c.DefaultQuery("granularity", "day"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, trend)
}

func (h *UsageHandler) DashboardAPIKeysUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		APIKeyIDs []int64 `json:"api_key_ids"`
	}
	_ = c.ShouldBindJSON(&req)
	start, end := parseUsageWindow()
	items, err := h.usageService.GetUserAPIKeysUsage(c.Request.Context(), subject.UserID, req.APIKeyIDs, start, end)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *UsageHandler) GetMyAPIKeyDailyUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if h.apiKeyService != nil {
		key, err := h.apiKeyService.GetByID(c.Request.Context(), keyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if key.UserID != subject.UserID {
			response.Forbidden(c, "Not authorized to access this key")
			return
		}
	}
	days := 30
	if raw := c.Query("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 90 {
			response.BadRequest(c, "Invalid days")
			return
		}
		days = parsed
	}
	response.Success(c, gin.H{"items": []usagestats.APIKeyDailyUsagePoint{}, "days": days})
}

func parseUsageWindow() (time.Time, time.Time) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)
	return start, end
}

func sanitizeUsageLogResults(items []service.UsageLog) []service.UsageLog {
	if len(items) == 0 {
		return items
	}
	sanitized := make([]service.UsageLog, len(items))
	copy(sanitized, items)
	for i := range sanitized {
		sanitized[i].ResultMessage = shortUserTaskResult(sanitized[i].ResultMessage)
	}
	return sanitized
}

func usageLogFiltersFromQuery(c *gin.Context, userID int64) (usagestats.UsageLogFilters, error) {
	filters := usagestats.UsageLogFilters{
		UserID: userID,
		Model:  normalizeUsageQueryValue(firstUsageQuery(c, "operation", "model")),
		Status: normalizeUsageQueryValue(c.Query("status")),
	}
	if start, err := parseUsageQueryTime(c.Query("start_date"), false); err != nil {
		return filters, err
	} else if start != nil {
		filters.StartTime = start
	}
	if end, err := parseUsageQueryTime(c.Query("end_date"), true); err != nil {
		return filters, err
	} else if end != nil {
		filters.EndTime = end
	}
	return filters, nil
}

func firstUsageQuery(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(c.Query(name)); value != "" {
			return value
		}
	}
	return ""
}

func normalizeUsageQueryValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseUsageQueryTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		utc := parsed.UTC()
		return &utc, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, infraerrors.BadRequest("USAGE_DATE_INVALID", "invalid usage date filter")
	}
	parsed = parsed.UTC()
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
