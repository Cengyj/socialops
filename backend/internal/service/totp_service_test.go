//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
)

func TestTotpCompleteSetupRollsBackSecretWhenEnableFails(t *testing.T) {
	ctx := context.Background()
	userID := int64(7)
	secret := "JBSWY3DPEHPK3PXP"
	setupToken := "setup-token"
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	repo := &totpSetupRepoStub{
		user: &User{
			ID:     userID,
			Email:  "user@example.com",
			Status: StatusActive,
		},
		enableErr: errors.New("enable failed"),
	}
	cache := newTotpMemoryCache()
	require.NoError(t, cache.SetSetupSession(ctx, userID, &TotpSetupSession{
		Secret:     secret,
		SetupToken: setupToken,
		CreatedAt:  time.Now(),
	}, time.Minute))

	settingSvc := NewSettingService(&totpSettingRepoStub{values: map[string]string{
		SettingKeyTotpEnabled:        "true",
		SettingKeyEmailVerifyEnabled: "false",
	}}, &config.Config{})
	svc := NewTotpService(repo, totpEncryptorStub{}, cache, settingSvc, nil, nil)

	err = svc.CompleteSetup(ctx, userID, code, setupToken)

	require.Error(t, err)
	require.Equal(t, 1, repo.txCalls)
	require.Nil(t, repo.user.TotpSecretEncrypted)
	require.False(t, repo.user.TotpEnabled)
	require.Nil(t, repo.user.TotpEnabledAt)
}

func TestTotpEmailVerificationFailsClosedWhenEmailServiceMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int64(17)
	repo := &totpSetupRepoStub{
		user: &User{
			ID:           userID,
			Email:        "totp-email-missing@example.com",
			Status:       StatusActive,
			PasswordHash: "unused",
		},
	}
	settingSvc := NewSettingService(&totpSettingRepoStub{values: map[string]string{
		SettingKeyTotpEnabled:        "true",
		SettingKeyEmailVerifyEnabled: "true",
	}}, &config.Config{})
	svc := NewTotpService(repo, totpEncryptorStub{}, newTotpMemoryCache(), settingSvc, nil, nil)

	require.NotPanics(t, func() {
		_, err := svc.InitiateSetup(ctx, userID, "123456", "")
		require.ErrorIs(t, err, ErrServiceUnavailable)
	})
	require.NotPanics(t, func() {
		err := svc.SendVerifyCode(ctx, userID)
		require.ErrorIs(t, err, ErrServiceUnavailable)
	})
}

func TestTotpDisableEmailVerificationFailsClosedWhenEmailServiceMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	enabledAt := time.Now().UTC().Add(-time.Hour)
	secret := "encrypted-secret"
	userID := int64(18)
	repo := &totpSetupRepoStub{
		user: &User{
			ID:                  userID,
			Email:               "totp-disable-missing@example.com",
			Status:              StatusActive,
			TotpEnabled:         true,
			TotpEnabledAt:       &enabledAt,
			TotpSecretEncrypted: &secret,
		},
	}
	settingSvc := NewSettingService(&totpSettingRepoStub{values: map[string]string{
		SettingKeyTotpEnabled:        "true",
		SettingKeyEmailVerifyEnabled: "true",
	}}, &config.Config{})
	svc := NewTotpService(repo, totpEncryptorStub{}, newTotpMemoryCache(), settingSvc, nil, nil)

	require.NotPanics(t, func() {
		err := svc.Disable(ctx, userID, "123456", "")
		require.ErrorIs(t, err, ErrServiceUnavailable)
	})
	require.True(t, repo.user.TotpEnabled)
	require.Equal(t, &enabledAt, repo.user.TotpEnabledAt)
	require.Equal(t, &secret, repo.user.TotpSecretEncrypted)
}

type totpSetupRepoStub struct {
	UserRepository

	user      *User
	enableErr error
	txCalls   int
}

type totpTxContextKey struct{}

