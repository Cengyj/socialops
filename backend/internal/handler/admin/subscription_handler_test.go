package admin

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

func newAdminSubscriptionHandlerTestClient(t *testing.T) *dbent.Client {
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

type adminSubscriptionGroupRepoStub struct {
	group *service.Group
}

func (s *adminSubscriptionGroupRepoStub) Create(context.Context, *service.Group) error {
	panic("unexpected Create call")
}
func (s *adminSubscriptionGroupRepoStub) GetByID(context.Context, int64) (*service.Group, error) {
	return s.group, nil
}
func (s *adminSubscriptionGroupRepoStub) GetByIDLite(context.Context, int64) (*service.Group, error) {
	return s.group, nil
}
func (s *adminSubscriptionGroupRepoStub) Update(context.Context, *service.Group) error {
	panic("unexpected Update call")
}
func (s *adminSubscriptionGroupRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *adminSubscriptionGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}
func (s *adminSubscriptionGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *adminSubscriptionGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, *bool) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *adminSubscriptionGroupRepoStub) ListActive(context.Context) ([]service.Group, error) {
	panic("unexpected ListActive call")
}
func (s *adminSubscriptionGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	panic("unexpected ListActiveByPlatform call")
}
func (s *adminSubscriptionGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName call")
}
func (s *adminSubscriptionGroupRepoStub) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

func TestSubscriptionHandlerResetQuotaHonorsRequestedWindowsAndReturnsSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)

	user := client.User.Create().
		SetEmail("reset-quota@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("Reset Group").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-time.Hour)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(windowStart).
		SetWeeklyWindowStart(windowStart).
		SetMonthlyWindowStart(windowStart).
		SetDailyUsageUsd(10).
		SetWeeklyUsageUsd(20).
		SetMonthlyUsageUsd(30).
		SetNotes("reset").
		SaveX(ctx)

	svc := service.NewSubscriptionService(nil, repository.NewUserSubscriptionRepository(client), nil, nil, nil)
	handler := NewSubscriptionHandler(svc)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(sub.ID, 10)}}
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/subscriptions/"+strconv.FormatInt(sub.ID, 10)+"/reset-quota",
		bytes.NewBufferString(`{"daily":true,"weekly":false,"monthly":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ResetQuota(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			ID              int64   `json:"id"`
			GroupID         int64   `json:"group_id"`
			DailyUsageUSD   float64 `json:"daily_usage_usd"`
			WeeklyUsageUSD  float64 `json:"weekly_usage_usd"`
			MonthlyUsageUSD float64 `json:"monthly_usage_usd"`
			Group           *struct {
				Name     string `json:"name"`
				Platform string `json:"platform"`
			} `json:"group"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, sub.ID, envelope.Data.ID)
	require.Equal(t, group.ID, envelope.Data.GroupID)
	require.Zero(t, envelope.Data.DailyUsageUSD)
	require.Equal(t, 20.0, envelope.Data.WeeklyUsageUSD)
	require.Zero(t, envelope.Data.MonthlyUsageUSD)
	require.NotNil(t, envelope.Data.Group)
	require.Equal(t, "Reset Group", envelope.Data.Group.Name)
	require.Equal(t, "x_twitter", envelope.Data.Group.Platform)
}

