package service

import (
	"context"
	"math"
	"strings"

	dbent "github.com/Wei-Shaw/socialops/ent"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

func (s *SubscriptionService) resolveAssignSubscriptionInput(ctx context.Context, input *AssignSubscriptionInput) (*AssignSubscriptionInput, *Group, error) {
	if input == nil {
		return nil, nil, ErrSubscriptionNilInput
	}

	resolved := *input
	if resolved.GroupID <= 0 && (resolved.PlanID == nil || *resolved.PlanID <= 0) {
		return nil, nil, infraerrors.BadRequest("SUBSCRIPTION_TARGET_REQUIRED", "plan_id or group_id is required")
	}

	var plan *dbent.SubscriptionPlan
	if resolved.PlanID != nil && *resolved.PlanID > 0 {
		if s == nil || s.entClient == nil {
			return nil, nil, infraerrors.InternalServer("PLAN_CATALOG_UNAVAILABLE", "subscription plan catalog is unavailable")
		}
		loadedPlan, err := s.entClient.SubscriptionPlan.Get(ctx, *resolved.PlanID)
		if err != nil {
			return nil, nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
		}
		plan = loadedPlan
		resolved.GroupID = loadedPlan.GroupID
	}

	var group *Group
	if s != nil && s.groupRepo != nil && resolved.GroupID > 0 {
		loadedGroup, err := s.groupRepo.GetByID(ctx, resolved.GroupID)
		if err != nil {
			return nil, nil, err
		}
		if loadedGroup != nil && loadedGroup.SubscriptionType != "" && loadedGroup.SubscriptionType != SubscriptionTypeSubscription {
			return nil, nil, ErrGroupNotSubscriptionType
		}
		group = loadedGroup
	}

	if plan != nil {
		resolved.PlanName = firstNonEmpty(strings.TrimSpace(resolved.PlanName), strings.TrimSpace(plan.Name))
		resolved.PlanPlatform = firstNonEmpty(
			strings.TrimSpace(resolved.PlanPlatform),
			strings.TrimSpace(plan.Platform),
			groupPlatformValue(group),
		)
		resolved.DailyLimitUSD = firstNonNilAssignmentFloat64(resolved.DailyLimitUSD, plan.DailyLimitUsd, groupDailyLimitValue(group))
		resolved.WeeklyLimitUSD = firstNonNilAssignmentFloat64(resolved.WeeklyLimitUSD, plan.WeeklyLimitUsd, groupWeeklyLimitValue(group))
		resolved.MonthlyLimitUSD = firstNonNilAssignmentFloat64(resolved.MonthlyLimitUSD, plan.MonthlyLimitUsd, groupMonthlyLimitValue(group))
	}

	return &resolved, group, nil
}

func firstNonNilAssignmentFloat64(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func groupPlatformValue(group *Group) string {
	if group == nil {
		return ""
	}
	return group.Platform
}

func groupDailyLimitValue(group *Group) *float64 {
	if group == nil {
		return nil
	}
	return group.DailyLimitUSD
}

func groupWeeklyLimitValue(group *Group) *float64 {
	if group == nil {
		return nil
	}
	return group.WeeklyLimitUSD
}

func groupMonthlyLimitValue(group *Group) *float64 {
	if group == nil {
		return nil
	}
	return group.MonthlyLimitUSD
}

func nullableInt64Equal(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func nullableFloat64Equal(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return math.Abs(*left-*right) < 1e-9
}
