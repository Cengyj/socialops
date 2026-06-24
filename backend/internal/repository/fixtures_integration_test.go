//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"
)

func mustCreateUser(t *testing.T, client *dbent.Client, u *service.User) *service.User {
	t.Helper()
	ctx := context.Background()

	if u.Email == "" {
		u.Email = "user-" + time.Now().Format(time.RFC3339Nano) + "@example.com"
	}
	if u.PasswordHash == "" {
		u.PasswordHash = "test-password-hash"
	}
	if u.Role == "" {
		u.Role = service.RoleUser
	}
	if u.Status == "" {
		u.Status = service.StatusActive
	}
	if u.Concurrency == 0 {
		u.Concurrency = 5
	}

	create := client.User.Create().
		SetEmail(u.Email).
		SetPasswordHash(u.PasswordHash).
		SetRole(u.Role).
		SetStatus(u.Status).
		SetBalance(u.Balance).
		SetConcurrency(u.Concurrency).
		SetUsername(u.Username).
		SetNotes(u.Notes)
	if !u.CreatedAt.IsZero() {
		create.SetCreatedAt(u.CreatedAt)
	}
	if !u.UpdatedAt.IsZero() {
		create.SetUpdatedAt(u.UpdatedAt)
	}

	created, err := create.Save(ctx)
	require.NoError(t, err, "create user")

	u.ID = created.ID
	u.CreatedAt = created.CreatedAt
	u.UpdatedAt = created.UpdatedAt

	if len(u.AllowedGroups) > 0 {
		for _, groupID := range u.AllowedGroups {
			_, err := client.UserAllowedGroup.Create().
				SetUserID(u.ID).
				SetGroupID(groupID).
				Save(ctx)
			require.NoError(t, err, "create user_allowed_groups row")
		}
	}

	return u
}

func mustCreateGroup(t *testing.T, client *dbent.Client, g *service.Group) *service.Group {
	t.Helper()
	ctx := context.Background()

	if g.Platform == "" {
		g.Platform = "social"
	}
	if g.Status == "" {
		g.Status = service.StatusActive
	}
	if g.SubscriptionType == "" {
		g.SubscriptionType = service.SubscriptionTypeStandard
	}

	create := client.Group.Create().
		SetName(g.Name).
		SetPlatform(g.Platform).
		SetStatus(g.Status).
		SetSubscriptionType(g.SubscriptionType).
		SetRateMultiplier(g.RateMultiplier).
		SetIsExclusive(g.IsExclusive)
	if g.Description != "" {
		create.SetDescription(g.Description)
	}
	if g.DailyLimitUSD != nil {
		create.SetDailyLimitUsd(*g.DailyLimitUSD)
	}
	if g.WeeklyLimitUSD != nil {
		create.SetWeeklyLimitUsd(*g.WeeklyLimitUSD)
	}
	if g.MonthlyLimitUSD != nil {
		create.SetMonthlyLimitUsd(*g.MonthlyLimitUSD)
	}
	if !g.CreatedAt.IsZero() {
		create.SetCreatedAt(g.CreatedAt)
	}
	if !g.UpdatedAt.IsZero() {
		create.SetUpdatedAt(g.UpdatedAt)
	}

	created, err := create.Save(ctx)
	require.NoError(t, err, "create group")

	g.ID = created.ID
	g.CreatedAt = created.CreatedAt
	g.UpdatedAt = created.UpdatedAt
	return g
}

type historicalAPIKeyRow struct {
	ID        int64
	UserID    int64
	Key       string
	Name      string
	GroupID   *int64
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func mustCreateHistoricalAPIKey(t *testing.T, client *dbent.Client, k historicalAPIKeyRow) *dbent.APIKey {
	t.Helper()
	ctx := context.Background()

	if k.Status == "" {
		k.Status = service.StatusActive
	}
	if k.Key == "" {
		k.Key = "sk-" + time.Now().Format("150405.000000")
	}
	if k.Name == "" {
		k.Name = "default"
	}

	create := client.APIKey.Create().
		SetUserID(k.UserID).
		SetKey(k.Key).
		SetName(k.Name).
		SetStatus(k.Status)
	if k.GroupID != nil {
		create.SetGroupID(*k.GroupID)
	}
	if !k.CreatedAt.IsZero() {
		create.SetCreatedAt(k.CreatedAt)
	}
	if !k.UpdatedAt.IsZero() {
		create.SetUpdatedAt(k.UpdatedAt)
	}

	created, err := create.Save(ctx)
	require.NoError(t, err, "create historical api key row")

	return created
}

func mustCreateRedeemCode(t *testing.T, client *dbent.Client, c *service.RedeemCode) *service.RedeemCode {
	t.Helper()
	ctx := context.Background()

	if c.Status == "" {
		c.Status = service.StatusUnused
	}
	if c.Type == "" {
		c.Type = service.RedeemTypeBalance
	}
	if c.Code == "" {
		c.Code = "rc-" + time.Now().Format("150405.000000")
	}

	create := client.RedeemCode.Create().
		SetCode(c.Code).
		SetType(c.Type).
		SetValue(c.Value).
		SetStatus(c.Status).
		SetNotes(c.Notes).
		SetValidityDays(c.ValidityDays)
	if c.UsedBy != nil {
		create.SetUsedBy(*c.UsedBy)
	}
	if c.UsedAt != nil {
		create.SetUsedAt(*c.UsedAt)
	}
	if c.GroupID != nil {
		create.SetGroupID(*c.GroupID)
	}
	if !c.CreatedAt.IsZero() {
		create.SetCreatedAt(c.CreatedAt)
	}

	created, err := create.Save(ctx)
	require.NoError(t, err, "create redeem code")

	c.ID = created.ID
	c.CreatedAt = created.CreatedAt
	return c
}

func mustCreateSubscription(t *testing.T, client *dbent.Client, s *service.UserSubscription) *service.UserSubscription {
	t.Helper()
	ctx := context.Background()

	if s.Status == "" {
		s.Status = service.SubscriptionStatusActive
	}
	now := time.Now()
	if s.StartsAt.IsZero() {
		s.StartsAt = now.Add(-1 * time.Hour)
	}
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = now.Add(24 * time.Hour)
	}
	if s.AssignedAt.IsZero() {
		s.AssignedAt = now
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}

	create := client.UserSubscription.Create().
		SetUserID(s.UserID).
		SetGroupID(s.GroupID).
		SetStartsAt(s.StartsAt).
		SetExpiresAt(s.ExpiresAt).
		SetStatus(s.Status).
		SetAssignedAt(s.AssignedAt).
		SetNotes(s.Notes).
		SetDailyUsageUsd(s.DailyUsageUSD).
		SetWeeklyUsageUsd(s.WeeklyUsageUSD).
		SetMonthlyUsageUsd(s.MonthlyUsageUSD)

	if s.AssignedBy != nil {
		create.SetAssignedBy(*s.AssignedBy)
	}
	if !s.CreatedAt.IsZero() {
		create.SetCreatedAt(s.CreatedAt)
	}
	if !s.UpdatedAt.IsZero() {
		create.SetUpdatedAt(s.UpdatedAt)
	}

	created, err := create.Save(ctx)
	require.NoError(t, err, "create user subscription")

	s.ID = created.ID
	s.CreatedAt = created.CreatedAt
	s.UpdatedAt = created.UpdatedAt
	return s
}
