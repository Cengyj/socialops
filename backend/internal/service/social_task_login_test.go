//go:build unit

package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/stretchr/testify/require"
)

// loginExecutorStub implements both SocialPlatformExecutor and
// SocialAccountLoginExecutor for executor-level login tests.
type loginExecutorStub struct {
	result *SocialAccountCredentialResult
	err    error
}

func (s loginExecutorStub) Execute(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (string, error) {
	return "", nil
}

func (s loginExecutorStub) Login(context.Context, *dbent.SocialTaskLog, *dbent.SocialAccount) (*SocialAccountCredentialResult, error) {
	return s.result, s.err
}

func TestSocialTaskExecutorLoginWritesBackCredentialsAndCharges(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-writeback@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	// A freshly imported account: not yet available, no cookie, has a password.
	account := client.SocialAccount.Create().
		SetName("login_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_account").
		SetPassword("p@ssw0rd").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	platformUserID := "rest-12345"
	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).
		WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", loginExecutorStub{result: &SocialAccountCredentialResult{
		ExecutionAuth:  `{"access_token":"a","token_secret":"b","screen_name":"login_account"}`,
		AuthCookie:     `{"access_token":"a","token_secret":"b","screen_name":"login_account","guest_token":"g","client_uuid":"uuid"}`,
		PlatformUserID: &platformUserID,
		Message:        "login succeeded",
	}})

	executor.processTask(log.ID)

	storedLog, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusSuccess, storedLog.Status)
	require.Equal(t, SocialTaskChargeStatusCharged, storedLog.ChargeStatus)
	require.InEpsilon(t, SocialTaskUnitPrice, storedLog.ChargedAmount, 0.000001)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusStored, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.ExecutionAuth)
	require.NotContains(t, *storedAccount.ExecutionAuth, "access_token")
	require.NotContains(t, *storedAccount.ExecutionAuth, "token_secret")
	decryptedExecutionAuth, err := decryptTwitterExecutionAuthCiphertext(*storedAccount.ExecutionAuth, executionAuthEncryptorStub{})
	require.NoError(t, err)
	require.Equal(t, `{"access_token":"a","token_secret":"b","screen_name":"login_account"}`, decryptedExecutionAuth)
	require.NotNil(t, storedAccount.AuthCookie)
	require.Equal(t, `{"access_token":"a","token_secret":"b","screen_name":"login_account","guest_token":"g","client_uuid":"uuid"}`, *storedAccount.AuthCookie)
	require.NotEqual(t, *storedAccount.AuthCookie, *storedAccount.ExecutionAuth)
	require.NotNil(t, storedAccount.PlatformUserID)
	require.Equal(t, platformUserID, *storedAccount.PlatformUserID)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 0.9, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorLoginFailsClosedWithoutCharge(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-fail@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_fail_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_fail_account").
		SetPassword("p@ssw0rd").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", loginExecutorStub{err: newSocialExecutionError(SocialExecutionFailureAuthInvalid, "twitter login did not return OAuth credentials", nil)})

	executor.processTask(log.ID)

	storedLog, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedLog.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedLog.ChargeStatus)
	require.Zero(t, storedLog.ChargedAmount)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, storedAccount.ExecutionAuth)
	require.NotEqual(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestTwitterExecutorLoginUsesGlobalProxySnapshotWithoutUserProxyID(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-global-proxy-executor@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_global_proxy_executor").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_global_proxy_executor").
		SetPassword("p@ssw0rd").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	proxySnapshot := `{"scope":"global","id":17,"name":"global","ip_type":"residential","endpoint":"http://8.8.8.8:8080","status":"online"}`
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)
	registrar := &fakeSocialAccountCredentialRegistrar{res: &SocialAccountCredentialResult{
		ExecutionAuth: `{"access_token":"a","token_secret":"b","screen_name":"login_global_proxy_executor"}`,
		Message:       "login succeeded",
	}}
	executor := NewTwitterExecutor().WithLoginRegistrar(registrar)

	result, err := executor.Login(ctx, log, account)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(0), registrar.req.ProxyID)
	require.Equal(t, "http://8.8.8.8:8080", registrar.req.ProxyEndpoint)
}

func TestSocialTaskExecutorLoginDependencyUnconfiguredMarksTaskFailedWithoutChangingAccountStatus(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-unavailable@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_unavailable_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_unavailable_account").
		SetPassword("p@ssw0rd").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", NewTwitterExecutor().WithLoginRegistrar(
		NewTwitterAccountCredentialRegistrar().WithDeviceParamProvider(NewHTTPDeviceParamProvider(TwitterDeviceParamConfig{})),
	))

	executor.processTask(log.ID)

	storedLog, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedLog.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedLog.ChargeStatus)
	require.Zero(t, storedLog.ChargedAmount)
	require.NotNil(t, storedLog.ResultMessage)
	require.Equal(t, "登录依赖服务未配置，本次未扣费", *storedLog.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusFailed, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "登录依赖服务未配置，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorLoginPasswordFailureShowsPasswordMessage(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-password-fail@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_password_fail_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_password_fail_account").
		SetPassword("wrong-password").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusNotStored).
		SetTaskStatus(SocialTaskStatusPending).
		SaveX(ctx)
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", loginExecutorStub{err: newSocialExecutionError(SocialExecutionFailurePasswordInvalid, "twitter password is incorrect", nil)})

	executor.processTask(log.ID)

	storedLog, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedLog.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedLog.ChargeStatus)
	require.Zero(t, storedLog.ChargedAmount)
	require.NotNil(t, storedLog.ResultMessage)
	require.Equal(t, "密码错误，本次未扣费", *storedLog.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, storedAccount.ExecutionAuth)
	require.Equal(t, SocialAccountStatusInvalid, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusManualReview, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "密码错误，本次未扣费", *storedAccount.TaskMessage)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, storedUser.Balance, 0.000001)
}

func TestSocialTaskExecutorLoginWithoutRegistrarUsesConfigurationMessage(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-no-registrar@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("login_no_registrar_account").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_no_registrar_account").
		SetPassword("p@ssw0rd").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusAvailable).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)
	proxySnapshot := `{"id":1,"endpoint":"http://8.8.8.8:8080","status":"online"}`
	log := client.SocialTaskLog.Create().
		SetSocialAccountID(account.ID).
		SetUserID(user.ID).
		SetAction(SocialTaskActionLogin).
		SetProxySnapshot(proxySnapshot).
		SetStatus(SocialTaskLogStatusPending).
		SetPrice(SocialTaskUnitPrice).
		SetChargedAmount(0).
		SetChargeStatus(SocialTaskChargeStatusNotCharged).
		SaveX(ctx)

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1})
	executor.RegisterPlatformExecutor("x_twitter", NewTwitterExecutor())

	executor.processTask(log.ID)

	storedLog, err := client.SocialTaskLog.Get(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, SocialTaskLogStatusFailed, storedLog.Status)
	require.Equal(t, SocialTaskChargeStatusNotCharged, storedLog.ChargeStatus)
	require.NotNil(t, storedLog.ResultMessage)
	require.Equal(t, "登录依赖服务未配置，本次未扣费", *storedLog.ResultMessage)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, SocialAccountStatusAvailable, storedAccount.AccountStatus)
	require.Equal(t, SocialTaskStatusFailed, storedAccount.TaskStatus)
	require.NotNil(t, storedAccount.TaskMessage)
	require.Equal(t, "登录依赖服务未配置，本次未扣费", *storedAccount.TaskMessage)
}
