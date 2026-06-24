//go:build unit

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSocialAccountFieldModelKeepsAuthCookieAsFirstClassCredential(t *testing.T) {
	serviceSource := mustReadSource(t, "social_account_service.go")
	schemaSource := mustReadSource(t, "../../ent/schema/social_account.go")
	migrationSource, err := os.ReadFile("../../migrations/160_social_account_identity_model.sql")
	require.NoError(t, err)
	fieldMigrationSource, err := os.ReadFile("../../migrations/159_social_account_field_model.sql")
	require.NoError(t, err)
	restoreMigrationSource, err := os.ReadFile("../../migrations/161_restore_social_account_auth_cookie.sql")
	require.NoError(t, err)
	removedPlatformPlaceholder := "Leg" + "acy platform"

	for _, forbidden := range []string{
		"BoundIP",
		"`json:\"account_id\"`",
		"`json:\"bound_ip\"`",
		"SetAccountID",
		"SetBoundIP",
		"ClearAccountID",
		"ClearBoundIP",
		"firstTrimmed(input.ExecutionAuth",
		"firstTrimmed(e.ExecutionAuth",
		removedPlatformPlaceholder,
		"use default_proxy_snapshot",
	} {
		require.NotContains(t, serviceSource, forbidden)
		require.NotContains(t, schemaSource, forbidden)
	}
	for _, forbidden := range []string{
		`field.String("account_id")`,
		`field.String("bound_ip")`,
	} {
		require.NotContains(t, schemaSource, forbidden)
	}

	require.Contains(t, serviceSource, "AuthCookie")
	require.Contains(t, serviceSource, "`json:\"auth_cookie\"`")
	require.Contains(t, serviceSource, "SetAuthCookie")
	require.Contains(t, serviceSource, "ClearAuthCookie")
	require.Contains(t, schemaSource, `field.Text("auth_cookie")`)
	require.Contains(t, schemaSource, `field.String("identity_kind")`)
	require.Contains(t, schemaSource, `field.String("identity_key")`)
	require.Contains(t, string(migrationSource), "DROP COLUMN IF EXISTS account_id")
	require.Contains(t, string(migrationSource), "DROP COLUMN IF EXISTS bound_ip")
	require.NotContains(t, string(migrationSource), "DROP COLUMN IF EXISTS auth_cookie")
	require.NotContains(t, string(fieldMigrationSource), "SET execution_auth = auth_cookie")
	require.Contains(t, string(migrationSource), "auth_cookie is retained as")
	require.Contains(t, string(restoreMigrationSource), "ADD COLUMN IF NOT EXISTS auth_cookie TEXT")
	require.Contains(t, string(restoreMigrationSource), "explicit account credential field")
}

func TestSocialAccountBusinessIdentityIgnoresTwitterPlatformUserID(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)
	platformID := "44196397"

	created, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:           "@Jack",
		Platform:       "twitter",
		PlatformUserID: &platformID,
		Password:       socialIdentityStringPtr("account-secret"),
		TwoFactor:      socialIdentityStringPtr("totp-secret"),
	})
	require.NoError(t, err)
	require.Equal(t, "username", created.IdentityKind)
	require.Equal(t, "jack", created.IdentityKey)
	require.Equal(t, platformID, requireStringPtr(t, created.PlatformUserID))

	_, err = svc.Create(ctx, &CreateSocialAccountInput{
		Name:           " JACK ",
		Platform:       "x",
		PlatformUserID: socialIdentityStringPtr("99999999"),
		Password:       socialIdentityStringPtr("account-secret"),
		TwoFactor:      socialIdentityStringPtr("totp-secret"),
	})
	require.ErrorIs(t, err, ErrSocialAccountDuplicate)

	createdAlt, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:           "@jack_alt",
		Platform:       "x",
		PlatformUserID: &platformID,
		Password:       socialIdentityStringPtr("account-secret"),
		TwoFactor:      socialIdentityStringPtr("totp-secret"),
	})
	require.NoError(t, err)
	require.Equal(t, "jack_alt", createdAlt.IdentityKey)

	stored := client.SocialAccount.Query().Where(socialaccount.IDEQ(created.ID)).OnlyX(ctx)
	require.Equal(t, "x_twitter", stored.PlatformKey)
	require.Equal(t, "username", stored.IdentityKind)
	require.Equal(t, "jack", stored.IdentityKey)
	require.Equal(t, platformID, requireStringPtr(t, stored.PlatformUserID))
}

