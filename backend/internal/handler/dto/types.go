package dto

import (
	"bytes"
	"encoding/json"
	"time"
)

type User struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	Balance       float64    `json:"balance"`
	Concurrency   int        `json:"concurrency"`
	Status        string     `json:"status"`
	AllowedGroups []int64    `json:"allowed_groups"`
	LastActiveAt  *time.Time `json:"last_active_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	BalanceNotifyEnabled       bool               `json:"balance_notify_enabled"`
	BalanceNotifyThresholdType string             `json:"balance_notify_threshold_type"`
	BalanceNotifyThreshold     *float64           `json:"balance_notify_threshold"`
	BalanceNotifyExtraEmails   []NotifyEmailEntry `json:"balance_notify_extra_emails"`
	TotalRecharged             float64            `json:"total_recharged"`
	RPMLimit                   int                `json:"rpm_limit"`

	APIKeys       []APIKey           `json:"api_keys,omitempty"`
	Subscriptions []UserSubscription `json:"subscriptions,omitempty"`
}

type AdminUser struct {
	User

	Notes      string            `json:"notes"`
	LastUsedAt *time.Time        `json:"last_used_at"`
	GroupRates map[int64]float64 `json:"group_rates,omitempty"`
}

type APIKey struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	GroupID     *int64     `json:"group_id"`
	Status      string     `json:"status"`
	IPWhitelist []string   `json:"ip_whitelist"`
	IPBlacklist []string   `json:"ip_blacklist"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	Quota       float64    `json:"quota"`
	QuotaUsed   float64    `json:"quota_used"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	RateLimit5h   float64    `json:"rate_limit_5h"`
	RateLimit1d   float64    `json:"rate_limit_1d"`
	RateLimit7d   float64    `json:"rate_limit_7d"`
	Usage5h       float64    `json:"usage_5h"`
	Usage1d       float64    `json:"usage_1d"`
	Usage7d       float64    `json:"usage_7d"`
	Window5hStart *time.Time `json:"window_5h_start"`
	Window1dStart *time.Time `json:"window_1d_start"`
	Window7dStart *time.Time `json:"window_7d_start"`
	Reset5hAt     *time.Time `json:"reset_5h_at,omitempty"`
	Reset1dAt     *time.Time `json:"reset_1d_at,omitempty"`
	Reset7dAt     *time.Time `json:"reset_7d_at,omitempty"`

	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

type Group struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
	IsExclusive    bool    `json:"is_exclusive"`
	Status         string  `json:"status"`

	SubscriptionType string   `json:"subscription_type"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd"`
	RPMLimit         int      `json:"rpm_limit"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminGroup struct {
	Group

	AccountCount            int64 `json:"account_count,omitempty"`
	ActiveAccountCount      int64 `json:"active_account_count,omitempty"`
	RateLimitedAccountCount int64 `json:"rate_limited_account_count,omitempty"`
	SortOrder               int   `json:"sort_order"`
}

type RedeemCode struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Type      string     `json:"type"`
	Value     float64    `json:"value"`
	Status    string     `json:"status"`
	UsedBy    *int64     `json:"used_by"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	GroupID      *int64  `json:"group_id"`
	PlanID       *int64  `json:"plan_id"`
	ValidityDays int     `json:"validity_days"`
	Notes        *string `json:"notes,omitempty"`

	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

type AdminRedeemCode struct {
	RedeemCode

	Notes string `json:"notes"`
}

type NullableTimeField struct {
	Set   bool
	Value *time.Time
}

func (f *NullableTimeField) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(data, []byte("null")) {
		f.Value = nil
		return nil
	}
	var value time.Time
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

type NullableInt64Field struct {
	Set   bool
	Value *int64
}

func (f *NullableInt64Field) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(data, []byte("null")) {
		f.Value = nil
		return nil
	}
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

type BatchUpdateRedeemCodeFields struct {
	Status    *string            `json:"status,omitempty"`
	ExpiresAt NullableTimeField  `json:"expires_at,omitempty"`
	Notes     *string            `json:"notes,omitempty"`
	GroupID   NullableInt64Field `json:"group_id,omitempty"`
	PlanID    NullableInt64Field `json:"plan_id,omitempty"`

	Type  *string  `json:"type,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type BatchUpdateRedeemCodesRequest struct {
	IDs    []int64                     `json:"ids" binding:"required,min=1"`
	Fields BatchUpdateRedeemCodeFields `json:"fields" binding:"required"`
}

type UsageCleanupFilters struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	UserID      *int64    `json:"user_id,omitempty"`
	APIKeyID    *int64    `json:"api_key_id,omitempty"`
	AccountID   *int64    `json:"account_id,omitempty"`
	GroupID     *int64    `json:"group_id,omitempty"`
	Model       *string   `json:"model,omitempty"`
	RequestType *string   `json:"request_type,omitempty"`
	Stream      *bool     `json:"stream,omitempty"`
	BillingType *int8     `json:"billing_type,omitempty"`
}

type UsageCleanupTask struct {
	ID           int64               `json:"id"`
	Status       string              `json:"status"`
	Filters      UsageCleanupFilters `json:"filters"`
	CreatedBy    int64               `json:"created_by"`
	DeletedRows  int64               `json:"deleted_rows"`
	ErrorMessage *string             `json:"error_message,omitempty"`
	CanceledBy   *int64              `json:"canceled_by,omitempty"`
	CanceledAt   *time.Time          `json:"canceled_at,omitempty"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	FinishedAt   *time.Time          `json:"finished_at,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type Setting struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserSubscription struct {
	ID              int64    `json:"id"`
	UserID          int64    `json:"user_id"`
	GroupID         int64    `json:"group_id"`
	PlanID          *int64   `json:"plan_id,omitempty"`
	PlanName        string   `json:"plan_name"`
	PlanPlatform    string   `json:"plan_platform"`
	QuotaUSD        *float64 `json:"quota_usd,omitempty"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd,omitempty"`

	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`

	DailyWindowStart   *time.Time `json:"daily_window_start"`
	WeeklyWindowStart  *time.Time `json:"weekly_window_start"`
	MonthlyWindowStart *time.Time `json:"monthly_window_start"`

	DailyUsageUSD   float64 `json:"daily_usage_usd"`
	WeeklyUsageUSD  float64 `json:"weekly_usage_usd"`
	MonthlyUsageUSD float64 `json:"monthly_usage_usd"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

type AdminUserSubscription struct {
	UserSubscription

	AssignedBy     *int64    `json:"assigned_by"`
	AssignedAt     time.Time `json:"assigned_at"`
	Notes          string    `json:"notes"`
	AssignedByUser *User     `json:"assigned_by_user,omitempty"`
}

type BulkAssignResult struct {
	SuccessCount  int                     `json:"success_count"`
	CreatedCount  int                     `json:"created_count"`
	ReusedCount   int                     `json:"reused_count"`
	FailedCount   int                     `json:"failed_count"`
	Subscriptions []AdminUserSubscription `json:"subscriptions"`
	Errors        []string                `json:"errors"`
	Statuses      map[string]string       `json:"statuses,omitempty"`
}

type PromoCode struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	BonusAmount float64    `json:"bonus_amount"`
	MaxUses     int        `json:"max_uses"`
	UsedCount   int        `json:"used_count"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PromoCodeUsage struct {
	ID          int64     `json:"id"`
	PromoCodeID int64     `json:"promo_code_id"`
	UserID      int64     `json:"user_id"`
	BonusAmount float64   `json:"bonus_amount"`
	UsedAt      time.Time `json:"used_at"`

	User *User `json:"user,omitempty"`
}
