//go:build unit

package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/socialip"
	"github.com/Wei-Shaw/socialops/ent/socialtasklog"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
	infraerrors "github.com/Wei-Shaw/socialops/internal/pkg/errors"
	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func encryptedTwitterExecutionAuthForTest(t *testing.T) string {
	t.Helper()
	stored, err := normalizeTwitterExecutionAuthForEncryptedStorage(
		`{"access_token":"access-token","token_secret":"token-secret","screen_name":"northwind_ops"}`,
		"northwind_ops",
		executionAuthEncryptorStub{},
	)
	require.NoError(t, err)
	return stored
}

func encryptedTwitterExecutionAuthPayloadForTest(t *testing.T, payload, screenName string) string {
	t.Helper()
	stored, err := normalizeTwitterExecutionAuthForEncryptedStorage(payload, screenName, executionAuthEncryptorStub{})
	require.NoError(t, err)
	return stored
}

func newEncryptedTwitterExecutorForTest() *TwitterExecutor {
	return NewTwitterExecutor().WithCredentialEncryptor(executionAuthEncryptorStub{})
}

func TestAdminServiceSkeletonFailsClosed(t *testing.T) {
	svc := NewAdminServiceSkeleton()

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: "user@example.com"})
	require.Nil(t, user)
	require.ErrorIs(t, err, ErrAdminServiceNotConfigured)
	require.Equal(t, http.StatusNotImplemented, infraerrors.Code(err))

	code, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{Count: 1})
	require.Nil(t, code)
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
		SocialTaskActionPost,
		SocialTaskActionLike,
		SocialTaskActionRetweet,
	} {
		t.Run(action, func(t *testing.T) {
			task.Action = action
			result, err := executor.executeAction(context.Background(), task)
			require.Empty(t, result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not configured")
			kind, ok := socialExecutionFailureKind(err)
			require.True(t, ok)
			require.Equal(t, SocialExecutionFailurePlatform, kind)
		})
	}

	task.Action = "edit_ip"
	result, err := executor.executeAction(context.Background(), task)
	require.Empty(t, result)
	require.ErrorContains(t, err, "unsupported action")
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureUnsupported, kind)
}

func TestSocialTaskExecutorMissingActionInputFailsClosedWithSafeMessage(t *testing.T) {
	executor := &SocialTaskExecutor{}

	tests := []struct {
		name        string
		task        *dbent.SocialTaskLog
		wantMessage string
	}{
		{
			name:        "follow target",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionFollow},
			wantMessage: "follow target is required",
		},
		{
			name:        "post content or media",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionPost},
			wantMessage: "post content or media is required",
		},
		{
			name:        "profile payload",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionUpdateProfile},
			wantMessage: "profile payload is required",
		},
		{
			name:        "avatar media",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionUpdateAvatar},
			wantMessage: "avatar media is required",
		},
		{
			name:        "banner media",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionUpdateBanner},
			wantMessage: "banner media is required",
		},
		{
			name:        "like target",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionLike},
			wantMessage: "like target (post URL/ID) is required",
		},
		{
			name:        "retweet target",
			task:        &dbent.SocialTaskLog{Action: SocialTaskActionRetweet},
			wantMessage: "retweet target (post URL/ID) is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executor.executeAction(context.Background(), tc.task)

			require.Empty(t, result)
			require.Error(t, err)
			require.Equal(t, tc.wantMessage, err.Error())
			kind, ok := socialExecutionFailureKind(err)
			require.True(t, ok)
			require.Equal(t, SocialExecutionFailureActionInput, kind)
			require.Equal(t, "任务参数不完整，本次未扣费", safeSocialTaskFailureMessage(err))
		})
	}
}

func TestSocialTaskExecutorRejectsMissingTaskLogWithoutPanicRecovery(t *testing.T) {
	executor := &SocialTaskExecutor{}

	result, err := executor.executeActionSafely(context.Background(), nil)

	require.Empty(t, result)
	require.ErrorContains(t, err, "social task log is unavailable")
	require.NotContains(t, err.Error(), "unexpectedly")
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
	billingRequestID := "precreated-charge-request"
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetBillingRequestID(billingRequestID).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	executor.processTask(log.ID)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.ChargeSource)
	require.Nil(t, stored.BillingRequestID)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "follow is not configured: social platform executor is not available", *stored.ResultMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorProcessPendingTasksFailsClosedWhenQueueFull(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-queue-full@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_queue_full").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_queue_full").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	secondAccount := client.SocialAccount.Create().
		SetName("executor_queue_full_second").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_queue_full_second").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	billingRequestID := "queued-charge-request"
	first := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetCreatedAt(time.Now().Add(-time.Minute)).
		SaveX(ctx)
	second := client.SocialTaskLog.Create().
		SetSocialAccountID(secondAccount.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLike).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetBillingRequestID(billingRequestID).
		SetCreatedAt(time.Now()).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	enqueued, err := executor.ProcessPendingTasks(ctx, 2)

	require.NoError(t, err)
	require.Equal(t, 1, enqueued)

	storedFirst, err := client.SocialTaskLog.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusPending, storedFirst.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedFirst.ChargeStatus)
	require.Zero(t, storedFirst.ChargedAmount)

	storedSecond, err := client.SocialTaskLog.Get(ctx, second.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedSecond.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedSecond.ChargeStatus)
	require.Zero(t, storedSecond.ChargedAmount)
	require.Nil(t, storedSecond.ChargeSource)
	require.Nil(t, storedSecond.BillingRequestID)
	require.NotNil(t, storedSecond.ResultMessage)
	require.Equal(t, "任务队列繁忙，本次未扣费", *storedSecond.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, secondAccount.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "任务队列繁忙，本次未扣费", *storedAccount.TaskMessage)
}

func TestSocialTaskExecutorProcessPendingTasksSkipsAlreadyChargedPendingLogs(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-skip-charged-pending@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_skip_charged_pending").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_skip_charged_pending").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	secondAccount := client.SocialAccount.Create().
		SetName("executor_skip_charged_pending_second").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_skip_charged_pending_second").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	chargedSource := SocialTaskChargeSourceWallet
	chargedRequestID := "already-charged-request"
	chargedPending := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(SocialTaskUnitPrice).
		SetChargeStatus(SocialTaskChargeStatusCharged).
		SetChargeSource(chargedSource).
		SetBillingRequestID(chargedRequestID).
		SetCreatedAt(time.Now().Add(-time.Minute)).
		SaveX(ctx)
	notChargedPending := client.SocialTaskLog.Create().
		SetSocialAccountID(secondAccount.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLike).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetCreatedAt(time.Now()).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	executor.processTask(chargedPending.ID)
	enqueued, err := executor.ProcessPendingTasks(ctx, 2)

	require.NoError(t, err)
	require.Equal(t, 1, enqueued)

	storedCharged, err := client.SocialTaskLog.Get(ctx, chargedPending.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusPending, storedCharged.Status)
	require.Equal(t, SocialTaskChargeStatusCharged, storedCharged.ChargeStatus)
	require.InEpsilon(t, SocialTaskUnitPrice, storedCharged.ChargedAmount, 0.000001)
	require.NotNil(t, storedCharged.ChargeSource)
	require.Equal(t, chargedSource, *storedCharged.ChargeSource)
	require.NotNil(t, storedCharged.BillingRequestID)
	require.Equal(t, chargedRequestID, *storedCharged.BillingRequestID)
	require.Nil(t, storedCharged.ResultMessage)
	require.Nil(t, storedCharged.ExecutedAt)

	storedNotCharged, err := client.SocialTaskLog.Get(ctx, notChargedPending.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusPending, storedNotCharged.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedNotCharged.ChargeStatus)
	require.Zero(t, storedNotCharged.ChargedAmount)
}

func TestSocialTaskExecutorEnqueueBatchReturnsExactFailedTaskIDs(t *testing.T) {
	attempts := make([]int64, 0, 4)
	enqueued, failed := enqueueSocialTaskBatch([]int64{10, 11, 12, 13}, func(id int64) bool {
		attempts = append(attempts, id)
		return id == 11 || id == 13
	})

	require.Equal(t, 2, enqueued)
	require.Equal(t, []int64{10, 12}, failed)
	require.Equal(t, []int64{10, 11, 12, 13}, attempts)
}

func TestSocialTaskExecutorProcessPendingTasksReturnsUnavailableWithoutClient(t *testing.T) {
	enqueued, err := (&SocialTaskExecutor{}).ProcessPendingTasks(context.Background(), 1)

	require.Zero(t, enqueued)
	require.Error(t, err)
	require.Contains(t, err.Error(), "social task executor is unavailable")
}

func TestAccountWorkbenchServiceSubmitTaskFailsClosedWhenDependenciesMissing(t *testing.T) {
	target := "target"
	_, err := (&AccountWorkbenchService{}).SubmitTask(context.Background(), &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     1,
		AccountIDs: []int64{1},
		Action:     SocialTaskActionFollow,
		Target:     &target,
	})

	require.Error(t, err)
	require.Equal(t, http.StatusServiceUnavailable, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_TASK_SERVICE_UNAVAILABLE", infraerrors.Reason(err))
}

func TestAccountWorkbenchServiceRejectsUserTaskWithoutDefaultProxyBeforeLogOrBilling(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("user-task-missing-proxy@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("user_task_missing_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("user_task_missing_proxy").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)
	target := "@target"

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionFollow,
		Target:     &target,
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
	count, countErr := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, countErr)
	require.Zero(t, count)
}

func TestAccountWorkbenchServiceRejectsNonLoginTaskWithoutDefaultProxyEvenWhenGlobalFallbackExists(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	globalSvc := NewGlobalProxyService(client)
	user := client.User.Create().
		SetEmail("user-task-global-not-for-follow@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("user_task_global_not_for_follow").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("user_task_global_not_for_follow").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	global := client.GlobalProxy.Create().
		SetName("global only").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint("http://8.8.8.8:8080").
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchServiceWithGlobalProxies(accountSvc, NewSocialIPService(client), globalSvc, billing, nil)
	target := "@target"

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionFollow,
		Target:     &target,
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
	storedGlobal := client.GlobalProxy.GetX(ctx, global.ID)
	require.Nil(t, storedGlobal.LastUsedAt)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
	count, countErr := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, countErr)
	require.Zero(t, count)
}

func TestAccountWorkbenchServiceLoginUsesGlobalFallbackWhenAccountHasNoDefaultProxy(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)
	globalSvc := NewGlobalProxyService(client)
	user := client.User.Create().
		SetEmail("login-global-fallback@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_global_fallback").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_global_fallback").
		SetAssignedUserID(user.ID).
		SetPassword("secret").
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	global := client.GlobalProxy.Create().
		SetName("global fallback").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint(endpoint).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchServiceWithGlobalProxies(accountSvc, ipSvc, globalSvc, billing, nil)

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionLogin,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.FailedClosed)
	require.Len(t, result.Logs, 1)
	log := result.Logs[0]
	require.Nil(t, log.ProxyID)
	require.NotNil(t, log.ProxySnapshot)
	require.Contains(t, *log.ProxySnapshot, `"scope":"global"`)
	require.Contains(t, *log.ProxySnapshot, fmt.Sprintf(`"id":%d`, global.ID))
	require.Contains(t, *log.ProxySnapshot, endpoint)
	storedGlobal := client.GlobalProxy.GetX(ctx, global.ID)
	require.NotNil(t, storedGlobal.LastUsedAt)
}

func TestAccountWorkbenchServiceLoginPrefersAccountDefaultProxyOverGlobalFallback(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)
	globalSvc := NewGlobalProxyService(client)
	user := client.User.Create().
		SetEmail("login-default-proxy-preferred@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_default_proxy_preferred").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_default_proxy_preferred").
		SetAssignedUserID(user.ID).
		SetPassword("secret").
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	userEndpoint := "http://8.8.4.4:8080"
	userProxy, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "user login proxy",
		IPType:   SocialIPTypeResidential,
		Endpoint: &userEndpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(userProxy.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	userProxy, err = ipSvc.GetByID(ctx, userProxy.ID)
	require.NoError(t, err)
	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, userProxy)
	require.NoError(t, err)
	globalEndpoint := "http://1.1.1.1:8080"
	global := client.GlobalProxy.Create().
		SetName("unused global fallback").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint(globalEndpoint).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchServiceWithGlobalProxies(accountSvc, ipSvc, globalSvc, billing, nil)

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionLogin,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Logs, 1)
	log := result.Logs[0]
	require.NotNil(t, log.ProxyID)
	require.Equal(t, userProxy.ID, *log.ProxyID)
	require.NotNil(t, log.ProxySnapshot)
	require.Contains(t, *log.ProxySnapshot, userEndpoint)
	require.NotContains(t, *log.ProxySnapshot, globalEndpoint)
	storedGlobal := client.GlobalProxy.GetX(ctx, global.ID)
	require.Nil(t, storedGlobal.LastUsedAt)
}

func TestAccountWorkbenchServiceLoginWithoutAnyUsableProxyFailsBeforeLogOrBilling(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("login-no-global-proxy@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_no_global_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_no_global_proxy").
		SetAssignedUserID(user.ID).
		SetPassword("secret").
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchServiceWithGlobalProxies(accountSvc, NewSocialIPService(client), NewGlobalProxyService(client), billing, nil)

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionLogin,
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "GLOBAL_PROXY_NOT_AVAILABLE", infraerrors.Reason(err))
	count, countErr := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, countErr)
	require.Zero(t, count)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
}

func TestGlobalProxyServiceNextAvailableRotatesOnlineProxies(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewGlobalProxyService(client)
	first := client.GlobalProxy.Create().
		SetName("global one").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint("http://8.8.8.8:8080").
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	second := client.GlobalProxy.Create().
		SetName("global two").
		SetIPType(SocialIPTypeResidential).
		SetEndpoint("http://8.8.4.4:8080").
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)

	selectedFirst, err := svc.NextAvailable(ctx)
	require.NoError(t, err)
	selectedSecond, err := svc.NextAvailable(ctx)
	require.NoError(t, err)

	require.Equal(t, first.ID, selectedFirst.ID)
	require.Equal(t, second.ID, selectedSecond.ID)
}

func TestAccountWorkbenchServiceRejectsStaleDefaultProxySnapshotBeforeLogOrBilling(t *testing.T) {
	tests := []struct {
		name  string
		setup func(context.Context, *dbent.Client, *SocialIPService, int64) (string, []string)
	}{
		{
			name: "cross owner proxy",
			setup: func(ctx context.Context, client *dbent.Client, ipSvc *SocialIPService, ownerID int64) (string, []string) {
				otherUser := client.User.Create().
					SetEmail("user-task-cross-owner-proxy-other@example.com").
					SetPasswordHash("hash").
					SaveX(ctx)
				otherEndpoint := "http://8.8.4.4:8080"
				otherIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{
					UserID:   otherUser.ID,
					Name:     "cross owner secret proxy",
					IPType:   SocialIPTypeResidential,
					Endpoint: &otherEndpoint,
				})
				require.NoError(t, err)
				client.SocialIP.UpdateOneID(otherIP.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
				otherIP, err = ipSvc.GetByID(ctx, otherIP.ID)
				require.NoError(t, err)
				return SocialIPTaskSnapshot(otherIP), []string{otherEndpoint, otherIP.Name}
			},
		},
		{
			name: "deleted proxy",
			setup: func(ctx context.Context, client *dbent.Client, ipSvc *SocialIPService, ownerID int64) (string, []string) {
				endpoint := "http://8.8.8.8:8080"
				deletedIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{
					UserID:   ownerID,
					Name:     "deleted default proxy",
					IPType:   SocialIPTypeResidential,
					Endpoint: &endpoint,
				})
				require.NoError(t, err)
				client.SocialIP.UpdateOneID(deletedIP.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
				deletedIP, err = ipSvc.GetByID(ctx, deletedIP.ID)
				require.NoError(t, err)
				snapshot := SocialIPTaskSnapshot(deletedIP)
				require.NoError(t, ipSvc.Delete(ctx, deletedIP.ID))
				return snapshot, []string{endpoint, deletedIP.Name}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := newSocialOpsServiceTestClient(t)
			accountSvc := NewSocialAccountService(client)
			ipSvc := NewSocialIPService(client)
			owner := client.User.Create().
				SetEmail("user-task-stale-proxy-" + strings.ReplaceAll(tc.name, " ", "-") + "@example.com").
				SetPasswordHash("hash").
				SaveX(ctx)
			staleSnapshot, blockedText := tc.setup(ctx, client, ipSvc, owner.ID)
			account := client.SocialAccount.Create().
				SetName("user_task_stale_proxy_" + strings.ReplaceAll(tc.name, " ", "_")).
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("user_task_stale_proxy_" + strings.ReplaceAll(tc.name, " ", "_")).
				SetAssignedUserID(owner.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusStored).
				SetDefaultProxySnapshot(staleSnapshot).
				SaveX(ctx)
			userRepo := &socialBillingUserRepoStub{user: &User{ID: owner.ID, Balance: 1}}
			billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
			workbench := NewAccountWorkbenchService(accountSvc, ipSvc, billing, nil)
			target := "@target"

			result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
				Mode:       AccountWorkbenchTaskModeUser,
				UserID:     owner.ID,
				AccountIDs: []int64{account.ID},
				Action:     SocialTaskActionFollow,
				Target:     &target,
			})

			require.Nil(t, result)
			require.Error(t, err)
			require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
			require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
			for _, blocked := range blockedText {
				require.NotContains(t, err.Error(), blocked)
			}
			require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
			count, countErr := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, countErr)
			require.Zero(t, count)
		})
	}
}

func TestAccountWorkbenchServiceSubmitTaskRejectsAccountWithActiveTask(t *testing.T) {
	for _, activeStatus := range []string{SocialTaskLogStatusPending, SocialTaskLogStatusRunning} {
		t.Run(activeStatus, func(t *testing.T) {
			ctx := context.Background()
			client := newSocialOpsServiceTestClient(t)
			accountSvc := NewSocialAccountService(client)
			user := client.User.Create().
				SetEmail("busy-submit-" + activeStatus + "@example.com").
				SetPasswordHash("hash").
				SaveX(ctx)
			account := client.SocialAccount.Create().
				SetName("busy_submit_" + activeStatus).
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("busy_submit_" + activeStatus).
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusStored).
				SaveX(ctx)

			activeLog, err := accountSvc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
				AccountID: account.ID,
				UserID:    user.ID,
				Action:    SocialTaskActionLoginCheck,
				Status:    activeStatus,
			})
			require.NoError(t, err)

			userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
			billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
			workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)
			target := "@busy_target"

			result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
				Mode:       AccountWorkbenchTaskModeAdmin,
				AccountIDs: []int64{account.ID},
				Action:     SocialTaskActionFollow,
				Target:     &target,
			})

			require.Nil(t, result)
			require.Error(t, err)
			require.Equal(t, http.StatusConflict, infraerrors.Code(err))
			require.Equal(t, "SOCIAL_TASK_ACCOUNT_BUSY", infraerrors.Reason(err))
			require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
			count, countErr := client.SocialTaskLog.Query().Count(ctx)
			require.NoError(t, countErr)
			require.Equal(t, 1, count)
			stored := client.SocialTaskLog.GetX(ctx, activeLog.ID)
			require.Equal(t, activeStatus, stored.Status)
		})
	}
}

func TestAccountWorkbenchServiceSubmitTaskReplaysActiveIdempotentTask(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("busy-submit-idempotent@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("busy_submit_idempotent").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("busy_submit_idempotent").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	idempotencyKey := "busy-submit-idempotent-123"
	target := "@idempotent_busy_target"
	activeLog, err := accountSvc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionFollow,
		Target:         &target,
		Status:         SocialTaskLogStatusPending,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)

	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:           AccountWorkbenchTaskModeAdmin,
		AccountIDs:     []int64{account.ID},
		Action:         SocialTaskActionFollow,
		Target:         &target,
		IdempotencyKey: idempotencyKey,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.Submitted)
	require.Zero(t, result.Enqueued)
	require.Zero(t, result.FailedClosed)
	require.Len(t, result.Logs, 1)
	require.Equal(t, activeLog.ID, result.Logs[0].ID)
	require.InEpsilon(t, 1.0, userRepo.user.Balance, 0.000001)
	count, countErr := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, countErr)
	require.Equal(t, 1, count)
}

func TestSocialTaskExecutorProcessPendingTasksFailsClosedWhenQueueMissing(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-missing-queue@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_missing_queue").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_missing_queue").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	billingRequestID := "missing-queue-charge-request"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetBillingRequestID(billingRequestID).
		SaveX(ctx)
	executor := &SocialTaskExecutor{entClient: client}

	enqueued, err := executor.ProcessPendingTasks(ctx, 1)

	require.NoError(t, err)
	require.Zero(t, enqueued)

	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.ChargeSource)
	require.Nil(t, stored.BillingRequestID)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "social platform executor queue is not configured; task was not charged", *stored.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "social platform executor queue is not configured; task was not charged", *storedAccount.TaskMessage)
}

func TestSocialTaskExecutorStartRecoversExistingPendingTasks(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-start-recovery@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_start_recovery").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_start_recovery").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1, MinIntervalMs: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "follow succeeded", nil
	}))
	executor.Start()
	t.Cleanup(executor.Stop)

	require.Eventually(t, func() bool {
		stored, err := client.SocialTaskLog.Get(ctx, task.ID)
		return err == nil && stored.Status == SocialTaskLogStatusSuccess && stored.ChargeStatus == SocialTaskChargeStatusCharged
	}, time.Second, 20*time.Millisecond)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.90, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorStartFailsStaleRunningTasksWithoutCharge(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-stale-running@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_stale_running").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_stale_running").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	billingRequestID := "stale-running-charge-request"
	staleAt := time.Now().Add(-3 * time.Minute)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetBillingRequestID(billingRequestID).
		SetCreatedAt(staleAt).
		SetUpdatedAt(staleAt).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1, MinIntervalMs: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.Start()
	t.Cleanup(executor.Stop)

	require.Eventually(t, func() bool {
		stored, err := client.SocialTaskLog.Get(ctx, task.ID)
		return err == nil && stored.Status == SocialTaskLogStatusFailed && stored.ChargeStatus == SocialTaskChargeStatusNotCharged
	}, time.Second, 20*time.Millisecond)

	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.ChargeSource)
	require.Nil(t, stored.BillingRequestID)
	require.NotNil(t, stored.ExecutedAt)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "任务执行超时，本次未扣费", *stored.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "任务执行超时，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorDispatchesStructuredProfileActionsToPlatformExecutors(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		action         string
		buildPayload   func(t *testing.T) SocialTaskPayload
		expectedResult string
	}{
		{
			name:   "update profile",
			action: SocialTaskActionUpdateProfile,
			buildPayload: func(_ *testing.T) SocialTaskPayload {
				return SocialTaskPayload{
					Profile: &SocialProfileUpdateParams{
						DisplayName: "Northwind Ops",
						Description: "Operator account",
					},
				}
			},
			expectedResult: "profile updated",
		},
		{
			name:   "update avatar",
			action: SocialTaskActionUpdateAvatar,
			buildPayload: func(t *testing.T) SocialTaskPayload {
				return SocialTaskPayload{
					Avatar: &SocialTaskMediaRef{
						Source:      "inline",
						ContentType: "image/png",
						FileName:    "avatar.png",
						URL:         inlinePNGDataURLForSocialTaskValidation(t, 400, 400),
					},
				}
			},
			expectedResult: "avatar updated",
		},
		{
			name:   "update banner",
			action: SocialTaskActionUpdateBanner,
			buildPayload: func(t *testing.T) SocialTaskPayload {
				return SocialTaskPayload{
					Banner: &SocialTaskMediaRef{
						Source:      "inline",
						ContentType: "image/jpeg",
						FileName:    "banner.jpg",
						URL:         inlineJPEGDataURLForSocialTaskValidation(t, 1500, 500),
					},
				}
			},
			expectedResult: "banner updated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSocialOpsServiceTestClient(t)
			user := client.User.Create().SetEmail(strings.ReplaceAll(tc.action, "_", "-") + "@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
			account := client.SocialAccount.Create().
				SetName(strings.ReplaceAll(tc.action, "_", "_") + "_account").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(strings.ReplaceAll(tc.action, "_", "_") + "_account").
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusPending).
				SaveX(ctx)
			proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
			task := client.SocialTaskLog.Create().
				SetSocialAccountID(account.ID).
				SetUserID(user.ID).
				SetAction(tc.action).
				SetProxySnapshot(proxySnapshot).
				SetPayload(tc.buildPayload(t)).
				SetStatus(SocialTaskLogStatusPending).
				SetPrice(SocialTaskUnitPrice).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				SaveX(ctx)

			billing := NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
			executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})

			var executedAction string
			var executedPayload SocialTaskPayload
			executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(_ context.Context, log *dbent.SocialTaskLog, _ *dbent.SocialAccount) (string, error) {
				executedAction = log.Action
				executedPayload = log.Payload
				return tc.expectedResult, nil
			}))

			executor.processTask(task.ID)

			require.Equal(t, tc.action, executedAction)
			require.Equal(t, tc.buildPayload(t), executedPayload)

			stored, err := client.SocialTaskLog.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, SocialTaskLogStatusSuccess, stored.Status)
			require.Equal(t, SocialTaskChargeStatusCharged, stored.ChargeStatus)
			require.InEpsilon(t, SocialTaskUnitPrice, stored.ChargedAmount, 0.000001)
			require.NotNil(t, stored.ResultMessage)
			require.Equal(t, tc.expectedResult, *stored.ResultMessage)

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 0.90, storedUser.Balance, 0.000001)

			storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
			require.NoError(t, err)
			require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
			require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
			require.NotNil(t, storedAccount.TaskMessage)
			require.Equal(t, tc.expectedResult, *storedAccount.TaskMessage)
		})
	}
}

