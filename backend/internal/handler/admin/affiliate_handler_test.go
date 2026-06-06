//go:build unit

package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAffiliateHandlerClearUserSettingsDoesNotPartiallyClearOnResetFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &affiliateClearRepoStub{
		rateSet:  true,
		codeSet:  true,
		resetErr: errors.New("reset failed"),
	}
	handler := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), nil)

	router := gin.New()
	router.DELETE("/admin/affiliates/users/:user_id", handler.ClearUserSettings)

	req := httptest.NewRequest(http.MethodDelete, "/admin/affiliates/users/42", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.NotEqual(t, http.StatusOK, resp.Code)
	require.True(t, repo.rateSet, "failed clear request must not leave rebate rate cleared")
	require.True(t, repo.codeSet, "failed clear request must keep the custom code marker unchanged")
}

type affiliateClearRepoStub struct {
	rateSet  bool
	codeSet  bool
	resetErr error
}

func (r *affiliateClearRepoStub) EnsureUserAffiliate(context.Context, int64) (*service.AffiliateSummary, error) {
	panic("unexpected EnsureUserAffiliate call")
}

func (r *affiliateClearRepoStub) GetAffiliateByCode(context.Context, string) (*service.AffiliateSummary, error) {
	panic("unexpected GetAffiliateByCode call")
}

func (r *affiliateClearRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	panic("unexpected BindInviter call")
}

func (r *affiliateClearRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64, float64) (bool, error) {
	panic("unexpected AccrueQuota call")
}

func (r *affiliateClearRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (r *affiliateClearRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	panic("unexpected ThawFrozenQuota call")
}

func (r *affiliateClearRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}

func (r *affiliateClearRepoStub) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	panic("unexpected ListInvitees call")
}

func (r *affiliateClearRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (r *affiliateClearRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	if r.resetErr != nil {
		return "", r.resetErr
	}
	r.codeSet = false
	return "SYSTEM-CODE", nil
}

func (r *affiliateClearRepoStub) SetUserRebateRate(_ context.Context, _ int64, ratePercent *float64) error {
	r.rateSet = ratePercent != nil
	return nil
}

func (r *affiliateClearRepoStub) ClearUserAffiliateSettings(context.Context, int64) (string, error) {
	if r.resetErr != nil {
		return "", r.resetErr
	}
	r.rateSet = false
	r.codeSet = false
	return "SYSTEM-CODE", nil
}

func (r *affiliateClearRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *affiliateClearRepoStub) ListUsersWithCustomSettings(context.Context, service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (r *affiliateClearRepoStub) ListAffiliateInviteRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (r *affiliateClearRepoStub) ListAffiliateRebateRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (r *affiliateClearRepoStub) ListAffiliateTransferRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}

func (r *affiliateClearRepoStub) GetAffiliateUserOverview(context.Context, int64) (*service.AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}
