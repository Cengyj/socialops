package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/subscriptionplan"
	"github.com/Wei-Shaw/socialops/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
)

// Plan is the legacy service shape for old /plans compatibility.
// New subscription package catalog responses should use handler/dto.SubscriptionPlan.
type Plan struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"`
	Platform        string    `json:"platform"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Price           float64   `json:"price"`
	OriginalPrice   float64   `json:"original_price,omitempty"`
	ValidityDays    int       `json:"validity_days"`
	ValidityUnit    string    `json:"validity_unit"`
	Features        string    `json:"features"`
	ForSale         bool      `json:"for_sale"`
	SortOrder       int       `json:"sort_order"`
	DailyLimitUSD   *float64  `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64  `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64  `json:"monthly_limit_usd"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserPlan represents a user's active plan subscription.
type UserPlan struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	PlanID          int64     `json:"plan_id"`
	PlanName        string    `json:"plan_name"`
	PlanPlatform    string    `json:"plan_platform"`
	Status          string    `json:"status"`
	StartsAt        time.Time `json:"starts_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	DailyLimitUSD   *float64  `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD  *float64  `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD *float64  `json:"monthly_limit_usd,omitempty"`
}

// CreatePlanInput is the legacy input for creating a plan.
// Admin package creation should use PaymentConfigService.CreatePlan.
type CreatePlanInput struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	Price         float64 `json:"price" binding:"required"`
	OriginalPrice float64 `json:"original_price"`
	ValidityDays  int     `json:"validity_days"`
	Features      string  `json:"features"`
	ForSale       bool    `json:"for_sale"`
	SortOrder     int     `json:"sort_order"`
}

// UpdatePlanInput is the legacy input for updating a plan.
// Admin package updates should use PaymentConfigService.UpdatePlan.
type UpdatePlanInput struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	Price         *float64 `json:"price"`
	OriginalPrice *float64 `json:"original_price"`
	ValidityDays  *int     `json:"validity_days"`
	Features      *string  `json:"features"`
	ForSale       *bool    `json:"for_sale"`
	SortOrder     *int     `json:"sort_order"`
}

// PlanService keeps compatibility helpers for legacy user plan endpoints.
// PaymentConfigService owns the current quota package catalog.
type PlanService struct {
	entClient *dbent.Client
}

// NewPlanService creates a new PlanService.
func NewPlanService(entClient *dbent.Client) *PlanService {
	return &PlanService{entClient: entClient}
}

// ListPlans returns all plans.
func (s *PlanService) ListPlans(ctx context.Context) ([]*Plan, error) {
	ents, err := s.entClient.SubscriptionPlan.Query().
		Order(dbent.Asc(subscriptionplan.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]*Plan, len(ents))
	for i, e := range ents {
		plans[i] = planFromEnt(e)
	}
	return plans, nil
}

// ListPlansForSale returns plans available for purchase.
func (s *PlanService) ListPlansForSale(ctx context.Context) ([]*Plan, error) {
	ents, err := s.entClient.SubscriptionPlan.Query().
		Where(subscriptionplan.ForSaleEQ(true)).
		Order(dbent.Asc(subscriptionplan.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]*Plan, len(ents))
	for i, e := range ents {
		plans[i] = planFromEnt(e)
	}
	return plans, nil
}

// GetPlan returns a plan by ID.
func (s *PlanService) GetPlan(ctx context.Context, id int64) (*Plan, error) {
	e, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "plan not found")
		}
		return nil, err
	}
	return planFromEnt(e), nil
}

// CreatePlan creates a new plan.
func (s *PlanService) CreatePlan(ctx context.Context, input *CreatePlanInput) (*Plan, error) {
	if input.ValidityDays <= 0 {
		input.ValidityDays = 30
	}
	q := s.entClient.SubscriptionPlan.Create().
		SetName(input.Name).
		SetDescription(input.Description).
		SetPrice(input.Price).
		SetOriginalPrice(input.OriginalPrice).
		SetValidityDays(input.ValidityDays).
		SetFeatures(input.Features).
		SetForSale(input.ForSale).
		SetSortOrder(input.SortOrder)

	e, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	return planFromEnt(e), nil
}

// UpdatePlan updates a plan.
func (s *PlanService) UpdatePlan(ctx context.Context, id int64, input *UpdatePlanInput) (*Plan, error) {
	q := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if input.Name != nil {
		q.SetName(*input.Name)
	}
	if input.Description != nil {
		q.SetDescription(*input.Description)
	}
	if input.Price != nil {
		q.SetPrice(*input.Price)
	}
	if input.OriginalPrice != nil {
		q.SetOriginalPrice(*input.OriginalPrice)
	}
	if input.ValidityDays != nil {
		q.SetValidityDays(*input.ValidityDays)
	}
	if input.Features != nil {
		q.SetFeatures(*input.Features)
	}
	if input.ForSale != nil {
		q.SetForSale(*input.ForSale)
	}
	if input.SortOrder != nil {
		q.SetSortOrder(*input.SortOrder)
	}
	e, err := q.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "plan not found")
		}
		return nil, err
	}
	return planFromEnt(e), nil
}

// DeletePlan deletes a plan.
func (s *PlanService) DeletePlan(ctx context.Context, id int64) error {
	err := s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return infraerrors.NotFound("PLAN_NOT_FOUND", "plan not found")
		}
		return err
	}
	return nil
}

// GetUserActivePlan returns the user's current active plan.
func (s *PlanService) GetUserActivePlan(ctx context.Context, userID int64) (*UserPlan, error) {
	sub, err := s.entClient.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.StatusEQ("active"),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		WithGroup().
		Order(dbent.Desc(usersubscription.FieldExpiresAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil // no active plan
		}
		return nil, err
	}
	return &UserPlan{
		ID:              int64(sub.ID),
		UserID:          int64(sub.UserID),
		PlanID:          derefInt64(sub.PlanID),
		PlanName:        sub.PlanName,
		PlanPlatform:    userPlanEffectivePlatform(sub),
		Status:          sub.Status,
		StartsAt:        sub.StartsAt,
		ExpiresAt:       sub.ExpiresAt,
		DailyLimitUSD:   userPlanEffectiveDailyLimit(sub),
		WeeklyLimitUSD:  userPlanEffectiveWeeklyLimit(sub),
		MonthlyLimitUSD: userPlanEffectiveMonthlyLimit(sub),
	}, nil
}

func userPlanEffectivePlatform(sub *dbent.UserSubscription) string {
	if sub == nil {
		return ""
	}
	if sub.PlanPlatform != "" {
		return sub.PlanPlatform
	}
	if sub.Edges.Group != nil {
		return sub.Edges.Group.Platform
	}
	return ""
}

func userPlanEffectiveDailyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.DailyLimitUsd != nil {
		return sub.DailyLimitUsd
	}
	if sub.Edges.Group != nil {
		return sub.Edges.Group.DailyLimitUsd
	}
	return nil
}

func userPlanEffectiveWeeklyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.WeeklyLimitUsd != nil {
		return sub.WeeklyLimitUsd
	}
	if sub.Edges.Group != nil {
		return sub.Edges.Group.WeeklyLimitUsd
	}
	return nil
}

func userPlanEffectiveMonthlyLimit(sub *dbent.UserSubscription) *float64 {
	if sub == nil {
		return nil
	}
	if sub.MonthlyLimitUsd != nil {
		return sub.MonthlyLimitUsd
	}
	if sub.Edges.Group != nil {
		return sub.Edges.Group.MonthlyLimitUsd
	}
	return nil
}

// ListUserPlans returns all plans for a user with pagination.
func (s *PlanService) ListUserPlans(ctx context.Context, userID int64, params pagination.PaginationParams) ([]*UserPlan, *pagination.PaginationResult, error) {
	q := s.entClient.UserSubscription.Query().
		Where(usersubscription.UserIDEQ(userID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	subs, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(usersubscription.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	plans := make([]*UserPlan, len(subs))
	for i, sub := range subs {
		plans[i] = &UserPlan{
			ID:              int64(sub.ID),
			UserID:          int64(sub.UserID),
			PlanID:          derefInt64(sub.PlanID),
			PlanName:        sub.PlanName,
			PlanPlatform:    sub.PlanPlatform,
			Status:          sub.Status,
			StartsAt:        sub.StartsAt,
			ExpiresAt:       sub.ExpiresAt,
			DailyLimitUSD:   sub.DailyLimitUsd,
			WeeklyLimitUSD:  sub.WeeklyLimitUsd,
			MonthlyLimitUSD: sub.MonthlyLimitUsd,
		}
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return plans, result, nil
}

func planFromEnt(e *dbent.SubscriptionPlan) *Plan {
	return &Plan{
		ID:              int64(e.ID),
		GroupID:         e.GroupID,
		Platform:        e.Platform,
		Name:            e.Name,
		Description:     e.Description,
		Price:           e.Price,
		OriginalPrice:   derefFloat(e.OriginalPrice),
		ValidityDays:    e.ValidityDays,
		ValidityUnit:    e.ValidityUnit,
		Features:        e.Features,
		ForSale:         e.ForSale,
		SortOrder:       e.SortOrder,
		DailyLimitUSD:   e.DailyLimitUsd,
		WeeklyLimitUSD:  e.WeeklyLimitUsd,
		MonthlyLimitUSD: e.MonthlyLimitUsd,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
