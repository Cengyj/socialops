//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemStatsRepoStub struct {
	RedeemCodeRepository
	codes []RedeemCode
}

func (r *redeemStatsRepoStub) List(_ context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	limit := params.Limit()
	offset := params.Offset()
	if offset >= len(r.codes) {
		return []RedeemCode{}, &pagination.PaginationResult{Total: int64(len(r.codes)), Page: params.Page, PageSize: limit, Pages: 1}, nil
	}
	end := offset + limit
	if end > len(r.codes) {
		end = len(r.codes)
	}
	pages := (len(r.codes) + limit - 1) / limit
	return r.codes[offset:end], &pagination.PaginationResult{Total: int64(len(r.codes)), Page: params.Page, PageSize: limit, Pages: pages}, nil
}

func TestRedeemService_GetStatsUsesRealCodes(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	usedBy := int64(7)
	repo := &redeemStatsRepoStub{codes: []RedeemCode{
		{Code: "BAL-ACTIVE", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused, ExpiresAt: &future},
		{Code: "BAL-USED", Type: RedeemTypeBalance, Value: 25, Status: StatusUsed, UsedBy: &usedBy},
		{Code: "CONC-USED", Type: RedeemTypeConcurrency, Value: 3, Status: StatusUsed, UsedBy: &usedBy},
		{Code: "SUB-EXPIRED", Type: RedeemTypeSubscription, Value: 99, Status: StatusUnused, ExpiresAt: &past},
		{Code: "INV-DISABLED", Type: RedeemTypeInvitation, Value: 0, Status: StatusDisabled},
		{Code: "SUB-EXPLICIT-EXPIRED", Type: RedeemTypeSubscription, Value: 29, Status: StatusExpired},
	}}
	svc := &RedeemService{redeemRepo: repo}

	stats, err := svc.GetStats(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(6), stats.TotalCodes)
	require.Equal(t, int64(1), stats.ActiveCodes)
	require.Equal(t, int64(2), stats.UsedCodes)
	require.Equal(t, int64(2), stats.ExpiredCodes)
	require.Equal(t, int64(1), stats.DisabledCodes)
	require.InEpsilon(t, 28.0, stats.TotalValueDistributed, 0.000001)
	require.Equal(t, int64(2), stats.ByType[RedeemTypeBalance])
	require.Equal(t, int64(1), stats.ByType[RedeemTypeConcurrency])
	require.Equal(t, int64(2), stats.ByType[RedeemTypeSubscription])
	require.Equal(t, int64(1), stats.ByType[RedeemTypeInvitation])
}
