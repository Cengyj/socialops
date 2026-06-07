//go:build unit

package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthHandlerRefreshTokenBackendModeRejectsUserWithoutRotatingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	user := &service.User{
		ID:                   73,
		Email:                "backend-refresh-user@example.com",
		PasswordHash:         "stable-refresh-fingerprint",
		Role:                 service.RoleUser,
		Status:               service.StatusActive,
		TokenVersion:         5,
		TokenVersionResolved: true,
	}
	userRepo := &userHandlerRepoStub{user: user}
	refreshCache := &authRefreshTokenCacheStub{tokens: map[string]*service.RefreshTokenData{}}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-backend-mode-refresh-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
	}
	settingSvc := service.NewSettingService(
		&authRefreshSettingRepoStub{
			values: map[string]string{
				service.SettingKeyBackendModeEnabled: "true",
			},
		},
		cfg,
	)
	require.NoError(t, settingSvc.UpdateSettings(ctx, &service.SystemSettings{
		BackendModeEnabled: true,
	}))
	t.Cleanup(func() {
		_ = settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{
			BackendModeEnabled: false,
		})
	})

	authService := service.NewAuthService(nil, userRepo, nil, refreshCache, cfg, settingSvc, nil, nil, nil, nil, nil, nil)
	handler := &AuthHandler{
		authService: authService,
		settingSvc:  settingSvc,
	}

	tokenPair, err := authService.GenerateTokenPair(ctx, user, "")
	require.NoError(t, err)
	require.Equal(t, 1, refreshCache.storeCalls)

	body := bytes.NewBufferString(`{"refresh_token":"` + tokenPair.RefreshToken + `"}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshToken(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, 0, refreshCache.deleteCalls, "backend mode rejection must not consume the current refresh token")
	require.Equal(t, 1, refreshCache.storeCalls, "backend mode rejection must not issue an orphaned replacement refresh token")
}

type authRefreshSettingRepoStub struct {
	values map[string]string
}

func (s *authRefreshSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (s *authRefreshSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *authRefreshSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *authRefreshSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *authRefreshSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *authRefreshSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *authRefreshSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type authRefreshTokenCacheStub struct {
	tokens      map[string]*service.RefreshTokenData
	storeCalls  int
	deleteCalls int
}

func (s *authRefreshTokenCacheStub) StoreRefreshToken(_ context.Context, tokenHash string, data *service.RefreshTokenData, _ time.Duration) error {
	s.storeCalls++
	cloned := *data
	s.tokens[tokenHash] = &cloned
	return nil
}

func (s *authRefreshTokenCacheStub) GetRefreshToken(_ context.Context, tokenHash string) (*service.RefreshTokenData, error) {
	data, ok := s.tokens[tokenHash]
	if !ok {
		return nil, service.ErrRefreshTokenNotFound
	}
	cloned := *data
	return &cloned, nil
}

func (s *authRefreshTokenCacheStub) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	s.deleteCalls++
	delete(s.tokens, tokenHash)
	return nil
}

func (s *authRefreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}

func (s *authRefreshTokenCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}

func (s *authRefreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *authRefreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *authRefreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (s *authRefreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *authRefreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}