func TestSocialTaskExecutorFailsClosedWhenAccountExecutionScopeChangesBeforeWorkerRuns(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name   string
		mutate func(client *dbent.Client, accountID int64, ownerID int64)
	}{
		{
			name: "account reclaimed before execution",
			mutate: func(client *dbent.Client, accountID int64, _ int64) {
				client.SocialAccount.UpdateOneID(accountID).ClearAssignedUserID().SaveX(ctx)
			},
		},
		{
			name: "account reassigned before execution",
			mutate: func(client *dbent.Client, accountID int64, _ int64) {
				other := client.User.Create().SetEmail("executor-scope-other@example.com").SetPasswordHash("hash").SaveX(ctx)
				client.SocialAccount.UpdateOneID(accountID).SetAssignedUserID(other.ID).SaveX(ctx)
			},
		},
		{
			name: "account marked unavailable before execution",
			mutate: func(client *dbent.Client, accountID int64, _ int64) {
				client.SocialAccount.UpdateOneID(accountID).SetAccountStatus(SocialAccountStatusLimited).SaveX(ctx)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newSocialOpsServiceTestClient(t)
			user := client.User.Create().SetEmail("executor-scope@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
			account := client.SocialAccount.Create().
				SetName("executor_scope").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey("executor_scope").
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusStored).
				SaveX(ctx)
			target := "@target"
			task := client.SocialTaskLog.Create().
				SetSocialAccountID(account.ID).
				SetUserID(user.ID).
				SetAction(SocialTaskActionFollow).
				SetTarget(target).
				SetStatus(SocialTaskLogStatusPending).
				SetPrice(SocialTaskUnitPrice).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				SaveX(ctx)
			called := false
			executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
			executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
				called = true
				return "follow succeeded", nil
			}))

			tc.mutate(client, account.ID, user.ID)
			executor.processTask(task.ID)

			require.False(t, called, "worker must not call a platform executor after task ownership or account state changed")
			storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
			require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
			require.Zero(t, storedTask.ChargedAmount)
			require.Nil(t, storedTask.ChargeSource)
			require.NotNil(t, storedTask.ResultMessage)
			require.Equal(t, "social account is unavailable", *storedTask.ResultMessage)

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
			ledgerCount, err := client.UsageLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, ledgerCount)
		})
	}
}

func TestSocialTaskExecutorDoesNotWriteAccountFailureStateWhenFailedTaskResultCannotPersist(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-failure-result-race@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_failure_result_race").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_failure_result_race").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		client.SocialTaskLog.UpdateOneID(task.ID).
			SetStatus(SocialTaskLogStatusFailed).
			SetResultMessage("already finalized").
			SetChargedAmount(0).
			SetChargeStatus(SocialTaskChargeStatusNotCharged).
			SaveX(ctx)
		return "", newSocialExecutionError(SocialExecutionFailureChallengeRequired, "additional verification required", nil)
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "already finalized", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.Nil(t, storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSocialTaskExecutorDoesNotOverwriteConcurrentlyFinalizedSuccessWhenBillingRaces(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-success-race@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_success_race").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_success_race").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	target := "@target"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		client.SocialTaskLog.UpdateOneID(task.ID).
			SetStatus(SocialTaskLogStatusSuccess).
			SetResultMessage("already finalized").
			SetExecutedAt(time.Now()).
			SetChargedAmount(SocialTaskUnitPrice).
			SetChargeStatus(SocialTaskChargeStatusCharged).
			SetChargeSource(SocialTaskChargeSourceWallet).
			SetBillingRequestID("wallet:external-finalizer").
			SaveX(ctx)
		return "follow succeeded", nil
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusCharged, storedTask.ChargeStatus)
	require.InEpsilon(t, SocialTaskUnitPrice, storedTask.ChargedAmount, 0.000001)
	require.NotNil(t, storedTask.ChargeSource)
	require.Equal(t, SocialTaskChargeSourceWallet, *storedTask.ChargeSource)
	require.NotNil(t, storedTask.BillingRequestID)
	require.Equal(t, "wallet:external-finalizer", *storedTask.BillingRequestID)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "already finalized", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, storedAccount.TaskStatus)
	require.Nil(t, storedAccount.TaskMessage)
}

func TestSocialTaskExecutorFailsClosedWhenPlatformExecutorPanics(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-panic@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_panic").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_panic").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "123456789"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		panic("panic-secret-token")
	}))

	require.NotPanics(t, func() {
		executor.processTask(task.ID)
	})

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "social platform executor failed unexpectedly", *storedTask.ResultMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSocialTaskExecutorStoresSafeFailureMessage(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-safe-failure@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_safe_failure").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_safe_failure").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "123456789"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "", newSocialExecutionError(
			SocialExecutionFailureNetwork,
			"network request failed via proxy=http://user:pass@127.0.0.1:8080 Authorization Bearer secret-token",
			nil,
		)
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "平台网络请求失败，本次未扣费", *storedTask.ResultMessage)
	require.NotContains(t, *storedTask.ResultMessage, "user:pass")
	require.NotContains(t, *storedTask.ResultMessage, "secret-token")

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "平台网络请求失败，本次未扣费", *storedAccount.TaskMessage)
	require.NotContains(t, *storedAccount.TaskMessage, "user:pass")
	require.NotContains(t, *storedAccount.TaskMessage, "secret-token")

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSafeSocialTaskFailureMessageMapsUnsupportedMediaSourceToSafeResult(t *testing.T) {
	err := newSocialExecutionError(
		SocialExecutionFailureActionInput,
		"post media #1 media source is not supported for SocialOps execution",
		nil,
	)

	require.Equal(t, "媒体引用暂未开放，本次未扣费", safeSocialTaskFailureMessage(err))
}

func TestSafeSocialTaskFailureMessagePreservesSpecificTargetAndContentPlatformReasons(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "target not found",
			err: newSocialExecutionError(
				SocialExecutionFailureActionInput,
				"target not found",
				nil,
			),
			want: "执行目标不存在，本次未扣费",
		},
		{
			name: "duplicate post content",
			err: newSocialExecutionError(
				SocialExecutionFailureActionInput,
				"post content is duplicate",
				nil,
			),
			want: "内容或目标状态不符合平台要求，本次未扣费",
		},
		{
			name: "already liked",
			err: newSocialExecutionError(
				SocialExecutionFailureActionInput,
				"tweet is already liked",
				nil,
			),
			want: "内容或目标状态不符合平台要求，本次未扣费",
		},
		{
			name: "twitter account not found",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 399: Sorry, we could not find your account.",
				nil,
			),
			want: "账号不存在，本次未扣费",
		},
		{
			name: "twitter 399 wrong password prefers password classification",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 399: Wrong password!",
				nil,
			),
			want: "密码错误，本次未扣费",
		},
		{
			name: "twitter 399 without exact business message remains raw",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 399",
				nil,
			),
			want: "twitter error 399",
		},
		{
			name: "twitter 399 with unknown business message remains raw",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 399: Password checkpoint required.",
				nil,
			),
			want: "twitter error 399: Password checkpoint required.",
		},
		{
			name: "typed password failure wins over generic twitter 399 code",
			err: newSocialExecutionError(
				SocialExecutionFailurePasswordInvalid,
				"twitter error 399",
				nil,
			),
			want: "密码错误，本次未扣费",
		},
		{
			name: "unknown twitter platform error",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 999: Something unexpected happened.",
				nil,
			),
			want: "twitter error 999: Something unexpected happened.",
		},
		{
			name: "twitter anti automation",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 226: This request looks like it might be automated.",
				nil,
			),
			want: "账号状态或频率受限，本次未扣费",
		},
		{
			name: "twitter login verification",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter error 231: User must verify login",
				nil,
			),
			want: "账号需要额外验证，本次未扣费",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, safeSocialTaskFailureMessage(tc.err))
		})
	}
}

func TestSafeSocialTaskFailureMessageMapsTwitterMediaUploadFailuresToSafeResult(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "invalid media id",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter media upload returned invalid media id",
				nil,
			),
			want: "平台媒体上传失败，本次未扣费",
		},
		{
			name: "invalid response",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter media upload returned invalid response",
				nil,
			),
			want: "平台媒体上传失败，本次未扣费",
		},
		{
			name: "no media id",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter media upload returned no media id",
				nil,
			),
			want: "平台媒体上传失败，本次未扣费",
		},
		{
			name: "processing failed",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter media upload returned processing failed",
				nil,
			),
			want: "平台媒体上传失败，本次未扣费",
		},
		{
			name: "processing timeout",
			err: newSocialExecutionError(
				SocialExecutionFailurePlatform,
				"twitter media upload returned processing timeout",
				nil,
			),
			want: "平台媒体上传失败，本次未扣费",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, safeSocialTaskFailureMessage(tc.err))
		})
	}
}

func TestTwitterExecutorFailsClosedWhenVideoMediaProcessingFails(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 18, Name: "@video_processing_fail_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            318,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello video"),
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Post: &SocialPostPayload{
				Text: "hello video",
				Media: []SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "video/mp4",
					FileName:    "clip.mp4",
					URL:         "data:video/mp4;base64,AAECAwQFBgcICQ==",
				}},
			},
		},
	}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("video media should fail closed before building HTTP client")
		return nil, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.Empty(t, result)
	require.Error(t, err)
	require.ErrorContains(t, err, "video media is not supported for SocialOps execution")
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureActionInput, kind)
}

func TestTwitterFailureKindMapsChallengeSignalsToChallengeRequired(t *testing.T) {
	tests := []twitterActionResult{
		{StatusCode: http.StatusForbidden, Message: "additional verification required"},
		{StatusCode: http.StatusForbidden, Message: "login challenge required"},
		{StatusCode: http.StatusForbidden, Message: "captcha challenge required"},
		{StatusCode: http.StatusForbidden, Message: "confirm your identity to continue"},
	}

	for _, result := range tests {
		require.Equal(t, SocialExecutionFailureChallengeRequired, twitterFailureKind(&result))
	}
}

func TestTwitterErrorMessageMapsAntiAutomationAndLoginVerificationCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{
			name:       "anti automation",
			statusCode: http.StatusForbidden,
			body:       `{"errors":[{"code":226,"message":"This request looks like it might be automated."}]}`,
			want:       "twitter error 226: This request looks like it might be automated.",
		},
		{
			name:       "login verification",
			statusCode: http.StatusForbidden,
			body:       `{"errors":[{"code":231,"message":"User must verify login"}]}`,
			want:       "twitter error 231: User must verify login",
		},
		{
			name:       "account not found",
			statusCode: http.StatusForbidden,
			body:       `{"errors":[{"code":399,"message":"Sorry, we could not find your account."}]}`,
			want:       "twitter error 399: Sorry, we could not find your account.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, twitterErrorMessage([]byte(tc.body), tc.statusCode))
		})
	}
}

func TestTwitterFailureKindMapsAntiAutomationAndLoginVerificationSignals(t *testing.T) {
	tests := []struct {
		name   string
		result twitterActionResult
		want   SocialExecutionFailureKind
	}{
		{
			name: "anti automation",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "request looks automated",
			},
			want: SocialExecutionFailureAccountLimited,
		},
		{
			name: "login verification",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "login verification required",
			},
			want: SocialExecutionFailureChallengeRequired,
		},
		{
			name: "account not found",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "twitter error 399: Sorry, we could not find your account.",
			},
			want: SocialExecutionFailureAuthInvalid,
		},
		{
			name: "wrong password",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "twitter error 399: Wrong password!",
			},
			want: SocialExecutionFailurePasswordInvalid,
		},
		{
			name: "twitter code without exact message stays platform",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "twitter error 399",
			},
			want: SocialExecutionFailurePlatform,
		},
		{
			name: "twitter code with unknown message stays platform",
			result: twitterActionResult{
				StatusCode: http.StatusForbidden,
				Message:    "twitter error 399: Password checkpoint required.",
			},
			want: SocialExecutionFailurePlatform,
		},
		{
			name: "auth token invalid",
			result: twitterActionResult{
				StatusCode: http.StatusUnauthorized,
				Message:    "twitter error 89: Invalid or expired token.",
			},
			want: SocialExecutionFailureAuthInvalid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, twitterFailureKind(&tc.result))
		})
	}
}

func TestSocialTaskExecutorWritesBackChallengeRequiredFailureState(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-challenge-required@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("challenge_required_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("challenge_required_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "123456789"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "", newSocialExecutionError(
			SocialExecutionFailureChallengeRequired,
			"additional verification required",
			nil,
		)
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "账号需要额外验证，本次未扣费", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusPendingCheck, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusManualReview, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "账号需要额外验证，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSocialTaskExecutorStoresSafeMediaUploadFailureMessage(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-media-upload-failure@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_media_upload_failure").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_media_upload_failure").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionPost).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetPayload(SocialTaskPayload{
			Post: &SocialPostPayload{
				Text: "hello image",
				Media: []SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "social-task/post.png",
					ContentType: "image/png",
					FileName:    "post.png",
				}},
			},
		}).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "", newSocialExecutionError(
			SocialExecutionFailurePlatform,
			"twitter media upload returned no media id",
			nil,
		)
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "平台媒体上传失败，本次未扣费", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "平台媒体上传失败，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSocialTaskExecutorWritesBackAccountLimitedFailureState(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-account-limited@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("account_limited_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("account_limited_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "123456789"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "", newSocialExecutionError(
			SocialExecutionFailureAccountLimited,
			"request looks automated",
			nil,
		)
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "账号状态或频率受限，本次未扣费", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusLimited, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusManualReview, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "账号状态或频率受限，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestAccountWorkbenchServiceFailsClosedWhenExecutorStopped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("executor-stopped@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_stopped").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_stopped").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.Stop()
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, executor)
	target := "@target"
	billingRequestID := "stopped-executor-charge-request"

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:             AccountWorkbenchTaskModeAdmin,
		UserID:           user.ID,
		AccountIDs:       []int64{account.ID},
		Action:           SocialTaskActionFollow,
		Target:           &target,
		BillingRequestID: &billingRequestID,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Submitted)
	require.Zero(t, result.Enqueued)
	require.Equal(t, 1, result.FailedClosed)
	require.Len(t, result.Logs, 1)
	require.Equal(t, SocialTaskLogStatusFailed, result.Logs[0].Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, result.Logs[0].ChargeStatus)
	require.Zero(t, result.Logs[0].ChargedAmount)
	require.Nil(t, result.Logs[0].BillingRequestID)
	require.NotNil(t, result.Logs[0].ResultMessage)
	require.Equal(t, "social platform executor queue is not configured; task was not charged", *result.Logs[0].ResultMessage)

	stored, err := client.SocialTaskLog.Get(ctx, result.Logs[0].ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.BillingRequestID)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "social platform executor queue is not configured; task was not charged", *storedAccount.TaskMessage)
}

func TestAccountWorkbenchServiceReturnsErrorWhenFailClosedStatusCannotPersist(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("executor-fail-closed-persist@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_fail_closed_persist").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_fail_closed_persist").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "@target"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), NewSocialBillingService(nil, nil, nil, nil), nil)
	logs := []*SocialTaskLog{socialTaskLogFromEnt(task)}
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	enqueued, failedClosed, err := workbench.enqueueOrFailClosed(canceledCtx, []int64{task.ID}, &logs)

	require.Error(t, err)
	require.Zero(t, enqueued)
	require.Zero(t, failedClosed)

	stored, getErr := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, getErr)
	require.Equal(t, SocialTaskLogStatusPending, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
}

func TestSocialAccountServiceDoesNotFailClosedFinalizedTaskLog(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("finalized-log-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("finalized_log_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("finalized_log_account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	chargeSource := SocialTaskChargeSourceWallet
	billingRequestID := "wallet:already-finalized"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusSuccess).
		SetResultMessage("already succeeded").
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(SocialTaskUnitPrice).
		SetChargeStatus(SocialTaskChargeStatusCharged).
		SetChargeSource(chargeSource).
		SetBillingRequestID(billingRequestID).
		SaveX(ctx)

	updated, err := accountSvc.MarkTaskLogFailedNotCharged(ctx, task.ID, "queue is full")

	require.Nil(t, updated)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, infraerrors.Code(err))
	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, stored.Status)
	require.Equal(t, "already succeeded", *stored.ResultMessage)
	require.Equal(t, SocialTaskChargeStatusCharged, stored.ChargeStatus)
	require.InEpsilon(t, SocialTaskUnitPrice, stored.ChargedAmount, 0.000001)
	require.NotNil(t, stored.ChargeSource)
	require.Equal(t, chargeSource, *stored.ChargeSource)
	require.NotNil(t, stored.BillingRequestID)
	require.Equal(t, billingRequestID, *stored.BillingRequestID)
}

func TestSocialAccountServiceFailClosedTaskLogUpdatesAccountVisibleState(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("fail-closed-account-state@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("fail_closed_account_state").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("fail_closed_account_state").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusPending).
		SetTaskMessage("previous pending message").
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	updated, err := accountSvc.MarkTaskLogFailedNotCharged(ctx, task.ID, "任务队列繁忙，本次未扣费")

	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, updated.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, updated.ChargeStatus)
	require.Zero(t, updated.ChargedAmount)
	require.NotNil(t, updated.ResultMessage)
	require.Equal(t, "任务队列繁忙，本次未扣费", *updated.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "任务队列繁忙，本次未扣费", *storedAccount.TaskMessage)
}

func TestTwitterExecutorFailsClosedWithoutExecutionAuth(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-no-auth@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("twitter_no_auth").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("twitter_no_auth").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	target := "123456789"
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", newEncryptedTwitterExecutorForTest())

	executor.processTask(task.ID)

	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *stored.ResultMessage)
	require.NotContains(t, *stored.ResultMessage, "execution_auth")

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusNotStored, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusManualReview, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "账号认证信息不可用，本次未扣费", *storedAccount.TaskMessage)
	require.NotContains(t, *storedAccount.TaskMessage, "auth cookie")
}

func TestTwitterExecutorSuccessChargesOnlyAfterSuccessfulAction(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-success@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := client.SocialAccount.Create().
		SetName("twitter_success").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("twitter_success").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SetExecutionAuth(executionAuth).
		SaveX(ctx)
	target := "123456789"
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.Path, "/friendships/create.json")
		require.Contains(t, req.Header.Get("Authorization"), "OAuth ")
		raw, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(raw), "user_id=123456789")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}
	billing := NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", twitterExec)

	executor.processTask(task.ID)

	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, stored.Status)
	require.Equal(t, SocialTaskChargeStatusCharged, stored.ChargeStatus)
	require.Equal(t, SocialTaskChargeSourceWallet, *stored.ChargeSource)
	require.InEpsilon(t, SocialTaskUnitPrice, stored.ChargedAmount, 0.000001)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "follow succeeded", *stored.ResultMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.90, storedUser.Balance, 0.000001)
	require.Equal(t, 1, fakeTransport.calls)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "follow succeeded", *storedAccount.TaskMessage)
}

func TestSocialTaskExecutorDoesNotMarkAccountSuccessfulWhenBillingFails(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-billing-fails@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("billing_failure_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("billing_failure_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	target := "123456789"
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
		return "follow succeeded", nil
	}))

	executor.processTask(task.ID)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "执行已完成，但扣费确认异常，请联系管理员处理", *storedTask.ResultMessage)
	require.NotContains(t, *storedTask.ResultMessage, "insufficient")

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "执行已完成，但扣费确认异常，请联系管理员处理", *storedAccount.TaskMessage)
}

func TestTwitterExecutorSendsRealLoginLikePostAndRetweetRequests(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	account := &dbent.SocialAccount{
		ID:            101,
		Platform:      "x_twitter",
		PlatformKey:   "x_twitter",
		ExecutionAuth: &executionAuth,
	}

	for _, tc := range []struct {
		name       string
		action     string
		target     *string
		content    *string
		method     string
		endpoint   string
		response   string
		wantBody   []string
		wantResult string
	}{
		{
			name:       "login_check",
			action:     SocialTaskActionLoginCheck,
			method:     http.MethodGet,
			endpoint:   "/graphql/ViewerUser",
			response:   `{"data":{"viewer":{"userResult":{"result":{"rest_id":"42"}}}}}`,
			wantResult: "login check succeeded",
		},
		{
			name:       "like",
			action:     SocialTaskActionLike,
			target:     socialStringPtr("https://x.com/example/status/123456789"),
			method:     http.MethodPost,
			endpoint:   "/graphql/FavoriteTweet",
			response:   `{"data":{"favorite_tweet":"ok"}}`,
			wantBody:   []string{"variables", "123456789"},
			wantResult: "like succeeded",
		},
		{
			name:       "post",
			action:     SocialTaskActionPost,
			content:    socialStringPtr("hello from SocialOps"),
			method:     http.MethodPost,
			endpoint:   "/graphql/CreateTweet",
			response:   `{"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"88"}}}}}`,
			wantBody:   []string{"features", "variables", "hello from SocialOps"},
			wantResult: "post succeeded",
		},
		{
			name:       "retweet",
			action:     SocialTaskActionRetweet,
			target:     socialStringPtr("https://x.com/example/status/987654321"),
			method:     http.MethodPost,
			endpoint:   "/graphql/CreateRetweet",
			response:   `{"data":{"create_retweet":"ok"}}`,
			wantBody:   []string{"features", "variables", "987654321"},
			wantResult: "retweet succeeded",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
				require.Equal(t, tc.method, req.Method)
				require.Contains(t, req.URL.Path, tc.endpoint)
				require.Contains(t, req.Header.Get("Authorization"), "OAuth ")
				require.Equal(t, "TwitterAndroid", req.Header.Get("X-Twitter-Client"))
				var raw []byte
				if req.Body != nil {
					var err error
					raw, err = io.ReadAll(req.Body)
					require.NoError(t, err)
				}
				for _, part := range tc.wantBody {
					require.Contains(t, string(raw), part)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(tc.response)),
					Request:    req,
				}, nil
			}}
			twitterExec := newEncryptedTwitterExecutorForTest()
			twitterExec.endpoints = twitterEndpoints{
				createFriendship: "https://twitter.test/1.1/friendships/create.json",
				favoriteTweet:    "https://twitter.test/graphql/FavoriteTweet",
				createTweet:      "https://twitter.test/graphql/CreateTweet",
				createRetweet:    "https://twitter.test/graphql/CreateRetweet",
				viewerUser:       "https://twitter.test/graphql/ViewerUser",
			}
			twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
				require.Equal(t, "http://8.8.8.8:8080", proxyURL)
				return &http.Client{Transport: fakeTransport}, nil
			}
			task := &dbent.SocialTaskLog{
				ID:            202,
				Action:        tc.action,
				Target:        tc.target,
				Content:       tc.content,
				ProxySnapshot: &proxySnapshot,
			}

			result, err := twitterExec.Execute(ctx, task, account)

			require.NoError(t, err)
			require.Equal(t, tc.wantResult, result)
			require.Equal(t, 1, fakeTransport.calls)
		})
	}
}

