//go:build unit

package service

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAdminServiceGetUserUsageStatsCostsOnlyFinalSuccessfulCharges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(42)
	client, mock := newAdminBalanceHistorySQLClient(t)
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\), COALESCE\(SUM\(CASE WHEN status = 'success' AND charge_status = 'charged' THEN charged_amount ELSE 0 END\), 0\)::double precision.*FROM social_task_logs.*WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count", "charged_amount"}).AddRow(int64(3), 0.10))

	svc := &adminServiceImpl{entClient: client}

	stats, err := svc.GetUserUsageStats(ctx, userID, "month")

	require.NoError(t, err)
	body, ok := stats.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "month", body["period"])
	require.Equal(t, int64(3), body["total_requests"])
	require.InDelta(t, 0.10, body["total_cost"], 1e-9)
}
