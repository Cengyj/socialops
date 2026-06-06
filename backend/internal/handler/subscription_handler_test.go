package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/repository"
	servermiddleware "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newSubscriptionHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestSubscriptionHandlerGetSummaryMatchesFrontendContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newSubscriptionHandlerTestClient(t)

	user := client.User.Create().
		SetEmail("summary@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		SetWeeklyLimitUsd(50).
		SetMonthlyLimitUsd(100).
		SaveX(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanName("X Starter").
		SetPlanPlatform("x_twitter").
		SetWeeklyLimitUsd(40).
		SetStartsAt(now.Add(-time.Hour)).
		SetExpiresAt(now.Add(48 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now.Add(-time.Hour)).
		SetDailyWindowStart(now.Add(-time.Hour)).
		SetWeeklyWindowStart(now.Add(-time.Hour)).
		SetMonthlyWindowStart(now.Add(-time.Hour)).
		SetDailyUsageUsd(2.5).
		SetWeeklyUsageUsd(10).
		SetMonthlyUsageUsd(25).
		SaveX(ctx)

	handler := NewSubscriptionHandler(service.NewSubscriptionService(nil, repository.NewUserSubscriptionRepository(client), nil, nil, nil))
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/summary", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: user.ID})

	handler.GetSummary(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			ActiveCount   int `json:"active_count"`
			Subscriptions []struct {
				ID              int64    `json:"id"`
				GroupName       string   `json:"group_name"`
				Status          string   `json:"status"`
				DailyProgress   *float64 `json:"daily_progress"`
				WeeklyProgress  *float64 `json:"weekly_progress"`
				MonthlyProgress *float64 `json:"monthly_progress"`
				ExpiresAt       *string  `json:"expires_at"`
				DaysRemaining   *int     `json:"days_remaining"`
				UnexpectedUser  *int64   `json:"UserID"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, 1, envelope.Data.ActiveCount)
	require.Len(t, envelope.Data.Subscriptions, 1)

	item := envelope.Data.Subscriptions[0]
	require.Equal(t, "X Starter", item.GroupName)
	require.Equal(t, service.SubscriptionStatusActive, item.Status)
	require.NotNil(t, item.DailyProgress)
	require.InDelta(t, 25.0, *item.DailyProgress, 1e-9)
	require.NotNil(t, item.WeeklyProgress)
	require.InDelta(t, 25.0, *item.WeeklyProgress, 1e-9)
	require.NotNil(t, item.MonthlyProgress)
	require.InDelta(t, 25.0, *item.MonthlyProgress, 1e-9)
	require.NotNil(t, item.ExpiresAt)
	require.NotNil(t, item.DaysRemaining)
	require.Nil(t, item.UnexpectedUser)
}

func TestSubscriptionHandlerGetProgressPropagatesProgressErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &subscriptionProgressErrorRepo{
		active: []service.UserSubscription{
			{
				ID:        41,
				UserID:    7,
				GroupID:   3,
				Status:    service.SubscriptionStatusActive,
				StartsAt:  time.Now().Add(-time.Hour),
				ExpiresAt: time.Now().Add(time.Hour),
			},
		},
		getByIDErr: service.ErrSubscriptionNotFound,
	}
	handler := NewSubscriptionHandler(service.NewSubscriptionService(nil, repo, nil, nil, nil))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/progress", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})

	handler.GetProgress(c)

	require.NotEqual(t, http.StatusOK, rec.Code)
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.NotEqual(t, 0, envelope.Code)
	require.NotEmpty(t, envelope.Message)
}

type subscriptionProgressErrorRepo struct {
	active     []service.UserSubscription
	getByIDErr error
}

func (r *subscriptionProgressErrorRepo) Create(context.Context, *service.UserSubscription) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) GetByID(context.Context, int64) (*service.UserSubscription, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	return nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) GetByUserIDAndGroupID(context.Context, int64, int64) (*service.UserSubscription, error) {
	return nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*service.UserSubscription, error) {
	return nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) Update(context.Context, *service.UserSubscription) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) Delete(context.Context, int64) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ListByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return r.active, nil
}

func (r *subscriptionProgressErrorRepo) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) List(context.Context, pagination.PaginationParams, *int64, *int64, *int64, string, string, string, string) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return false, service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ExtendExpiry(context.Context, int64, time.Time) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) UpdateStatus(context.Context, int64, string) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) UpdateNotes(context.Context, int64, string) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ActivateWindows(context.Context, int64, time.Time) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ResetDailyUsage(context.Context, int64, time.Time) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ResetWeeklyUsage(context.Context, int64, time.Time) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) ResetMonthlyUsage(context.Context, int64, time.Time) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) IncrementUsage(context.Context, int64, float64) error {
	return service.ErrSubscriptionNotFound
}

func (r *subscriptionProgressErrorRepo) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	return 0, service.ErrSubscriptionNotFound
}