func TestTwitterExecutorOfflineProxyMarksAccountIPUnavailableOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-offline-proxy@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := client.SocialAccount.Create().
		SetName("twitter_offline_proxy").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("twitter_offline_proxy").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SetExecutionAuth(executionAuth).
		SaveX(ctx)
	target := "123456789"
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"offline"}`
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetTarget(target).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", newEncryptedTwitterExecutorForTest())

	executor.processTask(task.ID)

	stored, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "执行代理不可用，本次未扣费", *stored.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusIPUnavailable, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "执行代理不可用，本次未扣费", *storedAccount.TaskMessage)
}

func TestTwitterExecutorUsesStructuredPostPayloadForQuoteTweet(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 1, Name: "@quote_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            301,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello quote"),
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Post: &SocialPostPayload{
				Text:         "hello quote",
				QuotePostURL: "https://x.com/northwind/status/1",
			},
		},
	}

	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://twitter.test/graphql/CreateTweet", req.URL.String())
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var payload map[string]string
		require.NoError(t, json.Unmarshal(body, &payload))
		var variables map[string]any
		require.NoError(t, json.Unmarshal([]byte(payload["variables"]), &variables))
		require.Equal(t, "hello quote", variables["tweet_text"])
		require.Equal(t, "https://x.com/northwind/status/1", variables["attachment_url"])
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"1"}}}}}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		createTweet: "https://twitter.test/graphql/CreateTweet",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "post succeeded", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestSocialTaskExecutorProcessesMediaOnlyPostPayload(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("executor-media-only-post@example.com").
		SetPasswordHash("hash").
		SetBalance(1).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_media_only_post").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_media_only_post").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionPost).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(0).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SetPayload(SocialTaskPayload{
			Post: &SocialPostPayload{
				Media: []SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "social-task/media-only-post.png",
					ContentType: "image/png",
					FileName:    "media-only-post.png",
				}},
			},
		}).
		SaveX(ctx)

	executed := false
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", socialPlatformExecutorFunc(func(_ context.Context, taskLog *dbent.SocialTaskLog, socialAccount *dbent.SocialAccount) (string, error) {
		executed = true
		require.Equal(t, task.ID, taskLog.ID)
		require.Equal(t, account.ID, socialAccount.ID)
		require.Nil(t, taskLog.Content)
		require.NotNil(t, taskLog.Payload.Post)
		require.Equal(t, "", taskLog.Payload.Post.Text)
		require.Len(t, taskLog.Payload.Post.Media, 1)
		require.Equal(t, "social-task/media-only-post.png", taskLog.Payload.Post.Media[0].StorageKey)
		return "media-only post succeeded", nil
	}))

	executor.processTask(task.ID)

	require.True(t, executed)

	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.NotNil(t, storedTask.ResultMessage)
	require.Equal(t, "media-only post succeeded", *storedTask.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "media-only post succeeded", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestTwitterExecutorUsesStructuredInlineImageMediaForPost(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 11, Name: "@image_post_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            311,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello image"),
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Post: &SocialPostPayload{
				Text: "hello image",
				Media: []SocialTaskMediaRef{
					{
						Source:      "inline",
						ContentType: "image/png",
						FileName:    "inline.png",
						URL:         inlinePNGDataURLForSocialTaskValidation(t, 640, 640),
					},
				},
			},
		},
	}

	requests := 0
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			require.Equal(t, http.MethodPost, req.Method)
			require.Contains(t, req.URL.String(), "https://twitter.test/1.1/media/upload.json")
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `name="media"`)
			require.Contains(t, string(body), `filename="inline.png"`)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"media_id_string":"123456789"}`)),
				Request:    req,
			}, nil
		case 2:
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "https://twitter.test/graphql/CreateTweet", req.URL.String())
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			var payload map[string]string
			require.NoError(t, json.Unmarshal(body, &payload))
			var variables map[string]any
			require.NoError(t, json.Unmarshal([]byte(payload["variables"]), &variables))
			require.Equal(t, "hello image", variables["tweet_text"])
			media, ok := variables["media"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, false, media["possibly_sensitive"])
			entities, ok := media["media_entities"].([]any)
			require.True(t, ok)
			require.Len(t, entities, 1)
			first, ok := entities[0].(map[string]any)
			require.True(t, ok)
			require.Equal(t, float64(123456789), first["media_id"])
			require.Empty(t, first["tagged_users"])
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"99"}}}}}`)),
				Request:    req,
			}, nil
		default:
			t.Fatalf("unexpected twitter request %d to %s", requests, req.URL.String())
			return nil, nil
		}
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		mediaUpload: "https://twitter.test/1.1/media/upload.json",
		createTweet: "https://twitter.test/graphql/CreateTweet",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "post succeeded", result)
	require.Equal(t, 2, fakeTransport.calls)
}

func TestTwitterExecutorUsesTaskMediaAssetForPost(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-task-media@example.com").SetPasswordHash("hash").SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 11, Name: "@image_post_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	mediaSvc := NewSocialTaskMediaService(client)
	inlineRef := SocialTaskMediaRef{
		Source:      "inline",
		ContentType: "image/png",
		FileName:    "stored.png",
		URL:         inlinePNGDataURLForSocialTaskValidation(t, 640, 640),
	}
	payload, _, err := mediaSvc.MaterializeTaskLogMedia(ctx, user.ID, &SocialTaskPayload{
		Post: &SocialPostPayload{
			Text:  "hello image",
			Media: []SocialTaskMediaRef{inlineRef},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.Post)
	require.Len(t, payload.Post.Media, 1)
	require.Equal(t, "library", payload.Post.Media[0].Source)

	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            411,
		UserID:        user.ID,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello image"),
		ProxySnapshot: &proxySnapshot,
		Payload:       *payload,
	}

	requests := 0
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			require.Equal(t, http.MethodPost, req.Method)
			require.Contains(t, req.URL.String(), "https://twitter.test/1.1/media/upload.json")
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `name="media"`)
			require.Contains(t, string(body), `filename="stored.png"`)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"media_id_string":"123456789"}`)),
				Request:    req,
			}, nil
		case 2:
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "https://twitter.test/graphql/CreateTweet", req.URL.String())
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			var graphPayload map[string]string
			require.NoError(t, json.Unmarshal(body, &graphPayload))
			var variables map[string]any
			require.NoError(t, json.Unmarshal([]byte(graphPayload["variables"]), &variables))
			require.Equal(t, "hello image", variables["tweet_text"])
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"99"}}}}}`)),
				Request:    req,
			}, nil
		default:
			t.Fatalf("unexpected twitter request %d to %s", requests, req.URL.String())
			return nil, nil
		}
	}}
	twitterExec := newEncryptedTwitterExecutorForTest().WithMediaResolver(mediaSvc)
	twitterExec.endpoints = twitterEndpoints{
		mediaUpload: "https://twitter.test/1.1/media/upload.json",
		createTweet: "https://twitter.test/graphql/CreateTweet",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "post succeeded", result)
	require.Equal(t, 2, fakeTransport.calls)
}

func TestTwitterExecutorRejectsMissingTaskMediaAssetForPost(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-missing-task-media@example.com").SetPasswordHash("hash").SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 12, Name: "@missing_media_post_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            412,
		UserID:        user.ID,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello image"),
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Post: &SocialPostPayload{
				Text: "hello image",
				Media: []SocialTaskMediaRef{{
					Source:      "library",
					StorageKey:  "social-task/missing/image.png",
					ContentType: "image/png",
					FileName:    "missing.png",
				}},
			},
		},
	}
	twitterExec := newEncryptedTwitterExecutorForTest().WithMediaResolver(NewSocialTaskMediaService(client))
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("missing task media should fail before building HTTP client")
		return nil, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.Empty(t, result)
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureActionInput, kind)
	require.ErrorContains(t, err, "post media #1 media asset is unavailable")
}

func TestTwitterExecutorFailsClosedForStructuredVideoMediaBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 12, Name: "@video_post_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            312,
		Action:        SocialTaskActionPost,
		Content:       socialStringPtr("hello video"),
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Post: &SocialPostPayload{
				Text: "hello video",
				Media: []SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "video/mp4",
					FileName:    "clip.mp4",
					URL:         "data:video/mp4;base64,AAECAwQFBgcICQ==",
				}},
			},
		},
	}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("video media should fail closed before building HTTP client")
		return nil, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.Empty(t, result)
	require.Error(t, err)
	require.ErrorContains(t, err, "video media is not supported for SocialOps execution")
}

func TestTwitterExecutorUsesStructuredProfilePayloadForUpdateProfile(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 2, Name: "@profile_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            302,
		Action:        SocialTaskActionUpdateProfile,
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Profile: &SocialProfileUpdateParams{
				DisplayName: "Northwind Ops",
				ScreenName:  "northwind_ops",
				Description: "Operator account",
				Location:    "Singapore",
				URL:         "https://example.com",
			},
		},
	}

	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		values, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, "northwind_ops", values.Get("screen_name"))
		require.Equal(t, "Northwind Ops", values.Get("name"))
		require.Equal(t, "Operator account", values.Get("description"))
		require.Equal(t, "Singapore", values.Get("location"))
		require.Equal(t, "https://example.com", values.Get("url"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"screen_name":"northwind_ops"}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		updateProfile: "https://twitter.test/1.1/account/update_profile.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "profile updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestTwitterExecutorRejectsUpdateAvatarWithoutMedia(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 3, Name: "@avatar_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            303,
		Action:        SocialTaskActionUpdateAvatar,
		ProxySnapshot: &proxySnapshot,
		Payload:       SocialTaskPayload{},
	}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("avatar update without media should fail before building HTTP client")
		return nil, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.Empty(t, result)
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureActionInput, kind)
	require.ErrorContains(t, err, "avatar media is required")
}

func TestTwitterExecutorNormalizesAvatarImageBeforeUpload(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 13, Name: "@avatar_dimension_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            313,
		Action:        SocialTaskActionUpdateAvatar,
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/png",
				FileName:    "avatar.png",
				URL:         inlinePNGDataURLForSocialTaskValidation(t, 300, 300),
			},
		},
	}
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile_image.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `name="image"`)
		require.Contains(t, string(body), `filename="avatar.jpg"`)
		require.Contains(t, string(body), "Content-Type: image/jpeg")
		requireImagePartDimensions(t, req.Header.Get("Content-Type"), body, "image", 400, 400)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		updateProfileImage: "https://twitter.test/1.1/account/update_profile_image.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "avatar updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestTwitterExecutorUsesTaskMediaAssetForUpdateAvatar(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-avatar-task-media@example.com").SetPasswordHash("hash").SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 31, Name: "@avatar_asset_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	mediaSvc := NewSocialTaskMediaService(client)
	payload, _, err := mediaSvc.MaterializeTaskLogMedia(ctx, user.ID, &SocialTaskPayload{
		Avatar: &SocialTaskMediaRef{
			Source:      "inline",
			ContentType: "image/png",
			FileName:    "avatar.png",
			URL:         inlinePNGDataURLForSocialTaskValidation(t, 400, 400),
		},
	}, nil)
	require.NoError(t, err)

	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            513,
		UserID:        user.ID,
		Action:        SocialTaskActionUpdateAvatar,
		ProxySnapshot: &proxySnapshot,
		Payload:       *payload,
	}
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile_image.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `name="image"`)
		require.Contains(t, string(body), `filename="avatar.png"`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest().WithMediaResolver(mediaSvc)
	twitterExec.endpoints = twitterEndpoints{
		updateProfileImage: "https://twitter.test/1.1/account/update_profile_image.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "avatar updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestTwitterExecutorRejectsUpdateBannerWithoutMedia(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 4, Name: "@banner_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            304,
		Action:        SocialTaskActionUpdateBanner,
		ProxySnapshot: &proxySnapshot,
		Payload:       SocialTaskPayload{},
	}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		t.Fatal("banner update without media should fail before building HTTP client")
		return nil, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.Empty(t, result)
	require.Error(t, err)
	kind, ok := socialExecutionFailureKind(err)
	require.True(t, ok)
	require.Equal(t, SocialExecutionFailureActionInput, kind)
	require.ErrorContains(t, err, "banner media is required")
}

func TestTwitterExecutorUsesStructuredBannerPayloadForUpdateBanner(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 4, Name: "@banner_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            304,
		Action:        SocialTaskActionUpdateBanner,
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/jpeg",
				URL:         inlineJPEGDataURLForSocialTaskValidation(t, 1500, 500),
			},
		},
	}

	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile_banner.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `name="banner"`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		updateProfileBanner: "https://twitter.test/1.1/account/update_profile_banner.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "banner updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestTwitterExecutorUsesTaskMediaAssetForUpdateBanner(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("twitter-banner-task-media@example.com").SetPasswordHash("hash").SaveX(ctx)
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 41, Name: "@banner_asset_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	mediaSvc := NewSocialTaskMediaService(client)
	payload, _, err := mediaSvc.MaterializeTaskLogMedia(ctx, user.ID, &SocialTaskPayload{
		Banner: &SocialTaskMediaRef{
			Source:      "inline",
			ContentType: "image/jpeg",
			FileName:    "banner.jpg",
			URL:         inlineJPEGDataURLForSocialTaskValidation(t, 1500, 500),
		},
	}, nil)
	require.NoError(t, err)

	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            514,
		UserID:        user.ID,
		Action:        SocialTaskActionUpdateBanner,
		ProxySnapshot: &proxySnapshot,
		Payload:       *payload,
	}
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile_banner.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `name="banner"`)
		require.Contains(t, string(body), `filename="banner.jpg"`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest().WithMediaResolver(mediaSvc)
	twitterExec.endpoints = twitterEndpoints{
		updateProfileBanner: "https://twitter.test/1.1/account/update_profile_banner.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "banner updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestTwitterExecutorNormalizesBannerImageBeforeUpload(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	account := &dbent.SocialAccount{ID: 14, Name: "@banner_dimension_account", Platform: "x_twitter", ExecutionAuth: &executionAuth}
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	task := &dbent.SocialTaskLog{
		ID:            314,
		Action:        SocialTaskActionUpdateBanner,
		ProxySnapshot: &proxySnapshot,
		Payload: SocialTaskPayload{
			Banner: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "image/jpeg",
				FileName:    "banner.jpg",
				URL:         inlineJPEGDataURLForSocialTaskValidation(t, 1400, 500),
			},
		},
	}
	fakeTransport := &twitterFakeRoundTripper{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "https://twitter.test/1.1/account/update_profile_banner.json")
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `name="banner"`)
		require.Contains(t, string(body), `filename="banner.jpg"`)
		require.Contains(t, string(body), "Content-Type: image/jpeg")
		requireImagePartDimensions(t, req.Header.Get("Content-Type"), body, "banner", 1500, 500)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	}}
	twitterExec := newEncryptedTwitterExecutorForTest()
	twitterExec.endpoints = twitterEndpoints{
		updateProfileBanner: "https://twitter.test/1.1/account/update_profile_banner.json",
	}
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}

	result, err := twitterExec.Execute(ctx, task, account)

	require.NoError(t, err)
	require.Equal(t, "banner updated", result)
	require.Equal(t, 1, fakeTransport.calls)
}

func TestSocialTaskExecutorFailsClosedWithSpecificAvatarAndBannerInvalidMediaMessages(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`

	testCases := []struct {
		name            string
		action          string
		payload         SocialTaskPayload
		expectedMessage string
	}{
		{
			name:   "avatar invalid media",
			action: SocialTaskActionUpdateAvatar,
			payload: SocialTaskPayload{
				Avatar: &SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/png",
					FileName:    "avatar.png",
					URL:         "data:image/png;base64,not-base64",
				},
			},
			expectedMessage: "任务参数不完整，本次未扣费",
		},
		{
			name:   "banner invalid media",
			action: SocialTaskActionUpdateBanner,
			payload: SocialTaskPayload{
				Banner: &SocialTaskMediaRef{
					Source:      "inline",
					ContentType: "image/jpeg",
					FileName:    "banner.jpg",
					URL:         "data:image/jpeg;base64,not-base64",
				},
			},
			expectedMessage: "任务参数不完整，本次未扣费",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSocialOpsServiceTestClient(t)
			user := client.User.Create().SetEmail(strings.ReplaceAll(tc.action, "_", "-") + "-dimension@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
			account := client.SocialAccount.Create().
				SetName(strings.ReplaceAll(tc.action, "_", "_") + "_dimension_account").
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(strings.ReplaceAll(tc.action, "_", "_") + "_dimension_account").
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusPending).
				SetExecutionAuth(executionAuth).
				SaveX(ctx)
			task := client.SocialTaskLog.Create().
				SetSocialAccountID(account.ID).
				SetUserID(user.ID).
				SetAction(tc.action).
				SetProxySnapshot(proxySnapshot).
				SetPayload(tc.payload).
				SetStatus(SocialTaskLogStatusPending).
				SetPrice(SocialTaskUnitPrice).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				SaveX(ctx)

			billing := NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
			executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
			executor.RegisterPlatformExecutor("x_twitter", newEncryptedTwitterExecutorForTest())

			executor.processTask(task.ID)

			stored, err := client.SocialTaskLog.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
			require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
			require.Zero(t, stored.ChargedAmount)
			require.Nil(t, stored.ChargeSource)
			require.Nil(t, stored.BillingRequestID)
			require.NotNil(t, stored.ResultMessage)
			require.Equal(t, tc.expectedMessage, *stored.ResultMessage)

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)

			storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
			require.NoError(t, err)
			require.NotNil(t, storedAccount.TaskMessage)
			require.Equal(t, tc.expectedMessage, *storedAccount.TaskMessage)
		})
	}
}

