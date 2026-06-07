package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/group"
	"github.com/Wei-Shaw/socialops/ent/subscriptionplan"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
)

// validatePlanRequired checks that all required fields for a plan are provided.
func validatePlanRequired(name string, groupID *int64, platform string, price float64, validityDays int, validityUnit string, originalPrice *float64, monthlyLimitUSD, dailyLimitUSD, weeklyLimitUSD *float64) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if (groupID == nil || *groupID <= 0) && strings.TrimSpace(platform) == "" {
		return infraerrors.BadRequest("PLAN_PLATFORM_REQUIRED", "platform is required")
	}
	if price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if validityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if strings.TrimSpace(validityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if originalPrice != nil && *originalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if err := validatePlanQuotaLimits(monthlyLimitUSD, dailyLimitUSD, weeklyLimitUSD); err != nil {
		return err
	}
	return nil
}

// validatePlanPatch validates only the non-nil fields in a patch update.
func validatePlanPatch(req UpdatePlanRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if req.GroupID != nil && *req.GroupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if req.Platform != nil && strings.TrimSpace(*req.Platform) == "" {
		return infraerrors.BadRequest("PLAN_PLATFORM_REQUIRED", "platform is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if req.ValidityDays != nil && *req.ValidityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if req.ValidityUnit != nil && strings.TrimSpace(*req.ValidityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	return nil
}

func validatePlanQuotaLimits(monthlyLimitUSD, dailyLimitUSD, weeklyLimitUSD *float64) error {
	quota, err := requiredPositivePlanLimit(monthlyLimitUSD, "PLAN_QUOTA_REQUIRED", "quota amount must be greater than 0")
	if err != nil {
		return err
	}
	daily, err := optionalPlanGuardrail(dailyLimitUSD, "PLAN_DAILY_LIMIT_INVALID", "daily guardrail must be greater than or equal to 0")
	if err != nil {
		return err
	}
	weekly, err := optionalPlanGuardrail(weeklyLimitUSD, "PLAN_WEEKLY_LIMIT_INVALID", "weekly guardrail must be greater than or equal to 0")
	if err != nil {
		return err
	}
	if daily != nil && *daily > quota {
		return infraerrors.BadRequest("PLAN_DAILY_LIMIT_EXCEEDS_QUOTA", "daily guardrail cannot exceed quota amount")
	}
	if weekly != nil && *weekly > quota {
		return infraerrors.BadRequest("PLAN_WEEKLY_LIMIT_EXCEEDS_QUOTA", "weekly guardrail cannot exceed quota amount")
	}
	if daily != nil && weekly != nil && *daily > *weekly {
		return infraerrors.BadRequest("PLAN_DAILY_LIMIT_EXCEEDS_WEEKLY_LIMIT", "daily guardrail cannot exceed weekly guardrail")
	}
	return nil
}

func requiredPositivePlanLimit(value *float64, code, message string) (float64, error) {
	if value == nil || !isFiniteFloat64(*value) || *value <= 0 {
		return 0, infraerrors.BadRequest(code, message)
	}
	return *value, nil
}

func optionalPlanGuardrail(value *float64, code, message string) (*float64, error) {
	if value == nil || *value == 0 {
		return nil, nil
	}
	if !isFiniteFloat64(*value) || *value < 0 {
		return nil, infraerrors.BadRequest(code, message)
	}
	return value, nil
}

func isFiniteFloat64(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func resolvePlanQuotaInput(monthlyLimitUSD, quotaUSD *float64) (*float64, bool, error) {
	if quotaUSD == nil {
		return monthlyLimitUSD, monthlyLimitUSD != nil, nil
	}
	if monthlyLimitUSD == nil {
		return quotaUSD, true, nil
	}
	if !planFloat64Equal(*monthlyLimitUSD, *quotaUSD) {
		return nil, false, infraerrors.BadRequest("PLAN_QUOTA_CONFLICT", "quota_usd must match monthly_limit_usd when both are provided")
	}
	return quotaUSD, true, nil
}

func planFloat64Equal(left, right float64) bool {
	return math.Abs(left-right) < 1e-9
}

func PlanQuotaUSD(limits ...*float64) *float64 {
	for _, limit := range limits {
		if limit == nil {
			continue
		}
		value := *limit
		if !isFiniteFloat64(value) || value <= 0 {
			continue
		}
		return &value
	}
	return nil
}

// --- Plan CRUD ---

// PlanGroupInfo holds the group details needed for subscription plan display.
type PlanGroupInfo struct {
	Platform         string   `json:"platform"`
	Name             string   `json:"name"`
	RateMultiplier   float64  `json:"rate_multiplier"`
	Status           string   `json:"status"`
	SubscriptionType string   `json:"subscription_type"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd"`
	CapabilityScopes []string `json:"supported_capability_scopes"`
}

// GetGroupPlatformMap returns a map of group_id to platform for the given plans.
func (s *PaymentConfigService) GetGroupPlatformMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]string {
	info := s.GetGroupInfoMap(ctx, plans)
	m := make(map[int64]string, len(info))
	for id, gi := range info {
		m[id] = gi.Platform
	}
	return m
}

// GetGroupInfoMap returns group metadata needed to render subscription plans.
func (s *PaymentConfigService) GetGroupInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanGroupInfo {
	if s == nil || s.entClient == nil {
		return nil
	}
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]bool)
	for _, p := range plans {
		if p == nil || p.GroupID <= 0 || seen[p.GroupID] {
			continue
		}
		seen[p.GroupID] = true
		ids = append(ids, p.GroupID)
	}
	if len(ids) == 0 {
		return nil
	}
	groups, err := s.entClient.Group.Query().Where(group.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil
	}
	out := make(map[int64]PlanGroupInfo, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		out[int64(g.ID)] = PlanGroupInfo{
			Platform:         g.Platform,
			Name:             g.Name,
			RateMultiplier:   g.RateMultiplier,
			Status:           g.Status,
			SubscriptionType: g.SubscriptionType,
			DailyLimitUSD:    g.DailyLimitUsd,
			WeeklyLimitUSD:   g.WeeklyLimitUsd,
			MonthlyLimitUSD:  g.MonthlyLimitUsd,
			CapabilityScopes: []string{},
		}
	}
	return out
}

