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

// Plan represents a service plan for the social platform.
type Plan struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	OriginalPrice float64   `json:"original_price,omitempty"`
	ValidityDays  int       `json:"validity_days"`
	Features      string    `json:"features"`
	ForSale       bool      `json:"for_sale"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserPlan represents a user's active plan subscription.
type UserPlan struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	PlanID    int64     `json:"plan_id"`
	PlanName  string    `json:"plan_name"`
	Status    string    `json:"status"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreatePlanInput is the input for creating a plan.
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

// UpdatePlanInput is the input for updating a plan.
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

// PlanService manages service plans and user plan subscriptions.
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
		Order(dbent.Desc(usersubscription.FieldExpiresAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil // no active plan
		}
		return nil, err
	}
	return &UserPlan{
		ID:        int64(sub.ID),
		UserID:    int64(sub.UserID),
		PlanID:    int64(sub.GroupID),
		Status:    sub.Status,
		StartsAt:  sub.StartsAt,
		ExpiresAt: sub.ExpiresAt,
	}, nil
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
			ID:        int64(sub.ID),
			UserID:    int64(sub.UserID),
			PlanID:    int64(sub.GroupID),
			Status:    sub.Status,
			StartsAt:  sub.StartsAt,
			ExpiresAt: sub.ExpiresAt,
		}
	}

	result := &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.PageSize}
	return plans, result, nil
}

func planFromEnt(e *dbent.SubscriptionPlan) *Plan {
	return &Plan{
		ID:            int64(e.ID),
		Name:          e.Name,
		Description:   e.Description,
		Price:         e.Price,
		OriginalPrice: derefFloat(e.OriginalPrice),
		ValidityDays:  e.ValidityDays,
		Features:      e.Features,
		ForSale:       e.ForSale,
		SortOrder:     e.SortOrder,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
