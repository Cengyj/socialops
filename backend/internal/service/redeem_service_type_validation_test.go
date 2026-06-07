//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type redeemTypeValidationRepoStub struct {
	RedeemCodeRepository

	createCalls      int
	createBatchCalls int
}

func (s *redeemTypeValidationRepoStub) Create(_ context.Context, _ *RedeemCode) error {
	s.createCalls++
	return nil
}

func (s *redeemTypeValidationRepoStub) CreateBatch(_ context.Context, _ []RedeemCode) error {
	s.createBatchCalls++
	return nil
}

func TestRedeemServiceCreateCodeRejectsUnknownType(t *testing.T) {
	t.Parallel()

	for _, codeType := range []string{"unknown", RedeemTypeAffiliateBalance} {
		t.Run(codeType, func(t *testing.T) {
			t.Parallel()

			repo := &redeemTypeValidationRepoStub{}
			svc := &RedeemService{redeemRepo: repo}

			err := svc.CreateCode(context.Background(), &RedeemCode{
				Code:  "BADTYPE",
				Type:  codeType,
				Value: 1,
			})

			require.Error(t, err)
			require.True(t, infraerrors.IsBadRequest(err))
			require.Equal(t, "REDEEM_CODE_TYPE_INVALID", infraerrors.Reason(err))
			require.Zero(t, repo.createCalls)
		})
	}
}

func TestRedeemServiceGenerateCodesRejectsUnknownType(t *testing.T) {
	t.Parallel()

	repo := &redeemTypeValidationRepoStub{}
	svc := &RedeemService{redeemRepo: repo}

	codes, err := svc.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count: 1,
		Type:  "unknown",
		Value: 1,
	})

	require.Nil(t, codes)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_CODE_TYPE_INVALID", infraerrors.Reason(err))
	require.Zero(t, repo.createBatchCalls)
}

func TestAdminServiceGenerateRedeemCodesRejectsUnknownType(t *testing.T) {
	t.Parallel()

	repo := &redeemTypeValidationRepoStub{}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	codes, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{
		Count: 1,
		Type:  "unknown",
		Value: 1,
	})

	require.Nil(t, codes)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_CODE_TYPE_INVALID", infraerrors.Reason(err))
	require.Zero(t, repo.createCalls)
}
