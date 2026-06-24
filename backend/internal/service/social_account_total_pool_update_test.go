package service

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSocialAccountServiceUpdateTotalPoolIgnoresIdentityFields(t *testing.T) {
	ctx := context.Background()
	client := newSocialAccountTotalPoolUpdateTestClient(t)
	svc := NewSocialAccountService(client)

	account := client.SocialAccount.Create().
		SetName("@total_service_update").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_service_update").
		SetIdentityKind("username").
		SetIdentityKey("total_service_update").
		SetPlatformUserID("immutable-platform-id").
		SetPassword("old-password").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	newName := "@should_not_change"
	newPlatformUserID := "should-not-change"
	newPassword := "new-password"
	newRemark := "updated through service"
	updated, err := svc.UpdateTotalPool(ctx, account.ID, &UpdateSocialAccountInput{
		Name:           &newName,
		PlatformUserID: &newPlatformUserID,
		Password:       &newPassword,
		Remark:         &newRemark,
	})

	require.NoError(t, err)
	require.Equal(t, "@total_service_update", updated.Name)
	require.Equal(t, "immutable-platform-id", requireTotalPoolUpdateStringPtr(t, updated.PlatformUserID))
	require.Equal(t, "new-password", requireTotalPoolUpdateStringPtr(t, updated.Password))
	require.Equal(t, "updated through service", requireTotalPoolUpdateStringPtr(t, updated.Remark))

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@total_service_update", stored.Name)
	require.Equal(t, "total_service_update", stored.NameKey)
	require.Equal(t, "username", stored.IdentityKind)
	require.Equal(t, "total_service_update", stored.IdentityKey)
	require.Equal(t, "immutable-platform-id", requireTotalPoolUpdateStringPtr(t, stored.PlatformUserID))
	require.Equal(t, "new-password", requireTotalPoolUpdateStringPtr(t, stored.Password))
	require.Equal(t, "updated through service", requireTotalPoolUpdateStringPtr(t, stored.Remark))
}

func newSocialAccountTotalPoolUpdateTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func requireTotalPoolUpdateStringPtr(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}
