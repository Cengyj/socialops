//go:build unit

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/Wei-Shaw/socialops/ent/socialaccount"
	"github.com/Wei-Shaw/socialops/ent/usagelog"
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
		SocialTaskActionRetweet,
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})

	executor.processTask(log.ID)

	stored, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.ChargeSource)
	require.Nil(t, stored.BillingRequestID)
	require.NotNil(t, stored.ResultMessage)
	require.Equal(t, "该平台动作暂不可用，本次未扣费", *stored.ResultMessage)

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
		SetSocialAccountID(account.ID).
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})

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
	require.Contains(t, *storedSecond.ResultMessage, "queue is full")
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
	require.Contains(t, *stored.ResultMessage, "not configured")
	require.NotContains(t, *stored.ResultMessage, "queue is full")
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

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1, MinIntervalMs: 1})
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

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1, MinIntervalMs: 1})
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
	require.Contains(t, *stored.ResultMessage, "未扣费")

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
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
			name: "account removed from user workbench before execution",
			mutate: func(client *dbent.Client, accountID int64, _ int64) {
				client.SocialAccount.UpdateOneID(accountID).SetUserWorkbenchDeletedAt(time.Now()).SaveX(ctx)
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
			executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
			require.Equal(t, "任务执行失败，本次未扣费", *storedTask.ResultMessage)
			require.NotContains(t, *storedTask.ResultMessage, "social account is unavailable")

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
			ledgerCount, err := client.UsageLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, ledgerCount)
		})
	}
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
	require.Equal(t, "该平台动作暂不可用，本次未扣费", *storedTask.ResultMessage)
	require.NotContains(t, *storedTask.ResultMessage, "panic-secret-token")

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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
	require.Equal(t, "执行代理不可用，本次未扣费", *storedTask.ResultMessage)
	require.NotContains(t, *storedTask.ResultMessage, "user:pass")
	require.NotContains(t, *storedTask.ResultMessage, "secret-token")

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "执行代理不可用，本次未扣费", *storedAccount.TaskMessage)
	require.NotContains(t, *storedAccount.TaskMessage, "user:pass")
	require.NotContains(t, *storedAccount.TaskMessage, "secret-token")

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
	executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
	require.Contains(t, *result.Logs[0].ResultMessage, "not configured")

	stored, err := client.SocialTaskLog.Get(ctx, result.Logs[0].ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, stored.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, stored.ChargeStatus)
	require.Zero(t, stored.ChargedAmount)
	require.Nil(t, stored.BillingRequestID)
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", NewTwitterExecutor())

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
	executionAuth := `{"access_token":"access-token","token_secret":"token-secret","client_uuid":"client-uuid","twitter_client":"TwitterAndroid","client_version":"11.46.0-release.0","twitter_api_version":"5","client_language":"en-US","client_device_id":"device-id","client_limit_ad_tracking":"0","user_agent":"TwitterAndroid/11.46.0-release.0","accept_language":"en-US","accept_encoding":"gzip","timezone":"Pacific/Honolulu","os_security_patch_level":"2024-10-05"}`
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
	twitterExec := NewTwitterExecutor()
	twitterExec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
		require.Equal(t, "http://8.8.8.8:8080", proxyURL)
		return &http.Client{Transport: fakeTransport}, nil
	}
	billing := NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil)
	executor := NewSocialTaskExecutor(client, billing, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
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
	require.Equal(t, SocialTaskStatusPending, storedAccount.TaskStatus)
	require.Nil(t, storedAccount.TaskMessage)
}

func TestTwitterExecutorSendsRealLoginLikePostAndRetweetRequests(t *testing.T) {
	ctx := context.Background()
	executionAuth := `{"access_token":"access-token","token_secret":"token-secret","client_uuid":"client-uuid","twitter_client":"TwitterAndroid","client_version":"11.46.0-release.0","twitter_api_version":"5","client_language":"en-US","client_device_id":"device-id","client_limit_ad_tracking":"0","user_agent":"TwitterAndroid/11.46.0-release.0","accept_language":"en-US","accept_encoding":"gzip","timezone":"Pacific/Honolulu","os_security_patch_level":"2024-10-05"}`
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
			twitterExec := NewTwitterExecutor()
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
	executionAuth := `{"access_token":"access-token","token_secret":"token-secret","client_uuid":"client-uuid","twitter_client":"TwitterAndroid","client_version":"11.46.0-release.0","twitter_api_version":"5","client_language":"en-US","client_device_id":"device-id","client_limit_ad_tracking":"0","user_agent":"TwitterAndroid/11.46.0-release.0","accept_language":"en-US","accept_encoding":"gzip","timezone":"Pacific/Honolulu","os_security_patch_level":"2024-10-05"}`
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", NewTwitterExecutor())

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
	authCookie := "ct0=cookie; auth_token=secret"
	executionAuth := `{"access_token":"token","token_secret":"secret"}`
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
	require.Contains(t, body, `"execution_auth":"{\"access_token\":\"token\",\"token_secret\":\"secret\"}"`)
}

