package service

import infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"

// Restored definitions for symbols whose original homes were removed during
// platform cleanup but are still referenced by preserved platform services.

// ── Group ──

var (
	ErrGroupNotFound = infraerrors.NotFound("GROUP_NOT_FOUND", "group not found")
	ErrGroupExists   = infraerrors.Conflict("GROUP_EXISTS", "group already exists")
)

// ── Affiliate rebate bounds ──

const (
	AffiliateRebateFreezeHoursDefault   = 0
	AffiliateRebateFreezeHoursMax       = 720
	AffiliateRebateDurationDaysDefault  = 0
	AffiliateRebateDurationDaysMax      = 3650
	AffiliateRebatePerInviteeCapDefault = 0.0
)

// ── Misc platform constants ──

const AdminAPIKeyPrefix = "admin-"

const SubscriptionStatusActive = "active"

const MaxValidityDays = 36500

const NotificationEmailEventNotificationEmailVerifyCode = "notification_email.verify_code"