func TestSocialTaskExecutorFailsClosedWithSpecificPostMediaMessages(t *testing.T) {
	ctx := context.Background()
	executionAuth := encryptedTwitterExecutionAuthForTest(t)
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`

	testCases := []struct {
		name            string
		payload         SocialTaskPayload
		expectedMessage string
	}{
		{
			name: "post video unavailable",
			payload: SocialTaskPayload{
				Post: &SocialPostPayload{
					Text: "hello video",
					Media: []SocialTaskMediaRef{{
						Source:      "inline",
						ContentType: "video/mp4",
						FileName:    "clip.mp4",
						URL:         "data:video/mp4;base64,QUJD",
					}},
				},
			},
			expectedMessage: "视频发帖媒体暂未开放，本次未扣费",
		},
		{
			name: "post media type unsupported",
			payload: SocialTaskPayload{
				Post: &SocialPostPayload{
					Text: "hello file",
					Media: []SocialTaskMediaRef{{
						Source:      "inline",
						ContentType: "application/pdf",
						FileName:    "spec.pdf",
						URL:         "data:application/pdf;base64,QUJD",
					}},
				},
			},
			expectedMessage: "发帖媒体类型暂不支持，本次未扣费",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSocialOpsServiceTestClient(t)
			user := client.User.Create().SetEmail(strings.ReplaceAll(tc.name, " ", "-") + "@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
			account := client.SocialAccount.Create().
				SetName(strings.ReplaceAll(tc.name, " ", "_")).
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(strings.ReplaceAll(tc.name, " ", "_")).
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusPending).
				SetExecutionAuth(executionAuth).
				SaveX(ctx)
			task := client.SocialTaskLog.Create().
				SetSocialAccountID(account.ID).
				SetUserID(user.ID).
				SetAction(SocialTaskActionPost).
				SetContent("hello post media").
				SetProxySnapshot(proxySnapshot).
				SetPayload(tc.payload).
				SetStatus(SocialTaskLogStatusPending).
				SetPrice(SocialTaskUnitPrice).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				SaveX(ctx)

			billing := NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
			executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).WithCredentialEncryptor(executionAuthEncryptorStub{})
			executor.RegisterPlatformExecutor("x_twitter", newEncryptedTwitterExecutorForTest())

			executor.processTask(task.ID)

			stored, err := client.SocialTaskLog.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
			require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
			require.Zero(t, stored.ChargedAmount)
			require.Nil(t, stored.ChargeSource)
			require.Nil(t, stored.BillingRequestID)
			require.NotNil(t, stored.ResultMessage)
			require.Equal(t, tc.expectedMessage, *stored.ResultMessage)

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)

			storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
			require.NoError(t, err)
			require.NotNil(t, storedAccount.TaskMessage)
			require.Equal(t, tc.expectedMessage, *storedAccount.TaskMessage)
			require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
		})
	}
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

func TestDynamicProxyEndpointFromPayloadSupportsProviderFormats(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "data without credentials",
			body: `{"code":0,"success":true,"msg":"Successfully obtained","data":[{"ip":"51.79.203.53","port":16719}],"request_ip":"120.231.215.225"}`,
			want: "http://51.79.203.53:16719",
		},
		{
			name: "final with credentials",
			body: `{"final":[{"ip":"8.8.8.8","port":18080,"username":"bunnyqFPlqTKw-random-any-session-994682978-sessTime-2","password":"57668898"}]}`,
			want: "http://bunnyqFPlqTKw-random-any-session-994682978-sessTime-2:57668898@8.8.8.8:18080",
		},
		{
			name: "data with credentials",
			body: `{"success":true,"data":[{"username":"3486581-yishou123","password":"Aa123456-global-131608830","ip":"1.1.1.1","port":1000}],"msg":"操作成功","code":0}`,
			want: "http://3486581-yishou123:Aa123456-global-131608830@1.1.1.1:1000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dynamicProxyEndpointFromPayload([]byte(tc.body))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSocialIPServiceAcceptsDynamicProxySourceEndpoint(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("dynamic-proxy-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	endpoint := "https://8.8.8.8/proxy-api?count=1"
	ip, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "dynamic source",
		IPType:   "residential",
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	require.NotNil(t, ip.Endpoint)
	require.Equal(t, endpoint, *ip.Endpoint)
	require.Equal(t, SocialIPStatusUnknown, ip.Status)
}

func TestSocialIPServiceAcceptsIPWODynamicProxySourceEndpoint(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("ipwo-dynamic-proxy-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	endpoint := "https://www.ipwo.net/api/proxy/get_proxy_ip?num=1&regions=GLOBAL&protocol=http&return_type=json&lb=1"
	ip, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "ipwo source",
		IPType:   "residential",
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	require.NotNil(t, ip.Endpoint)
	require.Equal(t, endpoint, *ip.Endpoint)
}

func TestTwitterProxyEndpointFromTaskResolvesDynamicProxySource(t *testing.T) {
	previousClient := dynamicProxySourceHTTPClient
	dynamicProxySourceHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https://8.8.8.8/proxy-api?count=1", req.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":[{"ip":"51.79.203.53","port":16719}],"code":0}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	defer func() { dynamicProxySourceHTTPClient = previousClient }()

	source := "https://8.8.8.8/proxy-api?count=1"
	snapshot := fmt.Sprintf(`{"id":7,"name":"dynamic","endpoint":%q,"status":"online"}`, source)
	endpoint, err := twitterProxyEndpointFromTask(context.Background(), &dbent.SocialTaskLog{
		ProxySnapshot: &snapshot,
	})
	require.NoError(t, err)
	require.Equal(t, "http://51.79.203.53:16719", endpoint)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSocialAccountJSONIncludesCredentials(t *testing.T) {
	password := "secret"
	emailPassword := "mail-secret"
	authCookie := "ct0=cookie; auth_token=secret"
	executionAuth := "encrypted-social-account-json-execution-auth-ciphertext"
	account := SocialAccount{
		ID:            1,
		Name:          "x account",
		Platform:      "x",
		Password:      &password,
		EmailPassword: &emailPassword,
		AuthCookie:    &authCookie,
		ExecutionAuth: &executionAuth,
	}

	payload, err := json.Marshal(account)
	require.NoError(t, err)
	body := string(payload)
	require.Contains(t, body, `"password":"secret"`)
	require.Contains(t, body, `"email_password":"mail-secret"`)
	require.Contains(t, body, `"auth_cookie":"ct0=cookie; auth_token=secret"`)
	require.Contains(t, body, `"execution_auth":"encrypted-social-account-json-execution-auth-ciphertext"`)
	require.NotContains(t, body, "access_token")
	require.NotContains(t, body, "token_secret")
}

func TestSocialAccountServiceDoesNotExposePlainExecutionAuthPayloads(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	password := "secret"
	plainExecutionAuth := `{"access_token":"plain-access","token_secret":"plain-secret"}`
	account := client.SocialAccount.Create().
		SetName("@plain_execution_auth").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("plain_execution_auth").
		SetPassword(password).
		SetExecutionAuth(plainExecutionAuth).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	got, err := svc.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, got.ExecutionAuth)

	listed, _, err := svc.List(ctx, pagination.DefaultPagination(), SocialAccountListFilters{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Nil(t, listed[0].ExecutionAuth)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.ExecutionAuth)
	require.Equal(t, plainExecutionAuth, *stored.ExecutionAuth)
}

func TestSocialAccountServiceStoresExecutionAuthEncryptedAndOtherCredentialsPlain(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountServiceWithCredentialEncryptor(client, executionAuthEncryptorStub{})

	password := "x-account-secret"
	emailPassword := "mailbox-secret"
	twoFactor := "  create-2fa  "
	backupCode := "  create-backup  "
	emailClientID := "  create-client  "
	emailToken := "  create-token  "
	authCookie := "  ct0=create; auth_token=create  "
	executionAuth := `{"access_token":"access","token_secret":"secret"}`
	normalizedExecutionAuth := `{"access_token":"access","token_secret":"secret","screen_name":"northwind_ops"}`
	directProxySnapshot := `{"id":99,"name":"stale-proxy"}`
	account, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:                 "northwind_ops",
		Platform:             "x_twitter",
		Password:             &password,
		EmailPassword:        &emailPassword,
		TwoFactor:            &twoFactor,
		BackupCode:           &backupCode,
		EmailClientID:        &emailClientID,
		EmailToken:           &emailToken,
		AuthCookie:           &authCookie,
		ExecutionAuth:        &executionAuth,
		DefaultProxySnapshot: &directProxySnapshot,
	})
	require.NoError(t, err)
	require.NotNil(t, account.Password)
	require.NotNil(t, account.EmailPassword)
	require.NotNil(t, account.TwoFactor)
	require.NotNil(t, account.BackupCode)
	require.NotNil(t, account.EmailClientID)
	require.NotNil(t, account.EmailToken)
	require.NotNil(t, account.AuthCookie)
	require.NotNil(t, account.ExecutionAuth)
	require.NotNil(t, account.DefaultProxySnapshot)
	require.Equal(t, password, *account.Password)
	require.Equal(t, emailPassword, *account.EmailPassword)
	require.Equal(t, twoFactor, *account.TwoFactor)
	require.Equal(t, backupCode, *account.BackupCode)
	require.Equal(t, emailClientID, *account.EmailClientID)
	require.Equal(t, emailToken, *account.EmailToken)
	require.Equal(t, authCookie, *account.AuthCookie)
	require.NotContains(t, *account.ExecutionAuth, "access")
	require.NotContains(t, *account.ExecutionAuth, "token_secret")
	decryptedExecutionAuth, err := decryptTwitterExecutionAuthCiphertext(*account.ExecutionAuth, executionAuthEncryptorStub{})
	require.NoError(t, err)
	require.Equal(t, normalizedExecutionAuth, decryptedExecutionAuth)
	require.Equal(t, directProxySnapshot, *account.DefaultProxySnapshot)

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Password)
	require.NotNil(t, stored.EmailPassword)
	require.NotNil(t, stored.TwoFactor)
	require.NotNil(t, stored.BackupCode)
	require.NotNil(t, stored.EmailClientID)
	require.NotNil(t, stored.EmailToken)
	require.NotNil(t, stored.AuthCookie)
	require.NotNil(t, stored.ExecutionAuth)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Equal(t, password, *stored.Password)
	require.Equal(t, emailPassword, *stored.EmailPassword)
	require.Equal(t, twoFactor, *stored.TwoFactor)
	require.Equal(t, backupCode, *stored.BackupCode)
	require.Equal(t, emailClientID, *stored.EmailClientID)
	require.Equal(t, emailToken, *stored.EmailToken)
	require.Equal(t, authCookie, *stored.AuthCookie)
	require.Equal(t, *account.ExecutionAuth, *stored.ExecutionAuth)
	require.Equal(t, directProxySnapshot, *stored.DefaultProxySnapshot)

	updatedPassword := "rotated-secret"
	updatedTwoFactor := "  rotated-2fa  "
	updatedBackupCode := "  rotated-backup  "
	updatedEmailClientID := "  rotated-client  "
	updatedEmailToken := "  rotated-token  "
	updatedAuthCookie := "  ct0=rotated; auth_token=rotated  "
	updatedExecutionAuth := `{"access_token":"rotated","token_secret":"secret"}`
	normalizedUpdatedExecutionAuth := `{"access_token":"rotated","token_secret":"secret","screen_name":"northwind_ops"}`
	updated, err := svc.Update(ctx, account.ID, &UpdateSocialAccountInput{
		Password:      &updatedPassword,
		TwoFactor:     &updatedTwoFactor,
		BackupCode:    &updatedBackupCode,
		EmailClientID: &updatedEmailClientID,
		EmailToken:    &updatedEmailToken,
		AuthCookie:    &updatedAuthCookie,
		ExecutionAuth: &updatedExecutionAuth,
	})
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *updated.Password)
	require.Equal(t, updatedTwoFactor, *updated.TwoFactor)
	require.Equal(t, updatedBackupCode, *updated.BackupCode)
	require.Equal(t, updatedEmailClientID, *updated.EmailClientID)
	require.Equal(t, updatedEmailToken, *updated.EmailToken)
	require.Equal(t, updatedAuthCookie, *updated.AuthCookie)
	require.NotContains(t, *updated.ExecutionAuth, "rotated")
	decryptedUpdatedExecutionAuth, err := decryptTwitterExecutionAuthCiphertext(*updated.ExecutionAuth, executionAuthEncryptorStub{})
	require.NoError(t, err)
	require.Equal(t, normalizedUpdatedExecutionAuth, decryptedUpdatedExecutionAuth)

	stored, err = client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *stored.Password)
	require.Equal(t, updatedTwoFactor, *stored.TwoFactor)
	require.Equal(t, updatedBackupCode, *stored.BackupCode)
	require.Equal(t, updatedEmailClientID, *stored.EmailClientID)
	require.Equal(t, updatedEmailToken, *stored.EmailToken)
	require.Equal(t, updatedAuthCookie, *stored.AuthCookie)
	require.Equal(t, *updated.ExecutionAuth, *stored.ExecutionAuth)

	_, err = svc.Update(ctx, account.ID, &UpdateSocialAccountInput{DefaultProxySnapshot: &directProxySnapshot})
	require.ErrorIs(t, err, ErrSocialAccountDefaultProxyRoute)
}

func TestSocialAccountServiceBatchImportRowMatchesExistingPoolOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("social-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	password := "pool-secret"
	poolPhone := "+15550001000"
	poolRemark := "pool delivery note"
	poolAccount, err := client.SocialAccount.Create().
		SetName("@NorthWind_Ops").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("northwind_ops").
		SetPassword(password).
		SetPhone(poolPhone).
		SetRemark(poolRemark).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		Save(ctx)
	require.NoError(t, err)

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "northwind_ops",
		Password:  socialStringPtr("typed-secret"),
		Phone:     socialStringPtr("+15550009999"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
		Remark:    socialStringPtr("typed import note"),
	})
	require.NoError(t, err)
	require.Equal(t, int64(poolAccount.ID), imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)
	require.Equal(t, password, *imported.Password)
	require.Equal(t, poolPhone, *imported.Phone)
	require.Equal(t, poolRemark, *imported.Remark)
	require.Nil(t, imported.DefaultProxySnapshot)

	count, err := client.SocialAccount.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@northwind_ops",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	missingImported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:      "x_twitter",
		Name:          "missing_user",
		Password:      socialStringPtr("missing-secret"),
		Phone:         socialStringPtr("+15550001111"),
		Email:         socialStringPtr("mail@example.com"),
		EmailPassword: socialStringPtr("mail-secret"),
		Remark:        socialStringPtr("fresh delivery note"),
	})
	require.NoError(t, err)
	require.Equal(t, user.ID, *missingImported.AssignedUserID)
	require.Equal(t, SocialAccountStatusNotStored, missingImported.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, missingImported.TaskStatus)
	require.Equal(t, "missing_user", missingImported.Name)
	require.Equal(t, "missing-secret", *missingImported.Password)
	require.Equal(t, "+15550001111", *missingImported.Phone)
	require.Equal(t, "mail@example.com", *missingImported.Email)
	require.Equal(t, "mail-secret", *missingImported.EmailPassword)
	require.Equal(t, "fresh delivery note", *missingImported.Remark)
}

func TestSocialAccountServiceBatchImportRowCreatesNotStoredWorkbenchAccountWhenPoolMissing(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("missing-pool-workbench@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	account, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:       "X_Twitter",
		Name:           "@missing_pool_user",
		Password:       socialStringPtr("account-secret"),
		TwoFactor:      socialStringPtr("H6X33U477GHC22AR"),
		BackupCode:     socialStringPtr("r55ghdr3pkt8"),
		Email:          socialStringPtr("mail@example.com"),
		EmailPassword:  socialStringPtr("mail-secret"),
		EmailClientID:  socialStringPtr("client-id"),
		EmailToken:     socialStringPtr("mail-token"),
		RegistrationIP: socialStringPtr("127.0.0.1"),
		Phone:          socialStringPtr("+15550002222"),
		Remark:         socialStringPtr("missing pool delivery note"),
	})
	require.NoError(t, err)
	require.Equal(t, user.ID, *account.AssignedUserID)
	require.Equal(t, "x_twitter", account.Platform)
	require.Equal(t, "@missing_pool_user", account.Name)
	require.Equal(t, SocialAccountStatusNotStored, account.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, account.TaskStatus)
	require.Equal(t, "account-secret", *account.Password)
	require.Equal(t, "mail@example.com", *account.Email)
	require.Equal(t, "mail-secret", *account.EmailPassword)
	require.Equal(t, "H6X33U477GHC22AR", *account.TwoFactor)
	require.Equal(t, "r55ghdr3pkt8", *account.BackupCode)
	require.Equal(t, "client-id", *account.EmailClientID)
	require.Equal(t, "mail-token", *account.EmailToken)
	require.Equal(t, "127.0.0.1", *account.RegistrationIP)
	require.Equal(t, "+15550002222", *account.Phone)
	require.Equal(t, "missing pool delivery note", *account.Remark)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, visible, 1)
	require.Equal(t, account.ID, visible[0].ID)
}

func TestSocialAccountServiceListTotalPoolHidesWorkbenchStagingImports(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("pool-staging-boundary@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	poolAccount := client.SocialAccount.Create().
		SetName("@pool_visible").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_visible").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	staging, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@workbench_staging_only",
		Password:  socialStringPtr("account-secret"),
		TwoFactor: socialStringPtr("H6X33U477GHC22AR"),
	})
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusNotStored, staging.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, staging.TaskStatus)

	pool, page, err := svc.ListTotalPool(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, pool, 1)
	require.Equal(t, poolAccount.ID, pool[0].ID)
}

func TestSocialAccountServiceListTotalPoolIncludesAssignedUserEmail(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("total-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@pool_assigned_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_assigned_owner").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	pool, page, err := svc.ListTotalPool(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, pool, 1)
	require.Equal(t, account.ID, pool[0].ID)
	require.NotNil(t, pool[0].AssignedUserID)
	require.Equal(t, user.ID, *pool[0].AssignedUserID)
	require.NotNil(t, pool[0].AssignedUserEmail)
	require.Equal(t, "total-owner@example.com", *pool[0].AssignedUserEmail)
}

func TestSocialAccountServiceListTotalPoolSearchesDeliveryFieldsAndOwner(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("total-search-owner@example.com").
		SetUsername("total-search-owner").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	match := client.SocialAccount.Create().
		SetName("@pool_search_match").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_search_match").
		SetIdentityKind("username").
		SetIdentityKey("pool_search_match").
		SetAssignedUserID(user.ID).
		SetPassword("pool-delivery-secret").
		SetEmailPassword("pool-email-secret").
		SetAuthCookie("ct0=pool-search-cookie; auth_token=pool-search-token").
		SetRegistrationIP("198.51.100.77").
		SetDefaultProxySnapshot(`{"id":301,"endpoint":"http://pool-search-proxy.example:8080"}`).
		SetRemark("pool-search-remark").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@pool_search_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_search_other").
		SetIdentityKind("username").
		SetIdentityKey("pool_search_other").
		SetPassword("other-delivery-secret").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	_, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@workbench_search_staging",
		Password:  socialStringPtr("pool-delivery-secret"),
		TwoFactor: socialStringPtr("H6X33U477GHC22AR"),
	})
	require.NoError(t, err)

	for _, search := range []string{
		"#" + strconv.FormatInt(match.ID, 10),
		"total-search-owner@example.com",
		"pool-delivery-secret",
		"pool-email-secret",
		"pool-search-cookie",
		"198.51.100.77",
		"pool-search-proxy",
		"pool-search-remark",
	} {
		pool, page, err := svc.ListTotalPool(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{Search: search})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Total, "search %q", search)
		require.Len(t, pool, 1, "search %q", search)
		require.Equal(t, match.ID, pool[0].ID, "search %q", search)
		require.NotNil(t, pool[0].AssignedUserEmail)
		require.Equal(t, "total-search-owner@example.com", *pool[0].AssignedUserEmail)
	}
}

func TestSocialAccountServiceListByUserSearchesDeliveryFieldsAndFilters(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("workbench-filter@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	other := client.User.Create().
		SetEmail("workbench-filter-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	match := client.SocialAccount.Create().
		SetName("@workbench_filter_match").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_filter_match").
		SetIdentityKind("username").
		SetIdentityKey("workbench_filter_match").
		SetAssignedUserID(user.ID).
		SetPassword("workbench-delivery-secret").
		SetEmailPassword("workbench-email-secret").
		SetAuthCookie("ct0=workbench-search-cookie; auth_token=workbench-search-token").
		SetRegistrationIP("198.51.100.88").
		SetDefaultProxySnapshot(`{"id":301,"endpoint":"http://workbench-search-proxy.example:8080"}`).
		SetRemark("workbench-search-remark").
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@workbench_filter_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_filter_other").
		SetAssignedUserID(user.ID).
		SetPassword("other-delivery-secret").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@workbench_filter_cross_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_filter_cross_owner").
		SetAssignedUserID(other.ID).
		SetPassword("workbench-delivery-secret").
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)

	for _, search := range []string{
		"#" + strconv.FormatInt(match.ID, 10),
		"workbench-delivery-secret",
		"workbench-email-secret",
		"workbench-search-cookie",
		"198.51.100.88",
		"workbench-search-proxy",
		"workbench-search-remark",
	} {
		visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{
			Search:        search,
			Platform:      "x_twitter",
			AccountStatus: SocialAccountStatusInvalid,
			TaskStatus:    SocialTaskStatusPending,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Total, "search %q", search)
		require.Len(t, visible, 1, "search %q", search)
		require.Equal(t, match.ID, visible[0].ID, "search %q", search)
	}
}

func TestSocialAccountServiceListByUserDefaultsToIDAscending(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("workbench-id-order@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	first := client.SocialAccount.Create().
		SetName("workbench_id_order_first").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_id_order_first").
		SetAssignedUserID(user.ID).
		SetCreatedAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)).
		SaveX(ctx)
	second := client.SocialAccount.Create().
		SetName("workbench_id_order_second").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_id_order_second").
		SetAssignedUserID(user.ID).
		SetCreatedAt(time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC)).
		SaveX(ctx)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(2), page.Total)
	require.Len(t, visible, 2)
	require.Equal(t, []int64{first.ID, second.ID}, []int64{visible[0].ID, visible[1].ID})
}

func TestSocialAccountServiceStoreWorkbenchAccountsMovesSelectedStagingIntoTotalPool(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("store-workbench@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	staging, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@workbench_upload_selected",
		Password:  socialStringPtr("account-secret"),
		TwoFactor: socialStringPtr("H6X33U477GHC22AR"),
	})
	require.NoError(t, err)

	beforePool, beforePage, err := svc.ListTotalPool(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{})
	require.NoError(t, err)
	require.Empty(t, beforePool)
	require.Equal(t, int64(0), beforePage.Total)

	result, err := svc.StoreWorkbenchAccounts(ctx, []int64{staging.ID})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Zero(t, result.Failed)
	require.Zero(t, result.Skipped)

	updated, err := svc.GetByID(ctx, staging.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusPendingCheck, updated.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, updated.TaskStatus)
	require.NotNil(t, updated.AssignedUserID)
	require.Equal(t, user.ID, *updated.AssignedUserID)

	afterPool, afterPage, err := svc.ListTotalPool(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, SocialAccountListFilters{})
	require.NoError(t, err)
	require.Equal(t, int64(1), afterPage.Total)
	require.Len(t, afterPool, 1)
	require.Equal(t, staging.ID, afterPool[0].ID)
}

func TestSocialAccountServiceStoreWorkbenchAccountsSeparatesFailedFromSkipped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("store-workbench-summary@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	validStaging, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@workbench_upload_valid",
		Password:  socialStringPtr("account-secret"),
		TwoFactor: socialStringPtr("H6X33U477GHC22AR"),
	})
	require.NoError(t, err)
	invalidStaging := client.SocialAccount.Create().
		SetName("@workbench_upload_invalid").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_upload_invalid").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	storedAccount := client.SocialAccount.Create().
		SetName("@workbench_upload_stored").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_upload_stored").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	result, err := svc.StoreWorkbenchAccounts(ctx, []int64{validStaging.ID, invalidStaging.ID, storedAccount.ID})
	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.Items, 3)
	require.Equal(t, "succeeded", result.Items[0].Status)
	require.Equal(t, "failed", result.Items[1].Status)
	require.Equal(t, "invalid_credentials", result.Items[1].Reason)
	require.Equal(t, "skipped", result.Items[2].Status)
	require.Equal(t, "already_stored", result.Items[2].Reason)

	updatedValid := client.SocialAccount.GetX(ctx, validStaging.ID)
	require.Equal(t, SocialAccountStatusPendingCheck, updatedValid.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, updatedValid.TaskStatus)
	updatedInvalid := client.SocialAccount.GetX(ctx, invalidStaging.ID)
	require.Equal(t, SocialAccountStatusNotStored, updatedInvalid.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, updatedInvalid.TaskStatus)
	updatedStored := client.SocialAccount.GetX(ctx, storedAccount.ID)
	require.Equal(t, SocialAccountStatusAvailable, updatedStored.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, updatedStored.TaskStatus)
}

func TestSocialAccountServiceBatchImportRowRejectsIncompleteCredentials(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("incomplete-import@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	cases := []*UserImportSocialAccountInput{
		{Platform: "x_twitter", Name: "@missing_password", TwoFactor: socialStringPtr("H6X33U477GHC22AR")},
		{Platform: "x_twitter", Name: "@missing_factor", Password: socialStringPtr("account-secret")},
		{Platform: "x_twitter", Name: "@missing_email_secret", Password: socialStringPtr("account-secret"), Email: socialStringPtr("mail@example.com")},
	}
	for _, input := range cases {
		_, err := svc.importUserWorkbenchAccount(ctx, user.ID, input)
		require.ErrorIs(t, err, ErrSocialAccountImportIncomplete)
	}
	count, err := client.SocialAccount.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:   "x_twitter",
		Name:       "@cookie_is_credential",
		Password:   socialStringPtr("account-secret"),
		AuthCookie: socialStringPtr("ct0=import; auth_token=import"),
	})
	require.NoError(t, err)
	require.Equal(t, "ct0=import; auth_token=import", requireSocialStringPtr(t, imported.AuthCookie))
}

func TestSocialAccountServiceDeleteForUserHardDeletesOnlyCurrentUserAccount(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("delete-workbench-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("delete-workbench-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@delete_me").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("delete_me").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLoginCheck).
		SetStatus(SocialTaskLogStatusSuccess).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	group := client.Group.Create().
		SetName("delete ledger quota").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(1).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	subscriptionLedgerRequestID := socialTaskUsageLedgerRequestID(log.ID, SocialTaskChargeSourceSubscription, sub.ID)
	subscriptionLedger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(subscriptionLedgerRequestID).
		SetModel(socialUsageLedgerModel).
		SetGroupID(group.ID).
		SetSubscriptionID(sub.ID).
		SetActualCost(0.05).
		SetTotalCost(0.05).
		SetBillingType(socialUsageBillingTypeSubscription).
		SaveX(ctx)
	walletLedgerRequestID := socialTaskUsageLedgerRequestID(log.ID, SocialTaskChargeSourceWallet, 0)
	walletLedger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(walletLedgerRequestID).
		SetModel(socialUsageLedgerModel).
		SetActualCost(0.05).
		SetTotalCost(0.05).
		SetBillingType(socialUsageBillingTypeWallet).
		SaveX(ctx)
	unrelatedLedgerRequestID := socialTaskUsageLedgerRequestID(log.ID+999, SocialTaskChargeSourceWallet, 0)
	unrelatedLedger := client.UsageLog.Create().
		SetUserID(user.ID).
		SetRequestID(unrelatedLedgerRequestID).
		SetModel(socialUsageLedgerModel).
		SetActualCost(0.1).
		SetTotalCost(0.1).
		SetBillingType(socialUsageBillingTypeWallet).
		SaveX(ctx)
	proxy := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("delete-bound-proxy").
		SetBoundSocialAccountID(account.ID).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("@delete_workbench_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("delete_workbench_other").
		SetAssignedUserID(otherUser.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	err := svc.DeleteForUser(ctx, user.ID, otherAccount.ID)
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)
	require.Equal(t, otherAccount.ID, client.SocialAccount.GetX(ctx, otherAccount.ID).ID)

	err = svc.DeleteForUser(ctx, user.ID, account.ID)
	require.NoError(t, err)

	_, err = client.SocialAccount.Get(ctx, account.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), account.ID)
	require.True(t, dbent.IsNotFound(err), "deleted account must be physically removed")
	logExists, err := client.SocialTaskLog.Query().
		Where(socialtasklog.IDEQ(log.ID), socialtasklog.SocialAccountIDEQ(account.ID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, logExists)
	subscriptionLedgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(subscriptionLedger.ID), usagelog.RequestIDEQ(subscriptionLedgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, subscriptionLedgerExists)
	walletLedgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(walletLedger.ID), usagelog.RequestIDEQ(walletLedgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, walletLedgerExists)
	unrelatedLedgerExists, err := client.UsageLog.Query().
		Where(usagelog.IDEQ(unrelatedLedger.ID), usagelog.RequestIDEQ(unrelatedLedgerRequestID)).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, unrelatedLedgerExists)
	storedProxy := client.SocialIP.Query().
		Where(socialip.IDEQ(proxy.ID)).
		OnlyX(ctx)
	require.Nil(t, storedProxy.BoundSocialAccountID)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, visible)
	require.Zero(t, page.Total)
}

func TestSocialAccountServiceBatchImportCreatesFreshAccountAfterHardDelete(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("restore-workbench-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@restore_me").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("restore_me").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	require.NoError(t, svc.DeleteForUser(ctx, user.ID, account.ID))
	_, err := client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), account.ID)
	require.True(t, dbent.IsNotFound(err))

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@restore_me",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.NoError(t, err)
	require.NotEqual(t, account.ID, imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, visible, 1)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, imported.ID, visible[0].ID)
}

func TestSocialAccountServiceBatchImportDoesNotRestoreHardDeletedPlatformlessAccount(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("restore-platformless-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("restore-platformless-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	password := "restore-delivery-secret"
	removed := client.SocialAccount.Create().
		SetName("@restore_cross").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("restore_cross").
		SetAssignedUserID(user.ID).
		SetPassword(password).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	require.NoError(t, svc.DeleteForUser(ctx, user.ID, removed.ID))
	other := client.SocialAccount.Create().
		SetName("restore_cross").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("restore_cross").
		SetAssignedUserID(otherUser.ID).
		SaveX(ctx)

	_, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Name:      "@restore_cross",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), removed.ID)
	require.True(t, dbent.IsNotFound(err))

	storedOther := client.SocialAccount.GetX(ctx, other.ID)
	require.Equal(t, otherUser.ID, int64(*storedOther.AssignedUserID))
	require.Equal(t, "instagram", storedOther.PlatformKey)
}

func TestSocialAccountServiceBatchImportForUserDedupesNormalizedPlatformName(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-dedupe-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	result, err := svc.BatchImportForUser(ctx, user.ID, []*UserImportSocialAccountInput{
		{Platform: "x_twitter", Name: "@Dup_User", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "X_Twitter", Name: "dup_user", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=duplicate; auth_token=duplicate")},
		{Platform: "instagram", Name: "dup_user", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=other; auth_token=other")},
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 2, result.Imported)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.Errors, 1)
	require.Len(t, result.Accounts, 2)

	count, err := client.SocialAccount.Query().
		Where(socialaccount.AssignedUserIDEQ(user.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestSocialAccountServiceBatchImportForUserReportsPreciseRowReasons(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-import-reasons@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	other := client.User.Create().
		SetEmail("batch-import-reasons-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	poolMatch := client.SocialAccount.Create().
		SetName("@reason_pool_match").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("reason_pool_match").
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@reason_existing_local").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("reason_existing_local").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@reason_existing_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("reason_existing_other").
		SetAssignedUserID(other.ID).
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@reason_ambiguous").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("reason_ambiguous").
		SaveX(ctx)
	client.SocialAccount.Create().
		SetName("@reason_ambiguous").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("reason_ambiguous").
		SaveX(ctx)

	result, err := svc.BatchImportForUser(ctx, user.ID, []*UserImportSocialAccountInput{
		{Platform: "x_twitter", Name: "@reason_pool_match", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "x_twitter", Name: "@reason_staging", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=staging; auth_token=staging")},
		{Platform: "x_twitter", Name: "@reason_batch_dup", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "twitter", Name: "reason_batch_dup", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=dup; auth_token=dup")},
		{Platform: "x_twitter", Name: "@reason_existing_local", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "x_twitter", Name: "@reason_existing_other", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "", Name: "@reason_ambiguous", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "x_twitter", Name: "@reason_invalid", TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
	})
	require.NoError(t, err)

	require.Equal(t, 8, result.Total)
	require.Equal(t, 3, result.Succeeded)
	require.Equal(t, 3, result.Imported)
	require.Equal(t, 5, result.Skipped)
	require.Equal(t, 2, result.Failed)
	require.Equal(t, 3, result.Duplicates)
	require.Len(t, result.Items, 8)
	require.Equal(t, poolMatch.ID, result.Items[0].ID)
	require.Equal(t, "succeeded", result.Items[0].Status)
	require.Equal(t, userImportReasonMatchedTotalPool, result.Items[0].Reason)
	require.Equal(t, "succeeded", result.Items[1].Status)
	require.Equal(t, userImportReasonStagedNotStored, result.Items[1].Reason)
	require.Equal(t, "succeeded", result.Items[2].Status)
	require.Equal(t, userImportReasonStagedNotStored, result.Items[2].Reason)
	require.Equal(t, "duplicate", result.Items[3].Status)
	require.Equal(t, userImportReasonDuplicateInBatch, result.Items[3].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonDuplicateInBatch), result.Items[3].Error)
	require.Equal(t, "duplicate", result.Items[4].Status)
	require.Equal(t, userImportReasonAlreadyInWorkbench, result.Items[4].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonAlreadyInWorkbench), result.Items[4].Error)
	require.Equal(t, "duplicate", result.Items[5].Status)
	require.Equal(t, userImportReasonAlreadyAssigned, result.Items[5].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonAlreadyAssigned), result.Items[5].Error)
	require.Equal(t, "failed", result.Items[6].Status)
	require.Equal(t, userImportReasonAmbiguousPoolMatch, result.Items[6].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonAmbiguousPoolMatch), result.Items[6].Error)
	require.Equal(t, "failed", result.Items[7].Status)
	require.Equal(t, userImportReasonInvalidInput, result.Items[7].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonInvalidInput), result.Items[7].Error)
}

func TestSocialAccountServiceBatchImportForUserRejectsIncompleteStructuredExecutionAuth(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-import-invalid-execution-auth@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	incompleteExecutionAuth := `{"access_token":"access"}`
	result, err := svc.BatchImportForUser(ctx, user.ID, []*UserImportSocialAccountInput{
		{
			Platform:      "x_twitter",
			Name:          "@invalid_execution_auth_import",
			Password:      socialStringPtr("typed-secret"),
			AuthCookie:    socialStringPtr("ct0=fresh; auth_token=fresh"),
			ExecutionAuth: &incompleteExecutionAuth,
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Zero(t, result.Succeeded)
	require.Zero(t, result.Imported)
	require.Equal(t, 1, result.Skipped)
	require.Equal(t, 1, result.Failed)
	require.Zero(t, result.Duplicates)
	require.Equal(t, []string{userAccountBatchImportReasonMessage(userImportReasonInvalidInput)}, result.Errors)
	require.Len(t, result.Items, 1)
	require.Equal(t, "failed", result.Items[0].Status)
	require.Equal(t, userImportReasonInvalidInput, result.Items[0].Reason)
	require.Equal(t, userAccountBatchImportReasonMessage(userImportReasonInvalidInput), result.Items[0].Error)

	exists, err := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("invalid_execution_auth_import")).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestSocialAccountServiceUserImportIgnoresDefaultProxySnapshot(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("user-import-proxy-snapshot@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	_, hasSnapshotField := reflect.TypeOf(UserImportSocialAccountInput{}).FieldByName("DefaultProxySnapshot")
	require.False(t, hasSnapshotField)

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@user_import_proxy_snapshot",
		Password:  socialStringPtr("account-secret"),
		TwoFactor: socialStringPtr("totp-secret"),
	})

	require.NoError(t, err)
	require.Nil(t, imported.DefaultProxySnapshot)
	stored := client.SocialAccount.GetX(ctx, imported.ID)
	require.Nil(t, stored.DefaultProxySnapshot)
}

func TestSocialAccountServiceUserImportPreservesDeliveryFieldWhitespace(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("user-import-delivery-whitespace@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	password := "  account-secret  "
	email := "  mail@example.com  "
	emailPassword := "  mail-secret  "
	twoFactor := "  totp-secret  "
	backupCode := "  backup-code  "
	emailClientID := "  mail-client  "
	emailToken := "  mail-token  "
	authCookie := "  ct0=import; auth_token=import  "
	executionAuth := "  encrypted-user-import-execution-auth-ciphertext  "
	storedExecutionAuth := "encrypted-user-import-execution-auth-ciphertext"
	remark := "  operator note  "
	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:       "x_twitter",
		Name:           "@user_import_delivery_whitespace",
		Password:       &password,
		Email:          &email,
		EmailPassword:  &emailPassword,
		TwoFactor:      &twoFactor,
		BackupCode:     &backupCode,
		EmailClientID:  &emailClientID,
		EmailToken:     &emailToken,
		AuthCookie:     &authCookie,
		ExecutionAuth:  &executionAuth,
		RegistrationIP: socialStringPtr("  203.0.113.20  "),
		Remark:         &remark,
	})

	require.NoError(t, err)
	require.Equal(t, password, requireSocialStringPtr(t, imported.Password))
	require.Equal(t, "mail@example.com", requireSocialStringPtr(t, imported.Email))
	require.Equal(t, emailPassword, requireSocialStringPtr(t, imported.EmailPassword))
	require.Equal(t, twoFactor, requireSocialStringPtr(t, imported.TwoFactor))
	require.Equal(t, backupCode, requireSocialStringPtr(t, imported.BackupCode))
	require.Equal(t, emailClientID, requireSocialStringPtr(t, imported.EmailClientID))
	require.Equal(t, emailToken, requireSocialStringPtr(t, imported.EmailToken))
	require.Equal(t, authCookie, requireSocialStringPtr(t, imported.AuthCookie))
	require.Equal(t, storedExecutionAuth, requireSocialStringPtr(t, imported.ExecutionAuth))
	require.Equal(t, "203.0.113.20", requireSocialStringPtr(t, imported.RegistrationIP))
	require.Equal(t, remark, requireSocialStringPtr(t, imported.Remark))

	stored := client.SocialAccount.GetX(ctx, imported.ID)
	require.Equal(t, password, requireSocialStringPtr(t, stored.Password))
	require.Equal(t, "mail@example.com", requireSocialStringPtr(t, stored.Email))
	require.Equal(t, emailPassword, requireSocialStringPtr(t, stored.EmailPassword))
	require.Equal(t, twoFactor, requireSocialStringPtr(t, stored.TwoFactor))
	require.Equal(t, backupCode, requireSocialStringPtr(t, stored.BackupCode))
	require.Equal(t, emailClientID, requireSocialStringPtr(t, stored.EmailClientID))
	require.Equal(t, emailToken, requireSocialStringPtr(t, stored.EmailToken))
	require.Equal(t, authCookie, requireSocialStringPtr(t, stored.AuthCookie))
	require.Equal(t, storedExecutionAuth, requireSocialStringPtr(t, stored.ExecutionAuth))
	require.Equal(t, "203.0.113.20", requireSocialStringPtr(t, stored.RegistrationIP))
	require.Equal(t, remark, requireSocialStringPtr(t, stored.Remark))
}

func TestSocialAccountServiceBatchImportClearsPoolDefaultProxySnapshotOnAssign(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("pool-proxy-import@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	poolSnapshot := `{"id":999,"name":"pool-proxy","endpoint":"http://pool-proxy.example:8080","status":"online"}`
	poolAccount := client.SocialAccount.Create().
		SetName("@pool_proxy_ops").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_proxy_ops").
		SetDefaultProxySnapshot(poolSnapshot).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@pool_proxy_ops",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.NoError(t, err)
	require.Equal(t, poolAccount.ID, imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)
	require.Nil(t, imported.DefaultProxySnapshot)

	stored := client.SocialAccount.GetX(ctx, poolAccount.ID)
	require.Equal(t, user.ID, *stored.AssignedUserID)
	require.Nil(t, stored.DefaultProxySnapshot)
}

func TestSocialAccountServiceBatchImportAndDeleteStayInUserScope(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-workbench-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("batch-workbench-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ownRemoved := client.SocialAccount.Create().
		SetName("@batch_removed").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_removed").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	require.NoError(t, svc.DeleteForUser(ctx, user.ID, ownRemoved.ID))
	fresh := client.SocialAccount.Create().
		SetName("@batch_fresh").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_fresh").
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("@batch_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_other").
		SetAssignedUserID(otherUser.ID).
		SaveX(ctx)

	importResult, err := svc.BatchImportForUser(ctx, user.ID, []*UserImportSocialAccountInput{
		{Platform: "x_twitter", Name: "@batch_removed", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "x_twitter", Name: "@batch_fresh", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=fresh; auth_token=fresh"), ExecutionAuth: socialStringPtr("encrypted-batch-fresh-execution-auth-ciphertext")},
		{Platform: "x_twitter", Name: "@batch_missing", Password: socialStringPtr("missing-secret"), Email: socialStringPtr("mail@example.com"), EmailPassword: socialStringPtr("mail-secret")},
	})
	require.NoError(t, err)
	require.Equal(t, 3, importResult.Total)
	require.Equal(t, 3, importResult.Imported)
	require.Equal(t, 0, importResult.Skipped)
	require.Len(t, importResult.Accounts, 3)
	require.Empty(t, importResult.Errors)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(3), page.Total)
	require.ElementsMatch(t, []int64{importResult.Accounts[0].ID, fresh.ID, importResult.Accounts[2].ID}, []int64{visible[0].ID, visible[1].ID, visible[2].ID})

	deleteResult, err := svc.BatchDeleteForUser(ctx, user.ID, []int64{importResult.Accounts[0].ID, fresh.ID, importResult.Accounts[2].ID, otherAccount.ID, 0})
	require.NoError(t, err)
	require.Equal(t, 5, deleteResult.Total)
	require.Equal(t, 3, deleteResult.Removed)
	require.Equal(t, 0, deleteResult.Skipped)
	require.Equal(t, 2, deleteResult.Failed)
	require.Equal(t, []string{"account could not be deleted", "account could not be deleted"}, deleteResult.Errors)
	require.Len(t, deleteResult.Items, 5)
	require.Equal(t, "failed", deleteResult.Items[3].Status)
	require.Equal(t, "delete_failed", deleteResult.Items[3].Reason)
	require.Equal(t, "failed", deleteResult.Items[4].Status)
	require.Equal(t, "invalid_id", deleteResult.Items[4].Reason)
	require.NotContains(t, strings.Join(deleteResult.Errors, " "), "error: code=")
	require.NotContains(t, strings.Join(deleteResult.Errors, " "), "SOCIAL_ACCOUNT_NOT_ASSIGNED")
	require.NotContains(t, strings.Join(deleteResult.Errors, " "), strconv.FormatInt(otherAccount.ID, 10))

	visible, page, err = svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, visible)
	require.Zero(t, page.Total)
	for _, removedID := range []int64{importResult.Accounts[0].ID, fresh.ID, importResult.Accounts[2].ID} {
		_, err := client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), removedID)
		require.True(t, dbent.IsNotFound(err))
	}

	storedOther := client.SocialAccount.GetX(ctx, otherAccount.ID)
	require.NotNil(t, storedOther.AssignedUserID)
	require.Equal(t, otherUser.ID, int64(*storedOther.AssignedUserID))
	require.Equal(t, otherAccount.ID, storedOther.ID)
}

func TestSocialAccountServiceBatchDeleteForUserReportsDuplicateIDsAsSkipped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-delete-duplicate-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@batch_delete_duplicate").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_delete_duplicate").
		SetAssignedUserID(user.ID).
		SaveX(ctx)

	result, err := svc.BatchDeleteForUser(ctx, user.ID, []int64{account.ID, account.ID})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Removed)
	require.Equal(t, 1, result.Skipped)
	require.Zero(t, result.Failed)
	require.Empty(t, result.Errors)
	require.Len(t, result.Items, 2)
	require.Equal(t, account.ID, result.Items[0].ID)
	require.Equal(t, "succeeded", result.Items[0].Status)
	require.Equal(t, account.ID, result.Items[1].ID)
	require.Equal(t, "skipped", result.Items[1].Status)
	require.Equal(t, "duplicate_in_batch", result.Items[1].Reason)

	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), account.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestSocialAccountServiceBatchDeleteForUserReportsRepeatedInvalidIDsAsFailed(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("batch-delete-invalid-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)

	result, err := svc.BatchDeleteForUser(ctx, user.ID, []int64{0, 0, -1})

	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Zero(t, result.Succeeded)
	require.Zero(t, result.Removed)
	require.Zero(t, result.Skipped)
	require.Equal(t, 3, result.Failed)
	require.Len(t, result.Errors, 3)
	require.Len(t, result.Items, 3)
	for _, item := range result.Items {
		require.Equal(t, "failed", item.Status)
		require.Equal(t, "invalid_id", item.Reason)
		require.Equal(t, "account could not be deleted", item.Error)
	}
}

func TestSocialAccountServiceUpdateForUserMutatesOnlyEditableFields(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("mutable-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("mutable-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	proxySnapshot := `{"id":1,"endpoint":"http://proxy.local:8080"}`
	account := client.SocialAccount.Create().
		SetName("@locked_identity").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("locked_identity").
		SetIdentityKind("username").
		SetIdentityKey("x_twitter:username:locked_identity").
		SetPlatformUserID("real-rest-id").
		SetPassword("old-password").
		SetTwoFactor("old-2fa").
		SetRegistrationIP("198.51.100.20").
		SetAuthCookie("ct0=old; auth_token=old").
		SetDefaultProxySnapshot(proxySnapshot).
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	original := client.SocialAccount.GetX(ctx, account.ID)

	newName := "@malicious_identity"
	newRestID := "fake-rest-id"
	newPassword := "  new-password  "
	newEmail := "  owner@example.com  "
	newEmailPassword := "  mailbox-secret  "
	newTwoFactor := "  totp-secret  "
	newBackupCode := "  backup-code  "
	newEmailClientID := "  mail-client  "
	newEmailToken := "  mail-token  "
	emptyTwoFactor := " "
	newRegistrationIP := "203.0.113.10"
	newAuthCookie := "  ct0=new; auth_token=new  "
	newExecutionAuth := "  encrypted-updated-execution-auth-ciphertext  "
	storedNewExecutionAuth := "encrypted-updated-execution-auth-ciphertext"
	newStatus := SocialAccountStatusInvalid
	newTaskStatus := SocialTaskStatusManualReview
	newProxySnapshot := `{"id":999,"endpoint":"http://attacker.proxy:8080"}`
	newRemark := "  operator note  "
	updated, err := svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{
		Name:                 &newName,
		PlatformUserID:       &newRestID,
		Password:             &newPassword,
		Email:                &newEmail,
		EmailPassword:        &newEmailPassword,
		TwoFactor:            &newTwoFactor,
		BackupCode:           &newBackupCode,
		EmailClientID:        &newEmailClientID,
		EmailToken:           &newEmailToken,
		RegistrationIP:       &newRegistrationIP,
		AuthCookie:           &newAuthCookie,
		ExecutionAuth:        &newExecutionAuth,
		AccountStatus:        &newStatus,
		TaskStatus:           &newTaskStatus,
		DefaultProxySnapshot: &newProxySnapshot,
		Remark:               &newRemark,
	})
	require.NoError(t, err)
	require.Equal(t, account.ID, updated.ID)
	require.Equal(t, "@locked_identity", updated.Name)
	require.Equal(t, "x_twitter", updated.Platform)
	require.Equal(t, "locked_identity", updated.Username)
	require.NotNil(t, updated.PlatformUserID)
	require.Equal(t, "real-rest-id", *updated.PlatformUserID)
	require.NotNil(t, updated.Password)
	require.Equal(t, newPassword, *updated.Password)
	require.NotNil(t, updated.Email)
	require.Equal(t, "owner@example.com", *updated.Email)
	require.NotNil(t, updated.EmailPassword)
	require.Equal(t, newEmailPassword, *updated.EmailPassword)
	require.NotNil(t, updated.TwoFactor)
	require.Equal(t, newTwoFactor, *updated.TwoFactor)
	require.NotNil(t, updated.BackupCode)
	require.Equal(t, newBackupCode, *updated.BackupCode)
	require.NotNil(t, updated.EmailClientID)
	require.Equal(t, newEmailClientID, *updated.EmailClientID)
	require.NotNil(t, updated.EmailToken)
	require.Equal(t, newEmailToken, *updated.EmailToken)
	require.NotNil(t, updated.RegistrationIP)
	require.Equal(t, newRegistrationIP, *updated.RegistrationIP)
	require.NotNil(t, updated.AuthCookie)
	require.Equal(t, newAuthCookie, *updated.AuthCookie)
	require.NotNil(t, updated.ExecutionAuth)
	require.Equal(t, storedNewExecutionAuth, *updated.ExecutionAuth)
	require.Equal(t, SocialAccountStatusAvailable, updated.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, updated.TaskStatus)
	require.NotNil(t, updated.DefaultProxySnapshot)
	require.Equal(t, proxySnapshot, *updated.DefaultProxySnapshot)
	require.NotNil(t, updated.Remark)
	require.Equal(t, newRemark, *updated.Remark)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Equal(t, "@locked_identity", stored.Name)
	require.Equal(t, "locked_identity", stored.NameKey)
	require.Equal(t, original.IdentityKind, stored.IdentityKind)
	require.Equal(t, original.IdentityKey, stored.IdentityKey)
	require.NotNil(t, stored.PlatformUserID)
	require.Equal(t, "real-rest-id", *stored.PlatformUserID)
	require.NotNil(t, stored.RegistrationIP)
	require.Equal(t, newRegistrationIP, *stored.RegistrationIP)
	require.NotNil(t, stored.Password)
	require.Equal(t, newPassword, *stored.Password)
	require.NotNil(t, stored.EmailPassword)
	require.Equal(t, newEmailPassword, *stored.EmailPassword)
	require.NotNil(t, stored.TwoFactor)
	require.Equal(t, newTwoFactor, *stored.TwoFactor)
	require.NotNil(t, stored.BackupCode)
	require.Equal(t, newBackupCode, *stored.BackupCode)
	require.NotNil(t, stored.EmailClientID)
	require.Equal(t, newEmailClientID, *stored.EmailClientID)
	require.NotNil(t, stored.EmailToken)
	require.Equal(t, newEmailToken, *stored.EmailToken)
	require.NotNil(t, stored.AuthCookie)
	require.Equal(t, newAuthCookie, *stored.AuthCookie)
	require.NotNil(t, stored.ExecutionAuth)
	require.Equal(t, storedNewExecutionAuth, *stored.ExecutionAuth)
	require.NotNil(t, stored.Remark)
	require.Equal(t, newRemark, *stored.Remark)
	require.Equal(t, SocialAccountStatusAvailable, stored.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, stored.TaskStatus)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Equal(t, proxySnapshot, *stored.DefaultProxySnapshot)

	_, err = svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{TwoFactor: &emptyTwoFactor})
	require.NoError(t, err)
	require.Nil(t, client.SocialAccount.GetX(ctx, account.ID).TwoFactor)

	invalidExecutionAuth := `{"access_token":"access"}`
	_, err = svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{
		Password:      socialStringPtr("partially-written-password"),
		ExecutionAuth: &invalidExecutionAuth,
	})
	require.ErrorIs(t, err, ErrSocialAccountExecutionAuthInvalid)
	storedAfterInvalidExecutionAuth := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAfterInvalidExecutionAuth.Password)
	require.Equal(t, newPassword, *storedAfterInvalidExecutionAuth.Password)
	require.NotNil(t, storedAfterInvalidExecutionAuth.ExecutionAuth)
	require.Equal(t, storedNewExecutionAuth, *storedAfterInvalidExecutionAuth.ExecutionAuth)

	_, err = svc.UpdateForUser(ctx, account.ID, otherUser.ID, &UpdateSocialAccountInput{Remark: socialStringPtr("cross-user")})
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)

	require.NoError(t, svc.DeleteForUser(ctx, user.ID, account.ID))
	_, err = svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{Remark: socialStringPtr("hidden")})
	require.Error(t, err)
	require.Equal(t, "SOCIAL_ACCOUNT_NOT_FOUND", infraerrors.Reason(err))
}

func TestSocialAccountServiceUpdateForUserRequiresCurrentAssignmentAtWrite(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("mutable-race-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("mutable-race-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@mutable_race").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("mutable_race").
		SetPassword("account-secret").
		SetTwoFactor("totp-secret").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	assignmentChanged := false
	client.SocialAccount.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdate) && !assignmentChanged {
				assignmentChanged = true
				if err := client.SocialAccount.UpdateOneID(account.ID).SetAssignedUserID(otherUser.ID).Exec(ctx); err != nil {
					return nil, err
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	_, err := svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{Remark: socialStringPtr("should-not-save")})
	require.ErrorIs(t, err, ErrSocialAccountAssignmentChanged)
	require.True(t, assignmentChanged)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.AssignedUserID)
	require.Equal(t, otherUser.ID, *stored.AssignedUserID)
	require.Nil(t, stored.Remark)
}

func TestAccountWorkbenchServiceRejectsDeletedUserAccountForTask(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("hidden-task-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, nil, nil)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)

	account := client.SocialAccount.Create().
		SetName("@hidden_task").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("hidden_task").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	require.NoError(t, accountSvc.DeleteForUser(ctx, user.ID, account.ID))

	_, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionFollow,
		Target:     socialStringPtr("target_user"),
	})
	require.Error(t, err)
	require.Equal(t, "SOCIAL_ACCOUNT_NOT_FOUND", infraerrors.Reason(err))
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

	_, err = svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Name:      "@same_name",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.ErrorIs(t, err, ErrSocialAccountImportAmbiguous)
}

func TestNormalizeSocialTaskActionSupportsRetweet(t *testing.T) {
	action, ok := NormalizeSocialTaskAction(" retweet ")
	require.True(t, ok)
	require.Equal(t, SocialTaskActionRetweet, action)
	require.False(t, IsBillableSocialTaskAction(SocialTaskActionRetweet))
	require.False(t, IsBillableSocialTaskAction(SocialTaskActionLoginCheck))
	require.True(t, IsBillableSocialTaskAction(SocialTaskActionLogin))
	require.InEpsilon(t, SocialTaskUnitPrice, SocialTaskPriceForAction(SocialTaskActionLogin), 0.000001)
	require.Zero(t, SocialTaskPriceForAction(SocialTaskActionFollow))
	_, ok = NormalizeSocialTaskAction("tweet")
	require.False(t, ok)
	require.False(t, IsBillableSocialTaskAction("tweet"))
	require.Zero(t, SocialTaskPriceForAction("tweet"))
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

	_, err = svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Name:      "@shared_name",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.ErrorIs(t, err, ErrSocialAccountImportAmbiguous)
}

func TestSocialAccountServiceAdminImportDedupesPoolByNormalizedPlatformName(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	_, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:      "@NorthWind_Ops",
		Platform:  "x_twitter",
		Password:  socialStringPtr("account-secret"),
		TwoFactor: socialStringPtr("totp-secret"),
	})
	require.NoError(t, err)

	importedProxySnapshot := `{"id":123,"name":"default-proxy","endpoint":"http://proxy.local:8080"}`
	result, err := svc.ImportPoolAccounts(ctx, []*CreateSocialAccountInput{
		{Name: "northwind_ops", Platform: "X_Twitter", Password: socialStringPtr("account-secret"), TwoFactor: socialStringPtr("totp-secret")},
		{Name: "@fresh_ops", Platform: "x_twitter", Password: socialStringPtr("account-secret"), TwoFactor: socialStringPtr("totp-secret"), DefaultProxySnapshot: &importedProxySnapshot},
		{Name: "FRESH_OPS", Platform: "x_twitter", Password: socialStringPtr("account-secret"), TwoFactor: socialStringPtr("totp-secret")},
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 1, result.Created)
	require.Equal(t, 2, result.Skipped)
	require.Equal(t, []string{"duplicate account in import batch"}, result.Errors)

	count, err := client.SocialAccount.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	imported := client.SocialAccount.Query().
		Where(socialaccount.NameKeyEQ("fresh_ops")).
		OnlyX(ctx)
	require.NotNil(t, imported.DefaultProxySnapshot)
	require.Equal(t, importedProxySnapshot, *imported.DefaultProxySnapshot)
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
	proxySnapshot := "proxy-one"
	account := client.SocialAccount.Create().
		SetName("assign_me").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("assign_me").
		SetDefaultProxySnapshot(proxySnapshot).
		SaveX(ctx)

	assigned, err := svc.Assign(ctx, account.ID, user1.ID)
	require.NoError(t, err)
	require.Equal(t, user1.ID, *assigned.AssignedUserID)
	require.Nil(t, assigned.DefaultProxySnapshot)

	_, err = svc.Assign(ctx, account.ID, user2.ID)
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	reclaimed, err := svc.Reclaim(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, reclaimed.AssignedUserID)
	require.Nil(t, reclaimed.DefaultProxySnapshot)
}

func TestSocialAccountAssignRejectsMissingTargetUser(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	account := client.SocialAccount.Create().
		SetName("assign_missing_user").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("assign_missing_user").
		SaveX(ctx)

	_, err := svc.Assign(ctx, account.ID, 404404)
	require.ErrorIs(t, err, ErrUserNotFound)
	require.Nil(t, client.SocialAccount.GetX(ctx, account.ID).AssignedUserID)
}

func TestSocialAccountBatchPoolOperationsReportDuplicateIDsAsSkipped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("pool-batch-duplicate-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	assignAccount := client.SocialAccount.Create().
		SetName("pool_batch_duplicate_assign").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_batch_duplicate_assign").
		SaveX(ctx)
	reclaimAccount := client.SocialAccount.Create().
		SetName("pool_batch_duplicate_reclaim").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_batch_duplicate_reclaim").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	deleteAccount := client.SocialAccount.Create().
		SetName("pool_batch_duplicate_delete").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("pool_batch_duplicate_delete").
		SaveX(ctx)

	assignResult, err := svc.BatchAssign(ctx, []int64{assignAccount.ID, assignAccount.ID}, user.ID)
	require.NoError(t, err)
	require.Equal(t, 2, assignResult.Total)
	require.Equal(t, 1, assignResult.Succeeded)
	require.Equal(t, 1, assignResult.Skipped)
	require.Zero(t, assignResult.Failed)
	require.Len(t, assignResult.Items, 2)
	require.Equal(t, "succeeded", assignResult.Items[0].Status)
	require.Equal(t, "skipped", assignResult.Items[1].Status)
	require.Equal(t, "duplicate_in_batch", assignResult.Items[1].Reason)
	require.Equal(t, user.ID, *client.SocialAccount.GetX(ctx, assignAccount.ID).AssignedUserID)

	reclaimResult, err := svc.BatchReclaim(ctx, []int64{reclaimAccount.ID, reclaimAccount.ID})
	require.NoError(t, err)
	require.Equal(t, 2, reclaimResult.Total)
	require.Equal(t, 1, reclaimResult.Succeeded)
	require.Equal(t, 1, reclaimResult.Skipped)
	require.Zero(t, reclaimResult.Failed)
	require.Len(t, reclaimResult.Items, 2)
	require.Equal(t, "succeeded", reclaimResult.Items[0].Status)
	require.Equal(t, "skipped", reclaimResult.Items[1].Status)
	require.Equal(t, "duplicate_in_batch", reclaimResult.Items[1].Reason)
	require.Nil(t, client.SocialAccount.GetX(ctx, reclaimAccount.ID).AssignedUserID)

	deleteResult, err := svc.BatchDelete(ctx, []int64{deleteAccount.ID, deleteAccount.ID})
	require.NoError(t, err)
	require.Equal(t, 2, deleteResult.Total)
	require.Equal(t, 1, deleteResult.Succeeded)
	require.Equal(t, 1, deleteResult.Skipped)
	require.Zero(t, deleteResult.Failed)
	require.Len(t, deleteResult.Items, 2)
	require.Equal(t, "succeeded", deleteResult.Items[0].Status)
	require.Equal(t, "skipped", deleteResult.Items[1].Status)
	require.Equal(t, "duplicate_in_batch", deleteResult.Items[1].Reason)
	_, err = client.SocialAccount.Get(mixins.SkipSoftDelete(ctx), deleteAccount.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestSocialAccountBatchPoolOperationsReturnSafeFailureMessages(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("pool-batch-safe-errors@example.com").SetPasswordHash("hash").SaveX(ctx)
	missingID := int64(404404)

	assignResult, err := svc.BatchAssign(ctx, []int64{missingID}, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, assignResult.Total)
	require.Zero(t, assignResult.Succeeded)
	require.Zero(t, assignResult.Skipped)
	require.Equal(t, 1, assignResult.Failed)
	require.Equal(t, []string{"account could not be assigned"}, assignResult.Errors)
	require.Len(t, assignResult.Items, 1)
	require.Equal(t, missingID, assignResult.Items[0].ID)
	require.Equal(t, "failed", assignResult.Items[0].Status)
	require.Equal(t, "assign_failed", assignResult.Items[0].Reason)
	require.Equal(t, "account could not be assigned", assignResult.Items[0].Error)

	reclaimResult, err := svc.BatchReclaim(ctx, []int64{missingID})
	require.NoError(t, err)
	require.Equal(t, 1, reclaimResult.Total)
	require.Zero(t, reclaimResult.Succeeded)
	require.Zero(t, reclaimResult.Skipped)
	require.Equal(t, 1, reclaimResult.Failed)
	require.Equal(t, []string{"account could not be reclaimed"}, reclaimResult.Errors)
	require.Len(t, reclaimResult.Items, 1)
	require.Equal(t, missingID, reclaimResult.Items[0].ID)
	require.Equal(t, "failed", reclaimResult.Items[0].Status)
	require.Equal(t, "reclaim_failed", reclaimResult.Items[0].Reason)
	require.Equal(t, "account could not be reclaimed", reclaimResult.Items[0].Error)

	deleteResult, err := svc.BatchDelete(ctx, []int64{missingID})
	require.NoError(t, err)
	require.Equal(t, 1, deleteResult.Total)
	require.Zero(t, deleteResult.Succeeded)
	require.Zero(t, deleteResult.Skipped)
	require.Equal(t, 1, deleteResult.Failed)
	require.Equal(t, []string{"account could not be deleted"}, deleteResult.Errors)
	require.Len(t, deleteResult.Items, 1)
	require.Equal(t, missingID, deleteResult.Items[0].ID)
	require.Equal(t, "failed", deleteResult.Items[0].Status)
	require.Equal(t, "delete_failed", deleteResult.Items[0].Reason)
	require.Equal(t, "account could not be deleted", deleteResult.Items[0].Error)

	allMessages := strings.Join(append(append(assignResult.Errors, reclaimResult.Errors...), deleteResult.Errors...), " ")
	require.NotContains(t, allMessages, "error: code=")
	require.NotContains(t, allMessages, "SOCIAL_ACCOUNT_NOT_FOUND")
	require.NotContains(t, allMessages, strconv.FormatInt(missingID, 10))
}

func TestSocialAccountTotalPoolOperationsRejectWorkbenchStagingAccounts(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	owner := client.User.Create().SetEmail("staging-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	target := client.User.Create().SetEmail("staging-target@example.com").SetPasswordHash("hash").SaveX(ctx)
	staging := client.SocialAccount.Create().
		SetName("total_pool_staging_guard").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("total_pool_staging_guard").
		SetIdentityKind("username").
		SetIdentityKey("total_pool_staging_guard").
		SetAssignedUserID(owner.ID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)

	_, err := svc.AssignTotalPool(ctx, staging.ID, target.ID)
	require.Error(t, err)
	require.Equal(t, owner.ID, *client.SocialAccount.GetX(ctx, staging.ID).AssignedUserID)

	_, err = svc.ReclaimTotalPool(ctx, staging.ID)
	require.Error(t, err)
	require.Equal(t, owner.ID, *client.SocialAccount.GetX(ctx, staging.ID).AssignedUserID)

	err = svc.DeleteTotalPool(ctx, staging.ID)
	require.Error(t, err)
	_, err = client.SocialAccount.Get(ctx, staging.ID)
	require.NoError(t, err)

	assignResult, err := svc.BatchAssignTotalPool(ctx, []int64{staging.ID}, target.ID)
	require.NoError(t, err)
	require.Equal(t, 1, assignResult.Failed)
	require.Equal(t, "assign_failed", assignResult.Items[0].Reason)

	reclaimResult, err := svc.BatchReclaimTotalPool(ctx, []int64{staging.ID})
	require.NoError(t, err)
	require.Equal(t, 1, reclaimResult.Failed)
	require.Equal(t, "reclaim_failed", reclaimResult.Items[0].Reason)

	deleteResult, err := svc.BatchDeleteTotalPool(ctx, []int64{staging.ID})
	require.NoError(t, err)
	require.Equal(t, 1, deleteResult.Failed)
	require.Equal(t, "delete_failed", deleteResult.Items[0].Reason)

	preserved := client.SocialAccount.GetX(ctx, staging.ID)
	require.Equal(t, owner.ID, *preserved.AssignedUserID)
	require.Equal(t, SocialAccountStatusNotStored, preserved.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, preserved.TaskStatus)
}

func TestSocialAccountBatchReclaimSkipsAlreadyUnassignedAccounts(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("batch-reclaim-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	assigned := client.SocialAccount.Create().
		SetName("batch_reclaim_assigned").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_reclaim_assigned").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(`{"id":1}`).
		SaveX(ctx)
	unassigned := client.SocialAccount.Create().
		SetName("batch_reclaim_unassigned").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_reclaim_unassigned").
		SaveX(ctx)

	result, err := svc.BatchReclaim(ctx, []int64{assigned.ID, unassigned.ID, 0})
	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 2, result.Skipped)
	require.Equal(t, 0, result.Failed)
	require.Empty(t, result.Errors)

	reasons := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		reasons = append(reasons, item.Reason)
	}
	require.Contains(t, reasons, "already_unassigned")
	require.Contains(t, reasons, "invalid_id")
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).AssignedUserID)
	require.Nil(t, client.SocialAccount.GetX(ctx, assigned.ID).DefaultProxySnapshot)
	require.Nil(t, client.SocialAccount.GetX(ctx, unassigned.ID).AssignedUserID)
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
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "proxy", IPType: "residential", Endpoint: &endpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	otherEndpoint := "http://8.8.4.4:8080"
	otherIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user2.ID, Name: "other-proxy", IPType: "residential", Endpoint: &otherEndpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(otherIP.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	otherIP, err = ipSvc.GetByID(ctx, otherIP.ID)
	require.NoError(t, err)
	untestedIP, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "untested-proxy", IPType: "residential"})
	require.NoError(t, err)
	onlineWithoutEndpoint, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "missing-endpoint-proxy", IPType: "residential"})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(onlineWithoutEndpoint.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	onlineWithoutEndpoint, err = ipSvc.GetByID(ctx, onlineWithoutEndpoint.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(ip)

	updated, err := accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, ip)
	require.NoError(t, err)
	require.NotNil(t, updated.DefaultProxySnapshot)
	require.Equal(t, snapshot, *updated.DefaultProxySnapshot)
	proxyID, ok := SocialIPIDFromSnapshot(*updated.DefaultProxySnapshot)
	require.True(t, ok)
	require.Equal(t, ip.ID, proxyID)

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, otherIP)
	require.ErrorIs(t, err, ErrSocialIPOwnerMismatch)

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, untestedIP)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, onlineWithoutEndpoint)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))

	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user2.ID, ip)
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)

	cleared, err := accountSvc.SetDefaultProxyForUser(ctx, account.ID, user1.ID, nil)
	require.NoError(t, err)
	require.Nil(t, cleared.DefaultProxySnapshot)
}

func TestSocialAccountDefaultProxyWriteRequiresCurrentUserAssignment(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user1 := client.User.Create().SetEmail("proxy-race-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	user2 := client.User.Create().SetEmail("proxy-race-current-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("default_proxy_assignment_changed").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("default_proxy_assignment_changed").
		SetAssignedUserID(user2.ID).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user1.ID, Name: "race proxy", IPType: "residential", Endpoint: &endpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(ip)

	_, err = accountSvc.setDefaultProxySnapshotForUser(ctx, account.ID, user1.ID, &snapshot)
	require.ErrorIs(t, err, ErrSocialAccountAssignmentChanged)

	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.DefaultProxySnapshot)
	require.NotNil(t, stored.AssignedUserID)
	require.Equal(t, user2.ID, *stored.AssignedUserID)
}

func TestSocialAccountBatchDefaultProxySeparatesFailedFromSkipped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().
		SetEmail("batch-proxy-summary-owner@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("batch-proxy-summary-other@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	ownedAccount := client.SocialAccount.Create().
		SetName("batch_proxy_summary_owned").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_proxy_summary_owned").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("batch_proxy_summary_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_proxy_summary_other").
		SetAssignedUserID(otherUser.ID).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "batch summary proxy", IPType: "residential", Endpoint: &endpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)

	result, err := accountSvc.BatchSetDefaultProxyForUser(
		ctx,
		user.ID,
		[]int64{ownedAccount.ID, otherAccount.ID, 0},
		DefaultProxyAssignmentSpecific,
		ip,
		nil,
	)
	require.NoError(t, err)

	require.Equal(t, 3, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 2, result.Failed)
	require.Zero(t, result.Skipped)
	require.Len(t, result.Items, 3)
	require.Equal(t, "succeeded", result.Items[0].Status)
	require.Equal(t, "failed", result.Items[1].Status)
	require.Equal(t, "account_not_assigned", result.Items[1].Reason)
	require.Empty(t, result.Items[1].Name)
	require.Equal(t, "failed", result.Items[2].Status)
	require.Equal(t, "invalid_id", result.Items[2].Reason)
}

func TestSocialAccountBatchDefaultProxyReportsUnavailableSpecificProxyPerAccount(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().
		SetEmail("batch-proxy-offline-owner@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("batch_proxy_offline_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_proxy_offline_account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "offline batch proxy", IPType: "residential", Endpoint: &endpoint})
	require.NoError(t, err)

	result, err := accountSvc.BatchSetDefaultProxyForUser(
		ctx,
		user.ID,
		[]int64{account.ID},
		DefaultProxyAssignmentSpecific,
		ip,
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Zero(t, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	require.Zero(t, result.Skipped)
	require.Len(t, result.Items, 1)
	require.Equal(t, account.ID, result.Items[0].ID)
	require.Equal(t, account.Name, result.Items[0].Name)
	require.Equal(t, "failed", result.Items[0].Status)
	require.Equal(t, "proxy_not_available", result.Items[0].Reason)
	require.Equal(t, "account proxy could not be assigned", result.Items[0].Error)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, stored.DefaultProxySnapshot)
}

func TestSocialAccountBatchDefaultProxyReportsDuplicateIDsAsSkipped(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().
		SetEmail("batch-proxy-duplicate-owner@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("batch_proxy_duplicate_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_proxy_duplicate_account").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "duplicate batch proxy", IPType: "residential", Endpoint: &endpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)

	result, err := accountSvc.BatchSetDefaultProxyForUser(
		ctx,
		user.ID,
		[]int64{account.ID, account.ID, 0},
		DefaultProxyAssignmentSpecific,
		ip,
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Skipped)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Items, 3)
	require.Equal(t, "succeeded", result.Items[0].Status)
	require.Equal(t, account.ID, result.Items[0].ID)
	require.Equal(t, "skipped", result.Items[1].Status)
	require.Equal(t, account.ID, result.Items[1].ID)
	require.Equal(t, "duplicate_in_batch", result.Items[1].Reason)
	require.Equal(t, "failed", result.Items[2].Status)
	require.Equal(t, int64(0), result.Items[2].ID)
	require.Equal(t, "invalid_id", result.Items[2].Reason)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.DefaultProxySnapshot)
	proxyID, ok := SocialIPIDFromSnapshot(*stored.DefaultProxySnapshot)
	require.True(t, ok)
	require.Equal(t, ip.ID, proxyID)
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
	endpoint := "http://8.8.8.8:8080"
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "shared proxy", IPType: "residential", Endpoint: &endpoint})
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

	require.NotNil(t, firstUpdated.DefaultProxySnapshot)
	require.NotNil(t, secondUpdated.DefaultProxySnapshot)
	require.Equal(t, *firstUpdated.DefaultProxySnapshot, *secondUpdated.DefaultProxySnapshot)
	proxyID, ok := SocialIPIDFromSnapshot(*secondUpdated.DefaultProxySnapshot)
	require.True(t, ok)
	require.Equal(t, ip.ID, proxyID)
}

func TestSocialAccountBatchRandomDefaultProxyRepeatsEvenlyWhenPoolIsSmallerThanAccounts(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	ipSvc := NewSocialIPService(client)

	user := client.User.Create().
		SetEmail("random-proxy-repeat-owner@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	accountIDs := make([]int64, 0, 5)
	for i := 0; i < 5; i++ {
		account := client.SocialAccount.Create().
			SetName(fmt.Sprintf("random_proxy_account_%d", i)).
			SetPlatform("x_twitter").
			SetPlatformKey("x_twitter").
			SetNameKey(fmt.Sprintf("random_proxy_account_%d", i)).
			SetAssignedUserID(user.ID).
			SaveX(ctx)
		accountIDs = append(accountIDs, account.ID)
	}

	firstEndpoint := "http://8.8.8.8:8080"
	secondEndpoint := "http://8.8.4.4:8080"
	firstProxy, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "random proxy one", IPType: SocialIPTypeResidential, Endpoint: &firstEndpoint})
	require.NoError(t, err)
	secondProxy, err := ipSvc.Create(ctx, &CreateSocialIPInput{UserID: user.ID, Name: "random proxy two", IPType: SocialIPTypeResidential, Endpoint: &secondEndpoint})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(firstProxy.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	client.SocialIP.UpdateOneID(secondProxy.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	firstProxy, err = ipSvc.GetByID(ctx, firstProxy.ID)
	require.NoError(t, err)
	secondProxy, err = ipSvc.GetByID(ctx, secondProxy.ID)
	require.NoError(t, err)

	result, err := accountSvc.BatchSetDefaultProxyForUser(ctx, user.ID, accountIDs, DefaultProxyAssignmentRandom, nil, []*SocialIP{firstProxy, secondProxy})
	require.NoError(t, err)
	require.Equal(t, 5, result.Total)
	require.Equal(t, 5, result.Succeeded)
	require.Zero(t, result.Failed)

	counts := map[int64]int{}
	for _, accountID := range accountIDs {
		stored := client.SocialAccount.GetX(ctx, accountID)
		require.NotNil(t, stored.DefaultProxySnapshot)
		proxyID, ok := SocialIPIDFromSnapshot(*stored.DefaultProxySnapshot)
		require.True(t, ok)
		counts[proxyID]++
	}
	require.Len(t, counts, 2)
	require.ElementsMatch(t, []int{2, 3}, []int{counts[firstProxy.ID], counts[secondProxy.ID]})
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
	require.NotNil(t, updated.DefaultProxySnapshot)
	require.Greater(t, len(*updated.DefaultProxySnapshot), 255)
	require.Contains(t, *updated.DefaultProxySnapshot, endpoint)
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
		Action:         SocialTaskActionFollow,
		Target:         &target,
		Content:        &content,
		Status:         SocialTaskLogStatusFailed,
		ResultMessage:  &message,
		ProxyID:        &proxyID,
		ProxySnapshot:  &proxySnapshot,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)
	require.Zero(t, log.Price)
	require.Zero(t, log.ChargedAmount)
	require.Equal(t, SocialTaskChargeStatusNotCharged, log.ChargeStatus)
	require.Nil(t, log.ChargeSource)
	require.Equal(t, proxyID, *log.ProxyID)
	require.Equal(t, proxySnapshot, *log.ProxySnapshot)
	require.Equal(t, idempotencyKey, *log.IdempotencyKey)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Zero(t, stored.Price)
	require.Zero(t, stored.ChargedAmount)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)

	loginLog, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLogin,
		Status:    SocialTaskLogStatusFailed,
	})
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, loginLog.Price, 0.000001)
	require.Zero(t, loginLog.ChargedAmount)
	require.Equal(t, SocialTaskChargeStatusNotCharged, loginLog.ChargeStatus)
}

func TestSocialIPSnapshotUsableRequiresExecutableSnapshot(t *testing.T) {
	require.True(t, SocialIPSnapshotUsable(`{"id":42,"name":"proxy","ip_type":"residential","endpoint":"http://proxy.local:8080","status":"online"}`))
	require.False(t, SocialIPSnapshotUsable(`http://proxy.local:8080`))
	require.False(t, SocialIPSnapshotUsable(`{"id":42,"name":"proxy","ip_type":"residential","endpoint":"http://proxy.local:8080","status":"offline"}`))
	require.False(t, SocialIPSnapshotUsable(`{"id":42,"name":"proxy","ip_type":"residential","endpoint":"","status":"online"}`))
	require.False(t, SocialIPSnapshotUsable(`{"id":0,"name":"proxy","ip_type":"residential","endpoint":"http://proxy.local:8080","status":"online"}`))
}

