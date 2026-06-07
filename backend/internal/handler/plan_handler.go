package handler

import (
	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

// PlanHandler handles plan-related endpoints for users.
type PlanHandler struct {
	planService   *service.PlanService
	configService *service.PaymentConfigService
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(planService *service.PlanService, configService *service.PaymentConfigService) *PlanHandler {
	return &PlanHandler{planService: planService, configService: configService}
}

// ListPlansForSale returns plans available for purchase.
// GET /api/v1/plans
func (h *PlanHandler) ListPlansForSale(c *gin.Context) {
	plans, err := h.configService.ListPlansForSale(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groupInfo := h.configService.GetGroupInfoMap(c.Request.Context(), plans)
	response.Success(c, dto.AvailableSubscriptionPlansFromEnt(plans, groupInfo))
}

// GetMyPlan returns the current user's active plan.
// GET /api/v1/my-plan
func (h *PlanHandler) GetMyPlan(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	plan, err := h.planService.GetUserActivePlan(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if plan == nil {
		response.Success(c, gin.H{"active": false})
		return
	}
	response.Success(c, gin.H{"active": true, "plan": plan})
}
