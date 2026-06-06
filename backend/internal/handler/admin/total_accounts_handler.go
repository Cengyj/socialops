package admin

import (
	"strconv"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// TotalAccountsHandler handles total account pool ownership operations.
type TotalAccountsHandler struct {
	svc *service.SocialAccountService
}

func NewTotalAccountsHandler(svc *service.SocialAccountService) *TotalAccountsHandler {
	return &TotalAccountsHandler{svc: svc}
}

// List returns the admin-governed total account pool.
// GET /api/v1/admin/total-accounts
func (h *TotalAccountsHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := service.SocialAccountListFilters{
		Platform:      c.Query("platform"),
		AccountStatus: c.Query("account_status"),
		TaskStatus:    c.Query("task_status"),
	}
	if c.Query("unassigned") == "true" {
		filters.UnassignedOnly = true
	}

	accounts, result, err := h.svc.ListTotalPool(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, accounts, result.Total, page, pageSize)
}

// Assign assigns a total-pool social account to a user.
// POST /api/v1/admin/total-accounts/:id/assign
func (h *TotalAccountsHandler) Assign(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	account, err := h.svc.Assign(c.Request.Context(), id, req.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Reclaim removes the user assignment from a total-pool social account.
// POST /api/v1/admin/total-accounts/:id/reclaim
func (h *TotalAccountsHandler) Reclaim(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	account, err := h.svc.Reclaim(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// BatchAssign assigns multiple total-pool social accounts to a user.
// POST /api/v1/admin/total-accounts/batch-assign
func (h *TotalAccountsHandler) BatchAssign(c *gin.Context) {
	var req struct {
		IDs    []int64 `json:"ids" binding:"required"`
		UserID int64   `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.BatchAssign(c.Request.Context(), req.IDs, req.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// BatchReclaim removes assignments from multiple total-pool social accounts.
// POST /api/v1/admin/total-accounts/batch-reclaim
func (h *TotalAccountsHandler) BatchReclaim(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.BatchReclaim(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// BatchDelete soft-deletes multiple total-pool social accounts.
// POST /api/v1/admin/total-accounts/batch-delete
func (h *TotalAccountsHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.BatchDelete(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
