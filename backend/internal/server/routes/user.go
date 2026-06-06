package routes

import (
	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers authenticated user routes.
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		// Profile, affiliate, and account binding routes.
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/aff", h.User.GetAffiliate)
			user.POST("/aff/transfer", h.User.TransferAffiliateQuota)
			user.POST("/account-bindings/email/send-code", h.User.SendEmailBindingCode)
			user.POST("/account-bindings/email", h.User.BindEmailIdentity)
			user.DELETE("/account-bindings/:provider", h.User.UnbindIdentity)
			user.POST("/auth-identities/bind/start", h.User.StartIdentityBinding)

			// Notification email settings.
			notifyEmail := user.Group("/notify-email")
			{
				notifyEmail.POST("/send-code", h.User.SendNotifyEmailCode)
				notifyEmail.POST("/verify", h.User.VerifyNotifyEmail)
				notifyEmail.PUT("/toggle", h.User.ToggleNotifyEmail)
				notifyEmail.DELETE("", h.User.RemoveNotifyEmail)
			}

			// TOTP two-factor authentication.
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
			}
		}

		// API key management.
		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// Usage records and dashboard summaries.
		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		// User announcements.
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// Redeem codes.
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// User subscriptions.
		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}

		// Unified account workbench.
		accounts := authenticated.Group("/accounts")
		{
			accounts.GET("", h.AccountWorkbench.ListMyAccounts)
			accounts.POST("/batch-import", h.AccountWorkbench.BatchImportMyAccounts)
			accounts.POST("/batch-delete", h.AccountWorkbench.BatchDeleteMyAccounts)
			accounts.GET("/export", h.AccountWorkbench.ExportMyAccounts)
			accounts.POST("/default-proxy", h.AccountWorkbench.BatchSetDefaultProxy)
			accounts.PUT("/:id", h.AccountWorkbench.UpdateMyAccount)
			accounts.DELETE("/:id", h.AccountWorkbench.DeleteMyAccount)
			accounts.PUT("/:id/default-proxy", h.AccountWorkbench.SetDefaultProxy)
			accounts.POST("/tasks", h.AccountWorkbench.SubmitTask)
		}

		proxies := authenticated.Group("/proxies")
		{
			proxies.GET("", h.Proxy.List)
			proxies.GET("/usable", h.Proxy.ListUsable)
			proxies.POST("", h.Proxy.Create)
			proxies.POST("/test", h.Proxy.TestAll)
			proxies.PUT("/:id", h.Proxy.Update)
			proxies.DELETE("/:id", h.Proxy.Delete)
			proxies.POST("/:id/test", h.Proxy.Test)
		}

		taskSettings := authenticated.Group("/task-settings")
		{
			templates := taskSettings.Group("/templates")
			{
				templates.GET("", h.TaskSettings.ListTemplates)
				templates.POST("", h.TaskSettings.SaveTemplate)
				templates.POST("/validate", h.TaskSettings.ValidateTemplate)
				templates.DELETE("/:id", h.TaskSettings.DeleteTemplate)
				templates.POST("/:id/copy", h.TaskSettings.CopyTemplate)
				templates.POST("/:id/default", h.TaskSettings.SetDefaultTemplate)
			}
		}

		// Public plan catalog.
		authenticated.GET("/plans", h.Plan.ListPlansForSale)
		authenticated.GET("/my-plan", h.Plan.GetMyPlan)
	}
}
