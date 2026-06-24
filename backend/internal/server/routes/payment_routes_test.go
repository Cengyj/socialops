package routes

import (
	"testing"

	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/handler/admin"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterPaymentRoutesKeepsRefundEligibleProvidersRouteReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	require.NotPanics(t, func() {
		RegisterPaymentRoutes(
			v1,
			&handler.PaymentHandler{},
			&handler.PaymentWebhookHandler{},
			&admin.PaymentHandler{},
			middleware.JWTAuthMiddleware(func(c *gin.Context) {
				c.Next()
			}),
			middleware.AdminAuthMiddleware(func(c *gin.Context) {
				c.Next()
			}),
			nil,
		)
	})

	requireRouteRegistered(t, router, "GET", "/api/v1/payment/orders/refund-eligible-providers")
	requireRouteRegistered(t, router, "GET", "/api/v1/payment/orders/:id")
}

func TestRegisterPaymentRoutesKeepsCurrentPlanCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterPaymentRoutes(
		v1,
		&handler.PaymentHandler{},
		&handler.PaymentWebhookHandler{},
		&admin.PaymentHandler{},
		middleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	requireRouteRegistered(t, router, "GET", "/api/v1/payment/plans")
}

func TestRegisterPaymentRoutesDoesNotExposeChannelManagementOrEmptyUserChannelList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterPaymentRoutes(
		v1,
		&handler.PaymentHandler{},
		&handler.PaymentWebhookHandler{},
		&admin.PaymentHandler{},
		middleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		middleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	requireRouteNotRegistered(t, router, "GET", "/api/v1/payment/channels")
	requireRouteNotRegistered(t, router, "GET", "/api/v1/admin/payment/channels")
	requireRouteNotRegistered(t, router, "POST", "/api/v1/admin/payment/channels")
	requireRouteNotRegistered(t, router, "PUT", "/api/v1/admin/payment/channels/:id")
	requireRouteNotRegistered(t, router, "DELETE", "/api/v1/admin/payment/channels/:id")
}