func TestEnsureSocialIPUsableForExecutionRequiresOnlineProxyWithEndpoint(t *testing.T) {
	onlineEndpoint := "http://proxy.local:8080"

	require.NoError(t, EnsureSocialIPUsableForExecution(&SocialIP{
		ID:       42,
		Endpoint: &onlineEndpoint,
		Status:   SocialIPStatusOnline,
	}))

	err := EnsureSocialIPUsableForExecution(nil)
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
	require.Contains(t, err.Error(), "social IP is not available")

	err = EnsureSocialIPUsableForExecution(&SocialIP{
		ID:       42,
		Endpoint: &onlineEndpoint,
		Status:   SocialIPStatusOffline,
	})
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
	require.Contains(t, err.Error(), "social IP must pass a connectivity test before execution")

	blankEndpoint := "  "
	err = EnsureSocialIPUsableForExecution(&SocialIP{
		ID:       42,
		Endpoint: &blankEndpoint,
		Status:   SocialIPStatusOnline,
	})
	require.Equal(t, "SOCIAL_IP_NOT_AVAILABLE", infraerrors.Reason(err))
	require.Contains(t, err.Error(), "social IP endpoint is required for execution")
}

func TestSocialTaskLogCapturesStructuredPayloadAndTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("task-payload@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("task_payload_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_payload_account").
		SaveX(ctx)
	payload := SocialTaskPayload{
		Post: &SocialPostPayload{
			Text:         "hello payload",
			QuotePostURL: "https://x.com/northwind/status/1",
			Media: []SocialTaskMediaRef{
				{
					Source:      "library",
					StorageKey:  "social-task/media/post-1.jpg",
					ContentType: "image/jpeg",
				},
			},
		},
	}
	snapshot := &SocialTaskTemplateSnapshot{
		TemplateID:   "tmpl_post_payload",
		TemplateName: "Payload post",
		TemplateType: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents:     []string{"hello payload"},
			QuotePostURL: "https://x.com/northwind/status/1",
			Media: []SocialTaskMediaRef{
				{
					Source:      "library",
					StorageKey:  "social-task/media/post-1.jpg",
					ContentType: "image/jpeg",
				},
			},
		},
	}

	log, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:        account.ID,
		UserID:           user.ID,
		Action:           SocialTaskActionPost,
		Content:          socialStringPtr("hello payload"),
		Payload:          &payload,
		TemplateSnapshot: snapshot,
		Status:           SocialTaskLogStatusPending,
	})

	require.NoError(t, err)
	require.NotNil(t, log.Payload)
	require.NotNil(t, log.Payload.Post)
	require.Equal(t, "hello payload", log.Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/1", log.Payload.Post.QuotePostURL)
	require.Len(t, log.Payload.Post.Media, 1)
	require.NotNil(t, log.TemplateSnapshot)
	require.Equal(t, snapshot.TemplateID, log.TemplateSnapshot.TemplateID)
	require.Equal(t, snapshot.TemplateName, log.TemplateSnapshot.TemplateName)
	require.Equal(t, snapshot.TemplateType, log.TemplateSnapshot.TemplateType)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Payload)
	require.NotNil(t, stored.TemplateSnapshot)
	require.Equal(t, "hello payload", stored.Payload.Post.Text)
	require.Equal(t, "https://x.com/northwind/status/1", stored.Payload.Post.QuotePostURL)
	require.Equal(t, "tmpl_post_payload", stored.TemplateSnapshot.TemplateID)
	require.Equal(t, "Payload post", stored.TemplateSnapshot.TemplateName)
}

