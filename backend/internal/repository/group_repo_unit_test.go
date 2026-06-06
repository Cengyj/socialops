//go:build unit

package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupRepository_GetByID_ReadsSocialOpsSubscriptionGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"name",
		"description",
		"platform",
		"rate_multiplier",
		"is_exclusive",
		"status",
		"subscription_type",
		"daily_limit_usd",
		"weekly_limit_usd",
		"monthly_limit_usd",
		"default_validity_days",
		"sort_order",
		"rpm_limit",
		"created_at",
		"updated_at",
	}).AddRow(
		int64(12),
		"Starter Social",
		"subscription group",
		"social",
		1.25,
		false,
		service.StatusActive,
		service.SubscriptionTypeSubscription,
		sql.NullFloat64{Float64: 10, Valid: true},
		sql.NullFloat64{},
		sql.NullFloat64{Float64: 100, Valid: true},
		45,
		3,
		120,
		now,
		now,
	)
	mock.ExpectQuery(regexp.QuoteMeta("FROM groups")).
		WithArgs(int64(12)).
		WillReturnRows(rows)

	repo := NewGroupRepository(nil, db)
	group, err := repo.GetByID(context.Background(), 12)

	require.NoError(t, err)
	require.NotNil(t, group)
	require.Equal(t, int64(12), group.ID)
	require.Equal(t, "Starter Social", group.Name)
	require.Equal(t, "social", group.Platform)
	require.Equal(t, service.SubscriptionTypeSubscription, group.SubscriptionType)
	require.True(t, group.Hydrated)
	require.Equal(t, 1.25, group.RateMultiplier)
	require.Equal(t, 45, group.DefaultValidityDays)
	require.Equal(t, 120, group.RPMLimit)
	require.NotNil(t, group.DailyLimitUSD)
	require.Equal(t, 10.0, *group.DailyLimitUSD)
	require.Nil(t, group.WeeklyLimitUSD)
	require.NotNil(t, group.MonthlyLimitUSD)
	require.Equal(t, 100.0, *group.MonthlyLimitUSD)
	require.NoError(t, mock.ExpectationsWereMet())
}
