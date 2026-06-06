//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type balanceHistoryRedeemRepo struct {
	redeemRepoStub
	codes []RedeemCode
	total int64
	sum   float64
}

func (r *balanceHistoryRedeemRepo) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return append([]RedeemCode(nil), r.codes...), &pagination.PaginationResult{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    r.total,
	}, nil
}

func (r *balanceHistoryRedeemRepo) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	return r.sum, nil
}

func newAdminBalanceHistorySQLClient(t *testing.T) (*dbent.Client, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, client.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})
	return client, mock
}

func expectAffiliateBalanceHistoryQuery(mock sqlmock.Sqlmock, userID int64, offset, limit int, rows *sqlmock.Rows) {
	mock.ExpectQuery(`(?s)SELECT id,.*amount::double precision,.*created_at.*FROM user_affiliate_ledger.*WHERE user_id = \$1.*AND action = 'transfer'.*OFFSET \$2.*LIMIT \$3`).
		WithArgs(userID, offset, limit).
		WillReturnRows(rows)
}

func expectAffiliateBalanceHistoryCount(mock sqlmock.Sqlmock, userID int64, total int64) {
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM user_affiliate_ledger.*WHERE user_id = \$1.*AND action = 'transfer'`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
}

func TestAdminServiceGetUserBalanceHistoryReturnsAffiliateTransfers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(42)
	createdAt := time.Date(2026, 6, 4, 10, 30, 0, 0, time.UTC)
	client, mock := newAdminBalanceHistorySQLClient(t)
	expectAffiliateBalanceHistoryQuery(
		mock,
		userID,
		0,
		10,
		sqlmock.NewRows([]string{"id", "amount", "created_at"}).AddRow(int64(7), 4.25, createdAt),
	)
	expectAffiliateBalanceHistoryCount(mock, userID, 1)

	svc := &adminServiceImpl{
		redeemCodeRepo: &balanceHistoryRedeemRepo{sum: 12.5},
		entClient:      client,
	}

	codes, total, totalRecharged, err := svc.GetUserBalanceHistory(ctx, userID, 1, 10, RedeemTypeAffiliateBalance)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.InDelta(t, 12.5, totalRecharged, 1e-9)
	require.Len(t, codes, 1)
	require.Equal(t, int64(-7), codes[0].ID)
	require.Equal(t, "AFF-7", codes[0].Code)
	require.Equal(t, RedeemTypeAffiliateBalance, codes[0].Type)
	require.Equal(t, StatusUsed, codes[0].Status)
	require.InDelta(t, 4.25, codes[0].Value, 1e-9)
	require.NotNil(t, codes[0].UsedAt)
	require.Equal(t, createdAt, *codes[0].UsedAt)
	require.Equal(t, createdAt, codes[0].CreatedAt)
}

func TestAdminServiceGetUserBalanceHistoryMergesAffiliateTransfersByDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(42)
	usedBy := userID
	redeemAt := time.Date(2026, 6, 4, 9, 0, 0, 0, time.UTC)
	affiliateAt := redeemAt.Add(time.Hour)
	client, mock := newAdminBalanceHistorySQLClient(t)
	expectAffiliateBalanceHistoryQuery(
		mock,
		userID,
		0,
		1000,
		sqlmock.NewRows([]string{"id", "amount", "created_at"}).AddRow(int64(8), 3.75, affiliateAt),
	)
	expectAffiliateBalanceHistoryCount(mock, userID, 1)

	svc := &adminServiceImpl{
		redeemCodeRepo: &balanceHistoryRedeemRepo{
			codes: []RedeemCode{
				{
					ID:        1,
					Type:      RedeemTypeBalance,
					Value:     8,
					Status:    StatusUsed,
					UsedBy:    &usedBy,
					UsedAt:    &redeemAt,
					CreatedAt: redeemAt,
				},
			},
			total: 1,
			sum:   12.5,
		},
		entClient: client,
	}

	codes, total, totalRecharged, err := svc.GetUserBalanceHistory(ctx, userID, 1, 10, "")

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.InDelta(t, 12.5, totalRecharged, 1e-9)
	require.Len(t, codes, 2)
	require.Equal(t, RedeemTypeAffiliateBalance, codes[0].Type)
	require.Equal(t, RedeemTypeBalance, codes[1].Type)
}
