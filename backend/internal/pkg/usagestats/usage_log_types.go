// Package usagestats provides types for usage statistics and reporting.
package usagestats

import "time"

// DashboardStats captures current SocialOps admin dashboard aggregates.
type DashboardStats struct {
	// 用户统计
	TotalUsers    int64 `json:"total_users"`
	TodayNewUsers int64 `json:"today_new_users"` // 今日新增用户数
	ActiveUsers   int64 `json:"active_users"`    // 今日有请求的用户数
	// 小时活跃用户数（UTC 当前小时）
	HourlyActiveUsers int64 `json:"hourly_active_users"`

	// 预聚合新鲜度
	StatsUpdatedAt string `json:"stats_updated_at"`
	StatsStale     bool   `json:"stats_stale"`

	// 账户统计
	TotalAccounts     int64 `json:"total_accounts"`
	NormalAccounts    int64 `json:"normal_accounts"`    // 正常账户数 (schedulable=true, status=active)
	ErrorAccounts     int64 `json:"error_accounts"`     // 异常账户数 (status=error)
	RateLimitAccounts int64 `json:"ratelimit_accounts"` // 限流账户数
	OverloadAccounts  int64 `json:"overload_accounts"`  // 过载账户数

	// 任务执行统计
	TotalOperations int64   `json:"total_operations"`
	TotalCharged    float64 `json:"total_charged"` // 累计实际扣除
	TodayOperations int64   `json:"today_operations"`
	TodayCharged    float64 `json:"today_charged"` // 今日实际扣除

	// 系统运行统计
	AverageDurationMs float64 `json:"average_duration_ms"` // 平均响应时间

	// 性能指标
	RecentOperationsPerMinute int64 `json:"recent_operations_per_minute"` // 近5分钟平均每分钟任务数
}

// TrendDataPoint represents a SocialOps operation trend bucket.
type TrendDataPoint struct {
	Date       string  `json:"date"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"` // 实际扣除
}

// UserUsageTrendPoint represents a user-level SocialOps operation trend bucket.
type UserUsageTrendPoint struct {
	Date       string  `json:"date"`
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Username   string  `json:"username"`
	Operations int64   `json:"operations"`
	Charged    float64 `json:"charged"` // 实际扣除
}

// UserSpendingRankingItem represents a user spending ranking row.
type UserSpendingRankingItem struct {
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Charged    float64 `json:"charged"` // 实际扣除
	Operations int64   `json:"operations"`
}

// UserSpendingRankingResponse represents ranking rows plus total spend for the time range.
type UserSpendingRankingResponse struct {
	Ranking         []UserSpendingRankingItem `json:"ranking"`
	TotalCharged    float64                   `json:"total_charged"`
	TotalOperations int64                     `json:"total_operations"`
}

// UserDashboardStats captures current-user SocialOps dashboard aggregates.
type UserDashboardStats struct {
	TotalOperations int64   `json:"total_operations"`
	TotalCharged    float64 `json:"total_charged"` // 累计实际扣除
	TodayOperations int64   `json:"today_operations"`
	TodayCharged    float64 `json:"today_charged"` // 今日实际扣除

	// 性能指标
	RecentOperationsPerMinute int64 `json:"recent_operations_per_minute"` // 近5分钟平均每分钟任务数

	// 按"有效平台"维度拆分（与 ops 路径口径一致：group.platform 优先，否则 account.platform）
	ByPlatform []PlatformDashboardStats `json:"by_platform,omitempty"`
}

// PlatformDashboardStats 单个平台的用量明细。
type PlatformDashboardStats struct {
	Platform        string  `json:"platform"`
	TotalOperations int64   `json:"total_operations"`
	TotalCharged    float64 `json:"total_charged"`
	TodayOperations int64   `json:"today_operations"`
	TodayCharged    float64 `json:"today_charged"`
}

// UsageLogFilters represents filters for usage log queries
type UsageLogFilters struct {
	UserID      int64
	Operation   string
	Platform    string
	AccountName string
	Status      string
	StartTime   *time.Time
	EndTime     *time.Time
}

// UsageStats represents current-user SocialOps usage statistics.
type UsageStats struct {
	TotalOperations int64   `json:"total_operations"`
	SuccessCount    int64   `json:"success_count"`
	FailedCount     int64   `json:"failed_count"`
	TotalCharged    float64 `json:"total_charged"`
}