func TestSocialTaskLogMaterializesInlinePostMediaIntoTaskMediaAssets(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("task-media-asset@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("task_media_asset_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("task_media_asset_account").
		SaveX(ctx)
	inlineMedia := SocialTaskMediaRef{
		Source:      "inline",
		ContentType: "image/png",
		FileName:    "materialized.png",
		URL:         inlinePNGDataURLForSocialTaskValidation(t, 640, 640),
	}
	payload := SocialTaskPayload{
		Post: &SocialPostPayload{
			Text:  "hello payload",
			Media: []SocialTaskMediaRef{inlineMedia},
		},
	}
	snapshot := &SocialTaskTemplateSnapshot{
		TemplateID:   "tmpl_post_media_materialized",
		TemplateName: "Payload media materialized",
		TemplateType: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents: []string{"hello payload"},
			Media:    []SocialTaskMediaRef{inlineMedia},
		},
	}

	log, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:        account.ID,
		UserID:           user.ID,
		Action:           SocialTaskActionPost,
		Content:          socialStringPtr("hello payload"),
		Payload:          &payload,
		TemplateSnapshot: snapshot,
		Status:           SocialTaskLogStatusPending,
	})

	require.NoError(t, err)
	require.NotNil(t, log.Payload)
	require.NotNil(t, log.Payload.Post)
	require.Len(t, log.Payload.Post.Media, 1)
	require.Equal(t, "library", log.Payload.Post.Media[0].Source)
	require.NotEmpty(t, log.Payload.Post.Media[0].StorageKey)
	require.Empty(t, log.Payload.Post.Media[0].URL)
	require.Equal(t, "image/png", log.Payload.Post.Media[0].ContentType)
	require.Equal(t, "materialized.png", log.Payload.Post.Media[0].FileName)
	require.Equal(t, 640, log.Payload.Post.Media[0].Width)
	require.Equal(t, 640, log.Payload.Post.Media[0].Height)
	require.NotNil(t, log.TemplateSnapshot)
	require.Len(t, log.TemplateSnapshot.Params.Media, 1)
	require.Equal(t, "library", log.TemplateSnapshot.Params.Media[0].Source)
	require.NotEmpty(t, log.TemplateSnapshot.Params.Media[0].StorageKey)
	require.Empty(t, log.TemplateSnapshot.Params.Media[0].URL)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Payload)
	require.NotNil(t, stored.Payload.Post)
	require.Equal(t, "library", stored.Payload.Post.Media[0].Source)
	require.NotEmpty(t, stored.Payload.Post.Media[0].StorageKey)
	require.Empty(t, stored.Payload.Post.Media[0].URL)
	require.Equal(t, stored.Payload.Post.Media[0].StorageKey, stored.TemplateSnapshot.Params.Media[0].StorageKey)

	rows, err := client.QueryContext(ctx, `
SELECT user_id, storage_provider, storage_key, url, content_type, file_name, byte_size, width, height
FROM social_task_media_assets`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	require.True(t, rows.Next())
	var storedUserID int64
	var provider string
	var storageKey string
	var rawURL string
	var contentType string
	var fileName string
	var byteSize int64
	var width int
	var height int
	require.NoError(t, rows.Scan(&storedUserID, &provider, &storageKey, &rawURL, &contentType, &fileName, &byteSize, &width, &height))
	require.Equal(t, user.ID, storedUserID)
	require.Equal(t, "inline", provider)
	require.Equal(t, stored.Payload.Post.Media[0].StorageKey, storageKey)
	require.Contains(t, rawURL, "data:image/png;base64,")
	require.Equal(t, "image/png", contentType)
	require.Equal(t, "materialized.png", fileName)
	require.Positive(t, byteSize)
	require.Equal(t, 640, width)
	require.Equal(t, 640, height)
	require.False(t, rows.Next())
}

func TestSocialTaskLogMaterializesInlineProfileMediaIntoTaskMediaAssets(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().SetEmail("task-profile-media-asset@example.com").SetPasswordHash("hash").SaveX(ctx)

	avatar := SocialTaskMediaRef{
		Source:      "inline",
		ContentType: "image/png",
		FileName:    "avatar.png",
		URL:         inlinePNGDataURLForSocialTaskValidation(t, 400, 400),
	}
	banner := SocialTaskMediaRef{
		Source:      "inline",
		ContentType: "image/jpeg",
		FileName:    "banner.jpg",
		URL:         inlineJPEGDataURLForSocialTaskValidation(t, 1500, 500),
	}

	testCases := []struct {
		name            string
		action          string
		payload         SocialTaskPayload
		template        *SocialTaskTemplateSnapshot
		assertLogMedia  func(t *testing.T, log *SocialTaskLog)
		assertStoredSQL func(t *testing.T, rows *sql.Rows)
	}{
		{
			name:   "avatar",
			action: SocialTaskActionUpdateAvatar,
			payload: SocialTaskPayload{
				Avatar: &avatar,
			},
			template: &SocialTaskTemplateSnapshot{
				TemplateID:   "tmpl_avatar_media_materialized",
				TemplateName: "Avatar materialized",
				TemplateType: SocialTaskActionUpdateAvatar,
				Params: TaskTemplateParams{
					Avatar: &avatar,
				},
			},
			assertLogMedia: func(t *testing.T, log *SocialTaskLog) {
				require.NotNil(t, log.Payload.Avatar)
				require.Equal(t, "library", log.Payload.Avatar.Source)
				require.NotEmpty(t, log.Payload.Avatar.StorageKey)
				require.Empty(t, log.Payload.Avatar.URL)
				require.NotNil(t, log.TemplateSnapshot.Params.Avatar)
				require.Equal(t, "library", log.TemplateSnapshot.Params.Avatar.Source)
			},
		},
		{
			name:   "banner",
			action: SocialTaskActionUpdateBanner,
			payload: SocialTaskPayload{
				Banner: &banner,
			},
			template: &SocialTaskTemplateSnapshot{
				TemplateID:   "tmpl_banner_media_materialized",
				TemplateName: "Banner materialized",
				TemplateType: SocialTaskActionUpdateBanner,
				Params: TaskTemplateParams{
					Banner: &banner,
				},
			},
			assertLogMedia: func(t *testing.T, log *SocialTaskLog) {
				require.NotNil(t, log.Payload.Banner)
				require.Equal(t, "library", log.Payload.Banner.Source)
				require.NotEmpty(t, log.Payload.Banner.StorageKey)
				require.Empty(t, log.Payload.Banner.URL)
				require.NotNil(t, log.TemplateSnapshot.Params.Banner)
				require.Equal(t, "library", log.TemplateSnapshot.Params.Banner.Source)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			accountKey := "task_profile_media_asset_account_" + tc.name
			account := client.SocialAccount.Create().
				SetName(accountKey).
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(accountKey).
				SaveX(ctx)
			log, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
				AccountID:        account.ID,
				UserID:           user.ID,
				Action:           tc.action,
				Payload:          &tc.payload,
				TemplateSnapshot: tc.template,
				Status:           SocialTaskLogStatusPending,
			})
			require.NoError(t, err)
			require.NotNil(t, log.TemplateSnapshot)
			tc.assertLogMedia(t, log)
		})
	}

	rows, err := client.QueryContext(ctx, `
SELECT storage_provider, file_name
FROM social_task_media_assets
ORDER BY id ASC`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var providers []string
	var fileNames []string
	for rows.Next() {
		var provider string
		var fileName string
		require.NoError(t, rows.Scan(&provider, &fileName))
		providers = append(providers, provider)
		fileNames = append(fileNames, fileName)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"inline", "inline"}, providers)
	require.ElementsMatch(t, []string{"avatar.png", "banner.jpg"}, fileNames)
}

func TestAccountWorkbenchServiceSubmitTaskCapturesTemplateSnapshotAndStructuredPayload(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("structured-submit@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("structured_submit_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("structured_submit_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ipSvc := NewSocialIPService(client)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "structured submit proxy",
		IPType:   SocialIPTypeResidential,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, ip)
	require.NoError(t, err)

	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchService(accountSvc, ipSvc, billing, nil)
	quoteURL := "https://x.com/northwind/status/1"
	content := "hello structured submit"
	media := []SocialTaskMediaRef{
		{
			Source:      "library",
			StorageKey:  "social-task/media/post-structured.jpg",
			ContentType: "image/jpeg",
		},
	}
	templateSnapshot := &SocialTaskTemplateSnapshot{
		TemplateID:   "tmpl_structured_submit",
		TemplateName: "Structured submit",
		TemplateType: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Contents:     []string{content},
			QuotePostURL: quoteURL,
			Media:        media,
		},
	}

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionPost,
		Content:    &content,
		Payload: &SocialTaskPayload{
			Post: &SocialPostPayload{
				Text:         content,
				QuotePostURL: quoteURL,
				Media:        media,
			},
		},
		TemplateSnapshot: templateSnapshot,
	})

	require.NoError(t, err)
	require.Len(t, result.Logs, 1)
	require.NotNil(t, result.Logs[0].Payload)
	require.NotNil(t, result.Logs[0].Payload.Post)
	require.Equal(t, content, result.Logs[0].Payload.Post.Text)
	require.Equal(t, quoteURL, result.Logs[0].Payload.Post.QuotePostURL)
	require.Len(t, result.Logs[0].Payload.Post.Media, 1)
	require.NotNil(t, result.Logs[0].TemplateSnapshot)
	require.Equal(t, templateSnapshot.TemplateID, result.Logs[0].TemplateSnapshot.TemplateID)
	require.Equal(t, templateSnapshot.TemplateName, result.Logs[0].TemplateSnapshot.TemplateName)
	require.Equal(t, templateSnapshot.TemplateType, result.Logs[0].TemplateSnapshot.TemplateType)

	stored, err := client.SocialTaskLog.Get(ctx, result.Logs[0].ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Payload)
	require.NotNil(t, stored.TemplateSnapshot)
	require.Equal(t, content, stored.Payload.Post.Text)
	require.Equal(t, quoteURL, stored.Payload.Post.QuotePostURL)
	require.Equal(t, "tmpl_structured_submit", stored.TemplateSnapshot.TemplateID)
}

