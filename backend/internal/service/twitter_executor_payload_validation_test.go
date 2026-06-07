//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/stretchr/testify/require"
)

func TestSocialTaskExecutorDispatchesPayloadTargetThroughTwitterAliasExecutor(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("payload-target-dispatch@example.com").SetPasswordHash("hash").SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("payload_target_dispatch").
		SetPlatform("x").
		SetPlatformKey("x").
		SetNameKey("payload_target_dispatch").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, nil, SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	var capturedTask *dbent.SocialTaskLog
	var capturedAccount *dbent.SocialAccount
	executor.RegisterPlatformExecutor("twitter", socialPlatformExecutorFunc(func(ctx context.Context, taskLog *dbent.SocialTaskLog, account *dbent.SocialAccount) (string, error) {
		capturedTask = taskLog
		capturedAccount = account
		return "fake twitter follow ok", nil
	}))

	result, err := executor.executeAction(ctx, &dbent.SocialTaskLog{
		ID:              101,
		SocialAccountID: account.ID,
		UserID:          user.ID,
		Action:          SocialTaskActionFollow,
		Payload: SocialTaskPayload{
			Target: "123456789",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "fake twitter follow ok", result)
	require.NotNil(t, capturedTask)
	require.Equal(t, "123456789", capturedTask.Payload.Target)
	require.NotNil(t, capturedAccount)
	require.Equal(t, account.ID, capturedAccount.ID)
}

func TestAccountWorkbenchTaskInputValidatesStructuredPayloadMediaBeforePersisting(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		payload   *SocialTaskPayload
		wantError string
	}{
		{
			name:   "post video fails closed",
			action: SocialTaskActionPost,
			payload: &SocialTaskPayload{Post: &SocialPostPayload{
				Text: "hello video",
				Media: []SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "video/mp4",
					FileName:    "clip.mp4",
					URL:         "data:video/mp4;base64,QUJD",
				}},
			}},
			wantError: "video media is not implemented yet",
		},
		{
			name:   "avatar rejects non-image",
			action: SocialTaskActionUpdateAvatar,
			payload: &SocialTaskPayload{Avatar: &SocialTaskMediaRef{
				Source:      "inline",
				ContentType: "application/pdf",
				FileName:    "avatar.pdf",
				URL:         "data:application/pdf;base64,QUJD",
			}},
			wantError: "avatar media must be an image",
		},
		{
			name:      "profile requires non-empty profile",
			action:    SocialTaskActionUpdateProfile,
			payload:   &SocialTaskPayload{Profile: &SocialProfileUpdateParams{}},
			wantError: "profile payload is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := normalizeAccountWorkbenchTaskInput(&AccountWorkbenchTaskInput{
				Mode:       AccountWorkbenchTaskModeUser,
				UserID:     1,
				AccountIDs: []int64{1},
				Action:     tc.action,
				Payload:    tc.payload,
			})

			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestSocialTaskExecutorFailuresRemainNotChargedAcrossAuthProxyAndMedia(t *testing.T) {
	ctx := context.Background()
	baseAuth := `{"access_token":"access-token","token_secret":"token-secret","client_uuid":"client-uuid","twitter_client":"TwitterAndroid","client_version":"11.46.0-release.0","twitter_api_version":"5","client_language":"en-US","client_device_id":"device-id","client_limit_ad_tracking":"0","user_agent":"TwitterAndroid/11.46.0-release.0","accept_language":"en-US","accept_encoding":"gzip","timezone":"Pacific/Honolulu","os_security_patch_level":"2024-10-05"}`
	onlineProxy := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	offlineProxy := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"offline"}`

	tests := []struct {
		name           string
		action         string
		content        *string
		payload        SocialTaskPayload
		auth           *string
		proxySnapshot  *string
		wantMessage    string
		wantStatus     string
		wantTaskStatus string
	}{
		{
			name:           "missing auth",
			action:         SocialTaskActionFollow,
			payload:        SocialTaskPayload{Target: "123456789"},
			proxySnapshot:  &onlineProxy,
			wantMessage:    "账号认证信息不可用，本次未扣费",
			wantStatus:     SocialAccountStatusNotStored,
			wantTaskStatus: SocialTaskStatusManualReview,
		},
		{
			name:           "invalid auth",
			action:         SocialTaskActionFollow,
			payload:        SocialTaskPayload{Target: "123456789"},
			auth:           socialStringPtr(`{"access_token":"access-token"}`),
			proxySnapshot:  &onlineProxy,
			wantMessage:    "账号认证信息不可用，本次未扣费",
			wantStatus:     SocialAccountStatusInvalid,
			wantTaskStatus: SocialTaskStatusManualReview,
		},
		{
			name:           "offline proxy",
			action:         SocialTaskActionFollow,
			payload:        SocialTaskPayload{Target: "123456789"},
			auth:           &baseAuth,
			proxySnapshot:  &offlineProxy,
			wantMessage:    "执行代理不可用，本次未扣费",
			wantStatus:     SocialAccountStatusAvailable,
			wantTaskStatus: SocialTaskStatusIPUnavailable,
		},
		{
			name:    "video media fail closed",
			action:  SocialTaskActionPost,
			content: socialStringPtr("hello video"),
			payload: SocialTaskPayload{Post: &SocialPostPayload{
				Text: "hello video",
				Media: []SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "video/mp4",
					FileName:    "clip.mp4",
					URL:         "data:video/mp4;base64,QUJD",
				}},
			}},
			auth:           &baseAuth,
			proxySnapshot:  &onlineProxy,
			wantMessage:    "视频发帖媒体暂未开放，本次未扣费",
			wantStatus:     SocialAccountStatusAvailable,
			wantTaskStatus: SocialTaskStatusStored,
		},
		{
			name:    "unsupported post media fail closed",
			action:  SocialTaskActionPost,
			content: socialStringPtr("hello file"),
			payload: SocialTaskPayload{Post: &SocialPostPayload{
				Text: "hello file",
				Media: []SocialTaskMediaRef{{
					Source:      "inline",
					ContentType: "application/pdf",
					FileName:    "spec.pdf",
					URL:         "data:application/pdf;base64,QUJD",
				}},
			}},
			auth:           &baseAuth,
			proxySnapshot:  &onlineProxy,
			wantMessage:    "发帖媒体类型暂不支持，本次未扣费",
			wantStatus:     SocialAccountStatusAvailable,
			wantTaskStatus: SocialTaskStatusStored,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newSocialOpsServiceTestClient(t)
			user := client.User.Create().SetEmail(strings.ReplaceAll(tc.name, " ", "-") + "@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
			accountCreate := client.SocialAccount.Create().
				SetName(strings.ReplaceAll(tc.name, " ", "_")).
				SetPlatform("x_twitter").
				SetPlatformKey("x_twitter").
				SetNameKey(strings.ReplaceAll(tc.name, " ", "_")).
				SetAssignedUserID(user.ID).
				SetAccountStatus(SocialAccountStatusAvailable).
				SetTaskStatus(SocialTaskStatusStored)
			if tc.auth != nil {
				accountCreate.SetExecutionAuth(*tc.auth)
			}
			account := accountCreate.SaveX(ctx)

			taskCreate := client.SocialTaskLog.Create().
				SetSocialAccountID(account.ID).
				SetUserID(user.ID).
				SetAction(tc.action).
				SetStatus(SocialTaskLogStatusPending).
				SetPrice(SocialTaskUnitPrice).
				SetChargedAmount(0).
				SetChargeStatus(SocialTaskChargeStatusNotCharged).
				SetPayload(tc.payload)
			if tc.content != nil {
				taskCreate.SetContent(*tc.content)
			}
			if tc.proxySnapshot != nil {
				taskCreate.SetProxySnapshot(*tc.proxySnapshot)
			}
			task := taskCreate.SaveX(ctx)

			executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
			exec := NewTwitterExecutor()
			exec.clientForProxy = func(proxyURL string) (twitterHTTPClient, error) {
				t.Fatalf("%s should fail closed before external HTTP", tc.name)
				return nil, nil
			}
			executor.RegisterPlatformExecutor("x_twitter", exec)

			executor.processTask(task.ID)

			storedTask, err := client.SocialTaskLog.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, SocialTaskLogStatusFailed, storedTask.Status)
			require.Equal(t, SocialTaskChargeStatusNotCharged, storedTask.ChargeStatus)
			require.Zero(t, storedTask.ChargedAmount)
			require.Nil(t, storedTask.ChargeSource)
			require.Nil(t, storedTask.BillingRequestID)
			require.NotNil(t, storedTask.ResultMessage)
			require.Equal(t, tc.wantMessage, *storedTask.ResultMessage)

			storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, storedAccount.AccountStatus)
			require.Equal(t, tc.wantTaskStatus, storedAccount.TaskStatus)
			require.NotNil(t, storedAccount.TaskMessage)
			require.Equal(t, tc.wantMessage, *storedAccount.TaskMessage)

			storedUser, err := client.User.Get(ctx, user.ID)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
			ledgerCount, err := client.UsageLog.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, ledgerCount)
		})
	}
}

func TestAccountWorkbenchTaskInputAcceptsStructuredPayloadTarget(t *testing.T) {
	err := normalizeAccountWorkbenchTaskInput(&AccountWorkbenchTaskInput{
		Mode:       AccountWorkbenchTaskModeUser,
		UserID:     1,
		AccountIDs: []int64{1},
		Action:     SocialTaskActionRetweet,
		Payload: &SocialTaskPayload{
			Target: "https://x.com/openai/status/123456789",
		},
	})

	require.NoError(t, err)
}
