package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/payment"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminPaymentHandlerPlanResponsesExposeQuotaUSD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)
	group := client.Group.Create().
		SetName("X Execution Package").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(20).
		SetWeeklyLimitUsd(80).
		SetMonthlyLimitUsd(200).
		SaveX(ctx)

	handler := NewPaymentHandler(nil, service.NewPaymentConfigService(client, nil, nil))

	createRec := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(createRec)
	createCtx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/payment/plans",
		bytes.NewBufferString(`{"group_id":`+strconv.FormatInt(group.ID, 10)+`,"name":"X Starter","platform":"x_twitter","price":19.9,"validity_days":1,"validity_unit":"months","quota_usd":120,"daily_limit_usd":10,"features":"Login\nFollow"}`),
	)
	createCtx.Request.Header.Set("Content-Type", "application/json")

	handler.CreatePlan(createCtx)

	require.Equal(t, http.StatusCreated, createRec.Code)
	created := decodeAdminPlanEnvelope(t, createRec)
	require.Equal(t, "X Starter", created.Name)
	require.Equal(t, group.ID, created.GroupID)
	require.Equal(t, service.StatusActive, created.GroupStatus)
	require.Equal(t, service.SubscriptionTypeSubscription, created.SubscriptionType)
	require.NotNil(t, created.QuotaUSD)
	require.Equal(t, 120.0, *created.QuotaUSD)
	require.NotNil(t, created.MonthlyLimitUSD)
	require.Equal(t, 120.0, *created.MonthlyLimitUSD)
	require.NotNil(t, created.DailyLimitUSD)
	require.Equal(t, 10.0, *created.DailyLimitUSD)
	require.NotNil(t, created.WeeklyLimitUSD)
	require.Equal(t, 80.0, *created.WeeklyLimitUSD)
	require.Equal(t, []string{"Login", "Follow"}, created.Features)

	updateRec := httptest.NewRecorder()
	updateCtx, _ := gin.CreateTestContext(updateRec)
	updateCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(created.ID, 10)}}
	updateCtx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/payment/plans/"+strconv.FormatInt(created.ID, 10),
		bytes.NewBufferString(`{"quota_usd":180,"weekly_limit_usd":90}`),
	)
	updateCtx.Request.Header.Set("Content-Type", "application/json")

	handler.UpdatePlan(updateCtx)

	require.Equal(t, http.StatusOK, updateRec.Code)
	updated := decodeAdminPlanEnvelope(t, updateRec)
	require.NotNil(t, updated.QuotaUSD)
	require.Equal(t, 180.0, *updated.QuotaUSD)
	require.NotNil(t, updated.MonthlyLimitUSD)
	require.Equal(t, 180.0, *updated.MonthlyLimitUSD)
	require.NotNil(t, updated.WeeklyLimitUSD)
	require.Equal(t, 90.0, *updated.WeeklyLimitUSD)

	listRec := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRec)
	listCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/payment/plans", nil)

	handler.ListPlans(listCtx)

	require.Equal(t, http.StatusOK, listRec.Code)
	listed := decodeAdminPlanListEnvelope(t, listRec)
	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].QuotaUSD)
	require.Equal(t, 180.0, *listed[0].QuotaUSD)
}

func TestAdminPaymentHandlerCreateProviderMasksSensitiveConfigInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newAdminSubscriptionHandlerTestClient(t)
	handler := newAdminPaymentProviderTestHandler(client)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/payment/providers",
		bytes.NewBufferString(`{"provider_key":"stripe","name":"Stripe Main","config":{"secretKey":"create-secret-should-not-return","publishableKey":"public-visible-key","webhookSecret":"create-webhook-secret-should-not-return","currency":"CNY"},"supported_types":["stripe"],"enabled":false,"payment_mode":"","sort_order":10,"limits":"","refund_enabled":true,"allow_user_refund":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateProvider(c)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotContains(t, rec.Body.String(), "create-secret-should-not-return")
	require.NotContains(t, rec.Body.String(), "create-webhook-secret-should-not-return")

	created := decodeAdminProviderEnvelope(t, rec)
	require.Equal(t, "Stripe Main", created.Name)
	require.Equal(t, "stripe", created.ProviderKey)
	require.Equal(t, []string{"stripe"}, created.SupportedTypes)
	require.Equal(t, "public-visible-key", created.Config["publishableKey"])
	require.Equal(t, "CNY", created.Config["currency"])
	require.NotContains(t, created.Config, "secretKey")
	require.NotContains(t, created.Config, "webhookSecret")
	require.True(t, created.RefundEnabled)
	require.True(t, created.AllowUserRefund)
}

func TestAdminPaymentHandlerUpdateProviderMasksSensitiveConfigInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)
	configService := service.NewPaymentConfigService(client, nil, nil)
	handler := newAdminPaymentProviderTestHandler(client)

	instance, err := configService.CreateProviderInstance(ctx, service.CreateProviderInstanceRequest{
		ProviderKey: "stripe",
		Name:        "Stripe Main",
		Config: map[string]string{
			"secretKey":      "existing-secret-should-not-return",
			"publishableKey": "public-visible-key",
			"webhookSecret":  "existing-webhook-secret-should-not-return",
			"currency":       "CNY",
		},
		SupportedTypes:  []string{"stripe"},
		Enabled:         false,
		RefundEnabled:   true,
		AllowUserRefund: true,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(instance.ID, 10)}}
	c.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/payment/providers/"+strconv.FormatInt(instance.ID, 10),
		bytes.NewBufferString(`{"config":{"secretKey":"update-secret-should-not-return","webhookSecret":"update-webhook-secret-should-not-return","currency":"USD"},"enabled":false}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateProvider(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "existing-secret-should-not-return")
	require.NotContains(t, rec.Body.String(), "existing-webhook-secret-should-not-return")
	require.NotContains(t, rec.Body.String(), "update-secret-should-not-return")
	require.NotContains(t, rec.Body.String(), "update-webhook-secret-should-not-return")

	updated := decodeAdminProviderEnvelope(t, rec)
	require.Equal(t, "Stripe Main", updated.Name)
	require.Equal(t, "stripe", updated.ProviderKey)
	require.Equal(t, "public-visible-key", updated.Config["publishableKey"])
	require.Equal(t, "USD", updated.Config["currency"])
	require.NotContains(t, updated.Config, "secretKey")
	require.NotContains(t, updated.Config, "webhookSecret")
}

