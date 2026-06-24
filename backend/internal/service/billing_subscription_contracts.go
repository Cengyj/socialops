package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

// BillingCache describes the cache operations used by billing and subscription
// services.
type BillingCache interface {
	GetUserBalance(ctx context.Context, userID int64) (float64, error)
	SetUserBalance(ctx context.Context, userID int64, balance float64) error
	DeductUserBalance(ctx context.Context, userID int64, amount float64) error
	InvalidateUserBalance(ctx context.Context, userID int64) error

	GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error)
	SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error
	UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error
	InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error
}

// Subscription errors.
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
