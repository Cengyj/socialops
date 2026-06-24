//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/apikey"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/usersubscription"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"
)

func uniqueSoftDeleteValue(t *testing.T, prefix string) string {
	t.Helper()
	safeName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	return fmt.Sprintf("%s-%s", prefix, safeName)
}

func createEntUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()

	u, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		Save(ctx)
	require.NoError(t, err, "create ent user")
	return u
}

func TestEntSoftDelete_ApiKey_DefaultFilterAndSkip(t *testing.T) {
	ctx := context.Background()
	// 使用全局 ent client，确保软删除验证在实际持久化数据上进行。
	client := testEntClient(t)

	u := createEntUser(t, ctx, client, uniqueSoftDeleteValue(t, "sd-user")+"@example.com")

	key := mustCreateHistoricalAPIKey(t, client, historicalAPIKeyRow{
		UserID: u.ID,
		Key:    uniqueSoftDeleteValue(t, "sk-soft-delete"),
		Name:   "soft-delete",
	})

	require.NoError(t, client.APIKey.DeleteOneID(key.ID).Exec(ctx), "soft delete historical api key row")

	_, err := client.APIKey.Query().Where(apikey.IDEQ(key.ID)).Only(ctx)
	require.Error(t, err, "default ent query should not see soft-deleted rows")
	require.True(t, dbent.IsNotFound(err), "expected ent not-found after default soft delete filter")

	got, err := client.APIKey.Query().
		Where(apikey.IDEQ(key.ID)).
		Only(mixins.SkipSoftDelete(ctx))
	require.NoError(t, err, "SkipSoftDelete should include soft-deleted rows")
	require.NotNil(t, got.DeletedAt, "deleted_at should be set after soft delete")
}

func TestEntSoftDelete_ApiKey_HardDeleteViaSkipSoftDelete(t *testing.T) {
	ctx := context.Background()
	// 使用全局 ent client，确保 SkipSoftDelete 的硬删除语义可验证。
	client := testEntClient(t)

	u := createEntUser(t, ctx, client, uniqueSoftDeleteValue(t, "sd-user3")+"@example.com")

	key := mustCreateHistoricalAPIKey(t, client, historicalAPIKeyRow{
		UserID: u.ID,
		Key:    uniqueSoftDeleteValue(t, "sk-soft-delete3"),
		Name:   "soft-delete3",
	})

	require.NoError(t, client.APIKey.DeleteOneID(key.ID).Exec(ctx), "soft delete historical api key row")

	// Hard delete using SkipSoftDelete so the hook doesn't convert it to update-deleted_at.
	_, err := client.APIKey.Delete().Where(apikey.IDEQ(key.ID)).Exec(mixins.SkipSoftDelete(ctx))
	require.NoError(t, err, "hard delete")

	_, err = client.APIKey.Query().
		Where(apikey.IDEQ(key.ID)).
		Only(mixins.SkipSoftDelete(ctx))
	require.True(t, dbent.IsNotFound(err), "expected row to be hard deleted")
}

// --- UserSubscription 软删除测试 ---

func createEntGroup(t *testing.T, ctx context.Context, client *dbent.Client, name string) *dbent.Group {
	t.Helper()

	g, err := client.Group.Create().
		SetName(name).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err, "create ent group")
	return g
}

func TestEntSoftDelete_UserSubscription_DefaultFilterAndSkip(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)

	u := createEntUser(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-user")+"@example.com")
	g := createEntGroup(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-group"))

	repo := NewUserSubscriptionRepository(client)
	sub := &service.UserSubscription{
		UserID:    u.ID,
		GroupID:   g.ID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, sub), "create user subscription")

	require.NoError(t, repo.Delete(ctx, sub.ID), "soft delete user subscription")

	_, err := repo.GetByID(ctx, sub.ID)
	require.Error(t, err, "deleted rows should be hidden by default")

	_, err = client.UserSubscription.Query().Where(usersubscription.IDEQ(sub.ID)).Only(ctx)
	require.Error(t, err, "default ent query should not see soft-deleted rows")
	require.True(t, dbent.IsNotFound(err), "expected ent not-found after default soft delete filter")

	got, err := client.UserSubscription.Query().
		Where(usersubscription.IDEQ(sub.ID)).
		Only(mixins.SkipSoftDelete(ctx))
	require.NoError(t, err, "SkipSoftDelete should include soft-deleted rows")
	require.NotNil(t, got.DeletedAt, "deleted_at should be set after soft delete")
}

func TestEntSoftDelete_UserSubscription_DeleteIdempotent(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)

	u := createEntUser(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-user2")+"@example.com")
	g := createEntGroup(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-group2"))

	repo := NewUserSubscriptionRepository(client)
	sub := &service.UserSubscription{
		UserID:    u.ID,
		GroupID:   g.ID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, sub), "create user subscription")

	require.NoError(t, repo.Delete(ctx, sub.ID), "first delete")
	require.NoError(t, repo.Delete(ctx, sub.ID), "second delete should be idempotent")
}

func TestEntSoftDelete_UserSubscription_ListExcludesDeleted(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)

	u := createEntUser(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-user3")+"@example.com")
	g1 := createEntGroup(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-group3a"))
	g2 := createEntGroup(t, ctx, client, uniqueSoftDeleteValue(t, "sd-sub-group3b"))

	repo := NewUserSubscriptionRepository(client)

	deletedSubscription := &service.UserSubscription{
		UserID:    u.ID,
		GroupID:   g1.ID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, deletedSubscription), "create subscription to delete")

	activeSubscription := &service.UserSubscription{
		UserID:    u.ID,
		GroupID:   g2.ID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, activeSubscription), "create subscription to keep")

	require.NoError(t, repo.Delete(ctx, deletedSubscription.ID), "soft delete one subscription")

	// ListByUserID 应只返回未删除的订阅
	subs, err := repo.ListByUserID(ctx, u.ID)
	require.NoError(t, err, "ListByUserID")
	require.Len(t, subs, 1, "should only return non-deleted subscriptions")
	require.Equal(t, activeSubscription.ID, subs[0].ID, "expected active subscription to be returned")
}