func TestSocialAccountBusinessIdentityFallsBackToNormalizedUsername(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)

	created, err := svc.Create(ctx, &CreateSocialAccountInput{Name: "  @CaseMix  ", Platform: "X/Twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.NoError(t, err)
	require.Equal(t, "username", created.IdentityKind)
	require.Equal(t, "casemix", created.IdentityKey)

	_, err = svc.Create(ctx, &CreateSocialAccountInput{Name: "casemix", Platform: "x_twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.ErrorIs(t, err, ErrSocialAccountDuplicate)
}

func TestSocialAccountBusinessIdentityIgnoresMetadataForUniqueness(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)
	platformID := "cookie-independent-platform-id"
	firstCookie := "ct0=first; auth_token=first"
	secondCookie := "ct0=second; auth_token=second"

	created, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:           "@cookie_identity",
		Platform:       "x_twitter",
		PlatformUserID: &platformID,
		Password:       socialIdentityStringPtr("account-secret"),
		AuthCookie:     &firstCookie,
	})
	require.NoError(t, err)
	require.Equal(t, firstCookie, requireStringPtr(t, created.AuthCookie))

	_, err = svc.Create(ctx, &CreateSocialAccountInput{
		Name:           " cookie_identity ",
		Platform:       "twitter",
		PlatformUserID: socialIdentityStringPtr("different-platform-id"),
		Password:       socialIdentityStringPtr("account-secret"),
		AuthCookie:     &secondCookie,
	})
	require.ErrorIs(t, err, ErrSocialAccountDuplicate)

	count, err := client.SocialAccount.Query().
		Where(socialaccount.PlatformKeyEQ("x_twitter"), socialaccount.NameKeyEQ("cookie_identity")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	stored := client.SocialAccount.GetX(ctx, created.ID)
	require.Equal(t, firstCookie, requireStringPtr(t, stored.AuthCookie))
}

func TestSocialAccountBatchImportReportsDuplicatesFailuresAndItems(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)
	_, err := svc.Create(ctx, &CreateSocialAccountInput{Name: "@existing", Platform: "x_twitter", PlatformUserID: socialIdentityStringPtr("existing-twitter-id"), Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.NoError(t, err)

	result, err := svc.ImportPoolAccounts(ctx, []*CreateSocialAccountInput{
		{Name: "@new_one", Platform: "x_twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")},
		{Name: "@new_one", Platform: "twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")},
		{Name: " existing ", Platform: "x", PlatformUserID: socialIdentityStringPtr("different-id"), Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")},
		{Name: "", Platform: "x_twitter"},
	})
	require.NoError(t, err)

	require.Equal(t, 4, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 3, result.Skipped)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, 2, result.Duplicates)
	require.Len(t, result.Items, 4)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	body := string(raw)
	require.Contains(t, body, `"succeeded":1`)
	require.Contains(t, body, `"duplicates":2`)
	require.Contains(t, body, `"items"`)
	require.Contains(t, body, `"duplicate_in_batch"`)
	require.Contains(t, body, `"duplicate_in_database"`)
}

func TestSocialAccountDeleteHardDeletesAccountWithTaskLogs(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("delete-social-account-with-log@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account, err := svc.Create(ctx, &CreateSocialAccountInput{Name: "@delete_with_log", Platform: "x_twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.NoError(t, err)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLoginCheck).
		SetStatus(SocialTaskLogStatusSuccess).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	ledgerRequestID := socialTaskUsageLedgerRequestID(log.ID, SocialTaskChargeSourceWallet, 0)
	ledger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(ledgerRequestID).
		SetModel(socialUsageLedgerModel).
		SetActualCost(0.1).
		SetTotalCost(0.1).
		SetBillingType(socialUsageBillingTypeWallet).
		SaveX(ctx)

	require.NoError(t, svc.Delete(ctx, account.ID))

	_, err = client.SocialAccount.Get(ctx, account.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), account.ID)
	require.True(t, dbent.IsNotFound(err), "deleted account should be physically removed")
	logExists, err := client.SocialTaskLog.Query().
		Where(socialtasklog.IDEQ(log.ID), socialtasklog.SocialAccountIDEQ(account.ID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, logExists)
	ledgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(ledger.ID), usagelog.RequestIDEQ(ledgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, ledgerExists)
}

func TestSocialAccountConcurrentDuplicateCreateUsesDatabaseConstraint(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(ctx, &CreateSocialAccountInput{
				Name:      "@concurrent",
				Platform:  "x_twitter",
				Password:  socialIdentityStringPtr("account-secret"),
				TwoFactor: socialIdentityStringPtr("totp-secret"),
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	duplicates := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if strings.Contains(err.Error(), "database table is locked") {
			t.Fatalf("sqlite locking hid the uniqueness behavior: %v", err)
		}
		require.ErrorIs(t, err, ErrSocialAccountDuplicate)
		duplicates++
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, duplicates)
	count, err := client.SocialAccount.Query().
		Where(socialaccount.PlatformKeyEQ("x_twitter"), socialaccount.NameKeyEQ("concurrent")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSocialAccountRestIDUpdateDoesNotChangeIdentity(t *testing.T) {
	client := newSocialAccountIdentityTestClient(t)
	ctx := context.Background()
	svc := NewSocialAccountService(client)

	created, err := svc.Create(ctx, &CreateSocialAccountInput{Name: "@rest_id_user", Platform: "x_twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.ID, &UpdateSocialAccountInput{PlatformUserID: socialIdentityStringPtr("44196397")})
	require.NoError(t, err)
	require.Equal(t, "username", updated.IdentityKind)
	require.Equal(t, "rest_id_user", updated.IdentityKey)
	require.Equal(t, "44196397", requireStringPtr(t, updated.PlatformUserID))

	_, err = svc.Create(ctx, &CreateSocialAccountInput{Name: " REST_ID_USER ", Platform: "twitter", Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.ErrorIs(t, err, ErrSocialAccountDuplicate)

	other, err := svc.Create(ctx, &CreateSocialAccountInput{Name: "@rest_id_other", Platform: "twitter", PlatformUserID: socialIdentityStringPtr("44196397"), Password: socialIdentityStringPtr("account-secret"), TwoFactor: socialIdentityStringPtr("totp-secret")})
	require.NoError(t, err)
	require.Equal(t, "rest_id_other", other.IdentityKey)
}

func newSocialAccountIdentityTestClient(t *testing.T) *dbent.Client {
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

func requireStringPtr(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func socialIdentityStringPtr(value string) *string {
	return &value
}
