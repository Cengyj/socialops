package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPromoHandlerUpdateClearsExpiresAtWithZeroTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	repo := &adminPromoRepoStub{
		code: &service.PromoCode{
			ID:          7,
			Code:        "WELCOME",
			BonusAmount: 5,
			MaxUses:     10,
			UsedCount:   1,
			Status:      service.PromoCodeStatusActive,
			ExpiresAt:   &expiresAt,
			Notes:       "initial",
			CreatedAt:   time.Now().UTC().Add(-time.Hour),
			UpdatedAt:   time.Now().UTC().Add(-time.Hour),
		},
	}
	handler := NewPromoHandler(service.NewPromoService(repo, nil, nil, nil))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	c.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/promo-codes/7",
		bytes.NewBufferString(`{"expires_at":0}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Nil(t, repo.code.ExpiresAt)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			ID        int64      `json:"id"`
			Code      string     `json:"code"`
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, int64(7), envelope.Data.ID)
	require.Equal(t, "WELCOME", envelope.Data.Code)
	require.Nil(t, envelope.Data.ExpiresAt)
}

type adminPromoRepoStub struct {
	code *service.PromoCode
}

func (r *adminPromoRepoStub) Create(context.Context, *service.PromoCode) error {
	panic("unexpected Create call")
}

func (r *adminPromoRepoStub) GetByID(_ context.Context, id int64) (*service.PromoCode, error) {
	if r.code == nil || r.code.ID != id {
		return nil, service.ErrPromoCodeNotFound
	}
	copy := *r.code
	return &copy, nil
}

func (r *adminPromoRepoStub) GetByCode(context.Context, string) (*service.PromoCode, error) {
	panic("unexpected GetByCode call")
}

func (r *adminPromoRepoStub) GetByCodeForUpdate(context.Context, string) (*service.PromoCode, error) {
	panic("unexpected GetByCodeForUpdate call")
}

func (r *adminPromoRepoStub) Update(_ context.Context, code *service.PromoCode) error {
	if code == nil {
		return service.ErrPromoCodeNotFound
	}
	copy := *code
	copy.UpdatedAt = time.Now().UTC()
	r.code = &copy
	return nil
}

func (r *adminPromoRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (r *adminPromoRepoStub) List(context.Context, pagination.PaginationParams) ([]service.PromoCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *adminPromoRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string) ([]service.PromoCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *adminPromoRepoStub) CreateUsage(context.Context, *service.PromoCodeUsage) error {
	panic("unexpected CreateUsage call")
}

func (r *adminPromoRepoStub) GetUsageByPromoCodeAndUser(context.Context, int64, int64) (*service.PromoCodeUsage, error) {
	panic("unexpected GetUsageByPromoCodeAndUser call")
}

func (r *adminPromoRepoStub) ListUsagesByPromoCode(context.Context, int64, pagination.PaginationParams) ([]service.PromoCodeUsage, *pagination.PaginationResult, error) {
	panic("unexpected ListUsagesByPromoCode call")
}

func (r *adminPromoRepoStub) IncrementUsedCount(context.Context, int64) error {
	panic("unexpected IncrementUsedCount call")
}
