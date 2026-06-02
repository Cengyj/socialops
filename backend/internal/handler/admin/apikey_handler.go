package admin

import (
	"strconv"

	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{adminService: adminService}
}

type AdminUpdateAPIKeyGroupRequest struct {
	GroupID             *int64 `json:"group_id"`
	ResetRateLimitUsage *bool  `json:"reset_rate_limit_usage"`
}

func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	var req AdminUpdateAPIKeyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var resetKey *service.APIKey
	if req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage {
		resetKey, err = h.adminService.AdminResetAPIKeyRateLimitUsage(c.Request.Context(), keyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	result, err := h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if resetKey != nil && req.GroupID == nil {
		result.APIKey = resetKey
	}
	response.Success(c, gin.H{
		"api_key":                   dto.APIKeyFromService(result.APIKey),
		"auto_granted_group_access": result.AutoGrantedGroupAccess,
		"granted_group_id":          result.GrantedGroupID,
		"granted_group_name":        result.GrantedGroupName,
	})
}
