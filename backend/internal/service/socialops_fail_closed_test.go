//go:build unit

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestAdminServiceSkeletonFailsClosed(t *testing.T) {
	svc := NewAdminServiceSkeleton()

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: "user@example.com"})
	require.Nil(t, user)
	require.ErrorIs(t, err, ErrAdminServiceNotConfigured)
	require.Equal(t, http.StatusNotImplemented, infraerrors.Code(err))

	code, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{Count: 1})
	require.Nil(t, code)
	require.ErrorIs(t, err, ErrAdminServiceNotConfigured)

	updated, err := svc.AdminUpdateAPIKeyGroupID(context.Background(), 1, nil)
	require.Nil(t, updated)
	require.ErrorIs(t, err, ErrAdminServiceNotConfigured)
}

func TestSubscriptionServiceUsesRepositoryForExposedOperations(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := &subscriptionRepoState{
		sub: &UserSubscription{
			ID:        7,
			UserID:    11,
			GroupID:   13,
			StartsAt:  now.Add(-24 * time.Hour),
			ExpiresAt: now.AddDate(0, 0, 10),
			Status:    SubscriptionStatusActive,
		},
	}
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)

	items, err := svc.ListUserSubscriptions(context.Background(), 11)
	require.NoError(t, err)
	require.Len(t, items, 1)

	extended, err := svc.ExtendSubscription(context.Background(), 7, -3)
	require.NoError(t, err)
	require.Equal(t, now.AddDate(0, 0, 7), extended.ExpiresAt)

	err = svc.RevokeSubscription(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusRevoked, repo.sub.Status)

	err = svc.ResetSubscriptionQuota(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, repo.sub.DailyWindowStart)
	require.Zero(t, repo.sub.DailyUsageUSD)
}

func TestSubscriptionServiceRejectsAdjustmentThatWouldExpire(t *testing.T) {
	now := time.Now().UTC()
	repo := &subscriptionRepoState{
		sub: &UserSubscription{
			ID:        7,
			UserID:    11,
			GroupID:   13,
			StartsAt:  now.Add(-24 * time.Hour),
			ExpiresAt: now.Add(24 * time.Hour),
			Status:    SubscriptionStatusActive,
		},
	}
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)

	extended, err := svc.ExtendSubscription(context.Background(), 7, -2)
	require.Nil(t, extended)
	require.ErrorIs(t, err, ErrAdjustWouldExpire)
}

func TestSocialTaskExecutorDoesNotMarkUnimplementedActionsSuccessful(t *testing.T) {
	target := "target"
	content := "content"
	task := &dbent.SocialTaskLog{Target: &target, Content: &content}
	executor := &SocialTaskExecutor{}

	for _, action := range []string{
		SocialTaskActionLoginCheck,
		SocialTaskActionFollow,
		SocialTaskActionMessage,
		SocialTaskActionPost,
		SocialTaskActionLike,
	} {
		t.Run(action, func(t *testing.T) {
			task.Action = action
			result, err := executor.executeAction(context.Background(), task)
			require.Empty(t, result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not configured")
		})
	}

	task.Action = "edit_ip"
	result, err := executor.executeAction(context.Background(), task)
	require.Empty(t, result)
	require.ErrorContains(t, err, "unsupported action")
}

func TestSocialTaskExecutorProcessesPendingTaskFailClosedWithoutCharge(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-fail-closed@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})

	executor.processTask(log.ID)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.ChargeSource)
	require.NotNil(t, stored.ResultMessage)
	require.Contains(t, *stored.ResultMessage, "not configured")

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestValidateProxyEndpointBlocksLocalAndPrivateTargets(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:8080",
		"http://10.0.0.1:8080",
		"http://169.254.169.254:80",
		"http://localhost:8080",
		"socks5://[::1]:1080",
	} {
		t.Run(raw, func(t *testing.T) {
			parsed, err := url.Parse(raw)
			require.NoError(t, err)
			require.Error(t, validateProxyEndpoint(parsed))
		})
	}

	parsed, err := url.Parse("http://8.8.8.8:8080")
	require.NoError(t, err)
	require.NoError(t, validateProxyEndpoint(parsed))

	parsed, err = url.Parse("http://8.8.8.8")
	require.NoError(t, err)
	require.ErrorContains(t, validateProxyEndpoint(parsed), "port is required")
}

