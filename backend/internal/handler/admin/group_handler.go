package admin

import (
	"github.com/Wei-Shaw/socialops/internal/pkg/response"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// GroupHandler is a SocialOps business/subscription-group skeleton. It does not
// expose AI routing group behavior.
type GroupHandler struct {
	adminService         service.AdminService
	dashboardService     *service.DashboardService
	groupCapacityService *service.GroupCapacityService
}

func NewGroupHandler(adminService service.AdminService, dashboardService *service.DashboardService, groupCapacityService *service.GroupCapacityService) *GroupHandler {
	return &GroupHandler{
		adminService:         adminService,
		dashboardService:     dashboardService,
		groupCapacityService: groupCapacityService,
	}
}

func (h *GroupHandler) List(c *gin.Context) {
	response.Paginated(c, []gin.H{}, 0, 1, 20)
}

func (h *GroupHandler) GetByID(c *gin.Context) {
	response.Success(c, gin.H{"id": c.Param("id"), "status": "skeleton"})
}

func (h *GroupHandler) Create(c *gin.Context) {
	response.Success(c, gin.H{"created": false, "message": "SocialOps group backend is not configured yet"})
}

func (h *GroupHandler) Update(c *gin.Context) {
	response.Success(c, gin.H{"updated": false, "message": "SocialOps group backend is not configured yet"})
}

func (h *GroupHandler) Delete(c *gin.Context) {
	response.Success(c, gin.H{"deleted": false, "message": "SocialOps group backend is not configured yet"})
}