func (s *PaymentConfigService) ListPlans(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) ListPlansForSale(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Where(subscriptionplan.ForSaleEQ(true)).Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) CreatePlan(ctx context.Context, req CreatePlanRequest) (*dbent.SubscriptionPlan, error) {
	quotaUSD, _, err := resolvePlanQuotaInput(req.MonthlyLimitUSD, req.QuotaUSD)
	if err != nil {
		return nil, err
	}
	if err := validatePlanRequired(req.Name, req.GroupID, req.Platform, req.Price, req.ValidityDays, req.ValidityUnit, req.OriginalPrice, quotaUSD, req.DailyLimitUSD, req.WeeklyLimitUSD); err != nil {
		return nil, err
	}
	groupID, platform, err := s.resolvePlanBindingGroup(ctx, req.GroupID, req.Platform)
	if err != nil {
		return nil, err
	}
	b := s.entClient.SubscriptionPlan.Create().
		SetGroupID(groupID).SetPlatform(platform).SetName(req.Name).SetDescription(req.Description).
		SetPrice(req.Price).SetValidityDays(req.ValidityDays).SetValidityUnit(req.ValidityUnit).
		SetFeatures(req.Features).SetProductName(req.ProductName).
		SetForSale(req.ForSale).SetSortOrder(req.SortOrder)
	if req.OriginalPrice != nil {
		b.SetOriginalPrice(*req.OriginalPrice)
	}
	if guardrail := normalizedPlanGuardrail(req.DailyLimitUSD); guardrail != nil {
		b.SetDailyLimitUsd(*guardrail)
	}
	if guardrail := normalizedPlanGuardrail(req.WeeklyLimitUSD); guardrail != nil {
		b.SetWeeklyLimitUsd(*guardrail)
	}
	b.SetMonthlyLimitUsd(*quotaUSD)
	return b.Save(ctx)
}