func TestSocialAccountJSONIncludesCredentials(t *testing.T) {
	password := "secret"
	emailPassword := "mail-secret"
	account := SocialAccount{
		ID:            1,
		Name:          "x account",
		Platform:      "x",
		Password:      &password,
		EmailPassword: &emailPassword,
	}

	payload, err := json.Marshal(account)
	require.NoError(t, err)
	body := string(payload)
	require.Contains(t, body, `"password":"secret"`)
	require.Contains(t, body, `"email_password":"mail-secret"`)
}

func TestSocialAccountServiceStoresCredentialsWithoutApplicationEncryption(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	password := "x-account-secret"
	emailPassword := "mailbox-secret"
	directProxySnapshot := `{"id":99,"name":"stale-proxy"}`
	account, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:          "northwind_ops",
		Platform:      "x_twitter",
		Password:      &password,
		EmailPassword: &emailPassword,
		Source:        SocialAccountSourceManualImport,
		BoundIP:       &directProxySnapshot,
	})
	require.NoError(t, err)
	require.NotNil(t, account.Password)
	require.NotNil(t, account.EmailPassword)
	require.Nil(t, account.BoundIP)
	require.Equal(t, password, *account.Password)
	require.Equal(t, emailPassword, *account.EmailPassword)

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Password)
	require.NotNil(t, stored.EmailPassword)
	require.Nil(t, stored.BoundIP)
	require.Equal(t, password, *stored.Password)
	require.Equal(t, emailPassword, *stored.EmailPassword)

	updatedPassword := "rotated-secret"
	updated, err := svc.Update(ctx, account.ID, &UpdateSocialAccountInput{Password: &updatedPassword})
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *updated.Password)

	stored, err = client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *stored.Password)

	_, err = svc.Update(ctx, account.ID, &UpdateSocialAccountInput{BoundIP: &directProxySnapshot})
	require.ErrorIs(t, err, ErrSocialAccountDefaultProxyRoute)
}

func TestSocialAccountServiceImportForUserMatchesExistingPoolOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("social-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	password := "pool-secret"
	poolAccount, err := client.SocialAccount.Create().
		SetName("@NorthWind_Ops").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("northwind_ops").
		SetPassword(password).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		Save(ctx)
	require.NoError(t, err)

	imported, err := svc.ImportForUser(ctx, user.ID, &UserImportSocialAccountInput{
		Platform: "x_twitter",
		Name:     "northwind_ops",
	})
	require.NoError(t, err)
	require.Equal(t, int64(poolAccount.ID), imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)
	require.Equal(t, password, *imported.Password)

	count, err := client.SocialAccount.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = svc.ImportForUser(ctx, user.ID, &UserImportSocialAccountInput{
		Platform: "x_twitter",
		Name:     "@northwind_ops",
	})
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	_, err = svc.ImportForUser(ctx, user.ID, &UserImportSocialAccountInput{
		Platform: "x_twitter",
		Name:     "missing_user",
	})
	require.ErrorIs(t, err, ErrSocialAccountImportNotFound)
}

func TestSocialAccountServiceImportRejectsAmbiguousUsername(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("ambiguous-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	_, err := client.SocialAccount.Create().
		SetName("@same_name").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("same_name").
		Save(ctx)
	require.NoError(t, err)
	_, err = client.SocialAccount.Create().
		SetName("same_name").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("same_name").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.ImportForUser(ctx, user.ID, &UserImportSocialAccountInput{
		Name: "@same_name",
	})
	require.ErrorIs(t, err, ErrSocialAccountImportAmbiguous)
}

func TestSocialAccountServiceImportRejectsPlatformlessAmbiguousAssignedMatches(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("platformless-ambiguous-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("platformless-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	_, err := client.SocialAccount.Create().
		SetName("@shared_name").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("shared_name").
		SetAssignedUserID(otherUser.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.SocialAccount.Create().
		SetName("shared_name").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("shared_name").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.ImportForUser(ctx, user.ID, &UserImportSocialAccountInput{
		Name: "@shared_name",
	})
	require.ErrorIs(t, err, ErrSocialAccountImportAmbiguous)
}

func TestSocialAccountServiceAdminImportDedupesPoolByNormalizedPlatformName(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	_, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:     "@NorthWind_Ops",
		Platform: "x_twitter",
		Source:   SocialAccountSourceManualImport,
	})
	require.NoError(t, err)

	importedBoundIP := `{"id":123,"name":"must-not-persist"}`
	result, err := svc.ImportPoolAccounts(ctx, []*CreateSocialAccountInput{
		{Name: "northwind_ops", Platform: "X_Twitter", Source: SocialAccountSourceFileUpload},
		{Name: "@fresh_ops", Platform: "x_twitter", Source: SocialAccountSourceFileUpload, BoundIP: &importedBoundIP},
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 1, result.Created)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.Errors, 0)

	count, err := client.SocialAccount.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("fresh_ops")).
		OnlyX(ctx)
	require.Nil(t, imported.BoundIP)
}

