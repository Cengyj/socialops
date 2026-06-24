package server

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/handler"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/server/routes"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/Wei-Shaw/socialops/internal/setup"
	"github.com/Wei-Shaw/socialops/internal/web"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const frameSrcRefreshTimeout = 5 * time.Second

// SetupRouter 配置路由器中间件和路由
func SetupRouter(
	r *gin.Engine,
	handlers *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
) *gin.Engine {
	var cachedFrameOrigins atomic.Pointer[[]string]
	emptyOrigins := []string{}
	cachedFrameOrigins.Store(&emptyOrigins)

	refreshFrameOrigins := func() {
		ctx, cancel := context.WithTimeout(context.Background(), frameSrcRefreshTimeout)
		defer cancel()
		origins, err := settingService.GetFrameSrcOrigins(ctx)
		if err != nil {
			return
		}
		cachedFrameOrigins.Store(&origins)
	}
	refreshFrameOrigins()

	r.Use(middleware2.RequestLogger())
	r.Use(middleware2.Logger())
	r.Use(middleware2.CORS(cfg.CORS))
	r.Use(middleware2.SecurityHeaders(cfg.Security.CSP, func() []string {
		if p := cachedFrameOrigins.Load(); p != nil {
			return *p
		}
		return nil
	}))

	if web.HasEmbeddedFrontend() {
		frontendServer, err := web.NewFrontendServer(settingService)
		if err != nil {
			log.Printf("Warning: Failed to create frontend server with settings injection: %v, using embedded frontend fallback", err)
			r.Use(web.ServeEmbeddedFrontend())
			settingService.SetOnUpdateCallback(refreshFrameOrigins)
		} else {
			settingService.SetOnUpdateCallback(func() {
				frontendServer.InvalidateCache()
				refreshFrameOrigins()
			})
			r.Use(frontendServer.Middleware())
		}
	} else {
		settingService.SetOnUpdateCallback(refreshFrameOrigins)
	}

	registerRoutes(r, handlers, jwtAuth, adminAuth, settingService, cfg, redisClient)

	return r
}

// registerRoutes 注册所有 HTTP 路由
func registerRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
) {
	routes.RegisterCommonRoutes(r)

	v1 := r.Group("/api/v1")

	routes.RegisterAuthRoutes(v1, h, jwtAuth, redisClient, settingService)
	routes.RegisterUserRoutes(v1, h, jwtAuth, settingService)
	routes.RegisterAdminRoutes(v1, h, adminAuth)
	routes.RegisterPaymentRoutes(v1, h.Payment, h.PaymentWebhook, h.Admin.Payment, jwtAuth, adminAuth, settingService)

	handler.RegisterPageRoutes(v1, setup.GetDataDir(), gin.HandlerFunc(jwtAuth), gin.HandlerFunc(adminAuth), settingService)
}
