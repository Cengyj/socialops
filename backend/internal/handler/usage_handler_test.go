//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageHandlerRepoStub struct {
	listParams   pagination.PaginationParams
	listFilters  usagestats.UsageLogFilters
	statsFilters usagestats.UsageLogFilters
	listItems    []service.UsageLog
}

func (s *usageHandlerRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listParams = params
	s.listFilters = filters
	return s.listItems, &pagination.PaginationResult{Total: int64(len(s.listItems)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *usageHandlerRepoStub) GetStatsWithFilters(_ context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsFilters = filters
	return &usagestats.UsageStats{}, nil
}

func (s *usageHandlerRepoStub) GetByID(context.Context, int64, int64) (*service.UsageLog, error) {
	return nil, service.ErrUsageLogNotFound
}

func TestUsageHandlerListPassesSocialTaskFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &usageHandlerRepoStub{}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage?page=2&page_size=5&sort_by=operation&sort_order=asc&operation=follow&status=success&start_date=2026-06-01T00:00:00Z&end_date=2026-06-02T00:00:00Z", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.List(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 2, repo.listParams.Page)
	require.Equal(t, 5, repo.listParams.PageSize)
	require.Equal(t, "operation", repo.listParams.SortBy)
	require.Equal(t, "asc", repo.listParams.SortOrder)
	require.Equal(t, int64(42), repo.listFilters.UserID)
	require.Equal(t, "follow", repo.listFilters.Model)
	require.Equal(t, "success", repo.listFilters.Status)
	require.NotNil(t, repo.listFilters.StartTime)
	require.NotNil(t, repo.listFilters.EndTime)
	require.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), *repo.listFilters.StartTime)
	require.Equal(t, time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), *repo.listFilters.EndTime)
}

func TestUsageHandlerStatsPassesSocialTaskFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &usageHandlerRepoStub{}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage/stats?operation=like&status=failed&start_date=2026-06-01T00:00:00Z&end_date=2026-06-02T00:00:00Z", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.Stats(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.statsFilters.UserID)
	require.Equal(t, "like", repo.statsFilters.Model)
	require.Equal(t, "failed", repo.statsFilters.Status)
	require.NotNil(t, repo.statsFilters.StartTime)
	require.NotNil(t, repo.statsFilters.EndTime)
}

func TestUsageHandlerListReturnsSafeSocialTaskFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawResult := "authorization Bearer abc token=secret proxy=http://127.0.0.1:8080 trace_id=trace-123"
	repo := &usageHandlerRepoStub{listItems: []service.UsageLog{{
		ID:              101,
		UserID:          42,
		SocialAccountID: 7,
		Platform:        "x_twitter",
		AccountName:     "delivery-account",
		Operation:       service.SocialTaskActionFollow,
		Status:          service.SocialTaskLogStatusFailed,
		Quantity:        1,
		Cost:            0,
		ChargeStatus:    service.SocialTaskChargeStatusNotCharged,
		ResultMessage:   &rawResult,
		CreatedAt:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}}}
	handler := NewUsageHandler(service.NewUsageService(repo, nil), nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/usage", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.List(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Items []struct {
				SocialAccountID int64   `json:"social_account_id"`
				Platform        string  `json:"platform"`
				AccountName      string  `json:"account_name"`
				ChargeStatus    string  `json:"charge_status"`
				ResultMessage   *string `json:"result_message"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, int64(7), body.Data.Items[0].SocialAccountID)
	require.Equal(t, "x_twitter", body.Data.Items[0].Platform)
	require.Equal(t, "delivery-account", body.Data.Items[0].AccountName)
	require.Equal(t, service.SocialTaskChargeStatusNotCharged, body.Data.Items[0].ChargeStatus)
	require.NotNil(t, body.Data.Items[0].ResultMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *body.Data.Items[0].ResultMessage)
	require.NotContains(t, rec.Body.String(), "Bearer abc")
	require.NotContains(t, rec.Body.String(), "127.0.0.1")
	require.NotContains(t, rec.Body.String(), "trace-123")
}
