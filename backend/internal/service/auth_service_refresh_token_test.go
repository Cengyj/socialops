//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceRefreshTokenPairFailsClosedWhenOldTokenCannotBeDeleted(t *testing.T) {
	ctx := context.Background()
	user := &User{
		ID:                   71,
		Email:                "refresh-rotate@example.com",
		PasswordHash:         "stable-password-fingerprint",
		Role:                 RoleUser,
		Status:               StatusActive,
		TokenVersion:         3,
		TokenVersionResolved: true,
	}
	cache := &refreshTokenRotationCacheStub{
		tokens:    make(map[string]*RefreshTokenData),
		deleteErr: errors.New("redis delete failed"),
	}
	svc := NewAuthService(
		nil,
		&userRepoStub{user: user},
		nil,
		cache,
		&config.Config{
			JWT: config.JWTConfig{
				Secret:                   "test-refresh-rotation-secret",
				ExpireHour:               1,
				AccessTokenExpireMinutes: 60,
				RefreshTokenExpireDays:   7,
			},
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	tokenPair, err := svc.GenerateTokenPair(ctx, user, "")
	require.NoError(t, err)
	require.Equal(t, 1, cache.storeCalls)

	refreshed, err := svc.RefreshTokenPair(ctx, tokenPair.RefreshToken)
	require.ErrorIs(t, err, ErrServiceUnavailable)
	require.Nil(t, refreshed)
	require.Equal(t, 1, cache.deleteCalls)
	require.Equal(t, 1, cache.storeCalls, "refresh must not issue a new token when the old one was not revoked")
}

type refreshTokenRotationCacheStub struct {
	tokens      map[string]*RefreshTokenData
	storeCalls  int
	deleteCalls int
	deleteErr   error
}

func (s *refreshTokenRotationCacheStub) StoreRefreshToken(_ context.Context, tokenHash string, data *RefreshTokenData, _ time.Duration) error {
	s.storeCalls++
	cloned := *data
	s.tokens[tokenHash] = &cloned
	return nil
}

func (s *refreshTokenRotationCacheStub) GetRefreshToken(_ context.Context, tokenHash string) (*RefreshTokenData, error) {
	data, ok := s.tokens[tokenHash]
	if !ok {
		return nil, ErrRefreshTokenNotFound
	}
	cloned := *data
	return &cloned, nil
}

func (s *refreshTokenRotationCacheStub) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	s.deleteCalls++
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.tokens, tokenHash)
	return nil
}

func (s *refreshTokenRotationCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}

func (s *refreshTokenRotationCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}

func (s *refreshTokenRotationCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *refreshTokenRotationCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *refreshTokenRotationCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenRotationCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenRotationCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}
