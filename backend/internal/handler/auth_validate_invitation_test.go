package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type validateInvitationSettingRepo struct {
	values map[string]string
}

func (s *validateInvitationSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *validateInvitationSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *validateInvitationSettingRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *validateInvitationSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *validateInvitationSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *validateInvitationSettingRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *validateInvitationSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type validateInvitationRedeemRepo struct {
	code *service.RedeemCode
}

func (r *validateInvitationRedeemRepo) Create(context.Context, *service.RedeemCode) error {
	panic("unexpected Create call")
}

func (r *validateInvitationRedeemRepo) CreateBatch(context.Context, []service.RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (r *validateInvitationRedeemRepo) GetByID(context.Context, int64) (*service.RedeemCode, error) {
	panic("unexpected GetByID call")
}

func (r *validateInvitationRedeemRepo) GetByCode(_ context.Context, code string) (*service.RedeemCode, error) {
	if r.code == nil || r.code.Code != code {
		return nil, service.ErrRedeemCodeNotFound
	}
	cloned := *r.code
	return &cloned, nil
}

func (r *validateInvitationRedeemRepo) Update(context.Context, *service.RedeemCode) error {
	panic("unexpected Update call")
}

func (r *validateInvitationRedeemRepo) BatchUpdate(context.Context, []int64, service.RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (r *validateInvitationRedeemRepo) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (r *validateInvitationRedeemRepo) Use(context.Context, int64, int64) error {
	panic("unexpected Use call")
}

func (r *validateInvitationRedeemRepo) List(context.Context, pagination.PaginationParams) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *validateInvitationRedeemRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *validateInvitationRedeemRepo) ListByUser(context.Context, int64, int) ([]service.RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (r *validateInvitationRedeemRepo) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (r *validateInvitationRedeemRepo) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func newValidateInvitationHandler(code *service.RedeemCode) *AuthHandler {
	cfg := &config.Config{}
	settingSvc := service.NewSettingService(&validateInvitationSettingRepo{
		values: map[string]string{
			service.SettingKeyInvitationCodeEnabled: "true",
		},
	}, cfg)
	redeemSvc := service.NewRedeemService(
		&validateInvitationRedeemRepo{code: code},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	return &AuthHandler{
		settingSvc:    settingSvc,
		redeemService: redeemSvc,
	}
}

func postValidateInvitationCode(t *testing.T, handler *AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/validate-invitation-code", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ValidateInvitationCode(c)
	return rec
}

func TestValidateInvitationCodeRejectsExpiredUnusedCode(t *testing.T) {
	expiresAt := time.Now().Add(-time.Minute)
	handler := newValidateInvitationHandler(&service.RedeemCode{
		ID:        1,
		Code:      "INVITE-EXPIRED",
		Type:      service.RedeemTypeInvitation,
		Status:    service.StatusUnused,
		ExpiresAt: &expiresAt,
	})

	rec := postValidateInvitationCode(t, handler, `{"code":"INVITE-EXPIRED"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var envelope struct {
		Code int                            `json:"code"`
		Data ValidateInvitationCodeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.False(t, envelope.Data.Valid)
	require.Equal(t, "INVITATION_CODE_INVALID", envelope.Data.ErrorCode)
}
