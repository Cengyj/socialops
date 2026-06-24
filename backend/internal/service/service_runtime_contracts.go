package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/config"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

// NotificationEmailService coordinates notification email templates, delivery,
// recipient locale preferences, and unsubscribe state.
type NotificationEmailService struct {
	settingRepo  SettingRepository
	emailService *EmailService
}

// GroupRepository exposes subscription package and access-group persistence used
// by billing, settings, and admin workflows.
type GroupRepository interface {
	Create(ctx context.Context, group *Group) error
	GetByID(ctx context.Context, id int64) (*Group, error)
	GetByIDLite(ctx context.Context, id int64) (*Group, error)
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id int64) error
	DeleteCascade(ctx context.Context, id int64) ([]int64, error)
	List(ctx context.Context, params pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, subscriptionType, search string, isExclusive *bool) ([]Group, *pagination.PaginationResult, error)
	ListActive(ctx context.Context) ([]Group, error)
	ListActiveByPlatform(ctx context.Context, platform string) ([]Group, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	UpdateSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error
}

// SubscriptionService manages subscription assignment, renewal, quota windows,
// and cache invalidation for SocialOps task execution billing.
type SubscriptionService struct {
	groupRepo           GroupRepository
	userSubRepo         UserSubscriptionRepository
	billingCacheService *BillingCacheService
	entClient           *dbent.Client
}

func NewSubscriptionService(groupRepo GroupRepository, userSubRepo UserSubscriptionRepository, billingCacheService *BillingCacheService, entClient *dbent.Client, _ *config.Config) *SubscriptionService {
	return &SubscriptionService{
		groupRepo:           groupRepo,
		userSubRepo:         userSubRepo,
		billingCacheService: billingCacheService,
		entClient:           entClient,
	}
}

func (s *SubscriptionService) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	if s == nil || s.groupRepo == nil || s.userSubRepo == nil {
		return nil, false, ErrSubscriptionNotFound
	}
	resolvedInput, _, err := s.resolveAssignSubscriptionInput(ctx, input)
	if err != nil {
		return nil, false, err
	}

	existing, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, resolvedInput.UserID, resolvedInput.GroupID)
	if err != nil || existing == nil {
		sub, created, assignErr := s.assignSubscription(ctx, resolvedInput)
		if assignErr != nil {
			return nil, false, assignErr
		}
		return sub, !created, nil
	}

	now := time.Now()
	validityDays := normalizeAssignValidityDays(resolvedInput.ValidityDays)
	newExpiresAt := existing.ExpiresAt.AddDate(0, 0, validityDays)
	isExpired := !existing.ExpiresAt.After(now)
	if isExpired {
		newExpiresAt = now.AddDate(0, 0, validityDays)
	}

	renewed := *existing
	renewed.ExpiresAt = newExpiresAt
	renewed.Status = SubscriptionStatusActive
	renewed.PlanID = resolvedInput.PlanID
	renewed.PlanName = resolvedInput.PlanName
	renewed.PlanPlatform = resolvedInput.PlanPlatform
	renewed.DailyLimitUSD = resolvedInput.DailyLimitUSD
	renewed.WeeklyLimitUSD = resolvedInput.WeeklyLimitUSD
	renewed.MonthlyLimitUSD = resolvedInput.MonthlyLimitUSD
	renewed.Notes = appendSubscriptionNotes(existing.Notes, resolvedInput.Notes)
	renewed.UpdatedAt = now
	if isExpired {
		windowStart := startOfDay(now)
		renewed.StartsAt = now
		renewed.DailyWindowStart = &windowStart
		renewed.WeeklyWindowStart = &windowStart
		renewed.MonthlyWindowStart = &windowStart
		renewed.DailyUsageUSD = 0
		renewed.WeeklyUsageUSD = 0
		renewed.MonthlyUsageUSD = 0
	}
	if err := s.userSubRepo.Update(ctx, &renewed); err != nil {
		return nil, false, err
	}
	s.invalidateSubscriptionCaches(ctx, resolvedInput.UserID, resolvedInput.GroupID)

	sub, err := s.userSubRepo.GetByID(ctx, existing.ID)
	return sub, true, err
}
func (s *SubscriptionService) Stop() {}

// AssignSubscriptionInput describes an admin, auth, payment, or redeem-code
// subscription assignment request.
type AssignSubscriptionInput struct {
	Notes           string
	UserID          int64
	GroupID         int64
	ValidityDays    int
	AssignedBy      int64
	PlanID          *int64
	PlanName        string
	PlanPlatform    string
	DailyLimitUSD   *float64
	WeeklyLimitUSD  *float64
	MonthlyLimitUSD *float64
}

// DefaultSubscriptionSetting stores a default package assigned during auth and
// registration flows.
type DefaultSubscriptionSetting struct {
	PlanID       int64 `json:"plan_id,omitempty"`
	GroupID      int64 `json:"group_id,omitempty"`
	ValidityDays int   `json:"validity_days"`
}

