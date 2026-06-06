package dto

import (
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/service"
)

type SubscriptionPlan struct {
	ID               int64    `json:"id"`
	Platform         string   `json:"platform"`
	GroupID          int64    `json:"group_id"`
	GroupPlatform    string   `json:"group_platform"`
	GroupName        string   `json:"group_name"`
	QuotaUSD         *float64 `json:"quota_usd"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD   *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd"`
	CapabilityScopes []string `json:"supported_capability_scopes,omitempty"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Price            float64  `json:"price"`
	OriginalPrice    *float64 `json:"original_price,omitempty"`
	ValidityDays     int      `json:"validity_days"`
	ValidityUnit     string   `json:"validity_unit"`
	Features         []string `json:"features"`
	ProductName      string   `json:"product_name"`
	ForSale          bool     `json:"for_sale"`
	SortOrder        int      `json:"sort_order"`
}

type AdminSubscriptionPlan struct {
	SubscriptionPlan

	GroupStatus      string    `json:"group_status"`
	SubscriptionType string    `json:"subscription_type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func SubscriptionPlansFromEnt(plans []*dbent.SubscriptionPlan, groupInfo map[int64]service.PlanGroupInfo) []SubscriptionPlan {
	out := make([]SubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		if item := SubscriptionPlanFromEnt(plan, groupInfo[planGroupID(plan)]); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func AvailableSubscriptionPlansFromEnt(plans []*dbent.SubscriptionPlan, groupInfo map[int64]service.PlanGroupInfo) []SubscriptionPlan {
	out := make([]SubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		gi, ok := groupInfo[planGroupID(plan)]
		if !ok || !IsAvailableSubscriptionPlanGroup(gi) {
			continue
		}
		if item := SubscriptionPlanFromEnt(plan, gi); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func AdminSubscriptionPlansFromEnt(plans []*dbent.SubscriptionPlan, groupInfo map[int64]service.PlanGroupInfo) []AdminSubscriptionPlan {
	out := make([]AdminSubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		if item := AdminSubscriptionPlanFromEnt(plan, groupInfo[planGroupID(plan)]); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func SubscriptionPlanFromEnt(plan *dbent.SubscriptionPlan, gi service.PlanGroupInfo) *SubscriptionPlan {
	if plan == nil {
		return nil
	}
	dailyLimitUSD := firstPlanLimit(plan.DailyLimitUsd, gi.DailyLimitUSD)
	weeklyLimitUSD := firstPlanLimit(plan.WeeklyLimitUsd, gi.WeeklyLimitUSD)
	monthlyLimitUSD := firstPlanLimit(plan.MonthlyLimitUsd, gi.MonthlyLimitUSD)
	return &SubscriptionPlan{
		ID:               int64(plan.ID),
		Platform:         planPlatform(plan.Platform, gi.Platform),
		GroupID:          plan.GroupID,
		GroupPlatform:    gi.Platform,
		GroupName:        gi.Name,
		QuotaUSD:         service.PlanQuotaUSD(monthlyLimitUSD),
		DailyLimitUSD:    dailyLimitUSD,
		WeeklyLimitUSD:   weeklyLimitUSD,
		MonthlyLimitUSD:  monthlyLimitUSD,
		CapabilityScopes: gi.CapabilityScopes,
		Name:             plan.Name,
		Description:      plan.Description,
		Price:            plan.Price,
		OriginalPrice:    plan.OriginalPrice,
		ValidityDays:     plan.ValidityDays,
		ValidityUnit:     plan.ValidityUnit,
		Features:         ParsePlanFeatures(plan.Features),
		ProductName:      plan.ProductName,
		ForSale:          plan.ForSale,
		SortOrder:        plan.SortOrder,
	}
}

func AdminSubscriptionPlanFromEnt(plan *dbent.SubscriptionPlan, gi service.PlanGroupInfo) *AdminSubscriptionPlan {
	base := SubscriptionPlanFromEnt(plan, gi)
	if base == nil {
		return nil
	}
	return &AdminSubscriptionPlan{
		SubscriptionPlan: *base,
		GroupStatus:      gi.Status,
		SubscriptionType: gi.SubscriptionType,
		CreatedAt:        plan.CreatedAt,
		UpdatedAt:        plan.UpdatedAt,
	}
}

func IsAvailableSubscriptionPlanGroup(gi service.PlanGroupInfo) bool {
	return gi.Status == service.StatusActive && gi.SubscriptionType == service.SubscriptionTypeSubscription
}

func ParsePlanFeatures(raw string) []string {
	out := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		if item := strings.TrimSpace(line); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func planGroupID(plan *dbent.SubscriptionPlan) int64 {
	if plan == nil {
		return 0
	}
	return plan.GroupID
}

func firstPlanLimit(primary, fallback *float64) *float64 {
	if primary != nil {
		return primary
	}
	return fallback
}

func planPlatform(primary, fallback string) string {
	if trimmed := strings.TrimSpace(primary); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}
