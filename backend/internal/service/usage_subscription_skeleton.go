package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
)

var ErrUsageLogNotFound = infraerrors.NotFound("USAGE_NOT_FOUND", "usage record not found")

// UsageLog is the SocialOps usage record shell. In the current recovery phase it
// represents social operation/task activity, not AI token billing.
type UsageLog struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	APIKeyID    *int64     `json:"api_key_id,omitempty"`
	GroupID     *int64     `json:"group_id,omitempty"`
	Operation   string     `json:"operation"`
	Status      string     `json:"status"`
	Quantity    int64      `json:"quantity"`
	Cost        float64    `json:"cost"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
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
}

func NewUsageService(repo UsageLogRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *UsageService {
	return &UsageService{repo: repo, authCacheInvalidator: authCacheInvalidator}
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
	return []UsageLog{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    1,
	}, nil
}

func (s *UsageService) Stats(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	if repo, ok := s.repo.(usageLogLister); ok {
		return repo.GetStatsWithFilters(ctx, filters)
	}
	return &usagestats.UsageStats{}, nil
}

func (s *UsageService) GetByID(ctx context.Context, id, userID int64) (*UsageLog, error) {
	if repo, ok := s.repo.(usageLogGetter); ok {
		return repo.GetByID(ctx, id, userID)
	}
	return nil, ErrUsageLogNotFound
}

func (s *UsageService) GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	if repo, ok := s.repo.(userDashboardStatsProvider); ok {
		return repo.GetUserDashboardStats(ctx, userID)
	}
	return &usagestats.UserDashboardStats{}, nil
}

func (s *UsageService) GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) ([]usagestats.TrendDataPoint, error) {
	if repo, ok := s.repo.(userUsageTrendProvider); ok {
		return repo.GetUserUsageTrendByUserID(ctx, userID, startTime, endTime, granularity)
	}
	return []usagestats.TrendDataPoint{}, nil
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
	return s.userSubRepo.List(ctx, pagination.PaginationParams{Page: page, PageSize: pageSize}, userID, groupID, status, platform, sortBy, sortOrder)
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
	if input == nil {
		return nil, false, ErrSubscriptionNilInput
	}
	if s.groupRepo != nil {
		group, err := s.groupRepo.GetByID(ctx, input.GroupID)
		if err != nil {
			return nil, false, err
		}
		if group != nil && group.SubscriptionType != "" && group.SubscriptionType != SubscriptionTypeSubscription {
			return nil, false, ErrGroupNotSubscriptionType
		}
	}
	if s.userSubRepo != nil {
		existing, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
		if err == nil && existing != nil {
			if _, conflict := detectAssignSemanticConflict(existing, input); conflict {
				return nil, false, ErrSubscriptionAssignConflict
			}
			return existing, false, nil
		}
	}

	now := time.Now().UTC()
	validityDays := normalizeAssignValidityDays(input.ValidityDays)
	sub := &UserSubscription{
		UserID:     input.UserID,
		GroupID:    input.GroupID,
		StartsAt:   now,
		ExpiresAt:  now.AddDate(0, 0, validityDays),
		Status:     SubscriptionStatusActive,
		AssignedBy: &input.AssignedBy,
		AssignedAt: now,
		Notes:      input.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if s.userSubRepo != nil {
		if err := s.userSubRepo.Create(ctx, sub); err != nil {
			return nil, false, err
		}
	}
	return sub, true, nil
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
	windowStart := startOfDay(time.Now())
	if err := s.userSubRepo.ResetDailyUsage(ctx, subscriptionID, windowStart); err != nil {
		return err
	}
	if err := s.userSubRepo.ResetWeeklyUsage(ctx, subscriptionID, windowStart); err != nil {
		return err
	}
	return s.userSubRepo.ResetMonthlyUsage(ctx, subscriptionID, windowStart)
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
	return s.userSubRepo.GetByID(ctx, subscriptionID)
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
		s.InvalidateSubCache(sub.UserID, sub.GroupID)
		if s.billingCacheService != nil {
			_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
		}
	}

	return nil
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
	return &usagestats.DashboardStats{}, nil
}

func (s *DashboardService) GetUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, requestType *int16, stream *bool) ([]usagestats.TrendDataPoint, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUsageTrendProvider); ok {
			return repo.GetUsageTrend(ctx, startTime, endTime, granularity, requestType, stream)
		}
	}
	return []usagestats.TrendDataPoint{}, nil
}

func (s *DashboardService) GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUserUsageTrendProvider); ok {
			return repo.GetUserUsageTrend(ctx, startTime, endTime, granularity, limit)
		}
	}
	return []usagestats.UserUsageTrendPoint{}, nil
}

func (s *DashboardService) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	if s != nil {
		if repo, ok := s.repo.(dashboardUserSpendingRankingProvider); ok {
			return repo.GetUserSpendingRanking(ctx, startTime, endTime, limit)
		}
	}
	return &usagestats.UserSpendingRankingResponse{Ranking: []usagestats.UserSpendingRankingItem{}}, nil
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
		GroupName:     group.Name,
		ExpiresAt:     sub.ExpiresAt,
		ExpiresInDays: sub.DaysRemaining(),
	}
	if group.HasDailyLimit() && sub.DailyWindowStart != nil {
		limit := *group.DailyLimitUSD
		resetsAt := sub.DailyWindowStart.Add(24 * time.Hour)
		if dailyResetTime := sub.DailyResetTime(); dailyResetTime != nil {
			resetsAt = *dailyResetTime
		}
		progress.Daily = usageWindowProgress(limit, sub.DailyUsageUSD, *sub.DailyWindowStart, resetsAt)
	}
	if group.HasWeeklyLimit() && sub.WeeklyWindowStart != nil {
		limit := *group.WeeklyLimitUSD
		progress.Weekly = usageWindowProgress(limit, sub.WeeklyUsageUSD, *sub.WeeklyWindowStart, sub.WeeklyWindowStart.Add(7*24*time.Hour))
	}
	if group.HasMonthlyLimit() && sub.MonthlyWindowStart != nil {
		limit := *group.MonthlyLimitUSD
		progress.Monthly = usageWindowProgress(limit, sub.MonthlyUsageUSD, *sub.MonthlyWindowStart, sub.MonthlyWindowStart.Add(30*24*time.Hour))
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
