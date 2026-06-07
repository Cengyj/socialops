package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterUserRoutesKeepsStaticUsageRoutesReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	require.NotPanics(t, func() {
		RegisterUserRoutes(
			v1,
			newUserRoutesTestHandlers(),
			middleware.JWTAuthMiddleware(func(c *gin.Context) {
				c.Next()
			}),
			nil,
		)
	})

	requireRouteRegistered(t, router, "GET", "/api/v1/usage/stats")
	requireRouteRegistered(t, router, "GET", "/api/v1/usage/dashboard/stats")
	requireRouteRegistered(t, router, "GET", "/api/v1/usage/dashboard/trend")
	requireRouteRegistered(t, router, "GET", "/api/v1/usage/:id")
	requireRouteRegistered(t, router, "GET", "/api/v1/usage/:id/media")
}

func TestRegisterUserRoutesKeepsTaskHistoryOnUsageProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		newUserRoutesTestHandlers(),
		middleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	requireRouteNotRegistered(t, router, "POST", "/api/v1/accounts/tasks/estimate")
	requireRouteRegistered(t, router, "POST", "/api/v1/accounts/tasks")
	requireRouteRegistered(t, router, "GET", "/api/v1/usage")
	requireRouteNotRegistered(t, router, "GET", "/api/v1/accounts/tasks")
}

func TestRegisterUserRoutesMatchesStaticUsageRoutesBeforeIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		newUserRoutesTestHandlersWithUsageRepo(&usageRouteRepoStub{}),
		middleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
			c.Next()
		}),
		nil,
	)

	tests := []struct {
		path string
		want string
	}{
		{path: "/api/v1/usage/stats", want: `"total_requests":11`},
		{path: "/api/v1/usage/dashboard/stats", want: `"today_requests":3`},
		{path: "/api/v1/usage/dashboard/trend", want: `"2026-06-01"`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), tt.want)
			require.NotContains(t, rec.Body.String(), "Invalid usage ID")
		})
	}
}

func newUserRoutesTestHandlers() *handler.Handlers {
	return &handler.Handlers{
		User:             &handler.UserHandler{},
		APIKey:           &handler.APIKeyHandler{},
		Usage:            &handler.UsageHandler{},
		Redeem:           &handler.RedeemHandler{},
		Subscription:     &handler.SubscriptionHandler{},
		Announcement:     &handler.AnnouncementHandler{},
		AccountWorkbench: &handler.AccountWorkbenchHandler{},
		Proxy:            &handler.ProxyHandler{},
		TaskSettings:     &handler.TaskSettingsHandler{},
		Plan:             &handler.PlanHandler{},
		Totp:             &handler.TotpHandler{},
	}
}

func TestRegisterUserRoutesExposesUserScopedProxiesAndTaskSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		newUserRoutesTestHandlers(),
		middleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	requireRouteRegistered(t, router, "GET", "/api/v1/proxies")
	requireRouteRegistered(t, router, "GET", "/api/v1/proxies/usable")
	requireRouteRegistered(t, router, "POST", "/api/v1/proxies")
	requireRouteRegistered(t, router, "POST", "/api/v1/proxies/test")
	requireRouteRegistered(t, router, "PUT", "/api/v1/proxies/:id")
	requireRouteRegistered(t, router, "DELETE", "/api/v1/proxies/:id")
	requireRouteRegistered(t, router, "POST", "/api/v1/proxies/:id/test")
	requireRouteRegistered(t, router, "POST", "/api/v1/accounts/default-proxy")
	requireRouteRegistered(t, router, "GET", "/api/v1/task-settings/media")
	requireRouteRegistered(t, router, "GET", "/api/v1/task-settings/templates")
	requireRouteRegistered(t, router, "POST", "/api/v1/task-settings/templates")
	requireRouteRegistered(t, router, "POST", "/api/v1/task-settings/templates/validate")
	requireRouteRegistered(t, router, "POST", "/api/v1/task-settings/templates/:id/default")
}

func newUserRoutesTestHandlersWithUsageRepo(repo service.UsageLogRepository) *handler.Handlers {
	h := newUserRoutesTestHandlers()
	h.Usage = handler.NewUsageHandler(service.NewUsageService(repo, nil), nil)
	return h
}

func requireRouteRegistered(t *testing.T, router *gin.Engine, method string, path string) {
	t.Helper()
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	require.Failf(t, "route not registered", "%s %s", method, path)
}

func requireRouteNotRegistered(t *testing.T, router *gin.Engine, method string, path string) {
	t.Helper()
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			require.Failf(t, "route unexpectedly registered", "%s %s", method, path)
		}
	}
}

type usageRouteRepoStub struct{}

func (s *usageRouteRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return []service.UsageLog{}, &pagination.PaginationResult{Total: 0, Page: 1, PageSize: 20, Pages: 0}, nil
}

func (s *usageRouteRepoStub) GetStatsWithFilters(context.Context, usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	return &usagestats.UsageStats{TotalRequests: 11}, nil
}

func (s *usageRouteRepoStub) GetByID(context.Context, int64, int64) (*service.UsageLog, error) {
	return nil, service.ErrUsageLogNotFound
}

func (s *usageRouteRepoStub) GetUserDashboardStats(context.Context, int64) (*usagestats.UserDashboardStats, error) {
	return &usagestats.UserDashboardStats{TotalRequests: 7, TodayRequests: 3}, nil
}

func (s *usageRouteRepoStub) GetUserUsageTrendByUserID(context.Context, int64, time.Time, time.Time, string) ([]usagestats.TrendDataPoint, error) {
	return []usagestats.TrendDataPoint{{Date: "2026-06-01", Requests: 2}}, nil
}
