package routes

import (
	"testing"

	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/handler/admin"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterAdminRoutesDoesNotExposeAccountTaskEstimate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	require.NotPanics(t, func() {
		RegisterAdminRoutes(
			v1,
			newAdminRoutesTestHandlers(),
			middleware.AdminAuthMiddleware(func(c *gin.Context) {
				c.Next()
			}),
		)
	})

	requireRouteRegistered(t, router, "POST", "/api/v1/admin/accounts/tasks")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/accounts/store-workbench")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/total-accounts")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/total-accounts/export")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/total-accounts/import")
	requireRouteRegistered(t, router, "PUT", "/api/v1/admin/total-accounts/:id")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/global-proxies")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/global-proxies")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/global-proxies/test")
	requireRouteRegistered(t, router, "PUT", "/api/v1/admin/global-proxies/:id")
	requireRouteRegistered(t, router, "DELETE", "/api/v1/admin/global-proxies/:id")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/global-proxies/:id/test")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/total-accounts")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/accounts/register")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/accounts/tasks/estimate")
	requireRouteNotRegistered(t, router, "PUT", "/api/v1/admin/accounts/:id/default-proxy")
	requireRouteNotRegistered(t, router, "GET", "/api/v1/admin/login-proxies")
	removedAdminProxyPath := "/api/v1/admin" + "/proxies"
	requireRouteNotRegistered(t, router, "GET", removedAdminProxyPath)
	requireRouteNotRegistered(t, router, "POST", removedAdminProxyPath)
}

func TestRegisterAdminRoutesDoesNotExposeRemovedAPIKeyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	tests := []struct {
		method string
		path   string
	}{
		{method: "PUT", path: "/api/v1/admin/api-keys/:id"},
		{method: "GET", path: "/api/v1/admin/dashboard/api-keys-trend"},
		{method: "POST", path: "/api/v1/admin/dashboard/api-keys-usage"},
		{method: "GET", path: "/api/v1/admin/users/:id/api-keys"},
		{method: "GET", path: "/api/v1/admin/users/:id/rpm-status"},
		{method: "POST", path: "/api/v1/admin/users/:id/replace-group"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			requireRouteNotRegistered(t, router, tt.method, tt.path)
		})
	}
}

func TestRegisterAdminRoutesDoesNotExposeEmptyDashboardCompatibilityRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	requireRouteRegistered(t, router, "GET", "/api/v1/admin/dashboard/stats")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/dashboard/trend")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/dashboard/users-trend")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/dashboard/users-ranking")

	tests := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/api/v1/admin/dashboard/snapshot-v2"},
		{method: "GET", path: "/api/v1/admin/dashboard/realtime"},
		{method: "GET", path: "/api/v1/admin/dashboard/groups"},
		{method: "POST", path: "/api/v1/admin/dashboard/users-usage"},
		{method: "GET", path: "/api/v1/admin/dashboard/user-breakdown"},
		{method: "POST", path: "/api/v1/admin/dashboard/aggregation/backfill"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			requireRouteNotRegistered(t, router, tt.method, tt.path)
		})
	}
}

func TestRegisterAdminRoutesKeepsSystemAdminAPIKeySettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	requireRouteRegistered(t, router, "GET", "/api/v1/admin/settings/admin-api-key")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/settings/admin-api-key/regenerate")
	requireRouteRegistered(t, router, "DELETE", "/api/v1/admin/settings/admin-api-key")
}

func TestRegisterAdminRoutesDoesNotExposeSubscriptionAssignCompatibilityRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	requireRouteRegistered(t, router, "POST", "/api/v1/admin/subscriptions")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/subscriptions/bulk")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/subscriptions/assign")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/subscriptions/bulk-assign")
}

func TestRegisterAdminRoutesDoesNotExposeDeprecatedDataManagementRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	requireRouteRegistered(t, router, "GET", "/api/v1/admin/backups")
	requireRouteRegistered(t, router, "POST", "/api/v1/admin/backups")
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/backups/s3-config")

	tests := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/api/v1/admin/data-management/agent/health"},
		{method: "GET", path: "/api/v1/admin/data-management/config"},
		{method: "PUT", path: "/api/v1/admin/data-management/config"},
		{method: "GET", path: "/api/v1/admin/data-management/sources/:source_type/profiles"},
		{method: "POST", path: "/api/v1/admin/data-management/sources/:source_type/profiles"},
		{method: "PUT", path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id"},
		{method: "DELETE", path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id"},
		{method: "POST", path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id/activate"},
		{method: "POST", path: "/api/v1/admin/data-management/s3/test"},
		{method: "GET", path: "/api/v1/admin/data-management/s3/profiles"},
		{method: "POST", path: "/api/v1/admin/data-management/s3/profiles"},
		{method: "PUT", path: "/api/v1/admin/data-management/s3/profiles/:profile_id"},
		{method: "DELETE", path: "/api/v1/admin/data-management/s3/profiles/:profile_id"},
		{method: "POST", path: "/api/v1/admin/data-management/s3/profiles/:profile_id/activate"},
		{method: "POST", path: "/api/v1/admin/data-management/backups"},
		{method: "GET", path: "/api/v1/admin/data-management/backups"},
		{method: "GET", path: "/api/v1/admin/data-management/backups/:job_id"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			requireRouteNotRegistered(t, router, tt.method, tt.path)
		})
	}
}

func TestRegisterAdminRoutesDoesNotExposeRiskControlStubs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		newAdminRoutesTestHandlers(),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
	)

	requireRouteNotRegistered(t, router, "GET", "/api/v1/admin/risk-control/status")
	requireRouteNotRegistered(t, router, "GET", "/api/v1/admin/risk-control/config")
	requireRouteNotRegistered(t, router, "GET", "/api/v1/admin/risk-control/logs")
}

func newAdminRoutesTestHandlers() *handler.Handlers {
	return &handler.Handlers{
		Admin: &handler.AdminHandlers{
			Dashboard:        &admin.DashboardHandler{},
			User:             &admin.UserHandler{},
			Group:            &admin.GroupHandler{},
			Announcement:     &admin.AnnouncementHandler{},
			Backup:           &admin.BackupHandler{},
			Redeem:           &admin.RedeemHandler{},
			Promo:            &admin.PromoHandler{},
			Setting:          &admin.SettingHandler{},
			System:           &admin.SystemHandler{},
			Subscription:     &admin.SubscriptionHandler{},
			UserAttribute:    &admin.UserAttributeHandler{},
			Affiliate:        &admin.AffiliateHandler{},
			AccountWorkbench: &admin.AccountWorkbenchAdminHandler{},
			TotalAccounts:    &admin.TotalAccountsHandler{},
			GlobalProxies:    &admin.GlobalProxyHandler{},
		},
	}
}
