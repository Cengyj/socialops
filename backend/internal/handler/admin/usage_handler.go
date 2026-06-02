package admin

import (
	"strconv"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type UsageHandler struct {
	usageService        *service.UsageService
	userService         *service.UserService
	apiKeyService       *service.APIKeyService
	usageCleanupService *service.UsageCleanupService
}

func NewUsageHandler(usageService *service.UsageService, userService *service.UserService, apiKeyService *service.APIKeyService, usageCleanupService *service.UsageCleanupService) *UsageHandler {
	return &UsageHandler{
		usageService:        usageService,
		userService:         userService,
		apiKeyService:       apiKeyService,
		usageCleanupService: usageCleanupService,
	}
}

func (h *UsageHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	filters := parseAdminUsageFilters(c)
	items, result, err := h.usageService.List(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if result == nil {
		response.Paginated(c, items, int64(len(items)), page, pageSize)
		return
	}
	response.Paginated(c, items, result.Total, page, pageSize)
}

func (h *UsageHandler) Stats(c *gin.Context) {
	stats, err := h.usageService.Stats(c.Request.Context(), parseAdminUsageFilters(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *UsageHandler) SearchUsers(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *UsageHandler) SearchAPIKeys(c *gin.Context) {
	response.Success(c, []gin.H{})
}

func (h *UsageHandler) ListCleanupTasks(c *gin.Context) {
	response.Success(c, gin.H{"items": []gin.H{}, "total": 0})
}

func (h *UsageHandler) CreateCleanupTask(c *gin.Context) {
	response.Success(c, gin.H{"created": false, "message": "usage cleanup is not configured yet"})
}

func (h *UsageHandler) CancelCleanupTask(c *gin.Context) {
	if _, err := strconv.ParseInt(c.Param("id"), 10, 64); err != nil {
		response.BadRequest(c, "Invalid cleanup task ID")
		return
	}
	response.Success(c, gin.H{"canceled": false})
}

func parseAdminUsageFilters(c *gin.Context) usagestats.UsageLogFilters {
	filters := usagestats.UsageLogFilters{}
	if raw := c.Query("user_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filters.UserID = id
		}
	}
	if raw := c.Query("api_key_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filters.APIKeyID = id
		}
	}
	return filters
}
