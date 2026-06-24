package admin

import (
	"encoding/json"
	"strconv"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
)

// GlobalProxyHandler manages administrator-owned global fallback proxies.
type GlobalProxyHandler struct {
	svc *service.GlobalProxyService
}

type GlobalProxyCreateRequest struct {
	Name     string  `json:"name"`
	IPType   string  `json:"ip_type"`
	Endpoint *string `json:"endpoint"`
	Remark   *string `json:"remark"`
}

func NewGlobalProxyHandler(svc *service.GlobalProxyService) *GlobalProxyHandler {
	return &GlobalProxyHandler{svc: svc}
}

func (h *GlobalProxyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	svc, ok := h.service(c)
	if !ok {
		return
	}
	proxies, result, err := svc.List(c.Request.Context(), params, service.GlobalProxyListFilters{
		Status: c.Query("status"),
		IPType: c.Query("ip_type"),
		Search: c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, proxies, result.Total, page, pageSize)
}

func (h *GlobalProxyHandler) Create(c *gin.Context) {
	var req GlobalProxyCreateRequest
	if !bindGlobalProxyRequestBody(c, &req) {
		return
	}
	svc, ok := h.service(c)
	if !ok {
		return
	}
	proxy, err := svc.Create(c.Request.Context(), &service.CreateGlobalProxyInput{
		Name:     req.Name,
		IPType:   req.IPType,
		Endpoint: req.Endpoint,
		Remark:   req.Remark,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, proxy)
}

func (h *GlobalProxyHandler) Update(c *gin.Context) {
	id, ok := GlobalProxyPathID(c)
	if !ok {
		return
	}
	var input service.UpdateGlobalProxyInput
	if !bindGlobalProxyRequestBody(c, &input) {
		return
	}
	svc, ok := h.service(c)
	if !ok {
		return
	}
	proxy, err := svc.Update(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, proxy)
}

func (h *GlobalProxyHandler) Delete(c *gin.Context) {
	id, ok := GlobalProxyPathID(c)
	if !ok {
		return
	}
	svc, ok := h.service(c)
	if !ok {
		return
	}
	if err := svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *GlobalProxyHandler) Test(c *gin.Context) {
	id, ok := GlobalProxyPathID(c)
	if !ok {
		return
	}
	svc, ok := h.service(c)
	if !ok {
		return
	}
	result, err := svc.Test(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GlobalProxyHandler) TestAll(c *gin.Context) {
	svc, ok := h.service(c)
	if !ok {
		return
	}
	results, err := svc.TestAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results)
}

func (h *GlobalProxyHandler) service(c *gin.Context) (*service.GlobalProxyService, bool) {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("GLOBAL_PROXY_SERVICE_UNAVAILABLE", "global proxy service is unavailable"))
		return nil, false
	}
	return h.svc, true
}

func GlobalProxyPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func bindGlobalProxyRequestBody(c *gin.Context, target any) bool {
	rawBody, err := c.GetRawData()
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("GLOBAL_PROXY_INPUT_REQUIRED", "global proxy input is required"))
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &fields); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("GLOBAL_PROXY_INPUT_REQUIRED", "global proxy input is required"))
		return false
	}
	if _, exists := fields["user_id"]; exists {
		response.ErrorFrom(c, infraerrors.BadRequest("GLOBAL_PROXY_USER_ID_NOT_ACCEPTED", "user_id is not accepted"))
		return false
	}
	if err := json.Unmarshal(rawBody, target); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("GLOBAL_PROXY_INPUT_REQUIRED", "global proxy input is required"))
		return false
	}
	return true
}
