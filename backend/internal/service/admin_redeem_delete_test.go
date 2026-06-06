//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type adminRedeemDeleteRepoStub struct {
	RedeemCodeRepository
	codes      map[int64]*RedeemCode
	deletedIDs []int64
}

func (r *adminRedeemDeleteRepoStub) GetByID(_ context.Context, id int64) (*RedeemCode, error) {
	code, ok := r.codes[id]
	if !ok {
		return nil, ErrRedeemCodeNotFound
	}
	cloned := *code
	return &cloned, nil
}

func (r *adminRedeemDeleteRepoStub) Delete(_ context.Context, id int64) error {
	r.deletedIDs = append(r.deletedIDs, id)
	delete(r.codes, id)
	return nil
}

func TestAdminServiceDeleteRedeemCodeRejectsUsedCode(t *testing.T) {
	repo := &adminRedeemDeleteRepoStub{
		codes: map[int64]*RedeemCode{
			7: {ID: 7, Code: "USED-7", Type: RedeemTypeBalance, Value: 10, Status: StatusUsed},
		},
	}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	err := svc.DeleteRedeemCode(context.Background(), 7)

	require.Error(t, err)
	require.True(t, infraerrors.IsConflict(err))
	require.Equal(t, "REDEEM_CODE_DELETE_USED", infraerrors.Reason(err))
	require.Empty(t, repo.deletedIDs)
	require.Contains(t, repo.codes, int64(7))
}

func TestAdminServiceBatchDeleteRedeemCodesRejectsUsedCodeWithoutPartialDelete(t *testing.T) {
	repo := &adminRedeemDeleteRepoStub{
		codes: map[int64]*RedeemCode{
			1: {ID: 1, Code: "UNUSED-1", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused},
			2: {ID: 2, Code: "USED-2", Type: RedeemTypeBalance, Value: 10, Status: StatusUsed},
			3: {ID: 3, Code: "UNUSED-3", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused},
		},
	}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	deleted, err := svc.BatchDeleteRedeemCodes(context.Background(), []int64{1, 2, 3})

	require.Zero(t, deleted)
	require.Error(t, err)
	require.True(t, infraerrors.IsConflict(err))
	require.Equal(t, "REDEEM_CODE_DELETE_USED", infraerrors.Reason(err))
	require.Empty(t, repo.deletedIDs)
	require.Contains(t, repo.codes, int64(1))
	require.Contains(t, repo.codes, int64(2))
	require.Contains(t, repo.codes, int64(3))
}