func TestSocialAccountSchemaEnforcesNormalizedPoolUniqueness(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	client.SocialAccount.Create().
		SetName("@Unique_Name").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("unique_name").
		SaveX(ctx)

	_, err := client.SocialAccount.Create().
		SetName("unique_name").
		SetPlatform("X_Twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("unique_name").
		Save(ctx)

	require.Error(t, err)
	require.True(t, dbent.IsConstraintError(err), "expected normalized pool unique constraint, got %v", err)
}

func TestSocialAccountAssignAndReclaimRespectPoolOwnership(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user1 := client.User.Create().SetEmail("u1@example.com").SetPasswordHash("hash").SaveX(ctx)
	user2 := client.User.Create().SetEmail("u2@example.com").SetPasswordHash("hash").SaveX(ctx)
	boundIP := "proxy-one"
	account := client.SocialAccount.Create().
		SetName("assign_me").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("assign_me").
		SetBoundIP(boundIP).
		SaveX(ctx)

	assigned, err := svc.Assign(ctx, account.ID, user1.ID)
	require.NoError(t, err)
	require.Equal(t, user1.ID, *assigned.AssignedUserID)

	_, err = svc.Assign(ctx, account.ID, user2.ID)
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	reclaimed, err := svc.Reclaim(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, reclaimed.AssignedUserID)
	require.Nil(t, reclaimed.BoundIP)
}

func TestSocialAccountAssignIsConditionalUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user1 := client.User.Create().SetEmail("race-u1@example.com").SetPasswordHash("hash").SaveX(ctx)
	user2 := client.User.Create().SetEmail("race-u2@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("race_assign").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("race_assign").
		SaveX(ctx)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, userID := range []int64{user1.ID, user2.ID} {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			_, err := svc.Assign(ctx, account.ID, uid)
			results <- err
		}(userID)
	}
	wg.Wait()
	close(results)

	var successes, conflicts int
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)
		conflicts++
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.AssignedUserID)
	require.Contains(t, []int64{user1.ID, user2.ID}, *stored.AssignedUserID)
}

func TestSocialAccountDefaultProxyRequiresAccountAndProxyOwnership(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user1 := client.User.Create().SetEmail("proxy-account-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	user2 := client.User.Create().SetEmail("proxy-other-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("default_proxy_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("default_proxy_account").
		SetAssignedUserID(user1.ID).
		SaveX(ctx)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "proxy", IPType: "residential"})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	otherIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user2.ID, Name: "other-proxy", IPType: "residential"})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(otherIP.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	otherIP, err = ipSvc.GetByID(ctx, otherIP.ID)
	require.NoError(t, err)
	untestedIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "untested-proxy", IPType: "residential"})
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(ip)

	updated, err := accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, ip)
	require.NoError(t, err)
	require.NotNil(t, updated.BoundIP)
	require.Equal(t, snapshot, *updated.BoundIP)
	proxyID, ok := SocialIPIDFromSnapshot(*updated.BoundIP)
	require.True(t, ok)
	require.Equal(t, ip.ID, proxyID)

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, otherIP)
	require.ErrorIs(t, err, ErrSocialIPOwnerMismatch)

	_, err = accountSvc.SetDefaultProxyForAdmin(ctx, account.ID, otherIP)
	require.ErrorIs(t, err, ErrSocialIPOwnerMismatch)

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, untestedIP)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user2.ID, ip)
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)

	cleared, err := accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, nil)
	require.NoError(t, err)
	require.Nil(t, cleared.BoundIP)
}

