package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ProxyHandler manages current-user SocialOps execution proxies.
type ProxyHandler struct {
	ipSvc     *service.SocialIPService
	ipChecker *service.SocialIPChecker
}

// NewProxyHandler creates a user-scoped proxy handler.
func NewProxyHandler(ipSvc *service.SocialIPService, ipChecker *service.SocialIPChecker) *ProxyHandler {
	return &ProxyHandler{ipSvc: ipSvc, ipChecker: ipChecker}
}

// List returns proxies owned by the current user.
// GET /api/v1/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := service.SocialIPListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		IPType: strings.TrimSpace(c.Query("ip_type")),
		Search: strings.TrimSpace(c.Query("search")),
	}
	ips, result, err := h.ipSvc.ListByUser(c.Request.Context(), subject.UserID, params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, ips, result.Total, page, pageSize)
}

// ListUsable returns assignable online proxies owned by the current user.
// GET /api/v1/proxies/usable
func (h *ProxyHandler) ListUsable(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	ips, err := h.ipSvc.ListUsableByUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ips)
}

// Create creates a proxy owned by the current user. The request body must not
// contain user_id; ownership always comes from the JWT subject.
// POST /api/v1/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	rawBody, err := c.GetRawData()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &fields); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if _, exists := fields["user_id"]; exists {
		response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_USER_ID_NOT_ACCEPTED", "user_id is not accepted"))
		return
	}
	var req struct {
		Name     string  `json:"name" binding:"required"`
		IPType   string  `json:"ip_type"`
		Endpoint *string `json:"endpoint"`
		Remark   *string `json:"remark"`
	}
	if err := binding.JSON.BindBody(rawBody, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ip, err := h.ipSvc.Create(c.Request.Context(), &service.CreateSocialIPInput{
		UserID:   subject.UserID,
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

// Update updates a proxy owned by the current user.
// PUT /api/v1/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
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
	ip, err := h.ipSvc.UpdateForUser(c.Request.Context(), id, subject.UserID, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ip)
}

// Delete removes a proxy owned by the current user.
// DELETE /api/v1/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.ipSvc.DeleteForUser(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// Test runs a free connectivity test for one current-user proxy.
// POST /api/v1/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if _, err := h.ipSvc.GetByIDForUser(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.ipChecker.TestIP(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// TestAll runs free connectivity tests for all current-user proxies.
// POST /api/v1/proxies/test
func (h *ProxyHandler) TestAll(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	results, err := h.ipChecker.TestAllByUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results)
}
