package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/socialops/internal/domain"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
)

var ErrUsageLogNotFound = infraerrors.NotFound("USAGE_NOT_FOUND", "usage record not found")
var ErrUsageServiceUnavailable = infraerrors.ServiceUnavailable("USAGE_SERVICE_UNAVAILABLE", "usage service is unavailable")

const (
	usageTaskMediaScopePayload  = "payload"
	usageTaskMediaScopeTemplate = "template"

	usageTaskMediaSectionPost   = "post"
	usageTaskMediaSectionAvatar = "avatar"
	usageTaskMediaSectionBanner = "banner"
)

// UsageLog is the SocialOps usage record shell. In the current recovery phase it
// represents social operation/task activity, not AI token billing.
type UsageLog struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	APIKeyID        *int64     `json:"api_key_id,omitempty"`
	GroupID         *int64     `json:"group_id,omitempty"`
	SocialAccountID int64      `json:"social_account_id"`
	Platform        string     `json:"platform"`
	AccountName     string     `json:"account_name"`
	Operation       string     `json:"operation"`
	Status          string     `json:"status"`
	Quantity        int64      `json:"quantity"`
	Cost            float64    `json:"cost"`
	ChargeStatus    string     `json:"charge_status"`
	ChargeSource    *string    `json:"charge_source,omitempty"`
	Target          *string    `json:"target,omitempty"`
	Content         *string    `json:"content,omitempty"`
	Payload         *domain.SocialTaskPayload          `json:"payload,omitempty"`
	TemplateSnapshot *domain.SocialTaskTemplateSnapshot `json:"template_snapshot,omitempty"`
	ResultMessage   *string    `json:"result_message,omitempty"`
	ProxySnapshot   *string    `json:"proxy_snapshot,omitempty"`
	BillingRequestID *string   `json:"billing_request_id,omitempty"`
	IdempotencyKey  *string    `json:"idempotency_key,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type UsageTaskMediaLocator struct {
	Scope   string `json:"scope"`
	Section string `json:"section"`
	Index   int    `json:"index,omitempty"`
}

type usageLogLister interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error)
	GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error)
}

type usageLogGetter interface {
	GetByID(ctx context.Context, id, userID int64) (*UsageLog, error)
}

type userDashboardStatsProvider interface {
	GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error)
}

type userUsageTrendProvider interface {
	GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) ([]usagestats.TrendDataPoint, error)
}

type dashboardStatsProvider interface {
	GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error)
}

type dashboardUsageTrendProvider interface {
	GetUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, requestType *int16, stream *bool) ([]usagestats.TrendDataPoint, error)
}

type dashboardUserUsageTrendProvider interface {
	GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error)
}

type dashboardUserSpendingRankingProvider interface {
	GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error)
}

type UsageLogRepository interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error)
	GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error)
	GetByID(ctx context.Context, id, userID int64) (*UsageLog, error)
}

type UsageService struct {
	repo                 UsageLogRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
	mediaResolver        SocialTaskMediaResolver
}

func NewUsageService(repo UsageLogRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *UsageService {
	return &UsageService{repo: repo, authCacheInvalidator: authCacheInvalidator}
}

func (s *UsageService) WithMediaResolver(resolver SocialTaskMediaResolver) *UsageService {
	if s == nil {
		return nil
	}
	s.mediaResolver = resolver
	return s
}

func (s *UsageService) invalidateUsageCaches(ctx context.Context, userID int64, balanceUpdated bool) {
	if !balanceUpdated || s.authCacheInvalidator == nil {
		return
	}
	s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
}

var _ = (*UsageService).invalidateUsageCaches

func (s *UsageService) List(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	if repo, ok := s.repo.(usageLogLister); ok {
		return repo.ListWithFilters(ctx, params, filters)
	}
	return nil, nil, ErrUsageServiceUnavailable
}

func (s *UsageService) Stats(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	if repo, ok := s.repo.(usageLogLister); ok {
		return repo.GetStatsWithFilters(ctx, filters)
	}
	return nil, ErrUsageServiceUnavailable
}

func (s *UsageService) GetByID(ctx context.Context, id, userID int64) (*UsageLog, error) {
	if repo, ok := s.repo.(usageLogGetter); ok {
		return repo.GetByID(ctx, id, userID)
	}
	return nil, ErrUsageLogNotFound
}

func (s *UsageService) PreviewTaskMedia(ctx context.Context, id, userID int64, locator UsageTaskMediaLocator) (*ResolvedSocialTaskMedia, error) {
	if s == nil || s.mediaResolver == nil {
		return nil, infraerrors.ServiceUnavailable("USAGE_TASK_MEDIA_SERVICE_UNAVAILABLE", "usage task media service is unavailable")
	}
	item, err := s.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	ref, err := locateUsageTaskMediaRef(item, locator)
	if err != nil {
		return nil, err
	}
	if !socialTaskMediaRefExecutableStored(ref) {
		return nil, infraerrors.BadRequest("USAGE_TASK_MEDIA_SOURCE_UNSUPPORTED", "usage task media source is not supported")
	}
	resolved, err := s.mediaResolver.Resolve(ctx, userID, ref)
	if err != nil {
		switch {
		case errors.Is(err, errSocialTaskMediaAssetUnavailable):
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		case errors.Is(err, errSocialTaskMediaAssetInvalid):
			return nil, infraerrors.BadRequest("USAGE_TASK_MEDIA_INVALID", "usage task media is invalid")
		default:
			return nil, err
		}
	}
	return resolved, nil
}

func (s *UsageService) GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	if repo, ok := s.repo.(userDashboardStatsProvider); ok {
		return repo.GetUserDashboardStats(ctx, userID)
	}
	return nil, ErrUsageServiceUnavailable
}

func (s *UsageService) GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	if repo, ok := s.repo.(userUsageTrendProvider); ok {
		return repo.GetUserUsageTrendByUserID(ctx, userID, startTime, endTime, granularity)
	}
	return nil, ErrUsageServiceUnavailable
}

func locateUsageTaskMediaRef(item *UsageLog, locator UsageTaskMediaLocator) (*domain.SocialTaskMediaRef, error) {
	if item == nil {
		return nil, ErrUsageLogNotFound
	}
	scope := strings.ToLower(strings.TrimSpace(locator.Scope))
	section := strings.ToLower(strings.TrimSpace(locator.Section))
	switch scope {
	case usageTaskMediaScopePayload:
		if item.Payload == nil || item.Payload.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return locateUsageTaskPayloadMediaRef(item.Payload, section, locator.Index)
	case usageTaskMediaScopeTemplate:
		if item.TemplateSnapshot == nil || item.TemplateSnapshot.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return locateUsageTaskTemplateMediaRef(item.TemplateSnapshot, section, locator.Index)
	default:
		return nil, infraerrors.BadRequest("USAGE_TASK_MEDIA_LOCATOR_INVALID", "usage task media locator is invalid")
	}
}

func locateUsageTaskPayloadMediaRef(payload *domain.SocialTaskPayload, section string, index int) (*domain.SocialTaskMediaRef, error) {
	if payload == nil {
		return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
	}
	switch section {
	case usageTaskMediaSectionPost:
		if payload.Post == nil || index < 0 || index >= len(payload.Post.Media) {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		ref := payload.Post.Media[index]
		if ref.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return &ref, nil
	case usageTaskMediaSectionAvatar:
		if payload.Avatar == nil || payload.Avatar.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return payload.Avatar, nil
	case usageTaskMediaSectionBanner:
		if payload.Banner == nil || payload.Banner.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return payload.Banner, nil
	default:
		return nil, infraerrors.BadRequest("USAGE_TASK_MEDIA_LOCATOR_INVALID", "usage task media locator is invalid")
	}
}

func locateUsageTaskTemplateMediaRef(snapshot *domain.SocialTaskTemplateSnapshot, section string, index int) (*domain.SocialTaskMediaRef, error) {
	if snapshot == nil {
		return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
	}
	switch section {
	case usageTaskMediaSectionPost:
		if index < 0 || index >= len(snapshot.Params.Media) {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		ref := snapshot.Params.Media[index]
		if ref.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return &ref, nil
	case usageTaskMediaSectionAvatar:
		if snapshot.Params.Avatar == nil || snapshot.Params.Avatar.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return snapshot.Params.Avatar, nil
	case usageTaskMediaSectionBanner:
		if snapshot.Params.Banner == nil || snapshot.Params.Banner.IsZero() {
			return nil, infraerrors.NotFound("USAGE_TASK_MEDIA_NOT_FOUND", "usage task media is not available")
		}
		return snapshot.Params.Banner, nil
	default:
		return nil, infraerrors.BadRequest("USAGE_TASK_MEDIA_LOCATOR_INVALID", "usage task media locator is invalid")
	}
}

func (s *UsageService) GetUserAPIKeysUsage(context.Context, int64, []int64, time.Time, time.Time) ([]usagestats.APIKeyUsageTrendPoint, error) {
	return []usagestats.APIKeyUsageTrendPoint{}, nil
}

type SubscriptionProgress struct {
	ID            int64                `json:"id"`
	GroupName     string               `json:"group_name"`
	ExpiresAt     time.Time            `json:"expires_at"`
	ExpiresInDays int                  `json:"expires_in_days"`
	Daily         *UsageWindowProgress `json:"daily,omitempty"`
	Weekly        *UsageWindowProgress `json:"weekly,omitempty"`
	Monthly       *UsageWindowProgress `json:"monthly,omitempty"`
}

type UsageWindowProgress struct {
	LimitUSD        float64   `json:"limit_usd"`
	UsedUSD         float64   `json:"used_usd"`
	RemainingUSD    float64   `json:"remaining_usd"`
	Percentage      float64   `json:"percentage"`
	WindowStart     time.Time `json:"window_start"`
	ResetsAt        time.Time `json:"resets_at"`
	ResetsInSeconds int64     `json:"resets_in_seconds"`
}

type BulkAssignSubscriptionInput struct {
	UserIDs      []int64
	GroupID      int64
	PlanID       *int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

type BulkAssignResult struct {
	SuccessCount  int
	CreatedCount  int
	ReusedCount   int
	FailedCount   int
	Subscriptions []UserSubscription
	Errors        []string
	Statuses      map[int64]string
}

func (s *SubscriptionService) ListUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.ListByUserID(ctx, userID)
}

func (s *SubscriptionService) ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.ListActiveByUserID(ctx, userID)
}

func (s *SubscriptionService) List(ctx context.Context, page, pageSize int, userID *int64, groupID *int64, status string, platform string, sortBy string, sortOrder string) ([]UserSubscription, *pagination.PaginationResult, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.List(ctx, pagination.PaginationParams{Page: page, PageSize: pageSize}, userID, groupID, nil, status, platform, sortBy, sortOrder)
}

func (s *SubscriptionService) ListWithPlan(ctx context.Context, page, pageSize int, userID *int64, groupID *int64, planID *int64, status string, platform string, sortBy string, sortOrder string) ([]UserSubscription, *pagination.PaginationResult, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.List(ctx, pagination.PaginationParams{Page: page, PageSize: pageSize}, userID, groupID, planID, status, platform, sortBy, sortOrder)
}

func (s *SubscriptionService) ListByUser(ctx context.Context, userID int64) ([]UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.ListByUserID(ctx, userID)
}

func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.GetByID(ctx, id)
}

func (s *SubscriptionService) GetSubscriptionProgress(ctx context.Context, id int64) (*SubscriptionProgress, error) {
	sub, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub.Group != nil {
		return s.calculateProgress(sub, sub.Group), nil
	}
	progress := &SubscriptionProgress{
		ID:            sub.ID,
		ExpiresAt:     sub.ExpiresAt,
		ExpiresInDays: sub.DaysRemaining(),
	}
	return progress, nil
}

func (s *SubscriptionService) AssignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	sub, _, err := s.assignSubscription(ctx, input)
	return sub, err
}

func (s *SubscriptionService) assignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	resolvedInput, _, err := s.resolveAssignSubscriptionInput(ctx, input)
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()
	if s.userSubRepo != nil {
		existing, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, resolvedInput.UserID, resolvedInput.GroupID)
		if err == nil && existing != nil {
			if reusableActiveAssignment(existing, now) {
				if _, conflict := detectAssignSemanticConflict(existing, resolvedInput); conflict {
					return nil, false, ErrSubscriptionAssignConflict
				}
				return existing, false, nil
			}
			renewed := renewSubscriptionAssignment(existing, resolvedInput, now)
			if err := s.userSubRepo.Update(ctx, renewed); err != nil {
				return nil, false, err
			}
			s.invalidateSubscriptionCaches(ctx, resolvedInput.UserID, resolvedInput.GroupID)
			sub, err := s.userSubRepo.GetByID(ctx, existing.ID)
			return sub, false, err
		}
	}

	validityDays := normalizeAssignValidityDays(resolvedInput.ValidityDays)
	sub := &UserSubscription{
		UserID:          resolvedInput.UserID,
		GroupID:         resolvedInput.GroupID,
		PlanID:          resolvedInput.PlanID,
		PlanName:        resolvedInput.PlanName,
		PlanPlatform:    resolvedInput.PlanPlatform,
		DailyLimitUSD:   resolvedInput.DailyLimitUSD,
		WeeklyLimitUSD:  resolvedInput.WeeklyLimitUSD,
		MonthlyLimitUSD: resolvedInput.MonthlyLimitUSD,
		StartsAt:        now,
		ExpiresAt:       now.AddDate(0, 0, validityDays),
		Status:          SubscriptionStatusActive,
		AssignedBy:      &resolvedInput.AssignedBy,
		AssignedAt:      now,
		Notes:           resolvedInput.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if s.userSubRepo != nil {
		if err := s.userSubRepo.Create(ctx, sub); err != nil {
			return nil, false, err
		}
	}
	return sub, true, nil
}

func reusableActiveAssignment(existing *UserSubscription, now time.Time) bool {
	return existing != nil && existing.Status == SubscriptionStatusActive && existing.ExpiresAt.After(now)
}

func renewSubscriptionAssignment(existing *UserSubscription, input *AssignSubscriptionInput, now time.Time) *UserSubscription {
	renewed := *existing
	validityDays := normalizeAssignValidityDays(input.ValidityDays)
	windowStart := startOfDay(now)
	assignedBy := input.AssignedBy

	renewed.PlanID = input.PlanID
	renewed.PlanName = input.PlanName
	renewed.PlanPlatform = input.PlanPlatform
	renewed.DailyLimitUSD = input.DailyLimitUSD
	renewed.WeeklyLimitUSD = input.WeeklyLimitUSD
	renewed.MonthlyLimitUSD = input.MonthlyLimitUSD
	renewed.StartsAt = now
	renewed.ExpiresAt = now.AddDate(0, 0, validityDays)
	renewed.Status = SubscriptionStatusActive
	renewed.AssignedBy = &assignedBy
	renewed.AssignedAt = now
	renewed.Notes = input.Notes
	renewed.UpdatedAt = now
	renewed.DailyWindowStart = &windowStart
	renewed.WeeklyWindowStart = &windowStart
	renewed.MonthlyWindowStart = &windowStart
	renewed.DailyUsageUSD = 0
	renewed.WeeklyUsageUSD = 0
	renewed.MonthlyUsageUSD = 0
	return &renewed
}

func (s *SubscriptionService) BulkAssignSubscription(ctx context.Context, input *BulkAssignSubscriptionInput) (*BulkAssignResult, error) {
	result := &BulkAssignResult{
		Subscriptions: []UserSubscription{},
		Errors:        []string{},
		Statuses:      map[int64]string{},
	}
	for _, userID := range input.UserIDs {
		sub, created, err := s.assignSubscription(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      input.GroupID,
			PlanID:       input.PlanID,
			ValidityDays: input.ValidityDays,
			AssignedBy:   input.AssignedBy,
			Notes:        input.Notes,
		})
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
			result.Statuses[userID] = "failed"
			continue
		}
		result.SuccessCount++
		if created {
			result.CreatedCount++
			result.Statuses[userID] = "created"
		} else {
			result.ReusedCount++
			result.Statuses[userID] = "reused"
		}
		result.Subscriptions = append(result.Subscriptions, userSubscriptionDTOFromService(sub))
	}
	return result, nil
}

func (s *SubscriptionService) ResetSubscriptionQuota(ctx context.Context, subscriptionID int64) error {
	if s == nil || s.userSubRepo == nil {
		return ErrSubscriptionNotFound
	}
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}
	windowStart := startOfDay(time.Now())
	if err := s.userSubRepo.ResetDailyUsage(ctx, subscriptionID, windowStart); err != nil {
		return err
	}
	if err := s.userSubRepo.ResetWeeklyUsage(ctx, subscriptionID, windowStart); err != nil {
		return err
	}
	if err := s.userSubRepo.ResetMonthlyUsage(ctx, subscriptionID, windowStart); err != nil {
		return err
	}
	s.invalidateSubscriptionCaches(ctx, sub.UserID, sub.GroupID)
	return nil
}

func (s *SubscriptionService) AdminResetQuota(ctx context.Context, subscriptionID int64, resetDaily, resetWeekly, resetMonthly bool) (*UserSubscription, error) {
	if !resetDaily && !resetWeekly && !resetMonthly {
		return nil, ErrInvalidInput
	}
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	windowStart := startOfDay(time.Now())
	if resetDaily {
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	if resetWeekly {
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	if resetMonthly {
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	refreshed, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	s.invalidateSubscriptionCaches(ctx, refreshed.UserID, refreshed.GroupID)
	return refreshed, nil
}

func (s *SubscriptionService) CheckAndResetWindows(ctx context.Context, sub *UserSubscription) error {
	if s == nil || sub == nil {
		return ErrSubscriptionNotFound
	}
	windowStart := startOfDay(time.Now())
	needsInvalidateCache := false

	if sub.NeedsDailyReset() {
		if s.userSubRepo == nil {
			return ErrSubscriptionNotFound
		}
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.DailyWindowStart = &windowStart
		sub.DailyUsageUSD = 0
		needsInvalidateCache = true
	}

	if sub.NeedsWeeklyReset() {
		if s.userSubRepo == nil {
			return ErrSubscriptionNotFound
		}
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.WeeklyWindowStart = &windowStart
		sub.WeeklyUsageUSD = 0
		needsInvalidateCache = true
	}

	if sub.NeedsMonthlyReset() {
		if s.userSubRepo == nil {
			return ErrSubscriptionNotFound
		}
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.MonthlyWindowStart = &windowStart
		sub.MonthlyUsageUSD = 0
		needsInvalidateCache = true
	}

	if needsInvalidateCache {
		s.invalidateSubscriptionCaches(ctx, sub.UserID, sub.GroupID)
	}

	return nil
}

func (s *SubscriptionService) invalidateSubscriptionCaches(ctx context.Context, userID, groupID int64) {
	if s == nil {
		return
	}
	s.InvalidateSubCache(userID, groupID)
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, userID, groupID)
	}
}

func userSubscriptionDTOFromService(sub *UserSubscription) UserSubscription {
	if sub == nil {
		return UserSubscription{}
	}
	return *sub
}

func (s *DashboardService) GetStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardStatsProvider); ok {
			return repo.GetDashboardStats(ctx)
		}
	}
	return nil, ErrUsageServiceUnavailable
}

func (s *DashboardService) GetUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, requestType *int16, stream *bool) ([]usagestats.TrendDataPoint, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUsageTrendProvider); ok {
			return repo.GetUsageTrend(ctx, startTime, endTime, granularity, requestType, stream)
		}
	}
	return nil, ErrUsageServiceUnavailable
}

func (s *DashboardService) GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUserUsageTrendProvider); ok {
			return repo.GetUserUsageTrend(ctx, startTime, endTime, granularity, limit)
		}
	}
	return nil, ErrUsageServiceUnavailable
}

func (s *DashboardService) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUserSpendingRankingProvider); ok {
			return repo.GetUserSpendingRanking(ctx, startTime, endTime, limit)
		}
	}
	return nil, ErrUsageServiceUnavailable
}

func normalizeAssignValidityDays(days int) int {
	if days <= 0 {
		return 30
	}
	if days > MaxValidityDays {
		return MaxValidityDays
	}
	return days
}

func detectAssignSemanticConflict(existing *UserSubscription, input *AssignSubscriptionInput) (string, bool) {
	if existing == nil || input == nil {
		return "", false
	}
	if !nullableInt64Equal(existing.PlanID, input.PlanID) {
		return "plan_mismatch", true
	}
	if strings.TrimSpace(existing.PlanName) != strings.TrimSpace(input.PlanName) {
		return "plan_name_mismatch", true
	}
	if strings.TrimSpace(existing.PlanPlatform) != strings.TrimSpace(input.PlanPlatform) {
		return "plan_platform_mismatch", true
	}
	if !nullableFloat64Equal(existing.DailyLimitUSD, input.DailyLimitUSD) {
		return "daily_limit_mismatch", true
	}
	if !nullableFloat64Equal(existing.WeeklyLimitUSD, input.WeeklyLimitUSD) {
		return "weekly_limit_mismatch", true
	}
	if !nullableFloat64Equal(existing.MonthlyLimitUSD, input.MonthlyLimitUSD) {
		return "monthly_limit_mismatch", true
	}
	normalizedDays := normalizeAssignValidityDays(input.ValidityDays)
	if !existing.StartsAt.IsZero() {
		expectedExpiresAt := existing.StartsAt.AddDate(0, 0, normalizedDays)
		if !existing.ExpiresAt.Equal(expectedExpiresAt) {
			return "validity_days_mismatch", true
		}
	}
	if strings.TrimSpace(existing.Notes) != strings.TrimSpace(input.Notes) {
		return "notes_mismatch", true
	}
	return "", false
}

func appendSubscriptionNotes(existingNotes, newNotes string) string {
	if strings.TrimSpace(newNotes) == "" {
		return existingNotes
	}
	if strings.TrimSpace(existingNotes) == "" {
		return newNotes
	}
	return existingNotes + "\n" + newNotes
}

func (s *SubscriptionService) calculateProgress(sub *UserSubscription, group *Group) *SubscriptionProgress {
	progress := &SubscriptionProgress{
		ID:            sub.ID,
		GroupName:     sub.EffectiveDisplayName(group),
		ExpiresAt:     sub.ExpiresAt,
		ExpiresInDays: sub.DaysRemaining(),
	}
	if limit := sub.EffectiveDailyLimitUSD(group); limit != nil && *limit > 0 && sub.DailyWindowStart != nil {
		resetsAt := sub.DailyWindowStart.Add(24 * time.Hour)
		if dailyResetTime := sub.DailyResetTime(); dailyResetTime != nil {
			resetsAt = *dailyResetTime
		}
		progress.Daily = usageWindowProgress(*limit, sub.DailyUsageUSD, *sub.DailyWindowStart, resetsAt)
	}
	if limit := sub.EffectiveWeeklyLimitUSD(group); limit != nil && *limit > 0 && sub.WeeklyWindowStart != nil {
		progress.Weekly = usageWindowProgress(*limit, sub.WeeklyUsageUSD, *sub.WeeklyWindowStart, sub.WeeklyWindowStart.Add(7*24*time.Hour))
	}
	if limit := sub.EffectiveMonthlyLimitUSD(group); limit != nil && *limit > 0 && sub.MonthlyWindowStart != nil {
		progress.Monthly = usageWindowProgress(*limit, sub.MonthlyUsageUSD, *sub.MonthlyWindowStart, sub.MonthlyWindowStart.Add(30*24*time.Hour))
	}
	return progress
}

func usageWindowProgress(limit, used float64, windowStart, resetsAt time.Time) *UsageWindowProgress {
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	percentage := 0.0
	if limit > 0 {
		percentage = (used / limit) * 100
	}
	if percentage > 100 {
		percentage = 100
	}
	resetsIn := int64(time.Until(resetsAt).Seconds())
	if resetsIn < 0 {
		resetsIn = 0
	}
	return &UsageWindowProgress{
		LimitUSD:        limit,
		UsedUSD:         used,
		RemainingUSD:    remaining,
		Percentage:      percentage,
		WindowStart:     windowStart,
		ResetsAt:        resetsAt,
		ResetsInSeconds: resetsIn,
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