func TestAccountWorkbenchServiceSubmitTaskAcceptsMediaOnlyPostPayload(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("media-only-submit@example.com").
		SetPasswordHash("hash").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("media_only_submit_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("media_only_submit_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ipSvc := NewSocialIPService(client)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "media only submit proxy",
		IPType:   SocialIPTypeResidential,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, ip)
	require.NoError(t, err)

	userRepo := &socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}
	billing := NewSocialBillingService(userRepo, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	workbench := NewAccountWorkbenchService(accountSvc, ipSvc, billing, nil)
	media := []SocialTaskMediaRef{{
		Source:      "library",
		StorageKey:  "social-task/media/post-media-only.jpg",
		ContentType: "image/jpeg",
	}}
	templateSnapshot := &SocialTaskTemplateSnapshot{
		TemplateID:   "tmpl_media_only_submit",
		TemplateName: "Media only submit",
		TemplateType: SocialTaskActionPost,
		Params: TaskTemplateParams{
			Media: media,
		},
	}

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionPost,
		Payload: &SocialTaskPayload{
			Post: &SocialPostPayload{
				Media: media,
			},
		},
		TemplateSnapshot: templateSnapshot,
	})

	require.NoError(t, err)
	require.Len(t, result.Logs, 1)
	require.NotNil(t, result.Logs[0].Payload)
	require.NotNil(t, result.Logs[0].Payload.Post)
	require.Equal(t, "", result.Logs[0].Payload.Post.Text)
	require.Len(t, result.Logs[0].Payload.Post.Media, 1)
	require.Equal(t, "library", result.Logs[0].Payload.Post.Media[0].Source)
	require.Equal(t, "social-task/media/post-media-only.jpg", result.Logs[0].Payload.Post.Media[0].StorageKey)
	require.NotNil(t, result.Logs[0].TemplateSnapshot)
	require.Equal(t, templateSnapshot.TemplateID, result.Logs[0].TemplateSnapshot.TemplateID)

	stored, err := client.SocialTaskLog.Get(ctx, result.Logs[0].ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Payload.Post)
	require.Equal(t, "", stored.Payload.Post.Text)
	require.Len(t, stored.Payload.Post.Media, 1)
	require.Equal(t, "social-task/media/post-media-only.jpg", stored.Payload.Post.Media[0].StorageKey)
}

func TestAccountWorkbenchServiceSubmitTaskUsesPlatformKeyForBillingPrecheck(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("platform-key-billing-submit@example.com").
		SetPasswordHash("hash").
		SetBalance(0).
		SaveX(ctx)
	limit := 0.20
	group := &Group{
		ID:               9,
		Name:             "X quota",
		Platform:         "x_twitter",
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &limit,
		Hydrated:         true,
	}
	subRepo := &subscriptionRepoState{
		sub: &UserSubscription{
			ID:               7,
			UserID:           user.ID,
			GroupID:          group.ID,
			PlanPlatform:     "x_twitter",
			StartsAt:         time.Now().AddDate(0, 0, -3),
			ExpiresAt:        time.Now().AddDate(0, 0, 3),
			Status:           SubscriptionStatusActive,
			DailyWindowStart: socialPtrTime(time.Now().Add(-time.Hour)),
			Group:            group,
		},
	}
	account := client.SocialAccount.Create().
		SetName("platform_key_submit_account").
		SetPlatform("twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("platform_key_submit_account").
		SetPassword("platform-key-secret").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "x_twitter", storedAccount.PlatformKey)
	accountView, err := accountSvc.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "x_twitter", accountView.Platform)
	endpoint := "http://8.8.8.8:8080"
	ipSvc := NewSocialIPService(client)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "platform key submit proxy",
		IPType:   SocialIPTypeResidential,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, ip)
	require.NoError(t, err)

	billing := NewSocialBillingService(
		&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 0}},
		subRepo,
		&socialBillingGroupRepoStub{group: group},
		nil,
	)
	workbench := NewAccountWorkbenchService(accountSvc, ipSvc, billing, nil)
	target := "@northwind"

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionLogin,
		Target:     &target,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Submitted)
	require.Equal(t, 1, result.FailedClosed)
	require.Len(t, result.Logs, 1)
	require.Equal(t, SocialTaskLogStatusFailed, result.Logs[0].Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, result.Logs[0].ChargeStatus)
	require.Zero(t, result.Logs[0].ChargedAmount)
	stored, err := client.SocialTaskLog.Get(ctx, result.Logs[0].ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
}

func TestAccountWorkbenchServiceSubmitTaskSkipsAffordabilityPrecheckForFreeActions(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("free-follow-submit@example.com").
		SetPasswordHash("hash").
		SetBalance(0).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("free_follow_submit_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("free_follow_submit_account").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	endpoint := "http://8.8.8.8:8080"
	ipSvc := NewSocialIPService(client)
	ip, err := ipSvc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "free follow submit proxy",
		IPType:   SocialIPTypeResidential,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	client.SocialIP.UpdateOneID(ip.ID).SetStatus(SocialIPStatusOnline).SaveX(ctx)
	ip, err = ipSvc.GetByID(ctx, ip.ID)
	require.NoError(t, err)
	_, err = accountSvc.SetDefaultProxyForUser(ctx, account.ID, user.ID, ip)
	require.NoError(t, err)

	billing := NewSocialBillingService(
		&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 0}},
		&subscriptionRepoState{},
		&socialBillingGroupRepoStub{},
		nil,
	)
	workbench := NewAccountWorkbenchService(accountSvc, ipSvc, billing, nil)
	target := "@northwind"

	result, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionFollow,
		Target:     &target,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Submitted)
	require.Equal(t, 1, result.FailedClosed)
	require.Len(t, result.Logs, 1)
	require.Equal(t, SocialTaskLogStatusFailed, result.Logs[0].Status)
	require.Zero(t, result.Logs[0].Price)
	require.Equal(t, SocialTaskChargeStatusNotCharged, result.Logs[0].ChargeStatus)
	require.Zero(t, result.Logs[0].ChargedAmount)
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

func TestCreateTaskLogMapsActiveTaskUniqueConstraintToBusy(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	createSocialTaskActiveUniqueIndexForTest(t, ctx, client)
	user := client.User.Create().SetEmail("active-unique-busy@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("active_unique_busy_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("active_unique_busy_account").
		SaveX(ctx)

	first, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLoginCheck,
		Status:    SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	second, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionFollow,
		Status:    SocialTaskLogStatusPending,
	})
	require.Nil(t, second)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_TASK_ACCOUNT_BUSY", infraerrors.Reason(err))

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestCreateTaskLogAllowsNewTaskAfterFailedTaskReleasesActiveConstraint(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	createSocialTaskActiveUniqueIndexForTest(t, ctx, client)
	user := client.User.Create().SetEmail("active-release-failed@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("active_release_failed_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("active_release_failed_account").
		SaveX(ctx)

	first, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLoginCheck,
		Status:    SocialTaskLogStatusPending,
	})
	require.NoError(t, err)

	failed, err := svc.MarkTaskLogFailedNotCharged(ctx, first.ID, "queue unavailable")
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, failed.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, failed.ChargeStatus)
	require.Zero(t, failed.ChargedAmount)

	second, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLoginCheck,
		Status:    SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	require.NotZero(t, second.ID)
	require.NotEqual(t, first.ID, second.ID)
	require.Equal(t, SocialTaskLogStatusPending, second.Status)
}

func TestFinalizeSuccessfulTaskReleasesActiveConstraintForNextTask(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	createSocialTaskActiveUniqueIndexForTest(t, ctx, client)
	user := client.User.Create().SetEmail("active-release-success@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("active_release_success_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("active_release_success_account").
		SaveX(ctx)
	first, err := accountSvc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLoginCheck,
		Status:    SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	_, err = client.SocialTaskLog.UpdateOneID(first.ID).SetStatus(SocialTaskLogStatusRunning).Save(ctx)
	require.NoError(t, err)

	billing := NewSocialBillingService(nil, nil, nil, nil)
	charge, err := billing.FinalizeSuccessfulTask(ctx, client, first.ID, user.ID, 0, "login check succeeded")
	require.NoError(t, err)
	require.NotNil(t, charge)
	require.Zero(t, charge.Amount)
	storedFirst := client.SocialTaskLog.GetX(ctx, first.ID)
	require.Equal(t, SocialTaskLogStatusSuccess, storedFirst.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedFirst.ChargeStatus)
	require.Zero(t, storedFirst.ChargedAmount)

	second, err := accountSvc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID: account.ID,
		UserID:    user.ID,
		Action:    SocialTaskActionLoginCheck,
		Status:    SocialTaskLogStatusPending,
	})
	require.NoError(t, err)
	require.NotZero(t, second.ID)
	require.NotEqual(t, first.ID, second.ID)
	require.Equal(t, SocialTaskLogStatusPending, second.Status)
}

func createSocialTaskActiveUniqueIndexForTest(t *testing.T, ctx context.Context, client *dbent.Client) {
	t.Helper()
	_, err := client.ExecContext(ctx, `
CREATE UNIQUE INDEX idx_social_task_logs_one_active_per_account_test
ON social_task_logs (social_account_id)
WHERE status IN ('pending', 'running')`)
	require.NoError(t, err)
}

func TestSocialTaskLogIdempotencyRejectsDifferentTaskPayload(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)
	user := client.User.Create().SetEmail("idempotency-conflict-user@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("idem_conflict_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("idem_conflict_account").
		SaveX(ctx)
	idempotencyKey := "idem-conflict-123"
	firstTarget := "@first_target"
	secondTarget := "@second_target"

	first, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionFollow,
		Target:         &firstTarget,
		Status:         SocialTaskLogStatusPending,
		IdempotencyKey: &idempotencyKey,
	})
	require.NoError(t, err)
	require.Equal(t, firstTarget, requireSocialStringPtr(t, first.Target))

	second, err := svc.CreateTaskLog(ctx, &CreateSocialTaskLogInput{
		AccountID:      account.ID,
		UserID:         user.ID,
		Action:         SocialTaskActionFollow,
		Target:         &secondTarget,
		Status:         SocialTaskLogStatusPending,
		IdempotencyKey: &idempotencyKey,
	})
	require.ErrorIs(t, err, ErrSocialTaskIdempotencyConflict)
	require.Nil(t, second)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAccountWorkbenchServiceSubmitTaskRejectsIdempotencyKeyWithDifferentTarget(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("workbench-idem-conflict@example.com").
		SetPasswordHash("hash").
		SetBalance(1).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("workbench_idem_conflict").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_idem_conflict").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	billing := NewSocialBillingService(
		&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}},
		&subscriptionRepoState{},
		nil,
		nil,
	)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)
	idempotencyKey := "workbench-idem-conflict-123"
	firstTarget := "@first_target"
	secondTarget := "@second_target"

	first, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:           AccountWorkbenchTaskModeAdmin,
		AccountIDs:     []int64{account.ID},
		Action:         SocialTaskActionFollow,
		Target:         &firstTarget,
		IdempotencyKey: idempotencyKey,
	})
	require.NoError(t, err)
	require.Equal(t, 1, first.Submitted)
	require.Equal(t, 1, first.FailedClosed)
	require.Len(t, first.Logs, 1)
	require.Equal(t, firstTarget, requireSocialStringPtr(t, first.Logs[0].Target))

	second, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:           AccountWorkbenchTaskModeAdmin,
		AccountIDs:     []int64{account.ID},
		Action:         SocialTaskActionFollow,
		Target:         &secondTarget,
		IdempotencyKey: idempotencyKey,
	})
	require.ErrorIs(t, err, ErrSocialTaskIdempotencyConflict)
	require.Nil(t, second)

	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	stored := client.SocialTaskLog.GetX(ctx, first.Logs[0].ID)
	require.NotNil(t, stored.Target)
	require.Equal(t, firstTarget, *stored.Target)
}

func TestAccountWorkbenchServiceSubmitTaskKeepsTemplatePoolIndexAfterIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	accountSvc := NewSocialAccountService(client)
	user := client.User.Create().
		SetEmail("workbench-idem-pool@example.com").
		SetPasswordHash("hash").
		SetBalance(1).
		SaveX(ctx)
	firstAccount := client.SocialAccount.Create().
		SetName("workbench_idem_pool_first").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_idem_pool_first").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	secondAccount := client.SocialAccount.Create().
		SetName("workbench_idem_pool_second").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("workbench_idem_pool_second").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	billing := NewSocialBillingService(
		&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}},
		&subscriptionRepoState{},
		nil,
		nil,
	)
	workbench := NewAccountWorkbenchService(accountSvc, NewSocialIPService(client), billing, nil)
	idempotencyKey := "workbench-idem-pool-123"
	targets := []string{"@first_target", "@second_target"}

	first, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:           AccountWorkbenchTaskModeAdmin,
		AccountIDs:     []int64{firstAccount.ID},
		Action:         SocialTaskActionFollow,
		TargetPool:     targets,
		IdempotencyKey: idempotencyKey,
	})
	require.NoError(t, err)
	require.Len(t, first.Logs, 1)
	require.Equal(t, targets[0], requireSocialStringPtr(t, first.Logs[0].Target))

	replayed, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:           AccountWorkbenchTaskModeAdmin,
		AccountIDs:     []int64{firstAccount.ID, secondAccount.ID},
		Action:         SocialTaskActionFollow,
		TargetPool:     targets,
		IdempotencyKey: idempotencyKey,
	})
	require.NoError(t, err)
	require.Equal(t, 2, replayed.Submitted)
	require.Equal(t, 1, replayed.FailedClosed)
	require.Len(t, replayed.Logs, 2)
	require.Equal(t, first.Logs[0].ID, replayed.Logs[0].ID)
	require.Equal(t, targets[1], requireSocialStringPtr(t, replayed.Logs[1].Target))

	storedSecond := client.SocialTaskLog.GetX(ctx, replayed.Logs[1].ID)
	require.NotNil(t, storedSecond.Target)
	require.Equal(t, targets[1], *storedSecond.Target)
	count, err := client.SocialTaskLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestSocialBillingAffordabilityUsesSubscriptionBeforeWallet(t *testing.T) {
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

	check, err := billing.CheckAffordability(ctx, 11, 3)
	require.NoError(t, err)
	require.InEpsilon(t, 0.30, check.RequiredTotal, 0.000001)
	require.InEpsilon(t, 0.20, check.SubscriptionUsage, 0.000001)
	require.InEpsilon(t, 0.10, check.WalletRequired, 0.000001)
	require.True(t, check.CanAfford)

	userRepo.user.Balance = 0.09
	check, err = billing.EnsureCanAfford(ctx, 11, 3)
	require.ErrorIs(t, err, ErrSocialTaskInsufficientFunds)
	require.False(t, check.CanAfford)
}

func TestSocialBillingAffordabilityFiltersSubscriptionsByPlatform(t *testing.T) {
	ctx := context.Background()
	limit := 0.20
	group := &Group{
		ID:               9,
		Name:             "Instagram quota",
		Platform:         "instagram",
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
			PlanPlatform:     "instagram",
			StartsAt:         time.Now().Add(-time.Hour),
			ExpiresAt:        time.Now().Add(time.Hour),
			Status:           SubscriptionStatusActive,
			DailyWindowStart: socialPtrTime(time.Now().Add(-time.Hour)),
			Group:            group,
		},
	}
	userRepo := &socialBillingUserRepoStub{user: &User{ID: 11, Balance: 0.09}}
	billing := NewSocialBillingService(userRepo, subRepo, &socialBillingGroupRepoStub{group: group}, nil)

	check, err := billing.CheckAffordabilityForPlatform(ctx, 11, "x_twitter", 1)
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, check.RequiredTotal, 0.000001)
	require.Zero(t, check.SubscriptionUsage)
	require.InEpsilon(t, SocialTaskUnitPrice, check.WalletRequired, 0.000001)
	require.False(t, check.CanAfford)

	userRepo.user.Balance = 0
	check, err = billing.CheckAffordabilityForPlatform(ctx, 11, "instagram", 1)
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, check.SubscriptionUsage, 0.000001)
	require.Zero(t, check.WalletRequired)
	require.True(t, check.CanAfford)
}

func TestSocialBillingAffordabilityResetsExpiredDailyWindowBeforeAllowance(t *testing.T) {
	ctx := context.Background()
	limit := 0.10
	expiredDailyWindow := time.Now().Add(-25 * time.Hour)
	group := &Group{
		ID:               9,
		Name:             "X daily quota",
		Platform:         "x_twitter",
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
			PlanPlatform:     "x_twitter",
			StartsAt:         time.Now().Add(-48 * time.Hour),
			ExpiresAt:        time.Now().Add(48 * time.Hour),
			Status:           SubscriptionStatusActive,
			DailyWindowStart: &expiredDailyWindow,
			DailyUsageUSD:    limit,
			Group:            group,
		},
	}
	userRepo := &socialBillingUserRepoStub{user: &User{ID: 11, Balance: 0}}
	billing := NewSocialBillingService(userRepo, subRepo, &socialBillingGroupRepoStub{group: group}, nil)

	check, err := billing.CheckAffordabilityForPlatform(ctx, 11, "x_twitter", 1)

	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, check.SubscriptionUsage, 0.000001)
	require.Zero(t, check.WalletRequired)
	require.True(t, check.CanAfford)
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
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

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

	ledgerRows, err := client.UsageLog.Query().
		Order(usagelog.ByBillingType(), usagelog.ByID()).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, ledgerRows, 2)

	var subscriptionLedger *dbent.UsageLog
	var walletLedger *dbent.UsageLog
	for _, row := range ledgerRows {
		require.Equal(t, user.ID, row.UserID)
		require.Nil(t, row.APIKeyID)
		require.Nil(t, row.AccountID)
		require.Equal(t, "social-action", row.Model)
		require.Zero(t, row.InputTokens)
		require.Zero(t, row.OutputTokens)
		require.NotNil(t, row.RequestID)
		require.Contains(t, *row.RequestID, "social-task:")
		require.InEpsilon(t, 0.05, row.TotalCost, 0.000001)
		require.InEpsilon(t, 0.05, row.ActualCost, 0.000001)
		switch {
		case row.SubscriptionID != nil:
			subscriptionLedger = row
		default:
			walletLedger = row
		}
	}
	require.NotNil(t, subscriptionLedger)
	require.Equal(t, sub.ID, *subscriptionLedger.SubscriptionID)
	require.NotNil(t, subscriptionLedger.GroupID)
	require.Equal(t, group.ID, *subscriptionLedger.GroupID)
	require.Contains(t, *subscriptionLedger.RequestID, ":subscription:")
	require.NotNil(t, walletLedger)
	require.Nil(t, walletLedger.GroupID)
	require.Nil(t, walletLedger.SubscriptionID)
	require.Contains(t, *walletLedger.RequestID, ":wallet")

	again, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")
	require.Nil(t, again)
	require.Error(t, err)
	count, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestSocialTaskExecutorRollsBackBillingWhenUsageLedgerFails(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-ledger-rollback@example.com").SetPasswordHash("hash").SetBalance(0.25).SaveX(ctx)
	limit := 0.05
	group := client.Group.Create().
		SetName("Executor rollback quota").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_ledger_rollback").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_ledger_rollback").
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
	ledgerErr := errors.New("usage ledger unavailable")
	client.UsageLog.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpCreate) {
				if usageMutation, ok := m.(*dbent.UsageLogMutation); ok {
					if requestID, exists := usageMutation.RequestID(); exists && strings.HasPrefix(requestID, "social-task:") {
						return nil, ledgerErr
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.Nil(t, charge)
	require.ErrorIs(t, err, ledgerErr)
	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusRunning, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.Nil(t, storedTask.BillingRequestID)
	require.Nil(t, storedTask.ExecutedAt)
	require.Nil(t, storedTask.ResultMessage)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.Zero(t, storedSub.DailyUsageUsd)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.25, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
}

func TestSocialTaskExecutorFinalizesSuccessFromSubscriptionOnly(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-subscription@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	limit := 0.20
	group := client.Group.Create().
		SetName("Executor subscription quota").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

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

func TestSocialTaskExecutorFinalizesSuccessActivatesSubscriptionWindows(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-window-activation@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	limit := 0.20
	group := client.Group.Create().
		SetName("Executor subscription window activation").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SetWeeklyLimitUsd(limit).
		SetMonthlyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
		SetStartsAt(time.Now().AddDate(0, 0, -3)).
		SetExpiresAt(time.Now().AddDate(0, 0, 3)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	require.Nil(t, sub.DailyWindowStart)
	require.Nil(t, sub.WeeklyWindowStart)
	require.Nil(t, sub.MonthlyWindowStart)
	account := client.SocialAccount.Create().
		SetName("executor_window_activation").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_window_activation").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	beforeFinalizeWindowStart := startOfDay(time.Now())

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceSubscription, charge.Source)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	afterFinalizeWindowStart := startOfDay(time.Now())
	require.NotNil(t, storedSub.DailyWindowStart)
	require.NotNil(t, storedSub.WeeklyWindowStart)
	require.NotNil(t, storedSub.MonthlyWindowStart)
	require.True(t,
		storedSub.DailyWindowStart.Equal(beforeFinalizeWindowStart) || storedSub.DailyWindowStart.Equal(afterFinalizeWindowStart),
		"daily window should activate at the settlement day start",
	)
	require.True(t,
		storedSub.WeeklyWindowStart.Equal(beforeFinalizeWindowStart) || storedSub.WeeklyWindowStart.Equal(afterFinalizeWindowStart),
		"weekly window should activate at the settlement day start",
	)
	require.True(t,
		storedSub.MonthlyWindowStart.Equal(beforeFinalizeWindowStart) || storedSub.MonthlyWindowStart.Equal(afterFinalizeWindowStart),
		"monthly window should activate at the settlement day start",
	)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.DailyUsageUsd, 0.000001)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.WeeklyUsageUsd, 0.000001)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.MonthlyUsageUsd, 0.000001)
}

func TestSocialTaskExecutorFinalizesSuccessResetsExpiredDailyWindowBeforeCharging(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-window-reset@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	limit := 0.10
	oldWindowStart := time.Now().Add(-25 * time.Hour)
	group := client.Group.Create().
		SetName("Executor expired window quota").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
		SetStartsAt(time.Now().Add(-48 * time.Hour)).
		SetExpiresAt(time.Now().Add(48 * time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SetDailyWindowStart(oldWindowStart).
		SetDailyUsageUsd(limit).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_window_reset").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_window_reset").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})
	beforeFinalizeWindowStart := startOfDay(time.Now())

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceSubscription, charge.Source)
	require.InEpsilon(t, SocialTaskUnitPrice, charge.SubscriptionAmount, 0.000001)
	require.Zero(t, charge.WalletAmount)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	afterFinalizeWindowStart := startOfDay(time.Now())
	require.True(t,
		storedSub.DailyWindowStart.Equal(beforeFinalizeWindowStart) || storedSub.DailyWindowStart.Equal(afterFinalizeWindowStart),
		"expired daily window should reset before settlement uses allowance",
	)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.DailyUsageUsd, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Zero(t, storedUser.Balance)
}

func TestSocialTaskExecutorFinalizesSuccessTreatsZeroGuardrailsAsUnlimited(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-zero-guardrails@example.com").SetPasswordHash("hash").SetBalance(0).SaveX(ctx)
	zero := 0.0
	monthlyLimit := 0.10
	group := client.Group.Create().
		SetName("Executor zero guardrail quota").
		SetPlatform("x_twitter").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(zero).
		SetWeeklyLimitUsd(zero).
		SetMonthlyLimitUsd(monthlyLimit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("x_twitter").
		SetDailyLimitUsd(zero).
		SetWeeklyLimitUsd(zero).
		SetMonthlyLimitUsd(monthlyLimit).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_zero_guardrails").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_zero_guardrails").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceSubscription, charge.Source)
	require.InEpsilon(t, SocialTaskUnitPrice, charge.SubscriptionAmount, 0.000001)
	require.Zero(t, charge.WalletAmount)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.DailyUsageUsd, 0.000001)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.WeeklyUsageUsd, 0.000001)
	require.InEpsilon(t, SocialTaskUnitPrice, storedSub.MonthlyUsageUsd, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Zero(t, storedUser.Balance)
}

func TestSocialTaskExecutorFinalizesSuccessIgnoresDifferentPlatformSubscription(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-platform-filter@example.com").SetPasswordHash("hash").SetBalance(0.09).SaveX(ctx)
	limit := 0.20
	group := client.Group.Create().
		SetName("Instagram subscription quota").
		SetPlatform("instagram").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetDailyLimitUsd(limit).
		SaveX(ctx)
	sub := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetPlanPlatform("instagram").
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetStatus(SubscriptionStatusActive).
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_x_platform_filter").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_x_platform_filter").
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.Nil(t, charge)
	require.ErrorIs(t, err, ErrSocialTaskInsufficientFunds)
	storedSub, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.Zero(t, storedSub.DailyUsageUsd)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.09, storedUser.Balance, 0.000001)
	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusRunning, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, SocialTaskUnitPrice, "ok")

	require.NoError(t, err)
	require.Equal(t, SocialTaskChargeSourceWallet, charge.Source)
	require.Zero(t, charge.SubscriptionAmount)
	require.InEpsilon(t, SocialTaskUnitPrice, charge.WalletAmount, 0.000001)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.15, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorFinalizesZeroPriceSuccessWithoutCharge(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("executor-free-success@example.com").SetPasswordHash("hash").SetBalance(0.25).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("executor_free_success").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("executor_free_success").
		SetAssignedUserID(user.ID).
		SaveX(ctx)
	task := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLoginCheck).
		SetStatus(SocialTaskLogStatusRunning).
		SetPrice(0).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

	charge, err := executor.finalizeSuccessfulTask(ctx, task.ID, user.ID, 0, "ok")

	require.NoError(t, err)
	require.NotNil(t, charge)
	require.Zero(t, charge.Amount)
	storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, storedTask.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
	require.Zero(t, storedTask.ChargedAmount)
	require.Nil(t, storedTask.ChargeSource)
	require.Nil(t, storedTask.BillingRequestID)
	require.NotNil(t, storedTask.ExecutedAt)
	require.Equal(t, "ok", *storedTask.ResultMessage)
	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.25, storedUser.Balance, 0.000001)
	ledgerCount, err := client.UsageLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, ledgerCount)
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{}).WithCredentialEncryptor(executionAuthEncryptorStub{})

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
		"https://127.0.0.1/proxy-api",
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

