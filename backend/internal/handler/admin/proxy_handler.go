package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// ProxyHandler manages SocialOps execution proxies for admins.
type ProxyHandler struct {
	ipSvc     *service.SocialIPService
	ipChecker *service.SocialIPChecker
}

// NewProxyHandler creates a SocialOps admin proxy handler.
func NewProxyHandler(ipSvc *service.SocialIPService, ipChecker *service.SocialIPChecker) *ProxyHandler {
	return &ProxyHandler{ipSvc: ipSvc, ipChecker: ipChecker}
}

// List returns all user-owned execution proxies.
// GET /api/v1/admin/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	filters := service.SocialIPListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		IPType: strings.TrimSpace(c.Query("ip_type")),
		Search: strings.TrimSpace(c.Query("search")),
	}
	if userIDRaw := strings.TrimSpace(c.Query("user_id")); userIDRaw != "" {
		userID, err := strconv.ParseInt(userIDRaw, 10, 64)
		if err != nil || userID <= 0 {
			response.BadRequest(c, "invalid user_id")
			return
		}
		filters.UserID = &userID
	}

	ips, result, err := h.ipSvc.ListForAdmin(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, ips, result.Total, page, pageSize)
}

// Create creates a user-owned execution proxy.
// POST /api/v1/admin/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	var req struct {
		UserID   int64   `json:"user_id" binding:"required"`
		Name     string  `json:"name" binding:"required"`
		IPType   string  `json:"ip_type"`
		Endpoint *string `json:"endpoint"`
		Remark   *string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.UserID <= 0 {
		response.BadRequest(c, "user_id is required")
		return
	}
	ip, err := h.ipSvc.Create(c.Request.Context(), &service.CreateSocialIPInput{
		UserID:   req.UserID,
		Name:     req.Name,
		IPType:   req.IPType,
		Endpoint: req.Endpoint,
		Remark:   req.Remark,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ip)
}

// Update updates a user-owned execution proxy without changing ownership.
// PUT /api/v1/admin/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var input service.UpdateSocialIPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ip, err := h.ipSvc.Update(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ip)
}

// Delete removes an execution proxy.
// DELETE /api/v1/admin/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.ipSvc.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// Test runs a free connectivity test for an execution proxy.
// POST /api/v1/admin/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	result, err := h.ipChecker.TestIP(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
