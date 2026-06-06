package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestPaymentHandlerGetPlansReturnsQuotaPackageCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newPaymentPlansHandlerTestClient(t)

	activeGroup := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetWeeklyLimitUsd(50).
		SaveX(ctx)
	inactiveGroup := client.Group.Create().
		SetName("Inactive Execution Pool").
		SetPlatform("x_twitter").
		SetStatus("inactive").
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)
	client.SubscriptionPlan.Create().
		SetGroupID(activeGroup.ID).
		SetPlatform("").
		SetName("X Starter").
		SetDescription("starter execution quota").
		SetPrice(19).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetFeatures("Login\nFollow").
		SetForSale(true).
		SetSortOrder(1).
		SetDailyLimitUsd(10).
		SetMonthlyLimitUsd(120).
		SaveX(ctx)
	client.SubscriptionPlan.Create().
		SetGroupID(inactiveGroup.ID).
		SetPlatform("x_twitter").
		SetName("Inactive Plan").
		SetPrice(19).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetForSale(true).
		SetSortOrder(2).
		SetMonthlyLimitUsd(120).
		SaveX(ctx)

	handler := NewPaymentHandler(nil, service.NewPaymentConfigService(client, nil, nil))
	recorder := httptest.NewRecorder()
	reqCtx, _ := gin.CreateTestContext(recorder)
	reqCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/plans", nil)

	handler.GetPlans(reqCtx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Code int                    `json:"code"`
		Data []dto.SubscriptionPlan `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Len(t, envelope.Data, 1)
	require.Equal(t, "X Starter", envelope.Data[0].Name)
	require.Equal(t, "x_twitter", envelope.Data[0].Platform)
	require.NotNil(t, envelope.Data[0].QuotaUSD)
	require.Equal(t, 120.0, *envelope.Data[0].QuotaUSD)
	require.NotNil(t, envelope.Data[0].WeeklyLimitUSD)
	require.Equal(t, 50.0, *envelope.Data[0].WeeklyLimitUSD)
	require.Equal(t, []string{"Login", "Follow"}, envelope.Data[0].Features)
	require.True(t, envelope.Data[0].ForSale)
}

func TestPaymentHandlerGetCheckoutInfoPropagatesPlanCatalogError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, db := newPaymentPlansHandlerTestClientWithDB(t)
	_, err := db.Exec("ALTER TABLE subscription_plans RENAME TO subscription_plans_unavailable")
	require.NoError(t, err)

	handler := NewPaymentHandler(nil, service.NewPaymentConfigService(
		client,
		&paymentHandlerSettingRepoStub{values: map[string]string{}},
		nil,
	))
	recorder := httptest.NewRecorder()
	reqCtx, _ := gin.CreateTestContext(recorder)
	reqCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)

	handler.GetCheckoutInfo(reqCtx)

	require.NotEqual(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.NotEqual(t, 0, envelope.Code)
	require.NotEmpty(t, envelope.Message)
}

func newPaymentPlansHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	client, _ := newPaymentPlansHandlerTestClientWithDB(t)
	return client
}

func newPaymentPlansHandlerTestClientWithDB(t *testing.T) (*dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client, db
}

type paymentHandlerSettingRepoStub struct {
	values map[string]string
}

func (s *paymentHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (s *paymentHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *paymentHandlerSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (s *paymentHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *paymentHandlerSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *paymentHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *paymentHandlerSettingRepoStub) Delete(context.Context, string) error {
	return nil
}
