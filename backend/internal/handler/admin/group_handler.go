package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// GroupHandler manages SocialOps business/subscription groups. It keeps the
// generic SaaS group model available for subscriptions, redeem codes, and
// account/task access without reintroducing AI gateway routing semantics.
type GroupHandler struct {
	adminService         service.AdminService
	dashboardService     *service.DashboardService
	groupCapacityService *service.GroupCapacityService
	groupRepo            service.GroupRepository
}

func NewGroupHandler(adminService service.AdminService, dashboardService *service.DashboardService, groupCapacityService *service.GroupCapacityService, groupRepo service.GroupRepository) *GroupHandler {
	return &GroupHandler{
		adminService:         adminService,
		dashboardService:     dashboardService,
		groupCapacityService: groupCapacityService,
		groupRepo:            groupRepo,
	}
}

type groupMutationRequest struct {
	Name                string   `json:"name"`
	Description         *string  `json:"description"`
	Platform            string   `json:"platform"`
	RateMultiplier      *float64 `json:"rate_multiplier"`
	IsExclusive         *bool    `json:"is_exclusive"`
	Status              string   `json:"status"`
	SubscriptionType    string   `json:"subscription_type"`
	DailyLimitUSD       *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD      *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD     *float64 `json:"monthly_limit_usd"`
	DefaultValidityDays *int     `json:"default_validity_days"`
	SortOrder           *int     `json:"sort_order"`
	RPMLimit            *int     `json:"rpm_limit"`
}

func (h *GroupHandler) List(c *gin.Context) {
	if h.groupRepo == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable"))
		return
	}
	page, pageSize := response.ParsePagination(c)
	var isExclusive *bool
	if raw := strings.TrimSpace(c.Query("is_exclusive")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "Invalid is_exclusive value")
			return
		}
		isExclusive = &parsed
	}

	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "sort_order"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}
	items, result, err := h.groupRepo.ListWithFilters(
		c.Request.Context(),
		params,
		c.Query("platform"),
		c.Query("status"),
		c.Query("subscription_type"),
		c.Query("search"),
		isExclusive,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.AdminGroup, 0, len(items))
	for i := range items {
		out = append(out, *dto.GroupFromServiceAdmin(&items[i]))
	}
	total := int64(len(out))
	if result != nil {
		total = result.Total
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *GroupHandler) GetByID(c *gin.Context) {
	group, err := h.getGroupFromParam(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.GroupFromServiceAdmin(group))
}

func (h *GroupHandler) Create(c *gin.Context) {
	if h.groupRepo == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable"))
		return
	}
	var req groupMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	group, err := groupFromCreateRequest(req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.groupRepo.Create(c.Request.Context(), group); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.GroupFromServiceAdmin(group))
}

func (h *GroupHandler) Update(c *gin.Context) {
	if h.groupRepo == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable"))
		return
	}
	group, err := h.getGroupFromParam(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var req groupMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := applyGroupUpdateRequest(group, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.groupRepo.Update(c.Request.Context(), group); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.groupRepo.GetByID(c.Request.Context(), group.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.GroupFromServiceAdmin(updated))
}

func (h *GroupHandler) Delete(c *gin.Context) {
	if h.groupRepo == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable"))
		return
	}
	id, err := parseGroupID(c)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	if _, err := h.groupRepo.DeleteCascade(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Group deleted successfully"})
}

func (h *GroupHandler) getGroupFromParam(c *gin.Context) (*service.Group, error) {
	if h.groupRepo == nil {
		return nil, infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable")
	}
	id, err := parseGroupID(c)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "invalid group id")
	}
	return h.groupRepo.GetByID(c.Request.Context(), id)
}