func TestSocialAccountServiceStoresCredentialsWithoutApplicationEncryption(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	password := "x-account-secret"
	emailPassword := "mailbox-secret"
	authCookie := "ct0=create; auth_token=create"
	executionAuth := `{"access_token":"access","token_secret":"secret"}`
	directProxySnapshot := `{"id":99,"name":"stale-proxy"}`
	account, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:                 "northwind_ops",
		Platform:             "x_twitter",
		Password:             &password,
		EmailPassword:        &emailPassword,
		AuthCookie:           &authCookie,
		ExecutionAuth:        &executionAuth,
		DefaultProxySnapshot: &directProxySnapshot,
	})
	require.NoError(t, err)
	require.NotNil(t, account.Password)
	require.NotNil(t, account.EmailPassword)
	require.NotNil(t, account.AuthCookie)
	require.NotNil(t, account.ExecutionAuth)
	require.NotNil(t, account.DefaultProxySnapshot)
	require.Equal(t, password, *account.Password)
	require.Equal(t, emailPassword, *account.EmailPassword)
	require.Equal(t, authCookie, *account.AuthCookie)
	require.Equal(t, executionAuth, *account.ExecutionAuth)
	require.Equal(t, directProxySnapshot, *account.DefaultProxySnapshot)

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.Password)
	require.NotNil(t, stored.EmailPassword)
	require.NotNil(t, stored.AuthCookie)
	require.NotNil(t, stored.ExecutionAuth)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Equal(t, password, *stored.Password)
	require.Equal(t, emailPassword, *stored.EmailPassword)
	require.Equal(t, authCookie, *stored.AuthCookie)
	require.Equal(t, executionAuth, *stored.ExecutionAuth)
	require.Equal(t, directProxySnapshot, *stored.DefaultProxySnapshot)

	updatedPassword := "rotated-secret"
	updatedAuthCookie := "ct0=rotated; auth_token=rotated"
	updatedExecutionAuth := `{"access_token":"rotated","token_secret":"secret"}`
	updated, err := svc.Update(ctx, account.ID, &UpdateSocialAccountInput{Password: &updatedPassword, AuthCookie: &updatedAuthCookie, ExecutionAuth: &updatedExecutionAuth})
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *updated.Password)
	require.Equal(t, updatedAuthCookie, *updated.AuthCookie)
	require.Equal(t, updatedExecutionAuth, *updated.ExecutionAuth)

	stored, err = client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, updatedPassword, *stored.Password)
	require.Equal(t, updatedAuthCookie, *stored.AuthCookie)
	require.Equal(t, updatedExecutionAuth, *stored.ExecutionAuth)

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

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "northwind_ops",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.NoError(t, err)
	require.Equal(t, int64(poolAccount.ID), imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)
	require.Equal(t, password, *imported.Password)

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
		Email:         socialStringPtr("mail@example.com"),
		EmailPassword: socialStringPtr("mail-secret"),
	})
	require.NoError(t, err)
	require.Equal(t, user.ID, *missingImported.AssignedUserID)
	require.Equal(t, SocialAccountStatusNotStored, missingImported.AccountStatus)
	require.Equal(t, SocialTaskStatusPending, missingImported.TaskStatus)
	require.Equal(t, "missing_user", missingImported.Name)
	require.Equal(t, "missing-secret", *missingImported.Password)
	require.Equal(t, "mail@example.com", *missingImported.Email)
	require.Equal(t, "mail-secret", *missingImported.EmailPassword)
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
	require.Nil(t, account.Remark)

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

func TestSocialAccountServiceRemoveFromUserWorkbenchDoesNotMutateTotalPoolOwnership(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user := client.User.Create().
		SetEmail("remove-workbench-user@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	otherUser := client.User.Create().
		SetEmail("remove-workbench-other@example.com").
		SetPasswordHash("hashed-password").
		SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@remove_me").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("remove_me").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	err := svc.RemoveFromUserWorkbench(ctx, user.ID, account.ID)
	require.NoError(t, err)

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err, "ordinary user removal must not soft-delete the total-pool record")
	require.Nil(t, stored.DeletedAt)
	require.NotNil(t, stored.AssignedUserID)
	require.Equal(t, user.ID, int64(*stored.AssignedUserID), "ordinary user removal must not reclaim ownership")

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, visible)
	require.Zero(t, page.Total)

	err = svc.RemoveFromUserWorkbench(ctx, otherUser.ID, account.ID)
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)
}

