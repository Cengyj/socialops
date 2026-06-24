// Package routes provides HTTP route registration and handlers.
package routes

import (
	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers administrator routes.
func RegisterAdminRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
) {
	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	{
		registerDashboardRoutes(admin, h)
		registerUserManagementRoutes(admin, h)
		registerAnnouncementRoutes(admin, h)
		registerRedeemCodeRoutes(admin, h)
		registerPromoCodeRoutes(admin, h)
		registerSettingsRoutes(admin, h)
		registerBackupRoutes(admin, h)
		registerSystemRoutes(admin, h)
		registerSubscriptionRoutes(admin, h)
		registerGroupRoutes(admin, h)
		registerUserAttributeRoutes(admin, h)
		registerAffiliateRoutes(admin, h)
		registerSocialAccountAdminRoutes(admin, h)
		registerTotalAccountAdminRoutes(admin, h)
		registerGlobalProxyAdminRoutes(admin, h)
	}
}

func registerDashboardRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	dashboard := admin.Group("/dashboard")
	{
		dashboard.GET("/stats", h.Admin.Dashboard.GetStats)
		dashboard.GET("/trend", h.Admin.Dashboard.GetUsageTrend)
		dashboard.GET("/users-trend", h.Admin.Dashboard.GetUserUsageTrend)
		dashboard.GET("/users-ranking", h.Admin.Dashboard.GetUserSpendingRanking)
	}
}

func registerUserManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	users := admin.Group("/users")
	{
		users.GET("", h.Admin.User.List)
		users.GET("/:id", h.Admin.User.GetByID)
		users.POST("/:id/auth-identities", h.Admin.User.BindAuthIdentity)
		users.POST("", h.Admin.User.Create)
		users.PUT("/:id", h.Admin.User.Update)
		users.DELETE("/:id", h.Admin.User.Delete)
		users.POST("/:id/balance", h.Admin.User.UpdateBalance)
		users.GET("/:id/usage", h.Admin.User.GetUserUsage)
		users.GET("/:id/balance-history", h.Admin.User.GetBalanceHistory)
		users.POST("/batch-concurrency", h.Admin.User.BatchUpdateConcurrency)

		users.GET("/:id/attributes", h.Admin.UserAttribute.GetUserAttributes)
		users.PUT("/:id/attributes", h.Admin.UserAttribute.UpdateUserAttributes)
	}
}

func registerAnnouncementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	announcements := admin.Group("/announcements")
	{
		announcements.GET("", h.Admin.Announcement.List)
		announcements.POST("", h.Admin.Announcement.Create)
		announcements.GET("/:id", h.Admin.Announcement.GetByID)
		announcements.PUT("/:id", h.Admin.Announcement.Update)
		announcements.DELETE("/:id", h.Admin.Announcement.Delete)
		announcements.GET("/:id/read-status", h.Admin.Announcement.ListReadStatus)
	}
}

func registerRedeemCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	codes := admin.Group("/redeem-codes")
	{
		codes.GET("", h.Admin.Redeem.List)
		codes.GET("/stats", h.Admin.Redeem.GetStats)
		codes.GET("/export", h.Admin.Redeem.Export)
		codes.GET("/:id", h.Admin.Redeem.GetByID)
		codes.POST("/create-and-redeem", h.Admin.Redeem.CreateAndRedeem)
		codes.POST("/generate", h.Admin.Redeem.Generate)
		codes.DELETE("/:id", h.Admin.Redeem.Delete)
		codes.POST("/batch-delete", h.Admin.Redeem.BatchDelete)
		codes.POST("/batch-update", h.Admin.Redeem.BatchUpdate)
		codes.POST("/:id/expire", h.Admin.Redeem.Expire)
	}
}

func registerPromoCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	promoCodes := admin.Group("/promo-codes")
	{
		promoCodes.GET("", h.Admin.Promo.List)
		promoCodes.GET("/:id", h.Admin.Promo.GetByID)
		promoCodes.POST("", h.Admin.Promo.Create)
		promoCodes.PUT("/:id", h.Admin.Promo.Update)
		promoCodes.DELETE("/:id", h.Admin.Promo.Delete)
		promoCodes.GET("/:id/usages", h.Admin.Promo.GetUsages)
	}
}

func registerSettingsRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	adminSettings := admin.Group("/settings")
	{
		adminSettings.GET("", h.Admin.Setting.GetSettings)
		adminSettings.PUT("", h.Admin.Setting.UpdateSettings)
		adminSettings.POST("/test-smtp", h.Admin.Setting.TestSMTPConnection)
		adminSettings.POST("/send-test-email", h.Admin.Setting.SendTestEmail)
		adminSettings.GET("/email-templates", h.Admin.Setting.ListEmailTemplates)
		adminSettings.POST("/email-template-preview", h.Admin.Setting.PreviewEmailTemplate)
		adminSettings.GET("/email-templates/:event/:locale", h.Admin.Setting.GetEmailTemplate)
		adminSettings.PUT("/email-templates/:event/:locale", h.Admin.Setting.UpdateEmailTemplate)
		adminSettings.POST("/email-templates/:event/:locale/restore-official", h.Admin.Setting.RestoreOfficialEmailTemplate)
		adminSettings.GET("/admin-api-key", h.Admin.Setting.GetAdminAPIKey)
		adminSettings.POST("/admin-api-key/regenerate", h.Admin.Setting.RegenerateAdminAPIKey)
		adminSettings.DELETE("/admin-api-key", h.Admin.Setting.DeleteAdminAPIKey)
	}
}

func registerBackupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	backup := admin.Group("/backups")
	{
		backup.GET("/s3-config", h.Admin.Backup.GetS3Config)
		backup.PUT("/s3-config", h.Admin.Backup.UpdateS3Config)
		backup.POST("/s3-config/test", h.Admin.Backup.TestS3Connection)
		backup.GET("/schedule", h.Admin.Backup.GetSchedule)
		backup.PUT("/schedule", h.Admin.Backup.UpdateSchedule)
		backup.POST("", h.Admin.Backup.CreateBackup)
		backup.GET("", h.Admin.Backup.ListBackups)
		backup.GET("/:id", h.Admin.Backup.GetBackup)
		backup.DELETE("/:id", h.Admin.Backup.DeleteBackup)
		backup.GET("/:id/download-url", h.Admin.Backup.GetDownloadURL)
		backup.POST("/:id/restore", h.Admin.Backup.RestoreBackup)
	}
}

func registerSystemRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	system := admin.Group("/system")
	{
		system.GET("/version", h.Admin.System.GetVersion)
		system.GET("/check-updates", h.Admin.System.CheckUpdates)
		system.POST("/update", h.Admin.System.PerformUpdate)
		system.POST("/rollback", h.Admin.System.Rollback)
		system.POST("/restart", h.Admin.System.RestartService)
	}
}

func registerSubscriptionRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	subscriptions := admin.Group("/subscriptions")
	{
		subscriptions.GET("", h.Admin.Subscription.List)
		subscriptions.POST("", h.Admin.Subscription.Create)
		subscriptions.POST("/bulk", h.Admin.Subscription.BulkCreate)
		subscriptions.GET("/:id", h.Admin.Subscription.GetByID)
		subscriptions.GET("/:id/progress", h.Admin.Subscription.GetProgress)
		subscriptions.POST("/:id/extend", h.Admin.Subscription.Extend)
		subscriptions.POST("/:id/reset-quota", h.Admin.Subscription.ResetQuota)
		subscriptions.DELETE("/:id", h.Admin.Subscription.Revoke)
	}

	admin.GET("/users/:id/subscriptions", h.Admin.Subscription.ListByUser)
	admin.GET("/groups/:id/subscriptions", h.Admin.Subscription.ListByGroup)
}

func registerGroupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	groups := admin.Group("/groups")
	{
		groups.GET("", h.Admin.Group.List)
		groups.POST("", h.Admin.Group.Create)
		groups.GET("/:id", h.Admin.Group.GetByID)
		groups.PUT("/:id", h.Admin.Group.Update)
		groups.DELETE("/:id", h.Admin.Group.Delete)
	}
}

func registerUserAttributeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	attrs := admin.Group("/user-attributes")
	{
		attrs.GET("", h.Admin.UserAttribute.ListDefinitions)
		attrs.POST("", h.Admin.UserAttribute.CreateDefinition)
		attrs.POST("/batch", h.Admin.UserAttribute.GetBatchUserAttributes)
		attrs.PUT("/reorder", h.Admin.UserAttribute.ReorderDefinitions)
		attrs.PUT("/:id", h.Admin.UserAttribute.UpdateDefinition)
		attrs.DELETE("/:id", h.Admin.UserAttribute.DeleteDefinition)
	}
}

func registerAffiliateRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	affiliates := admin.Group("/affiliates")
	{
		affiliates.GET("/invites", h.Admin.Affiliate.ListInviteRecords)
		affiliates.GET("/rebates", h.Admin.Affiliate.ListRebateRecords)
		affiliates.GET("/transfers", h.Admin.Affiliate.ListTransferRecords)

		users := affiliates.Group("/users")
		{
			users.GET("", h.Admin.Affiliate.ListUsers)
			users.GET("/lookup", h.Admin.Affiliate.LookupUsers)
			users.POST("/batch-rate", h.Admin.Affiliate.BatchSetRate)
			users.GET("/:user_id/overview", h.Admin.Affiliate.GetUserOverview)
			users.PUT("/:user_id", h.Admin.Affiliate.UpdateUserSettings)
			users.DELETE("/:user_id", h.Admin.Affiliate.ClearUserSettings)
		}
	}
}

func registerSocialAccountAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	sa := admin.Group("/accounts")
	{
		sa.GET("", h.Admin.AccountWorkbench.List)
		sa.GET("/stats", h.Admin.AccountWorkbench.GetStats)
		sa.GET("/export", h.Admin.AccountWorkbench.Export)
		sa.POST("/tasks", h.Admin.AccountWorkbench.SubmitTask)
		sa.POST("/store-workbench", h.Admin.AccountWorkbench.StoreWorkbench)
		sa.GET("/:id", h.Admin.AccountWorkbench.GetByID)
		sa.POST("", h.Admin.AccountWorkbench.Create)
		sa.POST("/import", h.Admin.AccountWorkbench.Import)
		sa.POST("/batch-delete", h.Admin.AccountWorkbench.BatchDelete)
		sa.PUT("/:id", h.Admin.AccountWorkbench.Update)
		sa.DELETE("/:id", h.Admin.AccountWorkbench.Delete)
	}
}

func registerTotalAccountAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	pool := admin.Group("/total-accounts")
	{
		pool.GET("", h.Admin.TotalAccounts.List)
		pool.GET("/export", h.Admin.TotalAccounts.Export)
		pool.POST("/import", h.Admin.TotalAccounts.Import)
		pool.POST("/batch-assign", h.Admin.TotalAccounts.BatchAssign)
		pool.POST("/batch-reclaim", h.Admin.TotalAccounts.BatchReclaim)
		pool.POST("/batch-delete", h.Admin.TotalAccounts.BatchDelete)
		pool.PUT("/:id", h.Admin.TotalAccounts.Update)
		pool.POST("/:id/assign", h.Admin.TotalAccounts.Assign)
		pool.POST("/:id/reclaim", h.Admin.TotalAccounts.Reclaim)
	}
}

func registerGlobalProxyAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	proxies := admin.Group("/global-proxies")
	{
		proxies.GET("", h.Admin.GlobalProxies.List)
		proxies.POST("", h.Admin.GlobalProxies.Create)
		proxies.POST("/test", h.Admin.GlobalProxies.TestAll)
		proxies.PUT("/:id", h.Admin.GlobalProxies.Update)
		proxies.DELETE("/:id", h.Admin.GlobalProxies.Delete)
		proxies.POST("/:id/test", h.Admin.GlobalProxies.Test)
	}
}
