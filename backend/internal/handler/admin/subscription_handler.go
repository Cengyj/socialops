package admin

import (
	"strconv"

	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

type CreateSubscriptionRequest struct {
	UserID       int64  `json:"user_id" binding:"required"`
	PlanID       int64  `json:"plan_id"`
	ValidityDays int    `json:"validity_days"`
	Notes        string `json:"notes"`
}

type BulkCreateSubscriptionRequest struct {
	UserIDs      []int64 `json:"user_ids" binding:"required,min=1"`
	PlanID       int64   `json:"plan_id"`
	ValidityDays int     `json:"validity_days"`
	Notes        string  `json:"notes"`
}

type AdjustSubscriptionRequest struct {
	Days int `json:"days" binding:"required"`
}

type ResetSubscriptionQuotaRequest struct {
	Daily   bool `json:"daily"`
	Weekly  bool `json:"weekly"`
	Monthly bool `json:"monthly"`
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var userID *int64
	if raw := c.Query("user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid user ID")
			return
		}
		userID = &id
	}
	var groupID *int64
	if raw := c.Query("group_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid group ID")
			return
		}
		groupID = &id
	}
	var planID *int64
	if raw := c.Query("plan_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid plan ID")
			return
		}
		planID = &id
	}
	items, result, err := h.subscriptionService.ListWithPlan(c.Request.Context(), page, pageSize, userID, groupID, planID, c.Query("status"), c.Query("platform"), c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminUserSubscription, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserSubscriptionFromServiceAdmin(&items[i]))
	}
	if result == nil {
		response.Paginated(c, out, int64(len(out)), page, pageSize)
		return
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}
	sub, err := h.subscriptionService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserSubscriptionFromServiceAdmin(sub))
}

func (h *SubscriptionHandler) GetProgress(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}
	progress, err := h.subscriptionService.GetSubscriptionProgress(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, progress)
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.PlanID <= 0 {
		response.BadRequest(c, "plan_id is required")
		return
	}
	sub, err := h.subscriptionService.AssignSubscription(c.Request.Context(), &service.AssignSubscriptionInput{
		UserID:       req.UserID,
		PlanID:       &req.PlanID,
		ValidityDays: req.ValidityDays,
		AssignedBy:   adminIDFromContext(c),
		Notes:        req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.UserSubscriptionFromServiceAdmin(sub))
}

func (h *SubscriptionHandler) BulkCreate(c *gin.Context) {
	var req BulkCreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.PlanID <= 0 {
		response.BadRequest(c, "plan_id is required")
		return
	}
	result, err := h.subscriptionService.BulkAssignSubscription(c.Request.Context(), &service.BulkAssignSubscriptionInput{
		UserIDs:      req.UserIDs,
		PlanID:       &req.PlanID,
		ValidityDays: req.ValidityDays,
		AssignedBy:   adminIDFromContext(c),
		Notes:        req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminUserSubscription, 0, len(result.Subscriptions))
	for i := range result.Subscriptions {
		out = append(out, *dto.UserSubscriptionFromServiceAdmin(&result.Subscriptions[i]))
	}
	response.Created(c, gin.H{
		"success_count": result.SuccessCount,
		"created_count": result.CreatedCount,
		"reused_count":  result.ReusedCount,
		"failed_count":  result.FailedCount,
		"subscriptions": out,
		"errors":        result.Errors,
	})
}

func (h *SubscriptionHandler) Extend(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}
	var req AdjustSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	sub, err := h.subscriptionService.ExtendSubscription(c.Request.Context(), id, req.Days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserSubscriptionFromServiceAdmin(sub))
}

func (h *SubscriptionHandler) ResetQuota(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}
	var req ResetSubscriptionQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	sub, err := h.subscriptionService.AdminResetQuota(c.Request.Context(), id, req.Daily, req.Weekly, req.Monthly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserSubscriptionFromServiceAdmin(sub))
}

func (h *SubscriptionHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}
	if err := h.subscriptionService.RevokeSubscription(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Subscription revoked successfully"})
}

func (h *SubscriptionHandler) ListByUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.subscriptionService.List(c.Request.Context(), page, pageSize, &userID, nil, c.Query("status"), c.Query("platform"), c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminUserSubscription, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserSubscriptionFromServiceAdmin(&items[i]))
	}
	if result == nil {
		response.Paginated(c, out, int64(len(out)), page, pageSize)
		return
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

func (h *SubscriptionHandler) ListByGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.subscriptionService.List(c.Request.Context(), page, pageSize, nil, &groupID, c.Query("status"), c.Query("platform"), c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminUserSubscription, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserSubscriptionFromServiceAdmin(&items[i]))
	}
	if result == nil {
		response.Paginated(c, out, int64(len(out)), page, pageSize)
		return
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

func adminIDFromContext(c *gin.Context) int64 {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		return 0
	}
	return subject.UserID
}