func TestSocialAccountServiceBatchImportRowRestoresRemovedWorkbenchAccount(t *testing.T) {
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

	require.NoError(t, svc.RemoveFromUserWorkbench(ctx, user.ID, account.ID))

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Platform:  "x_twitter",
		Name:      "@restore_me",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.NoError(t, err)
	require.Equal(t, account.ID, imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)

	visible, page, err := svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, visible, 1)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, account.ID, visible[0].ID)
}

func TestSocialAccountServiceBatchImportRowRestoresPlatformlessRemovedWorkbenchAccount(t *testing.T) {
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
	hiddenAt := time.Now().Add(-time.Hour)
	hidden := client.SocialAccount.Create().
		SetName("@restore_cross").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("restore_cross").
		SetAssignedUserID(user.ID).
		SetUserWorkbenchDeletedAt(hiddenAt).
		SetPassword(password).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	other := client.SocialAccount.Create().
		SetName("restore_cross").
		SetPlatform("instagram").
		SetPlatformKey("instagram").
		SetNameKey("restore_cross").
		SetAssignedUserID(otherUser.ID).
		SaveX(ctx)

	imported, err := svc.importUserWorkbenchAccount(ctx, user.ID, &UserImportSocialAccountInput{
		Name:      "@restore_cross",
		Password:  socialStringPtr("typed-secret"),
		TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP"),
	})
	require.NoError(t, err)
	require.Equal(t, hidden.ID, imported.ID)
	require.Equal(t, user.ID, *imported.AssignedUserID)
	require.Nil(t, imported.UserWorkbenchDeletedAt)
	require.Equal(t, password, *imported.Password)

	storedHidden := client.SocialAccount.GetX(ctx, hidden.ID)
	require.Nil(t, storedHidden.UserWorkbenchDeletedAt)
	require.Equal(t, user.ID, int64(*storedHidden.AssignedUserID))
	require.Equal(t, password, *storedHidden.Password)

	storedOther := client.SocialAccount.GetX(ctx, other.ID)
	require.Equal(t, otherUser.ID, int64(*storedOther.AssignedUserID))
	require.Nil(t, storedOther.UserWorkbenchDeletedAt)
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

func TestSocialAccountServiceBatchImportAndRemoveStayInUserScope(t *testing.T) {
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
	ownHidden := client.SocialAccount.Create().
		SetName("@batch_hidden").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("batch_hidden").
		SetAssignedUserID(user.ID).
		SetUserWorkbenchDeletedAt(time.Now()).
		SaveX(ctx)
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
		{Platform: "x_twitter", Name: "@batch_hidden", Password: socialStringPtr("typed-secret"), TwoFactor: socialStringPtr("JBSWY3DPEHPK3PXP")},
		{Platform: "x_twitter", Name: "@batch_fresh", Password: socialStringPtr("typed-secret"), AuthCookie: socialStringPtr("ct0=fresh; auth_token=fresh"), ExecutionAuth: socialStringPtr("cookie-secret")},
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
	require.ElementsMatch(t, []int64{ownHidden.ID, fresh.ID, importResult.Accounts[2].ID}, []int64{visible[0].ID, visible[1].ID, visible[2].ID})

	removeResult, err := svc.BatchRemoveFromUserWorkbench(ctx, user.ID, []int64{ownHidden.ID, fresh.ID, importResult.Accounts[2].ID, otherAccount.ID, 0})
	require.NoError(t, err)
	require.Equal(t, 5, removeResult.Total)
	require.Equal(t, 3, removeResult.Removed)
	require.Equal(t, 2, removeResult.Skipped)
	require.Equal(t, []string{"account could not be deleted", "account could not be deleted"}, removeResult.Errors)
	require.NotContains(t, strings.Join(removeResult.Errors, " "), "error: code=")
	require.NotContains(t, strings.Join(removeResult.Errors, " "), "SOCIAL_ACCOUNT_NOT_ASSIGNED")
	require.NotContains(t, strings.Join(removeResult.Errors, " "), strconv.FormatInt(otherAccount.ID, 10))

	visible, page, err = svc.ListByUser(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, visible)
	require.Zero(t, page.Total)

	storedOther := client.SocialAccount.GetX(ctx, otherAccount.ID)
	require.NotNil(t, storedOther.AssignedUserID)
	require.Equal(t, otherUser.ID, int64(*storedOther.AssignedUserID))
	require.Nil(t, storedOther.UserWorkbenchDeletedAt)
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
		SetAuthCookie("ct0=old; auth_token=old").
		SetDefaultProxySnapshot(proxySnapshot).
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	original := client.SocialAccount.GetX(ctx, account.ID)

	newName := "@malicious_identity"
	newRestID := "fake-rest-id"
	newPassword := "new-password"
	newEmail := "  owner@example.com  "
	emptyTwoFactor := " "
	newAuthCookie := "ct0=new; auth_token=new"
	newExecutionAuth := `{"access_token":"new"}`
	newStatus := SocialAccountStatusInvalid
	newTaskStatus := SocialTaskStatusManualReview
	newProxySnapshot := `{"id":999,"endpoint":"http://attacker.proxy:8080"}`
	newRemark := "operator note"
	updated, err := svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{
		Name:                 &newName,
		PlatformUserID:       &newRestID,
		Password:             &newPassword,
		Email:                &newEmail,
		TwoFactor:            &emptyTwoFactor,
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
	require.Equal(t, "new-password", *updated.Password)
	require.NotNil(t, updated.Email)
	require.Equal(t, "owner@example.com", *updated.Email)
	require.Nil(t, updated.TwoFactor)
	require.NotNil(t, updated.AuthCookie)
	require.Equal(t, newAuthCookie, *updated.AuthCookie)
	require.NotNil(t, updated.ExecutionAuth)
	require.Equal(t, newExecutionAuth, *updated.ExecutionAuth)
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
	require.Equal(t, SocialAccountStatusAvailable, stored.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, stored.TaskStatus)
	require.NotNil(t, stored.DefaultProxySnapshot)
	require.Equal(t, proxySnapshot, *stored.DefaultProxySnapshot)

	_, err = svc.UpdateForUser(ctx, account.ID, otherUser.ID, &UpdateSocialAccountInput{Remark: socialStringPtr("cross-user")})
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)

	require.NoError(t, svc.RemoveFromUserWorkbench(ctx, user.ID, account.ID))
	_, err = svc.UpdateForUser(ctx, account.ID, user.ID, &UpdateSocialAccountInput{Remark: socialStringPtr("hidden")})
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)
}