func TestSubscriptionHandlerAssignSupportsPlanDrivenCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)

	user := client.User.Create().
		SetEmail("plan-assignment@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	adminActor := client.User.Create().
		SetEmail("admin-plan-assignment@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleAdmin).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		SetWeeklyLimitUsd(50).
		SetMonthlyLimitUsd(200).
		SaveX(ctx)
	plan := client.SubscriptionPlan.Create().
		SetGroupID(group.ID).
		SetPlatform("x_twitter").
		SetName("X Starter Monthly").
		SetDescription("Starter package").
		SetPrice(29).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetDailyLimitUsd(6).
		SetMonthlyLimitUsd(120).
		SetForSale(true).
		SetSortOrder(1).
		SaveX(ctx)

	groupRepo := &adminSubscriptionGroupRepoStub{
		group: &service.Group{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			Status:           group.Status,
			SubscriptionType: group.SubscriptionType,
			DailyLimitUSD:    group.DailyLimitUsd,
			WeeklyLimitUSD:   group.WeeklyLimitUsd,
			MonthlyLimitUSD:  group.MonthlyLimitUsd,
		},
	}
	subscriptionSvc := service.NewSubscriptionService(groupRepo, repository.NewUserSubscriptionRepository(client), nil, client, nil)
	handler := NewSubscriptionHandler(subscriptionSvc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/subscriptions/assign",
		bytes.NewBufferString(`{"user_id":`+strconv.FormatInt(user.ID, 10)+`,"plan_id":`+strconv.FormatInt(plan.ID, 10)+`,"validity_days":45,"notes":"admin created package subscription"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: adminActor.ID})

	handler.Assign(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			UserID          int64    `json:"user_id"`
			GroupID         int64    `json:"group_id"`
			PlanID          *int64   `json:"plan_id"`
			PlanName        string   `json:"plan_name"`
			PlanPlatform    string   `json:"plan_platform"`
			DailyLimitUSD   *float64 `json:"daily_limit_usd"`
			WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
			MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
			Notes           string   `json:"notes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, user.ID, envelope.Data.UserID)
	require.Equal(t, group.ID, envelope.Data.GroupID)
	require.NotNil(t, envelope.Data.PlanID)
	require.Equal(t, plan.ID, *envelope.Data.PlanID)
	require.Equal(t, "X Starter Monthly", envelope.Data.PlanName)
	require.Equal(t, "x_twitter", envelope.Data.PlanPlatform)
	require.NotNil(t, envelope.Data.DailyLimitUSD)
	require.Equal(t, 6.0, *envelope.Data.DailyLimitUSD)
	require.NotNil(t, envelope.Data.WeeklyLimitUSD)
	require.Equal(t, 50.0, *envelope.Data.WeeklyLimitUSD)
	require.NotNil(t, envelope.Data.MonthlyLimitUSD)
	require.Equal(t, 120.0, *envelope.Data.MonthlyLimitUSD)
	require.Equal(t, "admin created package subscription", envelope.Data.Notes)
}

func TestSubscriptionHandlerCreateSupportsPlanOnlyCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)

	user := client.User.Create().
		SetEmail("plan-create@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	adminActor := client.User.Create().
		SetEmail("admin-plan-create@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleAdmin).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetWeeklyLimitUsd(50).
		SaveX(ctx)
	plan := client.SubscriptionPlan.Create().
		SetGroupID(group.ID).
		SetPlatform("x_twitter").
		SetName("X Growth Monthly").
		SetDescription("Growth package").
		SetPrice(59).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetMonthlyLimitUsd(220).
		SetForSale(true).
		SaveX(ctx)

	groupRepo := &adminSubscriptionGroupRepoStub{
		group: &service.Group{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			Status:           group.Status,
			SubscriptionType: group.SubscriptionType,
			WeeklyLimitUSD:   group.WeeklyLimitUsd,
		},
	}
	subscriptionSvc := service.NewSubscriptionService(groupRepo, repository.NewUserSubscriptionRepository(client), nil, client, nil)
	handler := NewSubscriptionHandler(subscriptionSvc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/subscriptions",
		bytes.NewBufferString(`{"user_id":`+strconv.FormatInt(user.ID, 10)+`,"plan_id":`+strconv.FormatInt(plan.ID, 10)+`,"validity_days":30,"notes":"created from package"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: adminActor.ID})

	handler.Create(c)

	require.Equal(t, http.StatusCreated, rec.Code)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			UserID          int64    `json:"user_id"`
			GroupID         int64    `json:"group_id"`
			PlanID          *int64   `json:"plan_id"`
			PlanName        string   `json:"plan_name"`
			WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
			MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
			Notes           string   `json:"notes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, user.ID, envelope.Data.UserID)
	require.Equal(t, group.ID, envelope.Data.GroupID)
	require.NotNil(t, envelope.Data.PlanID)
	require.Equal(t, plan.ID, *envelope.Data.PlanID)
	require.Equal(t, "X Growth Monthly", envelope.Data.PlanName)
	require.NotNil(t, envelope.Data.WeeklyLimitUSD)
	require.Equal(t, 50.0, *envelope.Data.WeeklyLimitUSD)
	require.NotNil(t, envelope.Data.MonthlyLimitUSD)
	require.Equal(t, 220.0, *envelope.Data.MonthlyLimitUSD)
	require.Equal(t, "created from package", envelope.Data.Notes)
}

func TestSubscriptionHandlerCreateRenewsExpiredExistingPlanSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)

	user := client.User.Create().
		SetEmail("expired-plan-create@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	adminActor := client.User.Create().
		SetEmail("admin-expired-plan-create@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleAdmin).
		SetStatus(service.StatusActive).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("X Renewal Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)
	plan := client.SubscriptionPlan.Create().
		SetGroupID(group.ID).
		SetPlatform("x_twitter").
		SetName("X Renewal Package").
		SetDescription("Renewal package").
		SetPrice(39).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetMonthlyLimitUsd(180).
		SetForSale(true).
		SaveX(ctx)
	oldStart := time.Now().UTC().AddDate(0, 0, -60)
	oldWindowStart := oldStart.AddDate(0, 0, 1)
	expired := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(oldStart).
		SetExpiresAt(oldStart.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusExpired).
		SetAssignedAt(oldStart).
		SetDailyWindowStart(oldWindowStart).
		SetWeeklyWindowStart(oldWindowStart).
		SetMonthlyWindowStart(oldWindowStart).
		SetDailyUsageUsd(10).
		SetWeeklyUsageUsd(20).
		SetMonthlyUsageUsd(30).
		SetNotes("old package").
		SaveX(ctx)

	groupRepo := &adminSubscriptionGroupRepoStub{
		group: &service.Group{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			Status:           group.Status,
			SubscriptionType: group.SubscriptionType,
		},
	}
	subscriptionSvc := service.NewSubscriptionService(groupRepo, repository.NewUserSubscriptionRepository(client), nil, client, nil)
	handler := NewSubscriptionHandler(subscriptionSvc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/subscriptions",
		bytes.NewBufferString(`{"user_id":`+strconv.FormatInt(user.ID, 10)+`,"plan_id":`+strconv.FormatInt(plan.ID, 10)+`,"validity_days":30,"notes":"renewed from package"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: adminActor.ID})

	handler.Create(c)

	require.Equal(t, http.StatusCreated, rec.Code)
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			ID              int64    `json:"id"`
			PlanID          *int64   `json:"plan_id"`
			PlanName        string   `json:"plan_name"`
			Status          string   `json:"status"`
			StartsAt        string   `json:"starts_at"`
			ExpiresAt       string   `json:"expires_at"`
			DailyUsageUSD   float64  `json:"daily_usage_usd"`
			WeeklyUsageUSD  float64  `json:"weekly_usage_usd"`
			MonthlyUsageUSD float64  `json:"monthly_usage_usd"`
			AssignedBy      *int64   `json:"assigned_by"`
			Notes           string   `json:"notes"`
			QuotaUSD        *float64 `json:"quota_usd"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, expired.ID, envelope.Data.ID)
	require.NotNil(t, envelope.Data.PlanID)
	require.Equal(t, plan.ID, *envelope.Data.PlanID)
	require.Equal(t, "X Renewal Package", envelope.Data.PlanName)
	require.Equal(t, service.SubscriptionStatusActive, envelope.Data.Status)
	require.Zero(t, envelope.Data.DailyUsageUSD)
	require.Zero(t, envelope.Data.WeeklyUsageUSD)
	require.Zero(t, envelope.Data.MonthlyUsageUSD)
	require.NotNil(t, envelope.Data.AssignedBy)
	require.Equal(t, adminActor.ID, *envelope.Data.AssignedBy)
	require.Equal(t, "renewed from package", envelope.Data.Notes)
	require.NotNil(t, envelope.Data.QuotaUSD)
	require.Equal(t, 180.0, *envelope.Data.QuotaUSD)

	startsAt, err := time.Parse(time.RFC3339, envelope.Data.StartsAt)
	require.NoError(t, err)
	expiresAt, err := time.Parse(time.RFC3339, envelope.Data.ExpiresAt)
	require.NoError(t, err)
	require.True(t, startsAt.After(oldStart))
	require.True(t, expiresAt.After(time.Now()))
}

func TestSubscriptionHandlerCreateRejectsGroupOnlyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSubscriptionHandler(service.NewSubscriptionService(nil, nil, nil, nil, nil))
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/subscriptions",
		bytes.NewBufferString(`{"user_id":1,"group_id":10,"validity_days":30}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "plan_id")
}