func TestSocialAccountDefaultProxyAllowsOneProxyForMultipleSameUserAccounts(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().
		SetEmail("shared-proxy-owner@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	firstAccount := client.SocialAccount.Create().
		SetName("shared_proxy_account_one").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("shared_proxy_account_one").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	secondAccount := client.SocialAccount.Create().
		SetName("shared_proxy_account_two").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("shared_proxy_account_two").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "shared proxy", IPType: "residential"})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).
		SetStatus(SocialIPStatusOnline).
		SetBoundSocialAccountID(firstAccount.ID).
		SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)

	firstUpdated, err := accountSvc.SetDefaultProxyForUser(ctx, firstAccount.ID, user.ID, ip)
	require.NoError(t, err)
	secondUpdated, err := accountSvc.SetDefaultProxyForUser(ctx, secondAccount.ID, user.ID, ip)
	require.NoError(t, err)

	require.NotNil(t, firstUpdated.BoundIP)
	require.NotNil(t, secondUpdated.BoundIP)
	require.Equal(t, *firstUpdated.BoundIP, *secondUpdated.BoundIP)
	proxyID, ok := SocialIPIDFromSnapshot(*secondUpdated.BoundIP)
	require.True(t, ok)
	require.Equal(t, ip.ID, proxyID)
}

func TestSocialAccountDefaultProxySnapshotAllowsLongEndpoint(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().SetEmail("long-proxy-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("long_proxy_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("long_proxy_account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	endpoint := "http://user:" + strings.Repeat("p", 320) + "@8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "long endpoint proxy",
		IPType:   "residential",
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)

	updated, err := accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, ip)
	require.NoError(t, err)
	require.NotNil(t, updated.BoundIP)
	require.Greater(t, len(*updated.BoundIP), 255)
	require.Contains(t, *updated.BoundIP, endpoint)
}

func TestSocialTaskLogCapturesBillingAndProxySnapshot(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("task-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("task_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_account").
		SaveX(ctx)
	target := "@target"
	content := "hello"
	proxyID := int64(42)
	proxySnapshot := `{"endpoint":"socks5://proxy.example:1080"}`
	idempotencyKey := "req-123"
	message := "social platform executor is not configured"

	log, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionMessage,
		Target:         &target,
		Content:        &content,
		Status:         SocialTaskLogStatusFailed,
		ResultMessage:  &message,
		ProxyID:        &proxyID,
		ProxySnapshot:  &proxySnapshot,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, log.Price, 0.000001)
	require.Zero(t, log.ChargedAmount)
	require.Equal(t, SocialTaskChargeStatusNotCharged, log.ChargeStatus)
	require.Nil(t, log.ChargeSource)
	require.Equal(t, proxyID, *log.ProxyID)
	require.Equal(t, proxySnapshot, *log.ProxySnapshot)
	require.Equal(t, idempotencyKey, *log.IdempotencyKey)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, stored.Price, 0.000001)
	require.Zero(t, stored.ChargedAmount)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
}

func TestSocialTaskLogIdempotencyReturnsExistingOnDuplicate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("idempotency-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("idem_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("idem_account").
		SaveX(ctx)
	idempotencyKey := "idem-123"

	first, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionFollow,
		Status:         SocialTaskLogStatusPending,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)

	second, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionFollow,
		Status:         SocialTaskLogStatusPending,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSocialBillingEstimateUsesSubscriptionBeforeWallet(t *testing.T) {
	ctx := context.Background()
	limit := 0.20
	group := &Group{
		ID:               9,
		Name:             "Social quota",
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &limit,
		Hydrated:         true,
	}
	subRepo := &subscriptionRepoState{
		sub: &UserSubscription{
			ID:               7,
			UserID:           11,
			GroupID:          group.ID,
			StartsAt:         time.Now().Add(-time.Hour),
			ExpiresAt:        time.Now().Add(time.Hour),
			Status:           SubscriptionStatusActive,
			DailyWindowStart: socialPtrTime(time.Now().Add(-time.Hour)),
			Group:            group,
		},
	}
	userRepo := &socialBillingUserRepoStub{user: &User{ID: 11, Balance: 0.10}}
	billing := NewSocialBillingService(userRepo, subRepo, &socialBillingGroupRepoStub{group: group}, nil)

	estimate, err := billing.Estimate(ctx, 11, 3)
	require.NoError(t, err)
	require.InEpsilon(t, 0.30, estimate.EstimatedTotal, 0.000001)
	require.InEpsilon(t, 0.20, estimate.SubscriptionEstimatedUsage, 0.000001)
	require.InEpsilon(t, 0.10, estimate.WalletRequired, 0.000001)
	require.True(t, estimate.CanAfford)

	userRepo.user.Balance = 0.09
	estimate, err = billing.EnsureCanAfford(ctx, 11, 3)
	require.ErrorIs(t, err, ErrSocialTaskInsufficientFunds)
	require.False(t, estimate.CanAfford)
}

