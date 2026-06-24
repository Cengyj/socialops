package handler

import (
	"encoding/json"
	"strconv"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

// ProxyHandler manages current-user SocialOps execution proxies.
type ProxyHandler struct {
	ipSvc     *service.SocialIPService
	ipChecker *service.SocialIPChecker
}

type proxyCreateRequest struct {
	Name     string  `json:"name"`
	IPType   string  `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

// NewProxyHandler creates a user-scoped proxy handler.
func NewProxyHandler(ipSvc *service.SocialIPService, ipChecker *service.SocialIPChecker) *ProxyHandler {
	return &ProxyHandler{ipSvc: ipSvc, ipChecker: ipChecker}
}

// List returns proxies owned by the current user.
// GET /api/v1/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := proxyFiltersFromQuery(c)
	ipSvc, ok := h.proxyIPService(c)
	if !ok {
		return
	}
	ips, result, err := ipSvc.ListByUser(c.Request.Context(), subject.UserID, params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, ips, result.Total, page, pageSize)
}

// ListUsable returns assignable online proxies owned by the current user.
// GET /api/v1/proxies/usable
func (h *ProxyHandler) ListUsable(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	ipSvc, ok := h.proxyIPService(c)
	if !ok {
		return
	}
	ips, err := ipSvc.ListUsableByUser(c.Request.Context(), subject.UserID)
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
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	var req proxyCreateRequest
	if !bindProxyRequestBody(c, &req) {
		return
	}
	ipSvc, ok := h.proxyIPService(c)
	if !ok {
		return
	}
	ip, err := ipSvc.Create(c.Request.Context(), &service.CreateSocialIPInput{
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

// Update updates a proxy owned by the current user. The request body must not
// contain user_id; ownership always comes from the JWT subject.
// PUT /api/v1/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	id, ok := proxyPathID(c)
	if !ok {
		return
	}
	var input service.UpdateSocialIPInput
	if !bindProxyRequestBody(c, &input) {
		return
	}
	ipSvc, ok := h.proxyIPService(c)
	if !ok {
		return
	}
	ip, err := ipSvc.UpdateForUser(c.Request.Context(), id, subject.UserID, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ip)
}

// Delete removes a proxy owned by the current user.
// DELETE /api/v1/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	id, ok := proxyPathID(c)
	if !ok {
		return
	}
	ipSvc, ok := h.proxyIPService(c)
	if !ok {
		return
	}
	if err := ipSvc.DeleteForUser(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// Test runs a free connectivity test for one current-user proxy.
// POST /api/v1/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	id, ok := proxyPathID(c)
	if !ok {
		return
	}
	ipChecker, ok := h.proxyIPChecker(c)
	if !ok {
		return
	}
	result, err := ipChecker.TestIPForUser(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// TestAll runs free connectivity tests for all current-user proxies.
// POST /api/v1/proxies/test
func (h *ProxyHandler) TestAll(c *gin.Context) {
	subject, ok := proxyAuthSubject(c)
	if !ok {
		return
	}
	ipChecker, ok := h.proxyIPChecker(c)
	if !ok {
		return
	}
	results, err := ipChecker.TestAllByUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results)
}

func proxyAuthSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func (h *ProxyHandler) proxyIPService(c *gin.Context) (*service.SocialIPService, bool) {
	if h == nil || h.ipSvc == nil {
		response.ErrorFrom(c, proxyServiceUnavailableError())
		return nil, false
	}
	return h.ipSvc, true
}

func (h *ProxyHandler) proxyIPChecker(c *gin.Context) (*service.SocialIPChecker, bool) {
	if h == nil || h.ipChecker == nil {
		response.ErrorFrom(c, proxyServiceUnavailableError())
		return nil, false
	}
	return h.ipChecker, true
}

func proxyPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func proxyFiltersFromQuery(c *gin.Context) service.SocialIPListFilters {
	return service.SocialIPListFilters{
		Status: c.Query("status"),
		IPType: c.Query("ip_type"),
		Search: c.Query("search"),
	}
}

func bindProxyRequestBody(c *gin.Context, target any) bool {
	rawBody, ok := readProxyRequestBody(c)
	if !ok {
		return false
	}
	if err := json.Unmarshal(rawBody, target); err != nil {
		response.ErrorFrom(c, proxyInputRequiredError())
		return false
	}
	return true
}

func readProxyRequestBody(c *gin.Context) ([]byte, bool) {
	rawBody, err := c.GetRawData()
	if err != nil {
		response.ErrorFrom(c, proxyInputRequiredError())
		return nil, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &fields); err != nil {
		response.ErrorFrom(c, proxyInputRequiredError())
		return nil, false
	}
	if _, exists := fields["user_id"]; exists {
		response.ErrorFrom(c, infraerrors.BadRequest("SOCIAL_IP_USER_ID_NOT_ACCEPTED", "user_id is not accepted"))
		return nil, false
	}
	return rawBody, true
}

func proxyInputRequiredError() error {
	return infraerrors.BadRequest("SOCIAL_IP_INPUT_REQUIRED", "social IP input is required")
}

func proxyServiceUnavailableError() error {
	return infraerrors.ServiceUnavailable("SOCIAL_IP_SERVICE_UNAVAILABLE", "social IP service is unavailable")
}
