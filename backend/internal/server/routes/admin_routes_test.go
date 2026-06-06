package routes

import (
	"net/http"
	"net/http/httptest"
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
	requireRouteRegistered(t, router, "GET", "/api/v1/admin/total-accounts")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/accounts/tasks/estimate")
	requireRouteNotRegistered(t, router, "PUT", "/api/v1/admin/accounts/:id/default-proxy")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/accounts/register")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/accounts/store-workbench")
	removedAdminProxyPath := "/api/v1/admin" + "/proxies"
	requireRouteNotRegistered(t, router, "GET", removedAdminProxyPath)
	requireRouteNotRegistered(t, router, "POST", removedAdminProxyPath)
}

func TestRegisterAdminRoutesRiskControlStubsUseStandardEnvelope(t *testing.T) {
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
		path string
		want []string
	}{
		{
			path: "/api/v1/admin/risk-control/status",
			want: []string{`"code":0`, `"message":"success"`, `"data":`, `"enabled":false`, `"status":"disabled"`},
		},
		{
			path: "/api/v1/admin/risk-control/config",
			want: []string{`"code":0`, `"message":"success"`, `"data":`, `"enabled":false`},
		},
		{
			path: "/api/v1/admin/risk-control/logs?page=2&page_size=5",
			want: []string{`"code":0`, `"message":"success"`, `"items":[]`, `"total":0`, `"page":2`, `"page_size":5`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			for _, want := range tt.want {
				require.Contains(t, rec.Body.String(), want)
			}
		})
	}
}

func newAdminRoutesTestHandlers() *handler.Handlers {
	return &handler.Handlers{
		Admin: &handler.AdminHandlers{
			Dashboard:        &admin.DashboardHandler{},
			User:             &admin.UserHandler{},
			Group:            &admin.GroupHandler{},
			Announcement:     &admin.AnnouncementHandler{},
			DataManagement:   &admin.DataManagementHandler{},
			Backup:           &admin.BackupHandler{},
			Redeem:           &admin.RedeemHandler{},
			Promo:            &admin.PromoHandler{},
			Setting:          &admin.SettingHandler{},
			System:           &admin.SystemHandler{},
			Subscription:     &admin.SubscriptionHandler{},
			UserAttribute:    &admin.UserAttributeHandler{},
			APIKey:           &admin.AdminAPIKeyHandler{},
			Affiliate:        &admin.AffiliateHandler{},
			AccountWorkbench: &admin.AccountWorkbenchAdminHandler{},
			TotalAccounts:    &admin.TotalAccountsHandler{},
		},
	}
}