func TestSocialBillingChargeSuccessfulActionFailsClosed(t *testing.T) {
	ctx := context.Background()
	limit := 0.10
	group := &Group{
		ID:               9,
		Name:             "Social quota",
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &limit,
		Hydrated:         true,
	}
	subRepo := &subscriptionRepoState{
		sub: &UserSubscription{
			ID:               7,
			UserID:           11,
			GroupID:          group.ID,
			StartsAt:         time.Now().Add(-time.Hour),
			ExpiresAt:        time.Now().Add(time.Hour),
			Status:           SubscriptionStatusActive,
			DailyWindowStart: socialPtrTime(time.Now().Add(-time.Hour)),
			Group:            group,
		},
	}
	userRepo := &socialBillingUserRepoStub{user: &User{ID: 11, Balance: 0.25}}
	billing := NewSocialBillingService(userRepo, subRepo, &socialBillingGroupRepoStub{group: group}, nil)

	charged, err := billing.ChargeSuccessfulAction(ctx, 11, 0)
	require.NoError(t, err)
	require.Zero(t, charged.Amount)
	require.InEpsilon(t, 0.25, userRepo.user.Balance, 0.000001)

	charged, err = billing.ChargeSuccessfulAction(ctx, 11, SocialTaskUnitPrice)
	require.Nil(t, charged)
	require.Error(t, err)
	require.Equal(t, http.StatusServiceUnavailable, infraerrors.Code(err))
	require.Zero(t, subRepo.sub.DailyUsageUSD)
	require.InEpsilon(t, 0.25, userRepo.user.Balance, 0.000001)
}

func TestSocialTaskExecutorFinalizesSuccessBillingAtomically(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-success@example.com").SetPasswordHash("hash").SetBalance(0.25).SaveX(ctx)
	limit := 0.05
	group := client.Group.Create().
		SetName("Executor social quota").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_success").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_success").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceMixed, charge.Source)
	require.InEpsilon(t, 0.05, charge.SubscriptionAmount, 0.000001)
	require.InEpsilon(t, 0.05, charge.WalletAmount, 0.000001)
	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusCharged, storedTask.ChargeStatus)
	require.Equal(t, SocialTaskChargeSourceMixed, *storedTask.ChargeSource)
	require.InEpsilon(t, SocialTaskUnitPrice, storedTask.ChargedAmount, 0.000001)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.05, storedSub.DailyUsageUsd, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.20, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorFinalizesSuccessFromSubscriptionOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-subscription@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	limit := 0.20
	group := client.Group.Create().
		SetName("Executor subscription quota").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_subscription").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_subscription").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceSubscription, charge.Source)
	require.InEpsilon(t, SocialTaskUnitPrice, charge.SubscriptionAmount, 0.000001)
	require.Zero(t, charge.WalletAmount)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.DailyUsageUsd, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Zero(t, storedUser.Balance)
}

func TestSocialTaskExecutorFinalizesSuccessFromWalletOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-wallet@example.com").SetPasswordHash("hash").SetBalance(0.25).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_wallet").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_wallet").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceWallet, charge.Source)
	require.Zero(t, charge.SubscriptionAmount)
	require.InEpsilon(t, SocialTaskUnitPrice, charge.WalletAmount, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.15, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorFinalizesSuccessRejectsInsufficientFunds(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-insufficient@example.com").SetPasswordHash("hash").SetBalance(0.09).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_insufficient").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_insufficient").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.Nil(t, charge)
	require.ErrorIs(t, err, ErrSocialTaskInsufficientFunds)
	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusRunning, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.09, storedUser.Balance, 0.000001)
}

func TestSocialIPServiceRejectsUnsafeEndpointOnCreateAndUpdate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	for _, raw := range []string{
		"http://127.0.0.1:8080",
		"http://10.0.0.1:8080",
		"http://169.254.169.254:80",
		"http://localhost:8080",
		"http://8.8.8.8",
		"ftp://8.8.8.8:21",
	} {
		t.Run("create_"+raw, func(t *testing.T) {
			_, err := svc.Create(ctx, &CreateSocialIPInput{
				UserID:   user.ID,
				Name:     "bad proxy",
				IPType:   "residential",
				Endpoint: &raw,
			})
			require.Error(t, err)
			require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
		})
	}

	safeEndpoint := "http://8.8.8.8:8080"
	ip, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "safe proxy",
		IPType:   "residential",
		Endpoint: &safeEndpoint,
	})
	require.NoError(t, err)

	unsafeEndpoint := "socks5://[::1]:1080"
	_, err = svc.Update(ctx, ip.ID, &UpdateSocialIPInput{Endpoint: &unsafeEndpoint})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))

	stored, err := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, safeEndpoint, *stored.Endpoint)
}

