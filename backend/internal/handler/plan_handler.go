package handler

import (
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

// PlanHandler handles plan-related endpoints for users.
type PlanHandler struct {
	planService *service.PlanService
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

// ListPlansForSale returns plans available for purchase.
// GET /api/v1/plans
func (h *PlanHandler) ListPlansForSale(c *gin.Context) {
	plans, err := h.planService.ListPlansForSale(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plans)
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