func parseGroupID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func groupFromCreateRequest(req groupMutationRequest) (*service.Group, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "name is required")
	}
	group := &service.Group{
		Name:                name,
		Platform:            normalizeGroupPlatform(req.Platform),
		RateMultiplier:      1,
		Status:              service.StatusActive,
		SubscriptionType:    service.SubscriptionTypeSubscription,
		DefaultValidityDays: 30,
	}
	if req.Description != nil {
		group.Description = strings.TrimSpace(*req.Description)
	}
	if req.RateMultiplier != nil {
		if *req.RateMultiplier <= 0 {
			return nil, infraerrors.BadRequest("INVALID_RATE_MULTIPLIER", "rate_multiplier must be greater than zero")
		}
		group.RateMultiplier = *req.RateMultiplier
	}
	if req.IsExclusive != nil {
		group.IsExclusive = *req.IsExclusive
	}
	if strings.TrimSpace(req.Status) != "" {
		group.Status = strings.TrimSpace(req.Status)
	}
	if strings.TrimSpace(req.SubscriptionType) != "" {
		group.SubscriptionType = strings.TrimSpace(req.SubscriptionType)
	}
	group.DailyLimitUSD = normalizeLimitPtr(req.DailyLimitUSD)
	group.WeeklyLimitUSD = normalizeLimitPtr(req.WeeklyLimitUSD)
	group.MonthlyLimitUSD = normalizeLimitPtr(req.MonthlyLimitUSD)
	if req.DefaultValidityDays != nil {
		if *req.DefaultValidityDays <= 0 {
			return nil, infraerrors.BadRequest("INVALID_VALIDITY_DAYS", "default_validity_days must be greater than zero")
		}
		group.DefaultValidityDays = *req.DefaultValidityDays
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}
	if req.RPMLimit != nil {
		if *req.RPMLimit < 0 {
			return nil, infraerrors.BadRequest("INVALID_RPM_LIMIT", "rpm_limit cannot be negative")
		}
		group.RPMLimit = *req.RPMLimit
	}
	return group, nil
}

func applyGroupUpdateRequest(group *service.Group, req groupMutationRequest) error {
	if strings.TrimSpace(req.Name) != "" {
		group.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != nil {
		group.Description = strings.TrimSpace(*req.Description)
	}
	if strings.TrimSpace(req.Platform) != "" {
		group.Platform = normalizeGroupPlatform(req.Platform)
	}
	if req.RateMultiplier != nil {
		if *req.RateMultiplier <= 0 {
			return infraerrors.BadRequest("INVALID_RATE_MULTIPLIER", "rate_multiplier must be greater than zero")
		}
		group.RateMultiplier = *req.RateMultiplier
	}
	if req.IsExclusive != nil {
		group.IsExclusive = *req.IsExclusive
	}
	if strings.TrimSpace(req.Status) != "" {
		group.Status = strings.TrimSpace(req.Status)
	}
	if strings.TrimSpace(req.SubscriptionType) != "" {
		group.SubscriptionType = strings.TrimSpace(req.SubscriptionType)
	}
	if req.DailyLimitUSD != nil {
		group.DailyLimitUSD = normalizeLimitPtr(req.DailyLimitUSD)
	}
	if req.WeeklyLimitUSD != nil {
		group.WeeklyLimitUSD = normalizeLimitPtr(req.WeeklyLimitUSD)
	}
	if req.MonthlyLimitUSD != nil {
		group.MonthlyLimitUSD = normalizeLimitPtr(req.MonthlyLimitUSD)
	}
	if req.DefaultValidityDays != nil {
		if *req.DefaultValidityDays <= 0 {
			return infraerrors.BadRequest("INVALID_VALIDITY_DAYS", "default_validity_days must be greater than zero")
		}
		group.DefaultValidityDays = *req.DefaultValidityDays
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}
	if req.RPMLimit != nil {
		if *req.RPMLimit < 0 {
			return infraerrors.BadRequest("INVALID_RPM_LIMIT", "rpm_limit cannot be negative")
		}
		group.RPMLimit = *req.RPMLimit
	}
	return nil
}

func normalizeGroupPlatform(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "social"
	}
	return value
}

func normalizeLimitPtr(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}
