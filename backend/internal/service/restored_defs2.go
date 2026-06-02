package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

// Restored definitions (batch 2) — interfaces/types/errors that preserved
// repository and middleware code depends on, whose original homes were removed.

// ── Billing cache ──

type BillingCache interface {
	GetUserBalance(ctx context.Context, userID int64) (float64, error)
	SetUserBalance(ctx context.Context, userID int64, balance float64) error
	DeductUserBalance(ctx context.Context, userID int64, amount float64) error
	InvalidateUserBalance(ctx context.Context, userID int64) error

	GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error)
	SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error
	UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error
	InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error

	GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error)
	SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error
	UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error
	InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error
}

type APIKeyRateLimitCacheData struct {
	Usage5h  float64 `json:"usage_5h"`
	Usage1d  float64 `json:"usage_1d"`
	Usage7d  float64 `json:"usage_7d"`
	Window5h int64   `json:"window_5h"`
	Window1d int64   `json:"window_1d"`
	Window7d int64   `json:"window_7d"`
}

// ── Identity cache (account fingerprint) ──

type IdentityCache interface {
	GetFingerprint(ctx context.Context, accountID int64) (*Fingerprint, error)
	SetFingerprint(ctx context.Context, accountID int64, fp *Fingerprint) error
	GetMaskedSessionID(ctx context.Context, accountID int64) (string, error)
	SetMaskedSessionID(ctx context.Context, accountID int64, sessionID string) error
}

type Fingerprint struct {
	ClientID                string
	UserAgent               string
	StainlessLang           string
	StainlessPackageVersion string
	StainlessOS             string
	StainlessArch           string
	StainlessRuntime        string
	StainlessRuntimeVersion string
	UpdatedAt               int64 `json:",omitempty"`
}

// ── Subscription errors ──

var (
	ErrSubscriptionAlreadyExists  = infraerrors.Conflict("SUBSCRIPTION_ALREADY_EXISTS", "subscription already exists for this user and group")
	ErrSubscriptionAssignConflict = infraerrors.Conflict("SUBSCRIPTION_ASSIGN_CONFLICT", "subscription exists but request conflicts with existing assignment semantics")
	ErrGroupNotSubscriptionType   = infraerrors.BadRequest("GROUP_NOT_SUBSCRIPTION_TYPE", "group is not a subscription type")
	ErrSubscriptionNilInput       = infraerrors.BadRequest("SUBSCRIPTION_NIL_INPUT", "subscription input cannot be nil")
	ErrInvalidInput               = infraerrors.BadRequest("INVALID_INPUT", "invalid input")
	ErrDailyLimitExceeded         = infraerrors.TooManyRequests("DAILY_LIMIT_EXCEEDED", "daily usage limit exceeded")
	ErrWeeklyLimitExceeded        = infraerrors.TooManyRequests("WEEKLY_LIMIT_EXCEEDED", "weekly usage limit exceeded")
	ErrMonthlyLimitExceeded       = infraerrors.TooManyRequests("MONTHLY_LIMIT_EXCEEDED", "monthly usage limit exceeded")
)

// ── Client access context markers ──

const (
	ClientAccessLimitedKey                 = "client_access_limited"
	ClientAccessLimitedReasonKey           = "client_access_limited_reason"
	ClientAccessLimitedReasonIPRestriction = "api_key_ip_restriction"
)

func MarkClientAccessLimited(c *gin.Context, reason string) {
	if c == nil {
		return
	}
	c.Set(ClientAccessLimitedKey, true)
	if reason = strings.TrimSpace(reason); reason != "" {
		c.Set(ClientAccessLimitedReasonKey, reason)
	}
}
