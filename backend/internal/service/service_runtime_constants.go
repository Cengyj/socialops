package service

import infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"

// Shared service errors and constants used by SocialOps runtime services.

// Group errors.
var (
	ErrGroupNotFound = infraerrors.NotFound("GROUP_NOT_FOUND", "group not found")
	ErrGroupExists   = infraerrors.Conflict("GROUP_EXISTS", "group already exists")
)

// Affiliate rebate bounds.
const (
	AffiliateRebateFreezeHoursDefault   = 0
	AffiliateRebateFreezeHoursMax       = 720
	AffiliateRebateDurationDaysDefault  = 0
	AffiliateRebateDurationDaysMax      = 3650
	AffiliateRebatePerInviteeCapDefault = 0.0
)

// Shared runtime constants.
const AdminAPIKeyPrefix = "admin-"

const SubscriptionStatusActive = "active"

const MaxValidityDays = 36500

const NotificationEmailEventNotificationEmailVerifyCode = "notification_email.verify_code"