func TestSocialIPServiceResetsConnectivityStatusWhenEndpointChanges(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-status-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)
	endpoint := "http://8.8.8.8:8080"
	ip, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "status proxy",
		IPType:   "residential",
		Endpoint: &endpoint,
	})
	require.NoError(t, err)

	latency := 12
	checkedAt := time.Now()
	client.SocialIP.UpdateOneID(ip.ID).
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(latency).
		SetLastCheckAt(checkedAt).
		SaveX(ctx)
	newEndpoint := "http://8.8.4.4:8080"
	updated, err := svc.Update(ctx, ip.ID, &UpdateSocialIPInput{Endpoint: &newEndpoint})
	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, updated.Status)
	require.Nil(t, updated.LatencyMs)
	require.Nil(t, updated.LastCheckAt)

	stored, err := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, stored.Status)
	require.Nil(t, stored.LatencyMs)
	require.Nil(t, stored.LastCheckAt)
	require.Equal(t, newEndpoint, *stored.Endpoint)
}

func newSocialOpsServiceTestClient(t *testing.T) *dbent.Client {
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

type subscriptionRepoState struct {
	UserSubscriptionRepository
	sub *UserSubscription
}

func (r *subscriptionRepoState) GetByID(context.Context, int64) (*UserSubscription, error) {
	if r.sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	out := *r.sub
	return &out, nil
}

func (r *subscriptionRepoState) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	if r.sub == nil {
		return nil, nil
	}
	return []UserSubscription{*r.sub}, nil
}

func (r *subscriptionRepoState) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	if r.sub == nil || r.sub.Status != SubscriptionStatusActive {
		return nil, nil
	}
	return []UserSubscription{*r.sub}, nil
}

func (r *subscriptionRepoState) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	items := []UserSubscription{}
	if r.sub != nil {
		items = append(items, *r.sub)
	}
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: 1, PageSize: 20, Pages: 1}, nil
}

func (r *subscriptionRepoState) ExtendExpiry(_ context.Context, _ int64, expiresAt time.Time) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.ExpiresAt = expiresAt
	return nil
}

func (r *subscriptionRepoState) UpdateStatus(_ context.Context, _ int64, status string) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.Status = status
	return nil
}

func (r *subscriptionRepoState) ResetDailyUsage(_ context.Context, _ int64, start time.Time) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.DailyUsageUSD = 0
	r.sub.DailyWindowStart = &start
	return nil
}

func (r *subscriptionRepoState) ResetWeeklyUsage(_ context.Context, _ int64, start time.Time) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.WeeklyUsageUSD = 0
	r.sub.WeeklyWindowStart = &start
	return nil
}

func (r *subscriptionRepoState) ResetMonthlyUsage(_ context.Context, _ int64, start time.Time) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.MonthlyUsageUSD = 0
	r.sub.MonthlyWindowStart = &start
	return nil
}

func (r *subscriptionRepoState) IncrementUsage(_ context.Context, _ int64, amount float64) error {
	if r.sub == nil {
		return ErrSubscriptionNotFound
	}
	r.sub.DailyUsageUSD += amount
	r.sub.WeeklyUsageUSD += amount
	r.sub.MonthlyUsageUSD += amount
	return nil
}

type socialBillingUserRepoStub struct {
	UserRepository
	user *User
}

func (r *socialBillingUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, ErrUserNotFound
	}
	out := *r.user
	return &out, nil
}

func (r *socialBillingUserRepoStub) DeductBalance(_ context.Context, id int64, amount float64) error {
	if r.user == nil || r.user.ID != id {
		return ErrUserNotFound
	}
	r.user.Balance -= amount
	return nil
}

type socialBillingGroupRepoStub struct {
	GroupRepository
	group *Group
}

func (r *socialBillingGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if r.group == nil || r.group.ID != id {
		return nil, ErrGroupNotFound
	}
	out := *r.group
	return &out, nil
}

func socialPtrTime(t time.Time) *time.Time {
	return &t
}