func TestAdminPaymentHandlerOrderResponsesExposeCurrencyWithoutSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)
	handler := newAdminPaymentProviderTestHandler(client)

	user, err := client.User.Create().
		SetEmail("admin-payment-currency@example.com").
		SetPasswordHash("hash").
		SetUsername("admin-payment-currency").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(100).
		SetPayAmount(103).
		SetFeeRate(3).
		SetRechargeCode("ADMIN-CURRENCY").
		SetOutTradeNo("sub2_admin_currency").
		SetPaymentType(payment.TypeStripe).
		SetPaymentTradeNo("stripe-trade-admin-currency").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(service.OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderSnapshot(map[string]any{
			"provider_key": payment.TypeStripe,
			"currency":     "HKD",
		}).
		Save(ctx)
	require.NoError(t, err)

	listRec := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRec)
	listCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/payment/orders", nil)

	handler.ListOrders(listCtx)

	require.Equal(t, http.StatusOK, listRec.Code)
	listItem := decodeAdminPaymentOrderListItem(t, listRec)
	require.Equal(t, "HKD", listItem["currency"])
	require.NotContains(t, listItem, "provider_snapshot")

	detailRec := httptest.NewRecorder()
	detailCtx, _ := gin.CreateTestContext(detailRec)
	detailCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(order.ID, 10)}}
	detailCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/payment/orders/"+strconv.FormatInt(order.ID, 10), nil)

	handler.GetOrderDetail(detailCtx)

	require.Equal(t, http.StatusOK, detailRec.Code)
	detailOrder := decodeAdminPaymentOrderDetail(t, detailRec)
	require.Equal(t, "HKD", detailOrder["currency"])
	require.NotContains(t, detailOrder, "provider_snapshot")
}

func TestAdminPaymentHandlerOrderDetailPropagatesAuditLogErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newAdminSubscriptionHandlerTestClient(t)
	handler := newAdminPaymentProviderTestHandler(client)

	user, err := client.User.Create().
		SetEmail("admin-payment-audit-error@example.com").
		SetPasswordHash("hash").
		SetUsername("admin-payment-audit-error").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(100).
		SetPayAmount(100).
		SetFeeRate(0).
		SetRechargeCode("ADMIN-AUDIT-ERROR").
		SetOutTradeNo("sub2_admin_audit_error").
		SetPaymentType(payment.TypeStripe).
		SetPaymentTradeNo("stripe-trade-admin-audit-error").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(service.OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ExecContext(ctx, "DROP TABLE payment_audit_logs")
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(order.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/payment/orders/"+strconv.FormatInt(order.ID, 10), nil)

	handler.GetOrderDetail(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), `"code":0`)
}

func newAdminPaymentProviderTestHandler(client *dbent.Client) *PaymentHandler {
	configService := service.NewPaymentConfigService(client, nil, nil)
	paymentService := service.NewPaymentService(
		client,
		payment.NewRegistry(),
		payment.NewDefaultLoadBalancer(client, nil),
		nil,
		nil,
		configService,
		nil,
		nil,
		nil,
	)
	return NewPaymentHandler(paymentService, configService)
}

func decodeAdminPlanEnvelope(t *testing.T, rec *httptest.ResponseRecorder) dto.AdminSubscriptionPlan {
	t.Helper()
	var envelope struct {
		Code int                       `json:"code"`
		Data dto.AdminSubscriptionPlan `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func decodeAdminPlanListEnvelope(t *testing.T, rec *httptest.ResponseRecorder) []dto.AdminSubscriptionPlan {
	t.Helper()
	var envelope struct {
		Code int                         `json:"code"`
		Data []dto.AdminSubscriptionPlan `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func decodeAdminProviderEnvelope(t *testing.T, rec *httptest.ResponseRecorder) service.ProviderInstanceResponse {
	t.Helper()
	var envelope struct {
		Code int                              `json:"code"`
		Data service.ProviderInstanceResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func decodeAdminPaymentOrderListItem(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Len(t, envelope.Data.Items, 1)
	return envelope.Data.Items[0]
}

func decodeAdminPaymentOrderDetail(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Order map[string]any `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data.Order
}