// IdempotencyRecord stores a write-idempotency claim and replay state.
type IdempotencyRecord struct {
	ID                 int64
	Scope              string
	IdempotencyKeyHash string
	RequestFingerprint string
	Status             IdempotencyStatus
	ResponseStatus     *int
	ResponseBody       *string
	ErrorReason        *string
	LockedUntil        *time.Time
	ExpiresAt          time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IdempotencyStatus keeps idempotency repository status values as plain strings.
type IdempotencyStatus = string

// LoginAgreementDocument is a public login/register agreement document.
type LoginAgreementDocument struct {
	ContentMD string `json:"content_md"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
}

// Affiliate constants
const (
	AffiliateEnabledDefault    = false
	AffiliateRebateRateDefault = 20.0
	AffiliateRebateRateMin     = 0.0
	AffiliateRebateRateMax     = 100.0
)

// Role/Status constants (from deleted domain/constants.go)
const (
	RoleUser     = "user"
	RoleAdmin    = "admin"
	StatusActive = "active"
)

const (
	SubscriptionTypeStandard     = "standard"
	SubscriptionTypeSubscription = "subscription"
)

const (
	RedeemTypeInvitation = "invitation"
	StatusUsed           = "used"
	StatusUnused         = "unused"
)

const (
	DingTalkConnectSyntheticEmailDomain = "@dingtalk-connect.invalid"
	LinuxDoConnectSyntheticEmailDomain  = "@linuxdo-connect.invalid"
	OIDCConnectSyntheticEmailDomain     = "@oidc-connect.invalid"
	WeChatConnectSyntheticEmailDomain   = "@wechat-connect.invalid"
)

const thresholdTypePercentage = "percentage"
const NotificationEmailEventBalanceLow = "balance.low"

type NotificationEmailSendInput struct {
	RecipientEmail string
	RecipientName  string
	SourceType     string
	SourceID       string
	ReminderKey    string
	Locale         string
	Event          string
	UserID         int64
	Variables      map[string]string
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

const NotificationEmailEventAuthVerifyCode = "auth.verify_code"

const NotificationEmailEventAuthPasswordReset = "auth.password_reset"

const (
	RedeemTypeBalance                                 = "balance"
	RedeemTypeAffiliateBalance                        = "affiliate_balance"
	NotificationEmailEventBalanceRechargeSuccess      = "balance.recharge_success"
	NotificationEmailEventSubscriptionPurchaseSuccess = "subscription.purchase_success"
	NotificationEmailEventSubscriptionExpiryReminder  = "subscription.expiry_reminder"
)

func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
}

func (s *SubscriptionService) ExtendSubscription(ctx context.Context, subID int64, days int) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sub, err := s.userSubRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, err
	}
	newExpiresAt := sub.ExpiresAt.AddDate(0, 0, days)
	if !newExpiresAt.After(time.Now()) {
		return nil, ErrAdjustWouldExpire
	}
	if err := s.userSubRepo.ExtendExpiry(ctx, subID, newExpiresAt); err != nil {
		return nil, err
	}
	updated, err := s.userSubRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, err
	}
	s.invalidateSubscriptionCaches(ctx, updated.UserID, updated.GroupID)
	return updated, nil
}

var ErrAdjustWouldExpire = infraerrors.BadRequest("ADJUST_WOULD_EXPIRE", "adjustment would expire subscription")

func (s *SubscriptionService) RevokeSubscription(ctx context.Context, subID int64) error {
	if s == nil || s.userSubRepo == nil {
		return ErrSubscriptionNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sub, err := s.userSubRepo.GetByID(ctx, subID)
	if err != nil {
		return err
	}
	if err := s.userSubRepo.UpdateStatus(ctx, subID, SubscriptionStatusRevoked); err != nil {
		return err
	}
	s.invalidateSubscriptionCaches(ctx, sub.UserID, sub.GroupID)
	return nil
}

const PromoCodeStatusActive = "active"

const PromoCodeStatusDisabled = "disabled"
const StatusExpired = "expired"

const (
	StatusDisabled            = "disabled"
	RedeemTypeSubscription    = "subscription"
	RedeemTypeConcurrency     = "concurrency"
	SubscriptionStatusExpired = "expired"
	SubscriptionStatusRevoked = "revoked"
)

var ErrSubscriptionNotFound = infraerrors.NotFound("SUBSCRIPTION_NOT_FOUND", "subscription not found")

func (s *SubscriptionService) InvalidateSubCache(userID, groupID int64) {}

func (s *SubscriptionService) ValidateAndCheckLimits(sub *UserSubscription, group *Group) (bool, error) {
	if sub == nil {
		return false, ErrSubscriptionNotFound
	}
	if sub.IsExpired() || sub.Status != SubscriptionStatusActive {
		return false, ErrSubscriptionNotFound
	}
	needsMaintenance := sub.NeedsDailyReset() || sub.NeedsWeeklyReset() || sub.NeedsMonthlyReset()
	if group == nil {
		return needsMaintenance, nil
	}
	if !sub.CheckDailyLimit(group, 0) {
		return needsMaintenance, ErrDailyLimitExceeded
	}
	if !sub.CheckWeeklyLimit(group, 0) {
		return needsMaintenance, ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimit(group, 0) {
		return needsMaintenance, ErrMonthlyLimitExceeded
	}
	return needsMaintenance, nil
}

func (s *SubscriptionService) DoWindowMaintenance(sub *UserSubscription) {
	if s == nil || s.userSubRepo == nil || sub == nil {
		return
	}
	ctx := context.Background()
	now := time.Now()
	if sub.NeedsDailyReset() {
		_ = s.userSubRepo.ResetDailyUsage(ctx, sub.ID, now)
	}
	if sub.NeedsWeeklyReset() {
		_ = s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, now)
	}
	if sub.NeedsMonthlyReset() {
		_ = s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, now)
	}
}
