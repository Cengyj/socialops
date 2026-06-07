//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// resetQuotaUserSubRepoStub 支持 GetByID、ResetDailyUsage、ResetWeeklyUsage、ResetMonthlyUsage，
// 其余方法继承 userSubRepoNoop（panic）。
type resetQuotaUserSubRepoStub struct {
	userSubRepoNoop

	sub *UserSubscription

	resetDailyCalled   bool
	resetWeeklyCalled  bool
	resetMonthlyCalled bool
	resetDailyErr      error
	resetWeeklyErr     error
	resetMonthlyErr    error
}

type subscriptionCacheInvalidationRecorder struct {
	billingCacheWorkerStub

	invalidatedSubscriptions []subscriptionCacheInvalidation
}

type subscriptionCacheInvalidation struct {
	userID  int64
	groupID int64
}

func (r *subscriptionCacheInvalidationRecorder) InvalidateSubscriptionCache(_ context.Context, userID, groupID int64) error {
	r.invalidatedSubscriptions = append(r.invalidatedSubscriptions, subscriptionCacheInvalidation{
		userID:  userID,
		groupID: groupID,
	})
	return nil
}

func (r *resetQuotaUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *resetQuotaUserSubRepoStub) ResetDailyUsage(_ context.Context, _ int64, windowStart time.Time) error {
	r.resetDailyCalled = true
	if r.resetDailyErr == nil && r.sub != nil {
		r.sub.DailyUsageUSD = 0
		r.sub.DailyWindowStart = &windowStart
	}
	return r.resetDailyErr
}

func (r *resetQuotaUserSubRepoStub) ResetWeeklyUsage(_ context.Context, _ int64, _ time.Time) error {
	r.resetWeeklyCalled = true
	return r.resetWeeklyErr
}

func (r *resetQuotaUserSubRepoStub) ResetMonthlyUsage(_ context.Context, _ int64, _ time.Time) error {
	r.resetMonthlyCalled = true
	return r.resetMonthlyErr
}

func (r *resetQuotaUserSubRepoStub) ExtendExpiry(_ context.Context, _ int64, expiresAt time.Time) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.ExpiresAt = expiresAt
	return nil
}

func (r *resetQuotaUserSubRepoStub) UpdateStatus(_ context.Context, _ int64, status string) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.Status = status
	return nil
}

func newResetQuotaSvc(stub *resetQuotaUserSubRepoStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func newResetQuotaSvcWithCache(stub *resetQuotaUserSubRepoStub, cache BillingCache) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, &BillingCacheService{cache: cache}, nil, nil)
}

func TestAdminResetQuota_ResetBoth(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 1, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 1, true, true, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.resetDailyCalled, "应调用 ResetDailyUsage")
	require.True(t, stub.resetWeeklyCalled, "应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_ResetDailyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 2, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 2, true, false, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.resetDailyCalled, "应调用 ResetDailyUsage")
	require.False(t, stub.resetWeeklyCalled, "不应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_ResetWeeklyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 3, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 3, false, true, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, stub.resetDailyCalled, "不应调用 ResetDailyUsage")
	require.True(t, stub.resetWeeklyCalled, "应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_BothFalseReturnsError(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 7, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 7, false, false, false)

	require.ErrorIs(t, err, ErrInvalidInput)
	require.False(t, stub.resetDailyCalled)
	require.False(t, stub.resetWeeklyCalled)
	require.False(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_SubscriptionNotFound(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{sub: nil}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 999, true, true, true)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.resetDailyCalled)
	require.False(t, stub.resetWeeklyCalled)
	require.False(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_ResetDailyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:           &UserSubscription{ID: 4, UserID: 10, GroupID: 20},
		resetDailyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 4, true, true, false)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetDailyCalled)
	require.False(t, stub.resetWeeklyCalled, "daily 失败后不应继续调用 weekly")
}

func TestAdminResetQuota_ResetWeeklyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:            &UserSubscription{ID: 5, UserID: 10, GroupID: 20},
		resetWeeklyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 5, false, true, false)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetWeeklyCalled)
}

func TestAdminResetQuota_ResetMonthlyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 8, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 8, false, false, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, stub.resetDailyCalled, "不应调用 ResetDailyUsage")
	require.False(t, stub.resetWeeklyCalled, "不应调用 ResetWeeklyUsage")
	require.True(t, stub.resetMonthlyCalled, "应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_ResetMonthlyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:             &UserSubscription{ID: 9, UserID: 10, GroupID: 20},
		resetMonthlyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 9, false, false, true)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_ReturnsRefreshedSub(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID:            6,
			UserID:        10,
			GroupID:       20,
			DailyUsageUSD: 99.9,
		},
	}

	svc := newResetQuotaSvc(stub)
	result, err := svc.AdminResetQuota(context.Background(), 6, true, false, false)

	require.NoError(t, err)
	// ResetDailyUsage stub 会将 sub.DailyUsageUSD 归零，
	// 服务应返回第二次 GetByID 的刷新值而非初始的 99.9
	require.Equal(t, float64(0), result.DailyUsageUSD, "返回的订阅应反映已归零的用量")
	require.True(t, stub.resetDailyCalled)
}

func TestAdminResetQuotaInvalidatesBillingCacheAfterSuccessfulReset(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 10, UserID: 101, GroupID: 202},
	}
	cache := &subscriptionCacheInvalidationRecorder{}
	svc := newResetQuotaSvcWithCache(stub, cache)

	_, err := svc.AdminResetQuota(context.Background(), 10, true, false, false)

	require.NoError(t, err)
	require.Equal(t, []subscriptionCacheInvalidation{{userID: 101, groupID: 202}}, cache.invalidatedSubscriptions)
}

func TestResetSubscriptionQuotaInvalidatesBillingCacheAfterSuccessfulReset(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 11, UserID: 111, GroupID: 222},
	}
	cache := &subscriptionCacheInvalidationRecorder{}
	svc := newResetQuotaSvcWithCache(stub, cache)

	err := svc.ResetSubscriptionQuota(context.Background(), 11)

	require.NoError(t, err)
	require.Equal(t, []subscriptionCacheInvalidation{{userID: 111, groupID: 222}}, cache.invalidatedSubscriptions)
}

func TestExtendSubscriptionInvalidatesBillingCacheAfterSuccessfulAdjustment(t *testing.T) {
	now := time.Now().UTC()
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID:        12,
			UserID:    121,
			GroupID:   242,
			StartsAt:  now.Add(-time.Hour),
			ExpiresAt: now.Add(24 * time.Hour),
			Status:    SubscriptionStatusActive,
		},
	}
	cache := &subscriptionCacheInvalidationRecorder{}
	svc := newResetQuotaSvcWithCache(stub, cache)

	_, err := svc.ExtendSubscription(context.Background(), 12, 1)

	require.NoError(t, err)
	require.Equal(t, []subscriptionCacheInvalidation{{userID: 121, groupID: 242}}, cache.invalidatedSubscriptions)
}

func TestRevokeSubscriptionInvalidatesBillingCacheAfterSuccessfulStatusUpdate(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID:      13,
			UserID:  131,
			GroupID: 262,
			Status:  SubscriptionStatusActive,
		},
	}
	cache := &subscriptionCacheInvalidationRecorder{}
	svc := newResetQuotaSvcWithCache(stub, cache)

	err := svc.RevokeSubscription(context.Background(), 13)

	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusRevoked, stub.sub.Status)
	require.Equal(t, []subscriptionCacheInvalidation{{userID: 131, groupID: 262}}, cache.invalidatedSubscriptions)
}
