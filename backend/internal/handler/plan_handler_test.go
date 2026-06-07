package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/handler/dto"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPlanHandlerListPlansForSaleUsesQuotaPackageContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newPaymentPlansHandlerTestClient(t)

	activeGroup := client.Group.Create().
		SetName("X Execution Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)
	inactiveGroup := client.Group.Create().
		SetName("Inactive Pool").
		SetPlatform("x_twitter").
		SetStatus("inactive").
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)

	client.SubscriptionPlan.Create().
		SetGroupID(activeGroup.ID).
		SetPlatform("x_twitter").
		SetName("X Starter").
		SetDescription("starter quota").
		SetPrice(19).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetForSale(true).
		SetMonthlyLimitUsd(120).
		SaveX(ctx)
	client.SubscriptionPlan.Create().
		SetGroupID(inactiveGroup.ID).
		SetPlatform("x_twitter").
		SetName("Hidden By Group").
		SetPrice(19).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetForSale(true).
		SetMonthlyLimitUsd(120).
		SaveX(ctx)

	handler := NewPlanHandler(nil, service.NewPaymentConfigService(client, nil, nil))
	recorder := httptest.NewRecorder()
	reqCtx, _ := gin.CreateTestContext(recorder)
	reqCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil)

	handler.ListPlansForSale(reqCtx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Code int                    `json:"code"`
		Data []dto.SubscriptionPlan `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Len(t, envelope.Data, 1)
	require.Equal(t, "X Starter", envelope.Data[0].Name)
	require.NotNil(t, envelope.Data[0].QuotaUSD)
	require.Equal(t, 120.0, *envelope.Data[0].QuotaUSD)
	require.Equal(t, "x_twitter", envelope.Data[0].Platform)
}
