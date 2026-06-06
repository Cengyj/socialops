package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newUserSubscriptionRepoSQLite(t *testing.T) (*userSubscriptionRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewUserSubscriptionRepository(client).(*userSubscriptionRepository), client
}

func TestUserSubscriptionRepositoryPreloadsGroupAndFiltersByPlatform(t *testing.T) {
	ctx := context.Background()
	repo, client := newUserSubscriptionRepoSQLite(t)

	user := client.User.Create().
		SetEmail("subscription-platform@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)

	xGroup := client.Group.Create().
		SetName("X Starter").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetRateMultiplier(1.2).
		SaveX(ctx)
	instagramGroup := client.Group.Create().
		SetName("Instagram Starter").
		SetPlatform("instagram").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetRateMultiplier(1.5).
		SaveX(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	xSub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(xGroup.ID).
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("x").
		SaveX(ctx)
	client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(instagramGroup.ID).
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("ig").
		SaveX(ctx)

	got, err := repo.GetByID(ctx, xSub.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Group)
	require.Equal(t, "x_twitter", got.Group.Platform)
	require.Equal(t, service.SubscriptionTypeSubscription, got.Group.SubscriptionType)

	items, page, err := repo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, nil, nil, nil, "", "x_twitter", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, items, 1)
	require.Equal(t, xSub.ID, items[0].ID)
	require.NotNil(t, items[0].Group)
	require.Equal(t, "X Starter", items[0].Group.Name)
}

func TestUserSubscriptionRepositoryPlatformFilterUsesPlanSnapshotBeforeGroup(t *testing.T) {
	ctx := context.Background()
	repo, client := newUserSubscriptionRepoSQLite(t)

	user := client.User.Create().
		SetEmail("subscription-platform-snapshot@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)

	xGroup := client.Group.Create().
		SetName("X Pool").
		SetPlatform("x_twitter").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)
	instagramGroup := client.Group.Create().
		SetName("Instagram Pool").
		SetPlatform("instagram").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	planXOnInstagramGroup := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(instagramGroup.ID).
		SetPlanPlatform("x_twitter").
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("plan snapshot x").
		SaveX(ctx)
	planInstagramOnXGroup := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(xGroup.ID).
		SetPlanPlatform("instagram").
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("plan snapshot instagram").
		SaveX(ctx)
	legacyXGroup := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(xGroup.ID).
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("legacy group platform").
		SaveX(ctx)

	items, page, err := repo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, nil, nil, nil, "", "twitter", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(2), page.Total)
	gotIDs := make([]int64, 0, len(items))
	for _, item := range items {
		gotIDs = append(gotIDs, item.ID)
	}
	require.ElementsMatch(t, []int64{planXOnInstagramGroup.ID, legacyXGroup.ID}, gotIDs)

	items, page, err = repo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, nil, nil, nil, "", "instagram", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, items, 1)
	require.Equal(t, planInstagramOnXGroup.ID, items[0].ID)
}

func TestUserSubscriptionRepositoryPlatformFilterIncludesLegacyPlanPlatformAliases(t *testing.T) {
	ctx := context.Background()
	repo, client := newUserSubscriptionRepoSQLite(t)

	user := client.User.Create().
		SetEmail("subscription-platform-legacy@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SaveX(ctx)

	instagramGroup := client.Group.Create().
		SetName("Instagram Pool").
		SetPlatform("instagram").
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SaveX(ctx)

	now := time.Now().UTC().Truncate(time.Second)
	legacyTwitterSnapshot := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(instagramGroup.ID).
		SetPlanPlatform("twitter").
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("legacy twitter snapshot").
		SaveX(ctx)
	legacyXSnapshot := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(instagramGroup.ID).
		SetPlanPlatform("x").
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("legacy x snapshot").
		SaveX(ctx)
	client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(instagramGroup.ID).
		SetPlanPlatform("instagram").
		SetStartsAt(now).
		SetExpiresAt(now.AddDate(0, 0, 30)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetNotes("instagram snapshot").
		SaveX(ctx)

	items, page, err := repo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, nil, nil, nil, "", "x_twitter", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(2), page.Total)
	gotIDs := make([]int64, 0, len(items))
	for _, item := range items {
		gotIDs = append(gotIDs, item.ID)
	}
	require.ElementsMatch(t, []int64{legacyTwitterSnapshot.ID, legacyXSnapshot.ID}, gotIDs)
}
