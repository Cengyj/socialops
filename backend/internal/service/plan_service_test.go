package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPlanServiceGetUserActivePlanReturnsEffectivePlatform(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := client.User.Create().
		SetEmail("my-plan-platform@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	group := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SaveX(ctx)
	client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanName("Legacy X Starter").
		SetPlanPlatform("").
		SetStatus(SubscriptionStatusActive).
		SetStartsAt(time.Now().UTC().Add(-time.Hour)).
		SetExpiresAt(time.Now().UTC().Add(24 * time.Hour)).
		SaveX(ctx)

	plan, err := NewPlanService(client).GetUserActivePlan(ctx, user.ID)

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Equal(t, "x_twitter", plan.PlanPlatform)
}

func TestPlanServiceGetUserActivePlanReturnsEffectiveQuotaLimits(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := client.User.Create().
		SetEmail("my-plan-limits@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	dailyLimit := 10.0
	weeklyLimit := 50.0
	monthlyLimit := 120.0
	group := client.Group.Create().
		SetName("X Quota Pool").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(dailyLimit).
		SetWeeklyLimitUsd(weeklyLimit).
		SetMonthlyLimitUsd(monthlyLimit).
		SaveX(ctx)
	client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanName("Legacy X Quota").
		SetStatus(SubscriptionStatusActive).
		SetStartsAt(time.Now().UTC().Add(-time.Hour)).
		SetExpiresAt(time.Now().UTC().Add(24 * time.Hour)).
		SaveX(ctx)

	plan, err := NewPlanService(client).GetUserActivePlan(ctx, user.ID)

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.NotNil(t, plan.DailyLimitUSD)
	require.NotNil(t, plan.WeeklyLimitUSD)
	require.NotNil(t, plan.MonthlyLimitUSD)
	require.Equal(t, dailyLimit, *plan.DailyLimitUSD)
	require.Equal(t, weeklyLimit, *plan.WeeklyLimitUSD)
	require.Equal(t, monthlyLimit, *plan.MonthlyLimitUSD)
}