func (r *totpSetupRepoStub) GetByID(ctx context.Context, _ int64) (*User, error) {
	if txUser, _ := ctx.Value(totpTxContextKey{}).(*User); txUser != nil {
		return cloneTotpUser(txUser), nil
	}
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	return cloneTotpUser(r.user), nil
}

func (r *totpSetupRepoStub) UpdateTotpSecret(ctx context.Context, _ int64, encryptedSecret *string) error {
	user := r.user
	if txUser, _ := ctx.Value(totpTxContextKey{}).(*User); txUser != nil {
		user = txUser
	}
	user.TotpSecretEncrypted = cloneStringPtr(encryptedSecret)
	return nil
}

func (r *totpSetupRepoStub) EnableTotp(ctx context.Context, _ int64) error {
	if r.enableErr != nil {
		return r.enableErr
	}
	user := r.user
	if txUser, _ := ctx.Value(totpTxContextKey{}).(*User); txUser != nil {
		user = txUser
	}
	now := time.Now()
	user.TotpEnabled = true
	user.TotpEnabledAt = &now
	return nil
}

func (r *totpSetupRepoStub) DisableTotp(context.Context, int64) error {
	return nil
}

func (r *totpSetupRepoStub) WithUserProfileIdentityTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	r.txCalls++
	txUser := cloneTotpUser(r.user)
	if err := fn(context.WithValue(ctx, totpTxContextKey{}, txUser)); err != nil {
		return err
	}
	r.user = txUser
	return nil
}

func cloneTotpUser(user *User) *User {
	if user == nil {
		return nil
	}
	cloned := *user
	cloned.TotpSecretEncrypted = cloneStringPtr(user.TotpSecretEncrypted)
	if user.TotpEnabledAt != nil {
		enabledAt := *user.TotpEnabledAt
		cloned.TotpEnabledAt = &enabledAt
	}
	return &cloned
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

type totpEncryptorStub struct{}

func (totpEncryptorStub) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (totpEncryptorStub) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

type totpSettingRepoStub struct {
	values map[string]string
}

func (s *totpSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *totpSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *totpSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *totpSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *totpSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *totpSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *totpSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type totpMemoryCache struct {
	mu            sync.Mutex
	setupSessions map[int64]*TotpSetupSession
	loginSessions map[string]*TotpLoginSession
	attempts      map[int64]int
}

func newTotpMemoryCache() *totpMemoryCache {
	return &totpMemoryCache{
		setupSessions: map[int64]*TotpSetupSession{},
		loginSessions: map[string]*TotpLoginSession{},
		attempts:      map[int64]int{},
	}
}

func (c *totpMemoryCache) GetSetupSession(_ context.Context, userID int64) (*TotpSetupSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setupSessions[userID], nil
}

func (c *totpMemoryCache) SetSetupSession(_ context.Context, userID int64, session *TotpSetupSession, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setupSessions[userID] = session
	return nil
}

func (c *totpMemoryCache) DeleteSetupSession(_ context.Context, userID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.setupSessions, userID)
	return nil
}

func (c *totpMemoryCache) GetLoginSession(_ context.Context, tempToken string) (*TotpLoginSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginSessions[tempToken], nil
}

func (c *totpMemoryCache) SetLoginSession(_ context.Context, tempToken string, session *TotpLoginSession, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loginSessions[tempToken] = session
	return nil
}

func (c *totpMemoryCache) DeleteLoginSession(_ context.Context, tempToken string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.loginSessions, tempToken)
	return nil
}

func (c *totpMemoryCache) IncrementVerifyAttempts(_ context.Context, userID int64) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts[userID]++
	return c.attempts[userID], nil
}

func (c *totpMemoryCache) GetVerifyAttempts(_ context.Context, userID int64) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.attempts[userID], nil
}

func (c *totpMemoryCache) ClearVerifyAttempts(_ context.Context, userID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.attempts, userID)
	return nil
}