// UpdatePlan updates a subscription plan by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update boilerplate
// plus a validation guard for non-nil fields.
func (s *PaymentConfigService) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanPatch(req); err != nil {
		return nil, err
	}
	quotaUSD, quotaTouched, err := resolvePlanQuotaInput(req.MonthlyLimitUSD, req.QuotaUSD)
	if err != nil {
		return nil, err
	}
	quotaReq := req
	if quotaTouched {
		quotaReq.MonthlyLimitUSD = quotaUSD
	}
	if err := s.validatePlanQuotaPatch(ctx, id, quotaReq); err != nil {
		return nil, err
	}
	var (
		resolvedGroupID  *int64
		resolvedPlatform *string
	)
	if req.GroupID != nil || req.Platform != nil {
		groupID, platform, err := s.resolvePlanBindingGroup(ctx, req.GroupID, planStringValue(req.Platform))
		if err != nil {
			return nil, err
		}
		resolvedGroupID = &groupID
		resolvedPlatform = &platform
	}
	u := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if resolvedGroupID != nil {
		u.SetGroupID(*resolvedGroupID)
	}
	if resolvedPlatform != nil {
		u.SetPlatform(*resolvedPlatform)
	}
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Description != nil {
		u.SetDescription(*req.Description)
	}
	if req.Price != nil {
		u.SetPrice(*req.Price)
	}
	if req.OriginalPrice != nil {
		u.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.ValidityDays != nil {
		u.SetValidityDays(*req.ValidityDays)
	}
	if req.ValidityUnit != nil {
		u.SetValidityUnit(*req.ValidityUnit)
	}
	if req.Features != nil {
		u.SetFeatures(*req.Features)
	}
	if req.ProductName != nil {
		u.SetProductName(*req.ProductName)
	}
	if req.ForSale != nil {
		u.SetForSale(*req.ForSale)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	if req.DailyLimitUSD != nil {
		if guardrail := normalizedPlanGuardrail(req.DailyLimitUSD); guardrail != nil {
			u.SetDailyLimitUsd(*guardrail)
		} else {
			u.ClearDailyLimitUsd()
		}
	}
	if req.WeeklyLimitUSD != nil {
		if guardrail := normalizedPlanGuardrail(req.WeeklyLimitUSD); guardrail != nil {
			u.SetWeeklyLimitUsd(*guardrail)
		} else {
			u.ClearWeeklyLimitUsd()
		}
	}
	if quotaTouched {
		u.SetMonthlyLimitUsd(*quotaUSD)
	}
	return u.Save(ctx)
}

func (s *PaymentConfigService) validatePlanQuotaPatch(ctx context.Context, id int64, req UpdatePlanRequest) error {
	if req.MonthlyLimitUSD == nil && req.DailyLimitUSD == nil && req.WeeklyLimitUSD == nil {
		return nil
	}
	if s == nil || s.entClient == nil {
		return infraerrors.InternalServer("PLAN_REPOSITORY_UNAVAILABLE", "subscription plan repository is unavailable")
	}
	current, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	quota := current.MonthlyLimitUsd
	daily := current.DailyLimitUsd
	weekly := current.WeeklyLimitUsd
	if req.MonthlyLimitUSD != nil {
		quota = req.MonthlyLimitUSD
	}
	if req.DailyLimitUSD != nil {
		daily = req.DailyLimitUSD
	}
	if req.WeeklyLimitUSD != nil {
		weekly = req.WeeklyLimitUSD
	}
	return validatePlanQuotaLimits(quota, daily, weekly)
}

func normalizedPlanGuardrail(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

func (s *PaymentConfigService) resolvePlanBindingGroup(ctx context.Context, requestedGroupID *int64, platform string) (int64, string, error) {
	if s == nil || s.entClient == nil {
		return 0, "", infraerrors.InternalServer("GROUP_REPOSITORY_UNAVAILABLE", "group repository is unavailable")
	}
	if requestedGroupID != nil && *requestedGroupID > 0 {
		g, err := s.entClient.Group.Query().Where(group.IDEQ(*requestedGroupID)).Only(ctx)
		if err != nil {
			return 0, "", infraerrors.NotFound("GROUP_NOT_FOUND", "subscription group is not available")
		}
		if g.SubscriptionType != SubscriptionTypeSubscription {
			return 0, "", infraerrors.BadRequest("GROUP_TYPE_MISMATCH", "group is not a subscription type")
		}
		resolvedPlatform := normalizePlanPlatform(platform)
		groupPlatform := normalizePlanPlatform(g.Platform)
		if resolvedPlatform == "" {
			resolvedPlatform = groupPlatform
		}
		if groupPlatform != "" && groupPlatform != "social" && resolvedPlatform != groupPlatform {
			return 0, "", infraerrors.BadRequest("PLAN_GROUP_PLATFORM_MISMATCH", "plan platform must match subscription group platform")
		}
		return g.ID, resolvedPlatform, nil
	}

	resolvedPlatform := normalizePlanPlatform(platform)
	if resolvedPlatform == "" {
		return 0, "", infraerrors.BadRequest("PLAN_PLATFORM_REQUIRED", "platform is required")
	}
	g, err := s.entClient.Group.Query().
		Where(
			group.PlatformIn(planPlatformAliases(resolvedPlatform)...),
			group.SubscriptionTypeEQ(SubscriptionTypeSubscription),
			group.StatusEQ(StatusActive),
		).
		Order(dbent.Asc(group.FieldSortOrder), dbent.Asc(group.FieldID)).
		First(ctx)
	if err != nil {
		return 0, "", infraerrors.NotFound("GROUP_NOT_FOUND", "no internal subscription group is available for the selected platform")
	}
	return g.ID, resolvedPlatform, nil
}

func normalizePlanPlatform(platform string) string {
	value := strings.TrimSpace(strings.ToLower(platform))
	switch value {
	case "", "social":
		return value
	case "twitter", "x":
		return "x_twitter"
	default:
		return value
	}
}

func NormalizePlanPlatform(platform string) string {
	return normalizePlanPlatform(platform)
}

func planPlatformAliases(platform string) []string {
	normalized := normalizePlanPlatform(platform)
	switch normalized {
	case "":
		return nil
	case "x_twitter":
		return []string{"x_twitter", "twitter", "x"}
	default:
		return []string{normalized}
	}
}

func planStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *PaymentConfigService) DeletePlan(ctx context.Context, id int64) error {
	count, err := s.countPendingOrdersByPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this plan has %d in-progress orders and cannot be deleted; wait for orders to complete first", count))
	}
	return s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

// GetPlan returns a subscription plan by ID.
func (s *PaymentConfigService) GetPlan(ctx context.Context, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	return plan, nil
}

// GetDefaultSubscriptionPlan validates a plan before it can be used as a default signup package.
func (s *PaymentConfigService) GetDefaultSubscriptionPlan(ctx context.Context, id int64) (DefaultSubscriptionPlanBinding, error) {
	if s == nil || s.entClient == nil {
		return DefaultSubscriptionPlanBinding{}, infraerrors.InternalServer("PLAN_REPOSITORY_UNAVAILABLE", "subscription plan repository is unavailable")
	}

	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil || plan == nil || plan.GroupID <= 0 {
		return DefaultSubscriptionPlanBinding{}, ErrDefaultSubPlanInvalid
	}

	bindingGroup, err := s.entClient.Group.Query().Where(group.IDEQ(plan.GroupID)).Only(ctx)
	if err != nil ||
		bindingGroup == nil ||
		bindingGroup.SubscriptionType != SubscriptionTypeSubscription ||
		bindingGroup.Status != StatusActive {
		return DefaultSubscriptionPlanBinding{}, ErrDefaultSubPlanInvalid
	}

	return DefaultSubscriptionPlanBinding{
		ID:      plan.ID,
		GroupID: plan.GroupID,
	}, nil
}
