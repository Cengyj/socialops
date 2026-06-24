package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/socialops/internal/handler/socialaccountcsv"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// TotalAccountsHandler handles total account pool ownership operations.
type TotalAccountsHandler struct {
	svc *service.SocialAccountService
}

type totalAccountAssignRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

type totalAccountBatchAssignRequest struct {
	IDs    []int64 `json:"ids" binding:"required"`
	UserID int64   `json:"user_id" binding:"required"`
}

type totalAccountBatchIDsRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

func NewTotalAccountsHandler(svc *service.SocialAccountService) *TotalAccountsHandler {
	return &TotalAccountsHandler{svc: svc}
}

// List returns the admin-governed total account pool.
// GET /api/v1/admin/total-accounts
func (h *TotalAccountsHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := totalAccountFiltersFromQuery(c)

	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	accounts, result, err := svc.ListTotalPool(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, accounts, result.Total, page, pageSize)
}

// Import imports social accounts into the admin-governed total account pool.
// POST /api/v1/admin/total-accounts/import
func (h *TotalAccountsHandler) Import(c *gin.Context) {
	if h == nil {
		response.ErrorFrom(c, socialAccountServiceUnavailableError())
		return
	}
	importPoolAccountsFromRequest(c, h.svc)
}

// Update updates mutable delivery/status fields on a total-pool account.
// PUT /api/v1/admin/total-accounts/:id
func (h *TotalAccountsHandler) Update(c *gin.Context) {
	id, ok := totalAccountPathID(c)
	if !ok {
		return
	}
	var input service.UpdateSocialAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, totalAccountInputRequiredError())
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	account, err := svc.UpdateTotalPool(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Export exports the admin-governed total account pool as CSV.
// GET /api/v1/admin/total-accounts/export
func (h *TotalAccountsHandler) Export(c *gin.Context) {
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	accounts, _, err := svc.ListTotalPool(c.Request.Context(), pagination.PaginationParams{Page: 1, PageSize: 10000}, totalAccountFiltersFromQuery(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=total_accounts.csv")

	if err := socialaccountcsv.WriteDeliveryExport(c.Writer, accounts); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// Assign assigns a total-pool social account to a user.
// POST /api/v1/admin/total-accounts/:id/assign
func (h *TotalAccountsHandler) Assign(c *gin.Context) {
	id, ok := totalAccountPathID(c)
	if !ok {
		return
	}
	var req totalAccountAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, totalAccountInputRequiredError())
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	account, err := svc.AssignTotalPool(c.Request.Context(), id, req.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// Reclaim removes the user assignment from a total-pool social account.
// POST /api/v1/admin/total-accounts/:id/reclaim
func (h *TotalAccountsHandler) Reclaim(c *gin.Context) {
	id, ok := totalAccountPathID(c)
	if !ok {
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	account, err := svc.ReclaimTotalPool(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

// BatchAssign assigns multiple total-pool social accounts to a user.
// POST /api/v1/admin/total-accounts/batch-assign
func (h *TotalAccountsHandler) BatchAssign(c *gin.Context) {
	var req totalAccountBatchAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, totalAccountInputRequiredError())
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	result, err := svc.BatchAssignTotalPool(c.Request.Context(), req.IDs, req.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func totalAccountFiltersFromQuery(c *gin.Context) service.SocialAccountListFilters {
	filters := service.SocialAccountListFilters{
		Platform:      c.Query("platform"),
		AccountStatus: c.Query("account_status"),
		TaskStatus:    c.Query("task_status"),
		Search:        c.Query("search"),
		AccountIDs:    totalAccountInt64ListQuery(c, "account_ids"),
	}
	if c.Query("unassigned") == "true" {
		filters.UnassignedOnly = true
	}
	if c.Query("assigned") == "true" {
		filters.AssignedOnly = true
	}
	return filters
}

func totalAccountInt64ListQuery(c *gin.Context, key string) []int64 {
	rawValues := c.QueryArray(key)
	if len(rawValues) == 0 {
		raw := strings.TrimSpace(c.Query(key))
		if raw != "" {
			rawValues = []string{raw}
		}
	}
	if len(rawValues) == 0 {
		return nil
	}
	values := make([]int64, 0, len(rawValues))
	for _, raw := range rawValues {
		for _, part := range strings.Split(raw, ",") {
			parsed, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err == nil && parsed > 0 {
				values = append(values, parsed)
			}
		}
	}
	return values
}

// BatchReclaim removes assignments from multiple total-pool social accounts.
// POST /api/v1/admin/total-accounts/batch-reclaim
func (h *TotalAccountsHandler) BatchReclaim(c *gin.Context) {
	var req totalAccountBatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, totalAccountInputRequiredError())
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	result, err := svc.BatchReclaimTotalPool(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// BatchDelete permanently removes multiple total-pool social accounts.
// POST /api/v1/admin/total-accounts/batch-delete
func (h *TotalAccountsHandler) BatchDelete(c *gin.Context) {
	var req totalAccountBatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, totalAccountInputRequiredError())
		return
	}
	svc, ok := h.totalAccountsService(c)
	if !ok {
		return
	}
	result, err := svc.BatchDeleteTotalPool(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func totalAccountPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func totalAccountInputRequiredError() error {
	return infraerrors.BadRequest("SOCIAL_ACCOUNT_INPUT_REQUIRED", "social account input is required")
}

func (h *TotalAccountsHandler) totalAccountsService(c *gin.Context) (*service.SocialAccountService, bool) {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, socialAccountServiceUnavailableError())
		return nil, false
	}
	return h.svc, true
}
