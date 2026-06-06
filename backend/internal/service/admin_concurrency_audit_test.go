package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type adminConcurrencyAuditUserRepo struct {
	UserRepository
	users map[int64]User

	setCalls []struct {
		ids   []int64
		value int
	}
	addCalls []struct {
		ids   []int64
		delta int
	}
}

func (r *adminConcurrencyAuditUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	copied := user
	return &copied, nil
}

func (r *adminConcurrencyAuditUserRepo) BatchSetConcurrency(_ context.Context, userIDs []int64, value int) (int, error) {
	r.setCalls = append(r.setCalls, struct {
		ids   []int64
		value int
	}{ids: append([]int64(nil), userIDs...), value: value})
	return len(userIDs), nil
}

func (r *adminConcurrencyAuditUserRepo) BatchAddConcurrency(_ context.Context, userIDs []int64, delta int) (int, error) {
	r.addCalls = append(r.addCalls, struct {
		ids   []int64
		delta int
	}{ids: append([]int64(nil), userIDs...), delta: delta})
	return len(userIDs), nil
}

type adminConcurrencyAuditRedeemRepo struct {
	RedeemCodeRepository
	created []RedeemCode
}

func (r *adminConcurrencyAuditRedeemRepo) Create(_ context.Context, code *RedeemCode) error {
	if code != nil {
		r.created = append(r.created, *code)
	}
	return nil
}

func TestAdminServiceBatchUpdateConcurrencyCreatesAdjustmentRecords(t *testing.T) {
	ctx := context.Background()
	userRepo := &adminConcurrencyAuditUserRepo{
		users: map[int64]User{
			9:  {ID: 9, Concurrency: 2},
			12: {ID: 12, Concurrency: 4},
		},
	}
	redeemRepo := &adminConcurrencyAuditRedeemRepo{}
	svc := &adminServiceImpl{
		userRepo:       userRepo,
		redeemCodeRepo: redeemRepo,
	}

	affected, err := svc.BatchUpdateConcurrency(ctx, []int64{9, 9, 0, 12}, 3, "add")

	require.NoError(t, err)
	require.Equal(t, 2, affected)
	require.Len(t, userRepo.addCalls, 1)
	require.Equal(t, []int64{9, 12}, userRepo.addCalls[0].ids)
	require.Len(t, redeemRepo.created, 2)
	for _, record := range redeemRepo.created {
		require.Equal(t, adminConcurrencyAdjustmentType, record.Type)
		require.Equal(t, float64(3), record.Value)
		require.Equal(t, StatusUsed, record.Status)
		require.NotNil(t, record.UsedBy)
		require.Contains(t, []int64{9, 12}, *record.UsedBy)
		require.NotNil(t, record.UsedAt)
	}
}

func TestAdminServiceBatchUpdateConcurrencyRecordsClampedActualDelta(t *testing.T) {
	ctx := context.Background()
	userRepo := &adminConcurrencyAuditUserRepo{
		users: map[int64]User{
			9:  {ID: 9, Concurrency: 2},
			12: {ID: 12, Concurrency: 8},
		},
	}
	redeemRepo := &adminConcurrencyAuditRedeemRepo{}
	svc := &adminServiceImpl{
		userRepo:       userRepo,
		redeemCodeRepo: redeemRepo,
	}

	affected, err := svc.BatchUpdateConcurrency(ctx, []int64{9, 12}, -5, "add")

	require.NoError(t, err)
	require.Equal(t, 2, affected)
	require.Len(t, redeemRepo.created, 2)
	require.InDelta(t, -2, adminConcurrencyAuditRecordValue(t, redeemRepo.created, 9), 1e-9)
	require.InDelta(t, -5, adminConcurrencyAuditRecordValue(t, redeemRepo.created, 12), 1e-9)
}

func adminConcurrencyAuditRecordValue(t *testing.T, records []RedeemCode, userID int64) float64 {
	t.Helper()
	for _, record := range records {
		if record.UsedBy != nil && *record.UsedBy == userID {
			return record.Value
		}
	}
	t.Fatalf("missing audit record for user %d", userID)
	return 0
}