func TestSocialIPServiceNormalizesSocks5EndpointThroughProxyURLParser(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-socks5-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	socks5Endpoint := "socks5://8.8.8.8:1080"
	ip, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID:   user.ID,
		Name:     "socks5 proxy",
		IPType:   "residential",
		Endpoint: &socks5Endpoint,
	})
	require.NoError(t, err)
	require.NotNil(t, ip.Endpoint)
	require.Equal(t, "socks5h://8.8.8.8:1080", *ip.Endpoint)

	stored, err := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, "socks5h://8.8.8.8:1080", *stored.Endpoint)

	nextSocks5Endpoint := "socks5://8.8.4.4:1080"
	updated, err := svc.Update(ctx, ip.ID, &UpdateSocialIPInput{Endpoint: &nextSocks5Endpoint})
	require.NoError(t, err)
	require.NotNil(t, updated.Endpoint)
	require.Equal(t, "socks5h://8.8.4.4:1080", *updated.Endpoint)
	require.Equal(t, SocialIPStatusUnknown, updated.Status)

	stored, err = client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, "socks5h://8.8.4.4:1080", *stored.Endpoint)
}

func TestSocialIPServiceRestrictsProxyTypesToExistingProductContract(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-type-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	defaulted, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID: user.ID,
		Name:   "default typed proxy",
	})
	require.NoError(t, err)
	require.Equal(t, "residential", defaulted.IPType)

	for _, proxyType := range []string{"residential", "static", "dynamic", "mobile", "datacenter"} {
		t.Run("create_"+proxyType, func(t *testing.T) {
			ip, err := svc.Create(ctx, &CreateSocialIPInput{
				UserID: user.ID,
				Name:   "allowed " + proxyType,
				IPType: proxyType,
			})
			require.NoError(t, err)
			require.Equal(t, proxyType, ip.IPType)
		})
	}

	invalidType := "rotating"
	_, err = svc.Create(ctx, &CreateSocialIPInput{
		UserID: user.ID,
		Name:   "invalid proxy",
		IPType: invalidType,
	})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))

	_, err = svc.Update(ctx, defaulted.ID, &UpdateSocialIPInput{IPType: &invalidType})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))

	stored, err := client.SocialIP.Get(ctx, defaulted.ID)
	require.NoError(t, err)
	require.Equal(t, "residential", stored.IPType)
}

func TestSocialIPServiceRequiresNonBlankProxyName(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-name-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	svc := NewSocialIPService(client)

	blankName := "   "
	_, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID: user.ID,
		Name:   blankName,
		IPType: "residential",
	})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_NAME_REQUIRED", infraerrors.Reason(err))

	trimmed, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID: user.ID,
		Name:   "  named proxy  ",
		IPType: "residential",
	})
	require.NoError(t, err)
	require.Equal(t, "named proxy", trimmed.Name)

	_, err = svc.Update(ctx, trimmed.ID, &UpdateSocialIPInput{Name: &blankName})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_NAME_REQUIRED", infraerrors.Reason(err))

	stored, err := client.SocialIP.Get(ctx, trimmed.ID)
	require.NoError(t, err)
	require.Equal(t, "named proxy", stored.Name)
}

func TestSocialIPServiceRejectsMissingOwnerBeforeCreate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialIPService(client)

	_, err := svc.Create(ctx, &CreateSocialIPInput{
		UserID: 404,
		Name:   "orphan proxy",
		IPType: "residential",
	})

	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_OWNER_NOT_FOUND", infraerrors.Reason(err))
}

func TestSocialIPServiceRejectsMissingUpdateInput(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-update-input-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("stable proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	svc := NewSocialIPService(client)

	_, err := svc.Update(ctx, ip.ID, nil)

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "SOCIAL_IP_INPUT_REQUIRED", infraerrors.Reason(err))
	stored := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, "stable proxy", stored.Name)
	require.Equal(t, SocialIPStatusOnline, stored.Status)
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

	blankEndpoint := "   "
	updated, err = svc.Update(ctx, ip.ID, &UpdateSocialIPInput{Endpoint: &blankEndpoint})
	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, updated.Status)
	require.Nil(t, updated.Endpoint)
	require.Nil(t, updated.LatencyMs)
	require.Nil(t, updated.LastCheckAt)

	stored, err = client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Nil(t, stored.Endpoint)
	require.Equal(t, SocialIPStatusUnknown, stored.Status)
	require.Nil(t, stored.LatencyMs)
	require.Nil(t, stored.LastCheckAt)
}

func TestSocialIPServiceKeepsConnectivityStatusWhenEndpointUnchanged(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-status-unchanged-owner@example.com").
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
	staleSnapshot := fmt.Sprintf(`{"id":%d,"name":"status proxy","ip_type":"residential","endpoint":%q,"status":"unknown"}`, ip.ID, endpoint)
	account := client.SocialAccount.Create().
		SetName("proxy_update_snapshot_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_update_snapshot_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(staleSnapshot).
		SaveX(ctx)
	newName := "renamed status proxy"
	updated, err := svc.Update(ctx, ip.ID, &UpdateSocialIPInput{
		Name:     &newName,
		Endpoint: &endpoint,
	})
	require.NoError(t, err)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, SocialIPStatusOnline, updated.Status)
	require.NotNil(t, updated.LatencyMs)
	require.Equal(t, latency, *updated.LatencyMs)
	require.NotNil(t, updated.LastCheckAt)

	stored, err := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, newName, stored.Name)
	require.Equal(t, SocialIPStatusOnline, stored.Status)
	require.NotNil(t, stored.LatencyMs)
	require.Equal(t, latency, *stored.LatencyMs)
	require.NotNil(t, stored.LastCheckAt)
	require.Equal(t, endpoint, *stored.Endpoint)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAccount.DefaultProxySnapshot)
	var snapshot map[string]any
	require.NoError(t, json.Unmarshal([]byte(*storedAccount.DefaultProxySnapshot), &snapshot))
	require.Equal(t, float64(ip.ID), snapshot["id"])
	require.Equal(t, newName, snapshot["name"])
	require.Equal(t, endpoint, snapshot["endpoint"])
	require.Equal(t, SocialIPStatusOnline, snapshot["status"])
}

func TestSocialIPServiceDeleteForUserClearsOwnReferencesBeforeDelete(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-delete-cleanup-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("proxy-delete-cleanup-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("delete cleanup proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	proxy, err := NewSocialIPService(client).GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(proxy)
	account := client.SocialAccount.Create().
		SetName("proxy_delete_cleanup_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_delete_cleanup_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)
	otherAccount := client.SocialAccount.Create().
		SetName("proxy_delete_cleanup_other_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_delete_cleanup_other_account").
		SetAssignedUserID(otherUser.ID).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)
	taskLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusPending).
		SetProxyID(ip.ID).
		SetProxySnapshot(snapshot).
		SaveX(ctx)

	inspectedBeforeDelete := false
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpDelete|dbent.OpDeleteOne) && !inspectedBeforeDelete {
				mutation, ok := m.(*dbent.SocialIPMutation)
				require.True(t, ok)
				ids, err := mutation.IDs(ctx)
				require.NoError(t, err)
				require.Contains(t, ids, ip.ID)
				tx, err := mutation.Tx()
				require.NoError(t, err)
				storedAccount, err := tx.Client().SocialAccount.Get(ctx, account.ID)
				require.NoError(t, err)
				require.Nil(t, storedAccount.DefaultProxySnapshot)
				storedOtherAccount, err := tx.Client().SocialAccount.Get(ctx, otherAccount.ID)
				require.NoError(t, err)
				require.NotNil(t, storedOtherAccount.DefaultProxySnapshot)
				require.Equal(t, snapshot, *storedOtherAccount.DefaultProxySnapshot)
				storedTaskLog, err := tx.Client().SocialTaskLog.Get(ctx, taskLog.ID)
				require.NoError(t, err)
				require.Nil(t, storedTaskLog.ProxyID)
				require.NotNil(t, storedTaskLog.ProxySnapshot)
				require.Equal(t, snapshot, *storedTaskLog.ProxySnapshot)
				inspectedBeforeDelete = true
			}
			return next.Mutate(ctx, m)
		})
	})

	require.NoError(t, NewSocialIPService(client).DeleteForUser(ctx, ip.ID, user.ID))
	require.True(t, inspectedBeforeDelete)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.Nil(t, storedAccount.DefaultProxySnapshot)
	storedOtherAccount := client.SocialAccount.GetX(ctx, otherAccount.ID)
	require.NotNil(t, storedOtherAccount.DefaultProxySnapshot)
	require.Equal(t, snapshot, *storedOtherAccount.DefaultProxySnapshot)
	storedTaskLog := client.SocialTaskLog.GetX(ctx, taskLog.ID)
	require.Nil(t, storedTaskLog.ProxyID)
	require.NotNil(t, storedTaskLog.ProxySnapshot)
	require.Equal(t, snapshot, *storedTaskLog.ProxySnapshot)
	_, err = client.SocialIP.Get(ctx, ip.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestSocialIPServiceDeleteForUserRechecksOwnershipAtDeleteTime(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-delete-race-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("proxy-delete-race-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("racy delete proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SaveX(ctx)
	proxy, err := NewSocialIPService(client).GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(proxy)
	account := client.SocialAccount.Create().
		SetName("proxy_delete_race_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_delete_race_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)
	taskLog := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionFollow).
		SetStatus(SocialTaskLogStatusPending).
		SetProxyID(ip.ID).
		SetProxySnapshot(snapshot).
		SaveX(ctx)

	ownershipMoved := false
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpDelete|dbent.OpDeleteOne) && !ownershipMoved {
				if mutation, ok := m.(*dbent.SocialIPMutation); ok {
					ids, err := mutation.IDs(ctx)
					if err != nil {
						return nil, err
					}
					require.Contains(t, ids, ip.ID)
					tx, err := mutation.Tx()
					if err != nil {
						return nil, err
					}
					_, err = tx.Client().SocialIP.UpdateOneID(ip.ID).SetUserID(otherUser.ID).Save(ctx)
					if err != nil {
						return nil, err
					}
					ownershipMoved = true
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	err = NewSocialIPService(client).DeleteForUser(ctx, ip.ID, user.ID)

	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_FOUND", infraerrors.Reason(err))
	require.True(t, ownershipMoved)
	storedIP := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, user.ID, storedIP.UserID)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAccount.DefaultProxySnapshot)
	require.Equal(t, snapshot, *storedAccount.DefaultProxySnapshot)
	storedTaskLog := client.SocialTaskLog.GetX(ctx, taskLog.ID)
	require.NotNil(t, storedTaskLog.ProxyID)
	require.Equal(t, ip.ID, *storedTaskLog.ProxyID)
	require.NotNil(t, storedTaskLog.ProxySnapshot)
	require.Equal(t, snapshot, *storedTaskLog.ProxySnapshot)
}

func TestSocialIPServiceUpdateForUserRechecksOwnershipAtWriteTime(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-update-race-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("proxy-update-race-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("racy update proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SetRemark("original remark").
		SaveX(ctx)
	proxy, err := NewSocialIPService(client).GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(proxy)
	account := client.SocialAccount.Create().
		SetName("proxy_update_race_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_update_race_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)

	ownershipMoved := false
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) && !ownershipMoved {
				if mutation, ok := m.(*dbent.SocialIPMutation); ok {
					ids, err := mutation.IDs(ctx)
					if err != nil {
						return nil, err
					}
					require.Contains(t, ids, ip.ID)
					tx, err := mutation.Tx()
					if err != nil {
						return nil, err
					}
					ownershipMoved = true
					_, err = tx.Client().SocialIP.UpdateOneID(ip.ID).SetUserID(otherUser.ID).Save(ctx)
					if err != nil {
						return nil, err
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	newName := "should not save"
	newRemark := "should not persist"
	updated, err := NewSocialIPService(client).UpdateForUser(ctx, ip.ID, user.ID, &UpdateSocialIPInput{
		Name:   &newName,
		Remark: &newRemark,
	})

	require.Nil(t, updated)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_FOUND", infraerrors.Reason(err))
	require.True(t, ownershipMoved)
	storedIP := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, user.ID, storedIP.UserID)
	require.Equal(t, "racy update proxy", storedIP.Name)
	require.NotNil(t, storedIP.Remark)
	require.Equal(t, "original remark", *storedIP.Remark)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAccount.DefaultProxySnapshot)
	require.Equal(t, snapshot, *storedAccount.DefaultProxySnapshot)
}

func TestSocialIPCheckerTestIPForUserRechecksOwnershipAtStatusWrite(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-test-race-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("proxy-test-race-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("racy test proxy").
		SetIPType(SocialIPTypeResidential).
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(321).
		SaveX(ctx)
	proxy, err := NewSocialIPService(client).GetByID(ctx, ip.ID)
	require.NoError(t, err)
	snapshot := SocialIPTaskSnapshot(proxy)
	account := client.SocialAccount.Create().
		SetName("proxy_test_race_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_test_race_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(snapshot).
		SaveX(ctx)

	ownershipMoved := false
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) && !ownershipMoved {
				if mutation, ok := m.(*dbent.SocialIPMutation); ok {
					ids, err := mutation.IDs(ctx)
					if err != nil {
						return nil, err
					}
					require.Contains(t, ids, ip.ID)
					tx, err := mutation.Tx()
					if err != nil {
						return nil, err
					}
					ownershipMoved = true
					_, err = tx.Client().SocialIP.UpdateOneID(ip.ID).SetUserID(otherUser.ID).Save(ctx)
					if err != nil {
						return nil, err
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	result, err := NewSocialIPChecker(client).TestIPForUser(ctx, ip.ID, user.ID)

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "SOCIAL_IP_NOT_FOUND", infraerrors.Reason(err))
	require.True(t, ownershipMoved)
	storedIP := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, user.ID, storedIP.UserID)
	require.Equal(t, SocialIPStatusOnline, storedIP.Status)
	require.NotNil(t, storedIP.LatencyMs)
	require.Equal(t, 321, *storedIP.LatencyMs)
	require.Nil(t, storedIP.LastCheckAt)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAccount.DefaultProxySnapshot)
	require.Equal(t, snapshot, *storedAccount.DefaultProxySnapshot)
}

func TestSocialIPCheckerClearsStaleLatencyWhenProxyIsUnknown(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-latency-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("stale latency proxy").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)

	result, err := NewSocialIPChecker(client).TestIP(ctx, ip.ID)

	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, result.Status)
	require.Zero(t, result.LatencyMs)
	stored, err := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, stored.Status)
	require.Nil(t, stored.LatencyMs)
	require.NotNil(t, stored.LastCheckAt)
}

func TestSocialIPCheckerReturnsSafeResultErrors(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-safe-result-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	checker := NewSocialIPChecker(client)
	ptr := func(value string) *string { return &value }

	tests := []struct {
		name        string
		endpoint    *string
		wantStatus  string
		wantMessage string
		blockedText []string
	}{
		{
			name:        "missing endpoint",
			wantStatus:  SocialIPStatusUnknown,
			wantMessage: "proxy endpoint is not ready for connectivity check",
			blockedText: []string{"no endpoint configured"},
		},
		{
			name:        "blocked private endpoint",
			endpoint:    ptr("http://127.0.0.1:8080"),
			wantStatus:  SocialIPStatusOffline,
			wantMessage: "proxy connectivity check failed",
			blockedText: []string{"127.0.0.1", "local address", "private"},
		},
		{
			name:        "malformed endpoint with credentials",
			endpoint:    ptr("http://user:secret_password@:0/"),
			wantStatus:  SocialIPStatusOffline,
			wantMessage: "proxy connectivity check failed",
			blockedText: []string{"secret_password", "missing host", "user:"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			create := client.SocialIP.Create().
				SetUserID(user.ID).
				SetName("safe result " + tc.name).
				SetIPType("residential").
				SetStatus(SocialIPStatusOnline).
				SetLatencyMs(123)
			if tc.endpoint != nil {
				create.SetEndpoint(*tc.endpoint)
			}
			ip := create.SaveX(ctx)

			result, err := checker.TestIP(ctx, ip.ID)

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, result.Status)
			require.Equal(t, tc.wantMessage, result.Error)
			require.Zero(t, result.LatencyMs)
			for _, blocked := range tc.blockedText {
				require.NotContains(t, strings.ToLower(result.Error), strings.ToLower(blocked))
			}
			stored := client.SocialIP.GetX(ctx, ip.ID)
			require.Equal(t, tc.wantStatus, stored.Status)
			require.Nil(t, stored.LatencyMs)
			require.NotNil(t, stored.LastCheckAt)
		})
	}
}

func TestSocialIPCheckerSyncsDefaultProxySnapshotsAfterStatusUpdate(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-snapshot-sync-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("proxy-snapshot-sync-other-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("snapshot sync proxy").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)
	staleSnapshot := fmt.Sprintf(`{"id":%d,"name":"stale proxy","ip_type":"static","endpoint":"http://203.0.113.10:8080","status":"online"}`, ip.ID)
	accountUpdatedAt := time.Date(2026, 6, 24, 10, 1, 0, 0, time.UTC)
	account := client.SocialAccount.Create().
		SetName("proxy_snapshot_sync_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_snapshot_sync_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(staleSnapshot).
		SetUpdatedAt(accountUpdatedAt).
		SaveX(ctx)
	otherSnapshot := `{"id":999999,"name":"other proxy","status":"online"}`
	otherAccount := client.SocialAccount.Create().
		SetName("proxy_snapshot_sync_other").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_snapshot_sync_other").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(otherSnapshot).
		SaveX(ctx)
	crossOwnerSnapshot := fmt.Sprintf(`{"id":%d,"name":"cross owner stale proxy","status":"online"}`, ip.ID)
	crossOwnerAccount := client.SocialAccount.Create().
		SetName("proxy_snapshot_sync_cross_owner").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_snapshot_sync_cross_owner").
		SetAssignedUserID(otherUser.ID).
		SetDefaultProxySnapshot(crossOwnerSnapshot).
		SaveX(ctx)

	result, err := NewSocialIPChecker(client).TestIP(ctx, ip.ID)

	require.NoError(t, err)
	require.Equal(t, SocialIPStatusUnknown, result.Status)
	stored := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, stored.DefaultProxySnapshot)
	var snapshot map[string]any
	require.NoError(t, json.Unmarshal([]byte(*stored.DefaultProxySnapshot), &snapshot))
	require.Equal(t, float64(ip.ID), snapshot["id"])
	require.Equal(t, "snapshot sync proxy", snapshot["name"])
	require.Equal(t, "residential", snapshot["ip_type"])
	require.Equal(t, "", snapshot["endpoint"])
	require.Equal(t, SocialIPStatusUnknown, snapshot["status"])
	require.True(t, stored.UpdatedAt.Equal(accountUpdatedAt), "proxy snapshot maintenance should not make account updated_at shared")
	otherStored := client.SocialAccount.GetX(ctx, otherAccount.ID)
	require.NotNil(t, otherStored.DefaultProxySnapshot)
	require.Equal(t, otherSnapshot, *otherStored.DefaultProxySnapshot)
	crossOwnerStored := client.SocialAccount.GetX(ctx, crossOwnerAccount.ID)
	require.NotNil(t, crossOwnerStored.DefaultProxySnapshot)
	require.Equal(t, crossOwnerSnapshot, *crossOwnerStored.DefaultProxySnapshot)
}

func TestSocialIPServiceMarkExecutionReachableSyncsDefaultProxySnapshots(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-execution-reachable-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	endpoint := "http://203.0.113.20:8080"
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("execution reachable proxy").
		SetIPType("residential").
		SetEndpoint(endpoint).
		SetStatus(SocialIPStatusOffline).
		SaveX(ctx)
	staleSnapshot := fmt.Sprintf(`{"id":%d,"name":"execution reachable proxy","ip_type":"residential","endpoint":%q,"status":"offline"}`, ip.ID, endpoint)
	accountUpdatedAt := time.Date(2026, 6, 24, 11, 2, 0, 0, time.UTC)
	account := client.SocialAccount.Create().
		SetName("proxy_execution_reachable_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("proxy_execution_reachable_account").
		SetAssignedUserID(user.ID).
		SetDefaultProxySnapshot(staleSnapshot).
		SetUpdatedAt(accountUpdatedAt).
		SaveX(ctx)

	err := NewSocialIPService(client).MarkExecutionReachable(ctx, ip.ID)

	require.NoError(t, err)
	storedIP := client.SocialIP.GetX(ctx, ip.ID)
	require.Equal(t, SocialIPStatusOnline, storedIP.Status)
	require.NotNil(t, storedIP.LastCheckAt)
	storedAccount := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, storedAccount.DefaultProxySnapshot)
	var snapshot map[string]any
	require.NoError(t, json.Unmarshal([]byte(*storedAccount.DefaultProxySnapshot), &snapshot))
	require.Equal(t, float64(ip.ID), snapshot["id"])
	require.Equal(t, endpoint, snapshot["endpoint"])
	require.Equal(t, SocialIPStatusOnline, snapshot["status"])
	require.True(t, storedAccount.UpdatedAt.Equal(accountUpdatedAt), "execution proxy reachability sync should not refresh account updated_at")
}

func TestSocialIPCheckerReturnsErrorWhenStatusPersistenceFails(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-persist-failure-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("status persistence proxy").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)
	persistErr := errors.New("status persistence failed")
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) {
				return nil, persistErr
			}
			return next.Mutate(ctx, m)
		})
	})

	result, err := NewSocialIPChecker(client).TestIP(ctx, ip.ID)

	require.Nil(t, result)
	require.ErrorIs(t, err, persistErr)
	stored, getErr := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, getErr)
	require.Equal(t, SocialIPStatusOnline, stored.Status)
	require.NotNil(t, stored.LatencyMs)
	require.Equal(t, 123, *stored.LatencyMs)
	require.Nil(t, stored.LastCheckAt)
}

func TestSocialIPCheckerTestAllReturnsErrorWhenStatusPersistenceFails(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-batch-persist-failure-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	ip := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("batch status persistence proxy").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(123).
		SaveX(ctx)
	persistErr := errors.New("batch status persistence failed")
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) {
				return nil, persistErr
			}
			return next.Mutate(ctx, m)
		})
	})

	results, err := NewSocialIPChecker(client).TestAllByUser(ctx, user.ID)

	require.Nil(t, results)
	require.ErrorIs(t, err, persistErr)
	stored, getErr := client.SocialIP.Get(ctx, ip.ID)
	require.NoError(t, getErr)
	require.Equal(t, SocialIPStatusOnline, stored.Status)
	require.NotNil(t, stored.LatencyMs)
	require.Equal(t, 123, *stored.LatencyMs)
	require.Nil(t, stored.LastCheckAt)
}

func TestSocialIPCheckerTestAllRollsBackEarlierStatusUpdatesWhenLaterPersistenceFails(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().
		SetEmail("proxy-batch-rollback-owner@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	first := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("batch rollback first").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(111).
		SaveX(ctx)
	second := client.SocialIP.Create().
		SetUserID(user.ID).
		SetName("batch rollback second").
		SetIPType("residential").
		SetStatus(SocialIPStatusOnline).
		SetLatencyMs(222).
		SaveX(ctx)
	persistErr := errors.New("second proxy status persistence failed")
	var updateCount int
	client.SocialIP.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			if m.Op().Is(dbent.OpUpdateOne) {
				updateCount++
				if updateCount == 2 {
					return nil, persistErr
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	results, err := NewSocialIPChecker(client).TestAllByUser(ctx, user.ID)

	require.Nil(t, results)
	require.ErrorIs(t, err, persistErr)
	storedFirst, firstErr := client.SocialIP.Get(ctx, first.ID)
	require.NoError(t, firstErr)
	require.Equal(t, SocialIPStatusOnline, storedFirst.Status)
	require.NotNil(t, storedFirst.LatencyMs)
	require.Equal(t, 111, *storedFirst.LatencyMs)
	require.Nil(t, storedFirst.LastCheckAt)
	storedSecond, secondErr := client.SocialIP.Get(ctx, second.ID)
	require.NoError(t, secondErr)
	require.Equal(t, SocialIPStatusOnline, storedSecond.Status)
	require.NotNil(t, storedSecond.LatencyMs)
	require.Equal(t, 222, *storedSecond.LatencyMs)
	require.Nil(t, storedSecond.LastCheckAt)
}

func newSocialOpsServiceTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS social_task_media_assets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	storage_provider TEXT NOT NULL DEFAULT 'inline',
	storage_key TEXT NOT NULL,
	url TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT '',
	file_name TEXT NOT NULL DEFAULT '',
	sha256 TEXT NOT NULL DEFAULT '',
	byte_size INTEGER NOT NULL DEFAULT 0,
	width INTEGER NOT NULL DEFAULT 0,
	height INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(user_id, storage_key)
)`)
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type socialPlatformExecutorFunc func(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error)

func (f socialPlatformExecutorFunc) Execute(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (string, error) {
	return f(ctx, taskLog, account)
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

func (r *subscriptionRepoState) List(context.Context, pagination.PaginationParams, *int64, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
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

func socialStringPtr(value string) *string {
	return &value
}

func requireSocialStringPtr(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func inlinePNGDataURLForSocialTaskValidation(t *testing.T, width, height int) string {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	require.NoError(t, png.Encode(&buf, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func inlineJPEGDataURLForSocialTaskValidation(t *testing.T, width, height int) string {
	t.Helper()

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func requireImagePartDimensions(t *testing.T, contentType string, body []byte, fieldName string, wantWidth, wantHeight int) {
	t.Helper()

	boundary := multipartBoundaryForTest(t, contentType)
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if part.FormName() != fieldName {
			continue
		}
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, wantWidth, cfg.Width)
		require.Equal(t, wantHeight, cfg.Height)
		return
	}
	t.Fatalf("multipart field %q not found", fieldName)
}

func multipartBoundaryForTest(t *testing.T, contentType string) string {
	t.Helper()

	for _, part := range strings.Split(contentType, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "boundary=") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(part, "boundary=")), "\"")
		}
	}
	t.Fatalf("multipart boundary missing from content type %q", contentType)
	return ""
}

type twitterFakeRoundTripper struct {
	handler func(*http.Request) (*http.Response, error)
	calls   int
}

func (t *twitterFakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	if t.handler == nil {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}, nil
	}
	return t.handler(req)
}