func TestAccountWorkbenchServiceRejectsRemovedUserAccountForTask(t *testing.T) {
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

	require.NoError(t, accountSvc.RemoveFromUserWorkbench(ctx, user.ID, account.ID))

	_, err := workbench.SubmitTask(ctx, &AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     user.ID,
		AccountIDs: []int64{account.ID},
		Action:     SocialTaskActionFollow,
		Target:     socialStringPtr("target_user"),
	})
	require.ErrorIs(t, err, ErrSocialAccountNotAssigned)
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
	require.True(t, IsBillableSocialTaskAction(SocialTaskActionRetweet))
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

	_, err = svc.Assign(ctx, account.ID, user2.ID)
	require.ErrorIs(t, err, ErrSocialAccountAlreadyAssigned)

	reclaimed, err := svc.Reclaim(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, reclaimed.AssignedUserID)
	require.Nil(t, reclaimed.DefaultProxySnapshot)
}

func TestSocialAccountAssignAndReclaimClearWorkbenchRemovalState(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountService(client)

	user1 := client.User.Create().SetEmail("hidden-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	user2 := client.User.Create().SetEmail("new-owner@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("reassign_hidden").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("reassign_hidden").
		SetAssignedUserID(user1.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	require.NoError(t, svc.RemoveFromUserWorkbench(ctx, user1.ID, account.ID))
	hidden := client.SocialAccount.GetX(ctx, account.ID)
	require.NotNil(t, hidden.UserWorkbenchDeletedAt)

	reclaimed, err := svc.Reclaim(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, reclaimed.AssignedUserID)
	require.Nil(t, reclaimed.UserWorkbenchDeletedAt)

	assigned, err := svc.Assign(ctx, account.ID, user2.ID)
	require.NoError(t, err)
	require.Equal(t, user2.ID, *assigned.AssignedUserID)
	require.Nil(t, assigned.UserWorkbenchDeletedAt)

	visible, page, err := svc.ListByUser(ctx, user2.ID, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, visible, 1)
	require.Equal(t, account.ID, visible[0].ID)
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
	require.Equal(t, "account_not_visible", result.Items[1].Reason)
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})
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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

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
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(nil, nil, nil, nil), SocialTaskExecutorConfig{})

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

	for _, proxyType := range []string{"residential", "static", "mobile", "datacenter"} {
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
