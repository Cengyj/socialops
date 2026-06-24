//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/stretchr/testify/require"
)

type executionAuthEncryptorStub struct{}

func (executionAuthEncryptorStub) Encrypt(plaintext string) (string, error) {
	return "enc:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (executionAuthEncryptorStub) Decrypt(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return "", errors.New("not encrypted")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func TestTwitterExecutionAuthEncryptedStorageRoundTrip(t *testing.T) {
	encryptor := executionAuthEncryptorStub{}
	payload := `{"access_token":"access","token_secret":"secret"}`
	normalized := `{"access_token":"access","token_secret":"secret","screen_name":"northwind_ops"}`

	stored, err := normalizeTwitterExecutionAuthForEncryptedStorage(payload, "@northwind_ops", encryptor)
	require.NoError(t, err)
	require.NotEqual(t, normalized, stored)
	require.NotContains(t, stored, "access")
	require.NotContains(t, stored, "secret")

	decrypted, err := decryptTwitterExecutionAuthCiphertext(stored, encryptor)
	require.NoError(t, err)
	require.Equal(t, normalized, decrypted)

	storedAgain, err := normalizeTwitterExecutionAuthForEncryptedStorage(stored, "@northwind_ops", encryptor)
	require.NoError(t, err)
	require.Equal(t, stored, storedAgain)

	headers, err := twitterAuthHeadersFromStoredExecutionAuth(stored, encryptor)
	require.NoError(t, err)
	require.Equal(t, "access", headers.AccessToken)
	require.Equal(t, "secret", headers.TokenSecret)
	require.Equal(t, "northwind_ops", headers.ScreenName)
}

func TestTwitterExecutionAuthReaderRejectsPlainJSONStorage(t *testing.T) {
	_, err := twitterAuthHeadersFromStoredExecutionAuth(`{"access_token":"stored","token_secret":"secret","screen_name":"old_ops"}`, executionAuthEncryptorStub{})
	require.Error(t, err)
}

func TestSocialAccountServiceStoresEncryptedExecutionAuthWhenConfigured(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountServiceWithCredentialEncryptor(client, executionAuthEncryptorStub{})

	account, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:          "@northwind_ops",
		Platform:      "x_twitter",
		Password:      stringPtr("password"),
		AuthCookie:    stringPtr("ct0=token; auth_token=auth"),
		ExecutionAuth: stringPtr(`{"access_token":"access","token_secret":"secret"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, account.ExecutionAuth)
	require.NotContains(t, *account.ExecutionAuth, "access")
	require.NotContains(t, *account.ExecutionAuth, "secret")

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.ExecutionAuth)
	require.Equal(t, *account.ExecutionAuth, *stored.ExecutionAuth)

	decrypted, err := decryptTwitterExecutionAuthCiphertext(*stored.ExecutionAuth, executionAuthEncryptorStub{})
	require.NoError(t, err)
	require.Equal(t, `{"access_token":"access","token_secret":"secret","screen_name":"northwind_ops"}`, decrypted)
}

func TestSocialAccountServiceSuppressesPlainJSONExecutionAuthOnOutput(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	plainJSON := `{"access_token":"stored-access","token_secret":"stored-secret","screen_name":"stored_ops"}`
	stored := client.SocialAccount.Create().
		SetName("@stored_ops").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("stored_ops").
		SetIdentityKind("username").
		SetIdentityKey("stored_ops").
		SetPassword("password").
		SetAuthCookie("ct0=stored; auth_token=stored").
		SetExecutionAuth(plainJSON).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
		SaveX(ctx)

	svc := NewSocialAccountServiceWithCredentialEncryptor(client, executionAuthEncryptorStub{})
	account, err := svc.GetByID(ctx, stored.ID)
	require.NoError(t, err)
	require.Nil(t, account.ExecutionAuth)

	updated, err := client.SocialAccount.Get(ctx, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.ExecutionAuth)
	require.Equal(t, plainJSON, *updated.ExecutionAuth)
}

func TestSocialAccountServicePreservesOpaqueExecutionAuthCiphertext(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	svc := NewSocialAccountServiceWithCredentialEncryptor(client, executionAuthEncryptorStub{})
	ciphertext := "encrypted-execution-auth-ciphertext"

	account, err := svc.Create(ctx, &CreateSocialAccountInput{
		Name:          "@opaque_ops",
		Platform:      "x_twitter",
		Password:      stringPtr("password"),
		AuthCookie:    stringPtr("ct0=token; auth_token=auth"),
		ExecutionAuth: &ciphertext,
	})
	require.NoError(t, err)
	require.NotNil(t, account.ExecutionAuth)
	require.Equal(t, ciphertext, *account.ExecutionAuth)

	stored, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.ExecutionAuth)
	require.Equal(t, ciphertext, *stored.ExecutionAuth)
}

func TestSocialTaskExecutorLoginWritesBackEncryptedExecutionAuthWhenConfigured(t *testing.T) {
	ctx := context.Background()
	client := newSocialOpsServiceTestClient(t)
	user := client.User.Create().SetEmail("login-encrypted@example.com").SetPasswordHash("hash").SetBalance(1).SaveX(ctx)
	account := client.SocialAccount.Create().
		SetName("@login_encrypted").
		SetPlatform("x_twitter").
		SetPlatformKey("x_twitter").
		SetNameKey("login_encrypted").
		SetIdentityKind("username").
		SetIdentityKey("login_encrypted").
		SetPassword("password").
		SetAssignedUserID(user.ID).
		SetAccountStatus(SocialAccountStatusPendingCheck).
		SetTaskStatus(SocialTaskStatusStored).
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

	executor := NewSocialTaskExecutor(client, NewSocialBillingService(&socialBillingUserRepoStub{user: &User{ID: user.ID, Balance: 1}}, &subscriptionRepoState{}, &socialBillingGroupRepoStub{}, nil), SocialTaskExecutorConfig{WorkerCount: 1, QueueSize: 1}).
		WithCredentialEncryptor(executionAuthEncryptorStub{})
	executor.RegisterPlatformExecutor("x_twitter", loginExecutorStub{result: &SocialAccountCredentialResult{
		ExecutionAuth: `{"access_token":"login-access","token_secret":"login-secret"}`,
		Message:       "login succeeded",
	}})

	executor.processTask(log.ID)

	storedAccount, err := client.SocialAccount.Get(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, storedAccount.ExecutionAuth)
	require.NotContains(t, *storedAccount.ExecutionAuth, "login-access")
	require.NotContains(t, *storedAccount.ExecutionAuth, "login-secret")

	twitter := NewTwitterExecutor().WithCredentialEncryptor(executionAuthEncryptorStub{})
	headers, err := twitter.twitterAuthHeadersFromAccount(&dbent.SocialAccount{ExecutionAuth: storedAccount.ExecutionAuth})
	require.NoError(t, err)
	require.Equal(t, "login-access", headers.AccessToken)
	require.Equal(t, "login-secret", headers.TokenSecret)
	require.Equal(t, "login_encrypted", headers.ScreenName)
}

func stringPtr(value string) *string {
	return &value
}
